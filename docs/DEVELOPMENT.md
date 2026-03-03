# Sub2API Plugin Market - 开发指南

本文档面向希望参与 Sub2API Plugin Market 开发的贡献者。

---

## 技术栈

### 后端
- **语言**: Go 1.21+
- **框架**: Gin (HTTP 路由)
- **ORM**: Ent (实体框架)
- **数据库**: PostgreSQL 14+
- **缓存**: Redis 7+
- **存储**: MinIO (S3 兼容)
- **签名**: Ed25519 (crypto/ed25519)

### 开发工具
- **构建**: Make
- **热重载**: Air
- **代码检查**: golangci-lint
- **测试**: Go testing + testify
- **容器**: Docker + Docker Compose

---

## 项目结构

```
sub2api-plugin-market/
├── cmd/
│   └── server/
│       └── main.go              # 应用入口
├── internal/
│   ├── handler/                 # HTTP 处理器
│   │   ├── plugin_handler.go   # 插件相关接口
│   │   ├── trust_key_handler.go # 信任密钥接口
│   │   └── router.go            # 路由配置
│   ├── service/                 # 业务逻辑层
│   │   ├── plugin_service.go
│   │   └── trust_key_service.go
│   ├── repository/              # 数据访问层
│   │   ├── plugin_repository.go
│   │   └── trust_key_repository.go
│   ├── middleware/              # 中间件
│   │   ├── cors.go
│   │   ├── logger.go
│   │   └── recovery.go
│   ├── storage/                 # 存储服务
│   │   └── minio.go
│   └── pluginsign/              # 签名验证
│       └── ed25519.go
├── ent/                         # Ent ORM 生成代码
│   └── schema/                  # 数据模型定义
│       ├── plugin.go
│       ├── plugin_version.go
│       ├── submission.go
│       ├── download_log.go
│       └── trust_key.go
├── config/                      # 配置管理
│   └── config.go
├── migrations/                  # 数据库迁移
├── tests/                       # 测试文件
│   ├── integration/
│   └── unit/
├── docs/                        # 文档
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── go.mod
```

---

## 开发环境搭建

### 1. 安装依赖

#### macOS

```bash
# 安装 Homebrew（如果未安装）
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 安装 Go
brew install go

# 安装 Docker Desktop
brew install --cask docker

# 安装开发工具
brew install golangci-lint
brew install air
```

#### Linux (Ubuntu/Debian)

```bash
# 安装 Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 安装 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# 安装 Docker Compose
sudo apt install docker-compose-plugin

# 安装开发工具
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/cosmtrek/air@latest
```

### 2. 克隆项目

```bash
git clone https://github.com/your-org/sub2api-plugin-market.git
cd sub2api-plugin-market
```

### 3. 安装 Go 依赖

```bash
go mod download
```

### 4. 启动依赖服务

```bash
# 启动 PostgreSQL, Redis, MinIO
docker-compose up -d postgres redis minio
```

### 5. 配置环境变量

```bash
cp .env.example .env.local
```

编辑 `.env.local`：

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

### 6. 运行数据库迁移

```bash
make migrate-up
```

### 7. 启动应用

```bash
# 使用 air 热重载
air

# 或使用 go run
go run cmd/server/main.go
```

---

## 开发工作流

### 1. 创建新功能分支

```bash
git checkout -b feature/your-feature-name
```

### 2. 编写代码

遵循项目的代码规范和架构模式。

### 3. 运行测试

```bash
# 运行所有测试
make test

# 运行单元测试
go test ./internal/...

# 运行集成测试
go test ./tests/integration/...

# 查看测试覆盖率
make test-coverage
```

### 4. 代码检查

```bash
# 运行 linter
make lint

# 自动修复格式问题
make fmt
```

### 5. 提交代码

```bash
git add .
git commit -m "feat(plugin): 添加插件搜索功能"
```

**提交信息规范**：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型（type）**：
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式（不影响功能）
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具链

**示例**：

```
feat(plugin): 添加插件搜索功能

- 实现全文搜索
- 支持按分类筛选
- 添加分页支持

Closes #123
```

### 6. 推送并创建 PR

```bash
git push origin feature/your-feature-name
```

然后在 GitHub 上创建 Pull Request。

---

## 数据模型开发

### 1. 定义 Ent Schema

在 `ent/schema/` 目录下创建新的 schema 文件：

```go
// ent/schema/plugin.go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/edge"
)

type Plugin struct {
    ent.Schema
}

func (Plugin) Fields() []ent.Field {
    return []ent.Field{
        field.String("name").Unique(),
        field.String("display_name"),
        field.Text("description"),
        field.String("category"),
        field.Bool("is_official").Default(false),
        field.Time("created_at"),
        field.Time("updated_at"),
    }
}

func (Plugin) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("versions", PluginVersion.Type),
    }
}
```

### 2. 生成 Ent 代码

```bash
make generate
```

---

## API 开发

### 1. 定义路由

在 `internal/handler/router.go` 中添加路由：

```go
func SetupRouter(r *gin.Engine, handlers *Handlers) {
    api := r.Group("/api/v1")
    {
        // 插件相关
        plugins := api.Group("/plugins")
        {
            plugins.GET("", handlers.Plugin.List)
            plugins.GET("/:name", handlers.Plugin.Get)
            plugins.GET("/:name/versions", handlers.Plugin.ListVersions)
            plugins.GET("/:name/versions/:version/download", handlers.Plugin.Download)
        }
    }
}
```

### 2. 实现 Handler

在 `internal/handler/plugin_handler.go` 中实现处理器：

```go
type PluginHandler struct {
    service *service.PluginService
}

func (h *PluginHandler) List(c *gin.Context) {
    // 解析查询参数
    var req ListPluginsRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        c.JSON(400, gin.H{"code": 1002, "message": "参数错误"})
        return
    }

    // 调用 service
    plugins, total, err := h.service.ListPlugins(c.Request.Context(), req)
    if err != nil {
        c.JSON(500, gin.H{"code": 5000, "message": err.Error()})
        return
    }

    // 返回响应
    c.JSON(200, gin.H{
        "code": 0,
        "message": "success",
        "data": gin.H{
            "plugins": plugins,
            "pagination": gin.H{
                "page": req.Page,
                "page_size": req.PageSize,
                "total": total,
            },
        },
    })
}
```

---

## 测试

### 单元测试

```go
// internal/service/plugin_service_test.go
package service_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestPluginService_ListPlugins(t *testing.T) {
    // 设置测试数据库
    client := setupTestDB(t)
    defer client.Close()

    // 创建 service
    repo := repository.NewPluginRepository(client)
    service := service.NewPluginService(repo)

    // 插入测试数据
    createTestPlugin(t, client, "test-plugin")

    // 执行测试
    plugins, total, err := service.ListPlugins(context.Background(), ListPluginsRequest{
        Page: 1,
        PageSize: 10,
    })

    // 断言
    assert.NoError(t, err)
    assert.Equal(t, 1, total)
    assert.Len(t, plugins, 1)
    assert.Equal(t, "test-plugin", plugins[0].Name)
}
```

---

## 调试

### 使用 Delve 调试器

```bash
# 安装 delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试
dlv debug cmd/server/main.go

# 在代码中设置断点
(dlv) break internal/handler/plugin_handler.go:25
(dlv) continue
```

### 使用 VS Code 调试

创建 `.vscode/launch.json`：

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Server",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/server",
            "env": {
                "GIN_MODE": "debug"
            },
            "args": []
        }
    ]
}
```

---

## 代码规范

### Go 代码风格

- 遵循 [Effective Go](https://go.dev/doc/effective_go)
- 使用 `gofmt` 格式化代码
- 变量命名使用驼峰命名法
- 导出的函数和类型添加注释

### 错误处理

```go
// 好的做法
if err != nil {
    return fmt.Errorf("failed to create plugin: %w", err)
}

// 避免
if err != nil {
    panic(err)  // 不要使用 panic
}
```

### 日志记录

```go
import "github.com/sirupsen/logrus"

// 使用结构化日志
log.WithFields(logrus.Fields{
    "plugin_name": name,
    "version": version,
}).Info("Plugin downloaded")
```

---

## 性能优化

### 数据库查询优化

```go
// 使用预加载避免 N+1 查询
plugins, err := client.Plugin.Query().
    WithVersions().  // 预加载版本
    All(ctx)

// 使用索引
field.String("name").Unique().StorageKey("idx_plugin_name")
```

### 缓存策略

```go
// 使用 Redis 缓存
func (s *PluginService) GetPlugin(ctx context.Context, name string) (*ent.Plugin, error) {
    // 尝试从缓存获取
    cached, err := s.cache.Get(ctx, "plugin:"+name)
    if err == nil {
        return cached, nil
    }

    // 从数据库查询
    plugin, err := s.repo.GetByName(ctx, name)
    if err != nil {
        return nil, err
    }

    // 写入缓存
    s.cache.Set(ctx, "plugin:"+name, plugin, 5*time.Minute)
    return plugin, nil
}
```

---

## 常见问题

### Q: 如何添加新的 API 接口？

1. 在 `internal/handler/` 添加处理器方法
2. 在 `internal/service/` 实现业务逻辑
3. 在 `internal/repository/` 实现数据访问
4. 在 `router.go` 注册路由
5. 编写测试

### Q: 如何修改数据模型？

1. 修改 `ent/schema/` 中的 schema 定义
2. 运行 `make generate` 生成代码
3. 创建数据库迁移
4. 运行 `make migrate-up` 应用迁移

### Q: 如何调试数据库查询？

```go
// 启用 SQL 日志
client := ent.NewClient(ent.Driver(drv), ent.Debug())
```

---

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 编写代码和测试
4. 确保所有测试通过
5. 提交 Pull Request

详见 [CONTRIBUTING.md](CONTRIBUTING.md)

---

## 资源链接

- **Gin 文档**: https://gin-gonic.com/docs/
- **Ent 文档**: https://entgo.io/docs/getting-started
- **Go 标准库**: https://pkg.go.dev/std
- **项目 Wiki**: https://github.com/your-org/sub2api-plugin-market/wiki
