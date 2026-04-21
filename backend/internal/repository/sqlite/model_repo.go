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

type modelRepository struct{ db *sql.DB }

func NewModelRepository(db *sql.DB) repository.ModelRepository {
	return &modelRepository{db: db}
}

const modelSelectCols = `
    m.id, m.provider_id, p.name, m.model_id, m.display_name,
    m.input_credits_per_1k, m.output_credits_per_1k, m.context_window,
    m.supports_streaming, m.supports_vision, m.status`

func scanModel(row *sql.Row) (*domain.Model, error) {
	m := &domain.Model{}
	var streaming, vision int
	err := row.Scan(&m.ID, &m.ProviderID, &m.ProviderName, &m.ModelID, &m.DisplayName,
		&m.InputCreditsPer1K, &m.OutputCreditsPer1K, &m.ContextWindow,
		&streaming, &vision, &m.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.SupportsStreaming = streaming != 0
	m.SupportsVision = vision != 0
	return m, nil
}

func scanModels(rows *sql.Rows) ([]*domain.Model, error) {
	var models []*domain.Model
	for rows.Next() {
		m := &domain.Model{}
		var streaming, vision int
		if err := rows.Scan(&m.ID, &m.ProviderID, &m.ProviderName, &m.ModelID, &m.DisplayName,
			&m.InputCreditsPer1K, &m.OutputCreditsPer1K, &m.ContextWindow,
			&streaming, &vision, &m.Status); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		m.SupportsStreaming = streaming != 0
		m.SupportsVision = vision != 0
		models = append(models, m)
	}
	return models, rows.Err()
}

func (r *modelRepository) FindByModelID(ctx context.Context, modelID string) (*domain.Model, error) {
	return scanModel(r.db.QueryRowContext(ctx,
		`SELECT`+modelSelectCols+`FROM models m JOIN providers p ON p.id = m.provider_id WHERE m.model_id = ?`,
		modelID,
	))
}

func (r *modelRepository) FindByID(ctx context.Context, id int) (*domain.Model, error) {
	return scanModel(r.db.QueryRowContext(ctx,
		`SELECT`+modelSelectCols+`FROM models m JOIN providers p ON p.id = m.provider_id WHERE m.id = ?`,
		id,
	))
}

func (r *modelRepository) ListActive(ctx context.Context) ([]*domain.Model, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT`+modelSelectCols+`FROM models m JOIN providers p ON p.id = m.provider_id WHERE m.status = 'active' ORDER BY m.provider_id, m.id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanModels(rows)
}

func (r *modelRepository) List(ctx context.Context) ([]*domain.Model, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT`+modelSelectCols+`FROM models m JOIN providers p ON p.id = m.provider_id ORDER BY m.provider_id, m.id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanModels(rows)
}

func (r *modelRepository) Create(ctx context.Context, m *domain.Model) error {
	var streaming, vision int
	if m.SupportsStreaming {
		streaming = 1
	}
	if m.SupportsVision {
		vision = 1
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO models (provider_id, model_id, display_name, input_credits_per_1k, output_credits_per_1k,
                             context_window, supports_streaming, supports_vision, status)
         VALUES (?,?,?,?,?,?,?,?,?)`,
		m.ProviderID, m.ModelID, m.DisplayName, m.InputCreditsPer1K, m.OutputCreditsPer1K,
		m.ContextWindow, streaming, vision, m.Status,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = int(id)
	m.CreatedAt = time.Now()
	return nil
}

func (r *modelRepository) Update(ctx context.Context, m *domain.Model) error {
	var streaming, vision int
	if m.SupportsStreaming {
		streaming = 1
	}
	if m.SupportsVision {
		vision = 1
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE models SET display_name=?, input_credits_per_1k=?, output_credits_per_1k=?,
                           context_window=?, supports_streaming=?, supports_vision=?, status=?, updated_at=datetime('now')
         WHERE id=?`,
		m.DisplayName, m.InputCreditsPer1K, m.OutputCreditsPer1K,
		m.ContextWindow, streaming, vision, m.Status, m.ID)
	return err
}
