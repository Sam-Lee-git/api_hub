package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
)

type pgUserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &pgUserRepository{db: db}
}

func (r *pgUserRepository) Create(ctx context.Context, u *domain.User) error {
	row := r.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, display_name, role, status)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, created_at, updated_at`,
		u.Email, u.PasswordHash, u.DisplayName, u.Role, u.Status,
	)
	return row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *pgUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u := &domain.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, role, status, created_at, updated_at, deleted_at
         FROM users WHERE email = $1 AND deleted_at IS NULL`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Status,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *pgUserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	u := &domain.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, role, status, created_at, updated_at, deleted_at
         FROM users WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Status,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *pgUserRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id)
	return err
}

func (r *pgUserRepository) UpdateProfile(ctx context.Context, id int64, displayName, passwordHash string) error {
	if passwordHash != "" {
		_, err := r.db.Exec(ctx,
			`UPDATE users SET display_name = $1, password_hash = $2, updated_at = NOW() WHERE id = $3`,
			displayName, passwordHash, id)
		return err
	}
	_, err := r.db.Exec(ctx,
		`UPDATE users SET display_name = $1, updated_at = NOW() WHERE id = $2`,
		displayName, id)
	return err
}

func (r *pgUserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, email, display_name, role, status, created_at, updated_at
         FROM users WHERE deleted_at IS NULL
         ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// CreateWithCreditAccount creates user + credit_account in a single transaction.
func CreateWithCreditAccount(ctx context.Context, db *pgxpool.Pool, u *domain.User) error {
	return pgx.BeginTxFunc(ctx, db, pgx.TxOptions{}, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx,
			`INSERT INTO users (email, password_hash, display_name, role, status)
             VALUES ($1, $2, $3, $4, $5)
             RETURNING id, created_at, updated_at`,
			u.Email, u.PasswordHash, u.DisplayName, u.Role, u.Status,
		)
		if err := row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO credit_accounts (user_id, balance) VALUES ($1, 0)`,
			u.ID)
		return err
	})
}

// ExistsEmail returns true if the email is already registered.
func ExistsEmail(ctx context.Context, db *pgxpool.Pool, email string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`,
		email,
	).Scan(&exists)
	return exists, err
}

// StoreRefreshToken inserts a refresh token record.
func StoreRefreshToken(ctx context.Context, db *pgxpool.Pool, userID int64, tokenHash string, expiresAt time.Time) error {
	_, err := db.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt)
	return err
}

// ValidateRefreshToken looks up and validates a refresh token, returns userID.
func ValidateRefreshToken(ctx context.Context, db *pgxpool.Pool, tokenHash string) (int64, error) {
	var userID int64
	var revoked bool
	var expiresAt time.Time
	err := db.QueryRow(ctx,
		`SELECT user_id, revoked, expires_at FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&userID, &revoked, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errors.New("token not found")
	}
	if err != nil {
		return 0, err
	}
	if revoked {
		return 0, errors.New("token revoked")
	}
	if time.Now().After(expiresAt) {
		return 0, errors.New("token expired")
	}
	return userID, nil
}

// RevokeRefreshToken marks a refresh token as revoked.
func RevokeRefreshToken(ctx context.Context, db *pgxpool.Pool, tokenHash string) error {
	_, err := db.Exec(ctx, `UPDATE refresh_tokens SET revoked = TRUE WHERE token_hash = $1`, tokenHash)
	return err
}
