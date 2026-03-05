# Sub2API Plugin Market

[![GitHub release](https://img.shields.io/github/v/release/IanShaw027/sub2api-plugin-market)](https://github.com/IanShaw027/sub2api-plugin-market/releases)
[![Go Version](https://img.shields.io/badge/Go-1.25.7-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![CI](https://github.com/IanShaw027/sub2api-plugin-market/workflows/CI/badge.svg)](https://github.com/IanShaw027/sub2api-plugin-market/actions)

Sub2API 插件市场服务 - 提供插件浏览、下载、签名验证功能。

**🔗 相关项目**
- [sub2api-pluginsign](https://github.com/IanShaw027/sub2api-pluginsign) - Ed25519 签名验证库
- [sub2api-storage](https://github.com/IanShaw027/sub2api-storage) - MinIO + 本地存储抽象库

## 功能特性

- **插件管理**：插件上传、版本管理、元数据存储
- **安全审核**：WASM 插件安全检查、签名验证
- **存储服务**：MinIO 对象存储集成
- **API 服务**：RESTful API，支持插件查询、下载
- **管理后台**：插件审核、用户管理、统计报表
- **数据库**：PostgreSQL + Ent ORM

## 技术栈

- **语言**：Go 1.25+
- **Web 框架**：Gin
- **ORM**：Ent
- **数据库**：PostgreSQL 15+
- **对象存储**：MinIO
- **容器化**：Docker + Docker Compose

## 快速开始

### 前置要求

- Go 1.25+
- Docker & Docker Compose
- Make

### 本地开发

```bash
# 1. 准备环境变量
cp .env.example .env

# 2. 启动依赖服务（PostgreSQL + MinIO）
docker-compose up -d

# 3. 初始化管理员账号
go run scripts/init_admin.go

# 4. 启动服务
go run cmd/server/main.go

# 5. 访问服务
# - API: http://localhost:8081/api/v1
# - 管理后台: http://localhost:8081/admin/login
# - 健康检查: http://localhost:8081/health
# - 健康检查响应: {"status":"ok"}
```

默认管理员账号：
- 用户名：`admin`
- 密码：`admin123`

### 测试

```bash
# 运行单元测试
go test ./...

# 测试管理后台
./scripts/test_admin.sh
```

### 环境变量

推荐基于 `.env.example`：

```bash
cp .env.example .env
```

关键变量说明（服务端实际读取）：

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
STORAGE_MINIO_ACCESS_KEY=minioadmin
STORAGE_MINIO_SECRET_KEY=minioadmin
STORAGE_MINIO_BUCKET=plugin-market
STORAGE_MINIO_USE_SSL=false

ADMIN_JWT_SECRET=YOUR_TOKEN_HERE
```

## API 文档

### 公开 API

- `GET /api/v1/plugins` - 获取插件列表
- `GET /api/v1/plugins/:name` - 获取插件详情
- `GET /api/v1/plugins/:name/versions` - 获取插件版本列表
- `GET /api/v1/plugins/:name/versions/:version/download` - 返回 302 并跳转到预签名下载 URL
- `GET /api/v1/trust-keys` - 获取信任密钥列表

### 管理后台 API

- `POST /admin/api/auth/login` - 管理员登录
- `GET /admin/api/auth/me` - 获取当前用户信息
- `GET /admin/api/submissions` - 获取提交列表
- `PUT /admin/api/submissions/:id/review` - 审核提交
- `GET /admin/api/submissions/stats` - 获取审核统计

详细文档：[管理后台使用指南](./docs/ADMIN_GUIDE.md)

## 项目结构

```
sub2api-plugin-market/
├── cmd/server/          # 应用入口
├── internal/
│   ├── api/v1/         # 公开 API 路由和处理器
│   ├── admin/          # 管理后台
│   │   ├── handler/    # 管理后台处理器
│   │   ├── service/    # 管理后台服务
│   │   └── middleware/ # 认证中间件
│   ├── auth/           # JWT 认证服务
│   ├── service/        # 业务逻辑层
│   ├── repository/     # 数据访问层
│   └── model/          # 数据模型
├── ent/schema/         # Ent schema 定义
│   ├── admin_user.go   # 管理员用户表
│   ├── plugin.go       # 插件表
│   ├── submission.go   # 提交审核表
│   └── ...
├── web/admin/          # 管理后台前端
│   ├── login.html      # 登录页面
│   └── index.html      # 审核管理页面
├── scripts/            # 工具脚本
│   ├── init_admin.go   # 初始化管理员
│   └── test_admin.sh   # 测试脚本
├── docs/               # 文档
│   └── ADMIN_GUIDE.md  # 管理后台使用指南
└── docker-compose.yml  # Docker 编排配置
```

## License

MIT
