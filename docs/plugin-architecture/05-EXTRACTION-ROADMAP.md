# 插件化实施路线图

> **文档状态**: Draft  
> **创建日期**: 2026-03-06  
> **适用仓库**: sub2api (数据平面) + sub2api-plugin-market (控制平面)

> **前置文档**: 本路线图基于 [04-PLUGIN-MARKET-REVIEW.md](./04-PLUGIN-MARKET-REVIEW.md) 的改进建议制定。

## 总体策略

分 5 个 Phase 推进（Phase 0-4），每个 Phase 可独立交付价值：

```
Phase 1 (链路打通)  → 打通插件链路，市场补齐缺失字段
Phase 2 (Provider)  → 4 个 Provider 插件化
Phase 3 (Transform) → 转换和拦截器插件化
Phase 4 (生态)      → 社区工具链和生态建设
```

---

## Phase 0: 安全加固（新增，最高优先级）

**目标**: 修复当前插件市场中发现的安全漏洞和数据完整性问题（详见 [04-PLUGIN-MARKET-REVIEW.md §7](./04-PLUGIN-MARKET-REVIEW.md)）。

**预计周期**: 1-2 周

### 0.1 P0 安全修复

| 任务 | 涉及文件 | 工作量 |
|------|---------|--------|
| POST /submissions 增加 rate limit | `api/v1/router.go`、新增 middleware | 小 |
| GITHUB_WEBHOOK_SECRET 生产强制校验 | `api/v1/handler/github_webhook_handler.go` | 小 |
| Submission 审核事务化 | `admin/service/submission_service.go`（用 `client.Tx()`） | 小 |

### 0.2 P1 数据完整性修复

| 任务 | 涉及文件 | 工作量 |
|------|---------|--------|
| 插件名正则校验 `^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$` | `ent/schema/plugin.go`（schema 层加固，service 层已有） | 小 |
| Official 插件审核角色限制 | `admin/router.go` 或 handler 层 | 小 |
| SyncJob 并发锁（`plugin_id + target_ref`） | `service/sync_service.go` | 中 |
| SyncJob 失败清理孤儿 WASM | `service/sync_service.go`（调整操作顺序或增加清理） | 小 |
| 审核接口乐观锁（条件更新 `status=pending`） | `admin/service/submission_service.go` | 小 |

### 0.3 验收标准

- [ ] 无认证的 POST /submissions 有 IP 级 rate limit
- [ ] 未配置 GITHUB_WEBHOOK_SECRET 时 webhook 请求被拒绝
- [ ] 审核操作在单事务中完成
- [ ] 插件名 `../hack` 被拦截
- [ ] 并发 Sync 同一 `(plugin_id, ref)` 只有一个执行

---

## Phase 1: 链路打通

**目标**: 让 DispatchRuntime 真正接入主请求链路，插件市场补齐关键缺失。

**预计周期**: 3-4 周

### 1.1 插件市场改动

| 任务 | 涉及文件 | 工作量 |
|------|---------|--------|
| Plugin Schema 加 `plugin_type` 字段 | `ent/schema/plugin.go` + 生成代码 | 小 |
| PluginVersion 加 `capabilities` 字段 | `ent/schema/plugin_version.go` + 生成代码 | 小 |
| 列表 API 支持 `?type=` 筛选 | `repository/`, `service/`, `handler/` | 小 |
| 版本 API 支持 `?compatible_with=` 过滤 | `repository/`, `service/`, `handler/` | 中 |
| 审核通过自动发布版本（含 Submission→Version 关联） | `admin/service/submission_service.go` | 中 |
| 设计并实现 Sync→签名→发布流程 | `service/sync_service.go`、新增签名步骤 | 中 |
| OpenAPI spec 同步更新 | `openapi/plugin-market-v1.yaml` | 小 |
| Admin API 错误码统一 | 已完成，仅需同步更新 `ERROR-CODE-REGISTRY.md` 文档 | 小 |

### 1.2 主项目改动

| 任务 | 涉及文件 | 工作量 |
|------|---------|--------|
| DispatchRuntime 接入 gateway_handler | `handler/gateway_handler.go` | 中 |
| 定义 Provider 调度契约 | `pluginapi/types.go` 扩展 | 中 |
| 核心 → 插件的上下文传递机制 | `pluginapi/types.go` 中的 Metadata | 中 |

### 1.3 验收标准

- [ ] 一个最简 Echo Provider 插件可通过市场安装并在 gateway 中执行
- [ ] 市场可按 `plugin_type` 筛选插件
- [ ] 审核通过后版本自动变为 `published`

---

## Phase 2: Provider 插件化

**目标**: 将 4 个内置 Provider 抽取为独立插件。

**预计周期**: 4-6 周

### 2.0 前置条件：Host 流式编排能力

> ⚠️ **关键依赖**: TinyGo WASM 在导出函数中无法安全使用 goroutine，且 Host API HTTP 仅支持完整 body（`io.ReadAll`）。因此 **Provider 插件无法在 WASM 内部实现 SSE 流式转发**。必须先实现 Host 流式编排。

| 前置任务 | 涉及文件 | 工作量 |
|---------|---------|--------|
| Host API HTTP 增加流式 Fetch | `pluginruntime/host_api_http.go` | 高 |
| ProviderPlugin 增加 `OnSSELine()` 回调 | `pluginapi/types.go` | 中 |
| StreamWriter 增加 `Flush()` + 客户端断开检测 | `pluginapi/types.go`、`pluginruntime/writer.go` | 中 |
| Host 侧 goroutine + channel 流式管道 | `pluginruntime/dispatch_runtime.go` | 高 |

### 2.1 ProviderPlugin 接口增强

当前 `ProviderPlugin.Handle()` 签名较通用，Provider 插件需要额外的上下文：

```go
type ProviderContext struct {
    Account     AccountInfo    // 核心选好的账号（脱敏）
    Token       string         // 核心刷新好的 Token
    Platform    string         // 平台标识
    BaseURL     string         // 上游 base URL（单 URL）
    BaseURLs    []string       // 上游 base URL 列表（Antigravity 多 URL）
    ProxyURL    string         // 代理地址（可选）
    MappedModel string         // 核心映射后的模型名
    OriginalModel string       // 原始请求的模型名（计费用）
    PlatformSpecific map[string]any // project_id、chatgpt-account-id 等
}

type ProviderResultMetadata struct {
    Usage       UsageInfo      // input/output/cache tokens
    Model       string         // 实际计费模型
    RequestID   string         // 上游 request-id
    FirstTokenMs *int          // 首 token 延迟
    Failover    bool           // 是否应触发核心账号切换
    ImageCount  int            // 图片计费
    ImageSize   string         // 图片尺寸
}
```

需要扩展 `GatewayRequest.Metadata` 或定义 `ProviderPlugin` 的增强版本。
同时约定 `GatewayResponse.Metadata` 的回传格式，供核心做计费和监控。

### 2.2 插件抽取顺序

建议按复杂度从低到高：

| 顺序 | 插件 | 理由 |
|------|------|------|
| 1 | `claude-provider` | 最成熟的 Provider，逻辑最清晰 |
| 2 | `openai-provider` | 次成熟，Codex OAuth 稍复杂 |
| 3 | `antigravity-provider` | 依赖 antigravity-transform |
| 4 | `gemini-provider` | 最复杂，依赖 claude-gemini-transform + 混合调度 |

### 2.3 每个 Provider 的抽取步骤

以 `claude-provider` 为例：

```
1. 定义插件 manifest
   - name: claude-provider
   - type: provider
   - capabilities: [host_http_fetch]
   - plugin_api_version: 1.0.0

2. 提取转发逻辑
   - 从 gateway_service.go 中提取 Forward 核心
   - 剥离账号选择（留在核心）
   - 剥离 Token 获取（留在核心）

3. 确认切割线
   - 核心侧保留: Token 获取刷新、rateLimitService、failover 账号切换、Usage 计费写入
   - 插件侧负责: 请求构建（URL + Header + Body）、非流式转发与解析
   - 流式侧由 Host 编排: 插件提供 OnSSELine() 回调、Usage/Model 通过 ProviderResultMetadata 回传

4. 实现 ProviderPlugin 接口
   - Handle(): 接收 GatewayRequest + ProviderContext → 非流式请求
   - HandleStreaming() 或 OnSSELine(): 流式场景的逐行转换
   - 通过 GatewayResponse.Metadata 回传 ProviderResultMetadata

5. 测试
   - 单元测试（mock Host API）
   - 集成测试（通过 DispatchRuntime）
   - 对比测试（插件 vs 内置行为一致性）
   - 计费一致性测试（Usage、Model、ImageCount 回传正确）

6. 编译为 WASM
   - TinyGo / Go WASM 编译
   - Ed25519 签名

7. 发布到插件市场
   - 创建 Submission
   - 审核通过
   - sub2api 安装并启用
```

### 2.4 内置降级机制

Provider 插件化后，需要保留内置实现作为降级：

```
请求到达 → 检查是否有已启用的 Provider 插件
    │
    ├─ 有 → DispatchRuntime 调度插件处理
    │
    └─ 无 → 回退到内置 GatewayService（原有逻辑）
```

### 2.5 验收标准

- [ ] Host 流式编排能力就绪（goroutine + channel + OnSSELine 回调）
- [ ] 4 个 Provider 均可作为插件安装运行
- [ ] 无插件时自动降级到内置实现
- [ ] 插件版本的行为与内置版本完全一致（对比测试通过）
- [ ] 计费与用量统计与内置实现一致（含 cache token、image 计费、reasoning_effort）
- [ ] ProviderResultMetadata 回传 Usage/Model/RequestID 正确
- [ ] 流式响应延迟增加：待基准测试验证（不设具体数字，需实测后定标）

---

## Phase 3: Transform 和 Interceptor 插件化

**目标**: 将协议转换、模型映射等抽取为插件。

**预计周期**: 3-4 周

### 3.1 Transform 插件

| 顺序 | 插件 | 依赖 |
|------|------|------|
| 1 | `antigravity-transform` | 无（已是独立 package） |
| 2 | `claude-gemini-transform` | 无 |
| 3 | `codex-tool-corrector` | 无 |
| 4 | `error-mapper` | 需要收集各 Service 的映射规则 |

### 3.2 Interceptor 插件

| 顺序 | 插件 | 备注 |
|------|------|------|
| 1 | `model-mapper` | 配置驱动，优先级高 |
| 2 | `claude-code-validator` | 纯校验，简单 |
| 3 | `gemini-signature-cleaner` | 纯数据清洗，简单 |
| 4 | `tls-fingerprint` | 可能作为内置可选插件 |

### 3.3 验收标准

- [ ] 所有 Transform 插件可独立安装/卸载
- [ ] 插件组合测试通过（如 claude-gemini-transform + gemini-provider）
- [ ] 无 Transform 插件时不影响 Provider 直通

---

## Phase 4: 生态建设

**目标**: 降低社区开发插件的门槛。

**预计周期**: 4+ 周

### 4.1 市场功能增强

| 任务 | 说明 |
|------|------|
| SyncJob 增强 | Manifest 解析、签名校验、GitHub API 限流保护、历史版本同步 |
| 市场端依赖解析校验 | 审核时校验插件依赖是否可解析、无冲突 |
| 插件配置模板 | `config_schema` 字段 + 配置表单生成 |
| 插件评分评论 | 用户反馈机制 |
| 插件搜索增强 | 全文搜索、标签搜索 |

### 4.2 开发者工具

| 工具 | 说明 |
|------|------|
| `sub2api-plugin-sdk` | Go SDK，封装 pluginapi 接口 + Host API 客户端 |
| `sub2api-plugin-cli` | CLI 工具：init / build / sign / publish |
| 插件模板 | 各类型插件的 cookiecutter 模板 |
| 本地测试框架 | Mock DispatchRuntime + Host API 的测试工具 |

### 4.3 文档

| 文档 | 内容 |
|------|------|
| Plugin Developer Guide | 从零开发一个插件的完整教程 |
| Plugin API Reference | pluginapi 接口文档 |
| Host API Reference | HTTP/KV/Log 使用指南 |
| Best Practices | 性能、安全、兼容性最佳实践 |

### 4.4 验收标准

- [ ] 一个社区开发者可以在 1 小时内开发并发布一个简单插件
- [ ] CLI 工具支持完整的开发→构建→签名→发布流程
- [ ] 至少有 3 个社区贡献的插件在市场中运行

---

## 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| **TinyGo WASM goroutine 限制** | 高 | Provider 无法做 SSE 流式 | Host 流式编排（Phase 2.0 前置任务）|
| **Host API HTTP 无流式** | 高 | 同上 | Host 实现流式 Fetch |
| **WASM 内存不足** | 中 | 大 JSON 转换 OOM | body 大小限制（2MB）+ 分块处理 |
| **Provider 与核心耦合过深** | 中 | 抽取工作量大 | 先明确切割线（见 03 文档），再渐进式迁移 |
| **计费数据回传不一致** | 中 | Usage 丢失、计费错误 | ProviderResultMetadata 强类型约定 + 对比测试 |
| 插件间交互复杂 | 低 | 调试困难 | 可观测性 + 插件隔离 |
| 社区参与度不足 | 中 | 生态冷启动 | 官方先发布核心插件做示范 |
| 安全漏洞（市场端） | 中 | 提交滥用、路径遍历 | Phase 0 安全加固 |
| 安全漏洞（运行时） | 低 | 插件越权 | 能力授权 + 签名验证 + 沙箱隔离 |
| GitHub API 限流 | 中 | SyncJob 批量同步受限 | 重试 + 指数退避 + 缓存 |
| 版本语义不明确 | 低 | compatible_with 比较规则歧义 | 明确 semver 范围匹配规则 |

---

## 里程碑总结

| Phase | 交付物 | 核心价值 | 预计周期 |
|-------|--------|---------|---------|
| Phase 0 | 安全加固 + 数据完整性修复 | 市场端安全可靠 | 1-2 周 |
| Phase 1 | 链路打通 + 市场补齐 | 插件系统可用 | 3-4 周 |
| Phase 2 | Host 流式编排 + 4 个 Provider 插件 | 新 Provider 不改核心代码 | 4-6 周 |
| Phase 3 | 7 个 Transform/Interceptor 插件 | 协议转换可独立升级 | 3-4 周 |
| Phase 4 | SDK + CLI + 文档 | 社区可以贡献插件 | 4+ 周 |
