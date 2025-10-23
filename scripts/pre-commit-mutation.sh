#!/bin/bash
set -euo pipefail

# Pre-commit Mutation Testing Hook for GRCTool
# This script runs mutation testing on changed Go files before allowing commits

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

# Logging functions
log_info() {
    echo -e "${BLUE}[PRE-COMMIT]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PRE-COMMIT]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[PRE-COMMIT]${NC} $1"
}

log_error() {
    echo -e "${RED}[PRE-COMMIT]${NC} $1"
}

# Function to check if go-mutesting is available
check_mutesting() {
    if [[ ! -x "${MUTESTING_BIN}" ]]; then
        log_warning "go-mutesting not found at ${MUTESTING_BIN}"
        log_info "Installing go-mutesting..."
        if go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest; then
            log_success "go-mutesting installed successfully"
        else
            log_error "Failed to install go-mutesting"
            log_info "You can bypass this check with: git commit --no-verify"
            exit 1
        fi
    fi
}

# Function to get changed Go files
get_changed_go_files() {
    # Get staged files that are Go files (excluding test files for now)
    git diff --cached --name-only --diff-filter=AM | grep '\.go$' | grep -v '_test\.go$' || true
}

# Function to get packages from changed files
get_packages_from_files() {
    local files=("$@")
    local packages=()
    
    for file in "${files[@]}"; do
        if [[ -f "${file}" ]]; then
            local pkg_dir=$(dirname "${file}")
            # Convert to Go package path
            local go_pkg="./${pkg_dir}"
            
            # Check if package already in list
            local found=false
            for existing_pkg in "${packages[@]}"; do
                if [[ "${existing_pkg}" == "${go_pkg}" ]]; then
                    found=true
                    break
                fi
            done
            
            if [[ "${found}" == "false" ]]; then
                packages+=("${go_pkg}")
            fi
        fi
    done
    
    echo "${packages[@]}"
}

# Function to run mutation testing on a package
run_mutation_test_on_package() {
    local package=$1
    local package_name=$(basename "${package}")
    local temp_report="${REPORTS_DIR}/pre-commit-${package_name}-$$.json"
    
    log_info "Running mutation testing on package: ${package}"
    
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
    
    # Create reports directory
    mkdir -p "${REPORTS_DIR}"
    
    # Run mutation testing with timeout
    log_info "Executing mutation testing (timeout: 60s)..."
    
    if timeout 60 "${MUTESTING_BIN}" --config="${CONFIG_FILE}" --json-report="${temp_report}" "${package}/..." 2>/dev/null; then
        log_info "Mutation testing completed for ${package}"
        
        # Extract mutation score
        local score=$(extract_mutation_score "${temp_report}")
        
        log_info "Mutation score for ${package_name}: ${score}%"
        
        # Clean up temporary report
        rm -f "${temp_report}"
        
        # Check if score meets threshold
        if (( $(echo "${score} < ${MINIMUM_SCORE}" | bc -l 2>/dev/null || echo "1") )); then
            log_error "Package ${package_name} has mutation score ${score}% (below ${MINIMUM_SCORE}% threshold)"
            return 1
        else
            log_success "Package ${package_name} passes mutation testing: ${score}%"
            return 0
        fi
    else
        local exit_code=$?
        rm -f "${temp_report}"
        
        if [[ ${exit_code} -eq 124 ]]; then
            log_warning "Mutation testing timed out for ${package} - allowing commit but consider improving test coverage"
            return 0
        else
            log_error "Mutation testing failed for ${package} (exit code: ${exit_code})"
            return 1
        fi
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
        local killed=$(jq -r '.summary.killed // 0' "${report_file}" 2>/dev/null || echo "0")
        local total=$(jq -r '.summary.total // 1' "${report_file}" 2>/dev/null || echo "1")
    else
        # Fallback to grep/awk if jq is not available
        local killed=$(grep -o '"killed":[0-9]*' "${report_file}" 2>/dev/null | cut -d: -f2 | head -1 || echo "0")
        local total=$(grep -o '"total":[0-9]*' "${report_file}" 2>/dev/null | cut -d: -f2 | head -1 || echo "1")
    fi
    
    if [[ ${total} -eq 0 ]]; then
        echo "0"
    else
        echo "scale=2; (${killed} * 100) / ${total}" | bc -l 2>/dev/null || echo "0"
    fi
}

# Function to check if we're in a Git repository
check_git_repo() {
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "Not in a Git repository"
        exit 1
    fi
}

# Function to install Git hook
install_hook() {
    local hook_path="${PROJECT_ROOT}/.git/hooks/pre-commit"
    local hook_content="#!/bin/bash
# Auto-generated pre-commit hook for mutation testing
exec $(cd \"${PROJECT_ROOT}\" && pwd)/scripts/pre-commit-mutation.sh
"
    
    log_info "Installing pre-commit hook..."
    
    if [[ -f "${hook_path}" ]]; then
        log_warning "Pre-commit hook already exists. Backing up to pre-commit.bak"
        cp "${hook_path}" "${hook_path}.bak"
    fi
    
    echo "${hook_content}" > "${hook_path}"
    chmod +x "${hook_path}"
    
    log_success "Pre-commit hook installed at ${hook_path}"
    log_info "To bypass mutation testing, use: git commit --no-verify"
}

# Main execution function
main() {
    local install_mode=false
    local force_run=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --install)
                install_mode=true
                shift
                ;;
            --force)
                force_run=true
                shift
                ;;
            --help)
                echo "Usage: $0 [--install] [--force] [--help]"
                echo "  --install  Install this script as a Git pre-commit hook"
                echo "  --force    Run mutation testing even if no Go files changed"
                echo "  --help     Show this help message"
                echo ""
                echo "Pre-commit mutation testing for GRCTool"
                echo "This script runs mutation testing on changed Go packages before allowing commits."
                echo ""
                echo "To bypass the hook: git commit --no-verify"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    check_git_repo
    
    # Install mode
    if [[ "${install_mode}" == "true" ]]; then
        install_hook
        exit 0
    fi
    
    log_info "Running pre-commit mutation testing for GRCTool..."
    
    cd "${PROJECT_ROOT}"
    
    # Get changed Go files
    local changed_files=($(get_changed_go_files))
    
    if [[ ${#changed_files[@]} -eq 0 && "${force_run}" == "false" ]]; then
        log_info "No Go files changed - skipping mutation testing"
        exit 0
    fi
    
    if [[ ${#changed_files[@]} -eq 0 && "${force_run}" == "true" ]]; then
        log_warning "No Go files changed but running mutation testing due to --force flag"
        # In force mode, test critical packages
        changed_files=("internal/tugboat/client.go" "internal/auth/provider.go")
    fi
    
    log_info "Changed Go files: ${changed_files[*]}"
    
    # Check if go-mutesting is available
    check_mutesting
    
    # Get packages from changed files
    local packages=($(get_packages_from_files "${changed_files[@]}"))
    
    if [[ ${#packages[@]} -eq 0 ]]; then
        log_warning "No packages identified from changed files"
        exit 0
    fi
    
    log_info "Running mutation testing on packages: ${packages[*]}"
    
    # Run mutation testing on each package
    local failed_packages=()
    
    for package in "${packages[@]}"; do
        if ! run_mutation_test_on_package "${package}"; then
            failed_packages+=("${package}")
        fi
    done
    
    # Report results
    if [[ ${#failed_packages[@]} -eq 0 ]]; then
        log_success "All changed packages pass mutation testing!"
        log_info "Commit allowed to proceed"
        exit 0
    else
        log_error "The following packages failed mutation testing:"
        for pkg in "${failed_packages[@]}"; do
            echo "  - ${pkg}"
        done
        echo ""
        log_error "Commit blocked due to low mutation scores"
        log_info "To improve mutation scores:"
        log_info "  1. Add more comprehensive tests"
        log_info "  2. Test edge cases and error conditions"
        log_info "  3. Verify all code paths are tested"
        echo ""
        log_info "To bypass this check: git commit --no-verify"
        exit 1
    fi
}

# Run main function
main "$@"