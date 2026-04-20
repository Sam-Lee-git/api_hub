package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
)

type pgProviderRepository struct {
	db *pgxpool.Pool
}

func NewProviderRepository(db *pgxpool.Pool) ProviderRepository {
	return &pgProviderRepository{db: db}
}

func (r *pgProviderRepository) List(ctx context.Context) ([]*domain.Provider, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, base_url, api_key, status, created_at, updated_at FROM providers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*domain.Provider
	for rows.Next() {
		p := &domain.Provider{}
		if err := rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

func (r *pgProviderRepository) FindByName(ctx context.Context, name string) (*domain.Provider, error) {
	p := &domain.Provider{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, base_url, api_key, status, created_at, updated_at FROM providers WHERE name = $1`,
		name,
	).Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

func (r *pgProviderRepository) Update(ctx context.Context, p *domain.Provider) error {
	_, err := r.db.Exec(ctx,
		`UPDATE providers SET base_url=$1, api_key=$2, status=$3, updated_at=NOW() WHERE id=$4`,
		p.BaseURL, p.APIKey, p.Status, p.ID)
	return err
}
