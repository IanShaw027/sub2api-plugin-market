# Sub2API Plugin Market - API 文档

## 概述

项目提供两组 API：

- 公开 API：插件浏览、下载、信任密钥查询
- 管理后台 API：管理员登录与提交流程审核

Base URL：

- 公开 API：`http://localhost:8081/api/v1`
- 管理 API：`http://localhost:8081/admin/api`

## 响应格式

统一返回结构：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

## HTTP 状态码与业务错误码

公开 API 使用“双层语义”：

- HTTP `200` + `code=0`：成功
- HTTP `400` + `code=1001`：参数错误
- HTTP `404` + `code=1002`：资源不存在
- HTTP `500` + `code=1003/1004/1005`：内部错误

业务错误码：

- `1001` - 参数错误
- `1002` - 资源不存在
- `1003` - 服务器内部错误
- `1004` - 数据库错误
- `1005` - 存储错误

## 公开 API

### 获取插件列表

`GET /api/v1/plugins`

查询参数：

- `category`：分类过滤
- `search`：关键词搜索
- `is_official`：官方插件过滤
- `page`：页码（默认 `1`）
- `page_size`：分页大小（默认 `20`，最大 `100`）

示例：

```bash
curl "http://localhost:8081/api/v1/plugins?page=1&page_size=20"
```

### 获取插件详情

`GET /api/v1/plugins/:name`

示例：

```bash
curl http://localhost:8081/api/v1/plugins/auth-jwt
```

### 获取插件版本列表

`GET /api/v1/plugins/:name/versions`

示例：

```bash
curl http://localhost:8081/api/v1/plugins/auth-jwt/versions
```

### 下载插件

`GET /api/v1/plugins/:name/versions/:version/download`

行为：

- 成功返回 `302 Found`，并在 `Location` 中提供预签名下载链接。
- 错误时返回 `400/404/500` 与统一错误体。

示例：

```bash
# 查看 302 响应头
curl -I http://localhost:8081/api/v1/plugins/auth-jwt/versions/1.0.0/download

# 跟随重定向下载
curl -L http://localhost:8081/api/v1/plugins/auth-jwt/versions/1.0.0/download -o auth-jwt-1.0.0.wasm
```

### 获取信任密钥列表

`GET /api/v1/trust-keys`

查询参数：

- `key_type`：密钥类型过滤
- `is_active`：是否激活

示例：

```bash
curl "http://localhost:8081/api/v1/trust-keys?key_type=official&is_active=true"
```

### 获取信任密钥详情

`GET /api/v1/trust-keys/:key_id`

示例：

```bash
curl http://localhost:8081/api/v1/trust-keys/key_official_2025
```

## 管理后台 API

管理后台 API 除登录外，均需要 `Authorization: Bearer <token>`。

### 登录

`POST /admin/api/auth/login`

请求体：

```json
{
  "username": "admin",
  "password": "admin123"
}
```

### 获取当前管理员

`GET /admin/api/auth/me`

### 刷新访问令牌

`POST /admin/api/auth/refresh`

请求体：

```json
{
  "refresh_token": "YOUR_TOKEN_HERE"
}
```

说明：仅接受登录返回的 `refresh_token`；`access token` 不能用于刷新。

### 登出

`POST /admin/api/auth/logout`

### 获取提交列表

`GET /admin/api/submissions?status=pending&page=1&page_size=20`

### 获取提交详情

`GET /admin/api/submissions/:id`

### 审核提交

`PUT /admin/api/submissions/:id/review`

请求体：

```json
{
  "action": "approve",
  "reviewer_notes": "审核通过"
}
```

兼容旧值：`action` 也接受 `approved/rejected`。

### 获取审核统计

`GET /admin/api/submissions/stats`

## 契约来源

如需精确字段定义与响应码，请以 OpenAPI 为准：

- `openapi/plugin-market-v1.yaml`
