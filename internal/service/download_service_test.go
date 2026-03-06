package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/pluginversion"
	"github.com/IanShaw027/sub2api-plugin-market/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

type fakeStorage struct{}

func (s *fakeStorage) Upload(_ context.Context, _ string, _ io.Reader) (string, error) {
	return "https://example.com/uploaded", nil
}
func (s *fakeStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBufferString("wasm-binary:"+key)), nil
}
func (s *fakeStorage) GetPresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return fmt.Sprintf("https://example.com/download%s", key), nil
}
func (s *fakeStorage) Delete(_ context.Context, _ string) error { return nil }
func (s *fakeStorage) Exists(_ context.Context, _ string) (bool, error) { return true, nil }

type fakeVerifier struct{ verifyErr error }

func (v *fakeVerifier) VerifyPlugin(_ context.Context, _ *ent.PluginVersion, _ io.Reader) error {
	return v.verifyErr
}

func setupDownloadTest(t *testing.T, verifyErr error) (*DownloadService, *ent.Client) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	pluginRepo := repository.NewPluginRepository(client)
	store := &fakeStorage{}
	verifier := &fakeVerifier{verifyErr: verifyErr}
	svc := NewDownloadService(pluginRepo, store, client, verifier)
	return svc, client
}

func createPluginWithPublishedVersion(t *testing.T, client *ent.Client, name, version, wasmURL string) (*ent.Plugin, *ent.PluginVersion) {
	ctx := context.Background()
	p, err := client.Plugin.Create().
		SetName(name).
		SetDisplayName("Test Plugin").
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryOther).
		SetSourceType(plugin.SourceTypeUpload).
		SetStatus(plugin.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	pv, err := client.PluginVersion.Create().
		SetPluginID(p.ID).
		SetVersion(version).
		SetWasmURL(wasmURL).
		SetWasmHash("abc123").
		SetFileSize(100).
		SetMinAPIVersion("1.0.0").
		SetPluginAPIVersion("1.0.0").
		SetStatus(pluginversion.StatusPublished).
		SetPublishedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	return p, pv
}

func TestDownloadService_DownloadPlugin_Success(t *testing.T) {
	svc, client := setupDownloadTest(t, nil)
	defer client.Close()
	ctx := context.Background()

	wasmURL := "plugins/test-dl/v1.0.0/plugin.wasm"
	_, _ = createPluginWithPublishedVersion(t, client, "test-dl", "v1.0.0", wasmURL)

	pv, body, err := svc.DownloadPlugin(ctx, "test-dl", "v1.0.0", "192.168.1.1", "test-agent")
	require.NoError(t, err)
	require.NotNil(t, pv)
	require.NotNil(t, body)
	defer body.Close()

	assert.Equal(t, "v1.0.0", pv.Version)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "wasm-binary:"+wasmURL, string(data))

	logs, err := client.DownloadLog.Query().All(ctx)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.True(t, logs[0].Success)
}

func TestDownloadService_DownloadPlugin_VersionNotFound(t *testing.T) {
	svc, client := setupDownloadTest(t, nil)
	defer client.Close()
	ctx := context.Background()

	_, _ = createPluginWithPublishedVersion(t, client, "exists-plugin", "v1.0.0", "plugins/exists/v1.0.0/plugin.wasm")

	_, _, err := svc.DownloadPlugin(ctx, "exists-plugin", "v99.0.0", "", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPluginVersionNotFound)
}

func TestDownloadService_DownloadPlugin_VerificationFailed(t *testing.T) {
	svc, client := setupDownloadTest(t, assert.AnError)
	defer client.Close()
	ctx := context.Background()

	wasmURL := "plugins/verify-fail/v1.0.0/plugin.wasm"
	_, _ = createPluginWithPublishedVersion(t, client, "verify-fail", "v1.0.0", wasmURL)

	_, _, err := svc.DownloadPlugin(ctx, "verify-fail", "v1.0.0", "10.0.0.1", "ua")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPluginVerificationFailed)

	logs, err := client.DownloadLog.Query().All(ctx)
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.False(t, logs[0].Success)
	assert.NotEmpty(t, logs[0].ErrorMessage)
}

func TestDownloadService_GetDownloadURL_Success(t *testing.T) {
	svc, client := setupDownloadTest(t, nil)
	defer client.Close()

	wasmURL := "plugins/url-test/v1.0.0/plugin.wasm"
	_, _ = createPluginWithPublishedVersion(t, client, "url-test", "v1.0.0", wasmURL)

	url, err := svc.GetDownloadURL(context.Background(), "url-test", "v1.0.0", "", "")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/download"+wasmURL, url)
}

func TestDownloadService_GetDownloadURL_VersionNotFound(t *testing.T) {
	svc, client := setupDownloadTest(t, nil)
	defer client.Close()

	_, _ = createPluginWithPublishedVersion(t, client, "url-plugin", "v1.0.0", "plugins/url/v1.0.0/plugin.wasm")

	_, err := svc.GetDownloadURL(context.Background(), "url-plugin", "v2.0.0", "", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPluginVersionNotFound)
}
