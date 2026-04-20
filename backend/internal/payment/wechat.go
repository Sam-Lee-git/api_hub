package payment

import (
	"context"
	"errors"
	"fmt"
)

// WechatClient wraps WeChat Pay V3 Native Pay APIs.
type WechatClient struct {
	mchID      string
	appID      string
	apiV3Key   string
	certSerial string
	notifyURL  string
}

func NewWechatClient(mchID, appID, apiV3Key, certSerial, notifyURL string) *WechatClient {
	return &WechatClient{
		mchID:      mchID,
		appID:      appID,
		apiV3Key:   apiV3Key,
		certSerial: certSerial,
		notifyURL:  notifyURL,
	}
}

// CreateOrder creates a WeChat Pay Native order and returns the code_url for QR code generation.
// amountFen is in fen (1 CNY = 100 fen).
func (c *WechatClient) CreateOrder(_ context.Context, orderNo, description string, amountFen int64) (string, error) {
	if c.mchID == "" {
		return "", errors.New("wechat pay not configured")
	}

	// TODO: Replace with actual wechatpay-apiv3/wechatpay-go SDK call.
	// Example:
	//
	//   mchPrivateKey, _ := utils.LoadPrivateKeyWithPath("/path/to/cert/apiclient_key.pem")
	//   ctx := context.Background()
	//   client, _ := core.NewClient(ctx,
	//       core.WithWechatPayAutoAuthCipher(c.mchID, c.certSerial, mchPrivateKey, c.apiV3Key),
	//   )
	//   svc := native.NativeApiService{Client: client}
	//   resp, _, _ := svc.Prepay(ctx, native.PrepayRequest{
	//       Appid:       core.String(c.appID),
	//       Mchid:       core.String(c.mchID),
	//       Description: core.String(description),
	//       OutTradeNo:  core.String(orderNo),
	//       NotifyUrl:   core.String(c.notifyURL),
	//       Amount:      &native.Amount{Total: core.Int64(amountFen)},
	//   })
	//   return *resp.CodeUrl, nil

	_ = description
	return fmt.Sprintf("weixin://wxpay/bizpayurl?pr=%s", orderNo), nil
}

// ParseNotify parses and verifies a WeChat Pay V3 payment notification.
// Returns (orderNo, providerOrderNo, error).
func (c *WechatClient) ParseNotify(_ context.Context, body []byte, headers map[string]string) (string, string, error) {
	if c.mchID == "" {
		return "", "", errors.New("wechat pay not configured")
	}

	// TODO: Implement WeChat Pay V3 notification verification.
	// The notification body is AES-256-GCM encrypted.
	//
	// Example flow:
	//   1. Verify HTTP signature using wechatpay-go's core/notify handler
	//   2. Decrypt the resource.ciphertext field using apiV3Key
	//   3. Parse the decrypted JSON to extract out_trade_no and transaction_id
	//
	//   handler := notify.NewNotifyHandler(c.apiV3Key, verifiers.NewSHA256WithRSAVerifier(...))
	//   transaction := &payments.Transaction{}
	//   _, err := handler.ParseNotifyRequest(ctx, req, transaction)
	//   if err != nil { return "", "", err }
	//   if *transaction.TradeState != "SUCCESS" { return "", "", nil }
	//   return *transaction.OutTradeNo, *transaction.TransactionId, nil

	_ = body
	_ = headers
	return "", "", errors.New("wechat notify parsing not yet implemented - add SDK")
}
