# 错误码注册表（v1）

- 状态：Active
- 更新时间：2026-03-04
- 适用范围：`sub2api-plugin-market` API

## 1. 注册规则

1. 错误码全局唯一，不允许同码不同义。
2. 文档与代码必须同时更新。
3. 新增错误码必须附带：触发条件、调用方处理建议、可观测字段。

## 2. 当前错误码

| 代码 | 名称 | 含义 | 来源 |
|---|---|---|---|
| `0` | success | 成功 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:19) |
| `1001` | invalid_param | 参数错误 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:43) |
| `1002` | not_found | 资源不存在 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:44) |
| `1003` | internal_error | 服务器内部错误 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:45) |
| `1004` | database_error | 数据库错误 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:46) |
| `1005` | storage_error | 存储服务错误 | [response.go](/Users/ianshaw/Documents/fork/sub2api-plugin-market/internal/api/v1/handler/response.go:47) |

## 3. 调用方建议

1. `1001`：提示用户检查输入。
2. `1002`：展示“资源不存在”并允许重试查询。
3. `1003`：记录 `request_id`（若有）并重试或降级。
4. `1004`：短暂重试 + 告警数据库健康。
5. `1005`：降级下载链路并告警存储服务。

## 4. 与契约同步

1. OpenAPI：`openapi/plugin-market-v1.yaml`
2. API 文档：`docs/API.md`
3. 代码常量：`internal/api/v1/handler/response.go`
