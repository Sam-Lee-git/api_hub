package domain

import "time"

type PaymentOrder struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	OrderNo         string     `json:"order_no"`
	Channel         string     `json:"channel"`    // "alipay" | "wechat"
	AmountCNY       int64      `json:"amount_cny"` // in fen (CNY * 100)
	CreditsToAdd    int64      `json:"credits_to_add"`
	Status          string     `json:"status"` // "pending" | "paid" | "failed" | "refunded"
	ProviderOrderNo string     `json:"provider_order_no"`
	PaidAt          *time.Time `json:"paid_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	Metadata        []byte     `json:"metadata"` // raw JSON from payment provider
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CreditPackage struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	AmountCNY    int64     `json:"amount_cny"` // in fen
	Credits      int64     `json:"credits"`
	BonusCredits int64     `json:"bonus_credits"`
	IsActive     bool      `json:"is_active"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
}
