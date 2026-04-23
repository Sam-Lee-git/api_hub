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

type paymentRepository struct{ db *sql.DB }

func NewPaymentRepository(db *sql.DB) repository.PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CreateOrder(ctx context.Context, o *domain.PaymentOrder) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO payment_orders (user_id, order_no, channel, amount_cny, credits_to_add, status, expires_at)
         VALUES (?,?,?,?,?,'pending',?)`,
		o.UserID, o.OrderNo, o.Channel, o.AmountCNY, o.CreditsToAdd,
		o.ExpiresAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	o.ID, err = res.LastInsertId()
	o.CreatedAt = time.Now()
	o.UpdatedAt = time.Now()
	return err
}

func (r *paymentRepository) FindByOrderNo(ctx context.Context, orderNo string) (*domain.PaymentOrder, error) {
	o := &domain.PaymentOrder{}
	var createdAt, updatedAt, expiresAt sqlTime
	var paidAt sqlNullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, order_no, channel, amount_cny, credits_to_add, status,
                provider_order_no, paid_at, expires_at, created_at, updated_at
         FROM payment_orders WHERE order_no = ?`,
		orderNo,
	).Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Channel, &o.AmountCNY, &o.CreditsToAdd, &o.Status,
		&o.ProviderOrderNo, &paidAt, &expiresAt, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	o.CreatedAt = createdAt.T
	o.UpdatedAt = updatedAt.T
	o.ExpiresAt = expiresAt.T
	o.PaidAt = paidAt.T
	return o, nil
}

func (r *paymentRepository) MarkPaid(ctx context.Context, orderNo, providerOrderNo string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE payment_orders
         SET status='paid', provider_order_no=?, paid_at=datetime('now'), updated_at=datetime('now')
         WHERE order_no=? AND status='pending'`,
		providerOrderNo, orderNo,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("order not found or already processed")
	}
	return nil
}

func (r *paymentRepository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*domain.PaymentOrder, int64, error) {
	var total int64
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM payment_orders WHERE user_id = ?`, userID).Scan(&total)

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, order_no, channel, amount_cny, credits_to_add, status,
                provider_order_no, paid_at, expires_at, created_at, updated_at
         FROM payment_orders WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	orders, err := scanOrders(rows)
	return orders, total, err
}

func (r *paymentRepository) ListAll(ctx context.Context, limit, offset int) ([]*domain.PaymentOrder, int64, error) {
	var total int64
	r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM payment_orders`).Scan(&total)

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, order_no, channel, amount_cny, credits_to_add, status,
                provider_order_no, paid_at, expires_at, created_at, updated_at
         FROM payment_orders ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	orders, err := scanOrders(rows)
	return orders, total, err
}

func (r *paymentRepository) ListPackages(ctx context.Context) ([]*domain.CreditPackage, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, amount_cny, credits, bonus_credits, is_active, display_order, created_at
         FROM credit_packages WHERE is_active = 1 ORDER BY display_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pkgs []*domain.CreditPackage
	for rows.Next() {
		p := &domain.CreditPackage{}
		var isActive int
		var createdAt sqlTime
		if err := rows.Scan(&p.ID, &p.Name, &p.AmountCNY, &p.Credits, &p.BonusCredits,
			&isActive, &p.DisplayOrder, &createdAt); err != nil {
			return nil, err
		}
		p.IsActive = isActive != 0
		p.CreatedAt = createdAt.T
		pkgs = append(pkgs, p)
	}
	return pkgs, rows.Err()
}

func (r *paymentRepository) FindPackageByID(ctx context.Context, id int) (*domain.CreditPackage, error) {
	p := &domain.CreditPackage{}
	var isActive int
	var createdAt sqlTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, amount_cny, credits, bonus_credits, is_active, display_order, created_at
         FROM credit_packages WHERE id = ? AND is_active = 1`, id,
	).Scan(&p.ID, &p.Name, &p.AmountCNY, &p.Credits, &p.BonusCredits,
		&isActive, &p.DisplayOrder, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.IsActive = isActive != 0
	p.CreatedAt = createdAt.T
	return p, nil
}

func scanOrders(rows *sql.Rows) ([]*domain.PaymentOrder, error) {
	var orders []*domain.PaymentOrder
	for rows.Next() {
		o := &domain.PaymentOrder{}
		var createdAt, updatedAt, expiresAt sqlTime
		var paidAt sqlNullTime
		if err := rows.Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Channel, &o.AmountCNY, &o.CreditsToAdd,
			&o.Status, &o.ProviderOrderNo, &paidAt, &expiresAt, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		o.CreatedAt = createdAt.T
		o.UpdatedAt = updatedAt.T
		o.ExpiresAt = expiresAt.T
		o.PaidAt = paidAt.T
		orders = append(orders, o)
	}
	return orders, rows.Err()
}
