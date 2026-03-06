package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/pluginversion"
	"github.com/IanShaw027/sub2api-plugin-market/ent/syncjob"
	"github.com/IanShaw027/sub2api-storage"
	"github.com/google/uuid"
)

const autoSyncDedupWindow = 10 * time.Minute

var syncLocks sync.Map // key: "plugin_id:target_ref", value: *sync.Mutex

// SyncService 同步任务服务（MVP）
type SyncService struct {
	client         *ent.Client
	storage        storage.Storage
	httpClient     *http.Client // optional; nil means http.DefaultClient (for tests: inject mock)
	githubAPIBase  string        // optional; empty means "https://api.github.com" (for tests: inject mock server URL)
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
func NewSyncService(client *ent.Client, storage storage.Storage) *SyncService {
	return &SyncService{client: client, storage: storage}
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

	if err := s.runGitHubSync(ctx, uid, targetRef); err != nil {
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

	if err := s.runGitHubSync(ctx, job.PluginID, job.TargetRef); err != nil {
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

// githubReleaseAsset represents a release asset from GitHub API.
type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// githubRelease represents a release from GitHub API.
type githubRelease struct {
	TagName string               `json:"tag_name"`
	Assets []githubReleaseAsset  `json:"assets"`
}

func extractOwnerRepo(githubRepoURL string) string {
	normalized := NormalizeGitHubRepoURL(githubRepoURL)
	if normalized == "" {
		return ""
	}
	return strings.TrimPrefix(normalized, "github.com/")
}

func (s *SyncService) acquireSyncLock(pluginID, targetRef string) (unlock func(), err error) {
	key := pluginID + ":" + targetRef
	mu := &sync.Mutex{}
	actual, loaded := syncLocks.LoadOrStore(key, mu)
	actualMu := actual.(*sync.Mutex)

	if !actualMu.TryLock() {
		return nil, fmt.Errorf("concurrent sync in progress for plugin %s ref %s", pluginID, targetRef)
	}

	return func() {
		actualMu.Unlock()
		if !loaded {
			syncLocks.Delete(key)
		}
	}, nil
}

func (s *SyncService) runGitHubSync(ctx context.Context, pluginID uuid.UUID, targetRef string) error {
	pluginRecord, err := s.client.Plugin.Get(ctx, pluginID)
	if err != nil {
		return err
	}

	if pluginRecord.SourceType != plugin.SourceTypeGithub {
		return fmt.Errorf("plugin source_type must be github")
	}
	if pluginRecord.GithubRepoURL == "" {
		return fmt.Errorf("plugin github_repo_url is empty")
	}

	ownerRepo := extractOwnerRepo(pluginRecord.GithubRepoURL)
	if ownerRepo == "" {
		return fmt.Errorf("cannot extract owner/repo from %s", pluginRecord.GithubRepoURL)
	}

	// Resolve targetRef: if empty, fetch latest release tag
	if targetRef == "" {
		tag, err := s.fetchLatestReleaseTag(ctx, ownerRepo)
		if err != nil {
			return fmt.Errorf("fetch latest release: %w", err)
		}
		targetRef = tag
	}

	// S-06: Acquire process-level sync lock for this plugin+ref
	unlock, err := s.acquireSyncLock(pluginID.String(), targetRef)
	if err != nil {
		return err
	}
	defer unlock()

	// 1. Fetch release info
	release, err := s.fetchReleaseByTag(ctx, ownerRepo, targetRef)
	if err != nil {
		return fmt.Errorf("fetch release: %w", err)
	}

	// 2. Find .wasm asset
	var wasmAsset *githubReleaseAsset
	for i := range release.Assets {
		if strings.HasSuffix(strings.ToLower(release.Assets[i].Name), ".wasm") {
			wasmAsset = &release.Assets[i]
			break
		}
	}
	if wasmAsset == nil {
		return fmt.Errorf("no .wasm asset found in release %s", targetRef)
	}

	// 3. Check version doesn't already exist (S-07: moved before download/upload)
	ok, err := s.versionAvailable(ctx, pluginID, targetRef)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("plugin version %s already exists", targetRef)
	}

	// 4. Download the .wasm asset
	wasmBytes, err := s.downloadAsset(ctx, wasmAsset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("download wasm asset: %w", err)
	}

	// 5. Compute SHA256 hash
	hash := sha256.Sum256(wasmBytes)
	wasmHash := "sha256-" + hex.EncodeToString(hash[:])
	fileSize := len(wasmBytes)

	// 6. Upload to storage
	storageKey := fmt.Sprintf("plugins/%s/%s/plugin.wasm", pluginRecord.Name, targetRef)
	if _, err := s.storage.Upload(ctx, storageKey, bytes.NewReader(wasmBytes)); err != nil {
		return fmt.Errorf("upload wasm to storage: %w", err)
	}

	// 7. Create PluginVersion — cleanup orphan WASM on failure (S-07)
	_, err = s.client.PluginVersion.Create().
		SetPluginID(pluginID).
		SetVersion(targetRef).
		SetStatus(pluginversion.StatusDraft).
		SetWasmURL(storageKey).
		SetWasmHash(wasmHash).
		SetSignature("").
		SetFileSize(fileSize).
		SetMinAPIVersion("1.0.0").
		SetPluginAPIVersion("1.0.0").
		Save(ctx)
	if err != nil {
		if delErr := s.storage.Delete(ctx, storageKey); delErr != nil {
			slog.Error("failed to cleanup orphan wasm", "key", storageKey, "error", delErr)
		}
		return fmt.Errorf("create plugin version: %w", err)
	}

	return nil
}

func (s *SyncService) getHTTPClient() *http.Client {
	if s.httpClient != nil {
		return s.httpClient
	}
	return http.DefaultClient
}

func (s *SyncService) getGitHubAPIBase() string {
	if base := strings.TrimSuffix(s.githubAPIBase, "/"); base != "" {
		return base
	}
	return "https://api.github.com"
}

func (s *SyncService) fetchLatestReleaseTag(ctx context.Context, ownerRepo string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", s.getGitHubAPIBase(), ownerRepo)
	release, err := s.fetchRelease(ctx, url)
	if err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("latest release has no tag_name")
	}
	return release.TagName, nil
}

func (s *SyncService) fetchReleaseByTag(ctx context.Context, ownerRepo, tag string) (*githubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/tags/%s", s.getGitHubAPIBase(), ownerRepo, tag)
	return s.fetchRelease(ctx, url)
}

func (s *SyncService) fetchRelease(ctx context.Context, url string) (*githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.getHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api %s: %s - %s", url, resp.Status, string(body))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func (s *SyncService) downloadAsset(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.getHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download %s: %s - %s", url, resp.Status, string(body))
	}

	return io.ReadAll(resp.Body)
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
