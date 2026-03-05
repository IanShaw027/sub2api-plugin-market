# Sub2API Plugin Market - 部署指南

本文档给出当前实现可用的部署方式与生产建议。

## 运行前检查

- Docker 20.10+
- Docker Compose 2.0+
- 可访问 PostgreSQL 与 MinIO
- 已设置强随机 `ADMIN_JWT_SECRET`

## Docker Compose（本地/测试环境）

```bash
cp .env.example .env
docker-compose up -d
go run scripts/init_admin.go
go run cmd/server/main.go
```

默认端口：

- API: `8081`
- PostgreSQL（主机）: `5433`
- MinIO: `9000`

## 生产环境建议

1. 使用反向代理（Nginx/Caddy）提供 HTTPS。
2. `ADMIN_JWT_SECRET` 使用高强度随机值，例如：

```bash
openssl rand -base64 32
```

3. 独立管理 PostgreSQL 与 MinIO，不建议直接使用开发配置。
4. 开启访问日志并接入监控告警。
5. 使用只读镜像标签和固定版本，不使用 `latest`。

## 最小环境变量清单

```env
PORT=8081
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=plugin_market
DB_PASSWORD=YOUR_TOKEN_HERE
DB_NAME=plugin_market
DB_SSLMODE=require

STORAGE_TYPE=minio
STORAGE_MINIO_ENDPOINT=minio.example.com:9000
STORAGE_MINIO_ACCESS_KEY=YOUR_API_KEY_HERE
STORAGE_MINIO_SECRET_KEY=YOUR_TOKEN_HERE
STORAGE_MINIO_BUCKET=plugin-market
STORAGE_MINIO_USE_SSL=true
STORAGE_MINIO_BASE_URL=https://minio.example.com

ADMIN_JWT_SECRET=YOUR_TOKEN_HERE
CORS_ALLOWED_ORIGINS=https://plugins.example.com
```

## 部署后验证

```bash
curl -f http://127.0.0.1:8081/health
curl -f http://127.0.0.1:8081/api/v1/plugins
```

如果需要验证管理接口：

```bash
curl -X POST http://127.0.0.1:8081/admin/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

## 故障排查

1. 启动失败且日志提示 `missing ADMIN_JWT_SECRET`：检查环境变量注入。
2. 数据库连接失败：核对 `DB_*`，尤其是端口和 `sslmode`。
3. 下载接口异常：检查 MinIO 访问权限和 `STORAGE_MINIO_*`。
