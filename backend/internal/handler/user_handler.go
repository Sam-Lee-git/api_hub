package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
	"github.com/youorg/ai-proxy-platform/backend/internal/middleware"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
	"github.com/youorg/ai-proxy-platform/backend/internal/service"
	"github.com/youorg/ai-proxy-platform/backend/pkg/crypto"
)

type UserHandler struct {
	userRepo  repository.UserRepository
	creditSvc *service.CreditService
	keyRepo   repository.APIKeyRepository
}

func NewUserHandler(userRepo repository.UserRepository, creditSvc *service.CreditService, keyRepo repository.APIKeyRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo, creditSvc: creditSvc, keyRepo: keyRepo}
}

func (h *UserHandler) Me(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)
	ctx := c.Request.Context()

	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	balance, _ := h.creditSvc.GetBalance(ctx, userID)

	c.JSON(http.StatusOK, gin.H{
		"id":           user.ID,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"role":         user.Role,
		"status":       user.Status,
		"balance":      balance,
		"created_at":   user.CreatedAt,
	})
}

type updateProfileRequest struct {
	DisplayName string `json:"display_name"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)
	ctx := c.Request.Context()

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	newHash := ""
	if req.NewPassword != "" {
		if !crypto.CheckPassword(user.PasswordHash, req.OldPassword) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect current password"})
			return
		}
		if len(req.NewPassword) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "new password must be at least 8 characters"})
			return
		}
		var hashErr error
		newHash, hashErr = crypto.HashPassword(req.NewPassword)
		if hashErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = user.DisplayName
	}

	if err := h.userRepo.UpdateProfile(ctx, userID, displayName, newHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

func (h *UserHandler) CreateAPIKey(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)

	var req struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Name == "" {
		req.Name = "Default Key"
	}

	rawKey, hash, prefix, err := crypto.GenerateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}

	apiKey := &domain.APIKey{
		UserID:    userID,
		KeyHash:   hash,
		KeyPrefix: prefix,
		Name:      req.Name,
		Status:    "active",
	}

	if err := h.keyRepo.Create(c.Request.Context(), apiKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create key"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         apiKey.ID,
		"name":       apiKey.Name,
		"key":        rawKey, // shown only once
		"key_prefix": prefix,
		"status":     "active",
		"created_at": apiKey.CreatedAt,
	})
}

func (h *UserHandler) ListAPIKeys(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)

	keys, err := h.keyRepo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": keys})
}

func (h *UserHandler) RevokeAPIKey(c *gin.Context) {
	userID := getContextInt64(c, middleware.CtxUserID)
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	if err := h.keyRepo.Revoke(c.Request.Context(), keyID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "key revoked"})
}

func getContextInt64(c *gin.Context, key string) int64 {
	val, _ := c.Get(key)
	if v, ok := val.(int64); ok {
		return v
	}
	return 0
}
