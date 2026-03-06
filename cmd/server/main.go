package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/internal/admin"
	adminHandler "github.com/IanShaw027/sub2api-plugin-market/internal/admin/handler"
	adminService "github.com/IanShaw027/sub2api-plugin-market/internal/admin/service"
	v1 "github.com/IanShaw027/sub2api-plugin-market/internal/api/v1"
	"github.com/IanShaw027/sub2api-plugin-market/internal/api/v1/handler"
	"github.com/IanShaw027/sub2api-plugin-market/internal/auth"
	"github.com/IanShaw027/sub2api-plugin-market/internal/repository"
	"github.com/IanShaw027/sub2api-plugin-market/internal/service"
	"github.com/IanShaw027/sub2api-storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const weakAdminJWTSecret = "your-secret-key-change-in-production"

func main() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	log.Println("Sub2API Plugin Market Server starting...")

	// 创建 shutdown 上下文，用于 goroutine 生命周期管理及优雅关闭
	shutdownCtx, shutdownCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer shutdownCancel()

	// 启动早期校验管理后台 JWT 密钥，避免被后续错误掩盖
	jwtSecret, err := resolveAdminJWTSecret()
	if err != nil {
		log.Fatalf("Invalid ADMIN_JWT_SECRET: %v", err)
	}

	// 初始化数据库连接
	client, err := initDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer client.Close()

	// 初始化 Storage
	storageBackend, err := storage.NewStorageFromEnv()
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	log.Println("Storage initialized successfully")

	// 初始化依赖
	pluginRepo := repository.NewPluginRepository(client)
	trustKeyRepo := repository.NewTrustKeyRepository(client)

	// 初始化签名验证服务（下载链路强制依赖）
	hostRuntime := getEnv("HOST_RUNTIME", "wasm")
	hostAPIVersion := getEnv("HOST_API_VERSION", "1.0.0")
	verificationService, err := service.NewVerificationService(trustKeyRepo, hostRuntime, hostAPIVersion)
	if err != nil {
		log.Fatalf("Failed to initialize verification service: %v", err)
	}
	log.Println("Verification service initialized successfully")

	pluginService := service.NewPluginService(pluginRepo)
	trustKeyService := service.NewTrustKeyService(trustKeyRepo)
	downloadService := service.NewDownloadService(pluginRepo, storageBackend, client, verificationService)
	submissionService := service.NewSubmissionService(client, storageBackend)
	syncService := service.NewSyncService(client, storageBackend)

	pluginHandler := handler.NewPluginHandler(pluginService)
	downloadHandler := handler.NewDownloadHandler(downloadService)
	trustKeyHandler := handler.NewTrustKeyHandler(trustKeyService)
	v1SubmissionHandler := handler.NewSubmissionHandler(submissionService)
	githubWebhookSecret := strings.TrimSpace(os.Getenv("GITHUB_WEBHOOK_SECRET"))
	if githubWebhookSecret == "" {
		log.Println("GitHub webhook secret is empty; signature verification is disabled")
	}
	githubWebhookSyncMaxAttempts := getEnvIntWithMin("GITHUB_WEBHOOK_SYNC_MAX_ATTEMPTS", 3, 1)
	githubWebhookSyncRetryDelaySeconds := getEnvIntWithMin("GITHUB_WEBHOOK_SYNC_RETRY_DELAY_SECONDS", 2, 1)
	githubWebhookHandler := handler.NewGitHubWebhookHandler(
		syncService,
		githubWebhookSecret,
		githubWebhookSyncMaxAttempts,
		githubWebhookSyncRetryDelaySeconds,
		shutdownCtx,
	)

	// 初始化管理后台服务
	jwtExpireHours := 2
	jwtRefreshExpireDays := 7
	jwtService := auth.NewJWTService(jwtSecret, jwtExpireHours, jwtRefreshExpireDays)
	authService := auth.NewAdminService(client)
	adminSubmissionService := adminService.NewSubmissionService(client)

	authHandler := adminHandler.NewAuthHandler(authService, jwtService)
	submissionHandler := adminHandler.NewSubmissionHandler(adminSubmissionService)
	syncHandler := adminHandler.NewSyncHandler(syncService)
	adminPluginHandler := adminHandler.NewAdminPluginHandler(client)
	adminVersionHandler := adminHandler.NewAdminVersionHandler(client)

	// 初始化 Gin 路由
	r := gin.Default()

	// 健康检查接口（含 DB 连通性）
	r.GET("/health", func(c *gin.Context) {
		result := gin.H{"status": "ok"}
		httpStatus := 200

		if _, err := client.Plugin.Query().Limit(1).All(c.Request.Context()); err != nil {
			result["db"] = "error: " + err.Error()
			result["status"] = "degraded"
			httpStatus = 503
		} else {
			result["db"] = "ok"
		}

		c.JSON(httpStatus, result)
	})

	// 注册 API 路由
	v1.RegisterRoutes(r, pluginHandler, downloadHandler, trustKeyHandler, v1SubmissionHandler, githubWebhookHandler)

	// 注册管理后台路由
	admin.RegisterRoutes(r, authHandler, submissionHandler, syncHandler, adminPluginHandler, adminVersionHandler, jwtService, authService)

	// 启动服务器（支持优雅关闭）
	port := getEnv("PORT", "8081")
	addr := fmt.Sprintf(":%s", port)

	srv := &http.Server{Addr: addr, Handler: r}
	go func() {
		log.Printf("Server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-shutdownCtx.Done()
	log.Println("Shutting down server...")
	shutdownTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownTimeout); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}

// initDatabase 初始化数据库连接
func initDatabase() (*ent.Client, error) {
	// 从环境变量读取数据库配置
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5433")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "plugin_market")
	sslMode := getEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, sslMode)

	client, err := ent.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := client.Schema.Create(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	log.Println("Database connected successfully")
	return client, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntWithMin(key string, defaultValue, minValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("Warning: invalid %s=%q, fallback to default=%d", key, raw, defaultValue)
		return defaultValue
	}
	if parsed < minValue {
		log.Printf("Warning: %s=%d below min=%d, use min value", key, parsed, minValue)
		return minValue
	}
	return parsed
}

// resolveAdminJWTSecret 解析并校验 ADMIN_JWT_SECRET
func resolveAdminJWTSecret() (string, error) {
	secret := strings.TrimSpace(os.Getenv("ADMIN_JWT_SECRET"))
	if secret == "" {
		return "", fmt.Errorf("missing ADMIN_JWT_SECRET, please set a strong secret (example: `openssl rand -base64 32`)")
	}

	if gin.Mode() == gin.ReleaseMode && secret == weakAdminJWTSecret {
		return "", fmt.Errorf("weak default secret is not allowed when GIN_MODE=release")
	}

	if secret == weakAdminJWTSecret {
		log.Println("Warning: ADMIN_JWT_SECRET is using a weak default value; do not use it in production")
	}

	return secret, nil
}
