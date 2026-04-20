package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
	"github.com/youorg/ai-proxy-platform/backend/internal/middleware"
	"github.com/youorg/ai-proxy-platform/backend/internal/proxy/providers"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
	"github.com/youorg/ai-proxy-platform/backend/internal/service"
	"go.uber.org/zap"
)

type Handler struct {
	registry  *Registry
	modelRepo repository.ModelRepository
	usageRepo repository.UsageRepository
	keyRepo   repository.APIKeyRepository
	creditSvc *service.CreditService
	log       *zap.Logger
}

func NewHandler(
	registry *Registry,
	modelRepo repository.ModelRepository,
	usageRepo repository.UsageRepository,
	keyRepo repository.APIKeyRepository,
	creditSvc *service.CreditService,
	log *zap.Logger,
) *Handler {
	return &Handler{
		registry:  registry,
		modelRepo: modelRepo,
		usageRepo: usageRepo,
		keyRepo:   keyRepo,
		creditSvc: creditSvc,
		log:       log,
	}
}

// ChatCompletions is the main OpenAI-compatible proxy endpoint.
func (h *Handler) ChatCompletions(c *gin.Context) {
	userID := getInt64(c, middleware.CtxUserID)
	apiKeyID := getInt64(c, middleware.CtxAPIKeyID)
	requestID := uuid.New().String()
	startTime := time.Now()

	var req providers.ChatRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	provider, err := h.registry.Get(req.Model)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model not supported: " + req.Model})
		return
	}

	model, err := h.modelRepo.FindByModelID(c.Request.Context(), req.Model)
	if err != nil || model == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model not found in pricing table"})
		return
	}

	reqCtx := c.Request.Context()

	if req.Stream {
		h.handleStream(c, reqCtx, provider, &req, model, userID, apiKeyID, requestID, startTime)
	} else {
		h.handleComplete(c, reqCtx, provider, &req, model, userID, apiKeyID, requestID, startTime)
	}
}

func (h *Handler) handleComplete(
	c *gin.Context, reqCtx context.Context,
	provider providers.Provider, req *providers.ChatRequest, model *domain.Model,
	userID, apiKeyID int64, requestID string, startTime time.Time,
) {
	resp, usage, err := provider.Complete(reqCtx, req)
	latencyMs := int(time.Since(startTime).Milliseconds())

	if err != nil {
		go h.saveUsage(context.Background(), &domain.UsageRecord{
			UserID: userID, APIKeyID: apiKeyID, ModelID: model.ID,
			RequestID: requestID, Status: "error", ErrorMessage: err.Error(), LatencyMs: latencyMs,
		}, 0)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	if usage == nil {
		usage = &providers.UsageInfo{}
	}

	credits := model.CalculateCost(usage.InputTokens, usage.OutputTokens)
	rec := &domain.UsageRecord{
		UserID: userID, APIKeyID: apiKeyID, ModelID: model.ID,
		RequestID:      requestID,
		InputTokens:    usage.InputTokens,
		OutputTokens:   usage.OutputTokens,
		TotalTokens:    usage.InputTokens + usage.OutputTokens,
		CreditsCharged: credits,
		Status:         "success",
		LatencyMs:      latencyMs,
	}

	go h.saveUsage(context.Background(), rec, apiKeyID)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) handleStream(
	c *gin.Context, reqCtx context.Context,
	provider providers.Provider, req *providers.ChatRequest, model *domain.Model,
	userID, apiKeyID int64, requestID string, startTime time.Time,
) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	usage, err := provider.CompleteStream(reqCtx, req, c.Writer)
	latencyMs := int(time.Since(startTime).Milliseconds())

	if usage == nil {
		usage = &providers.UsageInfo{}
	}

	credits := model.CalculateCost(usage.InputTokens, usage.OutputTokens)
	status := "success"
	errMsg := ""
	if err != nil {
		status = "error"
		errMsg = err.Error()
	}

	rec := &domain.UsageRecord{
		UserID: userID, APIKeyID: apiKeyID, ModelID: model.ID,
		RequestID:      requestID,
		InputTokens:    usage.InputTokens,
		OutputTokens:   usage.OutputTokens,
		TotalTokens:    usage.InputTokens + usage.OutputTokens,
		CreditsCharged: credits,
		Status:         status,
		ErrorMessage:   errMsg,
		LatencyMs:      latencyMs,
	}

	go h.saveUsage(context.Background(), rec, apiKeyID)
}

func (h *Handler) saveUsage(ctx context.Context, rec *domain.UsageRecord, apiKeyID int64) {
	if err := h.usageRepo.Create(ctx, rec); err != nil {
		h.log.Warn("failed to write usage record", zap.String("requestID", rec.RequestID), zap.Error(err))
	}
	if rec.CreditsCharged > 0 {
		if err := h.creditSvc.DeductForUsage(ctx, rec.UserID, rec.CreditsCharged, rec.RequestID); err != nil {
			h.log.Warn("credit deduction failed", zap.String("requestID", rec.RequestID), zap.Error(err))
		}
	}
	if apiKeyID > 0 {
		h.keyRepo.UpdateLastUsed(ctx, apiKeyID)
	}
}

// Models returns the OpenAI-compatible model list.
func (h *Handler) Models(c *gin.Context) {
	models, err := h.modelRepo.ListActive(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list models"})
		return
	}

	data := make([]map[string]any, 0, len(models))
	for _, m := range models {
		data = append(data, map[string]any{
			"id":       m.ModelID,
			"object":   "model",
			"created":  m.CreatedAt.Unix(),
			"owned_by": m.ProviderName,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

func getInt64(c *gin.Context, key string) int64 {
	val, _ := c.Get(key)
	if v, ok := val.(int64); ok {
		return v
	}
	return 0
}
