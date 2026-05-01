package repository

import (
	"context"
	"time"

	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id int64) (*domain.User, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateProfile(ctx context.Context, id int64, displayName, passwordHash string) error
	List(ctx context.Context, limit, offset int) ([]*domain.User, int64, error)
	// Transaction helpers — implemented by both pgx and SQLite backends.
	ExistsEmail(ctx context.Context, email string) (bool, error)
	CreateWithCreditAccount(ctx context.Context, u *domain.User) error
	StoreRefreshToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	ValidateRefreshToken(ctx context.Context, tokenHash string) (int64, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
}

type APIKeyRepository interface {
	Create(ctx context.Context, key *domain.APIKey) error
	FindByHash(ctx context.Context, hash string) (*domain.APIKey, error)
	ListByUser(ctx context.Context, userID int64) ([]*domain.APIKey, error)
	Revoke(ctx context.Context, id, userID int64) error
	UpdateLastUsed(ctx context.Context, id int64) error
}

type CreditRepository interface {
	GetAccount(ctx context.Context, userID int64) (*domain.CreditAccount, error)
	DeductCredits(ctx context.Context, userID, amount int64, refID string) (int64, error)
	AddCredits(ctx context.Context, userID, amount int64, refID, txType, description string) (int64, error)
	ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]*domain.CreditTransaction, int64, error)
}

type UsageFilters struct {
	UserID    int64
	ModelName string
	From      time.Time
	To        time.Time
}

type UsageSummary struct {
	TotalCalls   int64
	TotalTokens  int64
	TotalCredits int64
	ByDay        []DaySummary
	ByModel      []ModelSummary
}

type DaySummary struct {
	Date         time.Time
	Calls        int64
	InputTokens  int64
	OutputTokens int64
	Credits      int64
}

type ModelSummary struct {
	ModelName    string
	Calls        int64
	InputTokens  int64
	OutputTokens int64
	Credits      int64
}

type ModelRepository interface {
	FindByModelID(ctx context.Context, modelID string) (*domain.Model, error)
	ListActive(ctx context.Context) ([]*domain.Model, error)
	List(ctx context.Context) ([]*domain.Model, error)
	Create(ctx context.Context, m *domain.Model) error
	Update(ctx context.Context, m *domain.Model) error
	FindByID(ctx context.Context, id int) (*domain.Model, error)
}

type ProviderRepository interface {
	List(ctx context.Context) ([]*domain.Provider, error)
	FindByName(ctx context.Context, name string) (*domain.Provider, error)
	Update(ctx context.Context, p *domain.Provider) error
}

type UsageRepository interface {
	Create(ctx context.Context, r *domain.UsageRecord) error
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*domain.UsageRecord, int64, error)
	List(ctx context.Context, filters UsageFilters, limit, offset int) ([]*domain.UsageRecord, int64, error)
	Summarize(ctx context.Context, userID int64, from, to time.Time) (*UsageSummary, error)
	GlobalStats(ctx context.Context, from, to time.Time) (*UsageSummary, error)
}

type PaymentRepository interface {
	CreateOrder(ctx context.Context, o *domain.PaymentOrder) error
	FindByOrderNo(ctx context.Context, orderNo string) (*domain.PaymentOrder, error)
	MarkPaid(ctx context.Context, orderNo, providerOrderNo string) error
	FulfillPaidOrder(ctx context.Context, orderNo, providerOrderNo string) (*domain.PaymentOrder, bool, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*domain.PaymentOrder, int64, error)
	ListAll(ctx context.Context, limit, offset int) ([]*domain.PaymentOrder, int64, error)
	ListPackages(ctx context.Context) ([]*domain.CreditPackage, error)
	FindPackageByID(ctx context.Context, id int) (*domain.CreditPackage, error)
}
