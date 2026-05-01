package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
)

type pgPaymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) PaymentRepository {
	return &pgPaymentRepository{db: db}
}

func (r *pgPaymentRepository) CreateOrder(ctx context.Context, o *domain.PaymentOrder) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO payment_orders (user_id, order_no, channel, amount_cny, credits_to_add, status, expires_at)
         VALUES ($1,$2,$3,$4,$5,'pending',$6)
         RETURNING id, created_at, updated_at`,
		o.UserID, o.OrderNo, o.Channel, o.AmountCNY, o.CreditsToAdd, o.ExpiresAt,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
}

func (r *pgPaymentRepository) FindByOrderNo(ctx context.Context, orderNo string) (*domain.PaymentOrder, error) {
	o := &domain.PaymentOrder{}
	var providerOrderNo sql.NullString
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, order_no, channel, amount_cny, credits_to_add, status,
                provider_order_no, paid_at, expires_at, created_at, updated_at
         FROM payment_orders WHERE order_no = $1`,
		orderNo,
	).Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Channel, &o.AmountCNY, &o.CreditsToAdd, &o.Status,
		&providerOrderNo, &o.PaidAt, &o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if providerOrderNo.Valid {
		o.ProviderOrderNo = providerOrderNo.String
	}
	return o, err
}

func (r *pgPaymentRepository) MarkPaid(ctx context.Context, orderNo, providerOrderNo string) error {
	result, err := r.db.Exec(ctx,
		`UPDATE payment_orders
         SET status='paid', provider_order_no=$1, paid_at=NOW(), updated_at=NOW()
         WHERE order_no=$2 AND status='pending'`,
		providerOrderNo, orderNo)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("order not found or already processed")
	}
	return nil
}

func (r *pgPaymentRepository) FulfillPaidOrder(ctx context.Context, orderNo, providerOrderNo string) (*domain.PaymentOrder, bool, error) {
	var order *domain.PaymentOrder
	fulfilled := false

	err := pgx.BeginTxFunc(ctx, r.db, pgx.TxOptions{}, func(tx pgx.Tx) error {
		o := &domain.PaymentOrder{}
		var providerOrderNoValue sql.NullString
		err := tx.QueryRow(ctx,
			`SELECT id, user_id, order_no, channel, amount_cny, credits_to_add, status,
                    provider_order_no, paid_at, expires_at, created_at, updated_at
             FROM payment_orders WHERE order_no = $1 FOR UPDATE`,
			orderNo,
		).Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Channel, &o.AmountCNY, &o.CreditsToAdd, &o.Status,
			&providerOrderNoValue, &o.PaidAt, &o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			order = nil
			return nil
		}
		if err != nil {
			return err
		}
		if providerOrderNoValue.Valid {
			o.ProviderOrderNo = providerOrderNoValue.String
		}
		order = o

		if o.Status != "pending" {
			return nil
		}

		if _, err := tx.Exec(ctx,
			`UPDATE payment_orders
             SET status='paid', provider_order_no=$1, paid_at=NOW(), updated_at=NOW()
             WHERE id=$2`,
			providerOrderNo, o.ID); err != nil {
			return err
		}

		var currentBalance int64
		if err := tx.QueryRow(ctx,
			`SELECT balance FROM credit_accounts WHERE user_id = $1 FOR UPDATE`,
			o.UserID,
		).Scan(&currentBalance); err != nil {
			return fmt.Errorf("lock credit account: %w", err)
		}

		newBalance := currentBalance + o.CreditsToAdd
		if _, err := tx.Exec(ctx,
			`UPDATE credit_accounts SET balance = $1, total_topped = total_topped + $2, updated_at = NOW() WHERE user_id = $3`,
			newBalance, o.CreditsToAdd, o.UserID,
		); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx,
			`INSERT INTO credit_transactions (user_id, type, amount, balance_after, ref_id, description)
             VALUES ($1, 'topup', $2, $3, $4, 'Credit top-up')`,
			o.UserID, o.CreditsToAdd, newBalance, o.OrderNo,
		); err != nil {
			return err
		}

		fulfilled = true
		o.Status = "paid"
		o.ProviderOrderNo = providerOrderNo
		return nil
	})
	return order, fulfilled, err
}

func (r *pgPaymentRepository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*domain.PaymentOrder, int64, error) {
	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM payment_orders WHERE user_id = $1`, userID).Scan(&total)

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, order_no, channel, amount_cny, credits_to_add, status,
                provider_order_no, paid_at, expires_at, created_at, updated_at
         FROM payment_orders WHERE user_id = $1
         ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	orders, err := scanOrders(rows)
	return orders, total, err
}

func (r *pgPaymentRepository) ListAll(ctx context.Context, limit, offset int) ([]*domain.PaymentOrder, int64, error) {
	var total int64
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM payment_orders`).Scan(&total)

	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, order_no, channel, amount_cny, credits_to_add, status,
                provider_order_no, paid_at, expires_at, created_at, updated_at
         FROM payment_orders ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	orders, err := scanOrders(rows)
	return orders, total, err
}

func (r *pgPaymentRepository) ListPackages(ctx context.Context) ([]*domain.CreditPackage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, amount_cny, credits, bonus_credits, is_active, display_order, created_at
         FROM credit_packages WHERE is_active = TRUE ORDER BY display_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pkgs []*domain.CreditPackage
	for rows.Next() {
		p := &domain.CreditPackage{}
		if err := rows.Scan(&p.ID, &p.Name, &p.AmountCNY, &p.Credits, &p.BonusCredits,
			&p.IsActive, &p.DisplayOrder, &p.CreatedAt); err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, rows.Err()
}

func (r *pgPaymentRepository) FindPackageByID(ctx context.Context, id int) (*domain.CreditPackage, error) {
	p := &domain.CreditPackage{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, amount_cny, credits, bonus_credits, is_active, display_order, created_at
         FROM credit_packages WHERE id = $1 AND is_active = TRUE`,
		id,
	).Scan(&p.ID, &p.Name, &p.AmountCNY, &p.Credits, &p.BonusCredits,
		&p.IsActive, &p.DisplayOrder, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

func scanOrders(rows interface {
	Next() bool
	Scan(...interface{}) error
	Err() error
}) ([]*domain.PaymentOrder, error) {
	var orders []*domain.PaymentOrder
	for rows.Next() {
		o := &domain.PaymentOrder{}
		var providerOrderNo sql.NullString
		if err := rows.Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Channel, &o.AmountCNY, &o.CreditsToAdd,
			&o.Status, &providerOrderNo, &o.PaidAt, &o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		if providerOrderNo.Valid {
			o.ProviderOrderNo = providerOrderNo.String
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}
