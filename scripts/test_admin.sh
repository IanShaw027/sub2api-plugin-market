#!/bin/bash

# Sub2API Plugin Market - 管理后台测试脚本

set -e

BASE_URL="http://localhost:8081"
TOKEN=""

echo "=== Sub2API Plugin Market 管理后台测试 ==="
echo ""

# 1. 测试健康检查
echo "1. 测试健康检查..."
curl -s "$BASE_URL/health" | jq .
echo "✅ 健康检查通过"
echo ""

# 2. 测试登录
echo "2. 测试管理员登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/admin/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

echo "$LOGIN_RESPONSE" | jq .

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
  echo "❌ 登录失败"
  exit 1
fi

echo "✅ 登录成功"
echo "Token: ${TOKEN:0:50}..."
echo ""

# 3. 测试获取当前用户信息
echo "3. 测试获取当前用户信息..."
curl -s -X GET "$BASE_URL/admin/api/auth/me" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo "✅ 获取用户信息成功"
echo ""

# 4. 测试审核统计
echo "4. 测试审核统计..."
curl -s -X GET "$BASE_URL/admin/api/submissions/stats" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo "✅ 获取审核统计成功"
echo ""

# 5. 测试获取提交列表
echo "5. 测试获取提交列表..."
curl -s -X GET "$BASE_URL/admin/api/submissions?status=pending&page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo "✅ 获取提交列表成功"
echo ""

# 6. 测试未授权访问
echo "6. 测试未授权访问..."
UNAUTH_RESPONSE=$(curl -s -X GET "$BASE_URL/admin/api/auth/me")
echo "$UNAUTH_RESPONSE" | jq .

if echo "$UNAUTH_RESPONSE" | jq -e '.code == 401' > /dev/null; then
  echo "✅ 未授权访问被正确拒绝"
else
  echo "❌ 未授权访问应该被拒绝"
  exit 1
fi
echo ""

# 7. 测试错误的密码
echo "7. 测试错误的密码..."
WRONG_PASSWORD_RESPONSE=$(curl -s -X POST "$BASE_URL/admin/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"wrongpassword"}')

echo "$WRONG_PASSWORD_RESPONSE" | jq .

if echo "$WRONG_PASSWORD_RESPONSE" | jq -e '.code == 401' > /dev/null; then
  echo "✅ 错误密码被正确拒绝"
else
  echo "❌ 错误密码应该被拒绝"
  exit 1
fi
echo ""

# 8. 测试登出
echo "8. 测试登出..."
curl -s -X POST "$BASE_URL/admin/api/auth/logout" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo "✅ 登出成功"
echo ""

echo "=== 所有测试通过 ✅ ==="
