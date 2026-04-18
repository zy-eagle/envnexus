package system

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFileDownloadTool_Metadata(t *testing.T) {
	tool := NewFileDownloadTool()
	if tool.Name() != "file_download" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "file_download")
	}
	if tool.IsReadOnly() {
		t.Error("IsReadOnly() should be false")
	}
	if tool.RiskLevel() != "L2" {
		t.Errorf("RiskLevel() = %q, want %q", tool.RiskLevel(), "L2")
	}
	if !tool.NeedsApproval(nil) {
		t.Error("NeedsApproval() should be true")
	}
}

func TestFileDownloadTool_MissingParams(t *testing.T) {
	tool := NewFileDownloadTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("expected failed status, got %q", result.Status)
	}

	result, err = tool.Execute(ctx, map[string]interface{}{"path": "/tmp/foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("expected failed status for missing upload_url, got %q", result.Status)
	}
}

func TestFileDownloadTool_SensitivePath(t *testing.T) {
	tool := NewFileDownloadTool()
	ctx := context.Background()

	sensitive := []string{
		"/etc/shadow",
		"/home/user/.ssh/id_rsa",
	}
	for _, p := range sensitive {
		result, err := tool.Execute(ctx, map[string]interface{}{
			"path":       p,
			"upload_url": "http://localhost/upload",
		})
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", p, err)
		}
		if result.Status != "failed" {
			t.Errorf("expected failed status for sensitive path %q, got %q", p, result.Status)
		}
	}
}

func TestFileDownloadTool_FileNotExist(t *testing.T) {
	tool := NewFileDownloadTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"path":       "/nonexistent/file/path/xyz",
		"upload_url": "http://localhost/upload",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("expected failed status for missing file, got %q", result.Status)
	}
}

func TestFileDownloadTool_Directory(t *testing.T) {
	tool := NewFileDownloadTool()
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"path":       os.TempDir(),
		"upload_url": "http://localhost/upload",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("expected failed status for directory, got %q", result.Status)
	}
}

func TestFileDownloadTool_SuccessfulUpload(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	content := "hello world"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var err error
		received, err = io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tool := NewFileDownloadTool()
	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"path":       tmpFile,
		"upload_url": srv.URL + "/upload",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "succeeded" {
		t.Errorf("expected succeeded, got %q, error: %s", result.Status, result.Error)
	}
	if string(received) != content {
		t.Errorf("received %q, want %q", string(received), content)
	}
}

func TestFileDownloadTool_FileTooLarge(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "big.txt")
	if err := os.WriteFile(tmpFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := NewFileDownloadTool()
	ctx := context.Background()
	result, err := tool.Execute(ctx, map[string]interface{}{
		"path":        tmpFile,
		"upload_url":  "http://localhost/upload",
		"max_size_mb": float64(0.000001),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("expected failed for too-large file, got %q", result.Status)
	}
}

func TestIsSensitivePath(t *testing.T) {
	cases := []struct {
		path     string
		expected bool
	}{
		{"/etc/shadow", true},
		{"/home/user/.ssh/id_rsa", true},
		{"/home/user/.gnupg/keyring", true},
		{"/tmp/safe.txt", false},
		{"/var/log/syslog", false},
	}
	for _, tc := range cases {
		got := isSensitivePath(tc.path)
		if got != tc.expected {
			t.Errorf("isSensitivePath(%q) = %v, want %v", tc.path, got, tc.expected)
		}
	}
}
