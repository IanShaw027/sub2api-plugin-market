# Host API Reference

Host API 是运行时暴露给插件的系统级能力接口。插件通过 manifest 声明所需 capability，运行时在调用前执行权限检查。

## Capabilities 枚举

| Capability | 值 | 说明 |
|-----------|---|------|
| HTTP Fetch | `host.http.fetch` | 发起 HTTP/HTTPS 请求（含流式） |
| KV Read | `host.kv.read` | 读取 KV 存储、列出键 |
| KV Write | `host.kv.write` | 写入/删除 KV 存储 |
| Log Write | `host.log.write` | 写入日志 |
| Config Read | `host.config.read` | 读取插件配置 |

**声明方式：** 在 manifest 的 capabilities 字段中列出所需能力。运行时采用双重检查：

1. 插件是否声明了该 capability
2. 该 capability 是否在全局允许列表中

两项检查均通过后才允许调用。未声明的 capability 调用会返回 `CapabilityDeniedError`。

---

## HTTP Host API

需要 capability：`host.http.fetch`

### Fetch — 同步 HTTP 请求

```go
func (h *HostAPIHTTP) Fetch(pluginID string, req HTTPRequest) (HTTPResponse, error)
```

**请求参数：**

```go
type HTTPRequest struct {
    Method  HTTPMethod        // GET, POST, PUT, DELETE, PATCH
    URL     string            // 完整 URL
    Headers map[string]string // 请求头
    Body    string            // 请求体
    Timeout time.Duration     // 超时时间（0 使用默认 30s）
}
```

**响应：**

```go
type HTTPResponse struct {
    StatusCode int               // HTTP 状态码
    Headers    map[string]string // 响应头（每个 key 取第一个值）
    Body       string            // 响应体
}
```

**行为：**
- 执行前检查 `host.http.fetch` capability
- `Method` 必须是 `GET/POST/PUT/DELETE/PATCH` 之一，否则返回 `ErrInvalidHTTPMethod`
- `URL` 不能为空，否则返回 `ErrInvalidURL`
- 默认超时 30 秒，可通过 `Timeout` 字段覆盖

**示例：**

```go
resp, err := httpAPI.Fetch(pluginID, HTTPRequest{
    Method:  HTTPMethodPOST,
    URL:     "https://api.example.com/v1/chat",
    Headers: map[string]string{
        "Authorization": "Bearer " + token,
        "Content-Type":  "application/json",
    },
    Body:    `{"model": "gpt-4", "messages": []}`,
    Timeout: 60 * time.Second,
})
```

### FetchStreaming — 流式 HTTP 请求

```go
func (h *HostAPIHTTP) FetchStreaming(ctx context.Context, pluginID string, req *HTTPRequest, onLine SSELineCallback) error
```

**回调类型：**

```go
type SSELineCallback func(line []byte) error
```

**行为：**
- 建立 HTTP 连接后逐行扫描响应体，每行调用 `onLine` 回调
- `ctx` 控制流的生命周期，取消 ctx 会在下一行边界返回 `ctx.Err()`
- 上游返回 HTTP >= 400 时，返回 `*HostHTTPError`（含 StatusCode 和 Body）
- 默认流超时 5 分钟（可通过 `SetStreamClient` 自定义）
- 行缓冲区最大 1MB

**错误类型：**

```go
type HostHTTPError struct {
    StatusCode int    // 上游 HTTP 状态码
    Body       []byte // 上游响应体
}
```

---

## KV Host API

按 pluginID 做命名空间隔离，每个插件只能访问自己的数据。

### Read — 读取值

```go
func (h *HostAPIKV) Read(pluginID, key string) (string, error)
```

- 需要 capability：`host.kv.read`
- key 不能为空，否则返回 `ErrInvalidKey`
- key 不存在时返回 `ErrKeyNotFound`

### Write — 写入值

```go
func (h *HostAPIKV) Write(pluginID, key, value string) error
```

- 需要 capability：`host.kv.write`
- key 不能为空
- key 已存在时覆盖旧值

### Delete — 删除键

```go
func (h *HostAPIKV) Delete(pluginID, key string) error
```

- 需要 capability：`host.kv.write`（删除复用写权限）
- key 不存在时返回 `ErrKeyNotFound`

### List — 列出所有键

```go
func (h *HostAPIKV) List(pluginID string) ([]string, error)
```

- 需要 capability：`host.kv.read`
- 返回当前插件命名空间下的所有键名
- 命名空间不存在时返回空切片

**示例：**

```go
// 存储用户偏好
err := kvAPI.Write(pluginID, "user:123:lang", "zh-CN")

// 读取
lang, err := kvAPI.Read(pluginID, "user:123:lang")

// 列出所有键
keys, err := kvAPI.List(pluginID)

// 删除
err = kvAPI.Delete(pluginID, "user:123:lang")
```

---

## Log Host API

需要 capability：`host.log.write`

### Write — 写入日志

```go
func (h *HostAPILog) Write(pluginID string, level LogLevel, message string) error
```

**日志级别：**

| 级别 | 值 | 用途 |
|------|---|------|
| Debug | `debug` | 开发调试信息 |
| Info | `info` | 常规运行信息 |
| Warn | `warn` | 警告但不影响功能 |
| Error | `error` | 错误需要关注 |

**限速机制：**
- 默认每个插件每秒最多 100 条日志
- 超限后返回 `ErrLogRateLimited`，该日志被丢弃
- 可通过 `DroppedLogs(pluginID)` 查询被丢弃的日志数量
- 限速窗口每秒重置

**日志条目结构：**

```go
type LogEntry struct {
    PluginID  string    // 插件 ID
    Level     LogLevel  // 日志级别
    Message   string    // 日志内容
    Timestamp time.Time // UTC 时间戳（自动填充）
}
```

**示例：**

```go
logAPI.Write(pluginID, LogLevelInfo, "request processed successfully")
logAPI.Write(pluginID, LogLevelError, "upstream timeout: "+err.Error())
```

---

## Config Host API

需要 capability：`host.config.read`

配置按 pluginID 隔离，由管理侧通过 `SetPluginConfig` 注入，插件只能读取。

### Get — 读取单个配置项

```go
func (h *HostAPIConfig) Get(pluginID, key string) (string, error)
```

- 插件无配置时返回错误 `no config for plugin "xxx"`
- key 不存在时返回错误 `config key "xxx" not found for plugin "xxx"`

### GetAll — 读取全部配置

```go
func (h *HostAPIConfig) GetAll(pluginID string) (map[string]string, error)
```

- 返回该插件所有配置的只读副本
- 插件无配置时返回空 map（不报错）

**示例：**

```go
// 读取单个配置
apiKey, err := configAPI.Get(pluginID, "api_key")

// 读取所有配置
allCfg, err := configAPI.GetAll(pluginID)
for k, v := range allCfg {
    logAPI.Write(pluginID, LogLevelDebug, "config: "+k+"="+v)
}
```

---

## 错误处理

所有 Host API 共享一致的错误模式：

| 错误 | 触发条件 |
|------|---------|
| `CapabilityDeniedError` | capability 未声明或不在允许列表 |
| `ErrInvalidHTTPMethod` | HTTP 方法不在 GET/POST/PUT/DELETE/PATCH 中 |
| `ErrInvalidURL` | URL 为空 |
| `ErrInvalidKey` | KV 操作的 key 为空 |
| `ErrKeyNotFound` | KV 读取/删除不存在的 key |
| `ErrInvalidLogLevel` | 日志级别不在 debug/info/warn/error 中 |
| `ErrLogRateLimited` | 日志写入超过限速 |
| `*HostHTTPError` | 上游 HTTP 返回 >= 400 |

`CapabilityDeniedError` 支持 `errors.Is` 匹配：

```go
if errors.Is(err, pluginruntime.ErrCapabilityDenied) {
    // 权限被拒绝
}
if errors.Is(err, pluginruntime.ErrCapabilityNotDeclared) {
    // 插件未声明该 capability
}
```
