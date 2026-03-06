package service

import (
	"context"
	"testing"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/pluginversion"
	"github.com/IanShaw027/sub2api-plugin-market/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func newPluginService(t *testing.T) (*PluginService, func()) {
	client := enttest.Open(t, "sqlite3", "file:plugin_service_test?mode=memory&cache=shared&_fk=1")
	repo := repository.NewPluginRepository(client)
	svc := NewPluginService(repo)
	ctx := context.Background()

	for _, name := range []string{"auth-guard", "rate-limiter", "proxy-cache"} {
		_, err := client.Plugin.Create().
			SetName(name).
			SetDisplayName(name + " Display").
			SetDescription("desc " + name).
			SetAuthor("tester").
			SetCategory(plugin.CategorySecurity).
			SetStatus(plugin.StatusActive).
			Save(ctx)
		require.NoError(t, err)
	}

	_, err := client.Plugin.Create().
		SetName("analytics-plugin").
		SetDisplayName("Analytics Plugin").
		SetDescription("analytics").
		SetAuthor("tester").
		SetCategory(plugin.CategoryAnalytics).
		SetIsOfficial(true).
		SetStatus(plugin.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	return svc, func() { client.Close() }
}

func TestPluginService_ListPlugins_DefaultPagination(t *testing.T) {
	svc, cleanup := newPluginService(t)
	defer cleanup()

	resp, err := svc.ListPlugins(context.Background(), &ListPluginsRequest{})
	require.NoError(t, err)
	assert.Equal(t, 4, resp.Total)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
	assert.Len(t, resp.Plugins, 4)
}

func TestPluginService_ListPlugins_FilterByCategory(t *testing.T) {
	svc, cleanup := newPluginService(t)
	defer cleanup()

	resp, err := svc.ListPlugins(context.Background(), &ListPluginsRequest{
		Category: "analytics",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "analytics-plugin", resp.Plugins[0].Name)
}

func TestPluginService_ListPlugins_FilterByOfficial(t *testing.T) {
	svc, cleanup := newPluginService(t)
	defer cleanup()

	isOfficial := true
	resp, err := svc.ListPlugins(context.Background(), &ListPluginsRequest{
		IsOfficial: &isOfficial,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	assert.True(t, resp.Plugins[0].IsOfficial)
}

func TestPluginService_ListPlugins_Search(t *testing.T) {
	t.Skip("search uses tagsContainFold with PostgreSQL-specific ::text ILIKE; skipped on SQLite")
}

func TestPluginService_ListPlugins_CacheHit(t *testing.T) {
	svc, cleanup := newPluginService(t)
	defer cleanup()

	req := &ListPluginsRequest{Category: "security"}
	resp1, err := svc.ListPlugins(context.Background(), req)
	require.NoError(t, err)

	resp2, err := svc.ListPlugins(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, resp1.Total, resp2.Total)
	assert.Equal(t, resp1, resp2)
}

func TestPluginService_InvalidateCache(t *testing.T) {
	svc, cleanup := newPluginService(t)
	defer cleanup()

	req := &ListPluginsRequest{}
	_, err := svc.ListPlugins(context.Background(), req)
	require.NoError(t, err)

	svc.InvalidateCache()

	key := svc.cacheKey(req)
	_, ok := svc.cache.Load(key)
	assert.False(t, ok, "cache should be empty after invalidation")
}

func TestPluginService_GetPluginDetail(t *testing.T) {
	svc, cleanup := newPluginService(t)
	defer cleanup()

	p, err := svc.GetPluginDetail(context.Background(), "auth-guard")
	require.NoError(t, err)
	assert.Equal(t, "auth-guard", p.Name)
}

func TestPluginService_GetPluginDetail_NotFound(t *testing.T) {
	svc, cleanup := newPluginService(t)
	defer cleanup()

	_, err := svc.GetPluginDetail(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestPluginService_GetPluginVersions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:plugin_svc_versions_test?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	p, err := client.Plugin.Create().
		SetName("versioned-plugin").
		SetDisplayName("Versioned").
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryOther).
		SetStatus(plugin.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	for _, v := range []string{"1.0.0", "1.1.0", "2.0.0"} {
		_, err := client.PluginVersion.Create().
			SetPluginID(p.ID).
			SetVersion(v).
			SetWasmURL("test.wasm").
			SetWasmHash("sha256-" + v).
			SetFileSize(100).
			SetMinAPIVersion("1.0.0").
			SetPluginAPIVersion("1.0.0").
			SetSignature("sig-" + v).
			SetStatus(pluginversion.StatusPublished).
			SetPublishedAt(time.Now()).
			Save(ctx)
		require.NoError(t, err)
	}

	repo := repository.NewPluginRepository(client)
	svc := NewPluginService(repo)

	versions, err := svc.GetPluginVersions(ctx, "versioned-plugin", "")
	require.NoError(t, err)
	assert.Len(t, versions, 3)
}
