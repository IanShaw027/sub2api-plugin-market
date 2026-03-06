package service

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/pluginversion"
	"github.com/IanShaw027/sub2api-plugin-market/ent/syncjob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// syncTestFakeStorage implements storage.Storage for sync service tests.
type syncTestFakeStorage struct{}

func (s *syncTestFakeStorage) Upload(_ context.Context, _ string, _ io.Reader) (string, error) {
	return "https://example.com/uploaded", nil
}
func (s *syncTestFakeStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBufferString("wasm-binary:"+key)), nil
}
func (s *syncTestFakeStorage) GetPresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return fmt.Sprintf("https://example.com/download%s", key), nil
}
func (s *syncTestFakeStorage) Delete(_ context.Context, _ string) error { return nil }
func (s *syncTestFakeStorage) Exists(_ context.Context, _ string) (bool, error) { return true, nil }

func TestNormalizeGitHubRepoURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "https 标准 URL",
			input: "https://github.com/org/repo",
			want:  "github.com/org/repo",
		},
		{
			name:  "https 尾部斜杠",
			input: "https://github.com/org/repo/",
			want:  "github.com/org/repo",
		},
		{
			name:  "https git 后缀",
			input: "https://github.com/org/repo.git",
			want:  "github.com/org/repo",
		},
		{
			name:  "ssh scp 风格",
			input: "git@github.com:org/repo.git",
			want:  "github.com/org/repo",
		},
		{
			name:  "ssh URL 风格",
			input: "ssh://git@github.com/org/repo.git",
			want:  "github.com/org/repo",
		},
		{
			name:  "大小写归一",
			input: "HTTPS://GITHUB.COM/Org/Repo.GIT",
			want:  "github.com/org/repo",
		},
		{
			name:  "非 github 地址",
			input: "https://gitlab.com/org/repo",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeGitHubRepoURL(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSyncService_IsDuplicateAutoSync(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:sync_service_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewSyncService(client, &syncTestFakeStorage{})
	ctx := context.Background()

	p, err := client.Plugin.Create().
		SetName("sync-test-plugin").
		SetDisplayName("Sync Test Plugin").
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryOther).
		SetSourceType(plugin.SourceTypeGithub).
		SetStatus(plugin.StatusActive).
		SetGithubRepoURL("https://github.com/example/sync-test-plugin").
		Save(ctx)
	require.NoError(t, err)

	targetRef := "v1.2.3"
	_, err = client.SyncJob.Create().
		SetPluginID(p.ID).
		SetTriggerType(syncjob.TriggerTypeAuto).
		SetStatus(syncjob.StatusSucceeded).
		SetTargetRef(targetRef).
		SetCreatedAt(time.Now().Add(-5 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	dup, err := svc.IsDuplicateAutoSync(ctx, p.ID.String(), targetRef)
	require.NoError(t, err)
	assert.True(t, dup)

	_, err = client.SyncJob.Create().
		SetPluginID(p.ID).
		SetTriggerType(syncjob.TriggerTypeAuto).
		SetStatus(syncjob.StatusFailed).
		SetTargetRef("v2.0.0").
		SetCreatedAt(time.Now().Add(-2 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	dup, err = svc.IsDuplicateAutoSync(ctx, p.ID.String(), "v2.0.0")
	require.NoError(t, err)
	assert.False(t, dup)

	_, err = client.SyncJob.Create().
		SetPluginID(p.ID).
		SetTriggerType(syncjob.TriggerTypeAuto).
		SetStatus(syncjob.StatusSucceeded).
		SetTargetRef("v3.0.0").
		SetCreatedAt(time.Now().Add(-11 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	dup, err = svc.IsDuplicateAutoSync(ctx, p.ID.String(), "v3.0.0")
	require.NoError(t, err)
	assert.False(t, dup)
}

func TestSyncService_FindPluginByGitHubRepoURL_WithNormalizedFormats(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:sync_service_find_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewSyncService(client, &syncTestFakeStorage{})
	ctx := context.Background()

	created, err := client.Plugin.Create().
		SetName("repo-match-plugin").
		SetDisplayName("Repo Match Plugin").
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryOther).
		SetSourceType(plugin.SourceTypeGithub).
		SetStatus(plugin.StatusActive).
		SetGithubRepoURL("git@github.com:Org/Repo.git").
		SetGithubRepoNormalized(NormalizeGitHubRepoURL("git@github.com:Org/Repo.git")).
		Save(ctx)
	require.NoError(t, err)

	queries := []string{
		"https://github.com/org/repo",
		"https://github.com/org/repo/",
		"https://github.com/org/repo.git",
		"git@github.com:org/repo.git",
		"ssh://git@github.com/org/repo.git",
	}

	for _, q := range queries {
		t.Run(q, func(t *testing.T) {
			found, findErr := svc.FindPluginByGitHubRepoURL(ctx, q)
			require.NoError(t, findErr)
			assert.Equal(t, created.ID, found.ID)
		})
	}
}

func TestSyncService_EnqueueAutoSync_CreatesPendingAutoJob(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:sync_service_enqueue_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewSyncService(client, &syncTestFakeStorage{})
	ctx := context.Background()

	createdPlugin, err := client.Plugin.Create().
		SetName("enqueue-plugin").
		SetDisplayName("Enqueue Plugin").
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryOther).
		SetSourceType(plugin.SourceTypeGithub).
		SetStatus(plugin.StatusActive).
		SetGithubRepoURL("https://github.com/example/enqueue-plugin").
		Save(ctx)
	require.NoError(t, err)

	job, err := svc.EnqueueAutoSync(ctx, createdPlugin.ID.String(), "v1.2.3")
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Equal(t, createdPlugin.ID, job.PluginID)
	assert.Equal(t, syncjob.TriggerTypeAuto, job.TriggerType)
	assert.Equal(t, syncjob.StatusPending, job.Status)
	assert.Equal(t, "v1.2.3", job.TargetRef)
}

// syncTestCallbackStorage tracks upload/delete calls and supports an onUpload callback
// to inject side effects (e.g. creating a conflicting DB record mid-sync).
type syncTestCallbackStorage struct {
	uploadCount int
	deleteCount int
	deletedKeys []string
	onUpload    func(ctx context.Context, key string)
}

func (s *syncTestCallbackStorage) Upload(ctx context.Context, key string, _ io.Reader) (string, error) {
	s.uploadCount++
	if s.onUpload != nil {
		s.onUpload(ctx, key)
	}
	return "https://example.com/uploaded", nil
}
func (s *syncTestCallbackStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBufferString("wasm-binary:" + key)), nil
}
func (s *syncTestCallbackStorage) GetPresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return fmt.Sprintf("https://example.com/download%s", key), nil
}
func (s *syncTestCallbackStorage) Delete(_ context.Context, key string) error {
	s.deleteCount++
	s.deletedKeys = append(s.deletedKeys, key)
	return nil
}
func (s *syncTestCallbackStorage) Exists(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func TestSyncService_ConcurrentLock_BlocksDuplicate(t *testing.T) {
	svc := &SyncService{}

	unlock1, err := svc.acquireSyncLock("plugin-a", "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, unlock1)

	// Same key must be rejected while lock is held
	_, err = svc.acquireSyncLock("plugin-a", "v1.0.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "concurrent sync in progress")

	// Different key is independent — must succeed
	unlock2, err := svc.acquireSyncLock("plugin-b", "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, unlock2)
	unlock2()

	// After releasing the first lock, same key succeeds again
	unlock1()
	unlock3, err := svc.acquireSyncLock("plugin-a", "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, unlock3)
	unlock3()
}

func TestSyncService_OrphanCleanup_DeleteCalledOnCreateFailure(t *testing.T) {
	wasmBytes := []byte{0x00, 0x61, 0x73, 0x6d}
	releaseJSON := func(baseURL string) []byte {
		assetURL := baseURL + "/assets/plugin.wasm"
		b, _ := json.Marshal(map[string]any{
			"tag_name": "v1.0.0",
			"assets": []map[string]any{
				{"name": "plugin.wasm", "browser_download_url": assetURL},
			},
		})
		return b
	}

	mux := http.NewServeMux()
	var baseURL string
	mux.HandleFunc("/repos/example/orphan-cleanup-test/releases/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(releaseJSON(baseURL))
	})
	mux.HandleFunc("/assets/plugin.wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Write(wasmBytes)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	baseURL = server.URL

	client := enttest.Open(t, "sqlite3", "file:sync_service_orphan_cleanup_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	pluginRecord, err := client.Plugin.Create().
		SetName("orphan-cleanup-test").
		SetDisplayName("Orphan Cleanup Test").
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryOther).
		SetSourceType(plugin.SourceTypeGithub).
		SetStatus(plugin.StatusActive).
		SetGithubRepoURL("https://github.com/example/orphan-cleanup-test").
		Save(ctx)
	require.NoError(t, err)

	// The onUpload callback inserts a conflicting version AFTER versionAvailable
	// passes but BEFORE PluginVersion.Create, triggering a unique constraint
	// violation and exercising the orphan WASM cleanup path.
	trackingStorage := &syncTestCallbackStorage{
		onUpload: func(cbCtx context.Context, _ string) {
			_, _ = client.PluginVersion.Create().
				SetPluginID(pluginRecord.ID).
				SetVersion("v1.0.0").
				SetStatus(pluginversion.StatusDraft).
				SetWasmURL("conflict").
				SetWasmHash("sha256-conflict").
				SetSignature("").
				SetFileSize(1).
				SetMinAPIVersion("1.0.0").
				SetPluginAPIVersion("1.0.0").
				Save(cbCtx)
		},
	}

	svc := NewSyncService(client, trackingStorage)
	svc.githubAPIBase = baseURL
	svc.httpClient = server.Client()

	err = svc.runGitHubSync(ctx, pluginRecord.ID, "v1.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create plugin version")
	assert.Equal(t, 1, trackingStorage.uploadCount, "upload should have been called")
	assert.Equal(t, 1, trackingStorage.deleteCount, "delete should have been called to cleanup orphan wasm")
	assert.Contains(t, trackingStorage.deletedKeys[0], "orphan-cleanup-test")
}

func TestSyncService_ProcessSyncJobWithRetry_RetryDelayFallbackAndMaxAttemptsFallback(t *testing.T) {
	// Mock GitHub API server: release by tag and .wasm asset download
	wasmBytes := []byte{0x00, 0x61, 0x73, 0x6d} // minimal WASM magic
	releaseJSON := func(baseURL string) []byte {
		assetURL := baseURL + "/assets/plugin.wasm"
		b, _ := json.Marshal(map[string]any{
			"tag_name": "v1.0.0",
			"assets": []map[string]any{
				{"name": "plugin.wasm", "browser_download_url": assetURL},
			},
		})
		return b
	}

	mux := http.NewServeMux()
	var baseURL string
	mux.HandleFunc("/repos/example/retry-fallback-plugin/releases/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(releaseJSON(baseURL))
	})
	mux.HandleFunc("/assets/plugin.wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Write(wasmBytes)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	baseURL = server.URL

	client := enttest.Open(t, "sqlite3", "file:sync_service_retry_fallback_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewSyncService(client, &syncTestFakeStorage{})
	svc.githubAPIBase = baseURL
	svc.httpClient = server.Client()

	pluginRecord, err := client.Plugin.Create().
		SetName("retry-fallback-plugin").
		SetDisplayName("Retry Fallback Plugin").
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryOther).
		SetSourceType(plugin.SourceTypeGithub).
		SetStatus(plugin.StatusActive).
		SetGithubRepoURL("https://github.com/example/retry-fallback-plugin").
		Save(context.Background())
	require.NoError(t, err)

	job, err := client.SyncJob.Create().
		SetPluginID(pluginRecord.ID).
		SetTriggerType(syncjob.TriggerTypeAuto).
		SetStatus(syncjob.StatusPending).
		SetTargetRef("v1.0.0").
		Save(context.Background())
	require.NoError(t, err)

	started := time.Now()
	svc.ProcessSyncJobWithRetry(context.Background(), job.ID.String(), 0, -1*time.Second)
	elapsed := time.Since(started)

	updated, err := client.SyncJob.Get(context.Background(), job.ID)
	require.NoError(t, err)
	assert.Equal(t, syncjob.StatusSucceeded, updated.Status)
	assert.Less(t, elapsed, 1500*time.Millisecond)
}

func TestSyncService_SignAndPublish_WithKey(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:sync_sign_with_key_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	svc := NewSyncService(client, &syncTestFakeStorage{})
	svc.signingKeyID = "test-key-001"
	svc.signingKey = privKey

	ctx := context.Background()
	p, err := client.Plugin.Create().
		SetName("sign-test").
		SetDisplayName("Sign Test").
		SetCategory(plugin.CategoryOther).
		SetDescription("test").
		SetAuthor("test").
		SetSourceType(plugin.SourceTypeGithub).
		SetGithubRepoURL("https://github.com/test/sign-test").
		Save(ctx)
	require.NoError(t, err)

	pv, err := client.PluginVersion.Create().
		SetPluginID(p.ID).
		SetVersion("v1.0.0").
		SetWasmURL("plugins/sign-test/v1.0.0/plugin.wasm").
		SetWasmHash("sha256-abc123hash").
		SetFileSize(1024).
		SetMinAPIVersion("1.0.0").
		SetPluginAPIVersion("1.0.0").
		SetStatus(pluginversion.StatusDraft).
		Save(ctx)
	require.NoError(t, err)

	err = svc.signAndPublish(ctx, pv.ID, "sha256-abc123hash")
	require.NoError(t, err)

	got, err := client.PluginVersion.Get(ctx, pv.ID)
	require.NoError(t, err)
	assert.Equal(t, pluginversion.StatusPublished, got.Status)
	assert.NotEmpty(t, got.Signature)
	assert.Equal(t, "test-key-001", got.SignKeyID)
	assert.False(t, got.PublishedAt.IsZero())

	pubKey := privKey.Public().(ed25519.PublicKey)
	sigBytes, err := hex.DecodeString(got.Signature)
	require.NoError(t, err)
	assert.True(t, ed25519.Verify(pubKey, []byte("sha256-abc123hash"), sigBytes))
}

func TestSyncService_SignAndPublish_NoKey(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:sync_sign_no_key_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewSyncService(client, &syncTestFakeStorage{})

	ctx := context.Background()
	p, err := client.Plugin.Create().
		SetName("nosign-test").
		SetDisplayName("NoSign Test").
		SetCategory(plugin.CategoryOther).
		SetDescription("test").
		SetAuthor("test").
		SetSourceType(plugin.SourceTypeGithub).
		SetGithubRepoURL("https://github.com/test/nosign-test").
		Save(ctx)
	require.NoError(t, err)

	pv, err := client.PluginVersion.Create().
		SetPluginID(p.ID).
		SetVersion("v1.0.0").
		SetWasmURL("test.wasm").
		SetWasmHash("sha256-hash").
		SetFileSize(512).
		SetMinAPIVersion("1.0.0").
		SetPluginAPIVersion("1.0.0").
		SetStatus(pluginversion.StatusDraft).
		Save(ctx)
	require.NoError(t, err)

	err = svc.signAndPublish(ctx, pv.ID, "sha256-hash")
	require.NoError(t, err)

	got, err := client.PluginVersion.Get(ctx, pv.ID)
	require.NoError(t, err)
	assert.Equal(t, pluginversion.StatusDraft, got.Status)
	assert.Empty(t, got.Signature)
	assert.Empty(t, got.SignKeyID)
}

func TestSyncService_DoGitHubRequest_RateLimit(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 2 {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(1*time.Second).Unix()))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	svc := &SyncService{httpClient: server.Client()}
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test", nil)
	require.NoError(t, err)

	resp, err := svc.doGitHubRequest(ctx, req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.GreaterOrEqual(t, callCount, 3, "should have retried at least twice before succeeding")
}

func TestSyncService_DoGitHubRequest_AllRetriesFail(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(1*time.Second).Unix()))
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	svc := &SyncService{httpClient: server.Client()}
	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/test", nil)
	require.NoError(t, err)

	resp, err := svc.doGitHubRequest(ctx, req)
	// Last 429 response is returned when all retries are exhausted
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	assert.Equal(t, 4, callCount, "1 initial + 3 retries")
}

func TestSyncService_CanSign(t *testing.T) {
	svc := &SyncService{}
	assert.False(t, svc.CanSign())

	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	svc.signingKey = privKey
	assert.True(t, svc.CanSign())
}
