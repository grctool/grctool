#!/bin/bash
# install-go-essentials.sh

# Detect OS
OS="unknown"
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
fi

echo -e "${BLUE}Detected OS: $OS${NC}"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install tool
install_tool() {
    local tool=$1
    local install_cmd=$2

    if command_exists "$tool"; then
        echo -e "${GREEN}✓ $tool already installed${NC}"
    else
        echo -e "${YELLOW}Installing $tool...${NC}"
        eval "$install_cmd"
        if command_exists "$tool"; then
            echo -e "${GREEN}✓ $tool installed successfully${NC}"
        else
            echo -e "${RED}✗ Failed to install $tool${NC}"
        fi
    fi
}

# Install system dependencies
echo -e "\n${BLUE}Installing system dependencies...${NC}"

if [[ "$OS" == "macos" ]]; then
    # Install Homebrew if not present
    if ! command_exists brew; then
        echo "Installing Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    fi

    # Install system tools
    install_tool "fzf" "brew install fzf"
    install_tool "bat" "brew install bat"
    install_tool "rg" "brew install ripgrep"
    install_tool "fd" "brew install fd"
    install_tool "jq" "brew install jq"
    install_tool "tree" "brew install tree"
    install_tool "ctags" "brew install universal-ctags"
    install_tool "dot" "brew install graphviz"
elif [[ "$OS" == "linux" ]]; then
    # Update package list
    sudo apt-get update -qq

    # Install system tools
    install_tool "fzf" "sudo apt-get install -y fzf"
    install_tool "bat" "sudo apt-get install -y bat && sudo ln -sf /usr/bin/batcat /usr/local/bin/bat"
    install_tool "rg" "sudo apt-get install -y ripgrep"
    install_tool "fd" "sudo apt-get install -y fd-find && sudo ln -sf /usr/bin/fdfind /usr/local/bin/fd"
    install_tool "jq" "sudo apt-get install -y jq"
    install_tool "tree" "sudo apt-get install -y tree"
    install_tool "ctags" "sudo apt-get install -y universal-ctags"
    install_tool "dot" "sudo apt-get install -y graphviz"
fi

# #TODO move this up to the system specific section
brew install tree golang golangci-lint tokei sift ripgrep gopls \
    universal-ctags \
    fzf \
    rust \
    semgrep \
    bat

# cargo stuff
cargo install ast-grep


# Basics
go install github.com/go-task/task/v3/cmd/task@latest
go install github.com/cortesi/modd/cmd/modd@latest

# Performance
go install github.com/google/pprof@latest
go install golang.org/x/perf/cmd/benchstat@latest

# Code quality
go install github.com/kisielk/errcheck@latest
go install golang.org/x/tools/cmd/stringer@latest

# Testing
go install github.com/rakyll/hey@latest
go install github.com/onsi/ginkgo/v2/ginkgo@latest

# Debugging
go install github.com/google/gops@latest
go install github.com/go-delve/delve/cmd/dlv@latest

# Navigation
go install golang.org/x/tools/cmd/guru@latest
go install github.com/yuroyoro/goast-viewer/cmd/goast-viewer@latest
go install github.com/ofabry/go-callvis@latest
go install github.com/KyleBanks/depth/cmd/depth@latest
go install github.com/acroca/go-symbols@latest
go install golang.org/x/tools/cmd/godex@latest
go install github.com/loov/goda@latest

# Docs
go install github.com/robertkrimen/godocdown/godocdown@latest
go install github.com/ramya-rao-a/go-outline@latest

# Core Go tools
go install golang.org/x/tools/gopls@latest
go install golang.org/x/tools/cmd/guru@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install golang.org/x/tools/cmd/godoc@latest

# Navigation and analysis tools
go install github.com/ofabry/go-callvis@latest
go install github.com/KyleBanks/depth/cmd/depth@latest
go install github.com/acroca/go-symbols@latest
go install github.com/ramya-rao-a/go-outline@latest
go install github.com/loov/goda@latest

# Code quality tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/kisielk/errcheck@latest
go install golang.org/x/tools/cmd/stringer@latest
go install github.com/go-critic/go-critic/cmd/gocritic@latest

# Testing tools
go install github.com/rakyll/hey@latest
go install github.com/tsenart/vegeta@latest
go install golang.org/x/perf/cmd/benchstat@latest
