package domain

import "time"

type CreditAccount struct {
	ID          int64
	UserID      int64
	Balance     int64 // integer credits; 1 credit = 0.001 CNY
	TotalSpent  int64
	TotalTopped int64
	UpdatedAt   time.Time
}

type CreditTransaction struct {
	ID           int64
	UserID       int64
	Type         string // "topup" | "deduction" | "refund" | "admin_adjust"
	Amount       int64  // positive = added, negative = deducted
	BalanceAfter int64
	RefID        string
	Description  string
	CreatedAt    time.Time
}
