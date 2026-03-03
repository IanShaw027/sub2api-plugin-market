# Sub2API Plugin Market - 部署指南

## 环境要求

### 硬件要求
- **CPU**: 2 核心以上
- **内存**: 4GB 以上
- **磁盘**: 20GB 以上（用于存储插件文件）

### 软件要求
- **Docker**: 20.10+
- **Docker Compose**: 2.0+
- **操作系统**: Linux / macOS / Windows (WSL2)

---

## 快速部署（Docker Compose）

### 1. 克隆项目

```bash
git clone https://github.com/your-org/sub2api-plugin-market.git
cd sub2api-plugin-market
```

### 2. 配置环境变量

创建 `.env` 文件：

```bash
cp .env.example .env
```

编辑 `.env` 文件：

```env
# 服务配置
SERVER_PORT=8080
GIN_MODE=release

# 数据库配置
DATABASE_URL=postgres://plugin_market:your_password@postgres:5432/plugin_market?sslmode=disable

# Redis 配置
REDIS_URL=redis://redis:6379/0

# MinIO 配置
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=plugins
MINIO_USE_SSL=false

# 日志配置
LOG_LEVEL=info
LOG_FORMAT=json
```

### 3. 启动服务

```bash
docker-compose up -d
```

### 4. 验证部署

```bash
# 检查服务状态
docker-compose ps

# 查看日志
docker-compose logs -f app

# 测试 API
curl http://localhost:8080/api/v1/plugins
```

---

## 生产环境部署

### 架构图

```
                    ┌─────────────┐
                    │   Nginx     │
                    │  (反向代理)  │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │  Plugin     │
                    │  Market API │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   ┌────▼────┐      ┌─────▼─────┐     ┌─────▼─────┐
   │PostgreSQL│      │   Redis   │     │   MinIO   │
   │  (主库)  │      │  (缓存)   │     │  (存储)   │
   └─────────┘      └───────────┘     └───────────┘
```

### 1. 数据库配置

#### PostgreSQL 优化

```sql
-- 创建数据库
CREATE DATABASE plugin_market;

-- 创建用户
CREATE USER plugin_market WITH PASSWORD 'your_secure_password';

-- 授权
GRANT ALL PRIVILEGES ON DATABASE plugin_market TO plugin_market;

-- 性能优化（postgresql.conf）
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 4MB
min_wal_size = 1GB
max_wal_size = 4GB
```

#### 数据库迁移

```bash
# 进入容器
docker-compose exec app sh

# 运行迁移
./plugin-market migrate up

# 或使用 make 命令
make migrate-up
```

### 2. Redis 配置

```conf
# redis.conf
maxmemory 512mb
maxmemory-policy allkeys-lru
save 900 1
save 300 10
save 60 10000
```

### 3. MinIO 配置

```bash
# 创建存储桶
mc alias set myminio http://localhost:9000 minioadmin minioadmin
mc mb myminio/plugins
mc policy set download myminio/plugins

# 配置生命周期（可选）
mc ilm add --expiry-days 365 myminio/plugins/temp
```

### 4. Nginx 反向代理

```nginx
upstream plugin_market {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    server_name plugins.example.com;

    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name plugins.example.com;

    # SSL 证书
    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    # SSL 优化
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # 日志
    access_log /var/log/nginx/plugin_market_access.log;
    error_log /var/log/nginx/plugin_market_error.log;

    # 限流
    limit_req_zone $binary_remote_addr zone=api:10m rate=100r/m;
    limit_req zone=api burst=20 nodelay;

    location / {
        proxy_pass http://plugin_market;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # 下载接口特殊配置
    location ~ ^/api/v1/plugins/.*/versions/.*/download$ {
        proxy_pass http://plugin_market;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        
        # 增加超时时间
        proxy_read_timeout 300s;
        
        # 禁用缓冲（流式传输）
        proxy_buffering off;
        
        # 限流（下载接口）
        limit_req_zone $binary_remote_addr zone=download:10m rate=50r/m;
        limit_req zone=download burst=10 nodelay;
    }
}
```

### 5. 系统服务配置（Systemd）

创建 `/etc/systemd/system/plugin-market.service`：

```ini
[Unit]
Description=Sub2API Plugin Market
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=plugin-market
WorkingDirectory=/opt/plugin-market
ExecStart=/opt/plugin-market/plugin-market
Restart=on-failure
RestartSec=5s

# 环境变量
Environment="GIN_MODE=release"
EnvironmentFile=/opt/plugin-market/.env

# 资源限制
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable plugin-market
sudo systemctl start plugin-market
sudo systemctl status plugin-market
```

---

## 监控与日志

### 1. 健康检查

```bash
# 健康检查端点
curl http://localhost:8080/health

# 预期响应
{
  "status": "ok",
  "database": "connected",
  "redis": "connected",
  "storage": "connected"
}
```

### 2. 日志收集

#### 使用 Docker 日志

```bash
# 查看实时日志
docker-compose logs -f app

# 查看最近 100 行
docker-compose logs --tail=100 app

# 导出日志
docker-compose logs app > app.log
```

---

## 备份与恢复

### 1. 数据库备份

```bash
# 自动备份脚本
#!/bin/bash
BACKUP_DIR="/backup/postgres"
DATE=$(date +%Y%m%d_%H%M%S)

docker-compose exec -T postgres pg_dump -U plugin_market plugin_market | \
  gzip > "$BACKUP_DIR/plugin_market_$DATE.sql.gz"

# 保留最近 7 天的备份
find "$BACKUP_DIR" -name "*.sql.gz" -mtime +7 -delete
```

### 2. MinIO 备份

```bash
# 同步到备份存储
mc mirror myminio/plugins /backup/minio/plugins

# 或使用 rsync
rsync -avz /data/minio/plugins /backup/minio/
```

### 3. 恢复数据

```bash
# 恢复数据库
gunzip < plugin_market_20250301_120000.sql.gz | \
  docker-compose exec -T postgres psql -U plugin_market plugin_market

# 恢复 MinIO
mc mirror /backup/minio/plugins myminio/plugins
```

---

## 故障排查

### 问题 1: 服务无法启动

**症状**: `docker-compose up` 失败

**排查步骤**:
```bash
# 检查端口占用
sudo lsof -i :8080
sudo lsof -i :5432

# 检查 Docker 日志
docker-compose logs app

# 检查配置文件
cat .env
```

**解决方案**:
- 修改 `.env` 中的 `SERVER_PORT`
- 停止占用端口的进程
- 检查环境变量是否正确

### 问题 2: 数据库连接失败

**症状**: `database connection failed`

**排查步骤**:
```bash
# 检查 PostgreSQL 状态
docker-compose ps postgres

# 测试连接
docker-compose exec postgres psql -U plugin_market -d plugin_market

# 检查网络
docker-compose exec app ping postgres
```

**解决方案**:
- 确认 `DATABASE_URL` 配置正确
- 检查 PostgreSQL 容器是否运行
- 检查防火墙规则

---

## 安全加固

### 1. 防火墙配置

```bash
# 仅开放必要端口
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

### 2. 定期更新

```bash
# 更新 Docker 镜像
docker-compose pull
docker-compose up -d

# 更新系统
sudo apt update && sudo apt upgrade -y
```

---

## 联系支持

- **文档**: https://docs.sub2api.com
- **Issues**: https://github.com/your-org/sub2api-plugin-market/issues
- **邮件**: support@sub2api.com
