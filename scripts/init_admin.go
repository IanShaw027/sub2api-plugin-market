//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/internal/auth"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("初始化管理员账号...")

	// 从环境变量读取数据库配置
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5433")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "plugin_market")
	sslMode := getEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, sslMode)

	// 连接数据库
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer client.Close()

	// 运行数据库迁移
	ctx := context.Background()
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("创建数据库表失败: %v", err)
	}
	log.Println("数据库表创建成功")

	// 读取管理员配置
	adminUsername := getEnv("ADMIN_USERNAME", "admin")
	adminPassword := getEnv("ADMIN_PASSWORD", "admin123")
	adminEmail := getEnv("ADMIN_EMAIL", "admin@sub2api.com")

	// 创建管理员服务
	adminService := auth.NewAdminService(client)

	// 检查管理员是否已存在
	existingAdmin, err := adminService.GetByUsername(ctx, adminUsername)
	if err == nil && existingAdmin != nil {
		log.Printf("管理员账号 '%s' 已存在，跳过创建", adminUsername)
		return
	}

	// 创建超级管理员
	admin, err := adminService.CreateAdmin(ctx, adminUsername, adminEmail, adminPassword, "super_admin")
	if err != nil {
		log.Fatalf("创建管理员失败: %v", err)
	}

	log.Printf("管理员账号创建成功:")
	log.Printf("  用户名: %s", admin.Username)
	log.Printf("  邮箱: %s", admin.Email)
	log.Printf("  角色: %s", admin.Role)
	log.Printf("  密码: %s", adminPassword)
	log.Println("\n请妥善保管管理员密码，并在生产环境中修改默认密码！")
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
