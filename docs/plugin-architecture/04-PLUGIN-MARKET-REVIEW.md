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

#### ~~4.1.1 缺少 `plugin_type` 字段~~ ✅ 已实现

**现状**: ~~Plugin Schema 中没有 `plugin_type` 字段。~~ 已在 `ent/schema/plugin.go` 中实现：`field.Enum("plugin_type").Values("interceptor","transform","provider").Optional()`。OpenAPI spec 也已包含 `type` 查询参数。

**遗留**: handler/repository 层的 `?type=` 筛选实现需确认。

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

#### ~~4.1.3 审核通过后未自动发布版本~~ ✅ 已实现

**现状**: ~~Submission approved 后，关联的 PluginVersion 状态不会自动变为 `published`。~~ 已在 `admin/service/submission_service.go` 的事务中实现：审批通过时将关联的 draft PluginVersion 更新为 published + 设置 published_at。

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
| ~~Plugin~~ | ~~`plugin_type`~~ | ~~Enum~~ | ✅ 已实现 |
| ~~PluginVersion~~ | ~~`capabilities`~~ | ~~JSON~~ | ✅ 已实现 |
| ~~PluginVersion~~ | ~~`config_schema`~~ | ~~JSON~~ | ✅ 已实现 |

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
| `POST /submissions` | ✅ | 已有 IP 级 rate limit |
| `POST /integrations/github/webhook` | ✅ | 生产环境下 secret 为空时拒绝处理 |

### 6.2 Admin API

| 端点 | 评价 | 改进建议 |
|------|------|---------|
| `GET /submissions` | ✅ | — |
| `PUT /submissions/:id/review` | ✅ | 审批后已联动版本发布 |
| `POST /plugins/:id/sync` | ✅ | 已实现，可增强（见 4.2.1） |
| `GET /sync-jobs` | ✅ | — |

---

## 7. 安全与稳定性深度审查

### 7.1 🔴 安全漏洞

| 问题 | 严重度 | 代码位置 | 建议 |
|------|--------|----------|------|
| ~~**POST /submissions 无认证**~~ | ~~P0~~ | `api/v1/router.go` | ✅ 已实现 IP 级 rate limit（`NewIPRateLimiter`） |
| ~~**GITHUB_WEBHOOK_SECRET 为空时跳过签名校验**~~ | ~~P0~~ | `api/v1/handler/github_webhook_handler.go` | ✅ 已实现（`gin.ReleaseMode` 下拒绝空 secret） |
| **插件名未校验路径遍历** | P1 | `service/submission_service.go` | ⚠️ Service 层已有正则 `^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$`，但 Schema 层无约束。admin 直接创建插件时可绕过。建议在 schema 层加 `Match()` |
| ~~**管理员审核无角色分级**~~ | ~~P1~~ | `admin/service/submission_service.go` | ✅ 已实现 `if pluginRecord.IsOfficial && reviewerRole == "reviewer" { return ErrForbiddenReview }` |

### 7.2 🟡 数据完整性

| 问题 | 严重度 | 代码位置 | 建议 |
|------|--------|----------|------|
| ~~**Submission 审核与 Plugin 更新非原子**~~ | ~~P1~~ | `admin/service/submission_service.go` | ✅ 已用 `client.Tx()` 事务包裹 |
| **Submission 与 Version 无关联（手动上传路径）** | P1 | `service/submission_service.go` | ⚠️ Schema 已有 `edge.To("version", PluginVersion.Type)`，但 `CreateSubmission` 不创建 PluginVersion 也不上传 WASM。需在 1.5/1.6 中实现完整 multipart 上传流程 |
| ~~**SyncJob 失败产生孤儿 WASM**~~ | ~~P1~~ | `service/sync_service.go` | ✅ 已在 Create 失败时调用 `storage.Delete` 清理；操作顺序已改为先检查 `versionAvailable` 再下载上传 |
| **Sync 创建 draft 无签名** | P1 | `service/sync_service.go` | ⚠️ 部分修复——配置 `PLUGIN_SIGNING_KEY_ID` 时可自动签名并发布为 published；未配置时仍创建 draft 无签名。仍需实现从 Release 下载 `manifest.json` + `signature.sig` 的完整验签流程 |

### 7.3 🟡 并发竞态

| 问题 | 严重度 | 代码位置 | 建议 |
|------|--------|----------|------|
| ~~**手动 Sync 与 Webhook Sync 并发**~~ | ~~P1~~ | `service/sync_service.go` | ✅ 已有 `sync.Map` + `acquireSyncLock(pluginID, targetRef)` 进程内锁。⚠️ 多实例部署时需升级为分布式锁 |
| ~~**双管理员同时审核**~~ | ~~P2~~ | `admin/service/submission_service.go` | ✅ 已有乐观锁 `Where(submission.StatusEQ(submission.StatusPending))` |
| ~~**同一插件多笔 pending Submission**~~ | ~~P2~~ | `service/submission_service.go` | ✅ 已限制 `pendingCount >= 3` 时拒绝 |

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
