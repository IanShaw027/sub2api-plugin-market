#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
API_DOC="$ROOT_DIR/docs/API.md"
ERROR_REGISTRY_DOC="$ROOT_DIR/docs/ERROR-CODE-REGISTRY.md"
OPENAPI_SPEC="$ROOT_DIR/openapi/plugin-market-v1.yaml"
RESPONSE_CODE_FILE="$ROOT_DIR/internal/api/v1/handler/response.go"
SERVER_ENTRY_FILE="$ROOT_DIR/cmd/server/main.go"

fail() {
  echo "[contract-check] ❌ $1" >&2
  exit 1
}

for file in "$ERROR_REGISTRY_DOC" "$OPENAPI_SPEC" "$RESPONSE_CODE_FILE" "$SERVER_ENTRY_FILE"; do
  [[ -f "$file" ]] || fail "缺少必需文件: $file"
done

if command -v ruby >/dev/null 2>&1; then
  ruby -e "require 'yaml'; YAML.load_file('$OPENAPI_SPEC')" >/dev/null
else
  fail "未找到 ruby，无法校验 OpenAPI YAML 语法"
fi

registry_codes="$(
  grep -E '^\| `1[0-9]{3}` \|' "$ERROR_REGISTRY_DOC" |
    sed -E 's/^\| `([0-9]+)` \|.*/\1/' |
    sort -u |
    tr '\n' ' ' |
    xargs
)"
source_codes="$(
  awk -F '=' '
    /ErrCode[A-Za-z0-9_]+[[:space:]]*=[[:space:]]*[0-9]+/ {
      value=$2
      gsub(/[[:space:]]/, "", value)
      print value
    }
  ' "$RESPONSE_CODE_FILE" |
    sort -u |
    tr '\n' ' ' |
    xargs
)"

[[ -n "$registry_codes" ]] || fail "未从 docs/ERROR-CODE-REGISTRY.md 提取到错误码"
[[ -n "$source_codes" ]] || fail "未从 response.go 提取到错误码"

[[ "$registry_codes" == "$source_codes" ]] || fail "错误码注册表与源码不一致。Registry=[$registry_codes] Source=[$source_codes]"

if [[ -f "$API_DOC" ]]; then
  api_doc_codes="$(
    grep -Eo '`1[0-9]{3}`' "$API_DOC" |
      tr -d '`' |
      sort -u |
      tr '\n' ' ' |
      xargs || true
  )"
  if [[ -n "$api_doc_codes" ]]; then
    for code in $source_codes; do
      if [[ " $api_doc_codes " != *" $code "* ]]; then
        echo "[contract-check] ⚠️ docs/API.md 未覆盖错误码 $code（非阻断）"
      fi
    done
  else
    echo "[contract-check] ⚠️ docs/API.md 未解析到错误码（非阻断）"
  fi
else
  echo "[contract-check] ⚠️ docs/API.md 不存在，跳过文档错误码校验（非阻断）"
fi

grep -q "url: http://localhost:8081" "$OPENAPI_SPEC" || fail "OpenAPI servers.url 未设置为 http://localhost:8081"

grep -q 'getEnv("PORT", "8081")' "$SERVER_ENTRY_FILE" || fail "服务默认端口不是 8081，需同步契约与文档"

required_paths=(
  "/health"
  "/api/v1/plugins"
  "/api/v1/plugins/{name}/versions/{version}/download"
  "/api/v1/trust-keys"
  "/admin/api/auth/login"
  "/admin/api/auth/refresh"
  "/admin/api/auth/me"
  "/admin/api/auth/logout"
  "/admin/api/submissions"
  "/admin/api/submissions/{id}"
  "/admin/api/submissions/stats"
  "/admin/api/submissions/{id}/review"
)

for path in "${required_paths[@]}"; do
  grep -q "^  ${path}:" "$OPENAPI_SPEC" || fail "OpenAPI 缺少关键路径: ${path}"
done

download_block="$(awk '
  /^  \/api\/v1\/plugins\/\{name\}\/versions\/\{version\}\/download:/ { in_block=1; print; next }
  in_block && /^  \// { in_block=0 }
  in_block { print }
' "$OPENAPI_SPEC")"
[[ -n "$download_block" ]] || fail "OpenAPI 缺少下载接口定义块"
echo "$download_block" | grep -q "^        '302':" || fail "下载接口缺少 302 响应定义"
echo "$download_block" | grep -q "^        '404':" || fail "下载接口缺少 404 响应定义"
echo "$download_block" | grep -q "^        '500':" || fail "下载接口缺少 500 响应定义"

echo "[contract-check] ✅ 契约校验通过"
