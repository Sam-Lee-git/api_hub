package domain

import "time"

type Provider struct {
	ID        int
	Name      string // "openai" | "anthropic" | "google" | "alibaba"
	BaseURL   string
	APIKey    string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Model struct {
	ID                   int
	ProviderID           int
	ProviderName         string
	ModelID              string // "gpt-4o", "claude-3-5-sonnet-20241022"
	DisplayName          string
	InputCreditsPer1K    int64
	OutputCreditsPer1K   int64
	ContextWindow        int
	SupportsStreaming     bool
	SupportsVision       bool
	Status               string // "active" | "hidden" | "disabled"
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (m *Model) CalculateCost(inputTokens, outputTokens int) int64 {
	input := int64(inputTokens) * m.InputCreditsPer1K / 1000
	output := int64(outputTokens) * m.OutputCreditsPer1K / 1000
	return input + output
}
