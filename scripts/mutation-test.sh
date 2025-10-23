#!/bin/bash
set -euo pipefail

# Mutation Testing Script for GRCTool
# This script runs mutation testing on critical packages and generates reports

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
MUTESTING_BIN="${HOME}/go/bin/go-mutesting"
CONFIG_FILE="${PROJECT_ROOT}/.mutesting.yml"
REPORTS_DIR="${PROJECT_ROOT}/mutation-reports"
MINIMUM_SCORE=70

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Critical packages to test
CRITICAL_PACKAGES=(
    "internal/tugboat"
    "internal/tools"
    "internal/services" 
    "internal/auth"
    "internal/domain"
    "internal/adapters"
    "internal/models"
    "internal/config"
)

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if go-mutesting is available
check_mutesting() {
    if [[ ! -x "${MUTESTING_BIN}" ]]; then
        log_error "go-mutesting not found at ${MUTESTING_BIN}"
        log_info "Install with: go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest"
        exit 1
    fi
}

# Function to create reports directory
setup_reports_dir() {
    mkdir -p "${REPORTS_DIR}"
    log_info "Reports will be saved to: ${REPORTS_DIR}"
}

# Function to run mutation testing on a specific package
run_mutation_test() {
    local package=$1
    local package_name=$(basename "${package}")
    local report_file="${REPORTS_DIR}/${package_name}-mutation.json"
    local html_file="${REPORTS_DIR}/${package_name}-mutation.html"
    
    log_info "Running mutation testing on package: ${package}"
    
    cd "${PROJECT_ROOT}"
    
    # Check if package has Go files
    if [[ ! $(find "${package}" -name "*.go" -not -name "*_test.go" 2>/dev/null | head -1) ]]; then
        log_warning "No Go files found in package: ${package}"
        return 0
    fi
    
    # Check if package has tests
    if [[ ! $(find "${package}" -name "*_test.go" 2>/dev/null | head -1) ]]; then
        log_warning "No test files found in package: ${package} - skipping mutation testing"
        return 0
    fi
    
    # Run mutation testing
    local cmd="${MUTESTING_BIN} --config=${CONFIG_FILE} --json-report=${report_file} --html-report=${html_file} ./${package}/..."
    
    log_info "Executing: ${cmd}"
    
    if timeout 300 ${cmd}; then
        log_success "Mutation testing completed for ${package}"
        return 0
    else
        local exit_code=$?
        if [[ ${exit_code} -eq 124 ]]; then
            log_error "Mutation testing timed out for ${package}"
        else
            log_error "Mutation testing failed for ${package} (exit code: ${exit_code})"
        fi
        return ${exit_code}
    fi
}

# Function to extract mutation score from JSON report
extract_mutation_score() {
    local report_file=$1
    
    if [[ ! -f "${report_file}" ]]; then
        echo "0"
        return
    fi
    
    # Extract mutation score using jq or manual parsing
    if command -v jq &> /dev/null; then
        local killed=$(jq -r '.summary.killed // 0' "${report_file}")
        local total=$(jq -r '.summary.total // 1' "${report_file}")
    else
        # Fallback to grep/awk if jq is not available
        local killed=$(grep -o '"killed":[0-9]*' "${report_file}" | cut -d: -f2 | head -1 || echo "0")
        local total=$(grep -o '"total":[0-9]*' "${report_file}" | cut -d: -f2 | head -1 || echo "1")
    fi
    
    if [[ ${total} -eq 0 ]]; then
        echo "0"
    else
        echo "scale=2; (${killed} * 100) / ${total}" | bc -l 2>/dev/null || echo "0"
    fi
}

# Function to generate summary report
generate_summary() {
    local summary_file="${REPORTS_DIR}/mutation-summary.txt"
    local failed_packages=()
    local total_score=0
    local package_count=0
    
    log_info "Generating mutation testing summary..."
    
    {
        echo "=========================================="
        echo "Mutation Testing Summary Report"
        echo "=========================================="
        echo "Generated: $(date)"
        echo "Project: GRCTool"
        echo "Minimum Score Threshold: ${MINIMUM_SCORE}%"
        echo ""
        echo "Package Results:"
        echo "------------------------------------------"
    } > "${summary_file}"
    
    for package in "${CRITICAL_PACKAGES[@]}"; do
        local package_name=$(basename "${package}")
        local report_file="${REPORTS_DIR}/${package_name}-mutation.json"
        local score=$(extract_mutation_score "${report_file}")
        
        printf "%-20s %8.2f%%\n" "${package_name}:" "${score}" >> "${summary_file}"
        
        # Check if score meets threshold
        if (( $(echo "${score} < ${MINIMUM_SCORE}" | bc -l) )); then
            failed_packages+=("${package_name}")
            echo -e "${RED}  ✗ ${package_name}: ${score}% (below threshold)${NC}"
        else
            echo -e "${GREEN}  ✓ ${package_name}: ${score}%${NC}"
        fi
        
        total_score=$(echo "scale=2; ${total_score} + ${score}" | bc -l)
        package_count=$((package_count + 1))
    done
    
    local average_score="0"
    if [[ ${package_count} -gt 0 ]]; then
        average_score=$(echo "scale=2; ${total_score} / ${package_count}" | bc -l)
    fi
    
    {
        echo ""
        echo "Summary:"
        echo "------------------------------------------"
        echo "Average Mutation Score: ${average_score}%"
        echo "Packages Tested: ${package_count}"
        echo "Packages Below Threshold: ${#failed_packages[@]}"
        
        if [[ ${#failed_packages[@]} -gt 0 ]]; then
            echo ""
            echo "Failed Packages (below ${MINIMUM_SCORE}%):"
            for pkg in "${failed_packages[@]}"; do
                echo "  - ${pkg}"
            done
        fi
    } >> "${summary_file}"
    
    log_info "Summary report saved to: ${summary_file}"
    
    # Display summary
    echo ""
    log_info "Mutation Testing Results:"
    echo -e "Average Score: ${BLUE}${average_score}%${NC}"
    echo -e "Packages Tested: ${BLUE}${package_count}${NC}"
    echo -e "Failed Packages: ${RED}${#failed_packages[@]}${NC}"
    
    # Return non-zero if any packages failed
    if [[ ${#failed_packages[@]} -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Function to generate HTML report index
generate_html_index() {
    local index_file="${REPORTS_DIR}/index.html"
    
    log_info "Generating HTML report index..."
    
    cat > "${index_file}" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GRCTool Mutation Testing Reports</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f4f4f4; padding: 20px; border-radius: 5px; }
        .package-list { margin-top: 20px; }
        .package-item { margin: 10px 0; padding: 10px; border: 1px solid #ddd; border-radius: 3px; }
        .score { font-weight: bold; }
        .pass { color: green; }
        .fail { color: red; }
        .timestamp { color: #666; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="header">
        <h1>GRCTool Mutation Testing Reports</h1>
        <p class="timestamp">Generated: $(date)</p>
    </div>
    
    <div class="package-list">
        <h2>Package Reports</h2>
EOF

    for package in "${CRITICAL_PACKAGES[@]}"; do
        local package_name=$(basename "${package}")
        local html_file="${package_name}-mutation.html"
        local report_file="${REPORTS_DIR}/${package_name}-mutation.json"
        local score=$(extract_mutation_score "${report_file}")
        
        local score_class="pass"
        if (( $(echo "${score} < ${MINIMUM_SCORE}" | bc -l) )); then
            score_class="fail"
        fi
        
        cat >> "${index_file}" << EOF
        <div class="package-item">
            <h3><a href="${html_file}">${package_name}</a></h3>
            <p>Mutation Score: <span class="score ${score_class}">${score}%</span></p>
        </div>
EOF
    done
    
    cat >> "${index_file}" << 'EOF'
    </div>
</body>
</html>
EOF

    log_success "HTML index created: ${index_file}"
}

# Main execution function
main() {
    local quick_mode=false
    local generate_report_only=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --quick)
                quick_mode=true
                shift
                ;;
            --report-only)
                generate_report_only=true
                shift
                ;;
            --help)
                echo "Usage: $0 [--quick] [--report-only] [--help]"
                echo "  --quick       Run mutation testing on changed files only"
                echo "  --report-only Generate reports from existing results"
                echo "  --help        Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    log_info "Starting mutation testing for GRCTool..."
    
    check_mutesting
    setup_reports_dir
    
    if [[ "${generate_report_only}" == "false" ]]; then
        # Run mutation testing on all critical packages
        local failed_count=0
        
        for package in "${CRITICAL_PACKAGES[@]}"; do
            if ! run_mutation_test "${package}"; then
                failed_count=$((failed_count + 1))
            fi
        done
        
        log_info "Mutation testing completed. Failed packages: ${failed_count}"
    fi
    
    # Generate reports
    generate_html_index
    
    if generate_summary; then
        log_success "All packages meet mutation score threshold!"
        exit 0
    else
        log_error "Some packages are below the mutation score threshold (${MINIMUM_SCORE}%)"
        exit 1
    fi
}

# Run main function
main "$@"