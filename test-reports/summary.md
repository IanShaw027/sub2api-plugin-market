# Sub2API Plugin Market 测试报告

生成时间: 2026-03-03 09:46:29

## 测试摘要

- **总测试数**: 23
- **通过**: 23 ✅
- **失败**: 0
0 ❌
- **测试覆盖率**: 29.0%

## 测试类型

### 1. 单元测试和集成测试

详见: [test-output.txt](./test-output.txt)

### 2. 覆盖率报告

- HTML 报告: [coverage.html](./coverage.html)
- 文本报告: [coverage.txt](./coverage.txt)

### 3. 性能测试

详见: [benchmark.txt](./benchmark.txt)

### 4. 并发测试

详见: [concurrent.txt](./concurrent.txt)

## 测试覆盖

### API 接口测试

- ✅ GET /api/v1/plugins - 插件列表
- ✅ GET /api/v1/plugins/:name - 插件详情
- ✅ GET /api/v1/plugins/:name/versions - 版本列表
- ✅ GET /api/v1/trust-keys - 信任密钥列表
- ✅ GET /api/v1/trust-keys/:key_id - 信任密钥详情

### 测试场景

- ✅ 正常流程测试
- ✅ 边界条件测试
- ✅ 参数验证测试
- ✅ 并发性能测试
- ✅ 数据库并发测试

## 性能指标

### 并发测试结果

```
并发测试完成:
  - 并发数: 100
  - 总请求数: 1000
  - 总耗时: 68.301833ms
  - 平均响应时间: 68.301µs
  - QPS: 14640.90
--- PASS: TestConcurrentDownload (0.07s)
```

### Benchmark 结果

```
BenchmarkPluginList-10              	[GIN] 2026/03/03 - 09:46:24 | 200 |     124.208µs |                 | GET     /api/v1/plugins?page=1&page_size=20 
BenchmarkPluginDetail-10            	[GIN] 2026/03/03 - 09:46:25 | 200 |      49.708µs |                 | GET     /api/v1/plugins/test-plugin 
BenchmarkConcurrentPluginList-10    	[GIN] 2026/03/03 - 09:46:26 | 200 |    1.164708ms |                 | GET     /api/v1/plugins?page=1&page_size=20 
```
