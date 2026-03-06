# Database Migrations

本目录存放可审计的数据库迁移脚本，用于生产环境。开发环境仍使用 `cmd/server/main.go` 中的 `client.Schema.Create()` 自动迁移。

## 迁移文件命名

- `{序号}_{描述}.up.sql`：前向迁移（应用变更）
- `{序号}_{描述}.down.sql`：回滚迁移（撤销变更）

## 如何生成新的迁移脚本

1. **修改 Ent Schema**：编辑 `ent/schema/*.go`
2. **生成 Ent 代码**：`make generate`
3. **导出完整 Schema SQL**（需 PostgreSQL 可达）：
   ```bash
   make docker-up          # 启动 PostgreSQL
   make migrate-export      # 或: go run scripts/export_schema_sql.go -o migrations/000002_xxx.up.sql
   ```
4. **手动编辑**：将导出的 SQL 与当前迁移对比，只保留**增量变更**，保存为新迁移文件
5. **编写回滚脚本**：创建对应的 `.down.sql`，包含撤销变更的 DDL

## 在 Staging 验证迁移

```bash
# 1. 使用空数据库或从生产快照恢复
createdb plugin_market_staging
# 或: pg_restore ...

# 2. 应用迁移
psql "postgresql://user:pass@host:5432/plugin_market_staging" -f migrations/000001_initial_schema.up.sql

# 3. 启动服务验证
DB_NAME=plugin_market_staging make run

# 4. 可选：测试回滚
psql "postgresql://..." -f migrations/000001_initial_schema.down.sql
```

## 在生产执行迁移

1. **备份**：对生产数据库做完整备份
2. **维护窗口**：建议在低峰期执行
3. **执行**：
   ```bash
   psql "$DATABASE_URL" -f migrations/000001_initial_schema.up.sql
   ```
4. **验证**：启动服务，执行健康检查与冒烟测试

## 回滚策略

- **事前**：每个 `.up.sql` 应有对应的 `.down.sql`
- **执行回滚**：`psql "$DATABASE_URL" -f migrations/000001_initial_schema.down.sql`
- **注意**：回滚会删除表及数据，仅适用于紧急恢复。常规流程应通过新的前向迁移修复问题。
- **数据迁移**：若迁移包含不可逆的数据变更，回滚脚本可能无法完整恢复，需依赖备份。

## 辅助脚本

| 脚本 | 作用 |
|------|------|
| `scripts/export_schema_sql.go` | 从当前 Ent schema 导出完整 PostgreSQL DDL，输出到指定文件 |

## 当前迁移清单

| 序号 | 描述 | Phase |
|------|------|-------|
| 000001 | 初始 Schema（admin_users, trust_keys, plugins, plugin_versions, submissions, download_logs, sync_jobs） | Phase 0 + Phase 1 |

Phase 0：`plugin.name` Match 正则约束  
Phase 1：`plugin_type` enum、`capabilities`/`config_schema` JSON、submission→version edge、`sync_job` 表
