package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
)

type providerRepository struct{ db *sql.DB }

func NewProviderRepository(db *sql.DB) repository.ProviderRepository {
	return &providerRepository{db: db}
}

func (r *providerRepository) List(ctx context.Context) ([]*domain.Provider, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, base_url, api_key, status, created_at, updated_at FROM providers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*domain.Provider
	for rows.Next() {
		p := &domain.Provider{}
		var createdAt, updatedAt sqlTime
		if err := rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Status,
			&createdAt, &updatedAt); err != nil {
			return nil, err
		}
		p.CreatedAt = createdAt.T
		p.UpdatedAt = updatedAt.T
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

func (r *providerRepository) FindByName(ctx context.Context, name string) (*domain.Provider, error) {
	p := &domain.Provider{}
	var createdAt, updatedAt sqlTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, base_url, api_key, status, created_at, updated_at FROM providers WHERE name = ?`,
		name,
	).Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Status, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.CreatedAt = createdAt.T
	p.UpdatedAt = updatedAt.T
	return p, nil
}

func (r *providerRepository) Update(ctx context.Context, p *domain.Provider) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE providers SET base_url=?, api_key=?, status=?, updated_at=datetime('now') WHERE id=?`,
		p.BaseURL, p.APIKey, p.Status, p.ID)
	return err
}
