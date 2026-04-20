package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
)

type pgCreditRepository struct {
	db *pgxpool.Pool
}

func NewCreditRepository(db *pgxpool.Pool) CreditRepository {
	return &pgCreditRepository{db: db}
}

func (r *pgCreditRepository) GetAccount(ctx context.Context, userID int64) (*domain.CreditAccount, error) {
	a := &domain.CreditAccount{}
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, balance, total_spent, total_topped, updated_at
         FROM credit_accounts WHERE user_id = $1`,
		userID,
	).Scan(&a.ID, &a.UserID, &a.Balance, &a.TotalSpent, &a.TotalTopped, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// DeductCredits atomically deducts credits. Returns new balance.
// If balance would go negative, still deducts (micro-debt strategy) and returns new balance.
func (r *pgCreditRepository) DeductCredits(ctx context.Context, userID, amount int64, refID string) (int64, error) {
	var newBalance int64
	err := pgx.BeginTxFunc(ctx, r.db, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var currentBalance int64
		if err := tx.QueryRow(ctx,
			`SELECT balance FROM credit_accounts WHERE user_id = $1 FOR UPDATE`,
			userID,
		).Scan(&currentBalance); err != nil {
			return fmt.Errorf("lock credit account: %w", err)
		}

		newBalance = currentBalance - amount
		if _, err := tx.Exec(ctx,
			`UPDATE credit_accounts SET balance = $1, total_spent = total_spent + $2, updated_at = NOW() WHERE user_id = $3`,
			newBalance, amount, userID,
		); err != nil {
			return err
		}

		_, err := tx.Exec(ctx,
			`INSERT INTO credit_transactions (user_id, type, amount, balance_after, ref_id, description)
             VALUES ($1, 'deduction', $2, $3, $4, 'API usage')`,
			userID, -amount, newBalance, refID,
		)
		return err
	})
	return newBalance, err
}

// AddCredits atomically adds credits. Returns new balance.
func (r *pgCreditRepository) AddCredits(ctx context.Context, userID, amount int64, refID, txType, description string) (int64, error) {
	var newBalance int64
	err := pgx.BeginTxFunc(ctx, r.db, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var currentBalance int64
		if err := tx.QueryRow(ctx,
			`SELECT balance FROM credit_accounts WHERE user_id = $1 FOR UPDATE`,
			userID,
		).Scan(&currentBalance); err != nil {
			return fmt.Errorf("lock credit account: %w", err)
		}

		newBalance = currentBalance + amount
		if _, err := tx.Exec(ctx,
			`UPDATE credit_accounts SET balance = $1, total_topped = total_topped + $2, updated_at = NOW() WHERE user_id = $3`,
			newBalance, amount, userID,
		); err != nil {
			return err
		}

		_, err := tx.Exec(ctx,
			`INSERT INTO credit_transactions (user_id, type, amount, balance_after, ref_id, description)
             VALUES ($1, $2, $3, $4, $5, $6)`,
			userID, txType, amount, newBalance, refID, description,
		)
		return err
	})
	return newBalance, err
}

func (r *pgCreditRepository) ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]*domain.CreditTransaction, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM credit_transactions WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, type, amount, balance_after, ref_id, description, created_at
         FROM credit_transactions WHERE user_id = $1
         ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
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
