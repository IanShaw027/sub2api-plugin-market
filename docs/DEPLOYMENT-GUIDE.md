# 部署指南

## 部署顺序

### Phase 0-1: 先 Market → 再 Sub2API

```
1. 备份 Market DB
2. 执行 DB 迁移: psql "$DB_URL" -f migrations/000001_initial_schema.up.sql
3. 部署 Market 新版本
4. 验证: curl market:8081/health → {"status":"ok"}
5. 部署 Sub2API 新版本（启用 PLUGIN_DISPATCH_ENABLED=false）
6. 验证: 原有请求链路正常
```

### Phase 2-3: 先 Sub2API → 再安装插件

```
1. 部署 Sub2API（PLUGIN_DISPATCH_ENABLED=true）
2. 验证: 无插件时 fallback 到内置 Service
3. 在 Market 审核/发布插件
4. Sub2API 下载安装插件
5. 验证: 请求走插件路径
```

## 灰度策略

每个 Provider 按以下步骤上线:

```
Shadow (1周)
├── PLUGIN_TRAFFIC_CLAUDE=shadow:0
├── 内置结果响应客户端,插件结果丢弃
└── 对比日志确认一致性

Canary 10% (3天)
├── PLUGIN_TRAFFIC_CLAUDE=canary:10
└── 监控 Usage 偏差 / 错误率

Canary 50% (3天)
├── PLUGIN_TRAFFIC_CLAUDE=canary:50
└── 持续监控

Full 100%
├── PLUGIN_TRAFFIC_CLAUDE=full:0
└── 可随时回滚到 disabled
```

## 回滚预案

### 快速回滚（< 5min）

```bash
# 方式 1: 禁用插件调度
export PLUGIN_DISPATCH_ENABLED=false
# 重启 Sub2API

# 方式 2: 禁用单个 Provider
export PLUGIN_TRAFFIC_CLAUDE=disabled:0
# 无需重启（如支持热配置）
```

### 完整回滚

```bash
# 1. 回滚二进制到上一版本
# 2. 回滚数据库
psql "$DB_URL" -f migrations/000001_initial_schema.down.sql
# 3. 重启服务
```

## Health Check

```bash
# Market
curl http://market:8081/health
# 预期: {"status":"ok","db":"ok","storage":"ok"}

# Sub2API
curl http://sub2api:PORT/health
# 检查 DispatchRuntime 状态
```
