package domain

import "time"

type APIKey struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	KeyHash    string     `json:"-"`          // SHA-256 of actual key, stored in DB
	KeyPrefix  string     `json:"key_prefix"` // First 8 chars for display: "sk-xxxx..."
	Name       string     `json:"name"`
	Status     string     `json:"status"` // "active" | "revoked"
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	DeletedAt  *time.Time `json:"-"`
}

func (k *APIKey) IsActive() bool { return k.Status == "active" && k.DeletedAt == nil }
