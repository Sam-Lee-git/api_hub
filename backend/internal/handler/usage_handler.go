package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/youorg/ai-proxy-platform/backend/internal/middleware"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
)

type UsageHandler struct {
	usageRepo repository.UsageRepository
}

func NewUsageHandler(usageRepo repository.UsageRepository) *UsageHandler {
	return &UsageHandler{usageRepo: usageRepo}
}

func (h *UsageHandler) ListUsage(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)
	limit, offset := getPagination(c)

	filters := repository.UsageFilters{UserID: userID}
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

func (h *UsageHandler) GetSummary(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)

	from := time.Now().AddDate(0, 0, -7)
	to := time.Now()

	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			to = t.Add(24 * time.Hour)
		}
	}

	summary, err := h.usageRepo.Summarize(c.Request.Context(), userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}
