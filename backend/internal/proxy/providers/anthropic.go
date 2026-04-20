package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AnthropicProvider handles Claude models via the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	models  []string
	client  *http.Client
}

func NewAnthropicProvider(apiKey string, models []string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com",
		models:  models,
		client:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *AnthropicProvider) ModelIDs() []string { return p.models }

// anthropicRequest is the Anthropic Messages API request format.
type anthropicRequest struct {
	Model     string              `json:"model"`
	Messages  []anthropicMessage  `json:"messages"`
	MaxTokens int                 `json:"max_tokens"`
	Stream    bool                `json:"stream"`
	System    string              `json:"system,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (p *AnthropicProvider) Complete(ctx context.Context, req *ChatRequest) (*ChatResponse, *UsageInfo, error) {
	aReq := p.toAnthropicRequest(req)
	body, err := json.Marshal(aReq)
	if err != nil {
		return nil, nil, err
	}

	httpResp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, nil, err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, nil, err
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("anthropic error %d: %s", httpResp.StatusCode, string(respBody))
	}

	var aResp struct {
		ID      string `json:"id"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &aResp); err != nil {
		return nil, nil, fmt.Errorf("parse anthropic response: %w", err)
	}

	content := ""
	if len(aResp.Content) > 0 {
		content = aResp.Content[0].Text
	}

	usage := &UsageInfo{
		InputTokens:  aResp.Usage.InputTokens,
		OutputTokens: aResp.Usage.OutputTokens,
		TotalTokens:  aResp.Usage.InputTokens + aResp.Usage.OutputTokens,
	}

	return &ChatResponse{
		ID:    aResp.ID,
		Model: req.Model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{{
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{Role: "assistant", Content: content},
			FinishReason: "stop",
		}},
		Usage: usage,
	}, usage, nil
}

func (p *AnthropicProvider) CompleteStream(ctx context.Context, req *ChatRequest, w io.Writer) (*UsageInfo, error) {
	aReq := p.toAnthropicRequest(req)
	aReq.Stream = true

	body, err := json.Marshal(aReq)
	if err != nil {
		return nil, err
	}

	httpResp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("anthropic error %d: %s", httpResp.StatusCode, string(errBody))
	}

	usage := &UsageInfo{}
	scanner := bufio.NewScanner(httpResp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	var eventType string
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Extract token counts from Anthropic SSE events
		switch eventType {
		case "message_start":
			var evt struct {
				Message struct {
					Usage struct {
						InputTokens int `json:"input_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			if err := json.Unmarshal([]byte(data), &evt); err == nil {
				usage.InputTokens = evt.Message.Usage.InputTokens
			}

		case "message_delta":
			var evt struct {
				Usage struct {
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &evt); err == nil {
				usage.OutputTokens = evt.Usage.OutputTokens
				usage.TotalTokens = usage.InputTokens + usage.OutputTokens
			}

		case "content_block_delta":
			// Convert Anthropic delta to OpenAI-compatible SSE chunk for client
			var evt struct {
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &evt); err == nil && evt.Delta.Text != "" {
				chunk := map[string]any{
					"choices": []map[string]any{{
						"delta":         map[string]string{"content": evt.Delta.Text},
						"finish_reason": nil,
						"index":         0,
					}},
					"model": req.Model,
				}
				if chunkJSON, err := json.Marshal(chunk); err == nil {
					fmt.Fprintf(w, "data: %s\n\n", string(chunkJSON))
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				}
			}
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return usage, scanner.Err()
}

func (p *AnthropicProvider) toAnthropicRequest(req *ChatRequest) *anthropicRequest {
	aReq := &anthropicRequest{
		Model:     req.Model,
		MaxTokens: 4096,
	}
	if req.MaxTokens > 0 {
		aReq.MaxTokens = req.MaxTokens
	}

	for _, m := range req.Messages {
		if m.Role == "system" {
			if text, ok := m.Content.(string); ok {
				aReq.System = text
			}
			continue
		}
		content := ""
		if text, ok := m.Content.(string); ok {
			content = text
		}
		aReq.Messages = append(aReq.Messages, anthropicMessage{
			Role:    m.Role,
			Content: content,
		})
	}
	return aReq
}

func (p *AnthropicProvider) doRequest(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	return p.client.Do(req)
}
