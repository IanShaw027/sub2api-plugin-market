# 插件化实施路线图

> **文档状态**: Draft  
> **创建日期**: 2026-03-06  
> **适用仓库**: sub2api (数据平面) + sub2api-plugin-market (控制平面)

## 总体策略

分 4 个 Phase 推进，每个 Phase 可独立交付价值：

```
Phase 1 (基础)     → 打通插件链路，市场补齐缺失字段
Phase 2 (Provider)  → 4 个 Provider 插件化
Phase 3 (Transform) → 转换和拦截器插件化
Phase 4 (生态)      → 社区工具链和生态建设
```

---

## Phase 1: 基础设施完善

**目标**: 让 DispatchRuntime 真正接入主请求链路，插件市场补齐关键缺失。

**预计周期**: 2-3 周

### 1.1 插件市场改动

| 任务 | 涉及文件 | 工作量 |
|------|---------|--------|
| Plugin Schema 加 `plugin_type` 字段 | `ent/schema/plugin.go` + 生成代码 | 小 |
| PluginVersion 加 `capabilities` 字段 | `ent/schema/plugin_version.go` + 生成代码 | 小 |
| 列表 API 支持 `?type=` 筛选 | `repository/`, `service/`, `handler/` | 小 |
| 版本 API 支持 `?compatible_with=` 过滤 | `repository/`, `service/`, `handler/` | 中 |
| 审核通过自动发布版本 | `admin/service/submission_service.go` | 小 |
| OpenAPI spec 同步更新 | `openapi/plugin-market-v1.yaml` | 小 |
| Admin API 错误码统一 | `admin/handler/` 相关文件 | 中 |

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

### 2.1 ProviderPlugin 接口增强

当前 `ProviderPlugin.Handle()` 签名较通用，Provider 插件需要额外的上下文：

```go
type ProviderContext struct {
    Account     AccountInfo    // 核心选好的账号
    Token       string         // 核心刷新好的 Token
    Platform    string         // 平台标识
    BaseURL     string         // 上游 base URL
    ProxyURL    string         // 代理地址（可选）
}
```

需要扩展 `GatewayRequest.Metadata` 或定义 `ProviderPlugin` 的增强版本。

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

3. 实现 ProviderPlugin 接口
   - Handle(): 接收 GatewayRequest + ProviderContext
   - 构建 Anthropic 请求
   - 通过 Host API HTTP 转发
   - 解析 SSE / JSON 响应
   - 通过 StreamWriter 写回

4. 测试
   - 单元测试（mock Host API）
   - 集成测试（通过 DispatchRuntime）
   - 对比测试（插件 vs 内置行为一致性）

5. 编译为 WASM
   - TinyGo / Go WASM 编译
   - Ed25519 签名

6. 发布到插件市场
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

- [ ] 4 个 Provider 均可作为插件安装运行
- [ ] 无插件时自动降级到内置实现
- [ ] 插件版本的行为与内置版本完全一致（对比测试通过）
- [ ] 流式响应延迟增加不超过 5ms

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
| SyncJob 真实实现 | GitHub Release → WASM → Storage → Version |
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
| WASM 性能不达标 | 中 | 流式延迟增加 | 基准测试 + 内置降级 |
| Provider 逻辑和核心耦合过深 | 中 | 抽取工作量大 | 渐进式重构，先抽简单的 |
| 插件间交互复杂 | 低 | 调试困难 | 可观测性 + 插件隔离 |
| 社区参与度不足 | 中 | 生态冷启动 | 官方先发布核心插件做示范 |
| 安全漏洞 | 低 | 插件越权 | 能力授权 + 签名验证 + 沙箱隔离 |

---

## 里程碑总结

| Phase | 交付物 | 核心价值 |
|-------|--------|---------|
| Phase 1 | 链路打通 + 市场补齐 | 插件系统可用 |
| Phase 2 | 4 个 Provider 插件 | 新 Provider 不改核心代码 |
| Phase 3 | 7 个 Transform/Interceptor 插件 | 协议转换可独立升级 |
| Phase 4 | SDK + CLI + 文档 | 社区可以贡献插件 |
