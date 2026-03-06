# Plugin API Reference

## 基础接口

### Plugin

所有插件类型的公共接口。

```go
type Plugin interface {
    Metadata() Metadata
}
```

#### Metadata

```go
type Metadata struct {
    Name        string
    Version     string
    Description string
}
```

返回插件的基础信息，运行时在加载阶段调用一次。

---

## 请求/响应类型

### GatewayRequest

网关请求结构，贯穿整个插件处理链路。

```go
type GatewayRequest struct {
    Method   string              // HTTP 方法：GET, POST, PUT, DELETE, PATCH
    Path     string              // 请求路径
    Headers  map[string][]string // 请求头（多值）
    Query    map[string][]string // 查询参数（多值）
    Body     []byte              // 请求体（最大 2MB）
    Stream   bool                // 是否为流式请求
    Metadata map[string]any      // 内部元数据传递（如 ProviderContext）
}
```

`Metadata` 用于核心与插件之间传递上下文信息，键 `"provider_context"` 存放 `ProviderContext`。

### GatewayResponse

网关响应结构。

```go
type GatewayResponse struct {
    StatusCode int                 // HTTP 状态码
    Headers    map[string][]string // 响应头
    Body       []byte              // 响应体
    Metadata   map[string]any      // 内部元数据（如 ProviderResultMetadata）
}
```

`Metadata` 键 `"provider_result"` 存放 `ProviderResultMetadata`，供核心做计费和监控。

---

## 拦截器插件

### InterceptorPlugin

```go
type InterceptorPlugin interface {
    Plugin
    Intercept(ctx context.Context, req *GatewayRequest, next Handler) (*GatewayResponse, error)
}
```

**Intercept(ctx, req, next)**
- `ctx`：请求上下文，携带超时和取消信号
- `req`：当前网关请求，可修改后传递给下游
- `next`：下游处理函数，调用 `next(ctx, req)` 继续链路
- 返回：响应对象或错误。不调用 `next` 即短路返回

```go
type Handler func(ctx context.Context, req *GatewayRequest) (*GatewayResponse, error)
```

---

## 转换器插件

### TransformPlugin

```go
type TransformPlugin interface {
    Plugin
    TransformRequest(ctx context.Context, req *GatewayRequest) error
    TransformResponse(ctx context.Context, resp *GatewayResponse) error
}
```

**TransformRequest(ctx, req)**
- 在请求发往上游前调用，原地修改 `req` 的字段（Headers、Body 等）
- 返回 error 时请求被中止

**TransformResponse(ctx, resp)**
- 在响应返回客户端前调用，原地修改 `resp` 的字段
- 返回 error 时使用错误响应替代

### ChunkTransformer

`TransformPlugin` 的可选扩展，用于流式 SSE 场景逐行转换。

```go
type ChunkTransformer interface {
    TransformChunk(chunk []byte) ([]byte, error)
}
```

**TransformChunk(chunk)**
- 在 Provider 的 `OnSSELine` 之后被链式调用
- 输入：一行 SSE 数据
- 返回：转换后的数据。返回 `nil` 表示过滤掉该行

---

## 提供者插件

### ProviderPlugin

基础提供者接口，直接产出响应。

```go
type ProviderPlugin interface {
    Plugin
    Handle(ctx context.Context, req *GatewayRequest, writer StreamWriter) (*GatewayResponse, error)
}
```

**Handle(ctx, req, writer)**
- 非流式场景：忽略 `writer`，直接返回 `*GatewayResponse`
- 流式场景：通过 `writer` 写入 SSE 数据，返回的 `*GatewayResponse` 可为 `nil`

### StreamWriter

运行时暴露给插件的流式写入接口。

```go
type StreamWriter interface {
    State() WriteState
    SetHeader(key, value string) error
    SetStatus(code int) error
    WriteChunk(chunk []byte) error
    Flush() error
    Close() error
    ClientGone() bool
    Done() <-chan struct{}
}
```

| 方法 | 说明 |
|------|------|
| `State()` | 返回当前状态：`idle` / `started` / `closed` |
| `SetHeader(k, v)` | 设置响应头，必须在 `WriteChunk` 之前调用 |
| `SetStatus(code)` | 设置 HTTP 状态码，必须在 `WriteChunk` 之前调用 |
| `WriteChunk(chunk)` | 写入一块数据，首次调用后状态变为 `started` |
| `Flush()` | 刷新缓冲区，确保数据发送到客户端 |
| `Close()` | 关闭写入器，状态变为 `closed` |
| `ClientGone()` | 返回客户端是否已断开 |
| `Done()` | 返回客户端断开时关闭的 channel，遵循 `context.Done()` 约定 |

**状态机：** `idle` → `started`（首次 WriteChunk） → `closed`（Close）

### StreamingProviderPlugin

Host 管理 SSE 连接，插件提供逐行转换逻辑。

```go
type StreamingProviderPlugin interface {
    ProviderPlugin
    BuildStreamRequest(ctx context.Context, req *GatewayRequest) (*StreamRequest, error)
    OnSSELine(ctx context.Context, line []byte) ([][]byte, error)
    OnStreamEnd(ctx context.Context) (*GatewayResponse, error)
}
```

**BuildStreamRequest(ctx, req)** — 构造上游 HTTP 请求，Host 据此建立 SSE 连接

```go
type StreamRequest struct {
    Method  string
    URL     string
    Headers map[string][]string
    Body    []byte
}
```

**OnSSELine(ctx, line)** — 每收到一行 SSE 数据时调用。返回 `[][]byte` 可一对多映射，返回 `nil` 过滤该行

**OnStreamEnd(ctx)** — 上游流结束时调用，返回包含最终元数据的响应

### StreamProviderPlugin

委托式流式 Provider，插件维护请求级内部状态。

```go
type StreamProviderPlugin interface {
    ProviderPlugin
    PrepareStream(ctx context.Context, req *GatewayRequest) (*StreamDelegate, error)
    OnSSELine(line []byte) ([]byte, error)
    OnStreamEnd() (*ProviderResultMetadata, error)
}
```

**PrepareStream(ctx, req)** — 准备上游请求并初始化内部状态

```go
type StreamDelegate struct {
    URL     string
    Method  string
    Headers map[string][]string
}
```

**OnSSELine(line)** — 逐行转换，无 context 参数（状态在 PrepareStream 中初始化）

**OnStreamEnd()** — 流结束，返回计费和监控用的元数据

---

## Provider 上下文类型

### ProviderContext

核心在账号选择后注入到 `GatewayRequest.Metadata["provider_context"]`，供 Provider 插件构建上游请求。

```go
type ProviderContext struct {
    AccountID   string            // 账号 ID
    Platform    string            // 平台："anthropic", "openai", "gemini" 等
    AccountType string            // 账号类型："oauth", "apikey", "session_key" 等
    Token       string            // 已刷新的访问令牌
    TokenType   string            // 令牌类型："bearer", "api_key", "session_key"
    BaseURL     string            // 上游 API 基础 URL
    BaseURLs    []string          // 多 URL 容灾（可选）
    ProxyURL    string            // 代理 URL（可选）
    MappedModel   string          // 核心映射后的模型名（发往上游）
    OriginalModel string          // 原始请求模型名（用于计费）
    PlatformSpecific map[string]any // 平台特定字段
}
```

提取方法：

```go
pc, ok := pluginapi.GetProviderContext(req)
if !ok {
    return nil, fmt.Errorf("missing provider context")
}
```

### ProviderResultMetadata

Provider 插件通过 `GatewayResponse.Metadata["provider_result"]` 返回给核心。

```go
type ProviderResultMetadata struct {
    Usage        UsageInfo // Token 用量
    Model        string    // 实际使用的模型
    RequestID    string    // 上游请求 ID（审计用）
    FirstTokenMs *int      // 首 Token 延迟（毫秒）
    Failover     bool      // 是否需要故障转移
    ImageCount   int       // 图片数量（图片计费用）
    ImageSize    string    // 图片尺寸
}

type UsageInfo struct {
    InputTokens         int
    OutputTokens        int
    CacheCreationTokens int // 缓存创建消耗（可选）
    CacheReadTokens     int // 缓存读取消耗（可选）
}
```

设置方法：

```go
pluginapi.SetProviderResult(resp, &pluginapi.ProviderResultMetadata{
    Usage: pluginapi.UsageInfo{InputTokens: 100, OutputTokens: 50},
    Model: "claude-3.5-sonnet",
})
```

---

## 错误类型

### PluginError

统一的插件错误结构，支持错误链。

```go
type PluginError struct {
    Code    string // 错误码
    Message string // 错误描述
    Cause   error  // 原始错误
}
```

实现了 `error` 和 `Unwrap()` 接口。`Error()` 输出格式为 `"code: message"`。
