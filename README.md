# Sub2API Plugin Market

Sub2API 插件市场服务 - 提供插件上传、审核、分发和管理功能。

## 功能特性

- **插件管理**：插件上传、版本管理、元数据存储
- **安全审核**：WASM 插件安全检查、签名验证
- **存储服务**：MinIO 对象存储集成
- **API 服务**：RESTful API，支持插件查询、下载
- **数据库**：PostgreSQL + Ent ORM

## 技术栈

- **语言**：Go 1.21+
- **Web 框架**：Gin
- **ORM**：Ent
- **数据库**：PostgreSQL 15+
- **对象存储**：MinIO
- **容器化**：Docker + Docker Compose

## 快速开始

### 前置要求

- Go 1.21+
- Docker & Docker Compose
- Make

### 本地开发

```bash
# 启动依赖服务（PostgreSQL + MinIO）
make docker-up

# 运行数据库迁移
make migrate

# 启动服务
make run

# 运行测试
make test
```

### 环境变量

创建 `.env` 文件：

```env
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/plugin_market?sslmode=disable
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=plugins
SERVER_PORT=8080
```

## API 文档

服务启动后访问：`http://localhost:8080/swagger/index.html`

## 项目结构

```
sub2api-plugin-market/
├── cmd/server/          # 应用入口
├── internal/
│   ├── api/v1/         # API 路由和处理器
│   ├── service/        # 业务逻辑层
│   ├── repository/     # 数据访问层
│   ├── storage/        # MinIO 存储服务
│   ├── checker/        # WASM 安全检查器
│   └── model/          # 数据模型
├── ent/schema/         # Ent schema 定义
├── config/             # 配置文件
├── migrations/         # 数据库迁移文件
└── docker-compose.yml  # Docker 编排配置
```

## License

MIT
