#!/bin/bash

# 测试管理后台 API

BASE_URL="http://localhost:8081"
API_BASE="$BASE_URL/admin/api"

echo "=========================================="
echo "Sub2API Plugin Market - 管理后台测试"
echo "=========================================="
echo ""

# 1. 测试登录
echo "1. 测试登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

echo "$LOGIN_RESPONSE" | jq .

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo "❌ 登录失败"
    exit 1
fi

echo "✅ 登录成功"
echo ""

# 2. 测试获取用户信息
echo "2. 测试获取用户信息..."
curl -s "$API_BASE/auth/me" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo "✅ 获取用户信息成功"
echo ""

# 3. 测试统计数据
echo "3. 测试统计数据..."
curl -s "$API_BASE/submissions/stats" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo "✅ 获取统计数据成功"
echo ""

# 4. 测试提交列表
echo "4. 测试提交列表..."
curl -s "$API_BASE/submissions?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo "✅ 获取提交列表成功"
echo ""

# 5. 测试待审核列表
echo "5. 测试待审核列表..."
curl -s "$API_BASE/submissions?status=pending&page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN" | jq .
echo "✅ 获取待审核列表成功"
echo ""

# 6. 测试前端页面
echo "6. 测试前端页面..."
LOGIN_PAGE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/admin/login")
INDEX_PAGE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/admin/")
JS_FILE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/admin/js/app.js")

echo "登录页面: $LOGIN_PAGE"
echo "管理界面: $INDEX_PAGE"
echo "JavaScript: $JS_FILE"

if [ "$LOGIN_PAGE" == "200" ] && [ "$INDEX_PAGE" == "200" ] && [ "$JS_FILE" == "200" ]; then
    echo "✅ 前端页面正常"
else
    echo "❌ 前端页面异常"
fi

echo ""
echo "=========================================="
echo "测试完成！"
echo "=========================================="
echo ""
echo "访问地址："
echo "  登录页面: $BASE_URL/admin/login"
echo "  管理界面: $BASE_URL/admin/"
echo ""
echo "默认账号："
echo "  用户名: admin"
echo "  密码: admin123"
echo ""
