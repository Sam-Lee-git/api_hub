package payment

import (
	"context"
	"errors"
	"fmt"
)

// AlipayClient wraps Alipay PC/H5 Pay APIs.
// Uses the smartwf/go-alipay SDK.
type AlipayClient struct {
	appID      string
	privateKey string
	publicKey  string
	notifyURL  string
	sandbox    bool
}

func NewAlipayClient(appID, privateKey, publicKey, notifyURL string, sandbox bool) *AlipayClient {
	return &AlipayClient{
		appID:      appID,
		privateKey: privateKey,
		publicKey:  publicKey,
		notifyURL:  notifyURL,
		sandbox:    sandbox,
	}
}

// CreateOrder creates an Alipay PC pay order and returns the payment URL.
// amountFen is in fen (1 CNY = 100 fen).
func (c *AlipayClient) CreateOrder(_ context.Context, orderNo, subject string, amountFen int64) (string, error) {
	if c.appID == "" {
		return "", errors.New("alipay not configured")
	}

	// Convert fen to yuan string for Alipay (e.g. 1000 fen = "10.00")
	yuan := fmt.Sprintf("%.2f", float64(amountFen)/100.0)

	// TODO: Replace with actual go-alipay SDK call.
	// Example using smartwf/go-alipay:
	//
	//   client, _ := alipay.New(c.appID, c.privateKey, !c.sandbox)
	//   client.LoadAliPayPublicKey(c.publicKey)
	//   req := alipay.TradePagePay{
	//       OutTradeNo:  orderNo,
	//       TotalAmount: yuan,
	//       Subject:     subject,
	//       ReturnURL:   "https://yourdomain.com/billing",
	//       NotifyURL:   c.notifyURL,
	//   }
	//   payURL, err := client.TradePagePay(req)
	//   return payURL.String(), err

	_ = yuan
	return fmt.Sprintf("https://openapi.alipay.com/gateway.do?order=%s", orderNo), nil
}

// VerifyNotify verifies the Alipay async notification signature.
func (c *AlipayClient) VerifyNotify(params map[string]string) error {
	if c.appID == "" {
		return errors.New("alipay not configured")
	}

	// TODO: Replace with actual SDK signature verification.
	// Example:
	//   client, _ := alipay.New(c.appID, c.privateKey, !c.sandbox)
	//   client.LoadAliPayPublicKey(c.publicKey)
	//   return client.VerifySign(params)

	if params["app_id"] != c.appID {
		return errors.New("app_id mismatch")
	}
	return nil
}
