#!/bin/bash

# Sub2API Plugin Market - 快速启动脚本

set -euo pipefail

echo "🚀 Sub2API Plugin Market 启动脚本"
echo ""

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker 未运行，请先启动 Docker"
    exit 1
fi

# 启动依赖服务
echo "📦 启动依赖服务 (PostgreSQL + MinIO)..."
docker-compose up -d

# 等待服务就绪
echo "⏳ 等待服务就绪..."
sleep 5

# 检查数据库连接
echo "🔍 检查数据库连接..."
if [ ! -f ".env" ]; then
    echo "📝 检测到缺少 .env，复制 .env.example..."
    cp .env.example .env
fi

set -a
. ./.env
set +a

if ! PGPASSWORD="${DB_PASSWORD:-postgres}" psql -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5433}" -U "${DB_USER:-postgres}" -d "${DB_NAME:-plugin_market}" -c "SELECT 1" > /dev/null 2>&1; then
    echo "❌ 数据库连接失败"
    exit 1
fi
echo "✅ 数据库连接成功"

# 检查管理员账号
echo "🔍 检查管理员账号..."
ADMIN_EXISTS=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM admin_users WHERE username='admin'" 2>/dev/null || echo "0")

if [ "$ADMIN_EXISTS" -eq "0" ]; then
    echo "📝 创建管理员账号..."
    go run scripts/init_admin.go
else
    echo "✅ 管理员账号已存在"
fi

# 检查测试数据
echo "🔍 检查测试数据..."
SUBMISSION_COUNT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM submissions" 2>/dev/null || echo "0")

if [ "$SUBMISSION_COUNT" -eq "0" ]; then
    echo "📝 创建测试数据..."
    go run scripts/create_test_data.go
else
    echo "✅ 测试数据已存在 ($SUBMISSION_COUNT 条记录)"
fi

# 停止旧服务
echo "🛑 停止旧服务..."
lsof -ti:8081 | xargs kill 2>/dev/null || true
sleep 2

# 启动服务器
echo "🚀 启动服务器..."
nohup go run cmd/server/main.go > server.log 2>&1 &

# 等待服务器启动
echo "⏳ 等待服务器启动..."
for i in {1..10}; do
    if curl -s http://localhost:8081/health > /dev/null 2>&1; then
        echo "✅ 服务器启动成功！"
        break
    fi
    sleep 1
done

echo ""
echo "🎉 启动完成！"
echo ""
echo "📍 访问地址："
echo "   - 管理后台: http://localhost:8081/admin/login"
echo "   - API 文档: http://localhost:8081/api/v1"
echo "   - 健康检查: http://localhost:8081/health"
echo ""
echo "🔑 默认账号："
echo "   用户名: admin"
echo "   密码: admin123"
echo ""
echo "📊 当前数据："
echo "   - 提交总数: $SUBMISSION_COUNT"
echo ""
echo "📝 查看日志: tail -f server.log"
echo "🛑 停止服务: lsof -ti:8081 | xargs kill -9"
echo ""
