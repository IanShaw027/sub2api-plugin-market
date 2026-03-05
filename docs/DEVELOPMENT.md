# Sub2API Plugin Market - 开发指南

本文档说明本项目的本地开发流程、约定和验证方式。

## 技术栈

- Go `1.25+`
- Gin
- Ent
- PostgreSQL 15+
- MinIO（S3 兼容）
- GitHub Actions（CI）

## 本地开发启动

```bash
cp .env.example .env
docker-compose up -d
go run scripts/init_admin.go
go run cmd/server/main.go
```

关键端点：

- `GET /health`
- `GET /api/v1/plugins`
- `POST /admin/api/auth/login`

## 目录结构（核心）

```text
cmd/server/                 # 服务入口
internal/api/v1/            # 公开 API
internal/admin/             # 管理后台 API/中间件
internal/service/           # 业务逻辑
internal/repository/        # 数据访问
ent/schema/                 # Ent schema
openapi/plugin-market-v1.yaml
docs/
```

## 环境变量约定

项目服务端当前读取以下变量：

- `PORT`
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`
- `STORAGE_*`（由 `sub2api-storage` 读取）
- `ADMIN_JWT_SECRET`
- `CORS_ALLOWED_ORIGINS`（可选）

建议直接从 `.env.example` 复制并最小化改动。

## 开发命令

```bash
make run
make build
make check-contract
make test
make test-coverage
make migrate-up
make migrate-down
make lint
make fmt
```

## 测试策略

1. 修改 API 入参/出参时，先更新 `openapi/plugin-market-v1.yaml` 与 `docs/API.md`。
2. 提交前至少执行 `make check-contract` 与 `go test ./...`。
3. 涉及核心链路时，补充 `tests/integration` 回归用例。

## 开发注意事项

1. `ADMIN_JWT_SECRET` 为空时服务会直接退出。
2. 本地 Docker 默认映射 PostgreSQL 到主机 `5433`，与 `.env.example` 保持一致。
3. 管理后台静态资源入口为 `web/admin/index.html` 与 `web/admin/login.html`。
