package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/pluginversion"
	"github.com/IanShaw027/sub2api-plugin-market/ent/syncjob"
	"github.com/google/uuid"
)

const autoSyncDedupWindow = 10 * time.Minute

// SyncService 同步任务服务（MVP）
type SyncService struct {
	client *ent.Client
}

// ListSyncJobsParams 同步任务列表查询参数
type ListSyncJobsParams struct {
	Status      string
	PluginID    string
	TriggerType string
	Page        int
	PageSize    int
	From        *time.Time
	To          *time.Time
}

// NewSyncService 创建同步服务
func NewSyncService(client *ent.Client) *SyncService {
	return &SyncService{client: client}
}

// CreateAndRunManualSync 创建并立即执行手动同步任务
func (s *SyncService) CreateAndRunManualSync(ctx context.Context, pluginID, targetRef string) (*ent.SyncJob, error) {
	return s.createAndRunSync(ctx, pluginID, targetRef, syncjob.TriggerTypeManual)
}

// CreateAndRunAutoSync 创建并立即执行自动同步任务
func (s *SyncService) CreateAndRunAutoSync(ctx context.Context, pluginID, targetRef string) (*ent.SyncJob, error) {
	return s.createAndRunSync(ctx, pluginID, targetRef, syncjob.TriggerTypeAuto)
}

// EnqueueAutoSync 创建自动同步任务并返回 pending 状态
func (s *SyncService) EnqueueAutoSync(ctx context.Context, pluginID, targetRef string) (*ent.SyncJob, error) {
	uid, err := uuid.Parse(pluginID)
	if err != nil {
		return nil, err
	}

	targetRef = strings.TrimSpace(targetRef)
	create := s.client.SyncJob.Create().
		SetPluginID(uid).
		SetTriggerType(syncjob.TriggerTypeAuto).
		SetStatus(syncjob.StatusPending)
	if targetRef != "" {
		create = create.SetTargetRef(targetRef)
	}

	job, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	slog.Info("enqueued auto sync", "job_id", job.ID, "plugin_id", job.PluginID, "target_ref", targetRef)
	return job, nil
}

// ProcessSyncJobWithRetry 按 job_id 执行同步并带重试
func (s *SyncService) ProcessSyncJobWithRetry(ctx context.Context, jobID string, maxAttempts int, retryDelay time.Duration) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if retryDelay <= 0 {
		retryDelay = 2 * time.Second
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		slog.Info("processing sync", "job_id", jobID, "attempt", attempt, "max_attempts", maxAttempts)
		err := s.processSyncJobOnce(ctx, jobID)
		if err == nil {
			slog.Info("sync succeeded", "job_id", jobID, "attempt", attempt, "max_attempts", maxAttempts)
			return
		}

		slog.Warn("sync attempt failed", "job_id", jobID, "attempt", attempt, "max_attempts", maxAttempts, "error", err)
		if attempt == maxAttempts {
			slog.Error("sync failed permanently", "job_id", jobID, "attempts", maxAttempts)
			return
		}

		time.Sleep(retryDelay)
	}
}

// IsDuplicateAutoSync 判断是否为短窗口内重复自动同步
func (s *SyncService) IsDuplicateAutoSync(ctx context.Context, pluginID, targetRef string) (bool, error) {
	uid, err := uuid.Parse(pluginID)
	if err != nil {
		return false, err
	}

	targetRef = strings.TrimSpace(targetRef)
	if targetRef == "" {
		return false, nil
	}

	count, err := s.client.SyncJob.Query().
		Where(
			syncjob.PluginIDEQ(uid),
			syncjob.TriggerTypeEQ(syncjob.TriggerTypeAuto),
			syncjob.TargetRefEQ(targetRef),
			syncjob.StatusIn(syncjob.StatusPending, syncjob.StatusRunning, syncjob.StatusSucceeded),
			syncjob.CreatedAtGTE(time.Now().Add(-autoSyncDedupWindow)),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// FindPluginByGitHubRepoURL 根据 github_repo_url 匹配插件
func (s *SyncService) FindPluginByGitHubRepoURL(ctx context.Context, repoURL string) (*ent.Plugin, error) {
	normalized := NormalizeGitHubRepoURL(repoURL)
	if normalized == "" {
		return nil, &ent.NotFoundError{}
	}

	p, err := s.client.Plugin.Query().
		Where(
			plugin.SourceTypeEQ(plugin.SourceTypeGithub),
			plugin.StatusEQ(plugin.StatusActive),
			plugin.GithubRepoNormalizedEQ(normalized),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, &ent.NotFoundError{}
		}
		return nil, err
	}

	return p, nil
}

func (s *SyncService) createAndRunSync(ctx context.Context, pluginID, targetRef string, triggerType syncjob.TriggerType) (*ent.SyncJob, error) {
	uid, err := uuid.Parse(pluginID)
	if err != nil {
		return nil, err
	}

	targetRef = strings.TrimSpace(targetRef)
	create := s.client.SyncJob.Create().
		SetPluginID(uid).
		SetTriggerType(triggerType).
		SetStatus(syncjob.StatusPending)
	if targetRef != "" {
		create = create.SetTargetRef(targetRef)
	}

	job, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	job, err = job.Update().
		SetStatus(syncjob.StatusRunning).
		SetStartedAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.runPseudoSync(ctx, uid); err != nil {
		finishedAt := time.Now()
		_, _ = job.Update().
			SetStatus(syncjob.StatusFailed).
			SetErrorMessage(err.Error()).
			SetFinishedAt(finishedAt).
			Save(ctx)
		return nil, err
	}

	finishedAt := time.Now()
	job, err = job.Update().
		SetStatus(syncjob.StatusSucceeded).
		SetFinishedAt(finishedAt).
		ClearErrorMessage().
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// GetSyncJob 获取同步任务
func (s *SyncService) GetSyncJob(ctx context.Context, id string) (*ent.SyncJob, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.client.SyncJob.Get(ctx, uid)
}

// ListSyncJobs 获取同步任务列表
func (s *SyncService) ListSyncJobs(ctx context.Context, params ListSyncJobsParams) ([]*ent.SyncJob, int, error) {
	query := s.client.SyncJob.Query()

	if params.Status != "" {
		query = query.Where(syncjob.StatusEQ(syncjob.Status(params.Status)))
	}

	if params.PluginID != "" {
		pluginUID, err := uuid.Parse(params.PluginID)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where(syncjob.PluginIDEQ(pluginUID))
	}

	if params.TriggerType != "" {
		query = query.Where(syncjob.TriggerTypeEQ(syncjob.TriggerType(params.TriggerType)))
	}

	if params.From != nil {
		query = query.Where(syncjob.CreatedAtGTE(*params.From))
	}

	if params.To != nil {
		query = query.Where(syncjob.CreatedAtLTE(*params.To))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	jobs, err := query.
		Order(ent.Desc(syncjob.FieldCreatedAt)).
		Offset((params.Page - 1) * params.PageSize).
		Limit(params.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

func (s *SyncService) processSyncJobOnce(ctx context.Context, jobID string) error {
	uid, err := uuid.Parse(jobID)
	if err != nil {
		return err
	}

	job, err := s.client.SyncJob.Get(ctx, uid)
	if err != nil {
		return err
	}

	now := time.Now()
	job, err = job.Update().
		SetStatus(syncjob.StatusRunning).
		SetStartedAt(now).
		Save(ctx)
	if err != nil {
		return err
	}

	if err := s.runPseudoSync(ctx, job.PluginID); err != nil {
		finishedAt := time.Now()
		_, _ = job.Update().
			SetStatus(syncjob.StatusFailed).
			SetErrorMessage(err.Error()).
			SetFinishedAt(finishedAt).
			Save(ctx)
		return err
	}

	finishedAt := time.Now()
	_, err = job.Update().
		SetStatus(syncjob.StatusSucceeded).
		SetFinishedAt(finishedAt).
		ClearErrorMessage().
		Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *SyncService) runPseudoSync(ctx context.Context, pluginID uuid.UUID) error {
	pluginRecord, err := s.client.Plugin.Get(ctx, pluginID)
	if err != nil {
		return err
	}

	if pluginRecord.SourceType != plugin.SourceTypeGithub {
		return fmt.Errorf("plugin source_type 必须为 github")
	}

	latestVersion, err := s.client.PluginVersion.Query().
		Where(pluginversion.PluginIDEQ(pluginID)).
		Order(ent.Desc(pluginversion.FieldCreatedAt)).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if ent.IsNotFound(err) {
		latestVersion = nil
	}

	newVersion, err := s.buildSyncVersion(ctx, pluginID, latestVersion)
	if err != nil {
		return err
	}

	_, err = s.client.PluginVersion.Create().
		SetPluginID(pluginID).
		SetVersion(newVersion).
		SetStatus(pluginversion.StatusDraft).
		SetWasmURL("sync/manual-placeholder.wasm").
		SetWasmHash("sha256-sync-placeholder").
		SetSignature("signature-sync-placeholder").
		SetFileSize(1).
		SetMinAPIVersion("1.0.0").
		SetPluginAPIVersion("1.0.0").
		Save(ctx)
	return err
}

func (s *SyncService) buildSyncVersion(ctx context.Context, pluginID uuid.UUID, latest *ent.PluginVersion) (string, error) {
	if latest == nil {
		base := "v0.0.1-sync"
		ok, err := s.versionAvailable(ctx, pluginID, base)
		if err != nil {
			return "", err
		}
		if ok {
			return base, nil
		}
	}

	base := "v0.0.1"
	if latest != nil {
		base = latest.Version
	}

	for i := 0; i < 5; i++ {
		candidate := fmt.Sprintf("%s-sync-%d", base, time.Now().Unix()+int64(i))
		ok, err := s.versionAvailable(ctx, pluginID, candidate)
		if err != nil {
			return "", err
		}
		if ok {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("生成同步版本号失败")
}

func (s *SyncService) versionAvailable(ctx context.Context, pluginID uuid.UUID, version string) (bool, error) {
	count, err := s.client.PluginVersion.Query().
		Where(
			pluginversion.PluginIDEQ(pluginID),
			pluginversion.VersionEQ(version),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// NormalizeGitHubRepoURL normalizes a GitHub repo URL to a canonical form for lookup.
func NormalizeGitHubRepoURL(raw string) string {
	url := strings.TrimSpace(strings.ToLower(raw))
	if url == "" {
		return ""
	}

	url = strings.TrimSuffix(url, "/")

	switch {
	case strings.HasPrefix(url, "https://github.com/"):
		url = "github.com/" + strings.TrimPrefix(url, "https://github.com/")
	case strings.HasPrefix(url, "http://github.com/"):
		url = "github.com/" + strings.TrimPrefix(url, "http://github.com/")
	case strings.HasPrefix(url, "git@github.com:"):
		url = "github.com/" + strings.TrimPrefix(url, "git@github.com:")
	case strings.HasPrefix(url, "ssh://git@github.com/"):
		url = "github.com/" + strings.TrimPrefix(url, "ssh://git@github.com/")
	case strings.HasPrefix(url, "github.com/"):
		// no-op
	default:
		return ""
	}

	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	path := strings.TrimPrefix(url, "github.com/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return ""
	}

	return "github.com/" + parts[0] + "/" + parts[1]
}
