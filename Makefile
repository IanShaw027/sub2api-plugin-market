.PHONY: help run build test check-contract clean docker-up docker-down migrate migrate-up migrate-down install-tools \
	audit-archive-dry-run audit-restore-dry-run audit-restore-validate audit-restore-query-sample audit-dry-run

help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

run: ## 运行服务
	go run cmd/server/main.go

build: ## 编译二进制文件
	go build -o bin/server cmd/server/main.go

check-contract: ## 校验 OpenAPI 与错误码契约一致性
	./scripts/validate_contract.sh

test: check-contract ## 运行测试
	go test -v -race ./...

test-coverage: test ## 生成测试覆盖率报告
	go test -v -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html

clean: ## 清理构建产物
	rm -rf bin/ dist/ coverage.txt coverage.html

docker-up: ## 启动 Docker 服务（PostgreSQL + MinIO）
	docker-compose up -d

docker-down: ## 停止 Docker 服务
	docker-compose down

docker-clean: ## 清理 Docker 数据
	docker-compose down -v

migrate: ## 运行数据库迁移
	go run cmd/server/main.go migrate

migrate-up: ## 运行数据库迁移（别名）
	go run cmd/server/main.go migrate

migrate-down: ## 回滚数据库迁移
	go run cmd/server/main.go migrate-down

migrate-export: ## 导出 Ent schema 为 SQL（需 PostgreSQL 可达，如 make docker-up）
	go run scripts/export_schema_sql.go -o migrations/000001_initial_schema.up.sql

install-tools: ## 安装开发工具
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint: ## 运行代码检查
	golangci-lint run

fmt: ## 格式化代码
	go fmt ./...
	goimports -w .

deps: ## 安装依赖
	go mod download
	go mod tidy

generate: ## 生成 Ent 代码
	go generate ./ent

audit-archive-dry-run: ## 审计归档脚本 dry-run（默认 DRY_RUN=1）
	DRY_RUN=$${DRY_RUN:-1} ./scripts/archive-audit-logs.sh

audit-restore-dry-run: ## 审计恢复脚本 dry-run（默认 sample + DRY_RUN=1）
	INPUT_FILE=$${INPUT_FILE:-./scripts/restore-audit-logs-sample.jsonl} DRY_RUN=$${DRY_RUN:-1} ./scripts/restore-audit-logs.sh

audit-restore-validate: ## 校验恢复结果（默认 /tmp/restored-audit-logs.jsonl）
	RESTORED_FILE=$${RESTORED_FILE:-/tmp/restored-audit-logs.jsonl} EXPECTED_SHA256=$${EXPECTED_SHA256:-} ./scripts/validate-restored-audit-logs.sh

audit-restore-query-sample: ## 对 sample 执行分页过滤查询示例
	INPUT_FILE=$${INPUT_FILE:-./scripts/restore-audit-logs-sample.jsonl} PAGE=$${PAGE:-1} PAGE_SIZE=$${PAGE_SIZE:-2} ACTION=$${ACTION:-submission.approved} ./scripts/query-restored-audit-logs.sh

audit-dry-run: ## 串行执行审计 dry-run（archive/restore/query/validate）
	DRY_RUN=1 ./scripts/archive-audit-logs.sh
	INPUT_FILE=$${INPUT_FILE:-./scripts/restore-audit-logs-sample.jsonl} DRY_RUN=1 ./scripts/restore-audit-logs.sh --output /tmp/restored-audit-logs.jsonl
	INPUT_FILE=$${INPUT_FILE:-./scripts/restore-audit-logs-sample.jsonl} PAGE=$${PAGE:-1} PAGE_SIZE=$${PAGE_SIZE:-2} ACTION=$${ACTION:-submission.approved} ./scripts/query-restored-audit-logs.sh
	@if [ -f "$${RESTORED_FILE:-/tmp/restored-audit-logs.jsonl}" ]; then \
		RESTORED_FILE=$${RESTORED_FILE:-/tmp/restored-audit-logs.jsonl} EXPECTED_SHA256=$${EXPECTED_SHA256:-} ./scripts/validate-restored-audit-logs.sh; \
	else \
		echo "[audit-dry-run] skip validate: restored file not found ($${RESTORED_FILE:-/tmp/restored-audit-logs.jsonl})"; \
	fi
