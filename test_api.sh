#!/bin/bash

# API 测试脚本
# 使用方法: ./test_api.sh

BASE_URL="http://localhost:8080"

echo "=== Sub2API Plugin Market API 测试 ==="
echo ""

# 1. 健康检查
echo "1. 健康检查"
curl -s "${BASE_URL}/health" | jq .
echo ""

# 2. 获取插件列表
echo "2. 获取插件列表（所有插件）"
curl -s "${BASE_URL}/api/v1/plugins" | jq .
echo ""

# 3. 获取官方插件列表
echo "3. 获取官方插件列表"
curl -s "${BASE_URL}/api/v1/plugins?is_official=true" | jq .
echo ""

# 4. 按分类查询插件
echo "4. 按分类查询插件（proxy）"
curl -s "${BASE_URL}/api/v1/plugins?category=proxy" | jq .
echo ""

# 5. 搜索插件
echo "5. 搜索插件（关键词：auth）"
curl -s "${BASE_URL}/api/v1/plugins?search=auth" | jq .
echo ""

# 6. 分页查询
echo "6. 分页查询（第 1 页，每页 10 条）"
curl -s "${BASE_URL}/api/v1/plugins?page=1&page_size=10" | jq .
echo ""

# 7. 获取插件详情
echo "7. 获取插件详情（示例：example-plugin）"
curl -s "${BASE_URL}/api/v1/plugins/example-plugin" | jq .
echo ""

# 8. 获取插件版本列表
echo "8. 获取插件版本列表（示例：example-plugin）"
curl -s "${BASE_URL}/api/v1/plugins/example-plugin/versions" | jq .
echo ""

# 9. 下载插件
echo "9. 下载插件（示例：example-plugin v1.0.0）"
curl -I "${BASE_URL}/api/v1/plugins/example-plugin/versions/1.0.0/download"
echo ""

# 10. 获取信任密钥列表
echo "10. 获取信任密钥列表"
curl -s "${BASE_URL}/api/v1/trust-keys" | jq .
echo ""

# 11. 按类型查询信任密钥
echo "11. 按类型查询信任密钥（official）"
curl -s "${BASE_URL}/api/v1/trust-keys?key_type=official" | jq .
echo ""

# 12. 查询激活的信任密钥
echo "12. 查询激活的信任密钥"
curl -s "${BASE_URL}/api/v1/trust-keys?is_active=true" | jq .
echo ""

# 13. 获取信任密钥详情
echo "13. 获取信任密钥详情（示例：key-001）"
curl -s "${BASE_URL}/api/v1/trust-keys/key-001" | jq .
echo ""

echo "=== 测试完成 ==="
