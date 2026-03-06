# 插件开发指南

## 快速开始（5 分钟 Hello World）

### 1. 创建项目

```bash
mkdir my-plugin && cd my-plugin
go mod init github.com/yourname/my-plugin
go get github.com/IanShaw027/sub2api/backend/internal/pluginapi
```

### 2. 实现插件

```go
package main

import (
    "context"
    "github.com/IanShaw027/sub2api/backend/internal/pluginapi"
)

type HelloPlugin struct{}

func (p *HelloPlugin) Metadata() pluginapi.Metadata {
    return pluginapi.Metadata{
        Name:        "hello-world",
        Version:     "1.0.0",
        Description: "一个最简单的拦截器插件",
    }
}

func (p *HelloPlugin) Intercept(ctx context.Context, req *pluginapi.GatewayRequest, next pluginapi.Handler) (*pluginapi.GatewayResponse, error) {
    req.Headers["X-Hello"] = []string{"world"}
    return next(ctx, req)
}
```

### 3. 编译 WASM

```bash
GOOS=wasip1 GOARCH=wasm go build -o hello.wasm .
```

### 4. 签名并发布

```bash
sub2api-pluginsign sign --key your-key.pem --wasm hello.wasm --manifest manifest.json
sub2api plugin submit --archive hello-1.0.0.tar.gz
```

---

## 三种插件类型

### InterceptorPlugin — 拦截器

在请求链路中做前后置处理，可以修改请求/响应或短路返回。

```go
type InterceptorPlugin interface {
    Plugin
    Intercept(ctx context.Context, req *GatewayRequest, next Handler) (*GatewayResponse, error)
}
```

典型场景：鉴权、限流、请求日志、Header 注入。

```go
func (p *AuthPlugin) Intercept(ctx context.Context, req *pluginapi.GatewayRequest, next pluginapi.Handler) (*pluginapi.GatewayResponse, error) {
    token := req.Headers["Authorization"]
    if len(token) == 0 {
        return &pluginapi.GatewayResponse{StatusCode: 401, Body: []byte("unauthorized")}, nil
    }
    return next(ctx, req)
}
```

### TransformPlugin — 转换器

纯请求/响应转换，不控制调用链。可选实现 `ChunkTransformer` 支持流式逐行转换。

```go
type TransformPlugin interface {
    Plugin
    TransformRequest(ctx context.Context, req *GatewayRequest) error
    TransformResponse(ctx context.Context, resp *GatewayResponse) error
}

// 可选：流式 SSE 逐行转换
type ChunkTransformer interface {
    TransformChunk(chunk []byte) ([]byte, error)
}
```

典型场景：请求格式转换、响应字段映射、SSE 数据重写。

### ProviderPlugin — 提供者

直接产出响应，支持流式写入。又分为三种模式：

| 接口 | 流式方式 | 适用场景 |
|------|---------|---------|
| `ProviderPlugin` | 通过 `StreamWriter` 手动写 | 完全自定义响应 |
| `StreamingProviderPlugin` | Host 管理 SSE 连接，插件逐行转换 | 标准 SSE 代理 |
| `StreamProviderPlugin` | 委托式，插件维护内部状态 | 有状态的 SSE 代理 |

```go
// 基础 Provider
type ProviderPlugin interface {
    Plugin
    Handle(ctx context.Context, req *GatewayRequest, writer StreamWriter) (*GatewayResponse, error)
}

// SSE 流式 Provider（Host 管理连接）
type StreamingProviderPlugin interface {
    ProviderPlugin
    BuildStreamRequest(ctx context.Context, req *GatewayRequest) (*StreamRequest, error)
    OnSSELine(ctx context.Context, line []byte) ([][]byte, error)
    OnStreamEnd(ctx context.Context) (*GatewayResponse, error)
}

// 委托式流式 Provider（插件维护状态）
type StreamProviderPlugin interface {
    ProviderPlugin
    PrepareStream(ctx context.Context, req *GatewayRequest) (*StreamDelegate, error)
    OnSSELine(line []byte) ([]byte, error)
    OnStreamEnd() (*ProviderResultMetadata, error)
}
```

---

## 开发流程

```
创建项目 → 实现接口 → 编写 manifest.json → 编译 WASM → 签名 → 提交审核 → 发布
```

### manifest.json 格式

```json
{
  "id": "my-plugin",
  "version": "1.0.0",
  "runtime": "wasm",
  "plugin_api_version": "1.0.0",
  "sha256": "（编译后由工具自动填充）",
  "compatibility": {
    "min_plugin_api_version": "1.0.0",
    "max_plugin_api_version": "1.99.99"
  },
  "dependencies": [
    {
      "plugin_id": "some-other-plugin",
      "version_constraint": "^1.0.0",
      "optional": false
    }
  ]
}
```

**字段说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 插件唯一标识，全局唯一 |
| `version` | string | 是 | 语义化版本号 |
| `runtime` | string | 是 | 运行时类型，目前为 `"wasm"` |
| `plugin_api_version` | string | 是 | 插件编译时使用的 API 版本 |
| `sha256` | string | 是 | WASM 文件的 SHA256 哈希 |
| `compatibility` | object | 是 | 兼容性范围声明 |
| `dependencies` | array | 否 | 依赖的其他插件列表 |

---

## Host API 使用

插件通过 Host API 与运行时交互，需要在 manifest 中声明所需 capability。

| Capability | 说明 |
|-----------|------|
| `host.http.fetch` | 发起 HTTP 请求 |
| `host.kv.read` | 读取 KV 存储 |
| `host.kv.write` | 写入/删除 KV 存储 |
| `host.log.write` | 写日志 |
| `host.config.read` | 读取插件配置 |

详细用法参见 [Host API Reference](HOST-API-REFERENCE.md)。

---

## 常见问题 FAQ

**Q: 插件可以访问文件系统吗？**
A: 不可以。WASM 沙箱中没有文件系统访问权限，持久化数据请使用 KV Host API。

**Q: 如何调试插件？**
A: 使用 `host.log.write` 能力输出调试日志。日志有限速（默认 100 条/秒），避免在循环中大量打日志。

**Q: Body 大小有限制吗？**
A: 默认限制 2MB。超过此大小的请求/响应 Body 会被截断或拒绝。

**Q: StreamingProviderPlugin 和 StreamProviderPlugin 有什么区别？**
A: `StreamingProviderPlugin` 的 `OnSSELine` 接收 `context.Context` 参数，适合无状态转换；`StreamProviderPlugin` 不接收 context，适合在 `PrepareStream` 中初始化状态后在回调中使用。

**Q: 如何处理上游返回错误？**
A: `FetchStreaming` 在上游返回 HTTP >= 400 时会返回 `*HostHTTPError`，包含 `StatusCode` 和 `Body`，插件应据此返回合适的错误响应。

**Q: 插件更新后旧版本还可用吗？**
A: 是的，插件市场保留所有已发布版本，用户可以选择安装特定版本。
