package providers

import (
	"context"
	"io"
)

type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []ContentPart
}

type ContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	Stream      bool      `json:"stream"`
	TopP        *float64  `json:"top_p,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *UsageInfo `json:"usage,omitempty"`
}

type UsageInfo struct {
	InputTokens  int `json:"prompt_tokens"`
	OutputTokens int `json:"completion_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Provider is the interface all AI provider adapters must implement.
type Provider interface {
	// Complete handles non-streaming requests.
	Complete(ctx context.Context, req *ChatRequest) (*ChatResponse, *UsageInfo, error)

	// CompleteStream handles streaming requests.
	// It writes raw SSE bytes to w and returns token usage from the final chunk.
	CompleteStream(ctx context.Context, req *ChatRequest, w io.Writer) (*UsageInfo, error)

	// ModelIDs returns the model identifiers this provider handles.
	ModelIDs() []string
}
