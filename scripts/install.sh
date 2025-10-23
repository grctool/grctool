#!/usr/bin/env bash
# GRCTool installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/telepathdata/grctool/main/scripts/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
REPO="grctool/grctool"
BINARY_NAME="grctool"
DEFAULT_INSTALL_DIR="$HOME/.local/bin"
SYSTEM_INSTALL_DIR="/usr/local/bin"

# Runtime variables
VERSION=""
INSTALL_DIR=""
DRY_RUN=false
VERBOSE=false
UNINSTALL=false
SYSTEM_INSTALL=false

# Print functions
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1" >&2
}

print_verbose() {
    if [ "$VERBOSE" = true ]; then
        echo -e "${CYAN}→${NC} $1"
    fi
}

# Usage information
usage() {
    cat << EOF
GRCTool Installer

Usage: $0 [OPTIONS]

Options:
    --version VERSION       Install specific version (e.g., v0.1.0)
    --install-dir DIR       Custom installation directory
    --system                Install to /usr/local/bin (requires sudo)
    --uninstall             Remove grctool from system
    --dry-run               Preview installation without making changes
    --verbose               Show detailed output
    -h, --help              Show this help message

Examples:
    # Install latest version to ~/.local/bin
    $0

    # Install specific version
    $0 --version v0.1.0

    # Install system-wide
    $0 --system

    # Uninstall
    $0 --uninstall

EOF
    exit 0
}

# Parse command-line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --version)
                VERSION="$2"
                shift 2
                ;;
            --install-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --system)
                SYSTEM_INSTALL=true
                shift
                ;;
            --uninstall)
                UNINSTALL=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                print_error "Unknown option: $1"
                usage
                ;;
        esac
    done
}

# Detect OS and architecture
detect_platform() {
    local os=""
    local arch=""

    # Detect OS
    case "$(uname -s)" in
        Linux*)
            os="linux"
            ;;
        Darwin*)
            os="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            print_info "Supported platforms: Linux, macOS"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            print_info "Supported architectures: amd64, arm64"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Get latest version from GitHub
get_latest_version() {
    print_verbose "Fetching latest version from GitHub..."

    local latest_url="https://api.github.com/repos/${REPO}/releases/latest"
    local version

    if command -v curl >/dev/null 2>&1; then
        version=$(curl -fsSL "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        print_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi

    if [ -z "$version" ]; then
        print_error "Failed to fetch latest version"
        exit 1
    fi

    echo "$version"
}

# Download file
download_file() {
    local url="$1"
    local output="$2"

    print_verbose "Downloading: $url"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$output" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$output" "$url"
    else
        print_error "Neither curl nor wget found"
        exit 1
    fi
}

# Verify checksum
verify_checksum() {
    local archive="$1"
    local checksums_file="$2"
    local archive_name="$(basename "$archive")"

    print_verbose "Verifying checksum for $archive_name..."

    # Extract expected checksum
    local expected_checksum=$(grep "$archive_name" "$checksums_file" | awk '{print $1}')

    if [ -z "$expected_checksum" ]; then
        print_error "Checksum not found for $archive_name"
        return 1
    fi

    # Calculate actual checksum
    local actual_checksum=""
    if command -v sha256sum >/dev/null 2>&1; then
        actual_checksum=$(sha256sum "$archive" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual_checksum=$(shasum -a 256 "$archive" | awk '{print $1}')
    else
        print_warning "Neither sha256sum nor shasum found, skipping checksum verification"
        return 0
    fi

    if [ "$expected_checksum" != "$actual_checksum" ]; then
        print_error "Checksum verification failed!"
        print_error "Expected: $expected_checksum"
        print_error "Got:      $actual_checksum"
        return 1
    fi

    print_verbose "Checksum verified successfully"
    return 0
}

# Install binary
install_binary() {
    local binary_path="$1"
    local install_dir="$2"

    if [ "$DRY_RUN" = true ]; then
        print_info "[DRY RUN] Would install $binary_path to $install_dir/$BINARY_NAME"
        return 0
    fi

    # Create install directory if it doesn't exist
    if [ ! -d "$install_dir" ]; then
        print_verbose "Creating directory: $install_dir"
        mkdir -p "$install_dir"
    fi

    # Copy binary
    if [ "$install_dir" = "$SYSTEM_INSTALL_DIR" ]; then
        print_verbose "Installing to system directory (requires sudo)..."
        if ! sudo cp "$binary_path" "$install_dir/$BINARY_NAME"; then
            print_error "Failed to install binary (permission denied)"
            print_info "Try running without --system flag to install to ~/.local/bin"
            exit 1
        fi
        sudo chmod +x "$install_dir/$BINARY_NAME"
    else
        cp "$binary_path" "$install_dir/$BINARY_NAME"
        chmod +x "$install_dir/$BINARY_NAME"
    fi

    print_success "Installed to $install_dir/$BINARY_NAME"
}

# Update PATH in shell configuration
update_path() {
    local install_dir="$1"

    # Skip if already in PATH
    if echo "$PATH" | grep -q "$install_dir"; then
        print_verbose "Install directory already in PATH"
        return 0
    fi

    if [ "$DRY_RUN" = true ]; then
        print_info "[DRY RUN] Would add $install_dir to PATH in shell configuration"
        return 0
    fi

    # Determine which shell config file to update
    local shell_config=""
    local export_line="export PATH=\"$install_dir:\$PATH\""

    if [ -n "$BASH_VERSION" ] && [ -f "$HOME/.bashrc" ]; then
        shell_config="$HOME/.bashrc"
    elif [ -n "$ZSH_VERSION" ] && [ -f "$HOME/.zshrc" ]; then
        shell_config="$HOME/.zshrc"
    elif [ -f "$HOME/.profile" ]; then
        shell_config="$HOME/.profile"
    fi

    if [ -n "$shell_config" ]; then
        # Check if line already exists
        if grep -q "$install_dir" "$shell_config"; then
            print_verbose "PATH already configured in $shell_config"
            return 0
        fi

        print_verbose "Adding $install_dir to PATH in $shell_config"
        echo "" >> "$shell_config"
        echo "# Added by grctool installer" >> "$shell_config"
        echo "$export_line" >> "$shell_config"

        print_warning "PATH updated in $shell_config"
        print_info "Run 'source $shell_config' or start a new terminal session"
    else
        print_warning "Could not determine shell configuration file"
        print_info "Please add $install_dir to your PATH manually:"
        print_info "  $export_line"
    fi
}

# Uninstall grctool
uninstall() {
    local locations=(
        "$DEFAULT_INSTALL_DIR/$BINARY_NAME"
        "$SYSTEM_INSTALL_DIR/$BINARY_NAME"
    )

    if [ -n "$INSTALL_DIR" ]; then
        locations=("$INSTALL_DIR/$BINARY_NAME")
    fi

    local found=false

    for location in "${locations[@]}"; do
        if [ -f "$location" ]; then
            found=true
            if [ "$DRY_RUN" = true ]; then
                print_info "[DRY RUN] Would remove $location"
            else
                if [[ "$location" == "$SYSTEM_INSTALL_DIR"* ]]; then
                    sudo rm -f "$location"
                else
                    rm -f "$location"
                fi
                print_success "Removed $location"
            fi
        fi
    done

    if [ "$found" = false ]; then
        print_warning "grctool not found in expected locations"
    else
        print_success "grctool uninstalled"
        print_info "Note: Shell configuration files were not modified"
    fi

    exit 0
}

# Main installation flow
main() {
    echo -e "${CYAN}"
    cat << "EOF"
   ____ ____   ____ _____           _
  / ___|  _ \ / ___|_   _|__   ___ | |
 | |  _| |_) | |     | |/ _ \ / _ \| |
 | |_| |  _ <| |___  | | (_) | (_) | |
  \____|_| \_\\____| |_|\___/ \___/|_|

EOF
    echo -e "${NC}"
    print_info "GRCTool Installer"
    echo ""

    # Parse arguments
    parse_args "$@"

    # Handle uninstall
    if [ "$UNINSTALL" = true ]; then
        uninstall
    fi

    # Detect platform
    local platform=$(detect_platform)
    print_info "Platform: $platform"

    # Get version
    if [ -z "$VERSION" ]; then
        VERSION=$(get_latest_version)
    fi
    print_info "Version: $VERSION"

    # Set install directory
    if [ -z "$INSTALL_DIR" ]; then
        if [ "$SYSTEM_INSTALL" = true ]; then
            INSTALL_DIR="$SYSTEM_INSTALL_DIR"
        else
            INSTALL_DIR="$DEFAULT_INSTALL_DIR"
        fi
    fi
    print_info "Install directory: $INSTALL_DIR"

    if [ "$DRY_RUN" = true ]; then
        print_warning "DRY RUN MODE - No changes will be made"
    fi

    echo ""

    # Construct download URLs
    local archive_name="${BINARY_NAME}_${VERSION#v}_${platform}.tar.gz"
    local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${archive_name}"
    local checksums_url="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

    # Create temporary directory
    local tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    # Download archive
    print_info "Downloading $archive_name..."
    download_file "$download_url" "$tmp_dir/$archive_name"
    print_success "Downloaded"

    # Download checksums
    print_info "Downloading checksums..."
    download_file "$checksums_url" "$tmp_dir/checksums.txt"
    print_success "Downloaded"

    # Verify checksum
    print_info "Verifying checksum..."
    if ! verify_checksum "$tmp_dir/$archive_name" "$tmp_dir/checksums.txt"; then
        print_error "Installation aborted due to checksum failure"
        exit 1
    fi
    print_success "Checksum verified"

    # Extract archive
    print_info "Extracting archive..."
    tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"
    print_success "Extracted"

    # Install binary
    print_info "Installing binary..."
    install_binary "$tmp_dir/$BINARY_NAME" "$INSTALL_DIR"

    # Update PATH if needed (only for user installs)
    if [ "$INSTALL_DIR" != "$SYSTEM_INSTALL_DIR" ]; then
        update_path "$INSTALL_DIR"
    fi

    echo ""
    print_success "Installation complete!"
    echo ""

    # Show next steps
    if [ "$DRY_RUN" = false ]; then
        print_info "Next steps:"
        if [ "$INSTALL_DIR" != "$SYSTEM_INSTALL_DIR" ] && ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
            echo "  1. Add to current session: export PATH=\"$INSTALL_DIR:\$PATH\""
            echo "  2. Verify installation: grctool version"
        else
            echo "  1. Verify installation: grctool version"
        fi
        echo ""
        print_info "Documentation: https://github.com/${REPO}"
    fi
}

# Run main function
main "$@"
