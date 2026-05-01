package domain

import "time"

type UsageRecord struct {
	ID                         int64     `json:"id"`
	UserID                     int64     `json:"user_id"`
	APIKeyID                   int64     `json:"api_key_id"`
	ModelID                    int       `json:"model_id"`
	ModelName                  string    `json:"model_name"`
	RequestID                  string    `json:"request_id"`
	InputTokens                int       `json:"input_tokens"`
	OutputTokens               int       `json:"output_tokens"`
	TotalTokens                int       `json:"total_tokens"`
	InputCreditsPer1KSnapshot  int64     `json:"input_credits_per_1k_snapshot"`
	OutputCreditsPer1KSnapshot int64     `json:"output_credits_per_1k_snapshot"`
	CreditsCharged             int64     `json:"credits_charged"`
	Status                     string    `json:"status"` // "success" | "error" | "cancelled"
	LatencyMs                  int       `json:"latency_ms"`
	ErrorMessage               string    `json:"error_message"`
	CreatedAt                  time.Time `json:"created_at"`
}
