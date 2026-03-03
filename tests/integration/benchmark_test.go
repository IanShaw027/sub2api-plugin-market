package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// BenchmarkPluginList 插件列表查询性能测试
func BenchmarkPluginList(b *testing.B) {
	tc := SetupTestContext(&testing.T{})
	defer tc.Cleanup()

	// 创建测试数据
	for i := 0; i < 100; i++ {
		tc.CreateTestPlugin(&testing.T{}, fmt.Sprintf("plugin-%d", i), "analytics", i%2 == 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc.PerformRequest("GET", "/api/v1/plugins?page=1&page_size=20")
	}
}

// BenchmarkPluginDetail 插件详情查询性能测试
func BenchmarkPluginDetail(b *testing.B) {
	tc := SetupTestContext(&testing.T{})
	defer tc.Cleanup()

	plugin := tc.CreateTestPlugin(&testing.T{}, "test-plugin", "analytics", true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc.PerformRequest("GET", "/api/v1/plugins/test-plugin")
	}
	_ = plugin
}

// BenchmarkConcurrentPluginList 并发插件列表查询性能测试
func BenchmarkConcurrentPluginList(b *testing.B) {
	tc := SetupTestContext(&testing.T{})
	defer tc.Cleanup()

	// 创建测试数据
	for i := 0; i < 100; i++ {
		tc.CreateTestPlugin(&testing.T{}, fmt.Sprintf("plugin-%d", i), "analytics", i%2 == 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tc.PerformRequest("GET", "/api/v1/plugins?page=1&page_size=20")
		}
	})
}

// TestConcurrentDownload 并发下载性能测试
func TestConcurrentDownload(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	// 创建测试插件和版本
	plugin := tc.CreateTestPlugin(t, "test-plugin", "analytics", true)
	for i := 1; i <= 10; i++ {
		tc.CreateTestPluginVersion(t, plugin.ID, fmt.Sprintf("1.%d.0", i))
	}

	// 并发测试配置
	concurrency := 100
	requestsPerWorker := 10

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(concurrency)

	// 启动并发请求
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < requestsPerWorker; j++ {
				path := fmt.Sprintf("/api/v1/plugins/test-plugin/versions")
				w := tc.PerformRequest("GET", path)
				require.Equal(t, 200, w.Code)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalRequests := concurrency * requestsPerWorker
	t.Logf("并发测试完成:")
	t.Logf("  - 并发数: %d", concurrency)
	t.Logf("  - 总请求数: %d", totalRequests)
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 平均响应时间: %v", duration/time.Duration(totalRequests))
	t.Logf("  - QPS: %.2f", float64(totalRequests)/duration.Seconds())

	// 验收标准：100 并发下载 < 5s
	require.Less(t, duration, 5*time.Second, "并发测试应在 5 秒内完成")
}

// TestDatabaseConcurrency 数据库并发访问测试
func TestDatabaseConcurrency(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	concurrency := 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	start := time.Now()

	// 并发创建插件
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()
			plugin := tc.CreateTestPlugin(t, fmt.Sprintf("concurrent-plugin-%d", id), "analytics", true)
			require.NotNil(t, plugin)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// 验证所有插件都创建成功
	count, err := tc.Client.Plugin.Query().Count(context.Background())
	require.NoError(t, err)
	require.Equal(t, concurrency, count)

	t.Logf("数据库并发测试完成:")
	t.Logf("  - 并发数: %d", concurrency)
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 平均创建时间: %v", duration/time.Duration(concurrency))
}

// TestStressPluginList 压力测试 - 大量数据查询
func TestStressPluginList(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	tc := SetupTestContext(t)
	defer tc.Cleanup()

	// 创建大量测试数据
	dataSize := 1000
	t.Logf("创建 %d 个测试插件...", dataSize)
	for i := 0; i < dataSize; i++ {
		tc.CreateTestPlugin(t, fmt.Sprintf("stress-plugin-%d", i), "analytics", i%2 == 0)
	}

	// 测试不同分页大小的性能
	pageSizes := []int{10, 20, 50, 100}
	for _, pageSize := range pageSizes {
		start := time.Now()
		w := tc.PerformRequest("GET", fmt.Sprintf("/api/v1/plugins?page=1&page_size=%d", pageSize))
		duration := time.Since(start)

		require.Equal(t, 200, w.Code)
		t.Logf("分页大小 %d: 响应时间 %v", pageSize, duration)
	}
}
