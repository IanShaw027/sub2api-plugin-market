# 环境变量清单

> 本文档列出 sub2api-plugin-market 服务的全部环境变量。

## 数据库

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `DB_HOST` | 否 | `localhost` | PostgreSQL 主机 |
| `DB_PORT` | 否 | `5433` | PostgreSQL 端口 |
| `DB_USER` | 否 | `postgres` | 数据库用户名 |
| `DB_PASSWORD` | 否 | `postgres` | 数据库密码 |
| `DB_NAME` | 否 | `plugin_market` | 数据库名称 |
| `DB_SSLMODE` | 否 | `disable` | SSL 模式（disable/require/verify-full） |

## 服务运行

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `PORT` | 否 | `8081` | 服务监听端口 |
| `GIN_MODE` | 否 | `debug` | Gin 运行模式（debug/release/test） |
| `HOST_RUNTIME` | 否 | `wasm` | 宿主运行时类型 |
| `HOST_API_VERSION` | 否 | `1.0.0` | Host API 版本号 |

## 安全

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `ADMIN_JWT_SECRET` | **生产必填** | — | 管理后台 JWT 密钥。`GIN_MODE=release` 时禁止弱默认值 |
| `GITHUB_WEBHOOK_SECRET` | **生产必填** | — | GitHub Webhook 签名验证密钥。`GIN_MODE=release` 时空值将拒绝 webhook |

## 插件签名

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `PLUGIN_SIGNING_KEY_ID` | 否 | — | 签名密钥 ID，与 `PLUGIN_SIGNING_PRIVATE_KEY` 配对使用 |
| `PLUGIN_SIGNING_PRIVATE_KEY` | 否 | — | Ed25519 私钥（hex 编码，64 字节），用于 Sync 自动签名发布 |

> 两个签名变量同时设置时启用自动签名。Sync 下载的 WASM 会自动签名并发布为 `published` 状态。

## GitHub API

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `GITHUB_TOKEN` | 否 | — | GitHub API 访问令牌，用于 Sync 下载 release assets。未设置时使用匿名访问（有速率限制） |

## 存储

存储后端由 `sub2api-storage` 包初始化，支持 MinIO 和本地文件系统。具体变量参见 storage 包文档。

---

## 生产部署最小配置示例

```bash
# 必填
export ADMIN_JWT_SECRET="your-strong-secret-at-least-32-chars"
export GITHUB_WEBHOOK_SECRET="your-webhook-secret"

# 数据库
export DB_HOST="pg.production.internal"
export DB_PORT="5432"
export DB_USER="plugin_market"
export DB_PASSWORD="strong-password"
export DB_NAME="plugin_market"
export DB_SSLMODE="require"

# 可选: 自动签名
export PLUGIN_SIGNING_KEY_ID="prod-key-2026"
export PLUGIN_SIGNING_PRIVATE_KEY="<hex-encoded-ed25519-private-key>"

# 可选: GitHub API（推荐设置以避免限流）
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

# 运行
export GIN_MODE="release"
export PORT="8081"
```
