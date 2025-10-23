# Contributing to GRCTool

Thank you for your interest in contributing to GRCTool! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Environment Setup](#development-environment-setup)
- [Building and Running](#building-and-running)
- [Testing](#testing)
- [Code Style and Guidelines](#code-style-and-guidelines)
- [Submitting Changes](#submitting-changes)
- [Reporting Bugs](#reporting-bugs)
- [Contact](#contact)

## Code of Conduct

This project adheres to the Contributor Covenant Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior by opening an issue at https://github.com/grctool/grctool/issues.

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for details.

## Getting Started

GRCTool is a CLI application for automating compliance evidence collection through Tugboat Logic integration. Before contributing, familiarize yourself with:

- The [README.md](README.md) for project overview
- The [documentation](docs/) for detailed features and usage
- The [ARCHITECTURE.md](docs/ARCHITECTURE.md) for system design

## Development Environment Setup

### Prerequisites

- **Go 1.21 or later** - [Download](https://golang.org/dl/)
- **Git** - Version control
- **Make** - Build automation
- **golangci-lint** - Code quality (optional, will be installed automatically)
- **macOS** - Required for browser authentication features (Safari-based)

### Initial Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/grctool/grctool.git
   cd grctool
   ```

2. **Install dependencies:**
   ```bash
   make deps
   ```

3. **Build the project:**
   ```bash
   make build
   ```
   This creates the binary at `build/grctool`.

4. **Set up configuration:**
   ```bash
   cp .grctool.example.yaml .grctool.yaml
   # Edit .grctool.yaml with your settings
   ```

5. **Set environment variables:**
   ```bash
   export CLAUDE_API_KEY="your-claude-api-key"
   export TUGBOAT_ORG_ID="your-org-id"
   ```

### Development Tools

Install recommended development tools:

```bash
# golangci-lint (linter)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# air (hot reload for development)
go install github.com/cosmtrek/air@latest
```

## Building and Running

### Build Commands

```bash
# Build for current platform
make build

# Build for all platforms (Linux, macOS, Windows)
make build-all

# Install to ~/.local/bin
make install

# Run with hot reload during development
make dev
```

### Running the Application

```bash
# Run from source
go run . --help

# Run built binary
./build/grctool --help

# Run with verbose logging
./build/grctool --verbose --log-level debug sync
```

## Testing

GRCTool uses a comprehensive three-tier testing strategy:

### Test Levels

1. **Unit Tests** - Fast, isolated tests with no external dependencies
2. **Integration Tests** - Tests using VCR recordings (no live APIs)
3. **Functional Tests** - End-to-end CLI tests with built binary

### Running Tests

```bash
# Unit tests only (fast, ~2-3 seconds)
make test-unit

# Integration tests with VCR playback
make test-integration

# Functional CLI tests
make test-functional

# All tests (unit + integration + functional)
make test-all

# Run with race detection
make test-race

# Generate coverage report
make test-coverage
```

### Test Organization

Tests are organized using Go build tags:

- **Unit tests**: No build tags, in `*_test.go` files alongside code
- **Integration tests**: `//go:build integration` tag, in `test/integration/`
- **Functional tests**: `//go:build functional` tag, in `test/functional/`

### Writing Tests

#### Unit Test Example

```go
// internal/storage/json_store_test.go
func TestJSONStore_Save(t *testing.T) {
    store := NewJSONStore("/tmp/test")
    err := store.Save("test.json", testData)
    assert.NoError(t, err)
}
```

#### Integration Test Example

```go
//go:build integration
// +build integration

// test/integration/tugboat_client_test.go
func TestTugboatClient_GetPolicies(t *testing.T) {
    // Uses VCR cassette for recorded responses
    client := setupTestClient(t)
    policies, err := client.GetPolicies(ctx)
    assert.NoError(t, err)
}
```

#### Functional Test Example

```go
//go:build functional
// +build functional

// test/functional/cli_test.go
func TestCLI_Sync(t *testing.T) {
    // Tests built binary end-to-end
    output := runCLI(t, "sync", "--policies")
    assert.Contains(t, output, "Synced")
}
```

### VCR Testing

Integration tests use VCR (Video Cassette Recorder) to record and replay HTTP interactions:

- **Recording mode**: `VCR_MODE=record go test -tags=integration ...`
- **Playback mode**: `VCR_MODE=playback go test -tags=integration ...` (default)
- Cassettes are stored in `test/integration/fixtures/`

See [RECORD_CASSETTES.md](docs/archive/RECORD_CASSETTES.md) for details.

### Test Coverage

Aim for:
- **80%+ coverage** for core business logic
- **60%+ coverage** for utilities and helpers
- **100% coverage** for critical security functions

Check coverage:
```bash
make test-coverage
# Opens HTML coverage report in browser
```

## Code Style and Guidelines

### Code Style

GRCTool follows standard Go conventions with strict linting rules defined in `.golangci.yml`.

#### Run Linters

```bash
# Format code
make fmt

# Run go vet
make vet

# Run all linters (golangci-lint)
make lint

# Run all quality checks
make ci
```

### Coding Standards

#### Formatting

- Use `gofmt` / `goimports` for formatting (automatic with `make fmt`)
- Line length: 120 characters maximum
- Indentation: Tabs (4-space width)

#### Naming Conventions

- **Packages**: Short, lowercase, single word (e.g., `storage`, `auth`)
- **Files**: Snake case (e.g., `json_store.go`, `tugboat_client.go`)
- **Types**: PascalCase (e.g., `JSONStore`, `TugboatClient`)
- **Functions**: PascalCase for exported, camelCase for private
- **Constants**: PascalCase or SCREAMING_SNAKE_CASE for package-level

#### Documentation

- All exported types, functions, and constants must have comments
- Comments should be complete sentences starting with the name
- Package-level documentation in `doc.go` or package comment

Example:
```go
// Policy represents a governance document in the security program.
// It contains the policy text, metadata, and associated controls.
type Policy struct {
    ID          string
    Name        string
    Description string
}

// GetPolicy retrieves a policy by its reference ID.
// It returns an error if the policy is not found.
func GetPolicy(id string) (*Policy, error) {
    // Implementation
}
```

#### Error Handling

- Always handle errors explicitly (no `_ = err` without justification)
- Wrap errors with context: `fmt.Errorf("failed to load config: %w", err)`
- Use custom error types for domain errors
- Log errors at appropriate levels

#### Code Organization

- Keep functions small and focused (max 50 lines)
- Cyclomatic complexity max: 15
- Extract complex logic into helper functions
- Use dependency injection for testability

### License Headers

All source files must include the Apache 2.0 license header:

```go
// Copyright 2024 GRCTool Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mypackage
```

### Commit Message Guidelines

Follow the Conventional Commits specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `test`: Test additions or changes
- `chore`: Maintenance tasks
- `perf`: Performance improvements

**Examples:**
```
feat(auth): add browser-based Safari authentication

Implements automatic cookie extraction from Safari for
Tugboat Logic authentication without manual token entry.

Closes #42
```

```
fix(storage): handle missing directory in JSON store

Create parent directories automatically when saving files
to prevent "directory not found" errors.
```

## Submitting Changes

### Pull Request Process

1. **Create a feature branch:**
   ```bash
   git checkout -b feat/my-new-feature
   ```

2. **Make your changes:**
   - Write code following style guidelines
   - Add/update tests
   - Update documentation if needed

3. **Ensure all checks pass:**
   ```bash
   make ci  # Runs fmt, vet, lint, and all tests
   ```

4. **Commit your changes:**
   ```bash
   git add .
   git commit -m "feat(scope): description"
   ```

5. **Push to your fork:**
   ```bash
   git push origin feat/my-new-feature
   ```

6. **Create a Pull Request:**
   - Use a clear, descriptive title
   - Reference any related issues
   - Describe what changed and why
   - Include screenshots for UI changes

### PR Requirements

Before your PR can be merged:

- [ ] All tests pass (`make test-all`)
- [ ] Code is formatted (`make fmt`)
- [ ] Linters pass (`make lint`)
- [ ] Documentation is updated
- [ ] Commit messages follow conventions
- [ ] Branch is up to date with `main`
- [ ] At least one approval from maintainers

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
Describe testing performed

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Linters pass
- [ ] Commit messages follow conventions
```

## Reporting Bugs

### Before Submitting

1. Check [existing issues](https://github.com/grctool/grctool/issues)
2. Try the latest version
3. Enable debug logging: `--log-level debug`

### Bug Report Template

```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce:
1. Run command '...'
2. See error

**Expected behavior**
What you expected to happen.

**Actual behavior**
What actually happened.

**Environment:**
- OS: [e.g., macOS 14.0]
- GRCTool version: [e.g., v0.1.0]
- Go version: [e.g., 1.21.0]

**Logs**
```
Paste relevant logs here
```

**Additional context**
Any other relevant information.
```

### Security Vulnerabilities

**DO NOT** open public issues for security vulnerabilities. Instead, use [GitHub Security Advisories](https://github.com/grctool/grctool/security/advisories/new) for private disclosure.

See [SECURITY.md](SECURITY.md) for details.

## Feature Requests

We welcome feature requests! Please:

1. Check if the feature already exists or is planned
2. Open an issue with the "enhancement" label
3. Clearly describe the use case and expected behavior
4. Explain why this feature would be valuable

## Development Workflow

### Typical Development Cycle

```bash
# 1. Update your main branch
git checkout main
git pull upstream main

# 2. Create feature branch
git checkout -b feat/my-feature

# 3. Develop with hot reload
make dev

# 4. Run tests frequently
make test-unit

# 5. Before committing
make ci

# 6. Commit and push
git add .
git commit -m "feat: my feature"
git push origin feat/my-feature

# 7. Create PR on GitHub
```

### Adding a New Tool

To add a new evidence collection tool:

1. Create tool implementation in `internal/tools/`
2. Register in `internal/registry/tool_registry.go`
3. Add CLI command in `cmd/tool_*.go`
4. Write unit tests
5. Add integration tests with VCR
6. Update documentation

See existing tools (e.g., `internal/tools/terraform/`) for examples.

## Project Structure

```
grctool/
├── cmd/                     # CLI command implementations
│   ├── auth.go             # Authentication commands
│   ├── sync.go             # Sync commands
│   └── tool_*.go           # Tool commands
├── internal/               # Private application code
│   ├── auth/              # Authentication logic
│   ├── config/            # Configuration management
│   ├── domain/            # Domain models
│   ├── formatters/        # Output formatters
│   ├── registry/          # Tool registration
│   ├── storage/           # Data persistence
│   ├── tools/             # Evidence collection tools
│   ├── tugboat/           # Tugboat API client
│   └── vcr/               # VCR test infrastructure
├── test/                  # Test suites
│   ├── integration/       # Integration tests
│   └── functional/        # Functional tests
├── docs/                  # Documentation
├── configs/              # Configuration examples
└── scripts/              # Build and utility scripts
```

## CI/CD Pipeline

### GitHub Actions Workflows

The project uses GitHub Actions for continuous integration with the following workflows:

#### 1. Continuous Integration (`.github/workflows/ci.yml`)

**Triggered on:** Push to main, Pull Requests

**Jobs:**
- **ci-checks**: Code formatting, linting, and basic checks using lefthook
- **test-matrix**: Unit and integration tests
- **coverage**: Test coverage analysis with 30% threshold
- **license-check**: Validates Apache 2.0 license headers in all Go files
- **security-scan**: gosec security scanning and secret detection
- **build-matrix**: Build verification for Linux amd64
- **quality-gates**: Final gate checking all jobs

#### 2. Advanced Testing (`.github/workflows/testing.yml`)

**Triggered on:** Weekly schedule (Monday 2 AM UTC), Manual dispatch

**Jobs:**
- Deep coverage analysis
- Performance benchmarks
- Mutation testing
- Multi-version Go testing

#### 3. Release (`.github/workflows/release.yml`)

**Triggered on:** Version tags (v*)

### Required GitHub Secrets

For full CI/CD functionality, configure these secrets in your repository:

| Secret Name | Required | Purpose | How to Obtain |
|-------------|----------|---------|---------------|
| `CODECOV_TOKEN` | Optional | Upload coverage to Codecov | [codecov.io](https://codecov.io) |
| `GITHUB_TOKEN` | Auto | GitHub API access, SARIF upload | Automatically provided |

**Note:** Most CI jobs work without secrets. Coverage upload to Codecov is optional and will not fail the build if the token is missing.

### Setting Up Secrets

1. Go to repository Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Add each secret with its value

### Local CI Testing

Test CI checks locally before pushing:

```bash
# Run full CI check suite
make ci

# Run specific checks
LEFTHOOK_CONFIG=lefthook-ci.yml lefthook run ci

# Test coverage threshold
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

### Coverage Thresholds

Current coverage threshold: **30%** (will increase as test coverage improves)

Target coverage by component:
- Core business logic: 80%+
- Utilities and helpers: 60%+
- Critical security functions: 100%

### Quality Gate Requirements

All PRs must pass these gates:
- ✅ Code formatting (gofmt, goimports)
- ✅ Linting (golangci-lint)
- ✅ All tests pass
- ✅ Coverage threshold met (30%)
- ✅ License headers present
- ✅ No secrets detected
- ℹ️  Security scan (informational only)

### Troubleshooting CI Failures

**Coverage below threshold:**
- Add tests for new code
- Current coverage: run `make test-coverage` locally
- See coverage report in CI artifacts

**License header check fails:**
- Run `./scripts/check-license-headers.sh` to identify files
- Add Apache 2.0 header to all Go files

**Secret detection fails:**
- Review flagged files with `./scripts/detect-secrets.sh`
- Remove hardcoded credentials
- Use environment variables or config files

**Security scan findings:**
- Review gosec results in CI artifacts
- Address high-severity issues
- Low/medium findings are informational only

## Getting Help

- **Documentation**: Check [docs/](docs/)
- **Issues**: Search [GitHub issues](https://github.com/grctool/grctool/issues)
- **Discussions**: Use [GitHub Discussions](https://github.com/grctool/grctool/discussions)
- **GitHub Issues**: https://github.com/grctool/grctool/issues

## Recognition

Contributors will be recognized in:
- GitHub contributors page
- Release notes
- Project documentation (if significant contribution)

Thank you for contributing to GRCTool!
