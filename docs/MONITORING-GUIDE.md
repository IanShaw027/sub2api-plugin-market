# 监控告警指南

本文档定义 Plugin Market（控制平面）和 Sub2API（数据平面）的监控指标、告警规则及 Grafana Dashboard 模板。

---

## 1. Market 监控指标

### 1.1 Submission（提交审核）

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `market_submission_total` | Counter | 提交总量，按 `type`（new_plugin/new_version）和 `status` 分 |
| `market_submission_review_duration_seconds` | Histogram | 从提交到审核完成的耗时 |
| `market_submission_pending_count` | Gauge | 当前待审核数量 |

**关注点**：
- 待审核堆积：`market_submission_pending_count > 20` 持续 1 小时
- 审核延迟：`market_submission_review_duration_seconds` p95 > 24h

### 1.2 SyncJob（同步任务）

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `market_syncjob_total` | Counter | 同步任务总量，按 `trigger_type` 和 `status` 分 |
| `market_syncjob_duration_seconds` | Histogram | 同步任务执行耗时 |
| `market_syncjob_consecutive_failures` | Gauge | 连续失败次数，按 `plugin_id` 分 |

**关注点**：
- 成功率：`rate(market_syncjob_total{status="success"}[5m]) / rate(market_syncjob_total[5m])` 应 > 95%
- 连续失败：单个插件连续失败 >= 3 次需告警

### 1.3 下载（Download）

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `market_download_total` | Counter | 下载总量，按 `success` 分 |
| `market_download_qps` | Gauge | 当前下载 QPS |
| `market_download_duration_seconds` | Histogram | 下载链路耗时（含验签） |

**关注点**：
- 下载失败率：`rate(market_download_total{success="false"}[5m]) / rate(market_download_total[5m])` 应 < 1%
- 下载延迟：p99 < 5s（含预签名 URL 生成）

---

## 2. Sub2API 插件监控

### 2.1 Dispatch 延迟

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `sub2api_dispatch_duration_seconds` | Histogram | Dispatcher 路由+调度延迟 |
| `sub2api_plugin_exec_duration_seconds` | Histogram | 插件 WASM 执行延迟，按 `plugin_name` 分 |

**关键分位数**：

| 分位 | 目标值 | 告警阈值 |
|------|--------|----------|
| p50 | < 10ms | — |
| p95 | < 50ms | > 100ms |
| p99 | < 200ms | > 500ms |

### 2.2 插件错误率

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `sub2api_plugin_error_total` | Counter | 插件执行错误，按 `plugin_name`、`error_type` 分 |
| `sub2api_plugin_request_total` | Counter | 插件请求总量，按 `plugin_name` 分 |
| `sub2api_plugin_error_rate` | Gauge | 错误率（由 Recording Rule 计算） |

错误类型分类：
- `exec_timeout`：WASM 执行超时
- `exec_panic`：WASM 运行时 panic
- `oom`：内存超限
- `validation`：输入/输出校验失败
- `internal`：其他内部错误

### 2.3 熔断状态

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `sub2api_circuit_breaker_state` | Gauge | 熔断器状态（0=closed, 1=half-open, 2=open），按 `plugin_name` 分 |
| `sub2api_circuit_breaker_trip_total` | Counter | 熔断触发次数 |

### 2.4 WASM 内存使用

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `sub2api_wasm_memory_bytes` | Gauge | 当前 WASM 实例内存使用，按 `plugin_name` 分 |
| `sub2api_wasm_memory_limit_bytes` | Gauge | WASM 内存上限 |
| `sub2api_wasm_oom_total` | Counter | OOM 事件次数 |

---

## 3. 灰度期专用指标

灰度期同时运行内置实现和插件实现，需对比两者行为。

### 3.1 Usage 偏差率

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `sub2api_canary_builtin_usage` | Counter | 内置实现 usage 消耗 |
| `sub2api_canary_plugin_usage` | Counter | 插件实现 usage 消耗 |
| `sub2api_canary_usage_deviation_rate` | Gauge | 偏差率 = abs(plugin - builtin) / builtin |

偏差率计算（Prometheus Recording Rule）：

```yaml
groups:
  - name: canary_rules
    rules:
      - record: sub2api_canary_usage_deviation_rate
        expr: |
          abs(
            rate(sub2api_canary_plugin_usage[5m])
            - rate(sub2api_canary_builtin_usage[5m])
          ) / rate(sub2api_canary_builtin_usage[5m])
```

### 3.2 延迟对比

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `sub2api_canary_builtin_duration_seconds` | Histogram | 内置实现延迟 |
| `sub2api_canary_plugin_duration_seconds` | Histogram | 插件实现延迟 |

对比看板应展示：
- p50/p95/p99 延迟并排对比
- 延迟差值趋势
- 超时率对比

---

## 4. 告警规则

### 4.1 告警级别定义

| 级别 | 含义 | 响应时间 | 通知方式 |
|------|------|----------|----------|
| **P0** | 系统不可用 / 数据丢失风险 | 5 分钟内 | 电话 + IM |
| **P1** | 核心功能受损 | 15 分钟内 | IM + 邮件 |
| **P2** | 非核心功能异常 | 1 小时内 | IM |
| **P3** | 需关注但不紧急 | 下个工作日 | 邮件 |

### 4.2 告警规则定义

```yaml
groups:
  - name: plugin_market_alerts
    rules:
      # P0: WASM OOM
      - alert: WasmOOM
        expr: increase(sub2api_wasm_oom_total[5m]) > 0
        for: 0m
        labels:
          severity: P0
        annotations:
          summary: "WASM 插件 OOM"
          description: "插件 {{ $labels.plugin_name }} 发生 OOM，需立即处理"

      # P1: 插件错误率 >1%
      - alert: PluginHighErrorRate
        expr: |
          rate(sub2api_plugin_error_total[5m])
          / rate(sub2api_plugin_request_total[5m])
          > 0.01
        for: 5m
        labels:
          severity: P1
        annotations:
          summary: "插件错误率超过 1%"
          description: "插件 {{ $labels.plugin_name }} 错误率 {{ $value | humanizePercentage }}，持续 5 分钟"

      # P1: 灰度期 Usage 偏差 >0.5%
      - alert: CanaryUsageDeviation
        expr: sub2api_canary_usage_deviation_rate > 0.005
        for: 10m
        labels:
          severity: P1
        annotations:
          summary: "灰度 Usage 偏差超过 0.5%"
          description: "内置 vs 插件 Usage 偏差率 {{ $value | humanizePercentage }}，持续 10 分钟，需检查插件实现正确性"

      # P2: SyncJob 连续失败 3 次
      - alert: SyncJobConsecutiveFailures
        expr: market_syncjob_consecutive_failures >= 3
        for: 0m
        labels:
          severity: P2
        annotations:
          summary: "SyncJob 连续失败 3 次"
          description: "插件 {{ $labels.plugin_id }} 同步任务连续失败 {{ $value }} 次"

      # P2: 下载失败率高
      - alert: DownloadHighFailureRate
        expr: |
          rate(market_download_total{success="false"}[5m])
          / rate(market_download_total[5m])
          > 0.05
        for: 5m
        labels:
          severity: P2
        annotations:
          summary: "下载失败率超过 5%"
          description: "下载失败率 {{ $value | humanizePercentage }}"

      # P2: Dispatch 延迟过高
      - alert: DispatchHighLatency
        expr: histogram_quantile(0.99, rate(sub2api_dispatch_duration_seconds_bucket[5m])) > 0.5
        for: 5m
        labels:
          severity: P2
        annotations:
          summary: "Dispatch p99 延迟超过 500ms"
          description: "当前 p99 延迟 {{ $value }}s"

      # P2: 熔断器打开
      - alert: CircuitBreakerOpen
        expr: sub2api_circuit_breaker_state == 2
        for: 1m
        labels:
          severity: P2
        annotations:
          summary: "插件熔断器打开"
          description: "插件 {{ $labels.plugin_name }} 熔断器处于 Open 状态"

      # P3: 待审核堆积
      - alert: SubmissionBacklog
        expr: market_submission_pending_count > 20
        for: 1h
        labels:
          severity: P3
        annotations:
          summary: "待审核提交堆积"
          description: "当前待审核数量 {{ $value }}，超过 20 且持续 1 小时"
```

### 4.3 告警规则速查表

| 告警 | 条件 | 级别 | 持续时间 |
|------|------|------|----------|
| WASM OOM | `wasm_oom_total` 增长 > 0 | **P0** | 立即 |
| 插件错误率高 | 错误率 > 1% | **P1** | 5 分钟 |
| 灰度 Usage 偏差 | 偏差率 > 0.5% | **P1** | 10 分钟 |
| SyncJob 连续失败 | 连续失败 >= 3 次 | **P2** | 立即 |
| 下载失败率高 | 失败率 > 5% | **P2** | 5 分钟 |
| Dispatch 延迟高 | p99 > 500ms | **P2** | 5 分钟 |
| 熔断器打开 | state = open | **P2** | 1 分钟 |
| 审核堆积 | pending > 20 | **P3** | 1 小时 |

---

## 5. Grafana Dashboard 模板（简化版）

以下 JSON 可直接导入 Grafana（Dashboard → Import → Paste JSON）：

```json
{
  "dashboard": {
    "title": "Sub2API Plugin Market Overview",
    "uid": "sub2api-plugin-market",
    "timezone": "browser",
    "refresh": "30s",
    "time": { "from": "now-1h", "to": "now" },
    "panels": [
      {
        "title": "下载 QPS",
        "type": "timeseries",
        "gridPos": { "h": 8, "w": 12, "x": 0, "y": 0 },
        "targets": [
          {
            "expr": "rate(market_download_total[5m])",
            "legendFormat": "{{ success }}"
          }
        ]
      },
      {
        "title": "插件错误率",
        "type": "timeseries",
        "gridPos": { "h": 8, "w": 12, "x": 12, "y": 0 },
        "targets": [
          {
            "expr": "rate(sub2api_plugin_error_total[5m]) / rate(sub2api_plugin_request_total[5m])",
            "legendFormat": "{{ plugin_name }}"
          }
        ]
      },
      {
        "title": "Dispatch 延迟分位数",
        "type": "timeseries",
        "gridPos": { "h": 8, "w": 12, "x": 0, "y": 8 },
        "targets": [
          {
            "expr": "histogram_quantile(0.50, rate(sub2api_dispatch_duration_seconds_bucket[5m]))",
            "legendFormat": "p50"
          },
          {
            "expr": "histogram_quantile(0.95, rate(sub2api_dispatch_duration_seconds_bucket[5m]))",
            "legendFormat": "p95"
          },
          {
            "expr": "histogram_quantile(0.99, rate(sub2api_dispatch_duration_seconds_bucket[5m]))",
            "legendFormat": "p99"
          }
        ]
      },
      {
        "title": "WASM 内存使用",
        "type": "timeseries",
        "gridPos": { "h": 8, "w": 12, "x": 12, "y": 8 },
        "targets": [
          {
            "expr": "sub2api_wasm_memory_bytes",
            "legendFormat": "{{ plugin_name }}"
          },
          {
            "expr": "sub2api_wasm_memory_limit_bytes",
            "legendFormat": "limit"
          }
        ]
      },
      {
        "title": "熔断器状态",
        "type": "stat",
        "gridPos": { "h": 4, "w": 6, "x": 0, "y": 16 },
        "targets": [
          {
            "expr": "sub2api_circuit_breaker_state",
            "legendFormat": "{{ plugin_name }}"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "mappings": [
              { "type": "value", "options": { "0": { "text": "Closed", "color": "green" } } },
              { "type": "value", "options": { "1": { "text": "Half-Open", "color": "yellow" } } },
              { "type": "value", "options": { "2": { "text": "Open", "color": "red" } } }
            ]
          }
        }
      },
      {
        "title": "SyncJob 成功率",
        "type": "gauge",
        "gridPos": { "h": 4, "w": 6, "x": 6, "y": 16 },
        "targets": [
          {
            "expr": "rate(market_syncjob_total{status=\"success\"}[5m]) / rate(market_syncjob_total[5m])"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "min": 0,
            "max": 1,
            "thresholds": {
              "steps": [
                { "value": 0, "color": "red" },
                { "value": 0.9, "color": "yellow" },
                { "value": 0.95, "color": "green" }
              ]
            }
          }
        }
      },
      {
        "title": "待审核数量",
        "type": "stat",
        "gridPos": { "h": 4, "w": 6, "x": 12, "y": 16 },
        "targets": [
          {
            "expr": "market_submission_pending_count"
          }
        ]
      },
      {
        "title": "灰度 Usage 偏差率",
        "type": "timeseries",
        "gridPos": { "h": 4, "w": 6, "x": 18, "y": 16 },
        "targets": [
          {
            "expr": "sub2api_canary_usage_deviation_rate",
            "legendFormat": "deviation"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "percentunit",
            "thresholds": {
              "steps": [
                { "value": 0, "color": "green" },
                { "value": 0.005, "color": "red" }
              ]
            }
          }
        }
      }
    ]
  }
}
```

### Dashboard 面板说明

| 面板 | 数据源 | 用途 |
|------|--------|------|
| 下载 QPS | `market_download_total` | 监控下载流量趋势 |
| 插件错误率 | `plugin_error_total / plugin_request_total` | 按插件追踪错误率 |
| Dispatch 延迟分位数 | `dispatch_duration_seconds` | 追踪 p50/p95/p99 延迟 |
| WASM 内存使用 | `wasm_memory_bytes` | 内存使用 vs 上限 |
| 熔断器状态 | `circuit_breaker_state` | 实时熔断状态 |
| SyncJob 成功率 | `syncjob_total` | 同步任务健康度 |
| 待审核数量 | `submission_pending_count` | 审核工作量 |
| 灰度偏差率 | `canary_usage_deviation_rate` | 灰度期核心指标 |
