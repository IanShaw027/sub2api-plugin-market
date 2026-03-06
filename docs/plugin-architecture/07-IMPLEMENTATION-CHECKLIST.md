# 插件系统实施清单

> **用途**: 逐项打勾的执行跟踪表，配合 [06-COMPLETE-IMPLEMENTATION-PLAN.md](./06-COMPLETE-IMPLEMENTATION-PLAN.md) 使用  
> **更新方式**: 完成一项勾一项，标注完成日期和负责人  
> **规则**: 每个 Phase 开始前，确认前置 Phase 全部完成；每个 Phase 结束时执行回归测试

---

## 进度总览

| Phase | 任务数 | 已完成 | 进度 | 状态 | 前置 |
|-------|-------|--------|------|------|------|
| Phase 0: 安全加固 | 11 | 5 | 45% | 进行中 | 无 |
| Phase 1: 链路打通 | 18 | 4 | 22% | 进行中 | Phase 0 |
| Phase 2: Provider 插件化 | 24 | 0 | 0% | 未开始 | Phase 1 |
| Phase 3: Transform/Interceptor | 10 | 0 | 0% | 未开始 | Phase 2.1 |
| Phase 4: 生态建设 | 17 | 0 | 0% | 未开始 | Phase 3 |
| 运维与上线准备 | 8 | 0 | 0% | 未开始 | 随各 Phase 同步 |
| **合计** | **88** | **9** | **10%** | | |

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

- [ ] **0.4** 插件名正则校验（Schema 层加固）
  - 文件: `ent/schema/plugin.go`
  - 现状: 正则校验在 `internal/service/submission_service.go` 的 service 层已有 `^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$`
  - 改动: 在 schema 层增加 `Match()` 作为双保险（防止 service 层被绕过）
  - 验证: 直接用 Ent client 创建 `../hack` → schema 层拦截
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

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

- [ ] **0.7** Sync 操作顺序 + 孤儿清理
  - 文件: `internal/service/sync_service.go`
  - 现状: 当前顺序为 下载 WASM → 上传存储 → 创建版本（顺序正确），但创建版本失败时未清理已上传 WASM
  - 改动: 创建版本失败时调用 `storage.Delete(wasmKey)` 清理
  - 验证: 模拟创建版本失败 → 存储中无孤儿 WASM
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [x] **0.8** 审核接口乐观锁
  - 文件: `internal/admin/service/submission_service.go`
  - 现状: ✅ 已有 `Where(submission.StatusEQ(submission.StatusPending))` 条件更新
  - 完成: ☑  日期: 已完成  负责人: ____

- [ ] **0.9** 同一插件 pending Submission 数量限制
  - 文件: `internal/service/submission_service.go`
  - 现状: ❌ 无此逻辑
  - 改动: `CreateSubmission` 前查询同一插件是否有 pending 提交
  - 验证: 同一插件连续提交 2 次 → 第 2 次返回 409
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

### P0 运行时安全 (V5 新增)

- [ ] **0.10** pluginruntime map 并发保护
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/dispatch_runtime.go`
  - 现状: `DispatchRuntime.plugins` 是普通 `map`，热重载注册/卸载与请求调度并发时会 panic
  - 改动: 改用 `sync.RWMutex` 保护 plugin map（读请求用 RLock，注册/卸载用 Lock），或改为 `sync.Map`
  - 验证: 100 并发请求 + 同时注册新插件 → 无 panic
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **0.11** GinStreamWriter.WriteChunk Flush 语义统一
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/writer.go`
  - 现状: `GinStreamWriter.WriteChunk()` 内部隐式调用 `Flush()`，与 DispatchRuntime 显式调用 `Flush()` 产生重复 Flush
  - 改动: `WriteChunk` 只写缓冲不自动 Flush，由调度层统一控制 Flush 时机
  - 验证: 流式输出正常 + 无多余 Flush 开销
  - 依赖: 2.1 (StreamWriter 扩展)
  - 完成: ☐  日期: ____  负责人: ____

### Phase 0 验收门禁

- [ ] 全部 11 项中：5 项已完成、6 项待完成（0.4/0.6/0.7/0.9/0.10/0.11）
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

- [ ] **1.4** make generate + 编译验证
  - 命令: `make generate && go mod tidy && make build && make test`
  - 验证: 编译通过，现有测试不受影响
  - 依赖: 1.1 + 1.2 + 1.3（均已完成）
  - 完成: ☐  日期: ____  负责人: ____

### 1.2 WASM 上传 + 签名 + 发布 (market)

**路径 A: 手动上传**

- [ ] **1.5** 提交 handler 支持 multipart 上传
  - 文件: `internal/api/v1/handler/submission_handler.go`
  - 改动: 改为 `multipart/form-data` 接收 `wasm_file`(binary) + `manifest`(json) + `signature`(base64) + `sign_key_id`(string) + 原有元数据字段
  - 验证: `curl -F wasm_file=@plugin.wasm -F manifest='{"name":"echo-provider"...}' ...` → 201
  - 依赖: 1.4
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.6** submission_service 完整创建流程
  - 文件: `internal/service/submission_service.go`
  - 改动: ①验证 manifest JSON 格式 → ②计算 WASM SHA-256 → ③验签（调用 `pluginsign.VerifySignature`）→ ④上传 WASM 到 storage → ⑤创建/更新 Plugin（含 plugin_type）→ ⑥创建 PluginVersion（status=draft, wasm_url, wasm_hash, signature, sign_key_id, capabilities, min/max_api_version）→ ⑦创建 Submission（关联 plugin_version_id）
  - 验证: 合法 WASM+签名 → draft 版本创建成功；伪造签名 → 400
  - 依赖: 1.5
  - 完成: ☐  日期: ____  负责人: ____

- [x] **1.7** 审核 service 联动发布版本
  - 文件: `internal/admin/service/submission_service.go`
  - 现状: ✅ 已实现 — approve 时在事务内将关联的 draft PluginVersion 更新为 published + 设置 published_at
  - 完成: ☑  日期: 已完成  负责人: ____

**路径 B: GitHub Sync 自动发布**

- [ ] **1.8** Sync 下载 manifest.json + signature.sig
  - 文件: `internal/service/sync_service.go`
  - 改动: 在 `fetchReleaseAssets` 中额外下载 `manifest.json` 和 `signature.sig`（从 Release assets 中按文件名匹配）
  - 验证: Release 含 3 个文件 → 全部下载成功；缺少 manifest → SyncJob 标记 failed + 错误原因
  - 依赖: 1.4
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.9** Sync 解析 manifest + 验签 + 创建 published 版本
  - 文件: `internal/service/sync_service.go`
  - 改动: ①解析 manifest.json 提取 plugin_type, capabilities, min/max_api_version ②计算 WASM SHA-256 ③验签（pluginsign.VerifySignature）④上传 WASM ⑤创建 PluginVersion（status=published, 含完整签名信息 + manifest 字段）
  - 验证: sub2api `GET /download` → 验签成功 → 302 预签名 URL
  - 依赖: 1.8
  - 完成: ☐  日期: ____  负责人: ____

### 1.3 API 增强 (market)

- [ ] **1.10** GET /plugins 支持 ?type= 筛选（handler + repository 实现）
  - 文件: `internal/repository/plugin_repository.go` + `internal/api/v1/handler/plugin_handler.go`
  - 现状: ⚠️ OpenAPI 已定义 `type` query param（enum: interceptor/transform/provider），但需确认 handler 和 repository 是否实际消费该参数
  - 改动: handler 接收 `type` 参数 → repository 做 `plugin.PluginTypeEQ(...)` 过滤
  - 验证: `GET /plugins?type=provider` → 仅返回 plugin_type=provider 的插件
  - 依赖: 1.4
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.11** GET /versions 支持 ?compatible_with= 过滤（handler + repository 实现）
  - 文件: `internal/repository/plugin_repository.go` + `internal/api/v1/handler/plugin_handler.go`
  - 现状: ⚠️ OpenAPI 已定义 `compatible_with` query param，但需确认 handler 和 repository 是否实际消费
  - 改动: handler 接收参数 → repository 做 semver 范围比较
  - 验证: `?compatible_with=1.2.0` → 仅返回 min≤1.2.0 且 max≥1.2.0 的版本
  - 依赖: 1.4
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.12** OpenAPI spec 同步更新
  - 文件: `openapi/plugin-market-v1.yaml`
  - 现状: ⚠️ 已有 `type` 和 `compatible_with` 参数定义；缺少 `capabilities` 数组 schema 和 multipart submission schema
  - 改动: 新增 capabilities 数组 schema、config_schema 对象、multipart submission request body schema
  - 验证: `make check-contract` 通过
  - 依赖: 1.10 + 1.11
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.13** ERROR-CODE-REGISTRY 同步更新
  - 文件: `docs/ERROR-CODE-REGISTRY.md`
  - 改动: 新增 Upload 相关错误码（签名校验失败、manifest 格式错误等）
  - 验证: `make check-contract` 通过
  - 依赖: 1.6
  - 完成: ☐  日期: ____  负责人: ____

### 1.4 DispatchRuntime 接入 (sub2api 主项目)

- [ ] **1.14** DispatchRuntime 接入 gateway_handler
  - 仓库: `sub2api`
  - 文件: `backend/internal/handler/gateway_handler.go`
  - 源码现状: handler 用 `if/else` 判断 `platform == PlatformGemini / PlatformAntigravity / 其他`，直接调用各 Service 的 `ForwardWithInput`。Dispatch() 仅在测试/examples 中使用
  - 改动: 在认证/计费/并发/选号（`SelectAccountWithLoadAwareness`）之后、platform `if/else` 之前，调用 `dispatchRuntime.Dispatch(ctx, req, writer)`。若返回非 nil 则用插件结果；若返回 `ErrNoProviderPlugin` 则 fallback 到原有 `if/else`
  - 架构决策: 可利用已有的 `backend/internal/pluginmarket/` 子系统（`lifecycle_service.go`、`registry.go`）做插件发现和生命周期管理
  - 验证: ①注册 Echo Provider → 请求走插件返回 echo ②无插件 → 走原有 Service
  - 依赖: 1.15
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.15** Interceptor next 链修复
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/dispatch_runtime.go`
  - 源码现状: `next` 为 `func(context.Context, *pluginapi.GatewayRequest)(*pluginapi.GatewayResponse, error) { return nil, nil }`（永远空返回）
  - 改动: `next` 改为闭包，执行后续阶段（TransformRequest → Provider → TransformResponse → 返回 GatewayResponse）
  - 注意: 同时需决定 `phase.go` 中 7 阶段定义 vs `dispatch_runtime.go` 中 4 阶段执行的不一致——建议保留 4 阶段（intercept/transform/provider/transform_response），移除未使用的 pre_auth/post_auth/post_proxy/log
  - 验证: 注册一个 Interceptor 调用 `next()` → 获得下游 Provider 的真实响应
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.16** 默认 Provider 注册（内置降级）
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/` 新增 `builtin_providers.go`
  - 改动: 将 4 个内置 Service 包装为 ProviderPlugin adapter：
    - `gateway_service.go` → `BuiltinClaudeProvider`
    - `openai_gateway_service.go` → `BuiltinOpenAIProvider`
    - `gemini_messages_compat_service.go` → `BuiltinGeminiProvider`
    - `antigravity_gateway_service.go` → `BuiltinAntigravityProvider`
  - 注册为 priority=最低 的默认 Provider。当无外部 WASM Provider 时自动使用
  - 验证: 未安装任何外部插件 → 请求正常走内置 Service（行为 100% 不变）
  - 依赖: 1.14
  - 完成: ☐  日期: ____  负责人: ____

### 1.5 部署准备 (market)

- [ ] **1.17** 数据库迁移脚本
  - 文件: `ent/migrate/` 生成文件 + 生产迁移 SQL
  - 改动: Phase 0 的 name 正则 + Phase 1 的 plugin_type/capabilities/config_schema/submission→version edge 需生产环境 ALTER TABLE。导出迁移 SQL 并在 staging 先行验证
  - 验证: staging 环境执行迁移 → 现有数据不丢失 → 应用启动正常
  - 依赖: 1.4
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.18** 管理后台 UI 适配
  - 文件: `frontend/` 或 `web/admin/`
  - 改动: 提交列表/详情页展示 plugin_type、capabilities；审核页展示关联的 PluginVersion 信息；筛选器增加按 type 过滤
  - 验证: 管理员可在后台看到插件类型并按类型筛选
  - 依赖: 1.4
  - 完成: ☐  日期: ____  负责人: ____

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

- [ ] **2.1** StreamWriter 扩展
  - 文件: `pluginapi/types.go` + `pluginruntime/writer.go`
  - 改动: 接口新增 `Flush() error`、`SetStatus(code int)`、`Done() <-chan struct{}`；writer.go 实现对接 Gin 的 `http.Flusher` 和 `context.Done()`
  - 验证: 单元测试 — Flush 后客户端立即收到数据；Done channel 在客户端断开时关闭
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.2** ProviderPlugin 接口扩展
  - 文件: `pluginapi/types.go`
  - 改动: 新增 `StreamDelegate{URL, Method string; Headers map[string][]string}`；ProviderPlugin 新增 `PrepareStream(ctx, req) (*StreamDelegate, error)`、`OnSSELine(line []byte) ([]byte, error)`、`OnStreamEnd() (*ProviderResultMetadata, error)`
  - 验证: Mock Provider 实现新接口 → 编译通过
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

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

- [ ] **2.4** ProviderResultMetadata 定义（详见 06 方案 §2.1.4）
  - 文件: `pluginapi/types.go`
  - 改动: 定义 ProviderResultMetadata 结构体，完整字段:
    - `InputTokens int` — 输入 token 数
    - `OutputTokens int` — 输出 token 数
    - `TotalTokens int` — 总 token 数
    - `ActualModel string` — 上游实际返回模型名
    - `StopReason string` — 停止原因
    - `NeedFailover bool` — 是否需要 failover
    - `FailoverReason string` — failover 原因
    - `UpstreamStatusCode int` — 上游 HTTP 状态码
    - `CacheCreationTokens int` — Claude cache 写入 token（可选）
    - `CacheReadTokens int` — Claude cache 读取 token（可选）
  - 约定: 通过 `GatewayResponse.Metadata["provider_result"]` 回传
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.5** Host 流式 HTTP
  - 文件: `pluginruntime/host_api_http.go`
  - 改动: 新增 `DoStream(req HTTPRequest, onLine func([]byte) error) error`，内部 goroutine + bufio.Scanner 逐行读取 SSE body
  - 验证: 对接 mock SSE 端点 → onLine 回调逐行收到数据 → 流结束返回 nil
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.6** DispatchRuntime Provider 流式调度
  - 文件: `pluginruntime/dispatch_runtime.go`
  - 改动: Provider 阶段判断 `req.Stream`：true 时 ①调 PrepareStream 获取 StreamDelegate ②Host DoStream 发起流式 HTTP ③每行调 OnSSELine 获取转换后的行 ④WriteChunk + Flush ⑤流结束调 OnStreamEnd 获取 Metadata；false 时走原有 Handle 路径
  - 验证: Mock SSE Provider → 客户端逐行收到数据 → Usage 正确回传
  - 依赖: 2.1 + 2.2 + 2.5
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.7** keepalive + interval timeout
  - 文件: `pluginruntime/dispatch_runtime.go`
  - 改动: 流式管道中增加 keepalive Ticker（如 30s 发 `: keepalive\n\n`）和上游数据间隔 Ticker（如 5min 无数据则超时断开）
  - 验证: 上游暂停 35s → 客户端收到 keepalive 注释；上游暂停 6min → 连接超时关闭
  - 依赖: 2.6
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.8** 核心注入 ProviderContext
  - 文件: `backend/internal/handler/gateway_handler.go`
  - 源码现状: handler 当前在 `SelectAccountWithLoadAwareness` 后直接调用各 ForwardWithInput，Account/Token/Model 分散在不同位置
  - 改动: 在 platform 分支之前统一构建 ProviderContext → 放入 `req.Metadata["provider_context"]`。需从当前分散的 `account`, `token`, `mappedModel` 变量收集
  - 注意: 当前 RecordUsage 在 `go func()` 中异步执行，ProviderContext 需包含 `InputTokens`/`OutputTokens` 以支持计费
  - 验证: 插件收到的 ProviderContext 字段完整正确
  - 依赖: 2.3
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.9** 核心消费 ProviderResultMetadata
  - 文件: `backend/internal/handler/gateway_handler.go`
  - 源码现状: RecordUsage 使用 `ForwardResult` 中的 Usage 信息，在 handler 的 `go func()` 异步记录
  - 改动: Provider 返回后从 `resp.Metadata["provider_result"]` 反序列化 → 替代当前 ForwardResult 的 Usage 字段 → 调用 RecordUsage
  - 验证: 插件回传 Usage{input:100, output:50} → 计费系统扣费 150 token
  - 依赖: 2.4 + 2.6
  - 完成: ☐  日期: ____  负责人: ____

### 2.2 Provider 插件开发 + 灰度上线

每个 Provider 按 **开发→对比→shadow→canary→全量** 上线：

#### claude-provider

- [ ] **2.10** claude-provider 开发
  - 源文件: `backend/internal/service/gateway_service.go`（入口 `ForwardWithInput`/`Forward`）
  - 核心保留: 选号/`GetAccessToken`、identity 指纹注入（`ApplyFingerprint`）、failover 决策循环、`RecordUsage`
  - 插件职责: 请求构建（Header 含 `anthropic-version`/`anthropic-beta`）、SSE 解析（`event:` 分发）、thinking content block 处理、Usage 提取（`message_start.usage` + `message_delta.usage`）
  - 特殊: 插件返回 `NeedFailover=true` + `FailoverReason="thinking_budget_token"` → 核心做 body 降级后重试
  - 注意: `claude_code_validator.go` 的 Claude Code CLI 检测逻辑应留核心（与认证相关）
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.11** claude-provider 对比测试
  - 验证: 非流式 body 一致 + 流式逐行一致 + Usage/Model/RequestID 一致 + 错误响应一致 + Failover 触发一致
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.12** claude-provider 灰度上线
  - 步骤: Shadow(1周) → Canary 10%(3天) → 50%(3天) → 100%
  - 回滚条件: Usage 偏差 >1% 或错误率上升 >0.1%
  - 完成: ☐  日期: ____  负责人: ____

#### openai-provider

- [ ] **2.13** openai-provider 开发
  - 源文件: `backend/internal/service/openai_gateway_service.go`（入口 `Forward`）+ `openai_codex_transform.go`
  - 核心保留: OAuth Token 刷新（`EnsureValidToken`）、Codex Usage Snapshot 写入（`codexUsageSnapshotService`）、选号
  - 插件职责: 请求构建（区分 Codex `codex-1` URL 与标准 Platform URL）、SSE 解析（`[DONE]` 终止）、model 字段替换
  - 特殊: `openai_tool_corrector.go` 的 `CodexToolCorrector` 独立为 Phase 3 Transform 插件
  - 注意: OpenAI WebSocket 转发 (`openai_ws_forwarder.go`) 暂不插件化（WASM 无 WebSocket 能力）
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.14** openai-provider 对比测试
  - 覆盖: OAuth + APIKey / Codex CLI + 非 CLI / 流式 + 非流式 / 含 tool_calls
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.15** openai-provider 灰度上线
  - 同 2.12 步骤
  - 完成: ☐  日期: ____  负责人: ____

#### antigravity-provider

- [ ] **2.16** antigravity-provider 开发
  - 源文件: `backend/internal/service/antigravity_gateway_service.go`（入口 `ForwardWithInput`/`ForwardGeminiWithInput`）
  - 依赖包: `backend/internal/pkg/antigravity/`（含 client/oauth/request_transformer/response_transformer/stream_transformer/schema_cleaner/gemini_types/claude_types）
  - 核心保留: Token 获取（`client.GetToken`）、`PromptTooLongError` 判断、`client.go`/`oauth.go` 完整保留
  - 插件职责: 请求构建（URL 路径拼接）、429 重试循环（插件内 `for` + `time.Sleep`）、SSE 解析、Gemini/Claude 类型映射
  - 特殊: 插件需要 `BaseURLs []string` 列表做 URL fallback；`pkg/antigravity/client.go` 和 `pkg/antigravity/oauth.go` 不抽出
  - 风险: WASM 内 `time.Sleep` 可能阻塞宿主线程（见 06 方案 R10），考虑将 retry 提升到 Host 侧
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.17** antigravity-provider 对比测试
  - 覆盖: 多 URL failover + 429 重试 + 流式 + identity 注入
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.18** antigravity-provider 灰度上线
  - 同 2.12 步骤
  - 完成: ☐  日期: ____  负责人: ____

#### gemini-provider

- [ ] **2.19** gemini-provider 开发
  - 源文件: `backend/internal/service/gemini_messages_compat_service.go`（入口 `ForwardWithInput`/`Forward`/`ForwardNative`）
  - 核心保留: Token 获取、`HandleTempUnschedulable` 状态检查、rate limiter 核心调度
  - 插件职责: 请求构建（区分 AI Studio 与 Code Assist URL/认证方式）、SSE 解析（JSON 数组拆分）、`usageMetadata` 提取
  - 前置: claude-gemini-transform (Phase 3 的 3.5) 可提前到此阶段并行开发
  - 注意: 有 ForwardNative（Gemini 原生 API）路径，需一并插件化；`thoughtSignature` 清理由 `gemini-signature-cleaner` Interceptor 处理
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.20** gemini-provider 对比测试
  - 覆盖: AI Studio + Code Assist + 签名重试 + Claude↔Gemini 转换
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.21** gemini-provider 灰度上线
  - 同 2.12 步骤
  - 完成: ☐  日期: ____  负责人: ____

### 2.3 上线基础设施

- [ ] **2.22** 对比测试框架
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/consistency_test.go`（新文件）
  - 改动: 实现 `ConsistencyTest` 结构体：同一请求同时走内置和插件，自动对比 StatusCode/Body/Usage/Model/Failover。支持流式逐行对比
  - 验证: 人为制造 Usage 偏差 → 框架报告差异
  - 依赖: 2.6
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.23** Shadow/Canary 流量切分机制
  - 仓库: `sub2api`
  - 文件: `backend/internal/handler/gateway_handler.go` + `backend/internal/pluginruntime/traffic_split.go`（新文件）
  - 改动: 实现三种模式：①Shadow（双写，仅内置响应）②Canary（按百分比路由）③Full（100% 插件）。通过环境变量或配置文件控制每个 Provider 的模式和百分比
  - 验证: 设 claude-provider=canary:10% → 约 10% 请求走插件
  - 依赖: 1.14
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.24** WASM body 大小限制
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/dispatch_runtime.go`
  - 改动: 在将 request body 传入 WASM 前检查大小。Provider: 超过 limit 则 fallback 内置；Transform: 超过 limit 则跳过该 Transform 并记日志。默认 2MB，可配置
  - 验证: 发送 3MB body + Transform 插件 → Transform 被跳过，请求正常处理
  - 依赖: 2.6
  - 完成: ☐  日期: ____  负责人: ____

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

- [ ] **3.1** Config Host API
  - 文件: `pluginapi/types.go`(接口) + `pluginruntime/host_api_config.go`(新文件) + `pluginruntime/capability.go`(新增 `CapabilityHostConfigRead`)
  - 改动: 定义 `ConfigReader{Get(key) (string,error); GetAll() (map[string]string,error)}`；实现从配置文件加载，按 pluginID namespace 隔离
  - 验证: model-mapper 插件可通过 `config.Get("model_mapping")` 读取映射表
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **3.2** TransformPlugin 增加 ChunkTransformer 可选接口
  - 文件: `pluginapi/types.go`
  - 改动: `type ChunkTransformer interface { TransformChunk(chunk []byte) ([]byte, error) }`
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **3.3** Host 流式管道链式调用
  - 文件: `pluginruntime/dispatch_runtime.go`
  - 改动: 流式管道中 Provider.OnSSELine 返回后，依次调用注册的 ChunkTransformer 链再写 StreamWriter
  - 验证: codex-tool-corrector 注册为 ChunkTransformer → 流式 SSE 中的 tool_calls 被矫正
  - 依赖: 3.2 + 2.6
  - 完成: ☐  日期: ____  负责人: ____

### 3.2 Transform 插件 (4 个)

- [ ] **3.4** antigravity-transform
  - 来源: `backend/internal/pkg/antigravity/`（7 个文件：client/oauth/request_transformer/response_transformer/stream_transformer/schema_cleaner/types）
  - 实现: TransformRequest（Claude→Gemini `request_transformer.go`）+ TransformResponse（Gemini→Claude `response_transformer.go`）+ ChunkTransformer（流式 `stream_transformer.go`）
  - 测试: Claude↔Gemini 互转、thinking blocks、tool_use/tool_result、schema_cleaner
  - 注意: `client.go` 和 `oauth.go` 属于核心（HTTP 客户端+OAuth），不应提取到插件
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **3.5** claude-gemini-transform
  - 来源: `backend/internal/service/gemini_messages_compat_service.go` 的 `convertClaudeMessagesToGemini*` + `convertGeminiToClaudeMessage`
  - 实现: TransformRequest + TransformResponse
  - 限制: body ≤ 2MB（WASM 内存安全）
  - 测试: 多轮对话、多图 base64、大 tools schema、cache_control、thinking blocks
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **3.6** codex-tool-corrector
  - 来源: `backend/internal/service/openai_tool_corrector.go`（`CodexToolCorrector` 含 9 个工具名映射：apply_patch→edit 等）
  - 实现: TransformResponse（非流式完整 body 矫正）+ ChunkTransformer（流式逐行 `CorrectToolCallsInSSEData`）
  - 测试: 全部 9 个映射、参数递归矫正（work_dir→workdir）、`GetToolNameMapping` 覆盖
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **3.7** error-mapper
  - 来源: 各 Service 散落的错误映射（`mapGeminiErrorBodyToClaudeError` 等）
  - 实现: TransformResponse（统一各厂商错误格式为 Claude 格式）
  - 测试: Gemini 400/429/5xx、Antigravity 错误、OpenAI 错误 → 统一格式
  - 完成: ☐  日期: ____  负责人: ____

### 3.3 Interceptor 插件 (3 个)

- [ ] **3.8** model-mapper
  - 来源: 散落在各 Service 的模型映射逻辑
  - 实现: InterceptorPlugin — 在 Intercept 阶段修改 `req.Body` 中的 model 字段
  - 依赖: 3.1（Config Host API 读取映射表）
  - 测试: Claude NormalizeModelID、Codex codexModelMap、Antigravity prefixMapping、account 级 model_mapping
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **3.9** claude-code-validator
  - 来源: `backend/internal/service/claude_code_validator.go`（含 Validate/ValidateUserAgent/IncludesClaudeCodeSystemPrompt/IsClaudeCodeClient/SetClaudeCodeClient）
  - 实现: InterceptorPlugin — 校验不通过短路返回 403
  - 注意: 检测逻辑含 UA 匹配 `claude-cli/x.x.x`、System Prompt Dice 系数、Headers（X-App/anthropic-beta）、metadata.user_id 格式
  - 决策点: 此逻辑与认证安全强相关，是否保留核心也可接受 — 需评估
  - 测试: 合法/非法 UA、system prompt Dice 相似度边界、metadata.user_id 格式
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **3.10** gemini-signature-cleaner
  - 来源: `backend/internal/service/gemini_native_signature_cleaner.go`（`CleanGeminiNativeThoughtSignatures`）
  - 实现: InterceptorPlugin（TransformRequest 阶段）— 清理 thoughtSignature，替换为 dummy 签名防止跨账户验证错误
  - 测试: 有/无 thoughtSignature 的请求、sticky session 切账户场景
  - 完成: ☐  日期: ____  负责人: ____

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

- [ ] **4.5** 审核时依赖解析校验
  - 文件: `internal/admin/service/submission_service.go`
  - 改动: approve 前调用 `dependency_resolver` 校验 dependencies 是否可解析、无冲突
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.6** 列表 API Redis 缓存
  - 文件: `internal/repository/plugin_repository.go`
  - 改动: GET /plugins 结果缓存到 Redis，TTL=3min，写操作时清缓存
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.7** 预签名 URL 缓存
  - 文件: `internal/service/download_service.go`
  - 改动: 相同 `(wasm_url, 5min 窗口)` 复用同一预签名 URL
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.8** Trust Key 多代并存轮换
  - 文件: `internal/service/verification_service.go`
  - 改动: 旧 key 标记 `deprecated` 但保留 `is_active=true` 一段时间（如 30 天），期间旧签名仍可验证
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.9** 插件搜索增强
  - 文件: `internal/repository/plugin_repository.go`
  - 改动: PostgreSQL `tsvector` 全文搜索 + 标签过滤
  - 完成: ☐  日期: ____  负责人: ____

### 4.3 运行时增强

- [ ] **4.10** 热重载实现
  - 文件: `pluginruntime/hot_reload_coordinator.go`
  - 改动: 实现 `HotReloadPlugin`: 旧实例 draining（处理进行中请求），新实例处理新请求
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.11** Prometheus metrics 导出
  - 文件: `pluginruntime/observability.go`
  - 改动: 导出 `plugin_dispatch_duration_seconds`, `plugin_dispatch_total`, `plugin_circuit_breaker_state` 等
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.12** 错误率熔断
  - 文件: `pluginruntime/circuit_breaker.go`
  - 改动: 除超时外，支持 `error_rate_threshold`（如连续 10 次中 5 次失败则熔断）
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.13** Log Host API 限速
  - 文件: `pluginruntime/host_api_log.go`
  - 改动: 每插件每秒最多 100 条日志，超出则采样
  - 完成: ☐  日期: ____  负责人: ____

### 4.4 文档

- [ ] **4.14** Plugin Developer Guide — 从零开发、构建、签名、发布的完整教程
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.15** Plugin API Reference — pluginapi 所有接口详细文档
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.16** Host API Reference — HTTP/KV/Log/Config 使用指南 + 限制
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **4.17** Best Practices — WASM 内存、body 限制、错误处理、性能调优
  - 完成: ☐  日期: ____  负责人: ____

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

- [ ] **OPS.1** 生产环境变量清单
  - 交付: 一份文档或 `.env.example`，列出所有新增/变更的环境变量
  - 涵盖: `GITHUB_WEBHOOK_SECRET`(强制)、Rate Limit 配置、Redis 连接（如用 SETNX）、WASM body 大小限制、Shadow/Canary 配置、Config Host API 配置目录
  - 验证: 新部署按文档配置后一次启动成功
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **OPS.2** 数据库迁移预演
  - 交付: 在 staging 环境完整执行一次 DB 迁移（Phase 0 的 name 校验 + Phase 1 的 schema 扩展），记录耗时和影响
  - 验证: staging 迁移后 → 旧数据正常读取 → 新字段 nullable → 应用启动正常
  - 完成: ☐  日期: ____  负责人: ____

### B. 部署流程

- [ ] **OPS.3** 部署顺序与回滚预案
  - 交付: 文档明确每个 Phase 的部署顺序：
    - Phase 0/1: **先部署 market → 再部署 sub2api**（market 提供新 API 后 sub2api 才能调用）
    - Phase 2/3: **先部署 sub2api**（新运行时能力） → 再安装插件
  - 回滚: 每个 Phase 部署前备份 DB → 出问题回滚二进制 + DB
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **OPS.4** CI/CD Pipeline 更新
  - 涵盖:
    - market 仓库: `make check-contract` 加入 CI（已有） + 新增 rate limit / 乐观锁 / 并发锁测试
    - sub2api 仓库: 新增 WASM 插件编译步骤（TinyGo） + 签名步骤 + 对比测试步骤
    - 签名密钥: CI 中的 Ed25519 私钥安全存储（GitHub Secrets / Vault）
  - 验证: PR 合并后自动编译 + 签名 + 测试
  - 完成: ☐  日期: ____  负责人: ____

### C. 监控告警

- [ ] **OPS.5** 监控大盘 + 告警规则
  - 涵盖:
    - **market**: Submission 提交量/审核延迟/SyncJob 成功率/下载 QPS
    - **sub2api 插件**: Dispatch 延迟/插件错误率/熔断状态/WASM 内存使用
    - **灰度期**: 内置 vs 插件 Usage 偏差率/延迟对比
  - 告警:
    - 插件错误率 >1% → P1 告警
    - Usage 偏差 >0.5% → P1 告警（灰度期）
    - WASM OOM → P0 告警
    - SyncJob 连续失败 3 次 → P2 告警
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **OPS.6** Health Check 端点增强
  - 文件: market `cmd/server/main.go` + sub2api health endpoint
  - 改动: health check 增加 DispatchRuntime 状态（已注册插件数/熔断插件数）+ market 增加 DB/Storage/Redis 连通性检查
  - 完成: ☐  日期: ____  负责人: ____

### D. 风险缓解

- [ ] **OPS.7** GitHub API 限流保护
  - 文件: `internal/service/sync_service.go`
  - 改动: GitHub API 调用增加 retry + exponential backoff（初始 1s，最大 60s，最多 3 次）；批量 Sync 时限制并发数（如 3 个并发）
  - 验证: 模拟 GitHub 429 → 重试后成功
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **OPS.8** Semver 兼容性匹配规则文档
  - 交付: 文档明确 `?compatible_with=` 的匹配算法：`min_api_version <= X` 且 `(max_api_version == "" || max_api_version >= X)`，使用 Go `semver` 包的比较语义
  - 验证: 边界用例测试（如 `1.0.0-beta` vs `1.0.0`）
  - 完成: ☐  日期: ____  负责人: ____

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
