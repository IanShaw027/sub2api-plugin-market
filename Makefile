.PHONY: help run build test clean docker-up docker-down migrate migrate-down

help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

run: ## 运行服务
	go run cmd/server/main.go

build: ## 编译二进制文件
	go build -o bin/server cmd/server/main.go

test: ## 运行测试
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-coverage: test ## 生成测试覆盖率报告
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

migrate-down: ## 回滚数据库迁移
	go run cmd/server/main.go migrate-down

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
