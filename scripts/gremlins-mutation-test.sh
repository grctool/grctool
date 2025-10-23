#!/bin/bash
set -euo pipefail

# Gremlins Mutation Testing Script for GRCTool
# This script runs mutation testing using the modern Gremlins tool

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
GREMLINS_BIN="${HOME}/go/bin/gremlins"
CONFIG_FILE="${PROJECT_ROOT}/.gremlins.yml"
REPORTS_DIR="${PROJECT_ROOT}/mutation-reports"
MINIMUM_EFFICACY=70
MINIMUM_COVERAGE=80

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Critical packages to test with higher thresholds
CRITICAL_PACKAGES=(
    "internal/auth"
    "internal/tugboat" 
    "internal/services"
)

# Standard packages with default thresholds
STANDARD_PACKAGES=(
    "internal/config"
    "internal/domain"
    "internal/models"
    "internal/adapters"
)

# Utility packages with relaxed thresholds
UTILITY_PACKAGES=(
    "internal/logger"
    "internal/utils"
    "internal/formatters"
    "internal/appcontext"
)

# All packages for comprehensive testing
ALL_PACKAGES=("${CRITICAL_PACKAGES[@]}" "${STANDARD_PACKAGES[@]}" "${UTILITY_PACKAGES[@]}")

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

# Function to check if Gremlins is available
check_gremlins() {
    if [[ ! -x "${GREMLINS_BIN}" ]]; then
        log_warning "Gremlins not found at ${GREMLINS_BIN}"
        log_info "Installing Gremlins..."
        if go install github.com/go-gremlins/gremlins/cmd/gremlins@latest; then
            log_success "Gremlins installed successfully"
        else
            log_error "Failed to install Gremlins"
            log_info "Install manually with: go install github.com/go-gremlins/gremlins/cmd/gremlins@latest"
            exit 1
        fi
    fi
}

# Function to create reports directory
setup_reports_dir() {
    mkdir -p "${REPORTS_DIR}"
    log_info "Reports will be saved to: ${REPORTS_DIR}"
}

# Function to get package thresholds based on package type
get_package_thresholds() {
    local package=$1
    local efficacy_threshold=${MINIMUM_EFFICACY}
    local coverage_threshold=${MINIMUM_COVERAGE}
    
    # Check if package is critical
    for critical_pkg in "${CRITICAL_PACKAGES[@]}"; do
        if [[ "${package}" == *"${critical_pkg}"* ]]; then
            efficacy_threshold=80  # Higher threshold for critical packages
            coverage_threshold=85
            break
        fi
    done
    
    # Check if package is utility (lower thresholds)
    for utility_pkg in "${UTILITY_PACKAGES[@]}"; do
        if [[ "${package}" == *"${utility_pkg}"* ]]; then
            efficacy_threshold=60  # Lower threshold for utility packages
            coverage_threshold=70
            break
        fi
    done
    
    echo "${efficacy_threshold} ${coverage_threshold}"
}

# Function to run mutation testing on a specific package
run_mutation_test() {
    local package=$1
    local package_name=$(basename "${package}")
    local json_report="${REPORTS_DIR}/${package_name}-gremlins.json"
    local text_report="${REPORTS_DIR}/${package_name}-gremlins.txt"
    
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
    
    # Get package-specific thresholds
    local thresholds=($(get_package_thresholds "${package}"))
    local efficacy_threshold=${thresholds[0]}
    local coverage_threshold=${thresholds[1]}
    
    log_info "Using thresholds - Efficacy: ${efficacy_threshold}%, Coverage: ${coverage_threshold}%"
    
    # Run mutation testing with Gremlins
    local gremlins_cmd="${GREMLINS_BIN} unleash"
    gremlins_cmd+=" --output=${json_report}"
    gremlins_cmd+=" --threshold-efficacy=0.$(printf "%02d" ${efficacy_threshold})"
    gremlins_cmd+=" --threshold-mcover=0.$(printf "%02d" ${coverage_threshold})"
    gremlins_cmd+=" --workers=0"
    gremlins_cmd+=" ${package}"
    
    log_info "Executing: ${gremlins_cmd}"
    
    # Run the command and capture output
    if timeout 300 ${gremlins_cmd} > "${text_report}" 2>&1; then
        log_success "Mutation testing completed for ${package}"
        
        # Extract scores from output
        local efficacy=$(grep "Test efficacy:" "${text_report}" | sed 's/.*Test efficacy: \([0-9.]*\)%.*/\1/' || echo "0")
        local coverage=$(grep "Mutator coverage:" "${text_report}" | sed 's/.*Mutator coverage: \([0-9.]*\)%.*/\1/' || echo "0")
        
        log_info "Results - Efficacy: ${efficacy}%, Coverage: ${coverage}%"
        
        # Check thresholds
        local efficacy_int=$(echo "${efficacy}" | cut -d. -f1)
        local coverage_int=$(echo "${coverage}" | cut -d. -f1)
        
        local failed=false
        if [[ ${efficacy_int} -lt ${efficacy_threshold} ]]; then
            log_error "Efficacy ${efficacy}% below threshold ${efficacy_threshold}%"
            failed=true
        fi
        
        if [[ ${coverage_int} -lt ${coverage_threshold} ]]; then
            log_error "Coverage ${coverage}% below threshold ${coverage_threshold}%"
            failed=true
        fi
        
        if [[ "${failed}" == "true" ]]; then
            return 1
        else
            return 0
        fi
    else
        local exit_code=$?
        
        if [[ ${exit_code} -eq 124 ]]; then
            log_error "Mutation testing timed out for ${package}"
        else
            log_error "Mutation testing failed for ${package} (exit code: ${exit_code})"
            # Show last few lines of output for debugging
            log_info "Last few lines of output:"
            tail -10 "${text_report}" || true
        fi
        return ${exit_code}
    fi
}

# Function to run dry-run analysis 
run_dry_run() {
    local package=$1
    local package_name=$(basename "${package}")
    
    log_info "Running dry-run analysis on package: ${package}"
    
    cd "${PROJECT_ROOT}"
    
    if timeout 30 "${GREMLINS_BIN}" unleash --dry-run "${package}" 2>/dev/null; then
        log_success "Dry-run completed for ${package}"
        return 0
    else
        log_warning "Dry-run failed for ${package}"
        return 1
    fi
}

# Function to generate summary report
generate_summary() {
    local summary_file="${REPORTS_DIR}/gremlins-summary.txt"
    local failed_packages=()
    local total_efficacy=0
    local total_coverage=0
    local package_count=0
    
    log_info "Generating mutation testing summary..."
    
    {
        echo "=========================================="
        echo "Gremlins Mutation Testing Summary Report"
        echo "=========================================="
        echo "Generated: $(date)"
        echo "Project: GRCTool"
        echo "Tool: Gremlins"
        echo ""
        echo "Package Results:"
        echo "------------------------------------------"
        printf "%-20s %12s %12s %10s\n" "Package" "Efficacy" "Coverage" "Status"
        echo "------------------------------------------"
    } > "${summary_file}"
    
    for package in "${ALL_PACKAGES[@]}"; do
        local package_name=$(basename "${package}")
        local text_report="${REPORTS_DIR}/${package_name}-gremlins.txt"
        
        if [[ -f "${text_report}" ]]; then
            local efficacy=$(grep "Test efficacy:" "${text_report}" | sed 's/.*Test efficacy: \([0-9.]*\)%.*/\1/' || echo "0")
            local coverage=$(grep "Mutator coverage:" "${text_report}" | sed 's/.*Mutator coverage: \([0-9.]*\)%.*/\1/' || echo "0")
            local thresholds=($(get_package_thresholds "${package}"))
            local efficacy_threshold=${thresholds[0]}
            local coverage_threshold=${thresholds[1]}
            
            printf "%-20s %11s%% %11s%% " "${package_name}" "${efficacy}" "${coverage}" >> "${summary_file}"
            
            # Check if package meets thresholds
            local efficacy_int=$(echo "${efficacy}" | cut -d. -f1)
            local coverage_int=$(echo "${coverage}" | cut -d. -f1)
            
            if [[ ${efficacy_int} -ge ${efficacy_threshold} && ${coverage_int} -ge ${coverage_threshold} ]]; then
                echo "PASS" >> "${summary_file}"
                echo -e "  ✓ ${GREEN}${package_name}${NC}: Efficacy ${efficacy}%, Coverage ${coverage}%"
            else
                echo "FAIL" >> "${summary_file}"
                failed_packages+=("${package_name}")
                echo -e "  ✗ ${RED}${package_name}${NC}: Efficacy ${efficacy}%, Coverage ${coverage}% (thresholds: ${efficacy_threshold}%/${coverage_threshold}%)"
            fi
            
            # Add to totals
            total_efficacy=$(echo "scale=2; ${total_efficacy} + ${efficacy}" | bc -l)
            total_coverage=$(echo "scale=2; ${total_coverage} + ${coverage}" | bc -l)
            package_count=$((package_count + 1))
        else
            printf "%-20s %11s %11s %10s\n" "${package_name}" "N/A" "N/A" "SKIP" >> "${summary_file}"
            log_warning "No results found for ${package_name}"
        fi
    done
    
    local avg_efficacy="0"
    local avg_coverage="0"
    if [[ ${package_count} -gt 0 ]]; then
        avg_efficacy=$(echo "scale=2; ${total_efficacy} / ${package_count}" | bc -l)
        avg_coverage=$(echo "scale=2; ${total_coverage} / ${package_count}" | bc -l)
    fi
    
    {
        echo ""
        echo "Summary:"
        echo "------------------------------------------"
        echo "Average Test Efficacy: ${avg_efficacy}%"
        echo "Average Mutant Coverage: ${avg_coverage}%"
        echo "Packages Tested: ${package_count}"
        echo "Packages Failed: ${#failed_packages[@]}"
        
        if [[ ${#failed_packages[@]} -gt 0 ]]; then
            echo ""
            echo "Failed Packages:"
            for pkg in "${failed_packages[@]}"; do
                echo "  - ${pkg}"
            done
            echo ""
            echo "Recommendations:"
            echo "1. Review test coverage for failed packages"
            echo "2. Add tests for edge cases and error conditions"  
            echo "3. Ensure all code paths are tested"
            echo "4. Use 'gremlins unleash --dry-run <package>' to see mutation opportunities"
        fi
    } >> "${summary_file}"
    
    log_info "Summary report saved to: ${summary_file}"
    
    # Display summary
    echo ""
    log_info "Mutation Testing Results:"
    echo -e "Average Efficacy: ${BLUE}${avg_efficacy}%${NC}"
    echo -e "Average Coverage: ${BLUE}${avg_coverage}%${NC}"
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
    <title>GRCTool Gremlins Mutation Testing Reports</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f4f4f4; padding: 20px; border-radius: 5px; }
        .package-list { margin-top: 20px; }
        .package-item { margin: 10px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .score { font-weight: bold; margin: 5px 0; }
        .efficacy { color: #2196F3; }
        .coverage { color: #4CAF50; }
        .pass { color: green; }
        .fail { color: red; }
        .timestamp { color: #666; font-size: 0.9em; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f5f5f5; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🧪 GRCTool Gremlins Mutation Testing Reports</h1>
        <p class="timestamp">Generated: $(date)</p>
        <p>Mutation testing results using the Gremlins tool</p>
    </div>
    
    <table>
        <thead>
            <tr>
                <th>Package</th>
                <th>Test Efficacy</th>
                <th>Mutant Coverage</th>
                <th>Status</th>
                <th>Report</th>
            </tr>
        </thead>
        <tbody>
EOF

    for package in "${ALL_PACKAGES[@]}"; do
        local package_name=$(basename "${package}")
        local text_report="${package_name}-gremlins.txt"
        local json_report="${package_name}-gremlins.json"
        local report_file="${REPORTS_DIR}/${package_name}-gremlins.txt"
        
        if [[ -f "${report_file}" ]]; then
            local efficacy=$(grep "Test efficacy:" "${report_file}" | sed 's/.*Test efficacy: \([0-9.]*\)%.*/\1/' || echo "0")
            local coverage=$(grep "Mutator coverage:" "${report_file}" | sed 's/.*Mutator coverage: \([0-9.]*\)%.*/\1/' || echo "0")
            local thresholds=($(get_package_thresholds "${package}"))
            local efficacy_threshold=${thresholds[0]}
            local coverage_threshold=${thresholds[1]}
            
            local efficacy_int=$(echo "${efficacy}" | cut -d. -f1)
            local coverage_int=$(echo "${coverage}" | cut -d. -f1)
            
            local status="pass"
            local status_class="pass"
            if [[ ${efficacy_int} -lt ${efficacy_threshold} || ${coverage_int} -lt ${coverage_threshold} ]]; then
                status="fail"
                status_class="fail"
            fi
            
            cat >> "${index_file}" << EOF
            <tr>
                <td><strong>${package_name}</strong></td>
                <td class="efficacy">${efficacy}%</td>
                <td class="coverage">${coverage}%</td>
                <td class="${status_class}">${status^^}</td>
                <td><a href="${text_report}">View Report</a></td>
            </tr>
EOF
        else
            cat >> "${index_file}" << EOF
            <tr>
                <td><strong>${package_name}</strong></td>
                <td>N/A</td>
                <td>N/A</td>
                <td>SKIP</td>
                <td>No report available</td>
            </tr>
EOF
        fi
    done
    
    cat >> "${index_file}" << 'EOF'
        </tbody>
    </table>
    
    <div style="margin-top: 30px;">
        <h3>📊 Understanding the Metrics</h3>
        <p><strong>Test Efficacy:</strong> Percentage of mutants killed by your tests (killed / (killed + lived))</p>
        <p><strong>Mutant Coverage:</strong> Percentage of mutants that are runnable (have test coverage)</p>
        <p><strong>Thresholds:</strong> Critical packages: 80%/85%, Standard packages: 70%/80%, Utility packages: 60%/70%</p>
    </div>
    
    <div style="margin-top: 20px; padding: 15px; background-color: #fff3cd; border-radius: 5px;">
        <h4>💡 Improving Scores</h4>
        <ul>
            <li>Add tests for edge cases and error conditions</li>
            <li>Test all code paths and conditional branches</li>
            <li>Verify boundary conditions and arithmetic operations</li>
            <li>Use <code>gremlins unleash --dry-run &lt;package&gt;</code> to see mutation opportunities</li>
        </ul>
    </div>
</body>
</html>
EOF

    log_success "HTML index created: ${index_file}"
}

# Main execution function
main() {
    local dry_run=false
    local generate_report_only=false
    local quick_mode=false
    local target_packages=()
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run)
                dry_run=true
                shift
                ;;
            --report-only)
                generate_report_only=true
                shift
                ;;
            --quick)
                quick_mode=true
                shift
                ;;
            --package)
                target_packages+=("$2")
                shift 2
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo "Options:"
                echo "  --dry-run         Run analysis only, no mutation testing"
                echo "  --report-only     Generate reports from existing results"
                echo "  --quick           Quick mode - test fewer packages"
                echo "  --package PKG     Test specific package (can be used multiple times)"
                echo "  --help            Show this help message"
                echo ""
                echo "Examples:"
                echo "  $0                      # Run full mutation testing"
                echo "  $0 --dry-run            # Analyze mutations without testing"
                echo "  $0 --package internal/auth  # Test specific package"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    log_info "Starting Gremlins mutation testing for GRCTool..."
    
    check_gremlins
    setup_reports_dir
    
    # Determine which packages to test
    local packages_to_test=()
    if [[ ${#target_packages[@]} -gt 0 ]]; then
        packages_to_test=("${target_packages[@]}")
        log_info "Testing specific packages: ${packages_to_test[*]}"
    elif [[ "${quick_mode}" == "true" ]]; then
        packages_to_test=("${CRITICAL_PACKAGES[@]}" "${STANDARD_PACKAGES[@]}")
        log_info "Quick mode - testing critical and standard packages"
    else
        packages_to_test=("${ALL_PACKAGES[@]}")
        log_info "Testing all packages"
    fi
    
    if [[ "${generate_report_only}" == "false" ]]; then
        # Run mutation testing or dry-run
        local failed_count=0
        
        for package in "${packages_to_test[@]}"; do
            if [[ "${dry_run}" == "true" ]]; then
                run_dry_run "${package}" || failed_count=$((failed_count + 1))
            else
                run_mutation_test "${package}" || failed_count=$((failed_count + 1))
            fi
        done
        
        if [[ "${dry_run}" == "true" ]]; then
            log_info "Dry-run completed. Failed packages: ${failed_count}"
            exit 0
        else
            log_info "Mutation testing completed. Failed packages: ${failed_count}"
        fi
    fi
    
    # Generate reports
    generate_html_index
    
    if generate_summary; then
        log_success "All packages meet mutation testing thresholds!"
        exit 0
    else
        log_error "Some packages are below mutation testing thresholds"
        log_info "Check the reports in ${REPORTS_DIR}/ for details"
        exit 1
    fi
}

# Run main function
main "$@"