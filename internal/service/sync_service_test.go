package service

import (
	"context"
	"testing"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/syncjob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

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

	svc := NewSyncService(client)
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

	svc := NewSyncService(client)
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

	svc := NewSyncService(client)
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

func TestSyncService_ProcessSyncJobWithRetry_RetryDelayFallbackAndMaxAttemptsFallback(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:sync_service_retry_fallback_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewSyncService(client)
	ctx := context.Background()

	pluginRecord, err := client.Plugin.Create().
		SetName("retry-fallback-plugin").
		SetDisplayName("Retry Fallback Plugin").
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryOther).
		SetSourceType(plugin.SourceTypeGithub).
		SetStatus(plugin.StatusActive).
		SetGithubRepoURL("https://github.com/example/retry-fallback-plugin").
		Save(ctx)
	require.NoError(t, err)

	job, err := client.SyncJob.Create().
		SetPluginID(pluginRecord.ID).
		SetTriggerType(syncjob.TriggerTypeAuto).
		SetStatus(syncjob.StatusPending).
		SetTargetRef("v1.0.0").
		Save(ctx)
	require.NoError(t, err)

	started := time.Now()
	svc.ProcessSyncJobWithRetry(ctx, job.ID.String(), 0, -1*time.Second)
	elapsed := time.Since(started)

	updated, err := client.SyncJob.Get(ctx, job.ID)
	require.NoError(t, err)
	assert.Equal(t, syncjob.StatusSucceeded, updated.Status)
	assert.Less(t, elapsed, 1500*time.Millisecond)
}
