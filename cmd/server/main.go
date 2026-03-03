package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/IanShaw027/sub2api-storage"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/sub2api/plugin-market/ent"
	v1 "github.com/sub2api/plugin-market/internal/api/v1"
	"github.com/sub2api/plugin-market/internal/api/v1/handler"
	"github.com/sub2api/plugin-market/internal/repository"
	"github.com/sub2api/plugin-market/internal/service"
)

func main() {
	log.Println("Sub2API Plugin Market Server starting...")

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

	pluginService := service.NewPluginService(pluginRepo)
	trustKeyService := service.NewTrustKeyService(trustKeyRepo)
	downloadService := service.NewDownloadService(pluginRepo, storageBackend, client)

	// 初始化签名验证服务
	hostRuntime := getEnv("HOST_RUNTIME", "wasm")
	hostAPIVersion := getEnv("HOST_API_VERSION", "1.0.0")
	verificationService, err := service.NewVerificationService(trustKeyRepo, hostRuntime, hostAPIVersion)
	if err != nil {
		log.Printf("Warning: Failed to initialize verification service: %v", err)
	} else {
		log.Println("Verification service initialized successfully")
	}
	_ = verificationService // 暂时未使用，后续可在下载时验证

	pluginHandler := handler.NewPluginHandler(pluginService)
	downloadHandler := handler.NewDownloadHandler(downloadService)
	trustKeyHandler := handler.NewTrustKeyHandler(trustKeyService)

	// 初始化 Gin 路由
	r := gin.Default()

	// 健康检查接口
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 注册 API 路由
	v1.RegisterRoutes(r, pluginHandler, downloadHandler, trustKeyHandler)

	// 启动服务器
	port := getEnv("PORT", "8080")
	addr := fmt.Sprintf(":%s", port)

	log.Printf("Server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// initDatabase 初始化数据库连接
func initDatabase() (*ent.Client, error) {
	// 从环境变量读取数据库配置
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
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

