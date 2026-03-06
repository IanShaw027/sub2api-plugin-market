# 可插件化模块分析

> **文档状态**: Draft  
> **创建日期**: 2026-03-06  
> **适用仓库**: sub2api (数据平面)

## 概述

本文档分析 sub2api 主项目中可以抽取为插件的模块，按可行性分为三档。共识别出 **12 个候选插件**：4 个 Provider、4 个 Transform、3 个 Interceptor、1 个内置可选。

---

## WASM 运行时限制（关键约束）

在评估可行性前，必须理解以下 WASM 技术限制：

| 限制 | 说明 | 影响范围 |
|------|------|---------|
| **TinyGo goroutine** | TinyGo 编译的 WASM 在导出函数中无法安全使用 goroutine | 所有 Provider 的 SSE 流式处理 |
| **Host API HTTP 无流式** | `host_api_http.go` 使用 `io.ReadAll`，只能返回完整 body | Provider 无法在插件内做 SSE 转发 |
| **StreamWriter 无 Flush** | 当前接口缺少 `Flush()`、`SetStatus()`、客户端断开检测 | 流式写入能力不完整 |
| **内存边界** | 大 JSON 转换峰值内存可达 body 的 3-5 倍 | Transform 处理超长对话时可能 OOM |

**核心结论**：Transform / Interceptor 类插件（纯数据转换）可直接以 WASM 实现。Provider 类插件需要 **Host 负责流式编排** 的架构支持（见下文）。

### Provider 插件的可行架构：Host 流式编排

```
核心选号/Token → 调用插件
                    │
                    ├─ 非流式: 插件通过 Host API HTTP → 完整 Body → 解析 → StreamWriter
                    │
                    └─ 流式: 插件返回 StreamDelegate（URL + Headers）
                             → Host 发起流式 HTTP（goroutine + channel）
                             → 每行回调插件 OnSSELine(line) 做转换
                             → Host 通过 StreamWriter 写出
```

需要扩展：
- `ProviderPlugin` 增加 `HandleStreaming()` 或 `OnSSELine()` 回调接口
- Host API HTTP 增加流式 Fetch 能力
- StreamWriter 增加 `Flush()` 和客户端断开检测

---

## 可行性分档标准

| 等级 | 含义 | 条件 |
|------|------|------|
| 🟢 高 | 可直接 WASM 插件化 | 纯数据转换、无 I/O、无并发 |
| 🟡 中 | 需架构支持后可插件化 | 依赖 Host 流式编排或接口扩展 |
| 🔴 低 | 保留在核心 | 涉及安全/资金/全局协调/操作系统级 |

---

## 🟡 需架构支持 — Provider 插件

### P-01: claude-provider

| 属性 | 值 |
|------|-----|
| **插件类型** | `ProviderPlugin` |
| **当前代码** | `service/gateway_service.go` + `pkg/claude/constants.go` |
| **核心逻辑** | Claude/Anthropic API 转发、SSE 流式、Claude Code 模拟、模型 ID 归一化 |
| **外部依赖** | HTTP 上游 (`api.anthropic.com`) |
| **需要的 Host API** | HTTP Fetch |
| **抽取难度** | 中 — 需要把账号选择、Token 获取从 Forward 中分离 |
| **WASM 限制** | SSE 流式需 Host 编排；thinking 签名重试归属待定 |

**抽取方案**：
- 核心提供：已选好的 Account + Token + 请求 Body（通过 `ProviderContext`）
- 插件负责：构建 Anthropic 请求 Header、非流式转发、响应解析、Usage 提取
- 流式场景：插件提供 `OnSSELine()` 回调，Host 负责 goroutine + channel + keepalive
- 核心保留：Token 获取刷新、identity 指纹注入、rateLimitService 调用、failover 账号切换

---

### P-02: openai-provider

| 属性 | 值 |
|------|-----|
| **插件类型** | `ProviderPlugin` |
| **当前代码** | `service/openai_gateway_service.go` + `pkg/openai/` |
| **核心逻辑** | OpenAI/Codex API 转发、OAuth vs API Key 双通道、Codex 模型归一化、SSE 流式 |
| **外部依赖** | HTTP 上游 (`api.openai.com`, `chatgpt.com`) |
| **需要的 Host API** | HTTP Fetch |
| **抽取难度** | 中 — Codex OAuth 路径较复杂 |

**抽取方案**：
- 核心提供：Account（含 OAuth/API Key 凭证）、Body（通过 `ProviderContext`）
- 插件负责：URL 构建（Codex vs Platform）、Header 构建、响应解析、Usage 提取
- 流式场景：Host 编排，插件提供 `OnSSELine()` 回调
- 特殊处理：Codex Usage 来自响应 Header，由 Host 提取后注入回调参数
- 核心保留：OAuth 刷新（在 TokenProvider 内）、Codex Usage Snapshot 写入 account、failover

---

### P-03: gemini-provider

| 属性 | 值 |
|------|-----|
| **插件类型** | `ProviderPlugin` |
| **当前代码** | `service/gemini_messages_compat_service.go` + `pkg/gemini/` + `pkg/geminicli/` |
| **核心逻辑** | Gemini API 转发、Claude↔Gemini 格式互转、AI Studio vs Code Assist 双模式、签名重试 |
| **外部依赖** | HTTP 上游 (`generativelanguage.googleapis.com`) |
| **需要的 Host API** | HTTP Fetch |
| **抽取难度** | 高 — 含协议转换 + 签名重试 + Antigravity 混合调度 |

**抽取方案**：
- 可拆分为 `gemini-provider` (转发) + `claude-gemini-transform` (转换)
- 核心提供：Account、Token、目标格式（通过 `ProviderContext`）
- 插件负责：请求构建（AI Studio vs Code Assist URL）、响应解析
- 流式场景：Host 编排，插件提供 `OnSSELine()` 做 Gemini→Claude SSE 转换
- 核心保留：Token 获取、HandleGeminiUpstreamError、HandleTempUnschedulable、failover

---

### P-04: antigravity-provider

| 属性 | 值 |
|------|-----|
| **插件类型** | `ProviderPlugin` |
| **当前代码** | `service/antigravity_gateway_service.go` + `pkg/antigravity/` |
| **核心逻辑** | Antigravity v1internal API 转发、Claude↔Gemini 转换、身份注入、URL Fallback、签名重试 |
| **外部依赖** | HTTP 上游 (Antigravity 多个 base URL) |
| **需要的 Host API** | HTTP Fetch |
| **抽取难度** | 高 — URL 多重 fallback + 身份注入 |

**抽取方案**：
- 可拆分为 `antigravity-provider` (转发) + `antigravity-transform` (转换)
- 转换层 (`pkg/antigravity/`) 已经是独立模块，天然适合抽取
- 核心提供：Account、Token、BaseURLs 列表（通过 `ProviderContext`）
- 插件内重试：URL fallback + 429 是 Antigravity 特有逻辑，适合在插件内处理
- 流式场景：Host 编排，插件提供 `OnSSELine()` 做 Gemini→Claude SSE 转换
- 核心保留：Token 获取、PromptTooLongError 判断、failover 触发、rateLimitService

---

### T-01: claude-gemini-transform

| 属性 | 值 |
|------|-----|
| **插件类型** | `TransformPlugin` |
| **当前代码** | `gemini_messages_compat_service.go` 内 `convertClaudeMessagesToGemini*` + `convertGeminiToClaudeMessage` |
| **核心逻辑** | Claude Messages 格式 ↔ Gemini GenerateContent 格式互转 |
| **外部依赖** | 无 |
| **需要的 Host API** | 无 |
| **抽取难度** | 中 — 函数边界清晰但代码量大 |

**转换内容**：
- `TransformRequest`: Claude role/content → Gemini parts/role
- `TransformResponse`: Gemini candidates → Claude content blocks
- 特殊处理：thinking blocks、tool_use/tool_result、system prompt、cache_control

**WASM 注意**：大 JSON 转换峰值内存可达 body 的 3-5 倍（原始 + map + 序列化），超长对话/多图/大工具 schema 时需设上限（建议 2MB body limit）。

---

### T-02: antigravity-transform

| 属性 | 值 |
|------|-----|
| **插件类型** | `TransformPlugin` |
| **当前代码** | `pkg/antigravity/request_transformer.go` + `response_transformer.go` + `stream_transformer.go` + `schema_cleaner.go` |
| **核心逻辑** | Claude ↔ Gemini 转换（Antigravity 特化版本），流式 SSE 转换 |
| **外部依赖** | 无 |
| **需要的 Host API** | 无 |
| **抽取难度** | 低 — 已经是独立 package，函数入出参清晰 |

**代码清单**：
- `request_transformer.go` → `TransformClaudeToGeminiWithOptions()` → `TransformRequest`
- `response_transformer.go` → `TransformGeminiToClaude()` → `TransformResponse`
- `stream_transformer.go` → `NewStreamingProcessor()` → SSE 流式转换 + `NewNonStreamingProcessor()` → 非流式转换
- `schema_cleaner.go` → `CleanJSONSchema()` + `DeepCleanUndefined()` → Schema 清理辅助

**WASM 注意**：`StreamingProcessor` 的 `ProcessLine()` 本身是同步的（处理单行 SSE），但调用方在 goroutine 中循环调用。WASM 化时，转换逻辑可迁入插件（`OnSSELine` 回调），流式管道编排必须由 Host 完成。

---

### T-03: codex-tool-corrector

| 属性 | 值 |
|------|-----|
| **插件类型** | `TransformPlugin` |
| **当前代码** | `service/openai_tool_corrector.go` + `openai_tool_continuation.go` |
| **核心逻辑** | Codex CLI 工具名矫正 (`apply_patch` → `edit`)、工具续传处理 |
| **外部依赖** | 无 |
| **需要的 Host API** | 无 |
| **抽取难度** | 低 — 纯数据转换 |

> **注意**: 工具名矫正在 `openai_tool_corrector.go`；Codex OAuth 请求转换逻辑在 `openai_gateway_service.go` 内部，属于核心。

---

### I-01: model-mapper

| 属性 | 值 |
|------|-----|
| **插件类型** | `InterceptorPlugin` |
| **当前代码** | 散落在各 service 的模型映射逻辑 |
| **核心逻辑** | 模型 ID 归一化/反归一化、支持模型列表、模型别名 |
| **外部依赖** | 无 |
| **需要的 Host API** | 无（或 KV 用于缓存映射表） |
| **抽取难度** | 低 — 配置驱动 |

**散落位置**：
- `pkg/claude/constants.go` → `NormalizeModelID()`, `DenormalizeModelID()`
- `service/openai_gateway_service.go` → `normalizeCodexModel()`
- `service/antigravity_gateway_service.go` → `antigravitySupportedModels`, `antigravityPrefixMapping`
- `pkg/gemini/models.go` → `DefaultModels()`, `FallbackModelsList()`

---

### I-02: tls-fingerprint

| 属性 | 值 |
|------|-----|
| **插件类型** | 内置可选模块（非 WASM 插件） |
| **当前代码** | `pkg/tlsfingerprint/dialer.go` + `registry.go` |
| **核心逻辑** | TLS ClientHello 指纹伪装，可选能力 |
| **外部依赖** | 无 |
| **需要的 Host API** | 不适用（需要操作系统级网络访问） |
| **抽取难度** | 高 — 涉及底层 TCP/TLS 连接，WASM 沙箱无法直接操作 |

> **注意**: 此模块更适合作为「内置可选模块」（配置开关），而非 WASM 插件。TLS 指纹需要操作系统级 TCP 连接控制，超出 WASM 沙箱能力。

---

### I-03: claude-code-validator

| 属性 | 值 |
|------|-----|
| **插件类型** | `InterceptorPlugin` |
| **当前代码** | `service/claude_code_validator.go` |
| **核心逻辑** | 校验请求是否符合 Claude Code 规范 |
| **外部依赖** | 无 |
| **需要的 Host API** | 无 |
| **抽取难度** | 低 — 纯校验逻辑 |

---

### I-04: gemini-signature-cleaner

| 属性 | 值 |
|------|-----|
| **插件类型** | `InterceptorPlugin` (或 `TransformPlugin`) |
| **当前代码** | `service/gemini_native_signature_cleaner.go` |
| **核心逻辑** | 清理 Gemini native 请求中的 thoughtSignature 字段 |
| **外部依赖** | 无 |
| **需要的 Host API** | 无 |
| **抽取难度** | 低 — 纯数据清洗 |

---

### I-05: error-mapper

| 属性 | 值 |
|------|-----|
| **插件类型** | `TransformPlugin` (TransformResponse 阶段) |
| **当前代码** | 散落在各 service |
| **核心逻辑** | 将各厂商的错误格式映射为统一的客户端错误格式 |
| **外部依赖** | 无 |
| **需要的 Host API** | 无 |
| **抽取难度** | 中 — 需要从各 service 中收集错误映射规则 |

**散落位置**：
- `gemini_messages_compat_service.go` → `mapGeminiErrorBodyToClaudeError()`
- `antigravity_gateway_service.go` → 错误状态码映射
- `openai_gateway_service.go` → 错误处理逻辑

---

## 🟡 中可行性 — 抽象后可插件化

以下模块目前和核心耦合较深，需要先定义抽象接口才能插件化。

| 模块 | 当前状态 | 插件化路径 | 预计工作量 |
|------|---------|-----------|-----------|
| **账号选择策略** | 4 个 Service 各自实现 sticky/LRU/priority | 抽出 `AccountSelector` 接口，默认内置实现，允许插件覆盖 | 中 |
| **流式处理管道** | 每个 Service 都有 SSE goroutine + channel 模式 | 抽出 `SSEPipeline` 框架，插件实现 `OnChunk` 钩子 | 高 |
| **OAuth Provider** | Claude/Gemini/OpenAI/Antigravity 各自实现 | 进一步抽象 `TokenProvider` 接口 | 中 |
| **配额获取** | `antigravity_quota_fetcher.go`, `gemini_quota.go` | 抽出 `QuotaFetcher` 接口 | 低 |

---

## 🔴 不可插件化 — 必须保留在核心

详见 [01-CORE-MODULES.md](./01-CORE-MODULES.md)，核心理由汇总：

| 模块类别 | 不可外移理由 |
|---------|-------------|
| 认证鉴权 | 安全边界，WASM 沙箱不可信 |
| 计费/用量 | 资金安全，不能让插件绕过扣费 |
| 并发控制 | 全局原子操作 |
| 速率限制 | 全局原子操作 |
| 调度选号 | 跨请求全局状态 |
| Token 管理 | 凭证安全 |
| 用户/订阅 | 核心数据实体 |

---

## 插件清单汇总

> **代码量说明**: 下表中的「代码量估计」指**抽取后的插件代码量**（仅含需要迁入插件的逻辑），而非当前源文件的总行数。实际源文件通常更大（含核心耦合部分、测试等），抽取时需进一步核对。

| 编号 | 插件名 | 类型 | WASM 可行性 | 抽取难度 | 前置条件 | 代码量估计 |
|------|--------|------|-----------|---------|---------|-----------|
| P-01 | `claude-provider` | Provider | ⚠️ 需 Host 流式编排 | 中 | Host 流式 HTTP + OnSSELine 回调 | ~800 行 |
| P-02 | `openai-provider` | Provider | ⚠️ 需 Host 流式编排 | 中 | 同上 + Usage Header 提取 | ~700 行 |
| P-03 | `gemini-provider` | Provider | ⚠️ 需 Host 流式编排 | 高 | 同上 + 内存限制 | ~1200 行 |
| P-04 | `antigravity-provider` | Provider | ⚠️ 需 Host 流式编排 | 高 | 同上 + BaseURLs 列表 | ~1000 行 |
| T-01 | `claude-gemini-transform` | Transform | ✅ 直接可行 | 中 | body ≤ 2MB 限制 | ~600 行 |
| T-02 | `antigravity-transform` | Transform | ✅ 直接可行 | 低 | 流式管道由 Host 编排 | ~800-1000 行 |
| T-03 | `codex-tool-corrector` | Transform | ✅ 直接可行 | 低 | 无 | ~350-400 行 |
| I-01 | `model-mapper` | Interceptor | ✅ 直接可行 | 低 | 无 | ~300 行 |
| I-02 | `tls-fingerprint` | 内置可选模块 | ❌ 操作系统级 | 高 | N/A (非 WASM) | ~400 行 |
| I-03 | `claude-code-validator` | Interceptor | ✅ 直接可行 | 低 | 无 | ~150 行 |
| I-04 | `gemini-signature-cleaner` | Interceptor | ✅ 直接可行 | 低 | 无 | ~100 行 |
| I-05 | `error-mapper` | Transform | ✅ 直接可行 | 中 | 无 | ~300 行 |

### Provider 抽取时核心必须保留的能力

无论哪个 Provider 被插件化，以下能力必须保留在核心侧：

| 能力 | 理由 |
|------|------|
| Token 获取与刷新 | 凭证安全，插件不可接触 refresh_token |
| rateLimitService 调用 | 全局限流状态，跨请求/跨账号 |
| accountRepo 状态更新 | SetRateLimited / SetAntigravityQuotaScopeLimit 等 |
| Failover 账号切换 | 跨账号调度，需全局视图 |
| Ops 错误记录 | 全局可观测性 |
| Usage 计费写入 | 资金安全 |

### Provider 数据回传约定

插件执行后需通过 `GatewayResponse.Metadata` 回传以下信息供核心计费和监控：

| Key | 类型 | 用途 |
|-----|------|------|
| `usage` | `{input, output, cache_creation, cache_read}` | 计费必需 |
| `model` | string | 实际计费模型 |
| `request_id` | string | 上游 request-id，审计追踪 |
| `first_token_ms` | *int | 首 token 延迟，性能监控 |
| `failover` | bool | 是否应触发核心账号切换 |
| `image_count` / `image_size` | int / string | 图片模型计费 |

### 潜在候选（未来可考虑）

以下模块边界较清晰但当前优先级不高，可在 Phase 4+ 考虑：

| 模块 | 类型 | 来源 | 说明 |
|------|------|------|------|
| `gemini-retry-stripper` | Transform | `service/gateway_request.go`（定义处），在 `gemini_messages_compat_service.go` 等处被调用 | 签名重试时降级 thinking/tool 块 (`FilterThinkingBlocksForRetry`) |
| `gemini-thought-signature-injector` | Transform | `gemini_messages_compat_service.go` | 为 functionCall 注入 `thoughtSignature` |
| `response-header-filter` | Interceptor | `util/responseheaders/` | 响应头过滤 |
| `gemini-schema-cleaner` | Transform | `gemini_messages_compat_service.go` | Gemini 请求 Schema 清理，与 T-02 类似 |
