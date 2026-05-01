package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
)

type pgUsageRepository struct {
	db *pgxpool.Pool
}

func NewUsageRepository(db *pgxpool.Pool) UsageRepository {
	return &pgUsageRepository{db: db}
}

func (r *pgUsageRepository) Create(ctx context.Context, rec *domain.UsageRecord) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO usage_records
         (user_id, api_key_id, model_id, request_id, input_tokens, output_tokens, total_tokens,
          input_credits_per_1k_snapshot, output_credits_per_1k_snapshot,
          credits_charged, status, latency_ms, error_message)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
         RETURNING id, created_at`,
		rec.UserID, rec.APIKeyID, rec.ModelID, rec.RequestID,
		rec.InputTokens, rec.OutputTokens, rec.TotalTokens,
		rec.InputCreditsPer1KSnapshot, rec.OutputCreditsPer1KSnapshot,
		rec.CreditsCharged, rec.Status, rec.LatencyMs, rec.ErrorMessage,
	).Scan(&rec.ID, &rec.CreatedAt)
}

func (r *pgUsageRepository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*domain.UsageRecord, int64, error) {
	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM usage_records WHERE user_id = $1`, userID).Scan(&total)

	rows, err := r.db.Query(ctx,
		`SELECT u.id, u.user_id, u.api_key_id, u.model_id, m.model_id as model_name,
                u.request_id, u.input_tokens, u.output_tokens, u.total_tokens,
                u.credits_charged, u.status, u.latency_ms, u.created_at
         FROM usage_records u JOIN models m ON m.id = u.model_id
         WHERE u.user_id = $1
         ORDER BY u.created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	recs, err := scanUsageRecords(rows)
	return recs, total, err
}

func (r *pgUsageRepository) List(ctx context.Context, filters UsageFilters, limit, offset int) ([]*domain.UsageRecord, int64, error) {
	args := []interface{}{}
	where := "WHERE 1=1"
	idx := 1

	if filters.UserID > 0 {
		where += fmt.Sprintf(" AND u.user_id = $%d", idx)
		args = append(args, filters.UserID)
		idx++
	}
	if !filters.From.IsZero() {
		where += fmt.Sprintf(" AND u.created_at >= $%d", idx)
		args = append(args, filters.From)
		idx++
	}
	if !filters.To.IsZero() {
		where += fmt.Sprintf(" AND u.created_at <= $%d", idx)
		args = append(args, filters.To)
		idx++
	}
	if filters.ModelName != "" {
		where += fmt.Sprintf(" AND m.model_id = $%d", idx)
		args = append(args, filters.ModelName)
		idx++
	}

	var total int64
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	r.db.QueryRow(ctx, fmt.Sprintf(
		`SELECT COUNT(*) FROM usage_records u JOIN models m ON m.id = u.model_id %s`, where),
		countArgs...).Scan(&total)

	args = append(args, limit, offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(
		`SELECT u.id, u.user_id, u.api_key_id, u.model_id, m.model_id as model_name,
                u.request_id, u.input_tokens, u.output_tokens, u.total_tokens,
                u.credits_charged, u.status, u.latency_ms, u.created_at
         FROM usage_records u JOIN models m ON m.id = u.model_id
         %s ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d`, where, idx, idx+1),
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	recs, err := scanUsageRecords(rows)
	return recs, total, err
}

func (r *pgUsageRepository) Summarize(ctx context.Context, userID int64, from, to time.Time) (*UsageSummary, error) {
	return r.summarize(ctx, userID, from, to)
}

func (r *pgUsageRepository) GlobalStats(ctx context.Context, from, to time.Time) (*UsageSummary, error) {
	return r.summarize(ctx, 0, from, to)
}

func (r *pgUsageRepository) summarize(ctx context.Context, userID int64, from, to time.Time) (*UsageSummary, error) {
	s := &UsageSummary{}
	userFilter := ""
	args := []interface{}{from, to}
	if userID > 0 {
		userFilter = "AND user_id = $3"
		args = append(args, userID)
	}

	r.db.QueryRow(ctx, fmt.Sprintf(
		`SELECT COUNT(*), COALESCE(SUM(total_tokens),0), COALESCE(SUM(credits_charged),0)
         FROM usage_records WHERE created_at BETWEEN $1 AND $2 %s`, userFilter),
		args...).Scan(&s.TotalCalls, &s.TotalTokens, &s.TotalCredits)

	return s, nil
}

func scanUsageRecords(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]*domain.UsageRecord, error) {
	var recs []*domain.UsageRecord
	for rows.Next() {
		r := &domain.UsageRecord{}
		if err := rows.Scan(&r.ID, &r.UserID, &r.APIKeyID, &r.ModelID, &r.ModelName,
			&r.RequestID, &r.InputTokens, &r.OutputTokens, &r.TotalTokens,
			&r.CreditsCharged, &r.Status, &r.LatencyMs, &r.CreatedAt); err != nil {
			return nil, err
		}
		recs = append(recs, r)
	}
	return recs, rows.Err()
}
