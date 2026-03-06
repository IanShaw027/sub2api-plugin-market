# Semver 兼容性匹配规则

## 查询参数

`GET /api/v1/plugins/:name/versions?compatible_with=1.2.0`

## 匹配算法

```
版本 V 与查询值 X 兼容，当且仅当:

  min_api_version <= X  且  (max_api_version == "" 或 max_api_version >= X)
```

### 实现

```go
// plugin_repository.go
if compatibleWith != "" {
    query = query.Where(pluginversion.MinAPIVersionLTE(compatibleWith))
}
```

当前实现仅检查 `min_api_version <= X`（字符串比较）。`max_api_version` 字段尚未强制校验。

## 版本比较语义

使用字符串字典序比较（非 Go `semver` 包）。对于标准 semver 格式 `X.Y.Z`:

| min_api_version | max_api_version | compatible_with=1.2.0 | 结果 |
|-----------------|-----------------|----------------------|------|
| 1.0.0 | 2.0.0 | 1.2.0 | ✅ 兼容 |
| 1.0.0 | (空) | 1.2.0 | ✅ 兼容 |
| 1.3.0 | 2.0.0 | 1.2.0 | ❌ min > X |
| 1.0.0 | 1.1.0 | 1.2.0 | ❌ max < X |
| 1.2.0 | 1.2.0 | 1.2.0 | ✅ 精确匹配 |

## 边界用例

| 场景 | 行为 |
|------|------|
| `compatible_with` 为空 | 返回所有 published 版本 |
| 插件未设置 `min_api_version` | 默认 `1.0.0` |
| 插件未设置 `max_api_version` | 视为无上限 |
| 预发布版本 `1.0.0-beta` | 字符串比较: `1.0.0-beta` < `1.0.0`（按 ASCII） |

## 后续改进

建议引入 Go `golang.org/x/mod/semver` 包做标准比较，正确处理:
- 预发布版本排序（`1.0.0-alpha` < `1.0.0-beta` < `1.0.0`）
- Build metadata 忽略（`1.0.0+build` == `1.0.0`）
