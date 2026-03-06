# 可插件化模块分析

> **文档状态**: Draft  
> **创建日期**: 2026-03-06  
> **适用仓库**: sub2api (数据平面)

## 概述

本文档分析 sub2api 主项目中可以抽取为插件的模块，按可行性分为三档。共识别出 **12 个候选插件**：4 个 Provider、3 个 Transform、5 个 Interceptor。

---

## 可行性分档标准

| 等级 | 含义 | 条件 |
|------|------|------|
| 🟢 高 | 应该插件化 | 逻辑独立、边界清晰、无核心状态依赖 |
| 🟡 中 | 抽象后可插件化 | 需要先定义新接口或重构共享逻辑 |
| 🔴 低 | 保留在核心 | 涉及安全/资金/全局协调 |

---

## 🟢 高可行性 — 应优先插件化

### P-01: claude-provider

| 属性 | 值 |
|------|-----|
| **插件类型** | `ProviderPlugin` |
| **当前代码** | `service/gateway_service.go` + `pkg/claude/constants.go` |
| **核心逻辑** | Claude/Anthropic API 转发、SSE 流式、Claude Code 模拟、模型 ID 归一化 |
| **外部依赖** | HTTP 上游 (`api.anthropic.com`) |
| **需要的 Host API** | HTTP Fetch |
| **抽取难度** | 中 — 需要把账号选择、Token 获取从 Forward 中分离 |

**抽取方案**：
- 核心提供：已选好的 Account + Token + 请求 Body
- 插件负责：构建 Anthropic 请求 Header、转发、解析 SSE、提取 Usage
- 插件输入：`GatewayRequest` + `Account` metadata
- 插件输出：`GatewayResponse` 或 SSE 流（通过 `StreamWriter`）

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
- 核心提供：Account（含 OAuth/API Key 凭证）、Body
- 插件负责：URL 构建（Codex vs Platform）、Header 构建、转发、SSE 解析、Usage 提取
- 特殊处理：Codex Usage 来自响应 Header，需要插件回传 metadata

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
- 核心提供：Account、Token、目标格式（Claude or Gemini native）
- 插件负责：请求转换、转发、响应转换、SSE 流式

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

---

### T-02: antigravity-transform

| 属性 | 值 |
|------|-----|
| **插件类型** | `TransformPlugin` |
| **当前代码** | `pkg/antigravity/request_transformer.go` + `response_transformer.go` + `stream_transformer.go` |
| **核心逻辑** | Claude ↔ Gemini 转换（Antigravity 特化版本），流式 SSE 转换 |
| **外部依赖** | 无 |
| **需要的 Host API** | 无 |
| **抽取难度** | 低 — 已经是独立 package，函数入出参清晰 |

**代码清单**：
- `request_transformer.go` → `TransformClaudeToGeminiWithOptions()` → `TransformRequest`
- `response_transformer.go` → `TransformGeminiToClaude()` → `TransformResponse`
- `stream_transformer.go` → `NewStreamingProcessor()` → SSE 流式转换 + `NewNonStreamingProcessor()` → 非流式转换
- `schema_cleaner.go` → `CleanJSONSchema()` + `DeepCleanUndefined()` → Schema 清理辅助

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

> **注意**: 工具名矫正在 `openai_tool_corrector.go`（非 `openai_codex_transform.go`）；`openai_codex_transform.go` 主要是 Codex OAuth 请求转换逻辑，属于核心。

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
| **插件类型** | `InterceptorPlugin` |
| **当前代码** | `pkg/tlsfingerprint/dialer.go` + `registry.go` |
| **核心逻辑** | TLS ClientHello 指纹伪装，可选能力 |
| **外部依赖** | 无 |
| **需要的 Host API** | 无（需要特殊的网络层 Hook） |
| **抽取难度** | 高 — 涉及底层 TCP/TLS 连接，WASM 沙箱可能无法直接操作 |

> **注意**: 此模块可能更适合作为「内置可选插件」而非 WASM 插件，因为 TLS 指纹需要操作系统级网络访问。

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

| 编号 | 插件名 | 类型 | 可行性 | 抽取难度 | 代码量估计 |
|------|--------|------|--------|---------|-----------|
| P-01 | `claude-provider` | Provider | 🟢 高 | 中 | ~800 行 |
| P-02 | `openai-provider` | Provider | 🟢 高 | 中 | ~700 行 |
| P-03 | `gemini-provider` | Provider | 🟢 高 | 高 | ~1200 行 |
| P-04 | `antigravity-provider` | Provider | 🟢 高 | 高 | ~1000 行 |
| T-01 | `claude-gemini-transform` | Transform | 🟢 高 | 中 | ~600 行 |
| T-02 | `antigravity-transform` | Transform | 🟢 高 | 低 | ~800-1000 行 |
| T-03 | `codex-tool-corrector` | Transform | 🟢 高 | 低 | ~350-400 行 |
| I-01 | `model-mapper` | Interceptor | 🟢 高 | 低 | ~300 行 |
| I-02 | `tls-fingerprint` | Interceptor | 🟡 中 | 高 | ~400 行 |
| I-03 | `claude-code-validator` | Interceptor | 🟢 高 | 低 | ~150 行 |
| I-04 | `gemini-signature-cleaner` | Interceptor | 🟢 高 | 低 | ~100 行 |
| I-05 | `error-mapper` | Transform | 🟢 高 | 中 | ~300 行 |

### 潜在候选（未来可考虑）

以下模块边界较清晰但当前优先级不高，可在 Phase 4+ 考虑：

| 模块 | 类型 | 来源 | 说明 |
|------|------|------|------|
| `gemini-retry-stripper` | Transform | `gemini_messages_compat_service.go` | 签名重试时降级 thinking/tool 块 (`FilterThinkingBlocksForRetry`) |
| `gemini-thought-signature-injector` | Transform | `gemini_messages_compat_service.go` | 为 functionCall 注入 `thoughtSignature` |
| `response-header-filter` | Interceptor | `util/responseheaders/` | 响应头过滤 |
| `gemini-schema-cleaner` | Transform | `gemini_messages_compat_service.go` | Gemini 请求 Schema 清理，与 T-02 类似 |
