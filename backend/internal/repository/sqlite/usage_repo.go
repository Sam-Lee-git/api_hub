package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
)

type usageRepository struct{ db *sql.DB }

func NewUsageRepository(db *sql.DB) repository.UsageRepository {
	return &usageRepository{db: db}
}

func (r *usageRepository) Create(ctx context.Context, rec *domain.UsageRecord) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO usage_records
         (user_id, api_key_id, model_id, request_id, input_tokens, output_tokens, total_tokens,
          credits_charged, status, latency_ms, error_message)
         VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		rec.UserID, rec.APIKeyID, rec.ModelID, rec.RequestID,
		rec.InputTokens, rec.OutputTokens, rec.TotalTokens,
		rec.CreditsCharged, rec.Status, rec.LatencyMs, rec.ErrorMessage,
	)
	if err != nil {
		return err
	}
	rec.ID, err = res.LastInsertId()
	rec.CreatedAt = time.Now()
	return err
}

func (r *usageRepository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*domain.UsageRecord, int64, error) {
	var total int64
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM usage_records WHERE user_id = ?`, userID).Scan(&total)

	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.user_id, u.api_key_id, u.model_id, m.model_id,
                u.request_id, u.input_tokens, u.output_tokens, u.total_tokens,
                u.credits_charged, u.status, u.latency_ms, u.created_at
         FROM usage_records u JOIN models m ON m.id = u.model_id
         WHERE u.user_id = ?
         ORDER BY u.created_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	recs, err := scanUsageRecords(rows)
	return recs, total, err
}

func (r *usageRepository) List(ctx context.Context, filters repository.UsageFilters, limit, offset int) ([]*domain.UsageRecord, int64, error) {
	conds := []string{"1=1"}
	args := []interface{}{}

	if filters.UserID > 0 {
		conds = append(conds, "u.user_id = ?")
		args = append(args, filters.UserID)
	}
	if !filters.From.IsZero() {
		conds = append(conds, "u.created_at >= ?")
		args = append(args, filters.From.UTC().Format(time.RFC3339))
	}
	if !filters.To.IsZero() {
		conds = append(conds, "u.created_at <= ?")
		args = append(args, filters.To.UTC().Format(time.RFC3339))
	}
	if filters.ModelName != "" {
		conds = append(conds, "m.model_id = ?")
		args = append(args, filters.ModelName)
	}

	where := "WHERE " + strings.Join(conds, " AND ")
	base := fmt.Sprintf(`FROM usage_records u JOIN models m ON m.id = u.model_id %s`, where)

	var total int64
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) "+base, args...).Scan(&total)

	rows, err := r.db.QueryContext(ctx,
		`SELECT u.id, u.user_id, u.api_key_id, u.model_id, m.model_id,
                u.request_id, u.input_tokens, u.output_tokens, u.total_tokens,
                u.credits_charged, u.status, u.latency_ms, u.created_at `+
			base+` ORDER BY u.created_at DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	recs, err := scanUsageRecords(rows)
	return recs, total, err
}

func (r *usageRepository) Summarize(ctx context.Context, userID int64, from, to time.Time) (*repository.UsageSummary, error) {
	return r.summarize(ctx, userID, from, to)
}

func (r *usageRepository) GlobalStats(ctx context.Context, from, to time.Time) (*repository.UsageSummary, error) {
	return r.summarize(ctx, 0, from, to)
}

func (r *usageRepository) summarize(ctx context.Context, userID int64, from, to time.Time) (*repository.UsageSummary, error) {
	s := &repository.UsageSummary{}
	fromStr := from.UTC().Format(time.RFC3339)
	toStr := to.UTC().Format(time.RFC3339)

	var args []interface{}
	userFilter := ""
	if userID > 0 {
		userFilter = " AND user_id = ?"
		args = append(args, fromStr, toStr, userID)
	} else {
		args = append(args, fromStr, toStr)
	}

	r.db.QueryRowContext(ctx, fmt.Sprintf(
		`SELECT COUNT(*), COALESCE(SUM(total_tokens),0), COALESCE(SUM(credits_charged),0)
         FROM usage_records WHERE created_at BETWEEN ? AND ?%s`, userFilter),
		args...).Scan(&s.TotalCalls, &s.TotalTokens, &s.TotalCredits)
	return s, nil
}

func scanUsageRecords(rows *sql.Rows) ([]*domain.UsageRecord, error) {
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
