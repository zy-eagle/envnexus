package system

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ExpandUserPath resolves ~ prefix to the user's home directory on all platforms.
func ExpandUserPath(path string) (string, error) {
	if strings.HasPrefix(path, "~"+string(os.PathSeparator)) || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// NormPathKey normalises a path for case-insensitive matching on Windows.
func NormPathKey(p string) string {
	s := filepath.ToSlash(p)
	if runtime.GOOS == "windows" {
		return strings.ToLower(s)
	}
	return s
}

var blockedSubstrings = []string{
	"/etc/shadow",
	"/etc/passwd",
	`\windows\system32\config\sam`,
	`/windows/system32/config/sam`,
}

var sensitiveBasenames = map[string]struct{}{
	"id_rsa": {}, "id_ed25519": {}, "id_ecdsa": {}, "id_dsa": {},
}

// IsSensitivePath returns true if the absolute, cleaned path points to
// a well-known credential or system-secret location (cross-platform).
func IsSensitivePath(absClean string) bool {
	key := NormPathKey(absClean)
	for _, sub := range blockedSubstrings {
		cmp := sub
		if runtime.GOOS == "windows" {
			cmp = strings.ToLower(filepath.ToSlash(sub))
		}
		if strings.Contains(key, cmp) {
			return true
		}
	}

	segs := strings.Split(key, "/")
	for _, seg := range segs {
		if seg == ".ssh" || seg == ".gnupg" {
			return true
		}
	}

	base := strings.ToLower(filepath.Base(absClean))
	if _, ok := sensitiveBasenames[base]; ok {
		return true
	}

	return false
}

// ResolveSafePath expands ~, resolves to absolute, and rejects sensitive locations.
// Returns the clean absolute path or a non-nil error.
func ResolveSafePath(raw string) (string, error) {
	expanded, err := ExpandUserPath(raw)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(filepath.Clean(expanded))
	if err != nil {
		return "", err
	}
	if IsSensitivePath(abs) {
		return "", &SensitivePathError{Path: abs}
	}
	return abs, nil
}

type SensitivePathError struct{ Path string }

func (e *SensitivePathError) Error() string {
	return "access denied: path matches a sensitive location or credential pattern"
}
