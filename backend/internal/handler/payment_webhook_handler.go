package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/youorg/ai-proxy-platform/backend/internal/service"
)

type PaymentWebhookHandler struct {
	paymentSvc *service.PaymentService
}

func NewPaymentWebhookHandler(paymentSvc *service.PaymentService) *PaymentWebhookHandler {
	return &PaymentWebhookHandler{paymentSvc: paymentSvc}
}

// AlipayNotify handles the Alipay async payment notification.
func (h *PaymentWebhookHandler) AlipayNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}

	params := make(map[string]string)
	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	if err := h.paymentSvc.HandleAlipayNotify(c.Request.Context(), params); err != nil {
		// Log the error but don't return error to Alipay (it will retry)
		c.String(http.StatusOK, "fail")
		return
	}

	// Alipay requires exactly "success" to stop retrying
	c.String(http.StatusOK, "success")
}

// WechatNotify handles the WeChat Pay V3 payment notification.
func (h *PaymentWebhookHandler) WechatNotify(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "read body failed"})
		return
	}

	headers := map[string]string{
		"Wechatpay-Timestamp": c.GetHeader("Wechatpay-Timestamp"),
		"Wechatpay-Nonce":     c.GetHeader("Wechatpay-Nonce"),
		"Wechatpay-Signature": c.GetHeader("Wechatpay-Signature"),
		"Wechatpay-Serial":    c.GetHeader("Wechatpay-Serial"),
	}

	if err := h.paymentSvc.HandleWechatNotify(c.Request.Context(), body, headers); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": "FAIL", "message": err.Error()})
		return
	}

	// WeChat Pay requires this exact response format to stop retrying
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
}
