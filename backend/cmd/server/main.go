package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/safe-ai/excel-brushing-tool/internal/admin"
	"github.com/safe-ai/excel-brushing-tool/internal/assessment"
	"github.com/safe-ai/excel-brushing-tool/internal/auth"
	"github.com/safe-ai/excel-brushing-tool/internal/cleaning"
	"github.com/safe-ai/excel-brushing-tool/internal/comparison"
	"github.com/safe-ai/excel-brushing-tool/internal/evidence"
	"github.com/safe-ai/excel-brushing-tool/internal/export"
	"github.com/safe-ai/excel-brushing-tool/internal/middleware"
	"github.com/safe-ai/excel-brushing-tool/internal/qa"
	"github.com/safe-ai/excel-brushing-tool/internal/quota"
	"github.com/safe-ai/excel-brushing-tool/internal/settings"
	"github.com/safe-ai/excel-brushing-tool/internal/translation"
	"github.com/safe-ai/excel-brushing-tool/internal/upload"
	"github.com/safe-ai/excel-brushing-tool/migrations"
	"github.com/safe-ai/excel-brushing-tool/pkg/config"
	"github.com/safe-ai/excel-brushing-tool/pkg/database"
)

func main() {
	// 載入配置
	cfg := config.Load()

	// 連線資料庫（含重試機制）
	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("無法連線資料庫: %v", err)
	}
	defer pool.Close()

	// 執行資料庫 migration
	if err := migrations.Run(ctx, pool); err != nil {
		log.Fatalf("資料庫 migration 失敗: %v", err)
	}

	// 建立 Gin 引擎（不使用預設中介軟體）
	r := gin.New()

	// 註冊中介軟體
	r.Use(middleware.CORS())
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())

	// Health check endpoint
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Auth 模組初始化
	authRepo := auth.NewRepository(pool)
	authSvc := auth.NewService(authRepo, cfg.JWTSecret)
	authHandler := auth.NewHandler(authSvc)

	// Token blacklist（MVP: in-memory）
	tokenBlacklist := auth.NewTokenBlacklist()
	authHandler.SetBlacklist(tokenBlacklist)

	// Rate limiter（5 次失敗 / 15 分鐘）
	rateLimiter := auth.NewRateLimiter(pool, 5, 15*time.Minute)
	authHandler.SetRateLimiter(rateLimiter)

	// Public auth routes（不需要 JWT）
	r.POST("/api/auth/register", authHandler.Register)
	r.POST("/api/auth/login", authHandler.Login)

	// Upload 模組初始化
	uploadRepo := upload.NewRepository(pool)
	uploadSvc := upload.NewService(uploadRepo, cfg)
	uploadHandler := upload.NewHandler(uploadSvc)

	// Assessment 模組初始化
	assessRepo := assessment.NewRepository(pool)
	settingsRepo := assessment.NewSettingsRepository(pool)
	assessSvc := assessment.NewService(uploadRepo, assessRepo, settingsRepo)
	assessHandler := assessment.NewHandler(assessSvc)

	// Cleaning 模組初始化
	cleanRepo := cleaning.NewRepository(pool)
	cleanSvc := cleaning.NewService(cleanRepo, assessRepo, settingsRepo, uploadRepo, cfg)
	cleanHandler := cleaning.NewHandler(cleanSvc)

	// Export 模組初始化
	exportSvc := export.NewService(cleanRepo, assessRepo, cfg)
	exportHandler := export.NewHandler(exportSvc)

	// Comparison 模組初始化
	comparisonSvc := comparison.NewService(cleanRepo, assessRepo, assessSvc)
	comparisonHandler := comparison.NewHandler(comparisonSvc)

	// Evidence 模組初始化
	blockchainHTTPClient := &http.Client{Timeout: cfg.GetBlockchainTimeout()}
	blockchainClient := evidence.NewBlockchainClient(cfg.BlockchainURL, blockchainHTTPClient)
	evidenceRepo := evidence.NewRepository(pool)
	evidenceSvc := evidence.NewService(evidenceRepo, cleanRepo, exportSvc, blockchainClient, cfg)
	evidenceHandler := evidence.NewHandler(evidenceSvc)

	// QA 模組初始化
	geminiHTTPClient := &http.Client{Timeout: cfg.GetLLMTimeout()}
	geminiClient := qa.NewGeminiClient(cfg, geminiHTTPClient)
	qaSvc := qa.NewService(geminiClient, cleanRepo, assessRepo, uploadRepo, cfg)
	qaHandler := qa.NewHandler(qaSvc)

	// Settings 模組初始化
	settingsHandler := settings.NewHandler(pool)

	// Quota 模組初始化
	quotaRepo := quota.NewRepository(pool)
	quotaSvc := quota.NewService(quotaRepo)

	// Translation 模組初始化
	transRepo := translation.NewRepository(pool)
	transSvc := translation.NewService(transRepo)
	transHandler := translation.NewHandler(transSvc)

	// Admin 模組初始化
	adminHandler := admin.NewHandler(authRepo, quotaSvc, quotaRepo, transSvc, transRepo, assessRepo)

	// Public translation route（不需要認證）
	r.GET("/api/translations/:locale", transHandler.GetTranslations)

	// Protected routes（需要 JWT 認證）
	protected := r.Group("/api")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret, tokenBlacklist))
	{
		protected.POST("/auth/logout", authHandler.Logout)
		protected.GET("/auth/me", authHandler.GetMe)

		// Upload routes
		protected.POST("/upload", uploadHandler.Upload)
		protected.GET("/upload/:id/sheets", uploadHandler.GetSheets)
		protected.POST("/upload/:id/select-sheet", uploadHandler.SelectSheet)

		// Assessment routes
		protected.POST("/assess", assessHandler.RunAssessment)
		protected.GET("/assess/latest", assessHandler.GetLatest)
		protected.GET("/assess/:id", assessHandler.GetAssessment)
		protected.GET("/assess/:id/issues", assessHandler.GetIssues)

		// Cleaning routes
		protected.POST("/clean/preview-removals", cleanHandler.PreviewRemovals)
		protected.POST("/clean/apply", cleanHandler.ApplyRules)
		protected.POST("/clean/interactive", cleanHandler.ApplyInteractiveFix)
		protected.GET("/clean/latest", cleanHandler.GetLatest)
		protected.GET("/clean/:id/preview", cleanHandler.GetPreview)
		protected.GET("/clean/:id/log", cleanHandler.GetLog)

		// Export routes
		protected.GET("/export/:id/xlsx", exportHandler.DownloadExcel)
		protected.GET("/export/:id/pdf", exportHandler.DownloadPDF)
		protected.GET("/export/:id/log", exportHandler.DownloadLog)

		// Comparison routes
		protected.GET("/compare/:id", comparisonHandler.GetComparison)

		// Evidence routes
		protected.POST("/evidence/submit", evidenceHandler.Submit)
		protected.GET("/evidence/:record_id", evidenceHandler.Get)

		// QA routes
		protected.POST("/qa/ask", qaHandler.Ask)
		protected.GET("/qa/suggestions/:assess_id", qaHandler.GetSuggestions)

		// Settings routes
		protected.GET("/settings/weights", settingsHandler.GetWeights)
		protected.PUT("/settings/weights", settingsHandler.UpdateWeights)

		// Quota routes（一般使用者可查看自己的配額）
		protected.GET("/quota/me", func(c *gin.Context) {
			userIDVal, _ := c.Get("user_id")
			userID, ok := userIDVal.(uuid.UUID)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "無效的使用者"})
				return
			}
			info, err := quotaSvc.GetUserQuotaInfo(c.Request.Context(), userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "取得配額資訊失敗"})
				return
			}
			c.JSON(http.StatusOK, info)
		})
	}

	// Admin routes（需要 JWT + Admin 權限）
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(middleware.JWTAuth(cfg.JWTSecret, tokenBlacklist))
	adminGroup.Use(middleware.AdminAuth())
	{
		adminGroup.GET("/users", adminHandler.ListUsers)
		adminGroup.GET("/quota", adminHandler.GetQuotaSettings)
		adminGroup.PUT("/quota", adminHandler.UpdateQuotaSettings)
		adminGroup.GET("/translations", adminHandler.ListTranslations)
		adminGroup.PUT("/translations/:id", adminHandler.UpdateTranslation)
		adminGroup.GET("/assessments", adminHandler.ListAssessments)
	}

	// 建立 HTTP 伺服器
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// 啟動伺服器（非阻塞）
	go func() {
		log.Printf("伺服器啟動於 :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("伺服器啟動失敗: %v", err)
		}
	}()

	// 等待中斷信號以優雅關閉
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在關閉伺服器...")

	// 設定 5 秒的關閉超時
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("伺服器強制關閉: %v", err)
	}

	log.Println("伺服器已優雅關閉")
}
