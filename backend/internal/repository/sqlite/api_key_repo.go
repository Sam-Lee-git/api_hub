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

type apiKeyRepository struct{ db *sql.DB }

func NewAPIKeyRepository(db *sql.DB) repository.APIKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) Create(ctx context.Context, key *domain.APIKey) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO api_keys (user_id, key_hash, key_prefix, name, status) VALUES (?,?,?,?,?)`,
		key.UserID, key.KeyHash, key.KeyPrefix, key.Name, key.Status,
	)
	if err != nil {
		return err
	}
	key.ID, err = res.LastInsertId()
	key.CreatedAt = time.Now()
	return err
}

func (r *apiKeyRepository) FindByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	k := &domain.APIKey{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, key_hash, key_prefix, name, status, last_used_at, expires_at, created_at
         FROM api_keys WHERE key_hash = ? AND deleted_at IS NULL`,
		hash,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Status,
		&k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return k, err
}

func (r *apiKeyRepository) ListByUser(ctx context.Context, userID int64) ([]*domain.APIKey, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, key_prefix, name, status, last_used_at, created_at
         FROM api_keys WHERE user_id = ? AND deleted_at IS NULL ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		k := &domain.APIKey{}
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyPrefix, &k.Name, &k.Status,
			&k.LastUsedAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *apiKeyRepository) Revoke(ctx context.Context, id, userID int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE api_keys SET status = 'revoked', deleted_at = datetime('now')
         WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		id, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("api key not found")
	}
	return nil
}

func (r *apiKeyRepository) UpdateLastUsed(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE api_keys SET last_used_at = datetime('now') WHERE id = ?`, id)
	return err
}
