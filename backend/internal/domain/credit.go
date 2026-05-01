package domain

import "time"

type CreditAccount struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Balance     int64     `json:"balance"` // integer credits; 1 credit = 0.001 CNY
	TotalSpent  int64     `json:"total_spent"`
	TotalTopped int64     `json:"total_topped"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreditTransaction struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Type         string    `json:"type"`   // "topup" | "deduction" | "refund" | "admin_adjust"
	Amount       int64     `json:"amount"` // positive = added, negative = deducted
	BalanceAfter int64     `json:"balance_after"`
	RefID        string    `json:"ref_id"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}
