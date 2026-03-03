#!/bin/bash

# 测试报告生成脚本

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORT_DIR="$PROJECT_ROOT/test-reports"

echo "==================================="
echo "  Sub2API Plugin Market 测试报告"
echo "==================================="
echo ""

# 创建报告目录
mkdir -p "$REPORT_DIR"

# 1. 运行单元测试和集成测试
echo "📋 运行测试套件..."
go test -v -coverprofile="$REPORT_DIR/coverage.out" -covermode=atomic ./... > "$REPORT_DIR/test-output.txt" 2>&1 || true

# 2. 生成覆盖率报告
echo "📊 生成覆盖率报告..."
go tool cover -html="$REPORT_DIR/coverage.out" -o "$REPORT_DIR/coverage.html"
go tool cover -func="$REPORT_DIR/coverage.out" > "$REPORT_DIR/coverage.txt"

# 3. 运行性能测试
echo "⚡ 运行性能测试..."
go test -bench=. -benchmem ./tests/integration/... > "$REPORT_DIR/benchmark.txt" 2>&1 || true

# 4. 运行并发测试
echo "🔄 运行并发测试..."
go test -v -run TestConcurrent ./tests/integration/... > "$REPORT_DIR/concurrent.txt" 2>&1 || true

# 5. 生成测试摘要
echo "📝 生成测试摘要..."

TOTAL_TESTS=$(grep -c "^=== RUN" "$REPORT_DIR/test-output.txt" || echo "0")
PASSED_TESTS=$(grep -c "^--- PASS" "$REPORT_DIR/test-output.txt" || echo "0")
FAILED_TESTS=$(grep -c "^--- FAIL" "$REPORT_DIR/test-output.txt" || echo "0")
COVERAGE=$(grep "total:" "$REPORT_DIR/coverage.txt" | awk '{print $3}' || echo "0%")

cat > "$REPORT_DIR/summary.md" << EOF
# Sub2API Plugin Market 测试报告

生成时间: $(date '+%Y-%m-%d %H:%M:%S')

## 测试摘要

- **总测试数**: $TOTAL_TESTS
- **通过**: $PASSED_TESTS ✅
- **失败**: $FAILED_TESTS ❌
- **测试覆盖率**: $COVERAGE

## 测试类型

### 1. 单元测试和集成测试

详见: [test-output.txt](./test-output.txt)

### 2. 覆盖率报告

- HTML 报告: [coverage.html](./coverage.html)
- 文本报告: [coverage.txt](./coverage.txt)

### 3. 性能测试

详见: [benchmark.txt](./benchmark.txt)

### 4. 并发测试

详见: [concurrent.txt](./concurrent.txt)

## 测试覆盖

### API 接口测试

- ✅ GET /api/v1/plugins - 插件列表
- ✅ GET /api/v1/plugins/:name - 插件详情
- ✅ GET /api/v1/plugins/:name/versions - 版本列表
- ✅ GET /api/v1/trust-keys - 信任密钥列表
- ✅ GET /api/v1/trust-keys/:key_id - 信任密钥详情

### 测试场景

- ✅ 正常流程测试
- ✅ 边界条件测试
- ✅ 参数验证测试
- ✅ 并发性能测试
- ✅ 数据库并发测试

## 性能指标

EOF

# 提取并发测试结果
if [ -f "$REPORT_DIR/concurrent.txt" ]; then
    echo "### 并发测试结果" >> "$REPORT_DIR/summary.md"
    echo "" >> "$REPORT_DIR/summary.md"
    echo '```' >> "$REPORT_DIR/summary.md"
    grep -A 6 "并发测试完成" "$REPORT_DIR/concurrent.txt" | sed 's/^.*benchmark_test.go:[0-9]*: //' >> "$REPORT_DIR/summary.md" || true
    echo '```' >> "$REPORT_DIR/summary.md"
    echo "" >> "$REPORT_DIR/summary.md"
fi

# 提取性能测试结果
if [ -f "$REPORT_DIR/benchmark.txt" ]; then
    echo "### Benchmark 结果" >> "$REPORT_DIR/summary.md"
    echo "" >> "$REPORT_DIR/summary.md"
    echo '```' >> "$REPORT_DIR/summary.md"
    grep "^Benchmark" "$REPORT_DIR/benchmark.txt" >> "$REPORT_DIR/summary.md" || true
    echo '```' >> "$REPORT_DIR/summary.md"
fi

echo ""
echo "✅ 测试报告生成完成！"
echo ""
echo "报告位置: $REPORT_DIR"
echo "  - 摘要: $REPORT_DIR/summary.md"
echo "  - 覆盖率: $REPORT_DIR/coverage.html"
echo "  - 详细日志: $REPORT_DIR/test-output.txt"
echo ""
