# 插件最佳实践

## WASM 内存限制与管理

WASM 插件运行在沙箱环境中，内存资源有限：

- 避免在插件初始化时分配大量内存，按需分配
- 字符串拼接用 `strings.Builder` 代替 `+` 连接
- 处理完请求后及时释放临时缓冲区，不要在全局变量中缓存大量数据
- 避免使用 `map` 作为缓存——WASM 中 map 的内存开销比原生环境更大

```go
// 好：使用 Builder 拼接
var b strings.Builder
b.WriteString("data: ")
b.Write(chunk)
result := b.String()

// 避免：循环中反复拼接
s := ""
for _, chunk := range chunks {
    s += string(chunk) // 每次分配新内存
}
```

---

## Body 大小限制

默认请求/响应 Body 上限为 **2MB**。超出时行为取决于插件类型：

- `TransformPlugin`：Body 被截断，`TransformRequest/TransformResponse` 处理的是不完整数据
- `ProviderPlugin`：应通过 `StreamWriter` 分块写入，避免在内存中组装完整响应

```go
func (p *MyProvider) Handle(ctx context.Context, req *pluginapi.GatewayRequest, w pluginapi.StreamWriter) (*pluginapi.GatewayResponse, error) {
    if len(req.Body) > 2*1024*1024 {
        return &pluginapi.GatewayResponse{
            StatusCode: 413,
            Body:       []byte(`{"error":"request body too large"}`),
        }, nil
    }
    // ...
}
```

---

## 错误处理

### 返回结构化错误

使用 `PluginError` 返回带错误码的错误，便于调用方分类处理：

```go
return nil, &pluginapi.PluginError{
    Code:    "UPSTREAM_TIMEOUT",
    Message: "anthropic API did not respond within 30s",
    Cause:   err,
}
```

### 拦截器中的错误传播

拦截器中调用 `next` 失败时，决定是向上传播还是降级处理：

```go
func (p *RetryPlugin) Intercept(ctx context.Context, req *pluginapi.GatewayRequest, next pluginapi.Handler) (*pluginapi.GatewayResponse, error) {
    resp, err := next(ctx, req)
    if err != nil {
        // 重试一次
        resp, err = next(ctx, req)
    }
    return resp, err
}
```

### Host API 错误检查

检查 capability 被拒绝的情况，给出明确提示：

```go
resp, err := httpAPI.Fetch(pluginID, req)
if err != nil {
    var httpErr *pluginruntime.HostHTTPError
    if errors.As(err, &httpErr) {
        logAPI.Write(pluginID, LogLevelError,
            fmt.Sprintf("upstream %d: %s", httpErr.StatusCode, httpErr.Body))
    }
    return nil, err
}
```

---

## 流式处理注意事项

### OnSSELine 不要阻塞

`OnSSELine` 在流式读取循环中同步调用。如果回调阻塞，整个流会停顿，最终触发上游超时：

```go
// 好：快速返回
func (p *MyPlugin) OnSSELine(ctx context.Context, line []byte) ([][]byte, error) {
    transformed := bytes.Replace(line, []byte("gpt-4"), []byte("my-model"), 1)
    return [][]byte{transformed}, nil
}

// 避免：在回调中做网络请求或重计算
func (p *MyPlugin) OnSSELine(ctx context.Context, line []byte) ([][]byte, error) {
    resp, _ := httpAPI.Fetch(pluginID, req) // 阻塞！会导致流卡住
    return [][]byte{[]byte(resp.Body)}, nil
}
```

### 检查客户端断开

流式写入前检查客户端是否已断开，避免无效工作：

```go
func (p *MyProvider) Handle(ctx context.Context, req *pluginapi.GatewayRequest, w pluginapi.StreamWriter) (*pluginapi.GatewayResponse, error) {
    w.SetHeader("Content-Type", "text/event-stream")
    w.SetStatus(200)

    for chunk := range dataStream {
        if w.ClientGone() {
            break
        }
        w.WriteChunk(chunk)
        w.Flush()
    }
    w.Close()
    return nil, nil
}
```

### StreamWriter 状态机

严格遵循状态转换顺序，否则调用会返回错误：

```
idle → SetHeader/SetStatus（可多次）
idle → started（首次 WriteChunk）
started → WriteChunk/Flush（可多次）
started → closed（Close）
```

---

## 性能调优

### 避免大 JSON 解析

不要对整个 Body 做 `json.Unmarshal` 再 `json.Marshal`，特别是 Body 接近 2MB 时：

```go
// 好：只提取需要的字段
type partialReq struct {
    Model string `json:"model"`
}
var pr partialReq
json.Unmarshal(req.Body, &pr)

// 避免：解析整个请求再序列化回去
var full map[string]any
json.Unmarshal(req.Body, &full)
full["model"] = "new-model"
req.Body, _ = json.Marshal(full)
```

对简单的字段替换，考虑使用 `bytes.Replace`：

```go
req.Body = bytes.Replace(req.Body, []byte(`"gpt-4"`), []byte(`"gpt-4o"`), 1)
```

### 减少内存分配

- 复用 `[]byte` 缓冲区而非每次 `make`
- `OnSSELine` 返回的 `[]byte` 避免不必要的复制
- 使用 `sync.Pool` 管理频繁分配的临时对象（如果 WASM 运行时支持）

### KV 操作批量化

避免在请求处理路径中做多次 KV 读写。如果需要读取多个配置，用 `GetAll` 一次获取：

```go
// 好：一次获取所有配置
cfg, _ := configAPI.GetAll(pluginID)
apiKey := cfg["api_key"]
model := cfg["default_model"]

// 避免：多次调用
apiKey, _ := configAPI.Get(pluginID, "api_key")
model, _ := configAPI.Get(pluginID, "default_model")
timeout, _ := configAPI.Get(pluginID, "timeout")
```

---

## 安全考虑

### Token 不要泄露到响应

`ProviderContext` 中的 Token 仅用于构建上游请求，不要写入响应头或 Body：

```go
func (p *MyProvider) Handle(ctx context.Context, req *pluginapi.GatewayRequest, w pluginapi.StreamWriter) (*pluginapi.GatewayResponse, error) {
    pc, _ := pluginapi.GetProviderContext(req)

    // 好：Token 只用于上游请求
    upstreamReq := HTTPRequest{
        Headers: map[string]string{
            "Authorization": "Bearer " + pc.Token,
        },
    }

    // 禁止：Token 泄露到响应
    // resp.Headers["X-Debug-Token"] = []string{pc.Token}
}
```

### 日志中脱敏

日志中不要记录完整的 Token、API Key 等敏感信息：

```go
// 好：只记录前缀
logAPI.Write(pluginID, LogLevelDebug, "using token: "+pc.Token[:8]+"...")

// 避免：记录完整 Token
logAPI.Write(pluginID, LogLevelDebug, "token: "+pc.Token)
```

### 错误信息不暴露内部细节

返回给客户端的错误不要包含上游 URL、内部错误栈等信息：

```go
// 好：对外返回通用错误
return &pluginapi.GatewayResponse{
    StatusCode: 502,
    Body:       []byte(`{"error":"upstream service unavailable"}`),
}, nil

// 避免：暴露内部地址
return &pluginapi.GatewayResponse{
    StatusCode: 502,
    Body:       []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())),
}, nil
```

---

## 测试策略

### 单元测试

直接测试插件逻辑，mock 掉 Host API：

```go
func TestTransformRequest(t *testing.T) {
    p := &MyTransformPlugin{}
    req := &pluginapi.GatewayRequest{
        Method:  "POST",
        Headers: map[string][]string{"Content-Type": {"application/json"}},
        Body:    []byte(`{"model":"gpt-4"}`),
    }

    err := p.TransformRequest(context.Background(), req)
    assert.NoError(t, err)
    assert.Contains(t, string(req.Body), "gpt-4o")
}
```

### 拦截器链测试

验证拦截器正确调用 `next` 或短路返回：

```go
func TestInterceptor(t *testing.T) {
    p := &AuthPlugin{}
    called := false
    next := func(ctx context.Context, req *pluginapi.GatewayRequest) (*pluginapi.GatewayResponse, error) {
        called = true
        return &pluginapi.GatewayResponse{StatusCode: 200}, nil
    }

    // 无 Authorization 头 → 应短路
    req := &pluginapi.GatewayRequest{Headers: map[string][]string{}}
    resp, _ := p.Intercept(context.Background(), req, next)
    assert.Equal(t, 401, resp.StatusCode)
    assert.False(t, called)
}
```

### 流式插件测试

使用 mock StreamWriter 验证写入序列：

```go
type mockWriter struct {
    chunks [][]byte
    state  pluginapi.WriteState
}

func (m *mockWriter) WriteChunk(chunk []byte) error {
    m.chunks = append(m.chunks, append([]byte{}, chunk...))
    m.state = pluginapi.WriteStateStarted
    return nil
}
// ... 实现其他方法

func TestStreamProvider(t *testing.T) {
    p := &MyStreamProvider{}
    w := &mockWriter{state: pluginapi.WriteStateIdle}
    req := &pluginapi.GatewayRequest{Stream: true}

    p.Handle(context.Background(), req, w)
    assert.True(t, len(w.chunks) > 0)
}
```
