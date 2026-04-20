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

// GeminiProvider handles Google Gemini models via the Generative Language API.
type GeminiProvider struct {
	apiKey  string
	baseURL string
	models  []string
	client  *http.Client
}

func NewGeminiProvider(apiKey string, models []string) *GeminiProvider {
	return &GeminiProvider{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		models:  models,
		client:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *GeminiProvider) ModelIDs() []string { return p.models }

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent        `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
}

func (p *GeminiProvider) Complete(ctx context.Context, req *ChatRequest) (*ChatResponse, *UsageInfo, error) {
	gReq, system := p.toGeminiRequest(req)
	if system != "" {
		gReq.SystemInstruction = &geminiContent{Parts: []geminiPart{{Text: system}}}
	}

	body, err := json.Marshal(gReq)
	if err != nil {
		return nil, nil, err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, req.Model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, nil, err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, nil, err
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("gemini error %d: %s", httpResp.StatusCode, string(respBody))
	}

	var gResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(respBody, &gResp); err != nil {
		return nil, nil, err
	}

	content := ""
	if len(gResp.Candidates) > 0 && len(gResp.Candidates[0].Content.Parts) > 0 {
		content = gResp.Candidates[0].Content.Parts[0].Text
	}

	usage := &UsageInfo{
		InputTokens:  gResp.UsageMetadata.PromptTokenCount,
		OutputTokens: gResp.UsageMetadata.CandidatesTokenCount,
		TotalTokens:  gResp.UsageMetadata.TotalTokenCount,
	}

	return &ChatResponse{
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

func (p *GeminiProvider) CompleteStream(ctx context.Context, req *ChatRequest, w io.Writer) (*UsageInfo, error) {
	gReq, system := p.toGeminiRequest(req)
	if system != "" {
		gReq.SystemInstruction = &geminiContent{Parts: []geminiPart{{Text: system}}}
	}

	body, err := json.Marshal(gReq)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse", p.baseURL, req.Model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("gemini error %d: %s", httpResp.StatusCode, string(errBody))
	}

	usage := &UsageInfo{}
	scanner := bufio.NewScanner(httpResp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var chunk struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
				FinishReason string `json:"finishReason"`
			} `json:"candidates"`
			UsageMetadata struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				TotalTokenCount      int `json:"totalTokenCount"`
			} `json:"usageMetadata"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// Always update usage from latest chunk (Gemini includes it in every chunk)
		usage.InputTokens = chunk.UsageMetadata.PromptTokenCount
		usage.OutputTokens = chunk.UsageMetadata.CandidatesTokenCount
		usage.TotalTokens = chunk.UsageMetadata.TotalTokenCount

		// Forward text as OpenAI-compatible SSE chunk
		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			text := chunk.Candidates[0].Content.Parts[0].Text
			openaiChunk := map[string]any{
				"choices": []map[string]any{{
					"delta":         map[string]string{"content": text},
					"finish_reason": nil,
					"index":         0,
				}},
				"model": req.Model,
			}
			if chunkJSON, err := json.Marshal(openaiChunk); err == nil {
				fmt.Fprintf(w, "data: %s\n\n", string(chunkJSON))
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
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

func (p *GeminiProvider) toGeminiRequest(req *ChatRequest) (*geminiRequest, string) {
	gReq := &geminiRequest{}
	var system string

	for _, m := range req.Messages {
		if m.Role == "system" {
			if text, ok := m.Content.(string); ok {
				system = text
			}
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		text := ""
		if t, ok := m.Content.(string); ok {
			text = t
		}
		gReq.Contents = append(gReq.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: text}},
		})
	}

	if req.MaxTokens > 0 || req.Temperature != nil {
		gReq.GenerationConfig = &geminiGenerationConfig{}
		if req.MaxTokens > 0 {
			gReq.GenerationConfig.MaxOutputTokens = req.MaxTokens
		}
		if req.Temperature != nil {
			gReq.GenerationConfig.Temperature = *req.Temperature
		}
	}

	return gReq, system
}
