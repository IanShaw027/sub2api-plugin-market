# 集成测试

本目录包含 sub2api-plugin-market 项目的集成测试和性能测试。

## 测试覆盖

### API 接口测试

#### 插件相关接口
- `GET /api/v1/plugins` - 插件列表
  - 空列表测试
  - 有数据列表测试
  - 分类过滤测试
  - 搜索功能测试
  - 分页测试
- `GET /api/v1/plugins/:name` - 插件详情
  - 成功获取测试
  - 不存在插件测试
- `GET /api/v1/plugins/:name/versions` - 插件版本列表
  - 成功获取测试
  - 空版本列表测试
  - 不存在插件测试

#### 信任密钥相关接口
- `GET /api/v1/trust-keys` - 信任密钥列表
  - 空列表测试
  - 有数据列表测试
  - 按类型过滤测试
  - 按激活状态过滤测试
- `GET /api/v1/trust-keys/:key_id` - 信任密钥详情
  - 成功获取测试
  - 不存在密钥测试

### 性能测试

#### Benchmark 测试
- `BenchmarkPluginList` - 插件列表查询性能
- `BenchmarkPluginDetail` - 插件详情查询性能
- `BenchmarkConcurrentPluginList` - 并发插件列表查询

#### 并发测试
- `TestConcurrentDownload` - 100 并发下载测试（验收标准：< 5s）
- `TestDatabaseConcurrency` - 数据库并发访问测试
- `TestStressPluginList` - 压力测试（1000+ 数据）

## 运行测试

### 运行所有集成测试

```bash
go test -v ./tests/integration/...
```

### 运行特定测试

```bash
# 运行插件列表测试
go test -v ./tests/integration/... -run TestListPlugins

# 运行信任密钥测试
go test -v ./tests/integration/... -run TestListTrustKeys

# 运行并发测试
go test -v ./tests/integration/... -run TestConcurrent
```

### 运行性能测试

```bash
# 运行所有 benchmark
go test -bench=. -benchmem ./tests/integration/...

# 运行特定 benchmark
go test -bench=BenchmarkPluginList -benchmem ./tests/integration/...
```

### 查看测试覆盖率

```bash
go test -v -cover ./tests/integration/...
```

### 生成完整测试报告

```bash
# 运行测试报告生成脚本
./scripts/generate-test-report.sh

# 查看报告
open test-reports/summary.md
open test-reports/coverage.html
```

## 测试架构

### 文件结构

- `helper.go` - 测试辅助函数和测试上下文
- `setup_test.go` - 测试环境设置和测试数据创建
- `api_test.go` - API 集成测试
- `benchmark_test.go` - 性能测试和并发测试

### 测试上下文 (TestContext)

`TestContext` 提供了完整的测试环境：

- SQLite 内存数据库
- Ent Client
- Repository 层
- Service 层
- Gin 路由

### 辅助函数

- `SetupTestContext(t)` - 创建测试上下文
- `tc.Cleanup()` - 清理测试环境
- `tc.PerformRequest(method, path)` - 执行 HTTP 请求
- `tc.CreateTestPlugin(...)` - 创建测试插件
- `tc.CreateTestPluginVersion(...)` - 创建测试版本
- `tc.CreateTestTrustKey(...)` - 创建测试密钥

## 测试数据

测试使用 SQLite 内存数据库，每个测试用例独立运行，互不影响。

### 插件测试数据

- 名称：自定义
- 分类：analytics, security, proxy, auth, other
- 官方标识：true/false

### 版本测试数据

- 版本号：语义化版本（如 1.0.0）
- 状态：published（已发布）
- 发布时间：当前时间

### 信任密钥测试数据

- 密钥 ID：自定义
- 密钥类型：official, verified_publisher, community
- 激活状态：true/false

## 性能指标

### 并发测试结果

- **并发数**: 100
- **总请求数**: 1000
- **总耗时**: ~68ms
- **平均响应时间**: ~68µs
- **QPS**: ~14,640

### Benchmark 结果

- **插件列表查询**: ~124µs/op
- **插件详情查询**: ~50µs/op
- **并发插件列表**: ~1.16ms/op

## CI/CD 集成

项目配置了 GitHub Actions CI/CD 流水线：

- 自动运行所有测试
- 生成覆盖率报告
- 运行性能测试
- 构建 Docker 镜像

配置文件：`.github/workflows/ci.yml`

## 注意事项

1. 测试使用 SQLite 内存数据库，不需要外部数据库
2. 每个测试用例独立运行，自动清理数据
3. 测试数据符合 schema 约束
4. API 响应格式遵循统一的 JSON 格式
5. 性能测试可能受系统负载影响

## 依赖

- `github.com/stretchr/testify` - 断言库
- `github.com/mattn/go-sqlite3` - SQLite 驱动
- `github.com/gin-gonic/gin` - HTTP 框架
