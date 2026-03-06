# 插件系统实施清单

> **用途**: 逐项打勾的执行跟踪表，配合 [06-COMPLETE-IMPLEMENTATION-PLAN.md](./06-COMPLETE-IMPLEMENTATION-PLAN.md) 使用  
> **更新方式**: 完成一项勾一项，标注完成日期和负责人  
> **规则**: 每个 Phase 开始前，确认前置 Phase 全部完成；每个 Phase 结束时执行回归测试

---

## 进度总览

| Phase | 任务数 | 已完成 | 进度 | 状态 | 前置 |
|-------|-------|--------|------|------|------|
| Phase 0: 安全加固 | 8 | 0 | 0% | 未开始 | 无 |
| Phase 1: 链路打通 | 18 | 0 | 0% | 未开始 | Phase 0 |
| Phase 2: Provider 插件化 | 25 | 0 | 0% | 未开始 | Phase 1 |
| Phase 3: Transform/Interceptor | 13 | 0 | 0% | 未开始 | Phase 2.1 |
| Phase 4: 生态建设 | 17 | 0 | 0% | 未开始 | Phase 3 |
| **合计** | **81** | **0** | **0%** | | |

---

## Phase 0: 安全加固

**目标**: 修复生产安全漏洞 + 数据完整性问题  
**预计**: 1-2 周 | **仓库**: sub2api-plugin-market | **前置**: 无

### P0 安全修复 (生产阻断)

- [ ] **0.1** POST /submissions 增加 IP 级 rate limit
  - 仓库: `sub2api-plugin-market`
  - 文件: `internal/api/v1/router.go`
  - 改动: 新增 rate limit middleware（推荐 `golang.org/x/time/rate`），限制 POST /submissions 每 IP 10 次/分
  - 验证: 同一 IP 连续 15 次请求 → 第 11 次起返回 429
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **0.2** Webhook 签名强制校验
  - 仓库: `sub2api-plugin-market`
  - 文件: `internal/api/v1/handler/github_webhook_handler.go`
  - 改动: `if h.secret == ""` 时返回 500 + 日志告警，拒绝处理
  - 验证: 不配置 GITHUB_WEBHOOK_SECRET → webhook 返回 500
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **0.3** 审核操作事务化
  - 仓库: `sub2api-plugin-market`
  - 文件: `internal/admin/service/submission_service.go`
  - 改动: `ReviewSubmission` 用 `client.Tx()` 包裹 Submission 更新 + Plugin 更新，失败整体回滚
  - 验证: 模拟 Plugin 更新失败 → Submission 状态不变（仍为 pending）
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

### P1 数据完整性

- [ ] **0.4** 插件名正则校验
  - 仓库: `sub2api-plugin-market`
  - 文件: `ent/schema/plugin.go`
  - 改动: `field.String("name").Match(regexp.MustCompile("^[a-z0-9][a-z0-9-]{0,62}$"))`
  - 验证: `../hack` → 400；`My_Plugin` → 400；`my-plugin` → 通过
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **0.5** Official 插件审核角色限制
  - 仓库: `sub2api-plugin-market`
  - 文件: `internal/admin/handler/submission_handler.go`
  - 改动: 审核时查询关联 Plugin，若 `is_official=true` 则要求 `admin_user.role` 为 `super_admin` 或 `admin`
  - 验证: reviewer 审核 official 插件 → 403；admin 审核 → 通过
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **0.6** Sync 并发锁
  - 仓库: `sub2api-plugin-market`
  - 文件: `internal/service/sync_service.go`
  - 改动: `runGitHubSync` 入口加 Redis `SETNX` 或 PostgreSQL advisory lock，key=`sync:{plugin_id}:{ref}`，TTL=10min
  - 验证: 手动 Sync + Webhook Sync 同时触发同一 (plugin_id, ref) → 只有一个执行，另一个返回 "sync in progress"
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **0.7** Sync 操作顺序 + 孤儿清理
  - 仓库: `sub2api-plugin-market`
  - 文件: `internal/service/sync_service.go`
  - 改动: 调整为 ①检查版本是否存在 → ②上传 WASM → ③创建 PluginVersion；③失败时调用 `storage.Delete(wasmKey)` 清理
  - 验证: 模拟创建版本失败（如唯一约束冲突）→ 存储中无孤儿 WASM 文件
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **0.8** 审核接口乐观锁
  - 仓库: `sub2api-plugin-market`
  - 文件: `internal/admin/service/submission_service.go`
  - 改动: `client.Submission.UpdateOneID(id).Where(submission.StatusEQ(submission.StatusPending)).Set...`，受影响行数为 0 则返回 409 Conflict
  - 验证: 两管理员同时审核同一 Submission → 后到者返回 409
  - 依赖: 0.3（事务化后在此基础上加乐观锁）
  - 完成: ☐  日期: ____  负责人: ____

### Phase 0 验收门禁

- [ ] 全部 8 项 checkbox 已勾
- [ ] `make test` 通过（含新增测试用例）
- [ ] `make lint` 通过
- [ ] 安全测试覆盖: rate limit / webhook 强制 / 路径遍历 / 并发锁 / 乐观锁
- [ ] **Phase 0 完成签字**: 日期: ____  签字: ____

---

## Phase 1: 链路打通

**目标**: Echo 插件跑通「上传→审核→发布→下载→安装→执行」全链路  
**预计**: 3-4 周 | **仓库**: market + sub2api | **前置**: Phase 0 ✅

### 1.1 Schema 扩展 (market)

- [ ] **1.1** Plugin 加 plugin_type
  - 文件: `ent/schema/plugin.go`
  - 改动: `field.Enum("plugin_type").Values("interceptor","transform","provider").Optional().Comment("插件类型")`
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.2** PluginVersion 加 capabilities + config_schema
  - 文件: `ent/schema/plugin_version.go`
  - 改动: `field.JSON("capabilities", []string{}).Optional()` + `field.JSON("config_schema", map[string]any{}).Optional()`
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.3** Submission 加 plugin_version 关联
  - 文件: `ent/schema/submission.go`
  - 改动: `edge.To("plugin_version", PluginVersion.Type).Unique().Optional()`
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.4** make generate + 编译验证
  - 命令: `make generate && go mod tidy && make build && make test`
  - 验证: 编译通过，现有测试不受影响
  - 依赖: 1.1 + 1.2 + 1.3
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

- [ ] **1.7** 审核 service 联动发布版本
  - 文件: `internal/admin/service/submission_service.go`
  - 改动: approve 时在事务内：①更新 Submission → approved ②获取关联的 PluginVersion → SetStatus(published) + SetPublishedAt(time.Now()) ③更新 Plugin 元数据
  - 验证: 审核通过 → PluginVersion status=published + published_at 非空
  - 依赖: 1.6 + 0.3（事务化已完成）
  - 完成: ☐  日期: ____  负责人: ____

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

- [ ] **1.10** GET /plugins 支持 ?type= 筛选
  - 文件: `internal/repository/plugin_repository.go` + `internal/api/v1/handler/plugin_handler.go`
  - 改动: 接收 `type` query 参数，传到 repository 做 `plugin.PluginTypeEQ(...)` 过滤
  - 验证: `GET /plugins?type=provider` → 仅返回 plugin_type=provider 的插件
  - 依赖: 1.4
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.11** GET /versions 支持 ?compatible_with= 过滤
  - 文件: `internal/repository/plugin_repository.go` + `internal/api/v1/handler/plugin_handler.go`
  - 改动: 接收 `compatible_with` 参数，做 `min_api_version <= compatible_with` 和 `max_api_version >= compatible_with`（semver 比较）
  - 验证: `?compatible_with=1.2.0` → 仅返回 min≤1.2.0 且 max≥1.2.0 的版本
  - 依赖: 1.4
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.12** OpenAPI spec 同步更新
  - 文件: `openapi/plugin-market-v1.yaml`
  - 改动: 新增 plugin_type 枚举、capabilities 数组、config_schema 对象、?type= 和 ?compatible_with= 参数、multipart submission schema
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
  - 改动: 在认证/计费/并发/选号之后、`switch platform` 之前，调用 `dispatchRuntime.Dispatch(ctx, req, writer)`。若返回非 nil 响应则使用插件结果；若返回 `ErrNoProviderPlugin` 或 nil 则 fallback 到原有 `switch platform` 逻辑
  - 验证: ①注册 Echo Provider → 请求走插件返回 echo ②无插件 → 走原有 Service，行为不变
  - 依赖: 1.15
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.15** Interceptor next 链修复
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/dispatch_runtime.go`
  - 改动: Interceptor 传入的 `next` 函数改为实际执行后续阶段（TransformRequest → Provider → TransformResponse），而非返回 `(nil, nil)`
  - 验证: 注册一个 Interceptor 调用 `next()` → 获得下游 Provider 的真实响应
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **1.16** 默认 Provider 注册（内置降级）
  - 仓库: `sub2api`
  - 文件: `backend/internal/pluginruntime/` 新增 `builtin_providers.go`
  - 改动: 将 4 个内置 Service（GatewayService, OpenAIGatewayService, GeminiCompatService, AntigravityGatewayService）包装为 ProviderPlugin 接口的 adapter，注册为 priority=最低 的默认 Provider。当无外部 WASM Provider 时自动使用
  - 验证: 未安装任何外部插件 → 请求正常走内置 Service（行为 100% 不变）
  - 依赖: 1.14
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

- [ ] **2.3** ProviderContext 定义
  - 文件: `pluginapi/types.go`
  - 改动: 定义 `ProviderContext{Account AccountInfo, Token string, Platform string, BaseURL string, BaseURLs []string, ProxyURL string, MappedModel string, OriginalModel string, PlatformSpecific map[string]any}`，约定通过 `GatewayRequest.Metadata["provider_context"]` 传递
  - 依赖: 无
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.4** ProviderResultMetadata 定义
  - 文件: `pluginapi/types.go`
  - 改动: 定义 `ProviderResultMetadata{Usage UsageInfo, Model string, RequestID string, FirstTokenMs *int, Failover bool, ImageCount int, ImageSize string}`，约定通过 `GatewayResponse.Metadata["provider_result"]` 回传
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
  - 改动: 选号 + GetToken 后，将 Account/Token/BaseURL/ProxyURL/MappedModel/OriginalModel/PlatformSpecific 封装为 ProviderContext，放入 `req.Metadata["provider_context"]`
  - 验证: 插件收到的 ProviderContext 字段完整正确
  - 依赖: 2.3
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.9** 核心消费 ProviderResultMetadata
  - 文件: `backend/internal/handler/gateway_handler.go`
  - 改动: Provider 返回后从 `resp.Metadata["provider_result"]` 反序列化 ProviderResultMetadata → 调用 rateLimitService / RecordUsage / Ops 错误记录 / 检查 Failover 标记
  - 验证: 插件回传 Usage{input:100, output:50} → 计费系统扣费 150 token
  - 依赖: 2.4 + 2.6
  - 完成: ☐  日期: ____  负责人: ____

### 2.2 Provider 插件开发 + 灰度上线

每个 Provider 按 **开发→对比→shadow→canary→全量** 上线：

#### claude-provider

- [ ] **2.10** claude-provider 开发
  - 实现: manifest.json + Handle + PrepareStream + OnSSELine + OnStreamEnd
  - 核心切割: Token/identity/failover 留核心；请求构建/Header/SSE 解析/Usage 提取在插件
  - 特殊: thinking 签名重试由核心在收到 `Failover=true` 时做 body 降级重试
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
  - 核心切割: OAuth 刷新/Codex Usage Snapshot 写入留核心
  - 特殊: codex-tool-corrector 独立为 Phase 3 的 Transform 插件
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.14** openai-provider 对比测试
  - 覆盖: OAuth + APIKey / Codex CLI + 非 CLI / 流式 + 非流式 / 含 tool_calls
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.15** openai-provider 灰度上线
  - 同 2.12 步骤
  - 完成: ☐  日期: ____  负责人: ____

#### antigravity-provider

- [ ] **2.16** antigravity-provider 开发
  - 核心切割: Token/PromptTooLongError 留核心
  - 特殊: URL fallback + 429 在插件内 retry loop
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.17** antigravity-provider 对比测试
  - 覆盖: 多 URL failover + 429 重试 + 流式 + identity 注入
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.18** antigravity-provider 灰度上线
  - 同 2.12 步骤
  - 完成: ☐  日期: ____  负责人: ____

#### gemini-provider

- [ ] **2.19** gemini-provider 开发
  - 核心切割: Token/HandleTempUnschedulable 留核心
  - 前置: claude-gemini-transform (Phase 3 的 3.8) 可提前到此开发
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.20** gemini-provider 对比测试
  - 覆盖: AI Studio + Code Assist + 签名重试 + Claude↔Gemini 转换
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **2.21** gemini-provider 灰度上线
  - 同 2.12 步骤
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
  - 来源: `pkg/antigravity/`（已独立 package）
  - 实现: TransformRequest（Claude→Gemini）+ TransformResponse（Gemini→Claude）+ ChunkTransformer（流式行转换）
  - 测试: Claude↔Gemini 互转、thinking blocks、tool_use/tool_result、schema_cleaner
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **3.5** claude-gemini-transform
  - 来源: `gemini_messages_compat_service.go` 的 `convertClaudeMessagesToGemini*` + `convertGeminiToClaudeMessage`
  - 实现: TransformRequest + TransformResponse
  - 限制: body ≤ 2MB（WASM 内存安全）
  - 测试: 多轮对话、多图 base64、大 tools schema、cache_control、thinking blocks
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **3.6** codex-tool-corrector
  - 来源: `openai_tool_corrector.go` + `openai_tool_continuation.go`
  - 实现: TransformResponse（非流式完整 body 矫正）+ ChunkTransformer（流式逐行矫正）
  - 测试: apply_patch→edit、work_dir→workdir、全部 9 个映射、参数递归矫正
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
  - 来源: `claude_code_validator.go`
  - 实现: InterceptorPlugin — 校验不通过短路返回 403
  - 测试: 合法/非法 UA、system prompt Dice 相似度边界、metadata.user_id 格式
  - 完成: ☐  日期: ____  负责人: ____

- [ ] **3.10** gemini-signature-cleaner
  - 来源: `gemini_native_signature_cleaner.go`
  - 实现: InterceptorPlugin（TransformRequest 阶段）— 清理 thoughtSignature
  - 测试: 有/无 thoughtSignature 的请求
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

## 关键里程碑

| # | 里程碑 | 标志 | 目标周 | 实际日期 | 签字 |
|---|--------|------|-------|---------|------|
| M0 | 安全就绪 | Phase 0 全部通过 | Week 2 | ____ | ____ |
| M1 | 第一个插件全链路 | Echo Provider E2E 跑通 | Week 6 | ____ | ____ |
| M2 | 流式编排就绪 | Host 流式 HTTP + OnSSELine 可用 | Week 9 | ____ | ____ |
| M3 | 首个 Provider 灰度 | claude-provider 100% | Week 11 | ____ | ____ |
| M4 | 全部 Provider 就绪 | 4 个 Provider 100% | Week 13 | ____ | ____ |
| M5 | 12 个插件全部可用 | Phase 3 完成 | Week 17 | ____ | ____ |
| M6 | 生态就绪 | SDK + CLI + 文档 | Week 18+ | ____ | ____ |

---

## 阻断项跟踪

| # | 发现日期 | 描述 | 影响 Phase | 严重度 | 状态 | 解决方案 | 解决日期 |
|---|---------|------|-----------|--------|------|---------|---------|
| | | | | | | | |

---

## 变更记录

| 日期 | 变更内容 | 原因 |
|------|---------|------|
| 2026-03-06 | V1 创建 | 基于 06 方案生成 |
| 2026-03-06 | V2 完整性修复 | 补充缺失任务(1.16/2.8/2.9/3.1)、修正计数(81项)、增加每项依赖关系、Provider 灰度子步骤、回归测试、验收签字 |
