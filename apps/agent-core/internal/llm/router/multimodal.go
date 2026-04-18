package router

import "context"

// ContentPart represents a single part of multimodal content.
// For text, set Type="text" and Text. For images, set Type="image_url" and ImageURL.
type ContentPart struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	ImageURL *ImageURLPart  `json:"image_url,omitempty"`
}

type ImageURLPart struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// MultimodalMessage extends Message with content parts for vision models.
type MultimodalMessage struct {
	Role         string        `json:"role"`
	ContentParts []ContentPart `json:"content,omitempty"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID   string        `json:"tool_call_id,omitempty"`
	Name         string        `json:"name,omitempty"`
}

// MultimodalRequest supports mixed text+image content.
type MultimodalRequest struct {
	Messages    []MultimodalMessage `json:"messages"`
	Tools       []ToolDefinition    `json:"tools,omitempty"`
	Model       string              `json:"model,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
}

// VisionProvider can handle image inputs in addition to text.
type VisionProvider interface {
	Provider
	SupportsVision() bool
	CompleteMultimodal(ctx context.Context, req *MultimodalRequest) (*CompletionResponse, error)
}

// NewTextPart creates a text content part.
func NewTextPart(text string) ContentPart {
	return ContentPart{Type: "text", Text: text}
}

// NewImageURLPart creates an image_url content part from a URL.
func NewImageURLPart(url, detail string) ContentPart {
	if detail == "" {
		detail = "auto"
	}
	return ContentPart{
		Type:     "image_url",
		ImageURL: &ImageURLPart{URL: url, Detail: detail},
	}
}

// NewImageBase64Part creates an image_url content part from base64 data.
func NewImageBase64Part(mediaType string, base64Data string, detail string) ContentPart {
	if detail == "" {
		detail = "auto"
	}
	return ContentPart{
		Type: "image_url",
		ImageURL: &ImageURLPart{
			URL:    "data:" + mediaType + ";base64," + base64Data,
			Detail: detail,
		},
	}
}

// HasImages checks if a multimodal message contains image parts.
func (m *MultimodalMessage) HasImages() bool {
	for _, p := range m.ContentParts {
		if p.Type == "image_url" {
			return true
		}
	}
	return false
}
