package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/syncjob"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockGitHubWebhookSyncService struct {
	findPluginFn            func(ctx context.Context, repoURL string) (*ent.Plugin, error)
	isDuplicateAutoFn       func(ctx context.Context, pluginID, targetRef string) (bool, error)
	enqueueAutoSyncFn       func(ctx context.Context, pluginID, targetRef string) (*ent.SyncJob, error)
	processWithRetryFn      func(ctx context.Context, jobID string, maxAttempts int, retryDelay time.Duration)
	processWithRetryInvoked chan struct{}
}

func (m *mockGitHubWebhookSyncService) FindPluginByGitHubRepoURL(ctx context.Context, repoURL string) (*ent.Plugin, error) {
	if m.findPluginFn == nil {
		return nil, nil
	}
	return m.findPluginFn(ctx, repoURL)
}

func (m *mockGitHubWebhookSyncService) IsDuplicateAutoSync(ctx context.Context, pluginID, targetRef string) (bool, error) {
	if m.isDuplicateAutoFn == nil {
		return false, nil
	}
	return m.isDuplicateAutoFn(ctx, pluginID, targetRef)
}

func (m *mockGitHubWebhookSyncService) EnqueueAutoSync(ctx context.Context, pluginID, targetRef string) (*ent.SyncJob, error) {
	if m.enqueueAutoSyncFn == nil {
		return nil, nil
	}
	return m.enqueueAutoSyncFn(ctx, pluginID, targetRef)
}

func (m *mockGitHubWebhookSyncService) ProcessSyncJobWithRetry(ctx context.Context, jobID string, maxAttempts int, retryDelay time.Duration) {
	if m.processWithRetryFn != nil {
		m.processWithRetryFn(ctx, jobID, maxAttempts, retryDelay)
	}
	if m.processWithRetryInvoked != nil {
		select {
		case m.processWithRetryInvoked <- struct{}{}:
		default:
		}
	}
}

func TestGitHubWebhookHandler_IgnoredForNonReleaseEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockGitHubWebhookSyncService{}
	h := NewGitHubWebhookHandler(service, "", 3, 2)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/github/webhook", bytes.NewBufferString(`{"action":"published"}`))
	c.Request.Header.Set("X-GitHub-Event", "push")
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleGitHubWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "ignored", resp.Message)
}

func TestGitHubWebhookHandler_ReleasePublishedTriggersAutoSync(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pluginID := uuid.New()
	jobID := uuid.New()
	enqueueCalled := false
	processCalled := make(chan struct{}, 1)

	service := &mockGitHubWebhookSyncService{
		findPluginFn: func(ctx context.Context, repoURL string) (*ent.Plugin, error) {
			assert.Equal(t, "https://github.com/example/demo", repoURL)
			return &ent.Plugin{ID: pluginID}, nil
		},
		isDuplicateAutoFn: func(ctx context.Context, pluginIDArg, targetRef string) (bool, error) {
			assert.Equal(t, pluginID.String(), pluginIDArg)
			assert.Equal(t, "v1.2.3", targetRef)
			return false, nil
		},
		enqueueAutoSyncFn: func(ctx context.Context, pluginIDArg, targetRef string) (*ent.SyncJob, error) {
			enqueueCalled = true
			assert.Equal(t, pluginID.String(), pluginIDArg)
			assert.Equal(t, "v1.2.3", targetRef)
			return &ent.SyncJob{ID: jobID, PluginID: pluginID, TriggerType: syncjob.TriggerTypeAuto, Status: syncjob.StatusPending}, nil
		},
		processWithRetryFn: func(ctx context.Context, enqueuedJobID string, maxAttempts int, retryDelay time.Duration) {
			assert.Equal(t, jobID.String(), enqueuedJobID)
			assert.Equal(t, 3, maxAttempts)
			assert.Equal(t, 2*time.Second, retryDelay)
			processCalled <- struct{}{}
		},
	}
	h := NewGitHubWebhookHandler(service, "", 3, 2)

	body := `{
		"action":"published",
		"repository":{"html_url":"https://github.com/example/demo"},
		"release":{"tag_name":"v1.2.3"}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/github/webhook", bytes.NewBufferString(body))
	c.Request.Header.Set("X-GitHub-Event", "release")
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleGitHubWebhook(c)

	assert.True(t, enqueueCalled)
	select {
	case <-processCalled:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected ProcessSyncJobWithRetry to be invoked")
	}

	assert.Equal(t, http.StatusOK, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)

	data, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "pending", data["status"])
}

func TestGitHubWebhookHandler_InvalidSignatureWhenSecretEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &mockGitHubWebhookSyncService{}
	h := NewGitHubWebhookHandler(service, "my-secret", 3, 2)

	body := `{"action":"published","repository":{"html_url":"https://github.com/example/demo"},"release":{"tag_name":"v1.2.3"}}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/github/webhook", bytes.NewBufferString(body))
	c.Request.Header.Set("X-GitHub-Event", "release")
	c.Request.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleGitHubWebhook(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, ErrCodeInvalidParam, resp.Code)
}

func TestGitHubWebhookHandler_DuplicateEventIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pluginID := uuid.New()
	called := false

	service := &mockGitHubWebhookSyncService{
		findPluginFn: func(ctx context.Context, repoURL string) (*ent.Plugin, error) {
			assert.Equal(t, "https://github.com/example/demo", repoURL)
			return &ent.Plugin{ID: pluginID}, nil
		},
		isDuplicateAutoFn: func(ctx context.Context, pluginIDArg, targetRef string) (bool, error) {
			assert.Equal(t, pluginID.String(), pluginIDArg)
			assert.Equal(t, "v1.2.3", targetRef)
			return true, nil
		},
		enqueueAutoSyncFn: func(ctx context.Context, pluginIDArg, targetRef string) (*ent.SyncJob, error) {
			called = true
			return nil, nil
		},
	}
	h := NewGitHubWebhookHandler(service, "", 3, 2)

	body := `{
		"action":"published",
		"repository":{"html_url":"https://github.com/example/demo"},
		"release":{"tag_name":"v1.2.3"}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/github/webhook", bytes.NewBufferString(body))
	c.Request.Header.Set("X-GitHub-Event", "release")
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleGitHubWebhook(c)

	assert.False(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "ignored", resp.Message)
}

func TestGitHubWebhookHandler_PluginNotMatchedIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	service := &mockGitHubWebhookSyncService{
		findPluginFn: func(ctx context.Context, repoURL string) (*ent.Plugin, error) {
			assert.Equal(t, "https://github.com/example/missing", repoURL)
			return nil, &ent.NotFoundError{}
		},
		isDuplicateAutoFn: func(ctx context.Context, pluginIDArg, targetRef string) (bool, error) {
			called = true
			return false, nil
		},
		enqueueAutoSyncFn: func(ctx context.Context, pluginIDArg, targetRef string) (*ent.SyncJob, error) {
			called = true
			return nil, nil
		},
	}
	h := NewGitHubWebhookHandler(service, "", 3, 2)

	body := `{
		"action":"published",
		"repository":{"html_url":"https://github.com/example/missing"},
		"release":{"tag_name":"v9.9.9"}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/github/webhook", bytes.NewBufferString(body))
	c.Request.Header.Set("X-GitHub-Event", "release")
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleGitHubWebhook(c)

	assert.False(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "ignored", resp.Message)
}

func TestGitHubWebhookHandler_EmptySecretReleaseMode(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.TestMode)

	service := &mockGitHubWebhookSyncService{}
	h := NewGitHubWebhookHandler(service, "", 3, 2)

	body := `{"action":"published","repository":{"html_url":"https://github.com/example/demo"},"release":{"tag_name":"v1.0.0"}}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/github/webhook", bytes.NewBufferString(body))
	c.Request.Header.Set("X-GitHub-Event", "release")
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleGitHubWebhook(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, ErrCodeForbidden, resp.Code)
	assert.Contains(t, resp.Message, "生产环境拒绝处理")
}

func TestGitHubWebhookHandler_EmptySecretDebugMode(t *testing.T) {
	gin.SetMode(gin.DebugMode)
	defer gin.SetMode(gin.TestMode)

	pluginID := uuid.New()
	jobID := uuid.New()

	service := &mockGitHubWebhookSyncService{
		findPluginFn: func(ctx context.Context, repoURL string) (*ent.Plugin, error) {
			return &ent.Plugin{ID: pluginID}, nil
		},
		isDuplicateAutoFn: func(ctx context.Context, pluginIDArg, targetRef string) (bool, error) {
			return false, nil
		},
		enqueueAutoSyncFn: func(ctx context.Context, pluginIDArg, targetRef string) (*ent.SyncJob, error) {
			return &ent.SyncJob{ID: jobID, PluginID: pluginID, TriggerType: syncjob.TriggerTypeAuto, Status: syncjob.StatusPending}, nil
		},
		processWithRetryInvoked: make(chan struct{}, 1),
	}
	h := NewGitHubWebhookHandler(service, "", 3, 2)

	body := `{
		"action":"published",
		"repository":{"html_url":"https://github.com/example/demo"},
		"release":{"tag_name":"v1.0.0"}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/github/webhook", bytes.NewBufferString(body))
	c.Request.Header.Set("X-GitHub-Event", "release")
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleGitHubWebhook(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
}

func TestGitHubWebhookHandler_UsesInjectedRetryConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pluginID := uuid.New()
	jobID := uuid.New()
	processCalled := make(chan struct{}, 1)

	service := &mockGitHubWebhookSyncService{
		findPluginFn: func(ctx context.Context, repoURL string) (*ent.Plugin, error) {
			return &ent.Plugin{ID: pluginID}, nil
		},
		isDuplicateAutoFn: func(ctx context.Context, pluginIDArg, targetRef string) (bool, error) {
			return false, nil
		},
		enqueueAutoSyncFn: func(ctx context.Context, pluginIDArg, targetRef string) (*ent.SyncJob, error) {
			return &ent.SyncJob{ID: jobID, PluginID: pluginID, TriggerType: syncjob.TriggerTypeAuto, Status: syncjob.StatusPending}, nil
		},
		processWithRetryFn: func(ctx context.Context, enqueuedJobID string, maxAttempts int, retryDelay time.Duration) {
			assert.Equal(t, jobID.String(), enqueuedJobID)
			assert.Equal(t, 5, maxAttempts)
			assert.Equal(t, 1*time.Second, retryDelay)
			processCalled <- struct{}{}
		},
	}
	h := NewGitHubWebhookHandler(service, "", 5, 1)

	body := `{
		"action":"published",
		"repository":{"html_url":"https://github.com/example/demo"},
		"release":{"tag_name":"v9.9.9"}
	}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/github/webhook", bytes.NewBufferString(body))
	c.Request.Header.Set("X-GitHub-Event", "release")
	c.Request.Header.Set("Content-Type", "application/json")

	h.HandleGitHubWebhook(c)

	select {
	case <-processCalled:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected ProcessSyncJobWithRetry to be invoked with injected config")
	}

	assert.Equal(t, http.StatusOK, w.Code)
}
