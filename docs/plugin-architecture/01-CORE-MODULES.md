# 主项目核心模块清单

> **文档状态**: Draft  
> **创建日期**: 2026-03-06  
> **适用仓库**: sub2api (数据平面)  
> **路径基准**: 所有路径均相对于 `backend/internal/`，`ent/` 相对于 `backend/`

## 判断标准

一个模块属于核心，当且仅当满足以下任一条件：

1. **安全边界** — 认证、鉴权、令牌管理，不可被第三方 WASM 沙箱篡改
2. **资金安全** — 计费、扣费、余额，不可被插件绕过
3. **全局协调** — 并发控制、速率限制、调度，需要跨请求的原子性保证
4. **基础设施** — 数据库、缓存、HTTP Server，是所有模块的运行基座

> **范围说明**: 本文档不包含 `pluginmarket/`、`pluginruntime/`、`pluginsign/` 等插件相关模块，详见 [02-PLUGIN-INFRASTRUCTURE.md](./02-PLUGIN-INFRASTRUCTURE.md)。

---

## 1. 认证鉴权

| 模块 | 代码路径 | 职责 | 不可外移理由 |
|------|---------|------|-------------|
| JWT 认证 | `server/middleware/jwt_auth.go` | Web UI 用户令牌校验 | 安全边界，令牌校验不能被篡改 |
| API Key 认证 | `server/middleware/api_key_auth.go` | Gateway 请求入口校验 | 每次请求的守门员，安全边界 |
| API Key 认证 (Google) | `server/middleware/api_key_auth_google.go` | Gemini 风格 Key 校验 | 同上 |
| Admin 认证 | `server/middleware/admin_auth.go` | 管理后台 JWT 校验 | 后台权限控制 |
| Admin Only | `server/middleware/admin_only.go` | 仅管理员访问 | 权限控制 |
| Auth Subject | `server/middleware/auth_subject.go` | 提取认证主体 | 认证基础设施 |
| TOTP 2FA | `service/totp_service.go` | 二次验证 | 安全敏感 |
| Auth 服务 | `service/auth_service.go` | 注册、登录、验证码统一入口 | 认证流程聚合 |
| OAuth 服务 | `service/oauth_service.go` | 统一凭证刷新 | 涉及账号密钥 |
| LinuxDo OAuth | `handler/auth_linuxdo_oauth.go` | SSO 登录 | 认证通道 |
| Turnstile | `service/turnstile_service.go` | 人机验证 | 安全基础设施 |

## 2. 计费与用量

| 模块 | 代码路径 | 职责 | 不可外移理由 |
|------|---------|------|-------------|
| 计费服务 | `service/billing_service.go` | 扣费、余额计算、成本核算 | 资金安全 |
| 计费缓存 | `service/billing_cache_service.go` | 用户余额/订阅快照 | 余额一致性 |
| 定价服务 | `service/pricing_service.go` | 模型价格表（含 LiteLLM 远端） | 计费基础依赖 |
| 用量记录 | `service/usage_service.go` | 用量日志创建与查询 | 审计和计费依据 |
| 用量清理 | `service/usage_cleanup_service.go` | 历史用量归档与清理 | 数据生命周期 |

## 3. 并发控制

| 模块 | 代码路径 | 职责 | 不可外移理由 |
|------|---------|------|-------------|
| 并发服务 | `service/concurrency_service.go` | 全局并发槽位管理 | 全局资源调度，原子性保证 |
| 并发缓存 | `repository/concurrency_cache.go` | Redis 原子操作 | 和并发控制紧耦合 |
| 等待队列 | `handler/gateway_handler.go` 内 | 请求排队等待 | 和并发控制紧耦合 |

## 4. 速率限制

| 模块 | 代码路径 | 职责 | 不可外移理由 |
|------|---------|------|-------------|
| 限流中间件 | `middleware/rate_limiter.go` | IP + Key + 窗口限流 | Redis Lua 原子脚本 |
| 限流服务 | `service/ratelimit_service.go` | 模型级速率限制 | 全局生效 |

## 5. 调度与账号管理

| 模块 | 代码路径 | 职责 | 不可外移理由 |
|------|---------|------|-------------|
| 账号服务 | `service/account_service.go` | 账号 CRUD + 调度逻辑 | 凭证管理，安全敏感 |
| 账号过期 | `service/account_expiry_service.go` | 标记过期账号不可调度 | 自动维护 |
| 账号用量 | `service/account_usage_service.go` | 每账号用量统计 | 调度决策依赖 |
| 调度快照 | `service/scheduler_snapshot_service.go` | 账号负载均衡全局状态 | 跨请求全局状态 |
| 调度缓存 | `repository/scheduler_cache.go` | Redis 原子调度操作 | 原子性保证 |
| 调度事件 | `service/scheduler_events.go` | 调度状态变更事件 | 跨实例一致性 |
| 调度 Outbox | `service/scheduler_outbox.go` | 事件持久化 | 可靠投递 |
| 临时不可调度 | `service/temp_unsched.go` | 临时熔断标记 | 熔断联动 |
| Token 刷新 | `service/token_refresh_service.go` | 后台 OAuth 刷新 | 凭证管理 |
| Token 缓存失效 | `service/token_cache_invalidator.go` | 缓存一致性 | 调度依赖 |
| Claude Token | `service/claude_token_provider.go` | Claude OAuth Token | 凭证管理 |
| OpenAI Token | `service/openai_token_provider.go` | OpenAI OAuth Token | 凭证管理 |
| Gemini Token | `service/gemini_token_provider.go` | Gemini OAuth Token | 凭证管理 |
| Antigravity Token | `service/antigravity_token_provider.go` | Antigravity OAuth Token | 凭证管理 |
| Token Refresher | `service/token_refresher.go` 及 `*_token_refresher.go` | 后台定时刷新 | 凭证管理 |

## 6. 用户与订阅

| 模块 | 代码路径 | 职责 | 不可外移理由 |
|------|---------|------|-------------|
| 用户服务 | `service/user_service.go` | 用户 CRUD | 核心实体 |
| 用户属性 | `service/user_attribute_service.go` | 用户扩展属性 | 核心实体 |
| 订阅管理 | `service/subscription_service.go` | 套餐绑定、续期 | 和计费强关联 |
| 订阅过期 | `service/subscription_expiry_service.go` | 到期自动处理 | 自动维护 |
| API Key 服务 | `service/api_key_service.go` | Key CRUD + 校验 | 核心鉴权 |
| Auth 缓存 | `service/api_key_auth_cache_impl.go` | L1 (Ristretto) + L2 (Redis) | 高频路径缓存 |
| 分组服务 | `service/group_service.go` | 路由分组管理 | 调度依赖 |
| 身份服务 | `service/identity_service.go` | 身份解析 | 认证链路 |

## 7. 运维监控 (Ops)

| 模块 | 代码路径 | 职责 |
|------|---------|------|
| Ops 指标采集 | `service/ops_metrics_collector.go` | 请求/错误/延迟指标 |
| Ops 聚合 | `service/ops_aggregation_service.go` | 定时聚合统计 |
| Ops 仪表盘 | `service/ops_dashboard.go` | 运维面板数据 |
| Ops 告警 | `service/ops_alert_evaluator_service.go` | 告警规则评估 |
| Ops 实时流量 | `service/ops_realtime.go`, `ops_realtime_traffic.go` | 实时监控 |
| Ops 清理 | `service/ops_cleanup_service.go` | 历史数据归档 |
| Ops 报表 | `service/ops_scheduled_report_service.go` | 定时报告 |
| Ops 其他 | `service/ops_*.go` (模型、类型、配置) | 辅助类型 |

## 8. 请求转发核心

Gateway 是计费、调度、限流、并发控制的交汇点，虽然 Provider 逻辑可插件化（见 03-PLUGGABLE-MODULES），但 Gateway Handler 本身是核心调度入口。

| 模块 | 代码路径 | 职责 | 不可外移理由 |
|------|---------|------|-------------|
| Gateway Handler | `handler/gateway_handler.go` | 主请求入口，串联认证→计费→并发→调度→转发 | 核心调度编排 |
| Gateway Helper | `handler/gateway_helper.go` | 辅助函数 | 和 Handler 紧耦合 |
| Gateway Interceptor | `handler/gateway_interceptor.go` | 请求/响应拦截 | 核心链路 |
| Gemini v1beta Handler | `handler/gemini_v1beta_handler.go` | Gemini 原生 API 入口 | 核心路由 |
| OpenAI Handler | `handler/openai_gateway_handler.go` | OpenAI API 入口 | 核心路由 |
| Plugin 控制面 Handler | `handler/plugin_control_plane_handler.go` | 插件安装/卸载/生命周期 | 核心管理 |

## 9. 管理后台

| 模块 | 代码路径 | 职责 |
|------|---------|------|
| Admin 服务 | `service/admin_service.go` | 统一管理入口 |
| 账号管理 | `handler/admin/account_handler.go` | 账号 CRUD |
| 用户管理 | `handler/admin/user_handler.go` | 用户管理 |
| 用户属性管理 | `handler/admin/user_attribute_handler.go` | 用户扩展属性 |
| 分组管理 | `handler/admin/group_handler.go` | 分组管理 |
| 订阅管理 | `handler/admin/subscription_handler.go` | 套餐管理 |
| 代理管理 | `handler/admin/proxy_handler.go` | 代理管理 |
| 设置管理 | `handler/admin/setting_handler.go` | 系统设置 |
| 公告管理 | `handler/admin/announcement_handler.go` | 公告管理 |
| 用量管理 | `handler/admin/usage_handler.go` | 用量统计 |
| Ops 管理 | `handler/admin/ops_*.go` (6+ 文件) | 运维面板 |
| OAuth 管理 | `handler/admin/gemini_oauth_handler.go`, `openai_oauth_handler.go`, `antigravity_oauth_handler.go` | OAuth 配置 |
| 促销管理 | `handler/admin/promo_handler.go` | 促销码 |
| 兑换管理 | `handler/admin/redeem_handler.go` | 兑换码 |
| 仪表盘 | `handler/admin/dashboard_handler.go` | 统计面板 |
| 系统 | `handler/admin/system_handler.go` | 系统信息 |
| 错误穿透 | `handler/admin/error_passthrough_handler.go` | 上游错误规则 |

## 10. 其他基础服务

| 模块 | 代码路径 | 职责 | 不可外移理由 |
|------|---------|------|-------------|
| 配置 | `config/config.go` | Viper 全局配置 | 基础设施 |
| 设置服务 | `service/setting_service.go` | 运行时 Feature Flag | 全局设置 |
| 邮件服务 | `service/email_service.go` | 验证码/密码重置邮件 | 通知基础设施 |
| 邮件队列 | `service/email_queue_service.go` | 异步发送 | 通知基础设施 |
| 公告服务 | `service/announcement_service.go` | 用户通知 | 运营功能 |
| 促销服务 | `service/promo_service.go` | 促销码兑换 | 涉及余额变更 |
| 兑换服务 | `service/redeem_service.go` | 兑换码 | 涉及余额变更 |
| 错误穿透 | `service/error_passthrough_service.go` | 上游错误处理规则 | 全局策略 |
| 仪表盘 | `service/dashboard_service.go` | 统计数据 | 聚合服务 |
| 仪表盘聚合 | `service/dashboard_aggregation_service.go` | 定时聚合 | 性能优化 |
| CRS 同步 | `service/crs_sync_service.go` | 外部账号同步 | 账号管理 |
| 时间轮 | `service/timing_wheel_service.go` | 定时任务调度 | 基础设施 |
| 延迟初始化 | `service/deferred_service.go` | 懒加载服务 | 基础设施 |
| 代理服务 | `service/proxy_service.go` | 代理管理 | 网络基础设施 |

## 11. 基础设施层

| 模块 | 代码路径 | 职责 |
|------|---------|------|
| HTTP Server | `server/http.go`, `server/router.go` | Gin 服务器 |
| 路由 | `server/routes/*.go` | 路由注册 |
| 中间件 | `server/middleware/*.go` | 通用中间件 |
| 数据库/ORM | `ent/`, `repository/*.go` | PostgreSQL + Ent |
| Redis | `repository/redis.go` | 缓存层 |
| HTTP 客户端 | `repository/http_upstream.go`, `pkg/httpclient/` | 上游通信 |
| 存储 | `storage/` | 本地 / MinIO |
| AES 加密 | `repository/aes_encryptor.go` | 敏感数据加密 |
| 迁移 | `repository/migrations_runner.go` | 数据库迁移 |
| DI | `service/wire.go`, `handler/wire.go` | Wire 依赖注入 |
| Setup | `setup/*.go` | 首次部署向导 |
| 前端 | `web/`, `frontend/` | 嵌入式 Vue SPA |
