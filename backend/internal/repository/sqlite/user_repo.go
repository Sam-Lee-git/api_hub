package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
)

type userRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, display_name, role, status) VALUES (?,?,?,?,?)`,
		u.Email, u.PasswordHash, u.DisplayName, u.Role, u.Status,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = id
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	return nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, display_name, role, status, created_at, updated_at, deleted_at
         FROM users WHERE email = ? AND deleted_at IS NULL`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Status,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *userRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, display_name, role, status, created_at, updated_at, deleted_at
         FROM users WHERE id = ? AND deleted_at IS NULL`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Status,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *userRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET status = ?, updated_at = datetime('now') WHERE id = ?`, status, id)
	return err
}

func (r *userRepository) UpdateProfile(ctx context.Context, id int64, displayName, passwordHash string) error {
	if passwordHash != "" {
		_, err := r.db.ExecContext(ctx,
			`UPDATE users SET display_name = ?, password_hash = ?, updated_at = datetime('now') WHERE id = ?`,
			displayName, passwordHash, id)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = ?, updated_at = datetime('now') WHERE id = ?`,
		displayName, id)
	return err
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, display_name, role, status, created_at, updated_at
         FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.Status,
			&u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *userRepository) ExistsEmail(ctx context.Context, email string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM users WHERE email = ? AND deleted_at IS NULL`, email,
	).Scan(&n)
	return n > 0, err
}

func (r *userRepository) CreateWithCreditAccount(ctx context.Context, u *domain.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, display_name, role, status) VALUES (?,?,?,?,?)`,
		u.Email, u.PasswordHash, u.DisplayName, u.Role, u.Status,
	)
	if err != nil {
		return err
	}
	u.ID, err = res.LastInsertId()
	if err != nil {
		return err
	}
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	_, err = tx.ExecContext(ctx, `INSERT INTO credit_accounts (user_id, balance) VALUES (?, 0)`, u.ID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *userRepository) StoreRefreshToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?,?,?)`,
		userID, tokenHash, expiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *userRepository) ValidateRefreshToken(ctx context.Context, tokenHash string) (int64, error) {
	var userID int64
	var revoked int
	var expiresAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, revoked, expires_at FROM refresh_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&userID, &revoked, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errors.New("token not found")
	}
	if err != nil {
		return 0, err
	}
	if revoked != 0 {
		return 0, errors.New("token revoked")
	}
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, errors.New("invalid token expiry")
	}
	if time.Now().After(exp) {
		return 0, errors.New("token expired")
	}
	return userID, nil
}

func (r *userRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked = 1 WHERE token_hash = ?`, tokenHash)
	return err
}
