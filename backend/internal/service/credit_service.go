package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/youorg/ai-proxy-platform/backend/internal/db/cache"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
)

type CreditService struct {
	creditRepo repository.CreditRepository
	cache      cache.Client
}

func NewCreditService(creditRepo repository.CreditRepository, cache cache.Client) *CreditService {
	return &CreditService{creditRepo: creditRepo, cache: cache}
}

func (s *CreditService) GetBalance(ctx context.Context, userID int64) (int64, error) {
	acc, err := s.creditRepo.GetAccount(ctx, userID)
	if err != nil || acc == nil {
		return 0, err
	}
	return acc.Balance, nil
}

func (s *CreditService) GetAccount(ctx context.Context, userID int64) (*domain.CreditAccount, error) {
	return s.creditRepo.GetAccount(ctx, userID)
}

// HasCredits returns true if the user has a positive balance.
// Uses the cache with 30s TTL to avoid DB hits on every proxy request.
func (s *CreditService) HasCredits(ctx context.Context, userID int64) (bool, error) {
	key := fmt.Sprintf("balance:%d", userID)
	if val, err := s.cache.Get(ctx, key); err == nil {
		n, _ := strconv.ParseInt(val, 10, 64)
		return n > 0, nil
	}
	balance, err := s.GetBalance(ctx, userID)
	if err != nil {
		return false, err
	}
	s.cache.Set(ctx, key, strconv.FormatInt(balance, 10), 30*time.Second)
	return balance > 0, nil
}

func (s *CreditService) DeductForUsage(ctx context.Context, userID, amount int64, requestID string) error {
	if amount <= 0 {
		return nil
	}
	_, err := s.creditRepo.DeductCredits(ctx, userID, amount, requestID)
	if err != nil {
		return err
	}
	s.invalidateCache(ctx, userID)
	return nil
}

func (s *CreditService) TopUp(ctx context.Context, userID, amount int64, orderNo string) error {
	_, err := s.creditRepo.AddCredits(ctx, userID, amount, orderNo, "topup", "Credit top-up")
	if err != nil {
		return err
	}
	s.invalidateCache(ctx, userID)
	return nil
}

func (s *CreditService) AdminAdjust(ctx context.Context, userID, amount int64, description string) error {
	if amount > 0 {
		_, err := s.creditRepo.AddCredits(ctx, userID, amount, "", "admin_adjust", description)
		if err != nil {
			return err
		}
	} else {
		_, err := s.creditRepo.DeductCredits(ctx, userID, -amount, description)
		if err != nil {
			return err
		}
	}
	s.invalidateCache(ctx, userID)
	return nil
}

func (s *CreditService) invalidateCache(ctx context.Context, userID int64) {
	s.cache.Del(ctx, fmt.Sprintf("balance:%d", userID))
}
