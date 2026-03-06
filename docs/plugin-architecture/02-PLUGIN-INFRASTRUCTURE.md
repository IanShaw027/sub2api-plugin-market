# 插件基础设施

> **文档状态**: Draft  
> **创建日期**: 2026-03-06  
> **适用仓库**: sub2api (数据平面)

## 概述

插件基础设施是由**主项目提供**的一套运行框架，为 WASM 插件提供安全的执行环境、生命周期管理和受控的系统能力。插件开发者面向这些抽象编程，核心团队维护框架本身。

## 架构位置

```
                    主项目核心
                        │
                        ▼
              ┌──────────────────┐
              │  插件基础设施      │  ← 本文档描述的范围
              │  (核心提供)       │
              └────────┬─────────┘
                       │
                       ▼
                  WASM 插件
```

---

## 1. 运行时 (Plugin Runtime)

核心的请求调度引擎，按阶段执行插件链。

| 模块 | 代码路径 | 职责 |
|------|---------|------|
| **DispatchRuntime** | `pluginruntime/dispatch_runtime.go` | 按 Interceptor → TransformRequest → Provider → TransformResponse 顺序调度插件 |
| **Phase Scheduler** | `pluginruntime/phase.go` | 按优先级排序同阶段内多个插件的执行顺序 |
| **Plugin Invoker** | `pluginruntime/invoke.go` | 执行单个插件调用，处理超时和错误 |
| **执行策略** | `pluginruntime/policy.go` | fail-open（插件失败放行）/ fail-closed（插件失败拦截）|
| **熔断器** | `pluginruntime/circuit_breaker.go` | 连续超时自动熔断，保护核心链路 |
| **资源预算** | `pluginruntime/budget.go` | 限制插件的内存和 CPU 消耗 |
| **可观测性** | `pluginruntime/observability.go` | 采集插件执行延迟、成功率等指标 |

### 调度流程

```
请求进入 DispatchRuntime
    │
    ├─→ [Interceptor 阶段]  按优先级依次执行
    │   可短路（直接返回响应，跳过后续阶段）
    │
    ├─→ [TransformRequest 阶段]  按优先级依次执行
    │   修改 GatewayRequest（Header、Body、Query 等）
    │
    ├─→ [Provider 阶段]  第一个能处理的 Provider 执行
    │   转发到上游 API，支持流式（StreamWriter）
    │
    └─→ [TransformResponse 阶段]  按优先级依次执行
        修改 GatewayResponse（Header、Body 等）
```

---

## 2. WASM 加载与实例管理

| 模块 | 代码路径 | 职责 |
|------|---------|------|
| **WASM Loader** | `pluginruntime/loader.go` | 基于 Wazero 编译和实例化 WASM 模块 |
| **PluginInstanceManager** | `pluginruntime/plugin_instance_manager.go` | 管理插件实例的创建、销毁、状态跟踪 |
| **Plugin Instance** | `pluginruntime/plugin_instance.go` | 单个插件实例的状态封装 |
| **Metadata** | `pluginruntime/metadata.go` | 加载和校验插件 manifest |
| **热重载协调器** | `pluginruntime/hot_reload_coordinator.go` | 不停服更新插件版本 |

---

## 3. Host API（宿主能力）

插件在 WASM 沙箱内运行，通过 Host API 获取受控的系统能力。

| Host API | 代码路径 | 提供的能力 | 需要的 Capability |
|----------|---------|-----------|------------------|
| **HTTP** | `pluginruntime/host_api_http.go` | 发起 HTTP 请求（受白名单限制） | `CapabilityHostHTTPFetch` |
| **KV** | `pluginruntime/host_api_kv.go` | 键值存储（读/写/删） | 读: `CapabilityHostKVRead`，写: `CapabilityHostKVWrite` |
| **Log** | `pluginruntime/host_api_log.go` | 日志输出（Debug/Info/Warn/Error） | `CapabilityHostLogWrite` |

### 能力授权

```
pluginruntime/capability.go   ← Capability 常量与 CapabilityAuthorizer 接口
pluginruntime/host_api.go     ← HostAPIGuard 结构体，Require/RequireAll 方法
    │
    ├── CapabilityAuthorizer  ← 接口
    │   检查插件 manifest 声明了哪些 capability
    │
    └── HostAPIGuard
        在每次 Host API 调用前校验权限
```

---

## 4. 流式写入

| 模块 | 代码路径 | 职责 |
|------|---------|------|
| **StreamWriter** | `pluginruntime/writer.go` | 将 Gin ResponseWriter 适配为 pluginapi.StreamWriter 接口 |

StreamWriter 接口定义（来自 `pluginapi/types.go`）：

```go
type StreamWriter interface {
    State() WriteState
    SetHeader(key, value string) error
    WriteChunk(chunk []byte) error
    Close() error
}
```

---

## 5. Plugin API 契约

所有插件面向这些接口编程。

| 类型 | 代码路径 | 定义 |
|------|---------|------|
| **Plugin** | `pluginapi/types.go` | 基础接口：`Metadata() Metadata` |
| **InterceptorPlugin** | `pluginapi/types.go` | `Intercept(ctx, req, next) (*GatewayResponse, error)` |
| **TransformPlugin** | `pluginapi/types.go` | `TransformRequest(ctx, req) error` + `TransformResponse(ctx, resp) error` |
| **ProviderPlugin** | `pluginapi/types.go` | `Handle(ctx, req, writer) (*GatewayResponse, error)` |

### 核心数据类型

| 类型 | 字段 |
|------|------|
| **GatewayRequest** | Method, Path, Headers, Query, Body, Stream, Metadata |
| **GatewayResponse** | StatusCode, Headers, Body, Metadata |
| **Metadata** (pluginapi) | Name, Version, Description |
| **Metadata** (pluginruntime) | ID, Name, Version, Description, PluginAPIVersion 等（含更多 manifest 信息） |

---

## 6. 签名验证

| 模块 | 代码路径 | 职责 |
|------|---------|------|
| **签名验证** | `pluginsign/pluginsign.go` | Manifest 校验、哈希比对、版本兼容性 |
| **签名器** | `pluginsign/signer.go` | Ed25519 签名生成 |
| **Trust Store** | `pluginsign/trust_store_loader.go` | 加载信任公钥（本地 + 远端） |

验证链路（按代码实际执行顺序）：

```
安装前 → VerifyInstall()
    ├── 1. ValidateManifest (manifest 完整性)
    ├── 2. CheckHostCompatibility (plugin_api_version 兼容性)
    ├── 3. VerifySHA256 (WASM 哈希)
    └── 4. TrustStore.VerifySignature (Ed25519 签名 + sign_key_id)
```

---

## 7. 插件市场客户端（控制面集成）

| 模块 | 代码路径 | 职责 |
|------|---------|------|
| **控制面服务** | `pluginmarket/control_plane_service.go` | 编排 preflight → install → enable/disable → upgrade → rollback → uninstall |
| **HTTP Registry** | `pluginmarket/http_registry_store.go` | 连接远端 plugin-market 的 REST API |
| **Registry 接口** | `pluginmarket/registry_store.go` | RegistryStore 接口定义 |
| **Registry 类型** | `pluginmarket/registry.go` | RegistryEntry 等类型 |
| **生命周期类型** | `pluginmarket/lifecycle.go` | LifecycleState、DesiredEffectiveState、Transition 等核心类型 |
| **生命周期服务** | `pluginmarket/lifecycle_service.go` | 管理插件的 desired vs effective 状态 |
| **Lockfile** | `pluginmarket/lockfile_store.go` | 持久化安装状态到本地文件 |
| **依赖类型** | `pluginmarket/dependency.go` | Dependency、ResolvedDependency、DependencyGraph 类型 |
| **依赖解析** | `pluginmarket/dependency_resolver.go` | 插件间依赖和冲突检测 |
| **Manifest** | `pluginmarket/manifest.go` | Manifest 类型（re-export pluginsign） |
| **审计事件** | `pluginmarket/audit_event.go` | 记录插件操作日志（内存/文件/Postgres） |
| **提交服务** | `pluginmarket/submission_service.go` | 提交新插件/版本到市场 |

### 生命周期状态机

代码中使用 `desired` (期望) 和 `effective` (实际) 双状态模型，`Reconcile` 负责在重启后对齐两者。

```
(未安装)
    │
    ▼  install
 disabled (StateDisabled)
    │
    ├─→ enable  → active (StateActive, 运行中)
    │                │
    │                ├─→ disable → disabled
    │                ├─→ upgrade → active (新版本)
    │                └─→ 插件崩溃 → failed (StateFailed)
    │
    ├─→ 热重载 → pending_restart (StatePendingRestart)
    │                │
    │                └─→ 重启完成 → active
    │
    └─→ uninstall → (未安装)

任何 active 状态可 rollback → active (旧版本)
```

> **注意**: 简化文档中使用 "installed/enabled"，实际代码状态名为 `StateDisabled`/`StateActive`/`StatePendingRestart`/`StateFailed`。

---

## 8. 示例插件

| 示例 | 代码路径 | 演示的能力 |
|------|---------|-----------|
| Echo Provider | `pluginruntime/examples/provider_echo.go` | ProviderPlugin 基础实现 |
| Header Transform | `pluginruntime/examples/transform_header.go` | TransformPlugin 基础实现 |
| Guard Interceptor | `pluginruntime/examples/interceptor_guard.go` | InterceptorPlugin 基础实现 |
| Minimal Register | `pluginruntime/examples/register_minimal.go` | 最小注册流程 |
