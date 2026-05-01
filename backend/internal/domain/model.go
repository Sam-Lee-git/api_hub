package domain

import "time"

type Provider struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"` // "openai" | "anthropic" | "google" | "alibaba"
	BaseURL   string    `json:"base_url"`
	APIKey    string    `json:"api_key"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Model struct {
	ID                 int       `json:"id"`
	ProviderID         int       `json:"provider_id"`
	ProviderName       string    `json:"provider_name"`
	ModelID            string    `json:"model_id"` // "gpt-4o", "claude-3-5-sonnet-20241022"
	DisplayName        string    `json:"display_name"`
	InputCreditsPer1K  int64     `json:"input_credits_per_1k"`
	OutputCreditsPer1K int64     `json:"output_credits_per_1k"`
	ContextWindow      int       `json:"context_window"`
	SupportsStreaming  bool      `json:"supports_streaming"`
	SupportsVision     bool      `json:"supports_vision"`
	Status             string    `json:"status"` // "active" | "hidden" | "disabled"
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (m *Model) CalculateCost(inputTokens, outputTokens int) int64 {
	input := ceilCredits(int64(inputTokens), m.InputCreditsPer1K)
	output := ceilCredits(int64(outputTokens), m.OutputCreditsPer1K)
	return input + output
}

func ceilCredits(tokens, creditsPer1K int64) int64 {
	if tokens <= 0 || creditsPer1K <= 0 {
		return 0
	}
	return (tokens*creditsPer1K + 999) / 1000
}
