# Sub2API Plugin Market - 快速开始

本指南将帮助你在 5 分钟内启动 Sub2API Plugin Market 并测试核心功能。

---

## 前置要求

确保你的系统已安装：

- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- **curl** 或 **Postman**（用于测试 API）

---

## 快速启动

### 1. 克隆项目

```bash
git clone https://github.com/your-org/sub2api-plugin-market.git
cd sub2api-plugin-market
```

### 2. 启动服务

```bash
# 使用默认配置启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps
```

预期输出：

```
NAME                COMMAND                  SERVICE    STATUS
plugin-market-app   "./plugin-market"        app        Up
plugin-market-db    "docker-entrypoint.s…"   postgres   Up
plugin-market-redis "docker-entrypoint.s…"   redis      Up
plugin-market-minio "/usr/bin/docker-ent…"   minio      Up
```

### 3. 等待服务就绪

```bash
# 查看应用日志
docker-compose logs -f app

# 等待看到以下日志
# [GIN] Listening and serving HTTP on :8080
```

### 4. 验证服务

```bash
# 测试健康检查
curl http://localhost:8081/health

# 预期响应
{
  "status": "ok",
  "database": "connected",
  "redis": "connected",
  "storage": "connected"
}
```

---

## 测试 API

### 1. 浏览插件列表

```bash
curl http://localhost:8081/api/v1/plugins
```

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "plugins": [],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 0,
      "total_pages": 0
    }
  }
}
```

> 初始状态下插件列表为空，需要先添加插件数据。

### 2. 初始化测试数据

```bash
# 进入应用容器
docker-compose exec app sh

# 运行种子数据脚本
./plugin-market seed

# 或使用 make 命令
make seed
```

### 3. 再次查询插件列表

```bash
curl http://localhost:8081/api/v1/plugins
```

现在应该能看到测试插件数据。

### 4. 查询插件详情

```bash
curl http://localhost:8081/api/v1/plugins/auth-jwt
```

### 5. 查询插件版本

```bash
curl http://localhost:8081/api/v1/plugins/auth-jwt/versions
```

### 6. 下载插件

```bash
curl -O -J http://localhost:8081/api/v1/plugins/auth-jwt/versions/1.0.0/download
```

### 7. 查询信任密钥

```bash
# 获取所有密钥
curl http://localhost:8081/api/v1/trust-keys

# 获取指定密钥详情
curl http://localhost:8081/api/v1/trust-keys/key_official_2025
```

---

## 本地开发

### 1. 安装依赖

```bash
# 安装 Go 依赖
go mod download

# 安装开发工具
make install-tools
```

### 2. 配置环境变量

创建 `.env.local` 文件：

```env
SERVER_PORT=8080
GIN_MODE=debug
DATABASE_URL=postgres://plugin_market:password@localhost:5432/plugin_market?sslmode=disable
REDIS_URL=redis://localhost:6379/0
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=plugins
MINIO_USE_SSL=false
LOG_LEVEL=debug
```

### 3. 启动依赖服务

```bash
# 仅启动数据库、Redis 和 MinIO
docker-compose up -d postgres redis minio
```

### 4. 运行数据库迁移

```bash
make migrate-up
```

### 5. 启动应用

```bash
# 方式 1: 使用 go run
go run cmd/server/main.go

# 方式 2: 使用 make
make run

# 方式 3: 使用 air（热重载）
air
```

### 6. 验证开发环境

```bash
curl http://localhost:8081/health
```

---

## 使用 Makefile

项目提供了便捷的 Makefile 命令：

```bash
# 构建应用
make build

# 运行应用
make run

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint

# 数据库迁移
make migrate-up
make migrate-down

# 生成 Ent 代码
make generate

# 清理构建产物
make clean

# 查看所有命令
make help
```

---

## 常见问题

### Q1: 端口 8080 已被占用

**解决方案**：

```bash
# 修改 .env 文件中的 SERVER_PORT
SERVER_PORT=8081

# 或停止占用端口的进程
sudo lsof -i :8080
kill -9 <PID>
```

### Q2: 数据库连接失败

**解决方案**：

```bash
# 检查 PostgreSQL 是否运行
docker-compose ps postgres

# 重启 PostgreSQL
docker-compose restart postgres

# 检查连接字符串
echo $DATABASE_URL
```

### Q3: MinIO 上传失败

**解决方案**：

```bash
# 检查 MinIO 状态
docker-compose ps minio

# 访问 MinIO 控制台
open http://localhost:9001

# 登录凭证
# 用户名: minioadmin
# 密码: minioadmin

# 确认 plugins 存储桶存在
```

### Q4: 如何重置数据库

**解决方案**：

```bash
# 停止所有服务
docker-compose down

# 删除数据卷
docker-compose down -v

# 重新启动
docker-compose up -d

# 运行迁移
make migrate-up

# 初始化测试数据
make seed
```

### Q5: 如何查看详细日志

**解决方案**：

```bash
# 查看应用日志
docker-compose logs -f app

# 查看所有服务日志
docker-compose logs -f

# 查看最近 100 行
docker-compose logs --tail=100 app

# 设置日志级别为 debug
# 修改 .env
LOG_LEVEL=debug
```

---

## 下一步

现在你已经成功启动了 Sub2API Plugin Market，可以：

1. **阅读 API 文档** - 了解所有可用接口
   - 查看 `docs/API.md`

2. **学习开发指南** - 了解如何开发和贡献代码
   - 查看 `docs/DEVELOPMENT.md`

3. **部署到生产环境** - 了解生产部署最佳实践
   - 查看 `docs/DEPLOYMENT.md`

4. **了解架构设计** - 深入理解系统架构
   - 查看 `docs/ARCHITECTURE.md`

---

## 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并删除数据卷
docker-compose down -v

# 停止并删除镜像
docker-compose down --rmi all
```

---

## 获取帮助

- **文档**: https://docs.sub2api.com
- **Issues**: https://github.com/your-org/sub2api-plugin-market/issues
- **讨论**: https://github.com/your-org/sub2api-plugin-market/discussions
- **邮件**: support@sub2api.com
