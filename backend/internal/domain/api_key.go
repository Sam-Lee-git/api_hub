package domain

import "time"

type APIKey struct {
	ID         int64
	UserID     int64
	KeyHash    string // SHA-256 of actual key, stored in DB
	KeyPrefix  string // First 8 chars for display: "sk-xxxx..."
	Name       string
	Status     string // "active" | "revoked"
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
	CreatedAt  time.Time
	DeletedAt  *time.Time
}

func (k *APIKey) IsActive() bool { return k.Status == "active" && k.DeletedAt == nil }
