# 插件架构总览

> **文档状态**: Draft  
> **创建日期**: 2026-03-06  
> **适用仓库**: sub2api (数据平面) + sub2api-plugin-market (控制平面)

## 1. 背景

Sub2API 是一个 AI API 网关，支持将 AI 订阅配额（Claude、OpenAI、Gemini、Antigravity、Sora 等）通过 API Key 分发。当前所有 Provider 的转发逻辑、协议转换、模型映射都硬编码在主项目中。

本文档系列分析了哪些模块应保留在核心、哪些应插件化，以及插件市场设计的评审和改进建议。

> **关键发现**: Provider 插件化受 WASM 技术限制（TinyGo 无法在导出函数中使用 goroutine、Host API HTTP 不支持流式响应），需要「Host 负责流式编排」的架构支持。Transform / Interceptor 类插件可直接以 WASM 实现。此外，插件市场存在安全漏洞（提交无认证、Webhook 签名可跳过等），需优先修复。详见各分文档。

## 2. 核心设计原则

```
涉及安全 / 资金 / 全局协调 → 留在核心
涉及特定厂商协议 / 格式 / 策略 → 做成插件
```

## 3. 三层架构

```
┌─────────────────────────────────────────────────┐
│                 🔒 主项目核心                      │
│  认证 · 计费 · 并发控制 · 限流 · 调度 · 用户管理    │
│  API Key · Ops 监控 · 管理后台 · 配置 · 存储       │
├─────────────────────────────────────────────────┤
│              ⚙️ 插件基础设施 (核心提供)             │
│  DispatchRuntime · WASM Loader · Host API        │
│  熔断器 · 能力授权 · 热重载 · 生命周期管理          │
├─────────────────────────────────────────────────┤
│               🔌 插件 (可分发)                     │
│  Claude Provider · OpenAI Provider               │
│  Gemini Provider · Antigravity Provider          │
│  Claude↔Gemini 转换 · 模型映射 · TLS 指纹         │
│  Codex 工具矫正 · 错误映射 · 自定义拦截器          │
│  (Sora 能力为独立 Handler，暂不纳入插件化范围)      │
└─────────────────────────────────────────────────┘
```

## 4. 文档索引

| 文档 | 内容 |
|------|------|
| [01-CORE-MODULES.md](./01-CORE-MODULES.md) | 主项目核心模块完整清单（~75 个模块） |
| [02-PLUGIN-INFRASTRUCTURE.md](./02-PLUGIN-INFRASTRUCTURE.md) | 插件基础设施模块清单（~35 个模块） |
| [03-PLUGGABLE-MODULES.md](./03-PLUGGABLE-MODULES.md) | 可插件化模块分析（12 个候选插件 + WASM 可行性评级） |
| [04-PLUGIN-MARKET-REVIEW.md](./04-PLUGIN-MARKET-REVIEW.md) | 插件市场设计评审（含安全审查 §7） |
| [05-EXTRACTION-ROADMAP.md](./05-EXTRACTION-ROADMAP.md) | 插件化实施路线图（Phase 0-4） |
| [06-COMPLETE-IMPLEMENTATION-PLAN.md](./06-COMPLETE-IMPLEMENTATION-PLAN.md) | **完整实施方案（可执行蓝图）** |
| [07-IMPLEMENTATION-CHECKLIST.md](./07-IMPLEMENTATION-CHECKLIST.md) | **实施清单（逐项打勾跟踪表）** |

## 5. 请求链路中的分工

```
客户端请求
    │
    ▼
┌──────────────────────┐
│  核心: 认证鉴权        │  API Key / JWT 校验
│  核心: 计费检查        │  余额 / 配额
│  核心: 并发控制        │  槽位分配
│  核心: 调度选号        │  账号 + 平台选择
└──────────┬───────────┘
           │
           ▼  进入 DispatchRuntime
┌──────────────────────┐
│  插件: Interceptor    │  审计 / 过滤 / 路由
│  插件: TransformReq   │  协议转换 / 模型映射
│  插件: Provider       │  转发到上游 API
│  插件: TransformResp  │  格式统一 / 错误映射
└──────────┬───────────┘
           │
           ▼  退出 DispatchRuntime
┌──────────────────────┐
│  核心: 用量记录        │  记录 + 扣费
└──────────┬───────────┘
           │
           ▼
      返回响应
```

## 6. 数量统计

| 归属 | 模块数 | 一句话总结 |
|------|--------|-----------|
| 主项目核心 | ~75 | 管钱、管人、管锁、管调度 |
| 插件基础设施 | ~35 | 运行时沙箱 + 生命周期管理 |
| 可插件化模块 | 12 个候选 | 4 Provider + 4 Transform + 3 Interceptor + 1 内置可选 |

## 7. 相关文档

- [ADR-001: 混合架构](../ADR-001-hybrid-architecture.md) — 控制平面 / 数据平面分离决策
- [ARCHITECTURE.md](../ARCHITECTURE.md) — 插件市场整体架构
