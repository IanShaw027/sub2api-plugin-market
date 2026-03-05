#!/bin/bash

# Sub2API Plugin Market - 快速启动脚本

set -euo pipefail

cd "$(cd "$(dirname "$0")/.." && pwd)"

echo "=========================================="
echo "Sub2API Plugin Market - 快速启动"
echo "=========================================="
echo ""

# 检查 Docker 服务
echo "1. 检查 Docker 服务..."
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker 未运行，请先启动 Docker"
    exit 1
fi
echo "✅ Docker 正常运行"
echo ""

# 启动 Docker 服务
echo "2. 启动 PostgreSQL 和 MinIO..."
docker-compose up -d
sleep 3
echo "✅ Docker 服务已启动"
echo ""

# 初始化 .env
if [ ! -f ".env" ]; then
    echo "检测到缺少 .env，复制 .env.example..."
    cp .env.example .env
fi

# 检查数据库连接
echo "3. 检查数据库连接..."
set -a
. ./.env
set +a

if PGPASSWORD="${DB_PASSWORD:-postgres}" psql -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5433}" -U "${DB_USER:-postgres}" -d "${DB_NAME:-plugin_market}" -c "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ 数据库连接正常"
else
    echo "❌ 数据库连接失败"
    exit 1
fi
echo ""

# 初始化管理员账号（如果不存在）
echo "4. 初始化管理员账号..."
if [ -f "scripts/init_admin.go" ]; then
    go run scripts/init_admin.go 2>/dev/null || echo "管理员账号已存在或初始化失败（可忽略）"
fi
echo ""

# 停止旧进程
echo "5. 停止旧进程..."
OLD_PIDS=$(lsof -ti:8081 2>/dev/null || true)
if [ -n "$OLD_PIDS" ]; then
    kill $OLD_PIDS 2>/dev/null || true
    sleep 1
    echo "✅ 已停止旧进程"
else
    echo "✅ 无旧进程"
fi
echo ""

# 启动服务
echo "6. 启动服务..."
nohup go run cmd/server/main.go > /tmp/plugin-market.log 2>&1 &
sleep 3

# 检查服务状态
if curl -s http://localhost:8081/health > /dev/null 2>&1; then
    echo "✅ 服务启动成功"
else
    echo "❌ 服务启动失败，查看日志："
    tail -20 /tmp/plugin-market.log
    exit 1
fi
echo ""

echo "=========================================="
echo "启动完成！"
echo "=========================================="
echo ""
echo "服务信息："
echo "  API 地址: http://localhost:8081"
echo "  健康检查: http://localhost:8081/health"
echo "  管理后台: http://localhost:8081/admin/login"
echo ""
echo "默认账号："
echo "  用户名: admin"
echo "  密码: admin123"
echo ""
echo "日志文件："
echo "  /tmp/plugin-market.log"
echo ""
echo "停止服务："
echo "  kill \$(lsof -ti:8081)"
echo ""
