package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/gin-gonic/gin"
)

// GitHubWebhookSyncService 定义 webhook 触发同步需要的最小能力
// 使用接口便于 handler 单测注入 mock service。
type GitHubWebhookSyncService interface {
	FindPluginByGitHubRepoURL(ctx context.Context, repoURL string) (*ent.Plugin, error)
	IsDuplicateAutoSync(ctx context.Context, pluginID, targetRef string) (bool, error)
	EnqueueAutoSync(ctx context.Context, pluginID, targetRef string) (*ent.SyncJob, error)
	ProcessSyncJobWithRetry(ctx context.Context, jobID string, maxAttempts int, retryDelay time.Duration)
}

// GitHubWebhookHandler GitHub webhook 处理器
type GitHubWebhookHandler struct {
	syncService       GitHubWebhookSyncService
	secret            string
	retryMaxAttempts  int
	retryDelaySeconds int
	shutdownCtx       context.Context
}

// GitHubReleaseWebhookPayload GitHub release webhook 最小解析结构
type GitHubReleaseWebhookPayload struct {
	Action     string `json:"action"`
	Repository struct {
		HTMLURL string `json:"html_url"`
	} `json:"repository"`
	Release struct {
		TagName string `json:"tag_name"`
	} `json:"release"`
}

// NewGitHubWebhookHandler 创建 GitHub webhook 处理器。
// shutdownCtx 可选，用于 goroutine 生命周期管理；未提供时使用 context.Background()。
func NewGitHubWebhookHandler(syncService GitHubWebhookSyncService, secret string, retryMaxAttempts, retryDelaySeconds int, shutdownCtx ...context.Context) *GitHubWebhookHandler {
	if retryMaxAttempts < 1 {
		retryMaxAttempts = 1
	}
	if retryDelaySeconds < 1 {
		retryDelaySeconds = 1
	}
	ctx := context.Background()
	if len(shutdownCtx) > 0 && shutdownCtx[0] != nil {
		ctx = shutdownCtx[0]
	}
	return &GitHubWebhookHandler{
		syncService:       syncService,
		secret:            strings.TrimSpace(secret),
		retryMaxAttempts:  retryMaxAttempts,
		retryDelaySeconds: retryDelaySeconds,
		shutdownCtx:       ctx,
	}
}

// HandleGitHubWebhook 处理 GitHub webhook
// POST /api/v1/integrations/github/webhook
func (h *GitHubWebhookHandler) HandleGitHubWebhook(c *gin.Context) {
	eventType := strings.TrimSpace(c.GetHeader("X-GitHub-Event"))
	if eventType == "" {
		Error(c, ErrCodeInvalidParam, "缺少 X-GitHub-Event 请求头")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		Error(c, ErrCodeInvalidParam, "读取请求体失败")
		return
	}

	if h.secret == "" && gin.Mode() == gin.ReleaseMode {
		Error(c, ErrCodeForbidden, "webhook secret 未配置，生产环境拒绝处理")
		return
	}

	if h.secret != "" {
		signature := strings.TrimSpace(c.GetHeader("X-Hub-Signature-256"))
		if !verifyGitHubSignature(body, h.secret, signature) {
			Error(c, ErrCodeInvalidParam, "webhook 签名校验失败")
			return
		}
	}

	if eventType != "release" {
		slog.Info("ignored: non-release event", "event", eventType)
		c.JSON(http.StatusOK, Response{Code: 0, Message: "ignored"})
		return
	}

	var payload GitHubReleaseWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		Error(c, ErrCodeInvalidParam, "webhook payload 格式错误")
		return
	}

	action := strings.TrimSpace(payload.Action)
	repoURL := strings.TrimSpace(payload.Repository.HTMLURL)
	targetRef := strings.TrimSpace(payload.Release.TagName)
	slog.Info("received event", "event", eventType, "action", action, "repo", repoURL, "tag", targetRef)

	if action != "published" {
		slog.Info("ignored: non-published action", "action", action)
		c.JSON(http.StatusOK, Response{Code: 0, Message: "ignored"})
		return
	}

	if repoURL == "" {
		Error(c, ErrCodeInvalidParam, "repository.html_url 不能为空")
		return
	}

	if targetRef == "" {
		Error(c, ErrCodeInvalidParam, "release.tag_name 不能为空")
		return
	}

	pluginRecord, err := h.syncService.FindPluginByGitHubRepoURL(c.Request.Context(), repoURL)
	if err != nil {
		if ent.IsNotFound(err) {
			slog.Info("ignored: plugin not matched", "repo", repoURL, "tag", targetRef)
			c.JSON(http.StatusOK, Response{Code: 0, Message: "ignored"})
			return
		}
		slog.Error("failed: find plugin", "error", err)
		Error(c, ErrCodeDatabaseError, "查询插件失败")
		return
	}

	duplicate, err := h.syncService.IsDuplicateAutoSync(c.Request.Context(), pluginRecord.ID.String(), targetRef)
	if err != nil {
		slog.Error("failed: duplicate check", "plugin_id", pluginRecord.ID, "tag", targetRef, "error", err)
		Error(c, ErrCodeDatabaseError, "查询插件失败")
		return
	}
	if duplicate {
		slog.Info("ignored: duplicate auto sync", "plugin_id", pluginRecord.ID, "tag", targetRef)
		c.JSON(http.StatusOK, Response{Code: 0, Message: "ignored"})
		return
	}

	job, err := h.syncService.EnqueueAutoSync(c.Request.Context(), pluginRecord.ID.String(), targetRef)
	if err != nil {
		slog.Error("failed: enqueue auto sync", "plugin_id", pluginRecord.ID, "tag", targetRef, "error", err)
		Error(c, ErrCodeInternalError, "自动同步失败")
		return
	}

	slog.Info("enqueued", "plugin_id", pluginRecord.ID, "sync_job_id", job.ID, "tag", targetRef)
	go h.syncService.ProcessSyncJobWithRetry(
		h.shutdownCtx,
		job.ID.String(),
		h.retryMaxAttempts,
		time.Duration(h.retryDelaySeconds)*time.Second,
	)

	Success(c, gin.H{
		"sync_job_id":  job.ID,
		"plugin_id":    pluginRecord.ID,
		"trigger_type": job.TriggerType,
		"status":       job.Status,
	})
}

func verifyGitHubSignature(body []byte, secret, signature string) bool {
	if strings.TrimSpace(signature) == "" {
		return false
	}
	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return false
	}

	expectedMAC := hmac.New(sha256.New, []byte(secret))
	expectedMAC.Write(body)
	expected := expectedMAC.Sum(nil)

	provided, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	return hmac.Equal(expected, provided)
}
