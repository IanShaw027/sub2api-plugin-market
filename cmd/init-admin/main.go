package main

import (
	"context"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/internal/auth"
)

func main() {
	log.Println("Creating default admin user...")

	// 初始化数据库连接
	client, err := initDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer client.Close()

	// 创建管理员服务
	adminService := auth.NewAdminService(client)

	// 创建默认管理员
	username := getEnv("ADMIN_DEFAULT_USERNAME", "admin")
	email := getEnv("ADMIN_DEFAULT_EMAIL", "admin@sub2api.com")
	password := getEnv("ADMIN_DEFAULT_PASSWORD", "admin123")

	user, err := adminService.CreateAdmin(context.Background(), username, email, password, "super_admin")
	if err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	log.Printf("Admin user created successfully!")
	log.Printf("Username: %s", user.Username)
	log.Printf("Email: %s", user.Email)
	log.Printf("Password: %s", password)
	log.Printf("Role: %s", user.Role)
	log.Println("\n⚠️  请立即修改默认密码！")
}

func initDatabase() (*ent.Client, error) {
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

	// 确保 schema 存在
	if err := client.Schema.Create(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return client, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
