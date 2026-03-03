package integration

import (
	"encoding/json"
	"net/http"
	"testing"

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

	// API 返回 200 但 code 字段非 0 表示错误
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 检查错误码非 0
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

	// API 返回 200 但 code 字段非 0 表示错误
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 检查错误码非 0
	assert.NotEqual(t, float64(0), response["code"])
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

	// API 返回 200 但 code 字段非 0 表示错误
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// 检查错误码非 0
	assert.NotEqual(t, float64(0), response["code"])
}
