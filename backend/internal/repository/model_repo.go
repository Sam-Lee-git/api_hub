package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
)

type pgModelRepository struct {
	db *pgxpool.Pool
}

func NewModelRepository(db *pgxpool.Pool) ModelRepository {
	return &pgModelRepository{db: db}
}

func (r *pgModelRepository) FindByModelID(ctx context.Context, modelID string) (*domain.Model, error) {
	m := &domain.Model{}
	err := r.db.QueryRow(ctx,
		`SELECT m.id, m.provider_id, p.name, m.model_id, m.display_name,
                m.input_credits_per_1k, m.output_credits_per_1k, m.context_window,
                m.supports_streaming, m.supports_vision, m.status
         FROM models m JOIN providers p ON p.id = m.provider_id
         WHERE m.model_id = $1`,
		modelID,
	).Scan(&m.ID, &m.ProviderID, &m.ProviderName, &m.ModelID, &m.DisplayName,
		&m.InputCreditsPer1K, &m.OutputCreditsPer1K, &m.ContextWindow,
		&m.SupportsStreaming, &m.SupportsVision, &m.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *pgModelRepository) ListActive(ctx context.Context) ([]*domain.Model, error) {
	rows, err := r.db.Query(ctx,
		`SELECT m.id, m.provider_id, p.name, m.model_id, m.display_name,
                m.input_credits_per_1k, m.output_credits_per_1k, m.context_window,
                m.supports_streaming, m.supports_vision, m.status
         FROM models m JOIN providers p ON p.id = m.provider_id
         WHERE m.status = 'active'
         ORDER BY m.provider_id, m.id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanModels(rows)
}

func (r *pgModelRepository) List(ctx context.Context) ([]*domain.Model, error) {
	rows, err := r.db.Query(ctx,
		`SELECT m.id, m.provider_id, p.name, m.model_id, m.display_name,
                m.input_credits_per_1k, m.output_credits_per_1k, m.context_window,
                m.supports_streaming, m.supports_vision, m.status
         FROM models m JOIN providers p ON p.id = m.provider_id
         ORDER BY m.provider_id, m.id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanModels(rows)
}

func (r *pgModelRepository) FindByID(ctx context.Context, id int) (*domain.Model, error) {
	m := &domain.Model{}
	err := r.db.QueryRow(ctx,
		`SELECT m.id, m.provider_id, p.name, m.model_id, m.display_name,
                m.input_credits_per_1k, m.output_credits_per_1k, m.context_window,
                m.supports_streaming, m.supports_vision, m.status
         FROM models m JOIN providers p ON p.id = m.provider_id
         WHERE m.id = $1`,
		id,
	).Scan(&m.ID, &m.ProviderID, &m.ProviderName, &m.ModelID, &m.DisplayName,
		&m.InputCreditsPer1K, &m.OutputCreditsPer1K, &m.ContextWindow,
		&m.SupportsStreaming, &m.SupportsVision, &m.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *pgModelRepository) Create(ctx context.Context, m *domain.Model) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO models (provider_id, model_id, display_name, input_credits_per_1k, output_credits_per_1k,
                             context_window, supports_streaming, supports_vision, status)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at`,
		m.ProviderID, m.ModelID, m.DisplayName, m.InputCreditsPer1K, m.OutputCreditsPer1K,
		m.ContextWindow, m.SupportsStreaming, m.SupportsVision, m.Status,
	).Scan(&m.ID, &m.CreatedAt)
}

func (r *pgModelRepository) Update(ctx context.Context, m *domain.Model) error {
	_, err := r.db.Exec(ctx,
		`UPDATE models SET display_name=$1, input_credits_per_1k=$2, output_credits_per_1k=$3,
                           context_window=$4, supports_streaming=$5, supports_vision=$6, status=$7, updated_at=NOW()
         WHERE id=$8`,
		m.DisplayName, m.InputCreditsPer1K, m.OutputCreditsPer1K,
		m.ContextWindow, m.SupportsStreaming, m.SupportsVision, m.Status, m.ID)
	return err
}

func scanModels(rows pgx.Rows) ([]*domain.Model, error) {
	var models []*domain.Model
	for rows.Next() {
		m := &domain.Model{}
		if err := rows.Scan(&m.ID, &m.ProviderID, &m.ProviderName, &m.ModelID, &m.DisplayName,
			&m.InputCreditsPer1K, &m.OutputCreditsPer1K, &m.ContextWindow,
			&m.SupportsStreaming, &m.SupportsVision, &m.Status); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		models = append(models, m)
	}
	return models, rows.Err()
}
