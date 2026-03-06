package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/trustkey"
	v1 "github.com/IanShaw027/sub2api-plugin-market/internal/api/v1"
	"github.com/IanShaw027/sub2api-plugin-market/internal/api/v1/handler"
	"github.com/IanShaw027/sub2api-plugin-market/internal/repository"
	"github.com/IanShaw027/sub2api-plugin-market/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// TestContext 测试上下文
type TestContext struct {
	Client          *ent.Client
	Router          *gin.Engine
	PluginRepo      *repository.PluginRepository
	TrustKeyRepo    *repository.TrustKeyRepository
	PluginService   *service.PluginService
	DownloadService *service.DownloadService
	TrustKeyService *service.TrustKeyService
}

// SetupTestContext 设置测试上下文
func SetupTestContext(t *testing.T) *TestContext {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	pluginRepo := repository.NewPluginRepository(client)
	trustKeyRepo := repository.NewTrustKeyRepository(client)

	pluginService := service.NewPluginService(pluginRepo)
	downloadService := service.NewDownloadService(pluginRepo, &fakeStorage{}, client, &fakeVerifier{})
	trustKeyService := service.NewTrustKeyService(trustKeyRepo)

	pluginHandler := handler.NewPluginHandler(pluginService)
	downloadHandler := handler.NewDownloadHandler(downloadService)
	trustKeyHandler := handler.NewTrustKeyHandler(trustKeyService)
	submissionService := service.NewSubmissionService(client)
	syncService := service.NewSyncService(client, &fakeStorage{})
	submissionHandler := handler.NewSubmissionHandler(submissionService)
	githubWebhookHandler := handler.NewGitHubWebhookHandler(syncService, "", 1, 1)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1.RegisterRoutes(router, pluginHandler, downloadHandler, trustKeyHandler, submissionHandler, githubWebhookHandler)

	return &TestContext{
		Client:          client,
		Router:          router,
		PluginRepo:      pluginRepo,
		TrustKeyRepo:    trustKeyRepo,
		PluginService:   pluginService,
		DownloadService: downloadService,
		TrustKeyService: trustKeyService,
	}
}

// Cleanup 清理测试上下文
func (tc *TestContext) Cleanup() {
	tc.Client.Close()
}

// PerformRequest 执行 HTTP 请求
func (tc *TestContext) PerformRequest(method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	tc.Router.ServeHTTP(w, req)
	return w
}

// PerformRequestWithBody 执行带请求体的 HTTP 请求
func (tc *TestContext) PerformRequestWithBody(method, path string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	tc.Router.ServeHTTP(w, req)
	return w
}

// CreateTestPlugin 创建测试插件
func (tc *TestContext) CreateTestPlugin(t *testing.T, name, category string, isOfficial bool) *ent.Plugin {
	p, err := tc.Client.Plugin.Create().
		SetName(name).
		SetDisplayName(name + " Display").
		SetCategory(plugin.Category(category)).
		SetDescription("Test plugin: " + name).
		SetAuthor("test-author").
		SetIsOfficial(isOfficial).
		SetDownloadCount(0).
		Save(context.Background())
	require.NoError(t, err)
	return p
}

// CreateTestPluginVersion 创建测试插件版本
func (tc *TestContext) CreateTestPluginVersion(t *testing.T, pluginID uuid.UUID, version string) *ent.PluginVersion {
	pv, err := tc.Client.PluginVersion.Create().
		SetPluginID(pluginID).
		SetVersion(version).
		SetChangelog("Release notes for " + version).
		SetMinAPIVersion("1.0.0").
		SetPluginAPIVersion("1.0.0").
		SetWasmURL("/test/path/" + version).
		SetFileSize(1024).
		SetWasmHash("test-checksum-" + version).
		SetSignature("test-signature-" + version).
		SetStatus("published").
		SetPublishedAt(time.Now()).
		Save(context.Background())
	require.NoError(t, err)
	return pv
}

// CreateTestTrustKey 创建测试信任密钥
func (tc *TestContext) CreateTestTrustKey(t *testing.T, keyID, keyType string, isActive bool) *ent.TrustKey {
	key, err := tc.Client.TrustKey.Create().
		SetKeyID(keyID).
		SetKeyType(trustkey.KeyType(keyType)).
		SetPublicKey("test-public-key-" + keyID).
		SetOwnerName("test-owner").
		SetOwnerEmail("test@example.com").
		SetDescription("Test trust key: " + keyID).
		SetIsActive(isActive).
		SetCreatedAt(time.Now()).
		Save(context.Background())
	require.NoError(t, err)
	return key
}

type fakeStorage struct{}

func (s *fakeStorage) Upload(_ context.Context, _ string, _ io.Reader) (string, error) {
	return "https://example.com/uploaded", nil
}

func (s *fakeStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	content := fmt.Sprintf("wasm-binary:%s", key)
	return io.NopCloser(bytes.NewBufferString(content)), nil
}

func (s *fakeStorage) GetPresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return fmt.Sprintf("https://example.com/download%s", key), nil
}

func (s *fakeStorage) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *fakeStorage) Exists(_ context.Context, _ string) (bool, error) {
	return true, nil
}

type fakeVerifier struct{}

func (v *fakeVerifier) VerifyPlugin(_ context.Context, _ *ent.PluginVersion, _ io.Reader) error {
	return nil
}
