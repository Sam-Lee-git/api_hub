package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/youorg/ai-proxy-platform/backend/internal/config"
	"github.com/youorg/ai-proxy-platform/backend/internal/db"
	"github.com/youorg/ai-proxy-platform/backend/internal/db/cache"
	"github.com/youorg/ai-proxy-platform/backend/internal/domain"
	"github.com/youorg/ai-proxy-platform/backend/internal/handler"
	"github.com/youorg/ai-proxy-platform/backend/internal/middleware"
	"github.com/youorg/ai-proxy-platform/backend/internal/payment"
	"github.com/youorg/ai-proxy-platform/backend/internal/proxy"
	"github.com/youorg/ai-proxy-platform/backend/internal/proxy/providers"
	"github.com/youorg/ai-proxy-platform/backend/internal/repository"
	sqliterepo "github.com/youorg/ai-proxy-platform/backend/internal/repository/sqlite"
	"github.com/youorg/ai-proxy-platform/backend/internal/service"
	"github.com/youorg/ai-proxy-platform/backend/pkg/crypto"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	var logger *zap.Logger
	if cfg.Env == "production" {
		logger, _ = zap.NewProduction()
	} else {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	ctx := context.Background()

	// ---- Database ----
	var (
		userRepo     repository.UserRepository
		apiKeyRepo   repository.APIKeyRepository
		creditRepo   repository.CreditRepository
		modelRepo    repository.ModelRepository
		providerRepo repository.ProviderRepository
		usageRepo    repository.UsageRepository
		paymentRepo  repository.PaymentRepository
	)

	switch cfg.DBDriver {
	case "sqlite":
		logger.Info("using SQLite", zap.String("path", cfg.SQLitePath))
		sqlDB, err := db.NewSQLiteDB(ctx, cfg.SQLitePath)
		if err != nil {
			logger.Fatal("connect sqlite", zap.Error(err))
		}
		if err := db.MigrateSQLite(sqlDB, cfg.SQLitePath); err != nil {
			logger.Fatal("sqlite migration", zap.Error(err))
		}
		userRepo = sqliterepo.NewUserRepository(sqlDB)
		apiKeyRepo = sqliterepo.NewAPIKeyRepository(sqlDB)
		creditRepo = sqliterepo.NewCreditRepository(sqlDB)
		modelRepo = sqliterepo.NewModelRepository(sqlDB)
		providerRepo = sqliterepo.NewProviderRepository(sqlDB)
		usageRepo = sqliterepo.NewUsageRepository(sqlDB)
		paymentRepo = sqliterepo.NewPaymentRepository(sqlDB)

	default: // "postgres"
		logger.Info("using PostgreSQL")
		pgPool, err := db.NewPostgresPool(ctx, cfg.DatabaseURL)
		if err != nil {
			logger.Fatal("connect postgres", zap.Error(err))
		}
		defer pgPool.Close()
		userRepo = repository.NewUserRepository(pgPool)
		apiKeyRepo = repository.NewAPIKeyRepository(pgPool)
		creditRepo = repository.NewCreditRepository(pgPool)
		modelRepo = repository.NewModelRepository(pgPool)
		providerRepo = repository.NewProviderRepository(pgPool)
		usageRepo = repository.NewUsageRepository(pgPool)
		paymentRepo = repository.NewPaymentRepository(pgPool)
	}

	// ---- Cache ----
	var cacheClient cache.Client
	switch cfg.CacheDriver {
	case "memory":
		logger.Info("using in-memory cache")
		cacheClient = cache.NewMemoryCache()
	default: // "redis"
		logger.Info("using Redis cache")
		rdb, err := db.NewRedisClient(ctx, cfg.RedisURL)
		if err != nil {
			logger.Fatal("connect redis", zap.Error(err))
		}
		defer rdb.Close()
		cacheClient = cache.NewRedisCache(rdb)
	}

	// ---- Payment clients ----
	alipayClient := payment.NewAlipayClient(
		cfg.AlipayAppID, cfg.AlipayPrivateKey, cfg.AlipayPublicKey, cfg.AlipayNotifyURL,
		cfg.Env != "production",
	)
	wechatClient := payment.NewWechatClient(
		cfg.WechatMchID, cfg.WechatAppID, cfg.WechatAPIV3Key, cfg.WechatCertSerial, cfg.WechatNotifyURL,
	)

	// ---- Services ----
	authSvc := service.NewAuthService(userRepo, cfg.JWTAccessSecret, cfg.JWTRefreshSecret)
	creditSvc := service.NewCreditService(creditRepo, cacheClient)
	paymentSvc := service.NewPaymentService(paymentRepo, creditSvc, alipayClient, wechatClient)

	// ---- Proxy registry ----
	registry := proxy.NewRegistry()
	registry.Register(providers.NewOpenAIProvider(
		cfg.OpenAIAPIKey, "https://api.openai.com/v1",
		[]string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"},
	))
	if cfg.AnthropicAPIKey != "" {
		registry.Register(providers.NewAnthropicProvider(
			cfg.AnthropicAPIKey,
			[]string{"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022", "claude-3-opus-20240229"},
		))
	}
	if cfg.GoogleAPIKey != "" {
		registry.Register(providers.NewGeminiProvider(
			cfg.GoogleAPIKey,
			[]string{"gemini-1.5-pro", "gemini-1.5-flash", "gemini-2.0-flash"},
		))
	}
	if cfg.AlibabaAPIKey != "" {
		registry.Register(providers.NewOpenAIProvider(
			cfg.AlibabaAPIKey, "https://dashscope.aliyuncs.com/compatible-mode/v1",
			[]string{"qwen-max", "qwen-plus", "qwen-turbo"},
		))
	}

	// ---- Handlers ----
	proxyHandler := proxy.NewHandler(registry, modelRepo, usageRepo, apiKeyRepo, creditSvc, logger)
	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(userRepo, creditSvc, apiKeyRepo)
	billingHandler := handler.NewBillingHandler(creditSvc, paymentSvc, creditRepo, paymentRepo)
	usageHandler := handler.NewUsageHandler(usageRepo)
	webhookHandler := handler.NewPaymentWebhookHandler(paymentSvc)
	adminHandler := handler.NewAdminHandler(userRepo, usageRepo, paymentRepo, modelRepo, providerRepo, apiKeyRepo, creditSvc)

	// ---- Seed ----
	seedAdmin(ctx, userRepo, cfg, logger)
	seedProviderKeys(ctx, providerRepo, cfg, logger)

	// ---- Router ----
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
	}

	r.POST("/api/payment/alipay/notify", webhookHandler.AlipayNotify)
	r.POST("/api/payment/wechat/notify", webhookHandler.WechatNotify)
	r.GET("/api/models", proxyHandler.Models)

	jwtAuth := middleware.JWTAuth(cfg.JWTAccessSecret)
	api := r.Group("/api", jwtAuth)
	{
		api.GET("/user/me", userHandler.Me)
		api.PUT("/user/me", userHandler.UpdateProfile)

		api.GET("/keys", userHandler.ListAPIKeys)
		api.POST("/keys", userHandler.CreateAPIKey)
		api.DELETE("/keys/:id", userHandler.RevokeAPIKey)

		api.GET("/billing/balance", billingHandler.GetBalance)
		api.GET("/billing/transactions", billingHandler.GetTransactions)
		api.GET("/billing/packages", billingHandler.ListPackages)
		api.POST("/billing/orders", billingHandler.CreateOrder)
		api.GET("/billing/orders", billingHandler.ListOrders)
		api.GET("/billing/orders/:order_no", billingHandler.GetOrderStatus)

		api.GET("/usage", usageHandler.ListUsage)
		api.GET("/usage/summary", usageHandler.GetSummary)
	}

	admin := r.Group("/api/admin", jwtAuth, middleware.AdminOnly())
	{
		admin.GET("/dashboard", adminHandler.Dashboard)
		admin.GET("/users", adminHandler.ListUsers)
		admin.PUT("/users/:id/status", adminHandler.UpdateUserStatus)
		admin.POST("/users/:id/credits", adminHandler.AdjustCredits)
		admin.POST("/users/:id/keys", adminHandler.CreateUserAPIKey)
		admin.GET("/usage", adminHandler.ListUsage)
		admin.GET("/payments", adminHandler.ListPayments)
		admin.GET("/models", adminHandler.ListModels)
		admin.PUT("/models/:id", adminHandler.UpdateModel)
		admin.GET("/providers", adminHandler.ListProviders)
		admin.PUT("/providers/:id", adminHandler.UpdateProvider)
	}

	v1 := r.Group("/v1",
		middleware.APIKeyAuth(apiKeyRepo, cacheClient),
		middleware.RequireCredits(creditSvc),
		middleware.RateLimit(cacheClient, 60),
	)
	{
		v1.POST("/chat/completions", proxyHandler.ChatCompletions)
		v1.GET("/models", proxyHandler.Models)
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 330 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}
	logger.Info("server stopped")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func seedAdmin(ctx context.Context, userRepo repository.UserRepository, cfg *config.Config, logger *zap.Logger) {
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		return
	}
	existing, err := userRepo.FindByEmail(ctx, cfg.AdminEmail)
	if err != nil || existing != nil {
		return
	}
	hash, err := crypto.HashPassword(cfg.AdminPassword)
	if err != nil {
		logger.Warn("failed to hash admin password", zap.Error(err))
		return
	}
	u := &domain.User{
		Email:        cfg.AdminEmail,
		PasswordHash: hash,
		DisplayName:  "Admin",
		Role:         "admin",
		Status:       "active",
	}
	if err := userRepo.CreateWithCreditAccount(ctx, u); err != nil {
		logger.Warn("failed to seed admin user", zap.Error(err))
		return
	}
	logger.Info(fmt.Sprintf("admin user seeded: %s", cfg.AdminEmail))
}

func seedProviderKeys(ctx context.Context, provRepo repository.ProviderRepository, cfg *config.Config, logger *zap.Logger) {
	updates := map[string]string{
		"openai":    cfg.OpenAIAPIKey,
		"anthropic": cfg.AnthropicAPIKey,
		"google":    cfg.GoogleAPIKey,
		"alibaba":   cfg.AlibabaAPIKey,
	}
	for name, apiKey := range updates {
		if apiKey == "" {
			continue
		}
		p, err := provRepo.FindByName(ctx, name)
		if err != nil || p == nil {
			continue
		}
		p.APIKey = apiKey
		if err := provRepo.Update(ctx, p); err != nil {
			logger.Warn("failed to update provider key", zap.String("provider", name), zap.Error(err))
		}
	}
}
