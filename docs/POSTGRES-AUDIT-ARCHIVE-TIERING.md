# PostgreSQL 审计归档冷热分层策略

- 版本：v1
- 日期：2026-03-04
- 范围：`sub2api` 控制平面审计表 `plugin_market_audit_events`

## 1. 目标

将审计数据分层为热数据与冷数据，兼顾：
- 控制平面查询性能
- 长周期审计留存
- 运维可执行性（脚本化、可 dry-run）

## 2. 分层定义

- 热层（Hot）
  - 存储：PostgreSQL 主表 `plugin_market_audit_events`
  - 数据窗口：近 90 天（可配置）
  - 用途：管理后台在线查询与分页过滤

- 冷层（Cold）
  - 存储：本地归档文件，或对象存储（S3）
  - 数据窗口：超过热层阈值的历史数据
  - 用途：审计追溯、合规留存、离线分析

## 3. 当前可执行实现

归档脚本：`scripts/archive-audit-logs.sh`
恢复脚本：`scripts/restore-audit-logs.sh`
恢复结果查询脚本：`scripts/query-restored-audit-logs.sh`

支持参数：
- `PG_DSN`（非 dry-run 时必填）
- `ARCHIVE_BEFORE_DAYS`（默认 90）
- `ARCHIVE_MODE=move|copy|purge`
  - `move`：导出后删除热层（默认）
  - `copy`：仅导出，保留热层
  - `purge`：仅删除热层（无冷存导出）
- `EXPORT_FORMAT=csv|jsonl`
- `COLD_STORAGE_MODE=local|local_and_s3|none`
- `S3_URI=s3://bucket/path`（`local_and_s3` 必填）
- `DRY_RUN=1|0`
- `OUTPUT_DIR`（可选，默认 `scripts/../data/audit-archive`）

推荐生产参数：

```bash
PG_DSN="host=localhost port=5432 user=postgres password=YOUR_TOKEN_HERE dbname=sub2api sslmode=disable" \
ARCHIVE_BEFORE_DAYS=90 \
ARCHIVE_MODE=move \
EXPORT_FORMAT=csv \
COLD_STORAGE_MODE=local_and_s3 \
S3_URI="s3://your-bucket/sub2api/plugin-market-audit" \
DRY_RUN=0 \
./scripts/archive-audit-logs.sh
```

## 4. 运维编排建议

- 每日：nightly `DRY_RUN=1` 校验可执行性
- 每周：`copy` 导出冷存，不删除热层（审计备份）
- 每月：`move` 正式归档，收缩热层数据量
- 紧急：`purge` 仅用于极端存储压力处置

## 5. 风险与控制

- 风险：误删热层历史数据
- 控制：先执行 `DRY_RUN=1`，再执行 `copy`，最后执行 `move`

- 风险：S3 上传失败导致冷层不完整
- 控制：`local_and_s3` 模式先落地本地文件，再上传；上传失败中止删除

- 风险：归档窗口配置错误
- 控制：固定 `ARCHIVE_BEFORE_DAYS` 下限（建议不小于 30）并纳入变更评审

## 6. 后续增强（P2）

1. PostgreSQL 分区表策略（按月分区）
2. 冷层元数据清单（归档批次、行数、哈希）
3. 对象存储生命周期策略（自动转冷/归档存储）
4. 恢复查询与校验增强（多样本快照、统计告警阈值）
