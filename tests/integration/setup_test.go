package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSetup 测试环境设置
func TestSetup(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	// 验证数据库连接
	require.NotNil(t, tc.Client)

	// 验证可以执行查询
	count, err := tc.Client.Plugin.Query().Count(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

// TestCreateTestData 测试创建测试数据
func TestCreateTestData(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	// 创建测试插件
	plugin := tc.CreateTestPlugin(t, "test-plugin", "analytics", true)
	require.NotNil(t, plugin)
	require.Equal(t, "test-plugin", plugin.Name)

	// 创建测试版本
	version := tc.CreateTestPluginVersion(t, plugin.ID, "1.0.0")
	require.NotNil(t, version)
	require.Equal(t, "1.0.0", version.Version)

	// 创建测试信任密钥
	key := tc.CreateTestTrustKey(t, "test-key-1", "official", true)
	require.NotNil(t, key)
	require.Equal(t, "test-key-1", key.KeyID)
}
