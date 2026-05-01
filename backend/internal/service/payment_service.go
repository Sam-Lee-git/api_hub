package service

import (
	"context"
	"errors"
	"time"

	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
	"github.com/youorg/ai-proxy-platform/backend/internal/payment"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
	"github.com/youorg/ai-proxy-platform/backend/pkg/crypto"
)

type PaymentService struct {
	paymentRepo repository.PaymentRepository
	creditSvc   *CreditService
	alipay      *payment.AlipayClient
	wechat      *payment.WechatClient
}

func NewPaymentService(
	paymentRepo repository.PaymentRepository,
	creditSvc *CreditService,
	alipay *payment.AlipayClient,
	wechat *payment.WechatClient,
) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		creditSvc:   creditSvc,
		alipay:      alipay,
		wechat:      wechat,
	}
}

type OrderResult struct {
	OrderNo    string
	Channel    string
	PaymentURL string // for Alipay
	CodeURL    string // for WeChat QR
	ExpiresAt  time.Time
}

func (s *PaymentService) CreateOrder(ctx context.Context, userID int64, packageID int, channel string) (*OrderResult, error) {
	pkg, err := s.paymentRepo.FindPackageByID(ctx, packageID)
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, errors.New("package not found")
	}

	totalCredits := pkg.Credits + pkg.BonusCredits
	orderNo := crypto.GenerateOrderNo(channel[:2], userID)
	expiresAt := time.Now().Add(15 * time.Minute)

	order := &domain.PaymentOrder{
		UserID:       userID,
		OrderNo:      orderNo,
		Channel:      channel,
		AmountCNY:    pkg.AmountCNY,
		CreditsToAdd: totalCredits,
		ExpiresAt:    expiresAt,
	}

	if err := s.paymentRepo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	result := &OrderResult{
		OrderNo:   orderNo,
		Channel:   channel,
		ExpiresAt: expiresAt,
	}

	subject := pkg.Name
	amountFen := pkg.AmountCNY // stored in fen

	switch channel {
	case "alipay":
		payURL, err := s.alipay.CreateOrder(ctx, orderNo, subject, amountFen)
		if err != nil {
			return nil, err
		}
		result.PaymentURL = payURL
	case "wechat":
		codeURL, err := s.wechat.CreateOrder(ctx, orderNo, subject, amountFen)
		if err != nil {
			return nil, err
		}
		result.CodeURL = codeURL
	default:
		return nil, errors.New("unsupported payment channel")
	}

	return result, nil
}

// HandleAlipayNotify processes an Alipay payment notification.
func (s *PaymentService) HandleAlipayNotify(ctx context.Context, params map[string]string) error {
	if err := s.alipay.VerifyNotify(params); err != nil {
		return err
	}
	if params["trade_status"] != "TRADE_SUCCESS" {
		return nil // ignore non-success notifications
	}

	orderNo := params["out_trade_no"]
	providerOrderNo := params["trade_no"]
	return s.fulfillOrder(ctx, orderNo, providerOrderNo)
}

// HandleWechatNotify processes a WeChat Pay payment notification.
func (s *PaymentService) HandleWechatNotify(ctx context.Context, body []byte, headers map[string]string) error {
	orderNo, providerOrderNo, err := s.wechat.ParseNotify(ctx, body, headers)
	if err != nil {
		return err
	}
	return s.fulfillOrder(ctx, orderNo, providerOrderNo)
}

// fulfillOrder atomically marks the order paid and credits the user.
func (s *PaymentService) fulfillOrder(ctx context.Context, orderNo, providerOrderNo string) error {
	order, fulfilled, err := s.paymentRepo.FulfillPaidOrder(ctx, orderNo, providerOrderNo)
	if err != nil {
		return err
	}
	if order == nil {
		return errors.New("order not found")
	}
	if fulfilled {
		s.creditSvc.InvalidateBalance(ctx, order.UserID)
	}
	return nil
}

func (s *PaymentService) GetOrderStatus(ctx context.Context, orderNo string, userID int64) (*domain.PaymentOrder, error) {
	order, err := s.paymentRepo.FindByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	if order == nil || order.UserID != userID {
		return nil, errors.New("order not found")
	}
	return order, nil
}
