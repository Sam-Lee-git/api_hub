package service

import (
	"context"
	"errors"
	"time"

	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
	"github.com/youorg/ai-proxy-platform/backend/pkg/crypto"
	"github.com/youorg/ai-proxy-platform/backend/pkg/jwt"
)

var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserSuspended      = errors.New("account suspended")
)

type AuthService struct {
	userRepo         repository.UserRepository
	jwtAccessSecret  string
	jwtRefreshSecret string
}

func NewAuthService(userRepo repository.UserRepository, accessSecret, refreshSecret string) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		jwtAccessSecret:  accessSecret,
		jwtRefreshSecret: refreshSecret,
	}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds
}

func (s *AuthService) Register(ctx context.Context, email, password, displayName string) (*domain.User, error) {
	exists, err := s.userRepo.ExistsEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailExists
	}

	hash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u := &domain.User{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		Role:         "user",
		Status:       "active",
	}

	if err := s.userRepo.CreateWithCreditAccount(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, *TokenPair, error) {
	u, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if u == nil || !crypto.CheckPassword(u.PasswordHash, password) {
		return nil, nil, ErrInvalidCredentials
	}
	if !u.IsActive() {
		return nil, nil, ErrUserSuspended
	}

	tokens, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, nil, err
	}
	return u, tokens, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	tokenHash := crypto.SHA256Hex(refreshToken)
	userID, err := s.userRepo.ValidateRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || u == nil {
		return nil, errors.New("user not found")
	}
	if !u.IsActive() {
		return nil, ErrUserSuspended
	}

	if err := s.userRepo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, u)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := crypto.SHA256Hex(refreshToken)
	return s.userRepo.RevokeRefreshToken(ctx, tokenHash)
}

func (s *AuthService) issueTokens(ctx context.Context, u *domain.User) (*TokenPair, error) {
	accessToken, err := jwt.Sign(u.ID, u.Role, s.jwtAccessSecret, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := crypto.GenerateToken(32)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	tokenHash := crypto.SHA256Hex(rawRefresh)
	if err := s.userRepo.StoreRefreshToken(ctx, u.ID, tokenHash, expiresAt); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    900,
	}, nil
}
