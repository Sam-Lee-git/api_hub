package domain

import "time"

type PaymentOrder struct {
	ID              int64
	UserID          int64
	OrderNo         string
	Channel         string // "alipay" | "wechat"
	AmountCNY       int64  // in fen (CNY * 100)
	CreditsToAdd    int64
	Status          string // "pending" | "paid" | "failed" | "refunded"
	ProviderOrderNo string
	PaidAt          *time.Time
	ExpiresAt       time.Time
	Metadata        []byte // raw JSON from payment provider
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreditPackage struct {
	ID           int
	Name         string
	AmountCNY    int64 // in fen
	Credits      int64
	BonusCredits int64
	IsActive     bool
	DisplayOrder int
	CreatedAt    time.Time
}
