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

// OpenAIProvider handles OpenAI-compatible APIs (OpenAI, Qwen DashScope, etc.).
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	models  []string
	client  *http.Client
}

func NewOpenAIProvider(apiKey, baseURL string, models []string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		models:  models,
		client:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *OpenAIProvider) ModelIDs() []string { return p.models }

func (p *OpenAIProvider) Complete(ctx context.Context, req *ChatRequest) (*ChatResponse, *UsageInfo, error) {
	body, err := json.Marshal(req)
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
		return nil, nil, fmt.Errorf("upstream error %d: %s", httpResp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, nil, fmt.Errorf("parse response: %w", err)
	}

	return &chatResp, chatResp.Usage, nil
}

func (p *OpenAIProvider) CompleteStream(ctx context.Context, req *ChatRequest, w io.Writer) (*UsageInfo, error) {
	// Inject stream_options to get usage in the final chunk
	type streamReq struct {
		*ChatRequest
		StreamOptions struct {
			IncludeUsage bool `json:"include_usage"`
		} `json:"stream_options"`
	}
	sr := streamReq{ChatRequest: req}
	sr.StreamOptions.IncludeUsage = true
	sr.Stream = true

	body, err := json.Marshal(sr)
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
		return nil, fmt.Errorf("upstream error %d: %s", httpResp.StatusCode, string(errBody))
	}

	usage := &UsageInfo{}
	scanner := bufio.NewScanner(httpResp.Body)
	scanner.Buffer(make([]byte, 1024*64), 1024*64)

	for scanner.Scan() {
		line := scanner.Text()

		// Write raw SSE line to client
		fmt.Fprintf(w, "%s\n", line)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		// Parse chunk for usage info
		var chunk struct {
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err == nil && chunk.Usage != nil {
			usage.InputTokens = chunk.Usage.PromptTokens
			usage.OutputTokens = chunk.Usage.CompletionTokens
			usage.TotalTokens = chunk.Usage.TotalTokens
		}
	}

	// Ensure DONE is forwarded
	fmt.Fprintf(w, "data: [DONE]\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return usage, fmt.Errorf("stream scan error: %w", err)
	}

	return usage, nil
}

func (p *OpenAIProvider) doRequest(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	return p.client.Do(req)
}
