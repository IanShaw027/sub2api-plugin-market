# 插件市场设计评审

> **文档状态**: Draft  
> **创建日期**: 2026-03-06  
> **适用仓库**: sub2api-plugin-market (控制平面)

## 1. 评审范围

本文档对 `sub2api-plugin-market` 的架构设计进行全面评审，覆盖：

- 架构分层
- 数据模型
- API 设计
- 审核流程
- 信任链
- 与主项目集成

---

## 2. 架构评估

### 2.1 控制平面 / 数据平面分离 ✅

按 ADR-001 的决策，职责划分清晰：

| | 控制平面 (plugin-market) | 数据平面 (sub2api) |
|---|---|---|
| 元数据管理 | ✅ | ❌ |
| 版本分发 | ✅ (预签名 URL) | ❌ |
| 提交审核 | ✅ | ❌ |
| 信任链分发 | ✅ | ❌ |
| 安装状态/Lockfile | ❌ | ✅ |
| WASM 运行时 | ❌ | ✅ |
| 预装验签 | ❌ | ✅ (复用 pluginsign) |

**评价**: 分离合理，契约清晰。

### 2.2 分层架构 ✅

```
Handler (HTTP 入站)
    │  参数解析、统一响应
    ▼
Service (业务逻辑)
    │  聚合仓储、存储、验签
    ▼
Repository (数据访问)
    │  Ent 查询、筛选、分页
    ▼
Ent Schema (领域模型)
```

**评价**: 三层分明，职责单一，符合标准实践。

### 2.3 安全链 ✅

```
插件开发者 → Ed25519 签名 → 上传 WASM + manifest
                                    │
                                    ▼
                        Plugin Market 存储签名信息
                                    │
                                    ▼
                sub2api 下载时验签 (pluginsign.VerifyInstall)
                    ├── 哈希比对 (SHA-256)
                    ├── 签名校验 (Ed25519)
                    ├── Trust Store 校验
                    └── API 版本兼容性校验
```

**评价**: 先验签后分发，安全链完整。

---

## 3. 做对了的部分

| 维度 | 评价 | 详情 |
|------|------|------|
| **分层清晰** | ✅ | Handler → Service → Repository 三层分明 |
| **信任链完整** | ✅ | Ed25519 签名 + TrustStore，先验签后分发 |
| **审核流** | ✅ | pending → approved/rejected 状态机简单可控 |
| **GitHub 集成** | ✅ | Webhook + SyncJob 支持自动同步 Release |
| **契约一致性** | ✅ | OpenAPI + 错误码注册表 + CI 校验 |
| **下载审计** | ✅ | DownloadLog 记录 IP 哈希、UA、成功/失败 |
| **版本管理** | ✅ | draft → published → yanked 状态完整 |
| **管理后台** | ✅ | JWT 保护 + 角色分级 (super_admin/admin/reviewer) |

---

## 4. 需要改进的地方

### 4.1 🔴 P0 — 必须修复

#### 4.1.1 缺少 `plugin_type` 字段

**现状**: Plugin Schema 中没有 `plugin_type` 字段。

**影响**: 
- 市场无法按插件类型（Interceptor / Transform / Provider）筛选
- 客户端不知道插件属于哪个执行阶段
- 无法做类型级别的兼容性校验

**建议**: 在 `ent/schema/plugin.go` 中增加：

```go
field.Enum("plugin_type").
    Values("interceptor", "transform", "provider").
    Optional().
    Comment("插件类型，对应 DispatchRuntime 的三个阶段"),
```

同步更新：
- OpenAPI spec 的 Plugin schema
- 市场列表 API 增加 `?type=provider` 筛选
- 管理后台 UI

---

#### 4.1.2 版本兼容性查询不足

**现状**: `GET /api/v1/plugins/:name/versions` 返回所有版本，客户端需要自己判断兼容性。

**影响**: 客户端无法高效查找可安装的版本。

**建议**: 增加查询参数：

```
GET /api/v1/plugins/:name/versions?compatible_with=1.2.0
```

服务端按 `min_api_version <= compatible_with <= max_api_version` 过滤。

---

#### 4.1.3 审核通过后未自动发布版本

**现状**: Submission approved 后，关联的 PluginVersion 状态不会自动变为 `published`。

**影响**: 审核通过后还需要手动操作才能让版本可下载。

**建议**: 在 `admin/service/submission_service.go` 的 `ReviewSubmission` 中，审批通过时联动：

```
if action == "approved" && submission.version != nil {
    submission.version.Update().SetStatus("published").SetPublishedAt(now)
}
```

---

### 4.2 🟡 P1 — 重要改进

#### 4.2.1 SyncJob 增强

**现状**: `sync_service.go` 已实现完整的 GitHub Release 同步流程（下载 WASM → 计算 SHA256 → 上传 Storage → 创建 PluginVersion）。

**可改进项**:
1. Manifest 解析和校验（当前仅创建版本，未校验 manifest）
2. Ed25519 签名校验（安装前验签链路）
3. GitHub API 限流保护（重试 + 退避）
4. 同步历史版本（当前仅同步最新 Release）

---

#### 4.2.2 缺少依赖解析能力

**现状**: PluginVersion 有 `dependencies` JSON 字段，但市场没有解析和校验能力。

**影响**: 安装有依赖冲突的插件不会被阻止。

**建议**: 
- 市场端：在审核时校验依赖是否可解析
- 客户端（sub2api）：安装时做完整的依赖解析（已有 `dependency_resolver.go`）

---

#### ~~4.2.3 错误码风格不统一~~ ✅ 已修复

**现状**: Admin API 已统一使用 `{ code, message, data }` 信封格式和业务码（1001-1005、2001-2002），与 Public API 风格一致。

**遗留项**: `docs/ERROR-CODE-REGISTRY.md` 中关于"管理后台同步接口返回 0/400/404/500"的描述已与实际代码不符，需同步更新文档。

---

### 4.3 🟢 P2 — 增强项

#### 4.3.1 插件能力声明

**建议**: 在 Plugin/PluginVersion metadata 中增加 `capabilities` 字段：

```json
{
  "capabilities": ["host_http_fetch", "host_kv"]
}
```

市场展示时告知用户该插件需要的权限。

---

#### 4.3.2 插件配置模板

**建议**: 每个插件可定义自己的配置项 Schema：

```json
{
  "config_schema": {
    "type": "object",
    "properties": {
      "api_base_url": { "type": "string", "default": "https://api.anthropic.com" },
      "max_retries": { "type": "integer", "default": 3 }
    }
  }
}
```

sub2api 安装时用此 Schema 生成配置表单。

---

#### 4.3.3 插件评分与评论

**现状**: Plugin Schema 有 `rating` 字段但没有评分提交 API。

**建议**: 可在 Phase 4（生态阶段）实现用户评分和评论功能。

---

## 5. 数据模型评审

### 5.1 现有模型

| 实体 | 评价 | 备注 |
|------|------|------|
| Plugin | ✅ | 主表设计合理，包含分类、官方标记、下载量 |
| PluginVersion | ✅ | 含 WASM 签名全套字段 |
| Submission | ✅ | 审核流状态机清晰 |
| TrustKey | ✅ | 支持多级信任（official/verified_publisher/community） |
| DownloadLog | ✅ | IP 哈希保护隐私 |
| AdminUser | ✅ | 角色分级合理 |
| SyncJob | ✅ | 设计合理，已实现完整 GitHub Release 同步流程 |

### 5.2 建议新增字段

| 实体 | 字段 | 类型 | 说明 |
|------|------|------|------|
| Plugin | `plugin_type` | Enum | interceptor/transform/provider |
| PluginVersion | `capabilities` | JSON | 需要的 Host API 列表 |
| PluginVersion | `config_schema` | JSON | 配置项 Schema |

---

## 6. API 设计评审

### 6.1 Public API

| 端点 | 评价 | 改进建议 |
|------|------|---------|
| `GET /plugins` | ✅ | 增加 `?type=` 筛选 |
| `GET /plugins/:name` | ✅ | — |
| `GET /plugins/:name/versions` | 🟡 | 增加 `?compatible_with=` |
| `GET /trust-keys/:key_id` | ✅ | — |
| `GET /plugins/:name/versions/:version/download` | ✅ | 302 + 预签名 URL 设计好 |
| `GET /trust-keys` | ✅ | — |
| `POST /submissions` | 🔴 | **无认证**，需增加 rate limit 或 Token 校验（见 7.1） |
| `POST /integrations/github/webhook` | 🔴 | secret 为空时跳过 HMAC 校验（见 7.1） |

### 6.2 Admin API

| 端点 | 评价 | 改进建议 |
|------|------|---------|
| `GET /submissions` | ✅ | — |
| `PUT /submissions/:id/review` | 🟡 | 审批后联动版本发布 |
| `POST /plugins/:id/sync` | ✅ | 已实现，可增强（见 4.2.1） |
| `GET /sync-jobs` | ✅ | — |

---

## 7. 安全与稳定性深度审查

### 7.1 🔴 安全漏洞

| 问题 | 严重度 | 代码位置 | 建议 |
|------|--------|----------|------|
| **POST /submissions 无认证** | P0 | `api/v1/router.go`、`service/submission_service.go` | 任何人可提交插件，易被滥用（垃圾提交、恶意占名）。增加 rate limit 或 Token 校验 |
| **GITHUB_WEBHOOK_SECRET 为空时跳过签名校验** | P0 | `api/v1/handler/github_webhook_handler.go` | `if h.secret != ""` 才校验。生产环境未配置时应拒绝处理 webhook |
| **插件名未校验路径遍历** | P1 | `ent/schema/plugin.go`、`service/sync_service.go` | `name` 仅 `NotEmpty()`，未限制 `/`、`..`。Sync 使用 `fmt.Sprintf("plugins/%s/%s/plugin.wasm", name, ref)`，恶意 name 可导致存储路径越界。加正则 `^[a-z0-9][a-z0-9-]*$` |
| **管理员审核无角色分级** | P1 | `admin/router.go` | 审核路由仅用 `AdminAuth`，未使用 `RequireRole`。reviewer 可审核 official 插件。对 `is_official=true` 仅允许 super_admin/admin 审核 |

### 7.2 🟡 数据完整性

| 问题 | 严重度 | 代码位置 | 建议 |
|------|--------|----------|------|
| **Submission 审核与 Plugin 更新非原子** | P1 | `admin/service/submission_service.go` | `ReviewSubmission` 先更新 Submission 再更新 Plugin，无事务。Plugin 更新失败时状态不一致。用 `client.Tx()` 包裹 |
| **Submission 与 Version 无关联** | P1 | `service/submission_service.go` | `CreateSubmission` 只创建 Plugin + Submission，不创建 PluginVersion。审核通过只更新元数据，不发布版本。需明确审核→版本发布流程 |
| **SyncJob 失败产生孤儿 WASM** | P1 | `service/sync_service.go` | 执行顺序：上传 WASM → 检查版本 → 创建版本。Create 失败时 WASM 已写入存储。建议先检查再上传，或失败时清理 |
| **Sync 创建 draft 无签名** | P1 | `service/sync_service.go` | Sync 创建 `status=draft`、`signature=""`、`sign_key_id=""` 的版本。下载仅返回 `published`，且验签要求 `sign_key_id` 非空。当前 GitHub 同步的版本无法被下载。需实现签名→发布流程 |

### 7.3 🟡 并发竞态

| 问题 | 严重度 | 代码位置 | 建议 |
|------|--------|----------|------|
| **手动 Sync 与 Webhook Sync 并发** | P1 | `admin/handler/sync_handler.go`、`api/v1/handler/github_webhook_handler.go` | 手动同步执行 + Webhook 异步执行可能同时跑 `runGitHubSync`。`versionAvailable` 与 `Create` 之间无锁，可产生竞态和孤儿 WASM。对 `(plugin_id, target_ref)` 加分布式锁 |
| **双管理员同时审核** | P2 | `admin/service/submission_service.go` | 无乐观锁，后者覆盖前者。审核前检查 `status == pending`，用条件更新 |
| **同一插件多笔 pending Submission** | P2 | `service/submission_service.go` | 无限制，同一插件可多次提交。限制每插件同时 pending 数量 |

### 7.4 可扩展性

| 问题 | 严重度 | 建议 |
|------|--------|------|
| **列表 API 无缓存** | P2 | 1000+ 插件时直接查库压力大。对列表结果短期缓存（1-5 分钟） |
| **预签名 URL 无缓存** | P2 | 每次下载都生成新 URL。对相同 `(wasm_url, expiry)` 短时缓存 |
| **Trust Key 轮换影响旧版本** | P2 | Key 设为 `IsActive=false` 后，用旧 key 签名的版本无法通过验签。需支持多代 key 并存过渡期 |

---

## 8. 与主项目集成评审

### 8.1 集成点

| 集成方式 | 评价 |
|---------|------|
| REST API 通信 | ✅ 标准 HTTP，松耦合 |
| 预签名 URL 下载 | ✅ 避免 Market 成为流量瓶颈 |
| 共享 pluginsign 库 | ✅ 双端一致的签名验证 |
| 共享 sub2api-storage 库 | ✅ 存储抽象复用 |
| 错误码契约 | ✅ CI 校验 |
| OpenAPI 契约 | ✅ Spec 同步 |

### 8.2 缺失的集成

| 缺失项 | 影响 | 建议 |
|--------|------|------|
| 插件类型感知 | sub2api 不知道插件属于哪个阶段 | 加 `plugin_type` 字段 |
| 兼容性查询 API | sub2api 需要自己做兼容性过滤 | 增加服务端过滤 |
| 插件配置分发 | 市场没有配置模板能力 | P2 阶段实现 |

---

> **下一步**: 本文档的改进建议已纳入实施计划，详见 [05-EXTRACTION-ROADMAP.md](./05-EXTRACTION-ROADMAP.md)。
