# 插件系统实施清单

> **用途**: 逐项打勾的执行跟踪表，配合 [06-COMPLETE-IMPLEMENTATION-PLAN.md](./06-COMPLETE-IMPLEMENTATION-PLAN.md) 使用  
> **更新方式**: 完成一项勾一项，标注完成日期和负责人  
> **规则**: 每个 Phase 开始前，确认前置 Phase 全部完成；每个 Phase 结束时执行回归测试

---

## 进度总览

| Phase | 任务数 | 已完成 | 进度 | 状态 | 前置 |
|-------|-------|--------|------|------|------|
| Phase 0: 安全加固 | 11 | 9 | 82% | 收尾中 | 无 |
| Phase 1: 链路打通 | 18 | 18 | 100% | ✅ 完成 | Phase 0 |
| Phase 2: Provider 插件化 | 24 | 20 | 83% | 收尾中(灰度) | Phase 1 |
| Phase 3: Transform/Interceptor | 10 | 10 | 100% | ✅ 完成 | Phase 2.1 |
| Phase 4: 生态建设 | 17 | 13 | 76% | 进行中 | Phase 3 |
| 运维与上线准备 | 8 | 8 | 100% | ✅ 完成 | 随各 Phase 同步 |
| **合计** | **88** | **78** | **89%** | | |

---

## Phase 0: 安全加固

**目标**: 修复生产安全漏洞 + 数据完整性问题  
**预计**: 1 周（大部分已完成，仅剩收尾） | **仓库**: sub2api-plugin-market | **前置**: 无

### P0 安全修复 (生产阻断)

- [x] **0.1** POST /submissions 增加 IP 级 rate limit
  - 文件: `internal/api/v1/router.go`
  - 现状: ✅ 已实现 `middleware.NewIPRateLimiter`
  - 待验证: 确认限流阈值（当前值）是否合理
  - 完成: ☑  日期: 已完成  负责人: ____

- [x] **0.2** Webhook 签名强制校验
  - 文件: `internal/api/v1/handler/github_webhook_handler.go`
  - 现状: ⚠️ 仅在 `gin.ReleaseMode` 下拒绝空 secret
  - 遗留: 评估是否需要在所有模式下强制（当前行为对开发便利，生产安全）
  - 完成: ☑  日期: 已完成  负责人: ____

- [x] **0.3** 审核操作事务化
  - 文件: `internal/admin/service/submission_service.go`
  - 现状: ✅ 已用 `client.Tx()` 包裹
  - 完成: ☑  日期: 已完成  负责人: ____

### P1 数据完整性

- [x] **0.4** 插件名正则校验（Schema 层加固）
  - 文件: `ent/schema/plugin.go`
  - 改动: Schema 层增加 `Match(regexp.MustCompile(^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$))` + `make generate` 通过
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **0.5** Official 插件审核角色限制
  - 文件: `internal/admin/service/submission_service.go`
  - 现状: ✅ 已有 `if pluginRecord.IsOfficial && reviewerRole == "reviewer" { return ErrForbiddenReview }`
  - 完成: ☑  日期: 已完成  负责人: ____

- [ ] **0.6** Sync 并发锁（升级为分布式锁）
  - 文件: `internal/service/sync_service.go`
  - 现状: ⚠️ 已有 `sync.Map` 进程内锁，但多实例部署时无效
  - 改动: 替换为 Redis `SETNX` 或 PostgreSQL advisory lock（如单实例部署可保留现状并标注）
  - 验证: 两个 market 实例同时触发 Sync → 只有一个执行
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [x] **0.7** Sync 操作顺序 + 孤儿清理
  - 文件: `internal/service/sync_service.go`
  - 现状: ✅ 已实现 — `runGitHubSync` 第 493 行：版本创建失败时调用 `storage.Delete(ctx, storageKey)` 清理孤儿 WASM
  - 完成: ☑  日期: 已完成  负责人: ____

- [x] **0.8** 审核接口乐观锁
  - 文件: `internal/admin/service/submission_service.go`
  - 现状: ✅ 已有 `Where(submission.StatusEQ(submission.StatusPending))` 条件更新
  - 完成: ☑  日期: 已完成  负责人: ____

- [x] **0.9** 同一插件 pending Submission 数量限制
  - 文件: `internal/service/submission_service.go`
  - 现状: ✅ 已实现 — `CreateSubmission` 第 127-138 行：查询 `pendingCount >= 3` 时返回错误
  - 完成: ☑  日期: 已完成  负责人: ____

### P0 运行时安全 (V5 新增)

- [x] **0.10** pluginruntime map 并发保护
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/dispatch_runtime.go`
  - 现状: ✅ 已实现 — 第 73 行 `sync.RWMutex`，Register* 用 `Lock()`，snapshotSortedRegistrations/HasActivePlugins 用 `RLock()`
  - 完成: ☑  日期: 已完成  负责人: ____

- [ ] **0.11** GinStreamWriter.WriteChunk Flush 语义统一
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/writer.go`
  - 现状: `GinStreamWriter.WriteChunk()` 内部隐式调用 `Flush()`，与 DispatchRuntime 显式调用 `Flush()` 产生重复 Flush
  - 改动: `WriteChunk` 只写缓冲不自动 Flush，由调度层统一控制 Flush 时机
  - 验证: 流式输出正常 + 无多余 Flush 开销
  - 依赖: 2.1 (StreamWriter 扩展)
  - 完成: ☐  日期: ____  负责人: ____

### Phase 0 验收门禁

- [ ] 全部 11 项中：9 项已完成、1 项待完成（0.6 分布式锁评估）、1 项暂跳过（0.11 依赖 Phase 2）
- [ ] `make test` 通过（含新增测试用例）
- [ ] `make lint` 通过
- [ ] 安全测试覆盖: rate limit / webhook 强制 / 路径遍历 / 并发锁 / 乐观锁 / pending 数量限制
- [ ] **Phase 0 完成签字**: 日期: ____  签字: ____

---

## Phase 1: 链路打通

**目标**: Echo 插件跑通「上传→审核→发布→下载→安装→执行」全链路  
**预计**: 3-4 周 | **仓库**: market + sub2api | **前置**: Phase 0 ✅

### 1.1 Schema 扩展 (market)

- [x] **1.1** Plugin 加 plugin_type
  - 文件: `ent/schema/plugin.go`
  - 现状: ✅ 已有 `field.Enum("plugin_type").Values("interceptor","transform","provider").Optional()`
  - 完成: ☑  日期: 已完成  负责人: ____

- [x] **1.2** PluginVersion 加 capabilities + config_schema
  - 文件: `ent/schema/plugin_version.go`
  - 现状: ✅ 已有 `field.JSON("capabilities", []string{})` + `field.JSON("config_schema", map[string]interface{}{})`
  - 完成: ☑  日期: 已完成  负责人: ____

- [x] **1.3** Submission 加 plugin_version 关联
  - 文件: `ent/schema/submission.go`
  - 现状: ✅ 已有 `edge.To("version", PluginVersion.Type).Unique()`
  - 完成: ☑  日期: 已完成  负责人: ____

- [x] **1.4** make generate + 编译验证
  - 命令: `make generate && go build ./... && make test`
  - 结果: ✅ 全部通过（含 0.4 Schema Match 变更）
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### 1.2 WASM 上传 + 签名 + 发布 (market)

**路径 A: 手动上传**

- [x] **1.5** 提交 handler 支持 multipart 上传
  - 文件: `internal/api/v1/handler/submission_handler.go`
  - 改动: 根据 Content-Type 区分 JSON/multipart，multipart 解析 wasm_file/manifest/signature/sign_key_id + 元数据字段，有 wasm 时返回 201
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **1.6** submission_service 完整创建流程
  - 文件: `internal/service/submission_service.go` + `cmd/server/main.go`
  - 改动: 新增 PluginManifest 结构体 + storage 注入; WASM 流程: manifest 校验→SHA-256→trust_key 查公钥→Ed25519 验签→上传 storage→创建 Plugin(含 plugin_type)→创建 PluginVersion(draft)→创建 Submission(关联版本)
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **1.7** 审核 service 联动发布版本
  - 文件: `internal/admin/service/submission_service.go`
  - 现状: ✅ 已实现 — approve 时在事务内将关联的 draft PluginVersion 更新为 published + 设置 published_at
  - 完成: ☑  日期: 已完成  负责人: ____

**路径 B: GitHub Sync 自动发布**

- [x] **1.8** Sync 下载 manifest.json + signature.sig
  - 文件: `internal/service/sync_service.go`
  - 改动: runGitHubSync 步骤 2b 遍历 assets 查找 manifest.json 和 .sig 文件并下载，缺失时警告不报错（向后兼容）
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **1.9** Sync 解析 manifest + 验签 + 创建 published 版本
  - 文件: `internal/service/sync_service.go` + `internal/service/sync_service_test.go`
  - 改动: 新增 verifyExternalSignature (trust_key Ed25519)；有外部签名且验证通过→status=published；外部签名优先于内置签名；manifest 字段写入 PluginVersion；新增 2 个测试用例
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### 1.3 API 增强 (market)

- [x] **1.10** GET /plugins 支持 ?type= 筛选（handler + repository 实现）
  - 文件: `internal/api/v1/handler/plugin_handler.go` + `internal/repository/plugin_repository.go`
  - 现状: ✅ 已实现 — handler 第 28 行读取 `c.Query("type")`，repo 第 33-35 行做 `plugin.PluginTypeEQ(plugin.PluginType(pluginType))` 过滤
  - 完成: ☑  日期: 已完成  负责人: ____

- [x] **1.11** GET /versions 支持 ?compatible_with= 过滤（handler + repository 实现）
  - 文件: `internal/api/v1/handler/plugin_handler.go` + `internal/repository/plugin_repository.go`
  - 现状: ✅ 已实现 — handler 第 85 行读取 `c.Query("compatible_with")`，repo 第 102-104 行做 `pluginversion.MinAPIVersionLTE(compatibleWith)` 过滤
  - 完成: ☑  日期: 已完成  负责人: ____

- [x] **1.12** OpenAPI spec 同步更新
  - 文件: `openapi/plugin-market-v1.yaml`
  - 改动: 新增 multipart submission schema (CreateSubmissionMultipartRequest)、PluginManifest schema、Capability enum、PluginVersionDetail 补充 capabilities/config_schema/plugin_type; 409 PENDING_LIMIT_EXCEEDED 响应
  - 验证: `make check-contract` 通过
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **1.13** ERROR-CODE-REGISTRY 同步更新
  - 文件: `docs/ERROR-CODE-REGISTRY.md` + `internal/api/v1/handler/response.go`
  - 改动: 新增 6 个错误码 (1007-1012): MANIFEST_INVALID/WASM_HASH_MISMATCH/SIGNATURE_INVALID/SIGN_KEY_NOT_FOUND/WASM_UPLOAD_FAILED/PENDING_LIMIT_EXCEEDED + handler response.go 常量和 HTTP status 映射
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### 1.4 DispatchRuntime 接入 (sub2api 主项目)

- [x] **1.14** DispatchRuntime 接入 gateway_handler
  - 仓库: `sub2api`
  - 文件: `backend/internal/handler/gateway_handler.go` + `backend/internal/service/plugin_dispatch_service.go` + `backend/internal/pluginruntime/dispatch_facade.go`
  - 现状: ✅ 已实现 — `PluginDispatchService` 注入 GatewayHandler（L55/L73/L110），`TryDispatch` 在 Gemini 路径（L399-411）和 Claude 路径（L658-678）均已调用；fallback 到内置 Service 已就绪（`!pluginHandled` 分支）；`DispatchFacade` 封装 DispatchRuntime 并处理 `ErrNoPluginsActive`/`ErrPluginDispatchSkipped`
  - 完成: ☑  日期: 已完成  负责人: ____

- [x] **1.15** Interceptor next 链修复
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/dispatch_runtime.go`
  - 改动: 重构 Dispatch 为递归调用链；新增 `dispatchFromInterceptor` (递归处理 interceptor + next 闭包) 和 `dispatchPostInterceptor` (TransformRequest → Provider → TransformResponse 管线)；保持 failure policy/observability 语义不变
  - 测试: 新增 3 个测试（next 获真实响应 / next 修改响应 / nil 不调 next 继续管线），全部通过
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **1.16** 默认 Provider 注册（内置降级）
  - 仓库: `sub2api`
  - 文件: 新增 `backend/internal/pluginruntime/builtin_providers.go` + `builtin_providers_test.go`
  - 改动: `BuiltinProviderAdapter` 通用适配器 + 4 个工厂函数 (Claude/OpenAI/Gemini/Antigravity) + `RegisterBuiltinProviders` 注册函数 (Priority=9999); ForwardFunc 暂为 placeholder (返回 ErrRuntimeNoProvider)，待 Phase 2 接入真实 Service
  - 测试: 7 个测试全部通过（Metadata/自定义 Forward/Placeholder/注册/Dispatch fallback/外部优先/Builtin 兜底）
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### 1.5 部署准备 (market)

- [x] **1.17** 数据库迁移脚本
  - 文件: `migrations/000001_initial_schema.up.sql` + `down.sql` + `migrations/README.md` + `scripts/export_schema_sql.go`
  - 改动: 初始 schema 迁移 SQL（含所有表/索引/约束）+ down 回滚脚本 + Ent DDL 导出工具 + Makefile migrate-export 目标 + 迁移流程文档
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **1.18** 管理后台 UI 适配
  - 文件: `web/admin/js/app.js` + `internal/admin/service/submission_service.go`
  - 改动: 前端 — 插件列表页增加类型筛选器(All/Interceptor/Transform/Provider)、插件卡片展示 plugin_type badge（蓝/绿/紫）、提交卡片和详情展示 capabilities 标签、审核详情展示关联 PluginVersion 信息(版本号/状态/wasm_hash/capabilities); 后端 — ListSubmissions/GetSubmission 增加 WithVersion() eager-loading
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### Phase 1 端到端验收门禁

- [ ] **E2E-Upload**: Echo Provider 编译 WASM → 签名 → POST /submissions 上传 → 管理员 approve → PluginVersion published → sub2api GET /download → 验签 → 安装 → 请求走插件 → 返回 "echo"
- [ ] **E2E-Sync**: GitHub Release 含 plugin.wasm + manifest.json + signature.sig → Webhook 触发 → SyncJob → published → sub2api 可下载安装
- [ ] **E2E-Fallback**: 不安装任何插件 → 请求走内置 Service → 行为与 Phase 0 之前完全一致
- [ ] **回归**: Phase 0 安全测试全部重跑通过
- [ ] 两个仓库 `make test` + `make lint` 通过
- [ ] **Phase 1 完成签字**: 日期: ____  签字: ____

---

## Phase 2: Provider 插件化

**目标**: Host 流式编排 + 4 个 Provider WASM 插件 + 行为等价  
**预计**: 5-7 周 | **仓库**: sub2api | **前置**: Phase 1 ✅

### 2.1 Host 流式编排 (前置基础设施)

- [x] **2.1** StreamWriter 扩展
  - 文件: `pluginapi/types.go` + `pluginruntime/writer.go` + `pluginruntime/gateway_adapter.go`
  - 改动: `Done() <-chan struct{}` 新增（Flush/SetStatus/ClientGone 已存在）；GinStreamWriter.Done() 返回 request context.Done()；MemoryStreamWriter.Done() 返回内部 channel
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **2.2** ProviderPlugin 接口扩展
  - 文件: `pluginapi/types.go`
  - 改动: 新增 `StreamDelegate` 结构体 + `StreamProviderPlugin` 接口（PrepareStream/OnSSELine/OnStreamEnd），与已有 StreamingProviderPlugin 共存
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [ ] **2.3** ProviderContext 定义（详见 06 方案 §2.1.3）
  - 文件: `pluginapi/types.go`
  - 改动: 定义 ProviderContext 结构体，完整字段:
    - `AccountID int` — 选中账号 ID
    - `Token string` — 访问令牌
    - `BaseURL string` — 上游 API 基础 URL
    - `ProxyURL string` — 代理 URL（可空）
    - `OriginalModel string` — 用户请求原始模型名
    - `MappedModel string` — 经映射后的模型名
    - `Platform string` — 目标平台标识
    - `IsStream bool` — 是否流式请求
    - `MaxTokens int` — 最大 token 数限制
    - `OrganizationID string` — OpenAI 组织 ID（可空）
    - `ProjectID string` — OpenAI 项目 ID（可空）
    - `ExtraHeaders map[string]string` — 平台特有附加 Header
    - `Timeout int` — 请求超时秒数
  - 约定: 通过 `GatewayRequest.Metadata["provider_context"]` 传递
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [x] **2.4** ProviderResultMetadata 定义（详见 06 方案 §2.1.4）
  - 文件: `pluginapi/types.go` + `pluginapi/provider_context.go`
  - 改动: ProviderResultMetadata 已存在（含 UsageInfo 子结构/FirstTokenMs/Failover/ImageCount）；新增 `GetProviderResult`/`SetProviderResult` 辅助函数
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **2.5** Host 流式 HTTP
  - 文件: `pluginruntime/host_api_http.go` + `host_api_http_test.go`
  - 改动: FetchStreaming 增强 — context 绑定、HostHTTPError 错误类型、ctx.Done() 检查、bufio.Scanner 1MB 行限制；新增 3 个测试（HTTP 错误/context 取消/大行）
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **2.6** DispatchRuntime Provider 流式调度
  - 文件: `pluginruntime/dispatch_runtime.go` + `dispatch_stream_test.go`
  - 改动: 新增 `dispatchStreamProvider` 方法 — PrepareStream→Host FetchStreaming→OnSSELine 回调→WriteChunk→OnStreamEnd 获取 Metadata；Provider 阶段类型断言 StreamProviderPlugin 走流式路径
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **2.7** keepalive + interval timeout
  - 文件: `pluginruntime/dispatch_runtime.go`
  - 改动: 流式管道 select 循环监听 4 事件：upstream line/keepalive ticker(30s)/idle timeout(5min)/ctx.Done()；`DispatchRuntimeConfig` 新增 StreamKeepaliveInterval + StreamIdleTimeout
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **2.8** 核心注入 ProviderContext
  - 文件: `backend/internal/handler/gateway_handler.go`
  - 改动: Gemini 路径和 Claude 路径 TryDispatch 前调用 `buildProviderContext(account, model, platform, stream)` 构建完整 ProviderContext → metadata["provider_context"]；新增 `accountTokenAndType` 按账号类型提取 token
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **2.9** 核心消费 ProviderResultMetadata
  - 文件: `backend/internal/handler/gateway_handler.go`
  - 改动: pluginResult.Handled 后调用 `extractForwardResultFromPlugin` 从 ProviderResultMetadata 提取 ForwardResult 用于 RecordUsage 计费
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### 2.2 Provider 插件开发 + 灰度上线

每个 Provider 按 **开发→对比→shadow→canary→全量** 上线：

#### claude-provider

- [x] **2.10** claude-provider 开发
  - 文件: `backend/internal/plugins/claude/` (plugin.go + request.go + response.go + 测试，共 920 行)
  - 现状: ✅ 已实现 StreamingProviderPlugin — 请求构建(anthropic-version/anthropic-beta/auth)、SSE 解析(event 分发/message_start/content_block_delta/message_delta/message_stop)、Usage 提取、Failover 信号
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [x] **2.11** claude-provider 对比测试
  - 现状: ✅ `plugins/claude/plugin_test.go` + `request_test.go` + `response_test.go` 全部通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [ ] **2.12** claude-provider 灰度上线
  - 步骤: Shadow(1周) → Canary 10%(3天) → 50%(3天) → 100%
  - 完成: ☐  日期: ____  负责人: ____

#### openai-provider

- [x] **2.13** openai-provider 开发
  - 文件: `backend/internal/plugins/openai/` (plugin.go + request.go + response.go + 测试，共 1063 行)
  - 现状: ✅ 已实现 — 请求构建(Codex/Platform URL 区分/Organization/Project)、SSE 解析([DONE] 终止)、model 替换、Usage 提取
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [x] **2.14** openai-provider 对比测试
  - 现状: ✅ `plugins/openai/` 测试全部通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [ ] **2.15** openai-provider 灰度上线
  - 同 2.12 步骤
  - 完成: ☐  日期: ____  负责人: ____

#### antigravity-provider

- [x] **2.16** antigravity-provider 开发
  - 文件: `backend/internal/plugins/antigravity/` (plugin.go + request.go + response.go + 测试，共 1172 行)
  - 现状: ✅ 已实现 — 请求构建(URL 拼接)、SSE 解析、Gemini/Claude 类型映射、BaseURLs fallback
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [x] **2.17** antigravity-provider 对比测试
  - 现状: ✅ `plugins/antigravity/` 测试全部通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [ ] **2.18** antigravity-provider 灰度上线
  - 同 2.12 步骤
  - 完成: ☐  日期: ____  负责人: ____

#### gemini-provider

- [x] **2.19** gemini-provider 开发
  - 文件: `backend/internal/plugins/gemini/` (plugin.go + request.go + response.go + 测试，共 1126 行)
  - 现状: ✅ 已实现 — 请求构建(AI Studio/Code Assist URL 区分)、SSE 解析(JSON 数组)、usageMetadata 提取、ForwardNative 支持
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [x] **2.20** gemini-provider 对比测试
  - 现状: ✅ `plugins/gemini/` 测试全部通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [ ] **2.21** gemini-provider 灰度上线
  - 同 2.12 步骤
  - 完成: ☐  日期: ____  负责人: ____

### 2.3 上线基础设施

- [x] **2.22** 对比测试框架
  - 仓库: `sub2api`
  - 文件: 新增 `backend/internal/pluginruntime/consistency_test_framework.go` + `consistency_test_framework_test.go`
  - 改动: `ConsistencyTester` — 并发执行内置/插件双路转发，对比 StatusCode/Body(JSON deep equal 忽略动态字段)/Usage(容差比例)/Model；`CompareStream` 流式对比；`jsonDeepEqual` 递归比较。11 个测试全部通过
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **2.23** Shadow/Canary 流量切分机制
  - 仓库: `sub2api`
  - 文件: 新增 `backend/internal/pluginruntime/traffic_split.go` + `traffic_split_test.go`
  - 改动: `TrafficSplitter` — shadow/canary/full/disabled 四模式 + `PLUGIN_TRAFFIC_{NAME}=mode:pct` 环境变量加载 + `ShouldUsePlugin`/`ShouldReturnPluginResult`/`IsShadow` + sync.RWMutex 并发安全。10 个测试全部通过
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **2.24** WASM body 大小限制
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/dispatch_runtime.go` + `dispatch_runtime_test.go`
  - 改动: `MaxRequestBodyBytes` 配置(默认 2MB)；`checkBodySize` 检查；Interceptor/Transform 超限→跳过+日志；Provider 超限→返回 ErrRuntimeNoProvider fallback。6 个测试全部通过
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### Phase 2 验收门禁

- [ ] 4 个 Provider 均可作为 WASM 插件安装运行
- [ ] 无插件时自动降级到内置（1.16 默认 Provider 生效）
- [ ] 对比测试全部通过（body + Usage + Model + Failover + 错误）
- [ ] 性能: 非流式延迟增加 <10ms；流式首 token <15ms；后续 chunk <2ms
- [ ] 4 个 Provider 全部通过 100% 灰度
- [ ] **回归**: Phase 0 安全测试 + Phase 1 E2E 测试全部重跑通过
- [ ] **Phase 2 完成签字**: 日期: ____  签字: ____

---

## Phase 3: Transform + Interceptor 插件化

**目标**: 7 个 Transform/Interceptor 插件可运行，原有功能等价  
**预计**: 3-4 周 | **仓库**: sub2api | **前置**: Phase 2.1 (2.1~2.9) ✅

### 3.1 前置基础设施

- [x] **3.1** Config Host API
  - 文件: `pluginruntime/host_api_config.go` + `host_api_config_test.go` + `capability.go` 新增 `CapabilityHostConfigRead`
  - 改动: `HostAPIConfig` — Get/GetAll 按 pluginID namespace 隔离 + capability 检查; 5 个测试通过
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **3.2** TransformPlugin 增加 ChunkTransformer 可选接口
  - 文件: `pluginapi/types.go`
  - 改动: 新增 `ChunkTransformer` 接口 `TransformChunk(chunk []byte) ([]byte, error)`
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **3.3** Host 流式管道链式调用
  - 文件: `pluginruntime/stream_pipeline.go` + `stream_pipeline_test.go`
  - 现状: ✅ 已实现 — StreamPipeline 支持 TransformLines/ClientDisconnect/StatusAndHeaders；4 个测试全部通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

### 3.2 Transform 插件 (4 个)

- [x] **3.4** antigravity-transform
  - 文件: `backend/internal/plugins/antigravitytransform/transform.go` (519 行)
  - 现状: ✅ 已实现 TransformPlugin — Claude↔Gemini 双向转换，编译通过
  - 待补: 单元测试（目前只有代码无 _test.go）
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [x] **3.5** claude-gemini-transform
  - 文件: `backend/internal/plugins/claudegemini/` (plugin.go + transform.go + transform_test.go)
  - 现状: ✅ 已实现 TransformRequest + TransformResponse，测试通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [x] **3.6** codex-tool-corrector
  - 文件: `backend/internal/plugins/codextool/` (plugin.go + plugin_test.go)
  - 现状: ✅ 已实现 TransformResponse + 工具名映射，测试通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [x] **3.7** error-mapper
  - 文件: `backend/internal/plugins/errormapper/` (plugin.go + plugin_test.go)
  - 现状: ✅ 已实现 TransformResponse 统一错误格式，测试通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

### 3.3 Interceptor 插件 (3 个)

- [x] **3.8** model-mapper
  - 文件: `backend/internal/plugins/modelmapper/` (plugin.go + plugin_test.go)
  - 现状: ✅ 已实现 InterceptorPlugin — model 字段映射，测试通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [x] **3.9** claude-code-validator
  - 文件: `backend/internal/plugins/claudecodevalidator/` (plugin.go + plugin_test.go)
  - 现状: ✅ 已实现 InterceptorPlugin — UA/SystemPrompt/Headers 校验，不通过短路 403，测试通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [x] **3.10** gemini-signature-cleaner
  - 文件: `backend/internal/plugins/geminisigcleaner/` (plugin.go + plugin_test.go)
  - 现状: ✅ 已实现 InterceptorPlugin — thoughtSignature 清理/替换 dummy 签名，测试通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

### Phase 3 验收门禁

- [ ] 组合: gemini-provider + claude-gemini-transform → Claude 客户端透明使用 Gemini
- [ ] 组合: antigravity-provider + antigravity-transform → 正确
- [ ] 组合: openai-provider + codex-tool-corrector → 流式 tool_calls 被矫正
- [ ] 组合: model-mapper + claude-provider → 模型别名正确映射
- [ ] 组合: claude-code-validator + claude-provider → 非 Code 客户端被拦截
- [ ] 卸载任一插件 → fallback 到内置实现，服务不中断
- [ ] **回归**: Phase 0 + 1 + 2 全部测试重跑通过
- [ ] **Phase 3 完成签字**: 日期: ____  签字: ____

---

## Phase 4: 生态建设

**目标**: 社区开发者 1 小时内开发并发布一个简单插件  
**预计**: 4+ 周 | **前置**: Phase 3 ✅

### 4.1 开发者工具

- [ ] **4.1** sub2api-plugin-sdk
  - 仓库: 新建 `sub2api-plugin-sdk`
  - 交付: Go SDK，封装 pluginapi 接口 + Host API 客户端；提供 `sdk.NewProvider()`, `sdk.NewTransform()`, `sdk.NewInterceptor()` 便捷构造器
  - 验证: 用 SDK 重写 claude-code-validator，代码量减少 50%+
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.2** sub2api-plugin-cli
  - 仓库: 新建 `sub2api-plugin-cli`
  - 交付: CLI 子命令 `init`(生成脚手架) / `build`(TinyGo 编译) / `sign`(Ed25519) / `test`(本地测试) / `publish`(上传市场)
  - 验证: `sub2api-plugin-cli init --type provider --name test && sub2api-plugin-cli build && sub2api-plugin-cli sign && sub2api-plugin-cli publish` 全流程通过
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.3** 插件模板 x3
  - 仓库: `sub2api-plugin-cli` 内嵌
  - 交付: provider-template / transform-template / interceptor-template
  - 验证: 每个模板 `init → build` 可直接编译出合法 WASM
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.4** 本地测试框架
  - 仓库: `sub2api-plugin-sdk`
  - 交付: `testing.MockDispatchRuntime` + `testing.MockHostAPI`，支持 `go test`
  - 验证: 用框架测试 echo-provider → 无需启动任何服务
  - 完成: ☐  日期: ____  负责人: ____

### 4.2 市场增强

- [x] **4.5** 审核时依赖解析校验
  - 文件: `internal/admin/service/submission_service.go`
  - 改动: 新增 `validateDependencies` — approve 前遍历 PluginVersion dependencies 检查 Plugin 是否存在且 active
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **4.6** 列表 API 内存缓存
  - 文件: 新增 `internal/repository/cache.go` + 修改 `plugin_repository.go`
  - 改动: TTLCache (3min TTL) + ListPlugins 缓存 + IncrementDownloadCount 清缓存
  - 完成: ☐  日期: ____  负责人: ____

- [x] **4.7** 预签名 URL 缓存
  - 文件: `internal/service/download_service.go`
  - 改动: presignCache (5min TTL)，相同 (wasm_url, 5min窗口) 复用预签名 URL
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **4.8** Trust Key 多代并存轮换
  - 文件: `internal/service/verification_service.go`
  - 改动: loadTrustKeys 改用 ListActiveTrustKeys (仅 is_active=true)，deprecated 但 active 的 key 仍可验签
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **4.9** 插件搜索增强
  - 文件: `internal/repository/plugin_repository.go`
  - 改动: Contains→ContainsFold (ILIKE) + 新增 tags JSON 搜索谓词
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### 4.3 运行时增强

- [x] **4.10** 热重载实现
  - 文件: `pluginruntime/hot_reload_coordinator.go` + `hot_reload_coordinator_test.go`
  - 现状: ✅ 已实现 — HotReloadCoordinator 支持 Load/Reload/Unload/DrainTimeout/ContextCancellation; 6 个测试通过
  - 完成: ☑  日期: 已完成  负责人: sub2api 团队

- [x] **4.11** Prometheus metrics 导出
  - 文件: 新增 `pluginruntime/metrics_exporter.go` + `metrics_exporter_test.go`
  - 改动: MetricsExporter 接口 + InMemoryMetricsExporter (thread-safe) + NopMetricsExporter + expvar 暴露; 6 个测试通过
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **4.12** 错误率熔断
  - 文件: `pluginruntime/circuit_breaker.go`
  - 改动: 新增 ErrorRateCircuitBreaker (ringBuffer 滑动窗口 + 阈值触发 + 成功重置); 7 个测试通过
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **4.13** Log Host API 限速
  - 文件: `pluginruntime/host_api_log.go`
  - 改动: 新增 logRateLimiter (100条/秒/插件默认 + ErrLogRateLimited + DroppedLogs 计数); 5 个测试通过
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### 4.4 文档

- [x] **4.14** Plugin Developer Guide
  - 文件: `docs/PLUGIN-DEVELOPER-GUIDE.md`
  - 内容: 快速开始 + 三种插件类型 + 开发流程 + manifest 格式 + Host API 概览 + FAQ
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **4.15** Plugin API Reference
  - 文件: `docs/PLUGIN-API-REFERENCE.md`
  - 内容: pluginapi 全部接口详细文档 (13 个类型/接口)
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **4.16** Host API Reference
  - 文件: `docs/HOST-API-REFERENCE.md`
  - 内容: HTTP/KV/Log/Config 四类 Host API + Capability 枚举 + 错误类型
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **4.17** Best Practices
  - 文件: `docs/PLUGIN-BEST-PRACTICES.md`
  - 内容: WASM 内存/Body 限制/错误处理/流式注意事项/性能调优/安全脱敏/测试策略
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### Phase 4 验收门禁

- [ ] 社区开发者 1 小时内完成 init → build → sign → publish
- [ ] CLI 全流程可用
- [ ] 至少 3 个社区插件在市场运行
- [ ] **回归**: Phase 0 + 1 + 2 + 3 全部测试通过
- [ ] **Phase 4 完成签字**: 日期: ____  签字: ____

---

## 运维与上线准备

> 以下是跨 Phase 的生产上线必要条件，做完开发任务但不做这些 = 无法上线。

### A. 环境与配置

- [x] **OPS.1** 生产环境变量清单
  - 交付: `docs/ENV-VARIABLES.md` + `.env.example`
  - 涵盖: DB(6项)/服务(4项)/安全(2项)/签名(2项)/GitHub(1项) 共 15 个变量
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **OPS.2** 数据库迁移预演
  - 文件: `docs/DB-MIGRATION-RUNBOOK.md`
  - 交付: 迁移预演手册 — staging 流程/执行命令/验证清单(旧数据+新字段+启动+测试)/耗时记录模板/回滚步骤
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### B. 部署流程

- [x] **OPS.3** 部署顺序与回滚预案
  - 交付: `docs/DEPLOYMENT-GUIDE.md`
  - 涵盖: Phase 分步部署顺序 + 灰度策略 (Shadow→Canary→Full) + 快速/完整回滚预案 + Health Check
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **OPS.4** CI/CD Pipeline 更新
  - 文件: `docs/CI-CD-GUIDE.md` + `.github/workflows/ci.yml`
  - 交付: CI 指南 (Market 4阶段 + Sub2API WASM/签名) + GitHub Actions workflow (PostgreSQL service + contract/test/lint/build)
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### C. 监控告警

- [x] **OPS.5** 监控大盘 + 告警规则
  - 文件: `docs/MONITORING-GUIDE.md`
  - 交付: Market 指标 + Sub2API 插件指标 + 灰度期指标 + P0-P3 告警规则 + Grafana Dashboard JSON 模板
  - 完成: ☑  日期: 2026-03-06  负责人: AI

- [x] **OPS.6** Health Check 端点增强
  - 文件: `cmd/server/main.go`
  - 改动: `/health` 增加 DB 连通性检查（`client.Plugin.Query().Limit(1)`），失败返回 503 + `"status":"degraded"`
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### D. 风险缓解

- [x] **OPS.7** GitHub API 限流保护
  - 文件: `internal/service/sync_service.go`
  - 现状: ✅ 已实现 — `doGitHubRequest` 已有 maxRetries=3 + exponential backoff(1s/2s/4s) + 429/403 rate limit header 解析 + wait until reset
  - 完成: ☑  日期: 已完成  负责人: ____

- [x] **OPS.8** Semver 兼容性匹配规则文档
  - 交付: `docs/SEMVER-COMPATIBILITY.md`
  - 涵盖: 匹配算法 + 实现代码 + 边界用例表 + 后续改进建议
  - 完成: ☑  日期: 2026-03-06  负责人: AI

### 运维准备验收

- [ ] 全部 OPS.1-OPS.8 已完成
- [ ] staging 环境完整走通一次部署流程
- [ ] 监控大盘可用，告警规则已配置并测试
- [ ] **运维准备完成签字**: 日期: ____  签字: ____

---

## 关键里程碑

| # | 里程碑 | 标志 | 目标周 | 实际日期 | 签字 |
|---|--------|------|-------|---------|------|
| M0 | 安全就绪 | Phase 0 全部通过 + OPS.1-2 完成 | Week 2 | ____ | ____ |
| M1 | 第一个插件全链路 | Echo Provider E2E 跑通 + staging 部署验证 | Week 6 | ____ | ____ |
| M2 | 流式编排就绪 | Host 流式 HTTP + OnSSELine + 对比测试框架 | Week 9 | ____ | ____ |
| M3 | 首个 Provider 灰度 | claude-provider 100% + 监控大盘就绪 | Week 11 | ____ | ____ |
| M4 | 全部 Provider 就绪 | 4 个 Provider 100% | Week 13 | ____ | ____ |
| M5 | 12 个插件全部可用 | Phase 3 完成 | Week 17 | ____ | ____ |
| M6 | 生态就绪 | SDK + CLI + 文档 | Week 18+ | ____ | ____ |
| M7 | 运维就绪 | OPS.1-8 全部完成 | M3 之前 | ____ | ____ |

---

## 阻断项跟踪

| # | 发现日期 | 描述 | 影响 Phase | 严重度 | 状态 | 解决方案 | 解决日期 |
|---|---------|------|-----------|--------|------|---------|---------|
| | | | | | | | |

---

## V4 源码交叉校验报告

> 以下是对照 `sub2api` (`backend/internal/`) 和 `sub2api-plugin-market` 实际源码后发现的关键事实，部分直接影响清单任务的可行性。

### 发现 1: Interceptor next 链为 stub（影响 1.15）

`dispatch_runtime.go` 中 Interceptor 的 `next` 参数永远返回 `(nil, nil)`：

```go
next := func(context.Context, *pluginapi.GatewayRequest) (*pluginapi.GatewayResponse, error) {
    return nil, nil
}
```

这意味着 Interceptor 无法链式调用下游（Transform → Provider）。任务 1.15（next 链实现）是**必须做的**，否则 Interceptor 无法拦截并修改下游响应。

### 发现 2: phase.go 定义 7 阶段 vs dispatch_runtime.go 仅实现 4 阶段

- `phase.go` 定义: `pre_auth`, `post_auth`, `intercept`, `transform`, `proxy`, `post_proxy`, `log`
- `dispatch_runtime.go` 实际执行: interceptor → transform(request) → provider → transform(response)

**差距**: `pre_auth`/`post_auth`/`post_proxy`/`log` 阶段在 dispatch 中未被使用。需明确：是移除 phase.go 中多余的阶段定义，还是在 dispatch 中实现完整 7 阶段。

### 发现 3: gateway_handler 完全未调用 Dispatch()

`gateway_handler.go` 通过 `if/else` 判断 `account.Platform` 直接调用各 Gateway Service 的 `ForwardWithInput`，完全绕过了 `DispatchRuntime.Dispatch()`。Dispatch 仅在测试和 examples 中使用。

**意义**: 任务 1.14（gateway_handler 接入 DispatchRuntime）是最关键的架构变更——需要将现有的 platform switch 逻辑迁移为通过 Dispatch() 调度。

### 发现 4: OpenAI 路由隐藏在 Claude 路径中

`gateway_handler.go` 的 else 分支（非 Gemini、非 Antigravity）统一走 `gatewayService.ForwardWithInput`，这同时处理 Claude 和 OpenAI。但 `openai_gateway_service.go` 存在且有独立的 `Forward` 方法。

**意义**: 插件化时需要明确 OpenAI 是独立 Provider 还是共享 Claude Provider 路径。清单任务 2.16-2.18（openai-provider）可能需要额外的路由拆分工作。

### 发现 5: Usage 记录在 handler 层的 goroutine 中

`gateway_handler.go` 中 `RecordUsage` 在 `go func()` 中异步执行。ProviderPlugin 的结果需要返回足够的 Usage 信息（token count 等）才能让核心记录用量。

**意义**: 任务 2.8（ProviderResultMetadata）需包含 `InputTokens`、`OutputTokens`、`Model` 等字段，使 handler 能在 `RecordUsage` 中使用。

### 发现 6: StreamWriter 没有 Flush/SetStatus/Done

当前 `StreamWriter` 接口仅有 `State()`/`SetHeader()`/`WriteChunk()`/`Close()`。`GinStreamWriter` 内部调用 HTTP Flusher，但接口层不暴露。

**意义**: 清单任务 2.1（StreamWriter 扩展）是 Provider 流式输出的前置条件。

### 发现 7: Host HTTP 无流式能力

`host_api_http.go` 使用 `io.ReadAll(httpResp.Body)` 一次性读取响应体。无 `DoStream` 方法。

**意义**: 清单任务 2.2（ProviderPlugin 接口扩展，含 StreamDelegate/PrepareStream/OnSSELine）和 2.5（Host 流式 HTTP DoStream）及 2.6（DispatchRuntime Provider 流式调度）是解决此限制的核心方案。不做这三项，Provider 插件无法处理 SSE 流式响应。

### 发现 8: Market Sync 不下载 manifest 也不验签

`sync_service.go` 仅下载 `.wasm` 文件，自行计算 SHA256，不下载 `manifest.json` 或 `signature.sig`，不验证发布者签名。

**意义**: 任务 1.8/1.9 需从头实现 manifest 解析和签名验证流程，工作量比"修复"更大。

### 发现 9: Market 已有 pluginmarket 子系统（sub2api 侧）

`sub2api` 中已有 `backend/internal/pluginmarket/` 目录，包含 `control_plane_service.go`、`lifecycle_service.go`、`dependency_resolver.go`、`registry.go` 等。

**意义**: Phase 1 的 1.14 任务（DispatchRuntime 接入）应优先利用已有的 `pluginmarket` 子系统中的生命周期管理能力，而非重新实现。

### 发现 10: pluginsign 已有完整验签能力

`sub2api` 中 `backend/internal/pluginsign/` 已实现 `VerifySignature` 和 `VerifyInstall`，含信任链校验。

**意义**: 任务 1.9 的 market 端验签可复用 `pluginsign` 的逻辑（提取为共享包或在 market 端引用），避免重复实现。

---

## 变更记录

| 日期 | 变更内容 | 原因 |
|------|---------|------|
| 2026-03-06 | V1 创建 | 基于 06 方案生成 |
| 2026-03-06 | V2 完整性修复 | 补充缺失任务(1.16/2.8/2.9/3.1)、增加依赖关系、Provider 灰度子步骤、回归测试、验收签字 |
| 2026-03-06 | V3 上线完整性修复 | 新增 0.9 pending 限制、1.17 DB 迁移、1.18 管理后台 UI、2.22 对比测试框架、2.23 Shadow/Canary 机制、2.24 WASM body 限制；新增「运维与上线准备」章节(OPS.1-8)；里程碑增加 M7 运维就绪。总计 86 项 |
| 2026-03-06 | V4 源码交叉校验 | 对照 sub2api + market 双仓库源码逐项校验：标记已完成 9 项；修正 0.2/0.6 为"部分完成"；新增 10 条源码级发现（见下方交叉校验报告） |
| 2026-03-06 | V5 深度审查同步 | 新增 Phase 0 任务 0.10(pluginruntime map 并发保护)/0.11(WriteChunk Flush 语义统一)；ProviderContext 字段扩展至 13 个(新增 IsStream/MaxTokens/OrganizationID/ProjectID/ExtraHeaders/Timeout)；ProviderResultMetadata 字段扩展至 10 个(新增 StopReason/UpstreamStatusCode/CacheCreationTokens/CacheReadTokens)；4 个 Provider 切割线全部修正（详细到函数级别）；06 方案升级至 V2(新增 §2.1.3/§2.1.4 字段表 + 风险 R9-R12)。总计 88 项 |
