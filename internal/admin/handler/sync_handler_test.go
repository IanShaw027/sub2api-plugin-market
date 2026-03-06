package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/syncjob"
	"github.com/IanShaw027/sub2api-plugin-market/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// syncHandlerFakeStorage implements storage.Storage for sync handler tests.
type syncHandlerFakeStorage struct{}

func (s *syncHandlerFakeStorage) Upload(_ context.Context, _ string, _ io.Reader) (string, error) {
	return "https://example.com/uploaded", nil
}
func (s *syncHandlerFakeStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBufferString("wasm-binary:"+key)), nil
}
func (s *syncHandlerFakeStorage) GetPresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return fmt.Sprintf("https://example.com/download%s", key), nil
}
func (s *syncHandlerFakeStorage) Delete(_ context.Context, _ string) error { return nil }
func (s *syncHandlerFakeStorage) Exists(_ context.Context, _ string) (bool, error) { return true, nil }

func setupSyncHandlerTest(t *testing.T) (*SyncHandler, *ent.Client) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:sync_handler_test?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		_ = client.Close()
	})
	return NewSyncHandler(service.NewSyncService(client, &syncHandlerFakeStorage{})), client
}

func createTestPlugin(t *testing.T, client *ent.Client) *ent.Plugin {
	t.Helper()
	p, err := client.Plugin.Create().
		SetName("sync-list-plugin").
		SetDisplayName("Sync List Plugin").
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryOther).
		SetSourceType(plugin.SourceTypeGithub).
		SetStatus(plugin.StatusActive).
		SetGithubRepoURL("https://github.com/example/sync-list-plugin").
		Save(t.Context())
	require.NoError(t, err)
	return p
}

func TestListSyncJobs_DefaultPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, client := setupSyncHandlerTest(t)

	createdPlugin := createTestPlugin(t, client)

	for i := 0; i < 3; i++ {
		_, err := client.SyncJob.Create().
			SetPluginID(createdPlugin.ID).
			SetTriggerType(syncjob.TriggerTypeManual).
			SetStatus(syncjob.StatusPending).
			SetTargetRef("main").
			SetCreatedAt(time.Now().Add(time.Duration(i) * time.Second)).
			Save(t.Context())
		require.NoError(t, err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/api/sync-jobs", nil)

	h.ListSyncJobs(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "success", resp["message"])

	data := resp["data"].(map[string]any)
	jobs := data["jobs"].([]any)
	assert.Len(t, jobs, 3)

	pagination := data["pagination"].(map[string]any)
	assert.Equal(t, float64(1), pagination["page"])
	assert.Equal(t, float64(20), pagination["page_size"])
	assert.Equal(t, float64(3), pagination["total"])
	assert.Equal(t, float64(1), pagination["total_pages"])
}

func TestListSyncJobs_StatusFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, client := setupSyncHandlerTest(t)

	createdPlugin := createTestPlugin(t, client)

	_, err := client.SyncJob.Create().
		SetPluginID(createdPlugin.ID).
		SetTriggerType(syncjob.TriggerTypeManual).
		SetStatus(syncjob.StatusPending).
		SetTargetRef("main").
		Save(t.Context())
	require.NoError(t, err)

	_, err = client.SyncJob.Create().
		SetPluginID(createdPlugin.ID).
		SetTriggerType(syncjob.TriggerTypeAuto).
		SetStatus(syncjob.StatusSucceeded).
		SetTargetRef("v1.0.0").
		Save(t.Context())
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/api/sync-jobs?status=succeeded", nil)

	h.ListSyncJobs(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	data := resp["data"].(map[string]any)
	jobs := data["jobs"].([]any)
	require.Len(t, jobs, 1)

	job := jobs[0].(map[string]any)
	assert.Equal(t, "succeeded", job["status"])
}

func TestListSyncJobs_InvalidTimeParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _ := setupSyncHandlerTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/api/sync-jobs?from=not-rfc3339", nil)

	h.ListSyncJobs(c)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(ErrCodeInvalidParam), resp["code"])
	assert.Equal(t, "from 参数必须为 RFC3339 时间", resp["message"])
}
