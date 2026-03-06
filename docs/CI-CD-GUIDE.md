# CI/CD Pipeline 指南

本文档描述 Plugin Market（控制平面）和 Sub2API（数据平面）两个仓库的 CI/CD 流程、签名密钥管理和部署策略。

---

## 1. Market 仓库 CI

Market 仓库（`sub2api-plugin-market`）的 CI 流程覆盖契约校验、测试、代码质量和构建。

### 1.1 Pipeline 阶段

```
push/PR → contract → test → lint → build → (docker → deploy)
```

| 阶段 | 命令 | 说明 |
|------|------|------|
| **契约校验** | `make check-contract` | 校验 OpenAPI 定义与错误码注册表一致性 |
| **单元+集成测试** | `make test` | 运行 `go test -v -race ./...`，需要 PostgreSQL |
| **代码检查** | `make lint` | 运行 `golangci-lint run` |
| **构建** | `make build` | 编译二进制到 `bin/server` |

### 1.2 测试依赖

- **PostgreSQL 15**：CI 中通过 GitHub Actions Service Container 提供
- **MinIO**：单元测试通过 mock 覆盖，集成测试可选启用

### 1.3 Workflow 文件

位于 `.github/workflows/ci.yml`，触发条件：

- `push` 到任意分支
- `pull_request` 到 `main` 分支

---

## 2. Sub2API 仓库 CI

Sub2API 仓库（数据平面/执行平面）的 CI 流程覆盖运行时测试、插件编译和签名。

### 2.1 运行时测试

```bash
# 插件运行时核心测试
go test -v -race ./internal/pluginruntime/...

# 插件管理测试
go test -v -race ./internal/plugins/...
```

覆盖范围：
- Dispatcher 路由与调度
- PluginExecutor WASM 执行
- PluginManager 生命周期管理
- 熔断器（CircuitBreaker）状态转换
- 灰度对比逻辑

### 2.2 WASM 编译步骤

使用 TinyGo 将 Go 插件编译为 WASM 模块：

```bash
# 安装 TinyGo（CI 中通过 Action 安装）
# https://tinygo.org/getting-started/install/

# 编译插件
tinygo build -o plugin.wasm -target wasi -scheduler=none ./plugin/main.go

# 验证 WASM 文件
file plugin.wasm
wasm-validate plugin.wasm  # 需安装 wabt
```

CI 中的 TinyGo 安装：

```yaml
- name: Install TinyGo
  uses: nickg/setup-tinygo@v1
  with:
    tinygo-version: '0.31.0'

- name: Build WASM plugin
  run: tinygo build -o plugin.wasm -target wasi -scheduler=none ./plugin/main.go
```

### 2.3 签名步骤

使用 Ed25519 对编译产物签名：

```bash
# 计算 WASM 哈希
sha256sum plugin.wasm > plugin.wasm.sha256

# 使用 sub2api-pluginsign 工具签名
pluginsign sign \
  --key "$ED25519_PRIVATE_KEY" \
  --key-id "$SIGN_KEY_ID" \
  --file plugin.wasm \
  --output plugin.wasm.sig
```

CI 中签名集成：

```yaml
- name: Sign WASM artifact
  env:
    ED25519_PRIVATE_KEY: ${{ secrets.PLUGIN_SIGN_PRIVATE_KEY }}
    SIGN_KEY_ID: ${{ secrets.PLUGIN_SIGN_KEY_ID }}
  run: |
    pluginsign sign \
      --key "$ED25519_PRIVATE_KEY" \
      --key-id "$SIGN_KEY_ID" \
      --file plugin.wasm \
      --output plugin.wasm.sig
```

---

## 3. 签名密钥管理

### 3.1 密钥类型

| 类型 | 用途 | 存储位置 |
|------|------|----------|
| **官方签名私钥** | 官方插件签名 | GitHub Secrets / HashiCorp Vault |
| **官方签名公钥** | 写入 Market TrustStore | `trust_keys` 表 |
| **社区签名公钥** | 社区插件验签 | `trust_keys` 表（经审核） |

### 3.2 GitHub Secrets 配置

在仓库 Settings → Secrets and variables → Actions 中设置：

| Secret 名称 | 说明 |
|-------------|------|
| `PLUGIN_SIGN_PRIVATE_KEY` | Ed25519 私钥（Base64 编码） |
| `PLUGIN_SIGN_KEY_ID` | 密钥 ID（与 TrustStore 中 `key_id` 对应） |
| `DOCKER_USERNAME` | Docker Hub 用户名 |
| `DOCKER_PASSWORD` | Docker Hub 密码/Token |

### 3.3 HashiCorp Vault（生产推荐）

```bash
# 写入密钥到 Vault
vault kv put secret/plugin-market/signing \
  private_key="$(base64 < ed25519_private.key)" \
  key_id="official-2024"

# CI 中读取（通过 Vault Action）
- name: Import secrets from Vault
  uses: hashicorp/vault-action@v3
  with:
    url: ${{ secrets.VAULT_ADDR }}
    token: ${{ secrets.VAULT_TOKEN }}
    secrets: |
      secret/data/plugin-market/signing private_key | ED25519_PRIVATE_KEY ;
      secret/data/plugin-market/signing key_id | SIGN_KEY_ID
```

### 3.4 密钥轮换

1. 生成新的 Ed25519 密钥对
2. 将新公钥注册到 Market `trust_keys`（保持旧公钥 `is_active=true`）
3. 更新 GitHub Secrets / Vault 中的私钥
4. 验证新签名的插件可通过验签
5. 过渡期结束后，将旧公钥设为 `is_active=false`

---

## 4. 部署触发

### 4.1 自动部署（推荐 staging）

`main` 分支合并后自动触发：

```
PR 合并 → CI 通过 → Docker 镜像构建推送 → staging 自动部署
```

### 4.2 手动部署（推荐 production）

通过 GitHub Actions `workflow_dispatch` 手动触发：

```yaml
on:
  workflow_dispatch:
    inputs:
      environment:
        description: '部署环境'
        required: true
        type: choice
        options:
          - staging
          - production
      image_tag:
        description: 'Docker 镜像 Tag'
        required: true
        default: 'latest'
```

### 4.3 部署前检查清单

- [ ] CI 全部通过（contract + test + lint + build）
- [ ] 数据库迁移已在 staging 预演（见 `DB-MIGRATION-RUNBOOK.md`）
- [ ] 变更已经过 Code Review
- [ ] 灰度期相关监控已就绪（见 `MONITORING-GUIDE.md`）

### 4.4 回滚策略

```bash
# 快速回滚到上一版本
docker pull $REGISTRY/sub2api-plugin-market:$PREVIOUS_TAG
docker-compose up -d

# 或通过 Kubernetes
kubectl rollout undo deployment/plugin-market
```
