package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/youorg/ai-proxy-platform/backend/internal/middleware"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
	"github.com/youorg/ai-proxy-platform/backend/internal/service"
)

type BillingHandler struct {
	creditSvc   *service.CreditService
	paymentSvc  *service.PaymentService
	creditRepo  repository.CreditRepository
	paymentRepo repository.PaymentRepository
}

func NewBillingHandler(
	creditSvc *service.CreditService,
	paymentSvc *service.PaymentService,
	creditRepo repository.CreditRepository,
	paymentRepo repository.PaymentRepository,
) *BillingHandler {
	return &BillingHandler{
		creditSvc:   creditSvc,
		paymentSvc:  paymentSvc,
		creditRepo:  creditRepo,
		paymentRepo: paymentRepo,
	}
}

func (h *BillingHandler) GetBalance(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)
	ctx := c.Request.Context()

	account, err := h.creditSvc.GetAccount(ctx, userID)
	if err != nil || account == nil {
		c.JSON(http.StatusOK, gin.H{"balance": 0, "total_spent": 0, "total_topped": 0})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balance":      account.Balance,
		"total_spent":  account.TotalSpent,
		"total_topped": account.TotalTopped,
		"updated_at":   account.UpdatedAt,
	})
}

func (h *BillingHandler) GetTransactions(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)
	limit, offset := getPagination(c)

	txs, total, err := h.creditRepo.ListTransactions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": txs, "total": total})
}

func (h *BillingHandler) ListPackages(c *gin.Context) {
	pkgs, err := h.paymentRepo.ListPackages(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch packages"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pkgs})
}

type createOrderRequest struct {
	PackageID int    `json:"package_id" binding:"required"`
	Channel   string `json:"channel" binding:"required,oneof=alipay wechat"`
}

func (h *BillingHandler) CreateOrder(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)

	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.paymentSvc.CreateOrder(c.Request.Context(), userID, req.PackageID, req.Channel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_no":    result.OrderNo,
		"channel":     result.Channel,
		"payment_url": result.PaymentURL,
		"code_url":    result.CodeURL,
		"expires_at":  result.ExpiresAt,
	})
}

func (h *BillingHandler) GetOrderStatus(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)
	orderNo := c.Param("order_no")

	order, err := h.paymentSvc.GetOrderStatus(c.Request.Context(), orderNo, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_no":   order.OrderNo,
		"status":     order.Status,
		"channel":    order.Channel,
		"amount_cny": order.AmountCNY,
		"credits":    order.CreditsToAdd,
		"paid_at":    order.PaidAt,
		"expires_at": order.ExpiresAt,
	})
}

func (h *BillingHandler) ListOrders(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)
	limit, offset := getPagination(c)

	orders, total, err := h.paymentRepo.ListByUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": orders, "total": total})
}

func getPagination(c *gin.Context) (int, int) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}
	return limit, (page - 1) * limit
}
