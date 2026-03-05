# Contributing

感谢你关注 `sub2api-plugin-market`。

## 开发流程

1. Fork 仓库并创建分支：`feature/*` 或 `fix/*`
2. 本地执行：
   - `make check-contract`
   - `make test`
3. 提交 PR，说明变更背景、方案和验证结果

## 提交规范

建议使用 Conventional Commits：

- `feat:` 新功能
- `fix:` 缺陷修复
- `docs:` 文档更新
- `refactor:` 重构
- `test:` 测试相关
- `chore:` 工具链/流程

## API 变更要求

如果涉及接口契约，请同步更新：

- `openapi/plugin-market-v1.yaml`
- `docs/API.md`
- `docs/ERROR-CODE-REGISTRY.md`（如新增错误码）

## 代码风格

- Go 代码请通过 `go fmt ./...`
- 提交前执行 `make lint`（如本地已安装 golangci-lint）
