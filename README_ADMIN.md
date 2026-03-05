# Sub2API Plugin Market - 管理后台说明

该文档聚焦管理后台功能和使用方式。项目总体启动方式请参考根目录 `README.md`。

## 当前已实现能力

- 管理员登录（JWT）
- 当前用户查询
- 登出（客户端清理 token）
- 提交列表/详情查询
- 提交审核（approve/reject，兼容 approved/rejected）
- 审核统计

## 快速启动

```bash
cp .env.example .env
docker-compose up -d
go run scripts/init_admin.go
go run cmd/server/main.go
```

管理后台地址：

- 登录页：`http://localhost:8081/admin/login`
- 管理页：`http://localhost:8081/admin/`

默认账号：

- 用户名：`admin`
- 密码：`admin123`

## 关键配置

以 `.env.example` 为准，管理后台相关重点配置：

```env
PORT=8081
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=plugin_market
ADMIN_JWT_SECRET=YOUR_TOKEN_HERE
```

## 管理 API（简表）

- `POST /admin/api/auth/login`
- `POST /admin/api/auth/refresh`
- `GET /admin/api/auth/me`
- `POST /admin/api/auth/logout`
- `GET /admin/api/submissions`
- `GET /admin/api/submissions/:id`
- `PUT /admin/api/submissions/:id/review`
- `GET /admin/api/submissions/stats`

详细示例见：

- `docs/ADMIN_GUIDE.md`
- `docs/API.md`
- `openapi/plugin-market-v1.yaml`
