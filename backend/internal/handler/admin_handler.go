package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
	"github.com/youorg/ai-proxy-platform/backend/internal/service"
)

type AdminHandler struct {
	userRepo    repository.UserRepository
	usageRepo   repository.UsageRepository
	paymentRepo repository.PaymentRepository
	modelRepo   repository.ModelRepository
	provRepo    repository.ProviderRepository
	creditSvc   *service.CreditService
}

func NewAdminHandler(
	userRepo repository.UserRepository,
	usageRepo repository.UsageRepository,
	paymentRepo repository.PaymentRepository,
	modelRepo repository.ModelRepository,
	provRepo repository.ProviderRepository,
	creditSvc *service.CreditService,
) *AdminHandler {
	return &AdminHandler{
		userRepo:    userRepo,
		usageRepo:   usageRepo,
		paymentRepo: paymentRepo,
		modelRepo:   modelRepo,
		provRepo:    provRepo,
		creditSvc:   creditSvc,
	}
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	ctx := c.Request.Context()
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	todayStats, _ := h.usageRepo.GlobalStats(ctx, todayStart, now)
	monthStats, _ := h.usageRepo.GlobalStats(ctx, monthStart, now)

	c.JSON(http.StatusOK, gin.H{
		"today":  todayStats,
		"month":  monthStats,
	})
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	limit, offset := getPagination(c)

	users, total, err := h.userRepo.List(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}

	// Enrich with balances
	type userWithBalance struct {
		*domain.User
		Balance int64 `json:"balance"`
	}
	enriched := make([]userWithBalance, 0, len(users))
	for _, u := range users {
		balance, _ := h.creditSvc.GetBalance(c.Request.Context(), u.ID)
		enriched = append(enriched, userWithBalance{User: u, Balance: balance})
	}

	c.JSON(http.StatusOK, gin.H{"data": enriched, "total": total})
}

func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=active suspended"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userRepo.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

func (h *AdminHandler) AdjustCredits(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req struct {
		Amount      int64  `json:"amount" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.creditSvc.AdminAdjust(c.Request.Context(), id, req.Amount, req.Description); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "adjustment failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "credits adjusted"})
}

func (h *AdminHandler) ListModels(c *gin.Context) {
	models, err := h.modelRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch models"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": models})
}

func (h *AdminHandler) UpdateModel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid model id"})
		return
	}

	model, err := h.modelRepo.FindByID(c.Request.Context(), id)
	if err != nil || model == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
		return
	}

	var req struct {
		DisplayName         string `json:"display_name"`
		InputCreditsPer1K   *int64 `json:"input_credits_per_1k"`
		OutputCreditsPer1K  *int64 `json:"output_credits_per_1k"`
		Status              string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.DisplayName != "" {
		model.DisplayName = req.DisplayName
	}
	if req.InputCreditsPer1K != nil {
		model.InputCreditsPer1K = *req.InputCreditsPer1K
	}
	if req.OutputCreditsPer1K != nil {
		model.OutputCreditsPer1K = *req.OutputCreditsPer1K
	}
	if req.Status != "" {
		model.Status = req.Status
	}

	if err := h.modelRepo.Update(c.Request.Context(), model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "model updated", "model": model})
}

func (h *AdminHandler) ListUsage(c *gin.Context) {
	limit, offset := getPagination(c)

	filters := repository.UsageFilters{}
	if model := c.Query("model"); model != "" {
		filters.ModelName = model
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			filters.From = t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			filters.To = t.Add(24 * time.Hour)
		}
	}

	records, total, err := h.usageRepo.List(c.Request.Context(), filters, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch usage"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": records, "total": total})
}

func (h *AdminHandler) ListPayments(c *gin.Context) {
	limit, offset := getPagination(c)

	orders, total, err := h.paymentRepo.ListAll(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch payments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": orders, "total": total})
}

func (h *AdminHandler) ListProviders(c *gin.Context) {
	providers, err := h.provRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch providers"})
		return
	}

	// Mask API keys
	for _, p := range providers {
		if len(p.APIKey) > 8 {
			p.APIKey = p.APIKey[:8] + "****"
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": providers})
}

func (h *AdminHandler) UpdateProvider(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider id"})
		return
	}

	var req struct {
		APIKey  string `json:"api_key"`
		BaseURL string `json:"base_url"`
		Status  string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	providers, err := h.provRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch providers"})
		return
	}

	var target *domain.Provider
	for _, p := range providers {
		if p.ID == id {
			target = p
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	if req.APIKey != "" {
		target.APIKey = req.APIKey
	}
	if req.BaseURL != "" {
		target.BaseURL = req.BaseURL
	}
	if req.Status != "" {
		target.Status = req.Status
	}

	if err := h.provRepo.Update(c.Request.Context(), target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "provider updated"})
}
