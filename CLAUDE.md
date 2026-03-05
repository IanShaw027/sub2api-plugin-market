# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概览

- 这是一个 **Sub2API 插件市场控制平面** 服务，职责是插件元数据管理、版本分发、签名信任链分发、提交审核与管理后台。
- 按 `docs/ADR-001-hybrid-architecture.md`，本仓库是控制平面唯一实现；运行时安装与执行属于 `sub2api`（数据平面/执行平面）。
- 服务入口为 `cmd/server/main.go`，使用 Gin 提供两组路由：
  - 公开 API：`/api/v1/*`
  - 管理后台：`/admin/*`（含静态页面与 JWT 保护 API）

## 常用开发命令

### 环境与依赖

```bash
# 启动依赖（PostgreSQL + MinIO）
make docker-up

# 停止依赖
make docker-down

# 拉取并整理依赖
go mod download
go mod tidy
```

### 启动与构建

```bash
# 本地运行服务
make run
# 等价：go run cmd/server/main.go

# 构建二进制
make build
# 输出：bin/server
```

### 测试与契约校验

```bash
# 全量测试（包含 OpenAPI/错误码契约校验）
make test

# 仅运行契约校验
make check-contract

# 运行单个测试文件（示例）
go test -v ./internal/admin/handler -run TestSubmissionHandler

# 运行单个测试函数（示例）
go test -v ./internal/api/v1/handler -run TestSuccessResponse
```

### 代码质量与生成

```bash
# 代码检查
make lint

# 代码格式化
make fmt

# Ent 代码生成（修改 ent/schema 后必须执行）
make generate
```

### 管理后台开发辅助

```bash
# 初始化管理员账号
go run scripts/init_admin.go

# 管理后台接口快速验证
./scripts/test_admin.sh
```

## 运行配置与启动前提

- 默认端口：`8081`。
- 服务启动会在 `cmd/server/main.go` 中执行 `client.Schema.Create(...)`，因此数据库可达是启动前提。
- `ADMIN_JWT_SECRET` 必须设置；在 `GIN_MODE=release` 下禁止弱默认密钥（见 `cmd/server/main.go`）。
- 存储后端由 `sub2api-storage` 按环境变量初始化（MinIO/本地等）。

## 高层架构（跨文件理解）

### 1) 分层调用链

- `internal/api/v1/handler/*` 与 `internal/admin/handler/*`：HTTP 入站层，只做参数解析、调用 service、统一响应。
- `internal/service/*` 与 `internal/admin/service/*`：业务层，聚合仓储、存储、验签等能力。
- `internal/repository/*`：Ent 查询封装，处理筛选、排序、分页和状态约束。
- `ent/schema/*`：领域数据模型，`make generate` 后生成 `ent/*` 供 repository/service 使用。

典型下载链路：
1. `GET /api/v1/plugins/:name/versions/:version/download`
2. `DownloadHandler` 调用 `DownloadService.GetDownloadURL`
3. `DownloadService` 读取插件版本并下载工件字节
4. `VerificationService.VerifyPlugin` 使用 `sub2api-pluginsign` + TrustStore 做哈希/签名/兼容性校验
5. 校验成功后生成 MinIO 预签名 URL，记录 `download_logs`，返回 302

### 2) 路由组织

- 公开 API 路由注册在 `internal/api/v1/router.go`，并挂载 Recovery/Logger/CORS 中间件。
- 管理后台路由注册在 `internal/admin/router.go`：
  - `/admin/login` 等静态资源
  - `/admin/api/auth/login` 无 token
  - `/admin/api/*` 其余接口经 `internal/admin/middleware/auth.go` JWT 鉴权

### 3) 数据模型核心关系

关键实体（`ent/schema`）：
- `plugin`：插件主表（分类、官方标记、下载量、状态）
- `plugin_version`：版本与工件元信息（`wasm_url`、`wasm_hash`、`signature`、`sign_key_id`）
- `trust_key`：签名信任公钥
- `download_log`：下载审计日志（IP 哈希、UA、成功/失败）
- `submission`：审核流状态（pending/approved/rejected/cancelled）
- `admin_user`：后台管理员账号与角色

审核域在 `internal/admin/service/submission_service.go`，市场浏览/下载域在 `internal/service/*`。

### 4) API 与契约一致性

- OpenAPI 定义位于 `openapi/plugin-market-v1.yaml`。
- 错误码注册文档位于 `docs/ERROR-CODE-REGISTRY.md`。
- CI/本地通过 `make check-contract` 校验 OpenAPI 与错误码契约一致性，作为 `make test` 前置步骤。

## 开发时的关键约束

- 修改 `ent/schema/*` 后，必须执行 `make generate`，并确保生成代码与 schema 同步提交。
- 涉及 API 响应结构或错误码变更时，需同步更新：
  - `openapi/plugin-market-v1.yaml`
  - `docs/ERROR-CODE-REGISTRY.md`
  - 对应 handler/service 与测试
- 下载链路是“先验签后分发”，不要绕过 `VerificationService`。
- 若新增管理后台受保护接口，需挂到 `authorized` 路由组并复用现有 JWT 中间件。
