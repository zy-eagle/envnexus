package router

import "testing"

func TestNewTextPart(t *testing.T) {
	p := NewTextPart("hello")
	if p.Type != "text" {
		t.Errorf("expected type 'text', got %q", p.Type)
	}
	if p.Text != "hello" {
		t.Errorf("expected text 'hello', got %q", p.Text)
	}
	if p.ImageURL != nil {
		t.Error("expected nil ImageURL")
	}
}

func TestNewImageURLPart(t *testing.T) {
	p := NewImageURLPart("https://example.com/img.png", "high")
	if p.Type != "image_url" {
		t.Errorf("expected type 'image_url', got %q", p.Type)
	}
	if p.ImageURL == nil {
		t.Fatal("expected non-nil ImageURL")
	}
	if p.ImageURL.URL != "https://example.com/img.png" {
		t.Errorf("unexpected URL: %q", p.ImageURL.URL)
	}
	if p.ImageURL.Detail != "high" {
		t.Errorf("expected detail 'high', got %q", p.ImageURL.Detail)
	}
}

func TestNewImageURLPart_DefaultDetail(t *testing.T) {
	p := NewImageURLPart("https://example.com/img.png", "")
	if p.ImageURL.Detail != "auto" {
		t.Errorf("expected default detail 'auto', got %q", p.ImageURL.Detail)
	}
}

func TestNewImageBase64Part(t *testing.T) {
	p := NewImageBase64Part("image/png", "aGVsbG8=", "low")
	if p.Type != "image_url" {
		t.Errorf("expected type 'image_url', got %q", p.Type)
	}
	expected := "data:image/png;base64,aGVsbG8="
	if p.ImageURL.URL != expected {
		t.Errorf("expected URL %q, got %q", expected, p.ImageURL.URL)
	}
}

func TestMultimodalMessage_HasImages(t *testing.T) {
	textOnly := &MultimodalMessage{
		Role: "user",
		ContentParts: []ContentPart{NewTextPart("hello")},
	}
	if textOnly.HasImages() {
		t.Error("text-only message should not have images")
	}

	withImage := &MultimodalMessage{
		Role: "user",
		ContentParts: []ContentPart{
			NewTextPart("describe this"),
			NewImageURLPart("https://example.com/img.png", ""),
		},
	}
	if !withImage.HasImages() {
		t.Error("message with image should have images")
	}
}
