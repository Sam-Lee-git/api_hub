package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/youorg/ai-proxy-platform/backend/internal/service"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

type registerRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authSvc.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		switch err {
		case service.ErrEmailExists:
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":               user.ID,
		"email":            user.Email,
		"display_name":     user.DisplayName,
		"requires_payment": true,
		"message":          "Account created. Please add credits to start using the API.",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, tokens, err := h.authSvc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		case service.ErrUserSuspended:
			c.JSON(http.StatusForbidden, gin.H{"error": "account suspended"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		}
		return
	}

	// Store refresh token in httpOnly cookie
	c.SetCookie("refresh_token", tokens.RefreshToken, 7*24*3600, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokens.AccessToken,
		"expires_in":   tokens.ExpiresIn,
		"token_type":   "Bearer",
		"user": gin.H{
			"id":           user.ID,
			"email":        user.Email,
			"display_name": user.DisplayName,
			"role":         user.Role,
		},
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token missing"})
		return
	}

	tokens, err := h.authSvc.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		c.SetCookie("refresh_token", "", -1, "/", "", true, true)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.SetCookie("refresh_token", tokens.RefreshToken, 7*24*3600, "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{
		"access_token": tokens.AccessToken,
		"expires_in":   tokens.ExpiresIn,
		"token_type":   "Bearer",
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, _ := c.Cookie("refresh_token")
	if refreshToken != "" {
		h.authSvc.Logout(c.Request.Context(), refreshToken)
	}
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
