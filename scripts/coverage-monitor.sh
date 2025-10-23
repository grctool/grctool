#!/bin/bash

# Coverage monitoring script for GRCTool
# Generates coverage for all packages and checks thresholds

set -e

# Configuration
COVERAGE_FILE="coverage.out"
HTML_FILE="coverage.html"
CRITICAL_THRESHOLD=70
GENERAL_THRESHOLD=50
MINIMUM_THRESHOLD=20

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Critical packages that need higher coverage
CRITICAL_PACKAGES=(
    "github.com/7thsense/isms/grctool/internal/tugboat"
    "github.com/7thsense/isms/grctool/internal/storage"
    "github.com/7thsense/isms/grctool/internal/services/evidence"
    "github.com/7thsense/isms/grctool/internal/auth"
    "github.com/7thsense/isms/grctool/internal/services"
)

print_status $BLUE "=== GRCTool Coverage Analysis ==="
echo

# Generate coverage data
print_status $BLUE "Generating coverage data..."
if go test -coverprofile="$COVERAGE_FILE" ./... > /dev/null 2>&1; then
    print_status $GREEN "✓ Coverage data generated successfully"
else
    print_status $RED "⚠ Coverage generation completed with some test failures"
fi

# Generate HTML report
print_status $BLUE "Generating HTML coverage report..."
go tool cover -html="$COVERAGE_FILE" -o "$HTML_FILE"
print_status $GREEN "✓ HTML report generated: $HTML_FILE"

# Get overall coverage
OVERALL_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | tail -1 | awk '{print $3}' | sed 's/%//')
echo
print_status $BLUE "=== Overall Coverage: ${OVERALL_COVERAGE}% ==="

# Check overall threshold
if (( $(echo "$OVERALL_COVERAGE < $MINIMUM_THRESHOLD" | bc -l) )); then
    print_status $RED "❌ Overall coverage (${OVERALL_COVERAGE}%) is below minimum threshold (${MINIMUM_THRESHOLD}%)"
    OVERALL_STATUS="FAIL"
else
    print_status $GREEN "✓ Overall coverage (${OVERALL_COVERAGE}%) meets minimum threshold"
    OVERALL_STATUS="PASS"
fi

echo
print_status $BLUE "=== Package Coverage Analysis ==="

# Get package coverage breakdown
COVERAGE_DATA=$(go test -coverprofile="$COVERAGE_FILE" ./... 2>/dev/null | grep "coverage:" | sort -k3 -nr)

# Arrays to track packages by status
CRITICAL_BELOW=()
GENERAL_BELOW=()
PACKAGES_GOOD=()
PACKAGES_EXCELLENT=()

# Process each package
while IFS= read -r line; do
    if [[ $line == *"coverage:"* ]]; then
        # Extract package and coverage
        PACKAGE=$(echo "$line" | awk '{print $2}')
        COVERAGE=$(echo "$line" | awk '{print $NF}' | sed 's/%//' | sed 's/.*:///')
        
        # Skip empty coverage values
        if [[ -z "$COVERAGE" || "$COVERAGE" == "0.0" ]]; then
            continue
        fi
        
        # Check if it's a critical package
        IS_CRITICAL=false
        for critical_pkg in "${CRITICAL_PACKAGES[@]}"; do
            if [[ "$PACKAGE" == "$critical_pkg" ]]; then
                IS_CRITICAL=true
                break
            fi
        done
        
        # Determine status based on thresholds
        if $IS_CRITICAL; then
            if (( $(echo "$COVERAGE < $CRITICAL_THRESHOLD" | bc -l) )); then
                CRITICAL_BELOW+=("$PACKAGE:$COVERAGE")
                print_status $RED "❌ CRITICAL: $PACKAGE (${COVERAGE}%) - Below critical threshold (${CRITICAL_THRESHOLD}%)"
            else
                PACKAGES_EXCELLENT+=("$PACKAGE:$COVERAGE")
                print_status $GREEN "✓ CRITICAL: $PACKAGE (${COVERAGE}%) - Meets critical threshold"
            fi
        else
            if (( $(echo "$COVERAGE < $GENERAL_THRESHOLD" | bc -l) )); then
                GENERAL_BELOW+=("$PACKAGE:$COVERAGE")
                print_status $YELLOW "⚠ $PACKAGE (${COVERAGE}%) - Below general threshold (${GENERAL_THRESHOLD}%)"
            else
                PACKAGES_GOOD+=("$PACKAGE:$COVERAGE")
                print_status $GREEN "✓ $PACKAGE (${COVERAGE}%) - Meets general threshold"
            fi
        fi
    fi
done <<< "$COVERAGE_DATA"

# Check for packages with 0% coverage (no tests)
print_status $BLUE "=== Packages with No Test Coverage ==="
ZERO_COVERAGE=$(go test ./... 2>/dev/null | grep "coverage: 0.0%" | awk '{print $2}')
if [[ -n "$ZERO_COVERAGE" ]]; then
    while IFS= read -r package; do
        # Check if it's a critical package
        IS_CRITICAL=false
        for critical_pkg in "${CRITICAL_PACKAGES[@]}"; do
            if [[ "$package" == "$critical_pkg" ]]; then
                IS_CRITICAL=true
                break
            fi
        done
        
        if $IS_CRITICAL; then
            print_status $RED "❌ CRITICAL: $package (0.0%) - No tests found!"
            CRITICAL_BELOW+=("$package:0.0")
        else
            print_status $YELLOW "⚠ $package (0.0%) - No tests found"
        fi
    done <<< "$ZERO_COVERAGE"
else
    print_status $GREEN "✓ All packages have some test coverage"
fi

echo
print_status $BLUE "=== Coverage Summary ==="
echo "📊 Overall Coverage: ${OVERALL_COVERAGE}%"
echo "✅ Packages meeting thresholds: $((${#PACKAGES_GOOD[@]} + ${#PACKAGES_EXCELLENT[@]}))"
echo "⚠️  Packages below general threshold: ${#GENERAL_BELOW[@]}"
echo "❌ Critical packages below threshold: ${#CRITICAL_BELOW[@]}"

echo
print_status $BLUE "=== Recommendations ==="

if [[ ${#CRITICAL_BELOW[@]} -gt 0 ]]; then
    print_status $RED "🚨 URGENT: Add tests for critical packages:"
    for pkg_coverage in "${CRITICAL_BELOW[@]}"; do
        pkg=$(echo "$pkg_coverage" | cut -d: -f1)
        cov=$(echo "$pkg_coverage" | cut -d: -f2)
        echo "   - $pkg (currently ${cov}%)"
    done
fi

if [[ ${#GENERAL_BELOW[@]} -gt 0 ]]; then
    print_status $YELLOW "⚠️  Consider improving coverage for:"
    for pkg_coverage in "${GENERAL_BELOW[@]}"; do
        pkg=$(echo "$pkg_coverage" | cut -d: -f1)
        cov=$(echo "$pkg_coverage" | cut -d: -f2)
        echo "   - $pkg (currently ${cov}%)"
    done
fi

echo
print_status $BLUE "=== Next Steps ==="
echo "1. Focus on critical packages first (tugboat, storage, services/evidence, auth)"
echo "2. Add unit tests for packages with 0% coverage"
echo "3. Improve integration test coverage for existing functionality"
echo "4. Set up coverage tracking in CI/CD pipeline"

# Exit with error if critical packages are below threshold
if [[ ${#CRITICAL_BELOW[@]} -gt 0 ]]; then
    echo
    print_status $RED "❌ COVERAGE CHECK FAILED: Critical packages below threshold"
    exit 1
else
    echo
    print_status $GREEN "✅ COVERAGE CHECK PASSED: All critical packages meet requirements"
    exit 0
fi