# Sub2API Plugin Market - API 实现文档

## 项目概述

这是 sub2api-plugin-market 项目的核心 API 实现，提供插件浏览、下载和信任密钥管理功能。

## 技术栈

- **Web 框架**: Gin
- **ORM**: Ent
- **数据库**: PostgreSQL
- **存储**: MinIO（待集成）

## 项目结构

```
internal/
├── api/v1/
│   ├── handler/
│   │   ├── response.go           # 统一响应格式
│   │   ├── plugin_handler.go     # 插件接口处理器
│   │   ├── download_handler.go   # 下载接口处理器
│   │   └── trust_key_handler.go  # 信任密钥接口处理器
│   └── router.go                 # 路由注册
├── service/
│   ├── plugin_service.go         # 插件业务逻辑
│   ├── download_service.go       # 下载业务逻辑
│   └── trust_key_service.go      # 信任密钥业务逻辑
└── repository/
    ├── plugin_repository.go      # 插件数据访问
    └── trust_key_repository.go   # 信任密钥数据访问
```

## API 接口

### 1. 插件浏览接口

#### 获取插件列表
```
GET /api/v1/plugins
```

**查询参数**:
- `category` (可选): 插件分类 (proxy, auth, analytics, security, other)
- `is_official` (可选): 是否官方插件 (true/false)
- `search` (可选): 搜索关键词（匹配名称、显示名称、描述）
- `page` (可选): 页码，默认 1
- `page_size` (可选): 每页数量，默认 20，最大 100

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "plugins": [...],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

#### 获取插件详情
```
GET /api/v1/plugins/:name
```

**路径参数**:
- `name`: 插件名称

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid",
    "name": "example-plugin",
    "display_name": "Example Plugin",
    "description": "插件描述",
    "author": "作者名称",
    "category": "proxy",
    "is_official": true,
    "download_count": 1000,
    "versions": [...]
  }
}
```

#### 获取插件版本列表
```
GET /api/v1/plugins/:name/versions
```

**路径参数**:
- `name`: 插件名称

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "plugin_name": "example-plugin",
    "versions": [...],
    "total": 5
  }
}
```

### 2. 下载接口

#### 下载插件
```
GET /api/v1/plugins/:name/versions/:version/download
```

**路径参数**:
- `name`: 插件名称
- `version`: 版本号

**功能**:
- 验证插件和版本存在
- 记录下载日志
- 增加下载计数
- 重定向到预签名 URL（或直接返回文件流）

### 3. 信任密钥管理接口

#### 获取信任密钥列表
```
GET /api/v1/trust-keys
```

**查询参数**:
- `key_type` (可选): 密钥类型 (official, verified_publisher, community)
- `is_active` (可选): 是否激活 (true/false)

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "trust_keys": [...],
    "total": 10
  }
}
```

#### 获取信任密钥详情
```
GET /api/v1/trust-keys/:key_id
```

**路径参数**:
- `key_id`: 密钥 ID

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid",
    "key_id": "key-001",
    "public_key": "base64-encoded-key",
    "key_type": "official",
    "owner_name": "Sub2API Team",
    "is_active": true
  }
}
```

## 环境变量配置

```bash
# 服务器配置
PORT=8080

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=plugin_market
DB_SSLMODE=disable
```

## 启动服务

### 1. 启动数据库
```bash
docker-compose up -d postgres
```

### 2. 运行迁移
```bash
make migrate-up
```

### 3. 启动服务器
```bash
go run cmd/server/main.go
```

或编译后运行：
```bash
go build -o bin/server ./cmd/server
./bin/server
```

## 测试 API

使用提供的测试脚本：
```bash
./test_api.sh
```

或使用 curl 手动测试：
```bash
# 健康检查
curl http://localhost:8081/health

# 获取插件列表
curl http://localhost:8081/api/v1/plugins

# 获取插件详情
curl http://localhost:8081/api/v1/plugins/example-plugin
```

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 资源不存在 |
| 1003 | 内部错误 |
| 1004 | 数据库错误 |
| 1005 | 存储错误 |

## TODO

- [ ] 集成 MinIO 存储（实现文件上传和下载）
- [ ] 实现签名校验功能
- [ ] 实现下载日志记录
- [ ] 实现下载计数原子操作
- [ ] 添加 API 限流中间件
- [ ] 添加 CORS 中间件
- [ ] 添加日志中间件
- [ ] 添加单元测试
- [ ] 添加集成测试

## 开发指南

### 添加新接口

1. 在 `internal/repository` 添加数据访问方法
2. 在 `internal/service` 添加业务逻辑
3. 在 `internal/api/v1/handler` 添加 HTTP 处理器
4. 在 `internal/api/v1/router.go` 注册路由

### 代码规范

- 使用 Go 标准命名规范
- 错误处理使用 `fmt.Errorf` 包装
- 统一使用 `handler.Success` 和 `handler.Error` 返回响应
- 数据库查询使用 Ent ORM
- 业务逻辑放在 Service 层，不要在 Handler 中直接操作数据库
