package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
)

type creditRepository struct{ db *sql.DB }

func NewCreditRepository(db *sql.DB) repository.CreditRepository {
	return &creditRepository{db: db}
}

func (r *creditRepository) GetAccount(ctx context.Context, userID int64) (*domain.CreditAccount, error) {
	a := &domain.CreditAccount{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, balance, total_spent, total_topped, updated_at
         FROM credit_accounts WHERE user_id = ?`, userID,
	).Scan(&a.ID, &a.UserID, &a.Balance, &a.TotalSpent, &a.TotalTopped, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// DeductCredits atomically deducts credits. Allows micro-debt (balance can go negative).
// SQLite WAL mode with a single writer makes explicit row-level locking unnecessary.
func (r *creditRepository) DeductCredits(ctx context.Context, userID, amount int64, refID string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var currentBalance int64
	if err := tx.QueryRowContext(ctx,
		`SELECT balance FROM credit_accounts WHERE user_id = ?`, userID,
	).Scan(&currentBalance); err != nil {
		return 0, fmt.Errorf("lock credit account: %w", err)
	}

	newBalance := currentBalance - amount
	if _, err := tx.ExecContext(ctx,
		`UPDATE credit_accounts SET balance = ?, total_spent = total_spent + ?, updated_at = datetime('now') WHERE user_id = ?`,
		newBalance, amount, userID,
	); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO credit_transactions (user_id, type, amount, balance_after, ref_id, description)
         VALUES (?, 'deduction', ?, ?, ?, 'API usage')`,
		userID, -amount, newBalance, refID,
	); err != nil {
		return 0, err
	}
	return newBalance, tx.Commit()
}

func (r *creditRepository) AddCredits(ctx context.Context, userID, amount int64, refID, txType, description string) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var currentBalance int64
	if err := tx.QueryRowContext(ctx,
		`SELECT balance FROM credit_accounts WHERE user_id = ?`, userID,
	).Scan(&currentBalance); err != nil {
		return 0, fmt.Errorf("lock credit account: %w", err)
	}

	newBalance := currentBalance + amount
	if _, err := tx.ExecContext(ctx,
		`UPDATE credit_accounts SET balance = ?, total_topped = total_topped + ?, updated_at = datetime('now') WHERE user_id = ?`,
		newBalance, amount, userID,
	); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO credit_transactions (user_id, type, amount, balance_after, ref_id, description)
         VALUES (?, ?, ?, ?, ?, ?)`,
		userID, txType, amount, newBalance, refID, description,
	); err != nil {
		return 0, err
	}
	return newBalance, tx.Commit()
}

func (r *creditRepository) ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]*domain.CreditTransaction, int64, error) {
	var total int64
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM credit_transactions WHERE user_id = ?`, userID).Scan(&total)

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, type, amount, balance_after, ref_id, description, created_at
         FROM credit_transactions WHERE user_id = ?
         ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txs []*domain.CreditTransaction
	for rows.Next() {
		t := &domain.CreditTransaction{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.BalanceAfter,
			&t.RefID, &t.Description, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		txs = append(txs, t)
	}
	return txs, total, rows.Err()
}
