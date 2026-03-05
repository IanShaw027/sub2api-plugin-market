# Sub2API Plugin Market - 管理后台使用指南

## 概述

管理后台用于处理插件提交审核流程，包含登录、列表查询、详情查看、审核和统计。

## 启动步骤

```bash
cp .env.example .env
docker-compose up -d
go run scripts/init_admin.go
go run cmd/server/main.go
```

访问地址：`http://localhost:8081/admin/login`

默认管理员账号：

- 用户名：`admin`
- 密码：`admin123`

## 认证说明

- 登录接口不需要 token。
- 其余管理接口均需要 `Authorization: Bearer <token>`。

## API 说明

### 1) 登录

`POST /admin/api/auth/login`

请求：

```json
{
  "username": "admin",
  "password": "admin123"
}
```

成功返回：

- `code=0`
- `data.token`
- `data.refresh_token`
- `data.user`

### 2) 当前用户

`GET /admin/api/auth/me`

### 3) 刷新访问令牌

`POST /admin/api/auth/refresh`

请求体：

```json
{
  "refresh_token": "YOUR_TOKEN_HERE"
}
```

说明：该接口只接受 `refresh_token`，不接受 `access token`。

### 4) 登出

`POST /admin/api/auth/logout`

### 5) 提交列表

`GET /admin/api/submissions?status=pending&page=1&page_size=20`

参数：

- `status`: `pending | approved | rejected | cancelled`
- `page`: 默认 `1`
- `page_size`: 默认 `20`，最大 `100`

### 6) 提交详情

`GET /admin/api/submissions/:id`

### 7) 审核提交

`PUT /admin/api/submissions/:id/review`

请求体：

```json
{
  "action": "approve",
  "reviewer_notes": "审核通过"
}
```

兼容旧字段：

- `action` 支持 `approved/rejected`
- `comment` 可作为 `reviewer_notes` 兼容输入

### 8) 审核统计

`GET /admin/api/submissions/stats`

## 常见问题

1. 登录失败  
   检查是否已执行 `go run scripts/init_admin.go` 初始化管理员。

2. 返回 `401/403`  
   检查 token 是否过期，或账号是否被禁用。

3. 服务启动失败提示 `missing ADMIN_JWT_SECRET`  
   检查 `.env` 是否包含 `ADMIN_JWT_SECRET`。

## 安全建议

1. 首次启动后修改默认管理员密码。
2. `ADMIN_JWT_SECRET` 使用强随机值并定期轮换。
3. 生产环境通过 HTTPS 暴露管理后台。
4. 限制管理后台访问来源（CORS/反向代理/IP 白名单）。
