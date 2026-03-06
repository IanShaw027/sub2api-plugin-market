# 错误码注册表（v1）

- 状态：Active
- 更新时间：2026-03-06
- 适用范围：`sub2api-plugin-market` API

## 1. 注册规则

1. 错误码全局唯一，不允许同码不同义。
2. 文档与代码必须同时更新。
3. 新增错误码必须附带：触发条件、调用方处理建议、可观测字段。

## 2. 当前错误码

| 代码 | 名称 | 含义 | 来源 |
|---|---|---|---|
| `0` | success | 成功 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:19) |
| `1001` | invalid_param | 参数错误 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:43) |
| `1002` | not_found | 资源不存在 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:44) |
| `1003` | internal_error | 服务器内部错误 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:45) |
| `1004` | database_error | 数据库错误 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:46) |
| `1005` | storage_error | 存储服务错误 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:47) |
| `1006` | forbidden | 权限不足或安全配置缺失 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:48) |

## 3. 调用方建议

1. `1001`：提示用户检查输入。
2. `1002`：展示“资源不存在”并允许重试查询。
3. `1003`：记录 `request_id`（若有）并重试或降级。
4. `1004`：短暂重试 + 告警数据库健康。
5. `1005`：降级下载链路并告警存储服务。
6. `1006`：检查安全配置（如 `GITHUB_WEBHOOK_SECRET`），或确认操作权限。

## 4. 端点适用场景

### 4.1 公开 API（`internal/api/v1/handler/response.go` 体系）

#### POST `/api/v1/submissions`

- 成功：`code=0`
- 参数错误：`code=1001`
  - JSON 结构错误或缺少必填字段（如 `plugin_name`、`source_type`）
  - `plugin_name` 格式非法（不匹配 `^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$`）
  - `source_type` 非 `upload/github`
  - `source_type=github` 但 `github_repo_url` 为空
  - `submission_type` 非法
- 速率限制：`code=1001`，HTTP 429
  - 同一 IP 超过速率限制（默认 10 次/分钟，由 `SUBMISSION_RATE_LIMIT` 环境变量配置）
  - 响应头包含 `Retry-After`
- 数据库错误：`code=1004`
  - 插件或提交记录落库失败

#### POST `/api/v1/integrations/github/webhook`

- 成功或忽略：`code=0`
  - 非 `release` 事件，或 `action` 非 `published`，返回 `message=ignored`
  - `release published` 且匹配插件后触发自动同步成功，返回 `message=success`
- 参数错误：`code=1001`
  - 缺少 `X-GitHub-Event`
  - payload 解析失败，或缺少 `repository.html_url` / `release.tag_name`
  - `GITHUB_WEBHOOK_SECRET` 非空时，`X-Hub-Signature-256` 缺失/格式错误/签名不匹配
- 安全配置缺失：`code=1006`，HTTP 403
  - `GIN_MODE=release` 且 `GITHUB_WEBHOOK_SECRET` 未配置时，拒绝处理所有 webhook
- 内部错误：`code=1003`
  - 已匹配插件但自动同步执行失败（对应 sync_job 会写入 `failed`）
- 数据库错误：`code=1004`
  - 根据仓库地址查找插件失败

说明：该端点沿用公开 API 错误码常量（`1001/1003/1004`），未引入新编号。

### 4.2 管理后台 API（非 `response.go` 常量体系）

#### POST `/admin/api/plugins/{id}/sync`

- 成功：`code=0`
- 请求参数错误：`code=400`
  - 路径参数 `id` 非法（UUID 解析失败）
  - 请求体 JSON 解析失败（`target_ref` 为可选）
- 创建同步任务失败：`code=500`

#### GET `/admin/api/sync-jobs`

- 成功：`code=0`
- 请求参数错误：`code=400`
  - `status` 非 `pending/running/succeeded/failed/cancelled`
  - `trigger_type` 非 `manual/auto`
  - `plugin_id` 非 UUID
  - `from` / `to` 非 RFC3339 时间
- 查询失败：`code=500`

#### GET `/admin/api/sync-jobs/{id}`

- 成功：`code=0`
- 同步任务不存在或 ID 非法：`code=404`

说明：管理后台同步接口使用统一信封格式 `{ code, message, data }`。同步相关接口使用 `0/400/404/500` HTTP 风格业务码，审核相关接口已统一使用 `1001~1006` 常量体系。

#### PUT `/admin/api/submissions/{id}/review`

- 成功：`code=0`
- 权限不足：`code=1006`，HTTP 403
  - `reviewer` 角色审核 `is_official=true` 插件时被拦截
- 已被处理：`code=1003`
  - Submission 不再是 `pending` 状态（乐观锁保护）

## 5. 与契约同步

1. OpenAPI：`openapi/plugin-market-v1.yaml`
2. API 文档：`docs/API.md`
3. 代码常量：`internal/api/v1/handler/response.go`
