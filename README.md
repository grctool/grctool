# GRC Tool - Tugboat Logic Security Program Manager

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/grctool/grctool)](https://goreportcard.com/report/github.com/grctool/grctool)
[![GitHub Release](https://img.shields.io/github/v/release/grctool/grctool)](https://github.com/grctool/grctool/releases)
[![codecov](https://codecov.io/gh/grctool/grctool/branch/main/graph/badge.svg)](https://codecov.io/gh/grctool/grctool)
[![CI Status](https://github.com/grctool/grctool/workflows/Continuous%20Integration/badge.svg)](https://github.com/grctool/grctool/actions?query=workflow%3A%22Continuous+Integration%22)

A CLI application for managing security program compliance through Tugboat Logic integration. This tool automates the collection, analysis, and generation of evidence for security compliance frameworks like SOC 2, ISO 27001, and others.

## Table of Contents

- [Documentation](#-documentation)
- [Features](#features)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Usage](#usage)
- [Development](#development)
- [Contributing](#contributing)
- [Security](#security)
- [Community](#community)
- [License](#license)

## 📚 Documentation

**Complete documentation is available in the [`docs/`](docs/index.md) directory**

### User Documentation
- **Quick Start** - See [Installation](#installation) and [Initial Setup](#initial-setup) sections below
- **[User Guide](docs/)** - Full documentation for configuration and usage
- **[Feature Documentation](docs/features/)** - Evidence collection, integrations, and AI assistance

### Developer Documentation
- **[Architecture Guide](docs/ARCHITECTURE.md)** - System design, patterns, and data flow
- **[Development Guide](docs/04-Development/)** - Development setup, coding standards, and testing
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to the project
- **[Changelog](CHANGELOG.md)** - Release history and changes

### Project Policies
- **[Code of Conduct](CODE_OF_CONDUCT.md)** - Community guidelines
- **[Security Policy](SECURITY.md)** - Vulnerability reporting and security practices
- **[License](LICENSE)** - Apache 2.0 license terms

## Features

- **Automated Browser Authentication**: Safari-based login with automatic cookie extraction (macOS)
- **Data Synchronization**: Download policies, controls, and evidence tasks via REST API
- **AI-Powered Evidence Generation**: Uses Claude AI to intelligently generate compliance evidence
- **Evidence Analysis**: Maps relationships between evidence tasks, controls, and policies
- **29 Evidence Collection Tools**: Comprehensive toolset for Terraform, GitHub, Google Workspace, and more
  - **7 Terraform Tools**: Security analysis, indexing, snippets, HCL parsing, and Atmos support
  - **6 GitHub Tools**: Permissions, workflows, reviews, security features, and deployment controls
  - **Plus**: Google Workspace, evidence assembly, and utility tools
- **Security Control Mapping**: Automated mapping of infrastructure to compliance controls
- **Multiple Output Formats**: Generate evidence in CSV, Markdown, and JSON formats
- **High-Performance Indexing**: Sub-100ms queries with persistent caching
- **Local Data Storage**: JSON-based storage for offline access and analysis

## Quick Start

### Prerequisites

- Go 1.21 or later
- macOS (for automated browser authentication)
- Access to Tugboat Logic account
- Claude AI API key (for evidence generation)

### Installation

#### Option 1: Automated Installer (Recommended)

Install the latest release with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash
```

This installs grctool to `~/.local/bin` and updates your PATH automatically.

**Custom installation options:**

```bash
# Install specific version
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash -s -- --version v0.1.0

# Install to custom directory
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash -s -- --install-dir /usr/local/bin

# Install system-wide (requires sudo)
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash -s -- --system

# Preview without installing
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash -s -- --dry-run

# Uninstall
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash -s -- --uninstall
```

**Verify installation:**
```bash
grctool version
```

#### Option 2: Manual Build from Source

1. **Clone and build:**
   ```bash
   cd isms/grctool
   make build
   # Binary will be created at bin/grctool
   ```

2. **Optionally install to PATH:**
   ```bash
   cp bin/grctool ~/.local/bin/
   # or
   sudo cp bin/grctool /usr/local/bin/
   ```

### Initial Setup

After installation, configure and authenticate:

1. **Initialize configuration:**
   ```bash
   grctool config init
   ```

2. **Authenticate with Tugboat Logic:**
   ```bash
   grctool auth login
   ```

3. **Test the connection:**
   ```bash
   grctool auth status
   ```

## Configuration

The application uses a YAML configuration file (`.grctool.yaml`). See `configs/example.yaml` for a complete example.

### Basic Configuration

```yaml
tugboat:
  base_url: "https://api-my.tugboatlogic.com"
  org_id: "13888"  # Your organization ID
  timeout: "30s"
  rate_limit: 10
  auth_mode: "browser"  # Uses Safari for authentication

evidence:
  claude:
    api_key: "${CLAUDE_API_KEY}"
    model: "claude-3-5-sonnet-20241022"
    max_tokens: 8192

storage:
  data_dir: "../docs"
  cache_dir: "../docs/.cache"
  data_format: "json"

logging:
  level: "info"
  format: "text"
```

### Environment Variables

- `CLAUDE_API_KEY`: Claude AI API key for evidence generation (required)
- `TUGBOAT_BASE_URL`: Override the base URL
- `TUGBOAT_ORG_ID`: Your organization ID (find in URL: /org/{org_id}/policies)
- `LOG_LEVEL`: Override the log level (debug, info, warn, error)

## Usage

### Authentication

```bash
# Login using Safari browser (macOS only)
./bin/grctool auth login

# Check authentication status
./bin/grctool auth status

# Logout and remove credentials
./bin/grctool auth logout
```

### Data Synchronization

```bash
# Full sync from Tugboat Logic API
./bin/grctool sync

# Sync specific data types
./bin/grctool sync --policies
./bin/grctool sync --controls
./bin/grctool sync --evidence
```

**File Storage**: Synchronized data is stored locally using a unified naming pattern:
- Pattern: `{ReferenceID}_{NumericID}_{SanitizedName}.{extension}`
- Examples:
  - `docs/policies/P136_94623_Data_Retention_and_Disposal.json`
  - `docs/controls/AC1_778771_Access_Provisioning_and_Approval.json`
  - `docs/evidence_tasks/ET1_327992_Access_Control_Registration.json`
- To migrate from old file names: Delete `docs/` directory and re-sync

### Evidence Management

```bash
# List all evidence tasks
grctool evidence list

# Filter by status
grctool evidence list --status pending

# Filter by framework
grctool evidence list --framework soc2

# Analyze evidence task relationships
./bin/grctool evidence analyze [task-id]

# Map all evidence-control-policy relationships
./bin/grctool evidence map
```

### Evidence Generation (AI-Powered)

```bash
# Generate evidence for specific task using Claude AI
./bin/grctool evidence generate [task-id]

# Generate with specific tools and format
./bin/grctool evidence generate [task-id] --tools terraform,github --format csv

# Review generated evidence and AI reasoning
./bin/grctool evidence review [task-id] --show-reasoning
```

### Evidence Submission

```bash
# Submit evidence to Tugboat Logic (placeholder - not yet connected to API)
./bin/grctool evidence submit [task-id]
```

### Policy and Control Viewing

```bash
# List all policies
./bin/grctool policy list

# View specific policy
./bin/grctool policy view [policy-id]

# List all controls
./bin/grctool control list

# View specific control
./bin/grctool control view [control-id]
```

### Configuration Management

```bash
# Initialize configuration file
grctool config init

# Validate current configuration (placeholder)
./bin/grctool config validate
```

## Security Control Mappings

The application includes built-in mappings for common security controls:

### SOC 2 Type II Controls

- **CC6.1** - Logical Access Security Measures
  - Maps to: IAM roles, policies, groups, users
  - Files: `**/iam*.tf`, `**/aws-team-roles/**`

- **CC6.2** - Authentication and Authorization
  - Maps to: IAM users, access keys, identity center
  - Files: `**/aws-identity-center/**`, `**/aws-ssosync/**`

- **CC6.6** - Network Security Controls
  - Maps to: VPC, security groups, network ACLs
  - Files: `**/vpc/**`, `**/security-group*`

- **CC6.7** - Data Transmission Controls
  - Maps to: Load balancers, CloudFront, ACM certificates
  - Files: `**/acm/**`, `**/dns-*/**`

- **CC6.8** - Data Loss Prevention
  - Maps to: S3 policies, KMS keys, CloudTrail
  - Files: `**/s3*`, `**/kms*`, `**/cloudtrail*`

## Development

### Build Commands

```bash
# Install dependencies
make deps

# Build for current platform
make build

# Build for all platforms
make build-all

# Run unit tests (excludes integration tests)
make test

# Run integration tests (requires Tugboat Logic credentials)
make test-integration

# Run all tests (unit + integration)
make test-all

# Run with coverage (unit tests only)
make test-coverage

# Format and lint code
make fmt lint

# Run security scans
make security-scan
```

### Testing

The project uses Go's built-in testing framework with separate unit and integration tests:

- **Unit Tests**: Fast, isolated tests that don't require external dependencies
- **Integration Tests**: Tests that require Tugboat Logic API access and credentials

Integration tests are marked with build tags and excluded from normal test runs:
```go
//go:build integration
// +build integration
```

### Development Workflow

1. **Setup development environment:**
   ```bash
   make deps
   make config-init
   ```

2. **Run in development mode:**
   ```bash
   make dev  # Hot reload with air
   ```

3. **Run tests during development:**
   ```bash
   make test  # Unit tests only - fast feedback
   make test-race
   make test-coverage
   ```

4. **Check code quality:**
   ```bash
   make ci  # Runs all checks
   ```

### Project Structure

```
grctool/
├── cmd/                    # CLI commands
├── internal/
│   ├── auth/              # Browser authentication
│   ├── claude/            # Claude AI integration
│   ├── config/            # Configuration management
│   ├── domain/            # Domain models
│   ├── evidence/          # Evidence generation
│   ├── formatters/        # Output formatters
│   ├── storage/           # Local data storage
│   ├── tools/             # Evidence collection tools
│   └── tugboat/           # Tugboat Logic API client
├── configs/               # Configuration templates
└── .github/workflows/     # GitHub Actions
```

## GitHub Actions Integration

**Note**: The current authentication method uses browser-based login which requires manual intervention. For CI/CD pipelines, you have two options:

### Option 1: Automated Testing (Recommended)

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run Tests
        run: |
          cd isms/grctool
          make test  # Unit tests only - no auth required
```

### Option 2: Manual Evidence Sync

For evidence sync workflows, authentication must be handled manually:

```yaml
# .github/workflows/security-sync.yml
name: Security Evidence Sync

on:
  workflow_dispatch:  # Manual trigger only

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build Security Manager
        run: |
          cd isms/grctool
          make build
      
      # Note: Requires pre-authenticated .grctool.yaml
      # with valid cookies from 'auth login' command
      - name: Sync Evidence
        run: |
          cd isms/grctool
          # Copy authenticated config from secrets/artifacts
          # echo "${{ secrets.GRCTOOL_CONFIG }}" > .grctool.yaml
          ./bin/grctool sync
          ./bin/grctool evidence generate --all
```

## Troubleshooting

### Common Issues

1. **API Connection Failed**
   - Run `./bin/grctool auth login` to authenticate
   - Check the `base_url` in your configuration
   - Ensure your Tugboat Logic instance is accessible
   - Verify authentication with `./bin/grctool auth status`

2. **Configuration Validation Failed**
   - Run `grctool config validate` for detailed errors
   - Check file paths exist and are readable
   - Verify YAML syntax in configuration file

3. **Authentication Issues**
   - Ensure Safari is installed and configured
   - Enable "Allow JavaScript from Apple Events" in Safari Developer settings
   - Try manual cookie extraction if automatic fails

4. **Evidence Generation Issues**
   - Verify Claude API key is set correctly
   - Check evidence task exists in synchronized data
   - Review generated prompts in `evidence/prompts/` directory

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
grctool --verbose --log-level debug [command]
```

### Log Files

Logs are written to stdout/stderr by default. For persistent logging:

```bash
grctool [command] 2>&1 | tee grctool.log
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on:

- Setting up your development environment
- Running tests and quality checks
- Code style and guidelines
- Submitting pull requests
- Reporting bugs

Please read our [Code of Conduct](CODE_OF_CONDUCT.md) before contributing.

## Security

GRCTool takes security seriously. Please see our [Security Policy](SECURITY.md) for:

- Reporting security vulnerabilities (responsible disclosure)
- Supported versions
- Security best practices for users and contributors
- Authentication and credential management
- Known security considerations

**To report a security vulnerability**, use [GitHub Security Advisories](https://github.com/grctool/grctool/security/advisories/new) for private disclosure - do NOT open a public issue.

## Community

### Getting Help

- **Documentation**: Browse the [docs/](docs/) directory
- **Issues**: Search [GitHub Issues](https://github.com/grctool/grctool/issues)
- **Discussions**: Join [GitHub Discussions](https://github.com/grctool/grctool/discussions)
- **GitHub Issues**: https://github.com/grctool/grctool/issues

### Reporting Issues

Found a bug? Please [open an issue](https://github.com/grctool/grctool/issues/new) with:
- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, GRCTool version)
- Relevant logs (use `--log-level debug`)

### Feature Requests

Have an idea? We'd love to hear it! Please [open a feature request](https://github.com/grctool/grctool/issues/new) describing:
- The use case and problem it solves
- Expected behavior
- Why this would be valuable to the community

### Release Notes

See [CHANGELOG.md](CHANGELOG.md) for detailed release history and changes.

## Acknowledgments

GRCTool is built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [Claude AI](https://www.anthropic.com/claude) - AI-powered evidence generation
- [VCR](https://github.com/dnaeon/go-vcr) - HTTP testing infrastructure

Special thanks to all contributors who help make GRCTool better!

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

---

**Made with ❤️ for security and compliance teams**