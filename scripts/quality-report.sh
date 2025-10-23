#!/bin/bash

# Quality Report Generator
# Generates a comprehensive testing quality report

set -e

echo "# Testing Quality Report"
echo "Generated: $(date)"
echo ""

# Code Coverage
echo "## 📊 Code Coverage"
if [ -f coverage.out ]; then
    coverage=$(go test -coverprofile=coverage.out ./... 2>/dev/null | grep -oE 'coverage: [0-9]+\.[0-9]+%' | tail -1 | awk '{print $2}')
    echo "**Overall Coverage:** $coverage"
    echo ""
    
    # Package breakdown
    echo "### Package Coverage:"
    echo '```'
    go test -coverprofile=coverage.out ./... 2>/dev/null | grep "coverage:" | sort -k3 -rn | head -10
    echo '```'
else
    echo "⚠️ No coverage data available"
fi
echo ""

# Mutation Testing
echo "## 🧬 Mutation Testing"
if [ -f mutation-reports/gremlins-summary.txt ]; then
    echo '```'
    cat mutation-reports/gremlins-summary.txt | head -20
    echo '```'
else
    echo "Running quick mutation analysis..."
    if command -v gremlins &> /dev/null; then
        gremlins unleash --dry-run ./internal/config 2>/dev/null | grep -E "(efficacy|coverage)" || echo "⚠️ Mutation testing not configured"
    else
        echo "⚠️ Mutation testing tools not installed"
    fi
fi
echo ""

# Benchmark Performance
echo "## ⚡ Performance Benchmarks"
if [ -f benchmarks/current.txt ]; then
    echo "### Top 10 Operations by Time:"
    echo '```'
    grep "Benchmark" benchmarks/current.txt | sort -k3 -n | head -10
    echo '```'
    
    # Check for regressions
    if [ -f benchmarks/baseline.txt ]; then
        echo ""
        echo "### Performance Changes:"
        echo "Comparing with baseline..."
        # Simple comparison - in real scenario use benchstat
        echo '```'
        diff -u benchmarks/baseline.txt benchmarks/current.txt | grep "^[+-]Benchmark" | head -10 || echo "No significant changes"
        echo '```'
    fi
else
    echo "⚠️ No benchmark data available"
fi
echo ""

# Test Execution Summary
echo "## ✅ Test Execution Summary"
echo '```'
go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL)" | tail -20
echo '```'
echo ""

# Test Statistics
echo "## 📈 Test Statistics"
test_files=$(find . -name "*_test.go" -not -path "./vendor/*" | wc -l)
test_functions=$(grep -r "^func Test" --include="*_test.go" | wc -l)
benchmark_functions=$(grep -r "^func Benchmark" --include="*_test.go" | wc -l)
table_tests=$(grep -r "tests.*:=.*map\[string\]" --include="*_test.go" | wc -l)

echo "- **Test Files:** $test_files"
echo "- **Test Functions:** $test_functions"
echo "- **Benchmark Functions:** $benchmark_functions"
echo "- **Table-Driven Tests:** $table_tests"
echo ""

# Test Execution Time
echo "## ⏱️ Test Execution Time"
echo "Running quick test suite..."
start_time=$(date +%s)
go test -short ./... > /dev/null 2>&1
end_time=$(date +%s)
execution_time=$((end_time - start_time))
echo "**Total Execution Time:** ${execution_time}s"
echo ""

# Quality Metrics Summary
echo "## 🎯 Quality Metrics Summary"
echo ""
echo "| Metric | Target | Current | Status |"
echo "|--------|--------|---------|--------|"

# Coverage check
if [ -f coverage.out ]; then
    coverage_num=$(echo $coverage | sed 's/%//')
    if (( $(echo "$coverage_num >= 80" | bc -l) )); then
        echo "| Code Coverage | ≥80% | $coverage | ✅ |"
    else
        echo "| Code Coverage | ≥80% | $coverage | ❌ |"
    fi
else
    echo "| Code Coverage | ≥80% | Unknown | ⚠️ |"
fi

# Mutation score check
if [ -f mutation-reports/gremlins-summary.txt ]; then
    mutation_score=$(grep "Test Efficacy" mutation-reports/gremlins-summary.txt | grep -oE '[0-9]+\.[0-9]+' | head -1)
    if [ -n "$mutation_score" ]; then
        if (( $(echo "$mutation_score >= 70" | bc -l) )); then
            echo "| Mutation Score | ≥70% | ${mutation_score}% | ✅ |"
        else
            echo "| Mutation Score | ≥70% | ${mutation_score}% | ❌ |"
        fi
    else
        echo "| Mutation Score | ≥70% | Unknown | ⚠️ |"
    fi
else
    echo "| Mutation Score | ≥70% | Unknown | ⚠️ |"
fi

# Test execution time
if [ "$execution_time" -lt 5 ]; then
    echo "| Test Execution | <5s | ${execution_time}s | ✅ |"
else
    echo "| Test Execution | <5s | ${execution_time}s | ⚠️ |"
fi

# Benchmark coverage
if [ "$benchmark_functions" -gt 30 ]; then
    echo "| Benchmark Coverage | >30 | $benchmark_functions | ✅ |"
else
    echo "| Benchmark Coverage | >30 | $benchmark_functions | ⚠️ |"
fi

echo ""

# Recommendations
echo "## 💡 Recommendations"
echo ""

# Check for low coverage packages
if [ -f coverage.out ]; then
    low_coverage=$(go test -coverprofile=coverage.out ./... 2>/dev/null | grep "coverage:" | awk '$3 ~ /%$/ && substr($3, 1, length($3)-1) < 50 {print $1}' | head -5)
    if [ -n "$low_coverage" ]; then
        echo "### Packages Needing Coverage Improvement:"
        echo "$low_coverage" | while read pkg; do
            echo "- $pkg"
        done
        echo ""
    fi
fi

# Check for missing benchmarks
packages_without_benchmarks=$(for dir in internal/*/; do
    if [ -d "$dir" ]; then
        pkg=$(basename "$dir")
        if ! grep -q "^func Benchmark" "$dir"*_test.go 2>/dev/null; then
            echo "- internal/$pkg"
        fi
    fi
done)

if [ -n "$packages_without_benchmarks" ]; then
    echo "### Packages Missing Benchmarks:"
    echo "$packages_without_benchmarks"
    echo ""
fi

echo "---"
echo "*Report generated by GRCTool Testing Infrastructure*"