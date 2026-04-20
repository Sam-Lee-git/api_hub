package domain

import "time"

type UsageRecord struct {
	ID             int64
	UserID         int64
	APIKeyID       int64
	ModelID        int
	ModelName      string
	RequestID      string
	InputTokens    int
	OutputTokens   int
	TotalTokens    int
	CreditsCharged int64
	Status         string // "success" | "error" | "cancelled"
	LatencyMs      int
	ErrorMessage   string
	CreatedAt      time.Time
}
