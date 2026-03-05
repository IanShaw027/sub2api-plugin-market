//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/submission"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("创建测试数据...")

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

	ctx := context.Background()

	// 创建测试插件
	plugins := []struct {
		name        string
		displayName string
		description string
	}{
		{"openai-adapter", "OpenAI 适配器", "将 OpenAI API 转换为统一格式"},
		{"claude-adapter", "Claude 适配器", "将 Claude API 转换为统一格式"},
		{"gemini-adapter", "Gemini 适配器", "将 Gemini API 转换为统一格式"},
	}

	for _, p := range plugins {
		// 创建插件
		plugin, err := client.Plugin.Create().
			SetID(uuid.New()).
			SetName(p.name).
			SetDisplayName(p.displayName).
			SetDescription(p.description).
			SetAuthor("测试作者").
			SetRepositoryURL("https://github.com/example/" + p.name).
			SetHomepageURL("https://example.com/" + p.name).
			SetLicense("MIT").
			SetCategory("proxy").
			SetDownloadCount(0).
			Save(ctx)

		if err != nil {
			log.Printf("创建插件 %s 失败: %v", p.name, err)
			continue
		}
		log.Printf("创建插件: %s", p.name)

		// 创建提交记录
		statuses := []submission.Status{
			submission.StatusPending,
			submission.StatusApproved,
			submission.StatusRejected,
		}

		for i, status := range statuses {
			_, err := client.Submission.Create().
				SetID(uuid.New()).
				SetPluginID(plugin.ID).
				SetSubmissionType(submission.SubmissionTypeNewVersion).
				SetSubmitterEmail(fmt.Sprintf("submitter%d@example.com", i+1)).
				SetSubmitterName(fmt.Sprintf("提交者%d", i+1)).
				SetNotes(fmt.Sprintf("这是一个测试提交 - %s", status)).
				SetStatus(status).
				SetCreatedAt(time.Now().Add(-time.Duration(i*24) * time.Hour)).
				Save(ctx)

			if err != nil {
				log.Printf("创建提交记录失败: %v", err)
				continue
			}
		}
	}

	// 创建一些额外的待审核提交
	for i := 0; i < 5; i++ {
		_, err := client.Submission.Create().
			SetID(uuid.New()).
			SetPluginID(uuid.New()).
			SetSubmissionType(submission.SubmissionTypeNewPlugin).
			SetSubmitterEmail(fmt.Sprintf("user%d@example.com", i+1)).
			SetSubmitterName(fmt.Sprintf("用户%d", i+1)).
			SetNotes(fmt.Sprintf("新插件提交 #%d - 请审核", i+1)).
			SetStatus(submission.StatusPending).
			SetCreatedAt(time.Now().Add(-time.Duration(i*2) * time.Hour)).
			Save(ctx)

		if err != nil {
			log.Printf("创建待审核提交失败: %v", err)
			continue
		}
	}

	log.Println("测试数据创建完成！")
	log.Println("现在可以访问管理后台查看数据了")
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
