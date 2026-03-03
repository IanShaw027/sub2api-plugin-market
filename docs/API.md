# Sub2API Plugin Market - API 文档

## 概述

Sub2API Plugin Market 提供 RESTful API 用于插件浏览、下载和信任密钥管理。

**Base URL**: `http://localhost:8081/api/v1`

**认证**: 当前版本无需认证（浏览和下载为公开接口）

## 通用响应格式

### 成功响应
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

### 错误响应
```json
{
  "code": 1001,
  "message": "插件不存在",
  "data": null
}
```

### 错误码
- `0` - 成功
- `1001` - 资源不存在
- `1002` - 参数错误
- `1003` - 数据库错误
- `1004` - 存储服务错误
- `5000` - 服务器内部错误

---

## 插件接口

### 1. 获取插件列表

获取插件市场中的所有插件（支持分页和筛选）。

**请求**

```http
GET /api/v1/plugins
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 | 默认值 |
|------|------|------|------|--------|
| category | string | 否 | 插件分类（如 `auth`, `storage`, `ai`） | - |
| is_official | boolean | 否 | 是否官方插件 | - |
| search | string | 否 | 搜索关键词（匹配名称和描述） | - |
| page | integer | 否 | 页码（从 1 开始） | 1 |
| page_size | integer | 否 | 每页数量（最大 100） | 20 |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "plugins": [
      {
        "id": 1,
        "name": "auth-jwt",
        "display_name": "JWT 认证插件",
        "description": "提供 JWT 令牌验证和生成功能",
        "category": "auth",
        "author": "Sub2API Team",
        "is_official": true,
        "latest_version": "1.2.0",
        "download_count": 1523,
        "created_at": "2025-01-15T10:30:00Z",
        "updated_at": "2025-02-20T14:22:00Z"
      },
      {
        "id": 2,
        "name": "storage-minio",
        "display_name": "MinIO 存储插件",
        "description": "对接 MinIO 对象存储服务",
        "category": "storage",
        "author": "Community",
        "is_official": false,
        "latest_version": "0.9.5",
        "download_count": 842,
        "created_at": "2025-02-01T08:15:00Z",
        "updated_at": "2025-02-28T16:40:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 45,
      "total_pages": 3
    }
  }
}
```

**cURL 示例**

```bash
# 获取所有插件
curl http://localhost:8081/api/v1/plugins

# 筛选官方认证插件
curl "http://localhost:8081/api/v1/plugins?is_official=true"

# 搜索存储相关插件
curl "http://localhost:8081/api/v1/plugins?search=storage&page=1&page_size=10"

# 按分类筛选
curl "http://localhost:8081/api/v1/plugins?category=auth"
```

---

### 2. 获取插件详情

获取指定插件的详细信息。

**请求**

```http
GET /api/v1/plugins/:name
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| name | string | 插件名称（唯一标识） |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "auth-jwt",
    "display_name": "JWT 认证插件",
    "description": "提供 JWT 令牌验证和生成功能，支持 HS256/RS256 算法",
    "long_description": "# JWT 认证插件\n\n完整的 JWT 认证解决方案...",
    "category": "auth",
    "author": "Sub2API Team",
    "author_email": "team@sub2api.com",
    "homepage": "https://github.com/sub2api/plugin-auth-jwt",
    "repository": "https://github.com/sub2api/plugin-auth-jwt",
    "license": "MIT",
    "is_official": true,
    "latest_version": "1.2.0",
    "download_count": 1523,
    "tags": ["auth", "jwt", "security"],
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-02-20T14:22:00Z"
  }
}
```

**cURL 示例**

```bash
curl http://localhost:8081/api/v1/plugins/auth-jwt
```

**错误响应**

```json
{
  "code": 1001,
  "message": "插件不存在",
  "data": null
}
```

---

### 3. 获取插件版本列表

获取指定插件的所有版本信息。

**请求**

```http
GET /api/v1/plugins/:name/versions
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| name | string | 插件名称 |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "plugin_name": "auth-jwt",
    "versions": [
      {
        "id": 5,
        "version": "1.2.0",
        "plugin_api_version": "v1",
        "description": "修复安全漏洞，优化性能",
        "changelog": "- 修复 CVE-2025-1234\n- 性能提升 30%\n- 新增 RS512 算法支持",
        "file_size": 2457600,
        "file_hash": "sha256:a3f5b8c9d2e1...",
        "sign_key_id": "key_official_2025",
        "signature": "ed25519:9f8e7d6c5b4a...",
        "download_url": "/api/v1/plugins/auth-jwt/versions/1.2.0/download",
        "is_latest": true,
        "created_at": "2025-02-20T14:22:00Z"
      },
      {
        "id": 4,
        "version": "1.1.0",
        "plugin_api_version": "v1",
        "description": "新增 RS256 算法支持",
        "changelog": "- 支持 RS256 非对称加密\n- 改进错误提示",
        "file_size": 2301440,
        "file_hash": "sha256:b4e6c7d8a9f0...",
        "sign_key_id": "key_official_2024",
        "signature": "ed25519:8d7c6b5a4e3f...",
        "download_url": "/api/v1/plugins/auth-jwt/versions/1.1.0/download",
        "is_latest": false,
        "created_at": "2025-01-30T09:15:00Z"
      }
    ]
  }
}
```

**cURL 示例**

```bash
curl http://localhost:8081/api/v1/plugins/auth-jwt/versions
```

---

### 4. 下载插件

下载指定版本的插件 WASM 文件。

**请求**

```http
GET /api/v1/plugins/:name/versions/:version/download
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| name | string | 插件名称 |
| version | string | 版本号（如 `1.2.0`） |

**响应**

- **Content-Type**: `application/wasm`
- **Content-Disposition**: `attachment; filename="auth-jwt-1.2.0.wasm"`
- **X-File-Hash**: `sha256:a3f5b8c9d2e1...`
- **X-Signature**: `ed25519:9f8e7d6c5b4a...`
- **X-Sign-Key-ID**: `key_official_2025`

**cURL 示例**

```bash
# 下载插件
curl -O -J http://localhost:8081/api/v1/plugins/auth-jwt/versions/1.2.0/download

# 下载并验证哈希
curl -v http://localhost:8081/api/v1/plugins/auth-jwt/versions/1.2.0/download \
  -o auth-jwt-1.2.0.wasm
```

**错误响应**

```json
{
  "code": 1001,
  "message": "版本不存在",
  "data": null
}
```

---

## 信任密钥接口

### 5. 获取信任密钥列表

获取所有可信的签名公钥。

**请求**

```http
GET /api/v1/trust-keys
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 | 默认值 |
|------|------|------|------|--------|
| is_active | boolean | 否 | 是否仅返回激活的密钥 | true |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "trust_keys": [
      {
        "id": 1,
        "key_id": "key_official_2025",
        "public_key": "ed25519:AAAAC3NzaC1lZDI1NTE5AAAAIGq...",
        "owner": "Sub2API Official",
        "description": "Sub2API 官方签名密钥（2025）",
        "is_active": true,
        "created_at": "2025-01-01T00:00:00Z",
        "expires_at": "2026-01-01T00:00:00Z"
      },
      {
        "id": 2,
        "key_id": "key_official_2024",
        "public_key": "ed25519:AAAAC3NzaC1lZDI1NTE5AAAAIHr...",
        "owner": "Sub2API Official",
        "description": "Sub2API 官方签名密钥（2024，已过期）",
        "is_active": false,
        "created_at": "2024-01-01T00:00:00Z",
        "expires_at": "2025-01-01T00:00:00Z"
      }
    ]
  }
}
```

**cURL 示例**

```bash
# 获取所有激活的密钥
curl http://localhost:8081/api/v1/trust-keys

# 获取所有密钥（包括过期）
curl "http://localhost:8081/api/v1/trust-keys?is_active=false"
```

---

### 6. 获取信任密钥详情

获取指定密钥的详细信息。

**请求**

```http
GET /api/v1/trust-keys/:key_id
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| key_id | string | 密钥 ID |

**响应示例**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "key_id": "key_official_2025",
    "public_key": "ed25519:AAAAC3NzaC1lZDI1NTE5AAAAIGq...",
    "owner": "Sub2API Official",
    "description": "Sub2API 官方签名密钥（2025）",
    "is_active": true,
    "created_at": "2025-01-01T00:00:00Z",
    "expires_at": "2026-01-01T00:00:00Z",
    "signed_plugins_count": 23
  }
}
```

**cURL 示例**

```bash
curl http://localhost:8081/api/v1/trust-keys/key_official_2025
```

---

## 签名验证

所有插件下载都包含 Ed25519 签名，客户端应验证签名以确保插件完整性。

### 验证流程

1. 下载插件文件
2. 从响应头获取 `X-File-Hash`、`X-Signature`、`X-Sign-Key-ID`
3. 通过 `/api/v1/trust-keys/:key_id` 获取公钥
4. 使用 Ed25519 算法验证签名

### 示例代码（Go）

```go
package main

import (
    "crypto/ed25519"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "io"
    "net/http"
)

func verifyPlugin(fileData []byte, signature, publicKeyStr string) bool {
    // 解码公钥
    publicKey, _ := base64.StdEncoding.DecodeString(publicKeyStr)
    
    // 解码签名
    sig, _ := base64.StdEncoding.DecodeString(signature)
    
    // 计算文件哈希
    hash := sha256.Sum256(fileData)
    
    // 验证签名
    return ed25519.Verify(publicKey, hash[:], sig)
}
```

---

## 限流

当前版本无限流限制，生产环境建议配置：

- 插件列表：100 req/min
- 插件详情：200 req/min
- 下载接口：50 req/min

---

## 变更日志

### v1.0.0 (2025-03-01)
- 初始版本
- 实现 6 个核心 API 接口
- 支持插件浏览、下载和签名验证
