# 数据库迁移预演手册

本文档定义数据库迁移的标准预演流程，确保生产迁移可控、可回滚。

## 1. 预演步骤

**原则**：任何迁移在生产执行前，必须在 staging 环境完整预演一次。

### 1.1 准备 Staging 环境

```bash
# 从生产快照恢复 staging 数据库（推荐），或使用空数据库
pg_dump "$PROD_DATABASE_URL" | psql "$STAGING_DATABASE_URL"

# 或创建空数据库
createdb -h staging-host plugin_market_staging
```

### 1.2 记录迁移前状态

```bash
# 记录当前表结构
psql "$STAGING_DATABASE_URL" -c "\dt+" > pre_migration_tables.txt

# 记录行数（用于迁移后对比）
psql "$STAGING_DATABASE_URL" -c "
  SELECT schemaname, relname, n_live_tup
  FROM pg_stat_user_tables
  ORDER BY relname;
" > pre_migration_row_counts.txt
```

### 1.3 执行迁移

```bash
# 开始计时
time psql "$STAGING_DATABASE_URL" -f migrations/000001_initial_schema.up.sql
```

### 1.4 验证（见下方验证清单）

### 1.5 记录结果并归档

将预演结果填入下方「耗时记录模板」，存入团队文档。

---

## 2. 迁移文件位置

| 文件 | 说明 |
|------|------|
| `migrations/000001_initial_schema.up.sql` | 初始 Schema（前向迁移） |
| `migrations/000001_initial_schema.down.sql` | 初始 Schema（回滚迁移） |

新增迁移文件按 `{序号}_{描述}.up.sql` / `.down.sql` 命名，详见 `migrations/README.md`。

---

## 3. 执行命令

```bash
# 前向迁移
psql "$DATABASE_URL" -f migrations/000001_initial_schema.up.sql

# 如果迁移脚本有多个，按序号顺序执行
for f in migrations/*.up.sql; do
  echo "=== Applying: $f ==="
  psql "$DATABASE_URL" -f "$f"
done
```

**注意**：
- 迁移脚本使用 `CREATE TABLE IF NOT EXISTS`，重复执行是幂等的
- 但 `ALTER TABLE ADD CONSTRAINT` 不幂等，重复执行会报错（约束已存在）
- 建议在事务中执行：`psql "$DATABASE_URL" -1 -f migrations/000001_initial_schema.up.sql`（`-1` 表示单事务）

---

## 4. 验证清单

迁移完成后，逐项检查：

### 4.1 数据完整性

- [ ] 旧数据正常读取：`SELECT count(*) FROM plugins;` 行数与迁移前一致
- [ ] 旧数据关联完整：`SELECT p.name, count(v.id) FROM plugins p LEFT JOIN plugin_versions v ON p.id = v.plugin_id GROUP BY p.name;`
- [ ] 审核记录完整：`SELECT status, count(*) FROM submissions GROUP BY status;`

### 4.2 新字段兼容性

- [ ] `plugin_type` 为 nullable，旧数据为 NULL 不影响查询：`SELECT id, plugin_type FROM plugins LIMIT 5;`
- [ ] `capabilities` 为 nullable JSON，旧数据为 NULL：`SELECT id, capabilities FROM plugin_versions LIMIT 5;`
- [ ] `config_schema` 为 nullable JSON，旧数据为 NULL：`SELECT id, config_schema FROM plugin_versions LIMIT 5;`
- [ ] `submission_version` 外键为 nullable，旧数据为 NULL：`SELECT id, submission_version FROM plugin_versions LIMIT 5;`
- [ ] `sync_jobs` 表已创建：`SELECT count(*) FROM sync_jobs;`

### 4.3 索引验证

- [ ] 关键索引存在：`SELECT indexname FROM pg_indexes WHERE tablename = 'plugins';`
- [ ] `plugin_name` 唯一约束生效：尝试插入重复 name 应失败
- [ ] `plugins_name_format` CHECK 约束生效：尝试插入不合规 name（如 `UPPER_CASE`）应失败

### 4.4 应用层验证

- [ ] 应用启动正常：`make run`（或 `go run cmd/server/main.go`），无报错
- [ ] 健康检查通过：`curl http://localhost:8081/api/v1/plugins` 返回 200
- [ ] 管理后台可访问：`curl http://localhost:8081/admin/login` 返回 200
- [ ] `make test` 通过：全量测试无失败

---

## 5. 耗时记录模板

| 项目 | 值 |
|------|------|
| **迁移文件** | `migrations/000001_initial_schema.up.sql` |
| **执行环境** | staging / production |
| **数据库版本** | PostgreSQL 15.x |
| **数据规模** | plugins: ___ 行, plugin_versions: ___ 行, download_logs: ___ 行 |
| **迁移耗时** | ___ 秒 |
| **锁等待时间** | ___ 秒（`ALTER TABLE` 期间） |
| **验证耗时** | ___ 分钟 |
| **执行人** | |
| **执行时间** | YYYY-MM-DD HH:MM |
| **结果** | 成功 / 失败（原因：___） |
| **备注** | |

---

## 6. 回滚步骤

### 6.1 回滚决策

出现以下情况时触发回滚：

- 迁移执行报错且无法现场修复
- 验证清单有关键项未通过
- 应用启动后核心功能异常

### 6.2 回滚操作

```bash
# 1. 停止应用
# systemctl stop plugin-market  (或 docker-compose down)

# 2. 执行回滚脚本
psql "$DATABASE_URL" -f migrations/000001_initial_schema.down.sql

# 3. 从备份恢复（如果回滚脚本不足以恢复数据）
pg_restore -d plugin_market /path/to/backup.dump

# 4. 启动旧版本应用
# systemctl start plugin-market
```

### 6.3 回滚注意事项

- `000001_initial_schema.down.sql` 会 **DROP 所有表**，仅适用于初始部署场景
- 对于增量迁移，回滚脚本应只撤销增量变更（如 `DROP COLUMN`、`DROP INDEX`）
- 包含不可逆数据变更的迁移（如数据格式转换），回滚后需从备份恢复数据
- **生产迁移前必须做完整备份**：`pg_dump "$DATABASE_URL" > backup_$(date +%Y%m%d_%H%M%S).sql`

### 6.4 回滚后检查

- [ ] 表结构恢复到迁移前状态
- [ ] 数据行数与迁移前一致
- [ ] 应用启动正常
- [ ] 核心 API 响应正常
