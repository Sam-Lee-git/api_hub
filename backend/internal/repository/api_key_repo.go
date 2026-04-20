package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
)

type pgAPIKeyRepository struct {
	db *pgxpool.Pool
}

func NewAPIKeyRepository(db *pgxpool.Pool) APIKeyRepository {
	return &pgAPIKeyRepository{db: db}
}

func (r *pgAPIKeyRepository) Create(ctx context.Context, key *domain.APIKey) error {
	row := r.db.QueryRow(ctx,
		`INSERT INTO api_keys (user_id, key_hash, key_prefix, name, status)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, created_at`,
		key.UserID, key.KeyHash, key.KeyPrefix, key.Name, key.Status,
	)
	return row.Scan(&key.ID, &key.CreatedAt)
}

func (r *pgAPIKeyRepository) FindByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	k := &domain.APIKey{}
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, key_hash, key_prefix, name, status, last_used_at, expires_at, created_at
         FROM api_keys WHERE key_hash = $1 AND deleted_at IS NULL`,
		hash,
	).Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Status,
		&k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return k, err
}

func (r *pgAPIKeyRepository) ListByUser(ctx context.Context, userID int64) ([]*domain.APIKey, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, key_prefix, name, status, last_used_at, created_at
         FROM api_keys WHERE user_id = $1 AND deleted_at IS NULL
         ORDER BY created_at DESC`,
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

func (r *pgAPIKeyRepository) Revoke(ctx context.Context, id, userID int64) error {
	result, err := r.db.Exec(ctx,
		`UPDATE api_keys SET status = 'revoked', deleted_at = NOW() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		id, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("api key not found")
	}
	return nil
}

func (r *pgAPIKeyRepository) UpdateLastUsed(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}
