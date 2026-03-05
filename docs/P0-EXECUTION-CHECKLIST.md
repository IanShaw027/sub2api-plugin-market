# P0 执行清单（进行中）

- 更新时间：2026-03-04
- 目标：先消除阻断项，再进入 P1 功能建设

## 已完成

1. ✅ CI 红线修复：`go test ./...` 通过（解决 `scripts` 多 main 冲突）
2. ✅ 管理后台审核接口契约对齐（前后端字段统一 + 旧字段兼容）
3. ✅ 安全基线快修（`ADMIN_JWT_SECRET` 校验、CORS 白名单）
4. ✅ 文档与实现对齐（下载 302、错误码、端口与环境变量）
5. ✅ CI 触发策略修复（`push main` 与 `docker` job 条件一致）
6. ✅ 架构 ADR 冻结（见 `docs/ADR-001-hybrid-architecture.md`）
7. ✅ 契约注册表落地（`openapi/plugin-market-v1.yaml` + `docs/ERROR-CODE-REGISTRY.md`）
8. ✅ 契约门禁落地（`scripts/validate_contract.sh` + CI `contract` job + `make check-contract`）
9. ✅ 跨仓第一阶段落地（`sub2api` 新增 `plugin_market` 配置、HTTP RegistryStore、`/api/v1/admin/plugins` 命名空间）
10. ✅ 跨仓第二阶段落地（`sub2api` 主服务路由接线外部 market store，失败降级 + 日志告警）
11. ✅ 部署资产路径收敛（`deploy/plugin-market` 前端/文档/集成脚本统一到 `/api/v1/admin/plugins`）
12. ✅ 跨仓最小闭环 E2E（外部 market mock -> 注册表查询 -> 预检 -> 安装 -> 重启对齐）
13. ✅ 夜间 CI smoke 门禁（`plugin-market-nightly.yml`）
14. ✅ 发布门禁增强（`sub2api/backend/scripts/validate_plugin_market_boundary.sh` + `backend/Makefile` + CI/nightly 接线 + 旧路径 404 防回归测试）
15. ✅ P1 迁移文档与占位清理（`deploy/plugin-market/MIGRATION_GUIDE.md` + `deploy/plugin-market/P1-TODO.md` + UI 文档状态收敛）
16. ✅ 跨仓深度契约校验（`validate_plugin_market_boundary.sh` 复用 `sub2api-plugin-market/scripts/validate_contract.sh`）
17. ✅ 跨仓 OpenAPI diff 自动检查（`check_plugin_market_openapi_diff.sh` + 快照文件 + CI 输出差异）
18. ✅ 社区提交与审计日志接入（后端控制平面 API + 前端页面 + 集成文档/脚本更新）
19. ✅ 审核追溯增强与分页能力（reviewer 身份透传 + submissions/audit 分页 + 前端分页 UI + 集成脚本回归）
20. ✅ 集成测试脚本跨环境自适配（auto/sub2api/plugin_market 模式 + 文档同步）
21. ✅ 迁移运维手册固化（release-checklist 脚本 + Makefile + CI/nightly 接线）
22. ✅ nightly 可选写链路 smoke（secrets 注入后执行 create/review/audit 回归）
23. ✅ 预发布 full 检查 workflow_dispatch（远程 `base_url` + 可选 admin key）
24. ✅ 审计日志多后端持久化（`plugin_market.audit_store_driver` + JSONL/PostgreSQL + 启动回放/索引查询/保留策略 + 回退内存 + 覆盖测试）
25. ✅ 审计日志归档自动化（`archive-audit-logs.sh` + `release-checklist.sh` 可选 dry-run 门禁 + 迁移手册命令示例）
27. ✅ release-checklist 回滚演练可选校验（`rollback-drill.sh` + `CHECK_ROLLBACK_DRILL=1`）
28. ✅ PostgreSQL 审计归档冷热分层策略（`archive-audit-logs.sh` 分层模式 + nightly 可选 dry-run + 策略文档）
30. ✅ pre-release checklist 可选矩阵（归档 dry-run + 回滚演练）
31. ✅ 审计冷层恢复脚本（`restore-audit-logs.sh`，支持 CSV/JSONL dry-run 回放到临时表）
32. ✅ 恢复结果查询脚本（`query-restored-audit-logs.sh`，支持分页与过滤）
33. ✅ nightly/pre-release 接入归档与恢复 dry-run 可选矩阵
34. ✅ 恢复结果校验脚本（`validate-restored-audit-logs.sh`，行数/时间窗口/哈希摘要）
35. ✅ release-checklist 接入恢复结果校验可选 gate（`CHECK_AUDIT_RESTORE_VALIDATE=1`）
36. ✅ 恢复脚本增强：按时间窗口回放 + 重复导入保护（`DEDUP_MODE`）
37. ✅ pre-release 增加恢复校验 gate 可选输入（`check_audit_restore_validate`）
38. ✅ 固定样本恢复校验接线（`EXPECTED_SHA256`，`scripts/restore-audit-logs-sample.jsonl`）
39. ✅ nightly 接入恢复校验可选步骤（`.github/workflows/plugin-market-nightly.yml`）
40. ✅ 恢复回放去重统计增强（`restore-audit-logs.sh` 区分 `DUPLICATE_TARGET`/`DUPLICATE_BATCH`）

## 进行中

- （无）

## 下一步（建议按顺序）

1. 结合生产恢复样本扩展固定快照集（按归档批次维护 EXPECTED_SHA256 清单）
2. 在 nightly 增加 CSV 输入样本覆盖（当前仅 JSONL）
3. 将去重统计接入告警阈值（如重复率异常波动）
