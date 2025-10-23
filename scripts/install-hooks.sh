#!/bin/bash
# Installation script for GRCTool pre-commit hooks
# Installs required tools and sets up git hooks

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
HOOKS_DIR="${PROJECT_ROOT}/.git/hooks"
GO_BIN_DIR="${HOME}/go/bin"

# Ensure go/bin is in PATH
export PATH="${GO_BIN_DIR}:${PATH}"

# Track installation status
ERRORS=0
WARNINGS=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INSTALL]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[INSTALL]${NC} ✓ $1"
}

log_warning() {
    echo -e "${YELLOW}[INSTALL]${NC} ⚠ $1"
}

log_error() {
    echo -e "${RED}[INSTALL]${NC} ✗ $1"
}

show_progress() {
    echo -ne "${BLUE}[INSTALL]${NC} $1... "
}

complete_progress() {
    echo -e "${GREEN}done${NC}"
}

# Check for required commands
check_command() {
    local cmd="$1"
    if command -v "${cmd}" &> /dev/null; then
        return 0
    else
        return 1
    fi
}

# Install Go tool
install_go_tool() {
    local package="$1"
    local binary="$2"
    local description="$3"
    
    show_progress "Installing ${description}"
    
    if check_command "${binary}"; then
        complete_progress
        log_info "${description} already installed"
        return 0
    fi
    
    if go install "${package}" >> /tmp/install-hooks.log 2>&1; then
        complete_progress
        log_success "${description} installed successfully"
        return 0
    else
        echo -e "${RED}failed${NC}"
        log_error "Failed to install ${description}"
        log_info "Try manually: go install ${package}"
        ((ERRORS++))
        return 1
    fi
}

# Main installation
main() {
    echo ""
    echo "=================================================================================="
    echo -e "${BOLD}GRCTool Pre-commit Hooks Installation${NC}"
    echo "=================================================================================="
    echo ""
    
    # Check if we're in a git repository
    if [[ ! -d "${PROJECT_ROOT}/.git" ]]; then
        log_error "Not in a git repository!"
        log_info "Please run this script from the project root"
        exit 1
    fi
    
    # Check for Go installation
    if ! check_command "go"; then
        log_error "Go is not installed or not in PATH"
        log_info "Please install Go from https://golang.org/dl/"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Found Go version: ${GO_VERSION}"
    echo ""
    
    # Create hooks directory if it doesn't exist
    if [[ ! -d "${HOOKS_DIR}" ]]; then
        mkdir -p "${HOOKS_DIR}"
        log_success "Created hooks directory"
    fi
    
    # Install required Go tools
    echo -e "${BOLD}Installing Go tools...${NC}"
    echo ""
    
    # golangci-lint
    install_go_tool \
        "github.com/golangci/golangci-lint/cmd/golangci-lint@latest" \
        "golangci-lint" \
        "golangci-lint (comprehensive linter)"
    
    # goimports
    install_go_tool \
        "golang.org/x/tools/cmd/goimports@latest" \
        "goimports" \
        "goimports (import formatting)"
    
    # gosec
    install_go_tool \
        "github.com/securego/gosec/v2/cmd/gosec@latest" \
        "gosec" \
        "gosec (security scanner)"
    
    # staticcheck
    install_go_tool \
        "honnef.co/go/tools/cmd/staticcheck@latest" \
        "staticcheck" \
        "staticcheck (static analysis)"
    
    # gocyclo
    install_go_tool \
        "github.com/fzipp/gocyclo/cmd/gocyclo@latest" \
        "gocyclo" \
        "gocyclo (cyclomatic complexity)"
    
    # ineffassign
    install_go_tool \
        "github.com/gordonklaus/ineffassign@latest" \
        "ineffassign" \
        "ineffassign (ineffectual assignments)"
    
    echo ""
    
    # Check for hook files
    echo -e "${BOLD}Checking hook files...${NC}"
    echo ""
    
    # Pre-commit hook
    if [[ -f "${HOOKS_DIR}/pre-commit" ]]; then
        if [[ ! -f "${HOOKS_DIR}/pre-commit.backup" ]]; then
            cp "${HOOKS_DIR}/pre-commit" "${HOOKS_DIR}/pre-commit.backup"
            log_info "Backed up existing pre-commit hook"
        fi
    fi
    
    log_success "Pre-commit hook is installed at ${HOOKS_DIR}/pre-commit"
    
    # Check helper scripts
    if [[ ! -f "${PROJECT_ROOT}/scripts/detect-secrets.sh" ]]; then
        log_warning "Secret detection script not found"
        ((WARNINGS++))
    else
        log_success "Secret detection script found"
    fi
    
    # Check configuration files
    if [[ ! -f "${PROJECT_ROOT}/.golangci.yml" ]]; then
        log_warning ".golangci.yml configuration not found"
        ((WARNINGS++))
    else
        log_success "golangci-lint configuration found"
    fi
    
    if [[ ! -f "${PROJECT_ROOT}/.pre-commit-config.yaml" ]]; then
        log_warning ".pre-commit-config.yaml not found"
        ((WARNINGS++))
    else
        log_success "Pre-commit configuration found"
    fi
    
    echo ""
    
    # Test the hook
    echo -e "${BOLD}Testing pre-commit hook...${NC}"
    echo ""
    
    show_progress "Running basic validation"
    if bash -n "${HOOKS_DIR}/pre-commit" 2>/dev/null; then
        complete_progress
        log_success "Pre-commit hook syntax is valid"
    else
        echo -e "${RED}failed${NC}"
        log_error "Pre-commit hook has syntax errors"
        ((ERRORS++))
    fi
    
    # Summary
    echo ""
    echo "=================================================================================="
    
    if [[ ${ERRORS} -eq 0 ]]; then
        echo -e "${GREEN}✓ Installation completed successfully!${NC}"
        
        if [[ ${WARNINGS} -gt 0 ]]; then
            echo -e "${YELLOW}  (with ${WARNINGS} warning(s))${NC}"
        fi
        
        echo ""
        echo -e "${BOLD}The pre-commit hook is now active and will run on every commit.${NC}"
        echo ""
        echo "To test the hook manually:"
        echo "  ${HOOKS_DIR}/pre-commit"
        echo ""
        echo "To bypass the hook (not recommended):"
        echo "  git commit --no-verify"
        echo ""
        echo "To skip specific checks, use environment variables:"
        echo "  SKIP_LINT=true git commit"
        echo "  SKIP_TEST=true git commit"
        echo "  SKIP_ALL=true git commit"
        echo ""
        echo "Configuration files:"
        echo "  - .golangci.yml         : Linting rules"
        echo "  - .pre-commit-config.yaml : Hook configuration"
        echo ""
    else
        echo -e "${RED}✗ Installation completed with ${ERRORS} error(s)${NC}"
        echo ""
        echo "Please fix the errors above and try again."
        echo "You may need to install some tools manually."
        echo ""
        exit 1
    fi
}

# Create a log file
LOG_FILE="/tmp/install-hooks.log"
echo "Installation started at $(date)" > "${LOG_FILE}"

# Run main installation
main 2>&1 | tee -a "${LOG_FILE}"

# Clean up
rm -f "${LOG_FILE}"