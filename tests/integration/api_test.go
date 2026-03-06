package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent/downloadlog"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListPlugins_Empty 测试空插件列表
func TestListPlugins_Empty(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	w := tc.PerformRequest("GET", "/api/v1/plugins")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(0), response["code"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["total"])
	assert.Empty(t, data["plugins"])
}

// TestListPlugins_WithData 测试有数据的插件列表
func TestListPlugins_WithData(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	// 创建测试数据
	tc.CreateTestPlugin(t, "plugin-1", "analytics", true)
	tc.CreateTestPlugin(t, "plugin-2", "security", false)
	tc.CreateTestPlugin(t, "plugin-3", "analytics", true)

	w := tc.PerformRequest("GET", "/api/v1/plugins")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(3), data["total"])
	plugins := data["plugins"].([]interface{})
	assert.Len(t, plugins, 3)
}

// TestListPlugins_WithCategory 测试按分类过滤
func TestListPlugins_WithCategory(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	tc.CreateTestPlugin(t, "plugin-1", "analytics", true)
	tc.CreateTestPlugin(t, "plugin-2", "security", false)
	tc.CreateTestPlugin(t, "plugin-3", "analytics", true)

	w := tc.PerformRequest("GET", "/api/v1/plugins?category=analytics")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
}

// TestListPlugins_WithSearch 测试搜索功能
func TestListPlugins_WithSearch(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	tc.CreateTestPlugin(t, "analytics-assistant", "analytics", true)
	tc.CreateTestPlugin(t, "security-scanner", "security", false)
	tc.CreateTestPlugin(t, "analytics-translator", "analytics", true)

	w := tc.PerformRequest("GET", "/api/v1/plugins?search=assistant")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["total"])
}

// TestListPlugins_Pagination 测试分页
func TestListPlugins_Pagination(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	// 创建 5 个插件
	for i := 1; i <= 5; i++ {
		tc.CreateTestPlugin(t, "plugin-"+string(rune('0'+i)), "analytics", true)
	}

	// 第一页，每页 2 条
	w := tc.PerformRequest("GET", "/api/v1/plugins?page=1&page_size=2")
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(5), data["total"])
	assert.Equal(t, float64(1), data["page"])
	assert.Equal(t, float64(2), data["page_size"])
	plugins := data["plugins"].([]interface{})
	assert.Len(t, plugins, 2)
}

// TestListPlugins_WithType 测试按插件类型过滤
func TestListPlugins_WithType(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	p1, err := tc.Client.Plugin.Create().
		SetName("provider-plugin").
		SetDisplayName("Provider Plugin").
		SetCategory("proxy").
		SetDescription("A provider").
		SetAuthor("test").
		SetPluginType("provider").
		SetDownloadCount(0).
		Save(context.Background())
	require.NoError(t, err)
	_ = p1

	p2, err := tc.Client.Plugin.Create().
		SetName("transform-plugin").
		SetDisplayName("Transform Plugin").
		SetCategory("other").
		SetDescription("A transform").
		SetAuthor("test").
		SetPluginType("transform").
		SetDownloadCount(0).
		Save(context.Background())
	require.NoError(t, err)
	_ = p2

	w := tc.PerformRequest("GET", "/api/v1/plugins?type=provider")
	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["total"])
}

// TestGetPluginDetail_Success 测试获取插件详情成功
func TestGetPluginDetail_Success(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	plugin := tc.CreateTestPlugin(t, "test-plugin", "analytics", true)

	w := tc.PerformRequest("GET", "/api/v1/plugins/test-plugin")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(0), response["code"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "test-plugin", data["name"])
	assert.Equal(t, plugin.DisplayName, data["display_name"])
}

// TestGetPluginDetail_NotFound 测试插件不存在
func TestGetPluginDetail_NotFound(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	w := tc.PerformRequest("GET", "/api/v1/plugins/non-existent")

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 检查错误码非 0
	assert.NotEqual(t, float64(0), response["code"])
}

// TestGetPluginDetail_Suspended 测试下架插件不可见
func TestGetPluginDetail_Suspended(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	_, err := tc.Client.Plugin.Create().
		SetName("suspended-plugin").
		SetDisplayName("Suspended Plugin").
		SetCategory(plugin.CategoryAnalytics).
		SetDescription("Suspended plugin").
		SetAuthor("test-author").
		SetStatus(plugin.StatusSuspended).
		Save(context.Background())
	require.NoError(t, err)

	w := tc.PerformRequest("GET", "/api/v1/plugins/suspended-plugin")
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotEqual(t, float64(0), response["code"])
}

// TestGetPluginVersions_Success 测试获取插件版本列表成功
func TestGetPluginVersions_Success(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	plugin := tc.CreateTestPlugin(t, "test-plugin", "analytics", true)
	tc.CreateTestPluginVersion(t, plugin.ID, "1.0.0")
	tc.CreateTestPluginVersion(t, plugin.ID, "1.1.0")
	tc.CreateTestPluginVersion(t, plugin.ID, "2.0.0")

	w := tc.PerformRequest("GET", "/api/v1/plugins/test-plugin/versions")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(0), response["code"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "test-plugin", data["plugin_name"])
	assert.Equal(t, float64(3), data["total"])
	versions := data["versions"].([]interface{})
	assert.Len(t, versions, 3)
}

// TestGetPluginVersions_Empty 测试插件无版本
func TestGetPluginVersions_Empty(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	tc.CreateTestPlugin(t, "test-plugin", "analytics", true)

	w := tc.PerformRequest("GET", "/api/v1/plugins/test-plugin/versions")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["total"])
}

// TestGetPluginVersions_NotFound 测试插件不存在
func TestGetPluginVersions_NotFound(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	w := tc.PerformRequest("GET", "/api/v1/plugins/non-existent/versions")

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 检查错误码非 0
	assert.NotEqual(t, float64(0), response["code"])
}

// TestGetPluginVersions_Suspended 测试下架插件版本不可见
func TestGetPluginVersions_Suspended(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	pluginEntity, err := tc.Client.Plugin.Create().
		SetName("suspended-plugin").
		SetDisplayName("Suspended Plugin").
		SetCategory(plugin.CategoryAnalytics).
		SetDescription("Suspended plugin").
		SetAuthor("test-author").
		SetStatus(plugin.StatusSuspended).
		Save(context.Background())
	require.NoError(t, err)

	tc.CreateTestPluginVersion(t, pluginEntity.ID, "1.0.0")

	w := tc.PerformRequest("GET", "/api/v1/plugins/suspended-plugin/versions")
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotEqual(t, float64(0), response["code"])
}

// TestDownloadPlugin_Redirect 测试下载接口返回 302 重定向
func TestDownloadPlugin_Redirect(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	pluginEntity := tc.CreateTestPlugin(t, "download-plugin", "analytics", true)
	tc.CreateTestPluginVersion(t, pluginEntity.ID, "1.0.0")

	w := tc.PerformRequest("GET", "/api/v1/plugins/download-plugin/versions/1.0.0/download")

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com/download/test/path/1.0.0", w.Header().Get("Location"))

	updatedPlugin, err := tc.Client.Plugin.Get(context.Background(), pluginEntity.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, updatedPlugin.DownloadCount)

	logCount, err := tc.Client.DownloadLog.Query().
		Where(
			downloadlog.PluginIDEQ(pluginEntity.ID),
			downloadlog.VersionEQ("1.0.0"),
			downloadlog.SuccessEQ(true),
		).
		Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, logCount)
}

// TestDownloadPlugin_NotFound 测试下载不存在插件版本返回 404
func TestDownloadPlugin_NotFound(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	w := tc.PerformRequest("GET", "/api/v1/plugins/non-existent/versions/1.0.0/download")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, w.Header().Get("Location"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, float64(1002), response["code"])
}

// TestListTrustKeys_Empty 测试空信任密钥列表
func TestListTrustKeys_Empty(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	w := tc.PerformRequest("GET", "/api/v1/trust-keys")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(0), response["code"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["total"])
}

// TestListTrustKeys_WithData 测试有数据的信任密钥列表
func TestListTrustKeys_WithData(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	tc.CreateTestTrustKey(t, "key-1", "official", true)
	tc.CreateTestTrustKey(t, "key-2", "official", false)
	tc.CreateTestTrustKey(t, "key-3", "community", true)

	w := tc.PerformRequest("GET", "/api/v1/trust-keys")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(3), data["total"])
	keys := data["trust_keys"].([]interface{})
	assert.Len(t, keys, 3)
}

// TestListTrustKeys_FilterByType 测试按类型过滤
func TestListTrustKeys_FilterByType(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	tc.CreateTestTrustKey(t, "key-1", "official", true)
	tc.CreateTestTrustKey(t, "key-2", "official", false)
	tc.CreateTestTrustKey(t, "key-3", "community", true)

	w := tc.PerformRequest("GET", "/api/v1/trust-keys?key_type=official")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
}

// TestListTrustKeys_FilterByActive 测试按激活状态过滤
func TestListTrustKeys_FilterByActive(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	tc.CreateTestTrustKey(t, "key-1", "official", true)
	tc.CreateTestTrustKey(t, "key-2", "official", false)
	tc.CreateTestTrustKey(t, "key-3", "community", true)

	w := tc.PerformRequest("GET", "/api/v1/trust-keys?is_active=true")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
}

// TestGetTrustKeyDetail_Success 测试获取信任密钥详情成功
func TestGetTrustKeyDetail_Success(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	key := tc.CreateTestTrustKey(t, "test-key", "official", true)

	w := tc.PerformRequest("GET", "/api/v1/trust-keys/test-key")

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(0), response["code"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "test-key", data["key_id"])
	// JSON 序列化后 KeyType 变成 string
	assert.Equal(t, string(key.KeyType), data["key_type"])
}

// TestGetTrustKeyDetail_NotFound 测试信任密钥不存在
func TestGetTrustKeyDetail_NotFound(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	w := tc.PerformRequest("GET", "/api/v1/trust-keys/non-existent")

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 检查错误码非 0
	assert.NotEqual(t, float64(0), response["code"])
}

// TestGetPluginVersions_CompatibleWith 测试 compatible_with 兼容性过滤
func TestGetPluginVersions_CompatibleWith(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	p := tc.CreateTestPlugin(t, "compat-plugin", "proxy", false)

	// v1.0.0 requires API 0.9.0
	_, err := tc.Client.PluginVersion.Create().
		SetPluginID(p.ID).
		SetVersion("1.0.0").
		SetMinAPIVersion("0.9.0").
		SetPluginAPIVersion("1.0.0").
		SetWasmURL("/test/1.0.0").
		SetFileSize(1024).
		SetWasmHash("hash-1").
		SetSignature("sig-1").
		SetStatus("published").
		SetPublishedAt(time.Now()).
		Save(context.Background())
	require.NoError(t, err)

	// v2.0.0 requires API 2.0.0
	_, err = tc.Client.PluginVersion.Create().
		SetPluginID(p.ID).
		SetVersion("2.0.0").
		SetMinAPIVersion("2.0.0").
		SetPluginAPIVersion("2.0.0").
		SetWasmURL("/test/2.0.0").
		SetFileSize(2048).
		SetWasmHash("hash-2").
		SetSignature("sig-2").
		SetStatus("published").
		SetPublishedAt(time.Now().Add(time.Hour)).
		Save(context.Background())
	require.NoError(t, err)

	// compatible_with=1.0.0 → only v1.0.0 (min=0.9.0 <= 1.0.0)
	w := tc.PerformRequest("GET", "/api/v1/plugins/compat-plugin/versions?compatible_with=1.0.0")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["total"])

	// No filter → both versions
	w2 := tc.PerformRequest("GET", "/api/v1/plugins/compat-plugin/versions")
	assert.Equal(t, http.StatusOK, w2.Code)
	var resp2 map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	data2 := resp2["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data2["total"])
}
