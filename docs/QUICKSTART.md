# Sub2API Plugin Market - 快速开始

本文档用于在本地快速拉起服务并验证核心接口。

## 前置要求

- Go `1.25+`
- Docker + Docker Compose
- `psql`（可选，用于排查数据库连接）

## 5 分钟启动

```bash
# 1) 准备配置
cp .env.example .env

# 2) 启动依赖（PostgreSQL + MinIO）
docker-compose up -d

# 3) 初始化管理员账号
go run scripts/init_admin.go

# 4) 启动服务
go run cmd/server/main.go
```

启动后访问：

- API: `http://localhost:8081/api/v1`
- 管理后台: `http://localhost:8081/admin/login`
- 健康检查: `http://localhost:8081/health`

## 默认开发配置说明

项目默认使用以下关键配置（来自 `.env.example`）：

```env
PORT=8081
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=plugin_market
DB_SSLMODE=disable
STORAGE_TYPE=minio
STORAGE_MINIO_ENDPOINT=localhost:9000
STORAGE_MINIO_BUCKET=plugin-market
ADMIN_JWT_SECRET=YOUR_TOKEN_HERE
```

注意：`ADMIN_JWT_SECRET` 为必填项，未配置时服务会拒绝启动。

## 快速验证

```bash
# 健康检查
curl http://localhost:8081/health

# 插件列表
curl http://localhost:8081/api/v1/plugins

# 管理员登录
curl -X POST http://localhost:8081/admin/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

## 常用 Make 命令

```bash
make run           # 启动服务
make test          # 契约检查 + 测试
make check-contract
make migrate-up    # 迁移（兼容别名）
make migrate-down
make lint
make fmt
```

## 常见问题

1. 数据库连接失败  
   检查 `docker-compose ps`，确认 PostgreSQL 已启动，且 `.env` 中 `DB_PORT=5433` 与 Compose 映射一致。

2. 服务启动时报 `missing ADMIN_JWT_SECRET`  
   检查 `.env` 是否存在且包含 `ADMIN_JWT_SECRET`。

3. 端口冲突  
   修改 `.env` 中 `PORT`，例如 `PORT=8082`。
