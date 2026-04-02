---
title: "Build Procedures"
phase: "04-build"
category: "build"
tags: ["build", "compilation", "release", "goreleaser", "ci-cd", "lefthook", "quality-gates"]
related: ["development-practices", "test-procedures", "secure-coding"]
created: 2026-03-17
updated: 2026-03-17
helix_mapping: "04-Build / Build Procedures"
---

# Build Procedures

## Overview

This document describes how to build, test, and release GRCTool. It covers local development builds, cross-platform compilation, GoReleaser-based releases, CI/CD pipeline stages, and the quality gates that must pass before code merges to main.

## Development Environment Setup

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.24+ (toolchain 1.24.12) | Compiler |
| golangci-lint | v1.60.3+ | Linting |
| gosec | Latest | Security scanning |
| govulncheck | Latest | Dependency vulnerability check |
| goimports | Latest | Import organization |
| lefthook | Latest | Git hooks manager |
| GoReleaser | v2+ | Release automation |
| gh CLI | Latest | GitHub authentication |

### Initial Setup

```bash
# 1. Clone and enter the repository
git clone https://github.com/grctool/grctool.git
cd grctool

# 2. Download Go dependencies
make deps
# Runs: go mod download && go mod tidy

# 3. Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/evilmartians/lefthook@latest

# 4. Install git hooks
lefthook install

# 5. Verify the build
make build
```

### Build Variables

The Makefile injects version metadata at compile time via `ldflags`:

| Variable | Source | Injected As |
|----------|--------|-------------|
| `VERSION` | `VERSION` env var or `dev` | `main.Version` |
| `BUILD_TIME` | `date -u` | `main.BuildTime` |
| `GIT_COMMIT` | `git rev-parse --short HEAD` | `main.GitCommit` |

## Build Procedures

### Local Development Build

```bash
make build
```

**What it does**:
1. Runs `go mod download` and `go mod tidy`
2. Creates the `build/` directory
3. Compiles `go build -ldflags "..." -o build/grctool .`

**Output**: `build/grctool` (current platform only)

### Test Build

```bash
make build-test
```

Builds the binary to `bin/grctool` for use by functional tests. This is a prerequisite for `make test-functional`.

### Cross-Platform Builds

```bash
make build-all
```

**What it does**: Compiles binaries for all supported platforms:

| OS | Architecture | Output |
|----|-------------|--------|
| Linux | amd64 | `dist/grctool-linux-amd64` |
| Linux | arm64 | `dist/grctool-linux-arm64` |
| macOS | amd64 | `dist/grctool-darwin-amd64` |
| macOS | arm64 | `dist/grctool-darwin-arm64` |
| Windows | amd64 | `dist/grctool-windows-amd64.exe` |

### Install to Local Path

```bash
make install
```

**What it does**:
1. Runs unit tests (`make test-unit`)
2. Builds the binary (`make build`)
3. Copies `build/grctool` to `~/.local/bin/`

Ensure `~/.local/bin` is in your `PATH`.

## Release Builds (GoReleaser)

### Configuration

Release builds use GoReleaser v2, configured in `.goreleaser.yml`:

- **CGO disabled**: `CGO_ENABLED=0` for static binaries
- **Platforms**: Linux (amd64, arm64) and macOS (amd64, arm64) only — no Windows (unlike `make build-all` which also includes Windows)
- **Flags**: `-trimpath` for reproducible builds, `-s -w` for stripped binaries
- **Version injection**: Same `ldflags` as Makefile but using GoReleaser template variables
- **Archives**: `tar.gz` with LICENSE, README, and CHANGELOG
- **Checksums**: SHA-256 in `checksums.txt`

### Creating a Release

Releases are triggered by pushing a semver tag:

```bash
# 1. Ensure all quality gates pass
make ci

# 2. Tag the release
git tag -a v1.2.3 -m "Release v1.2.3"

# 3. Push the tag (triggers the release workflow)
git push origin v1.2.3
```

### Release Pipeline (`release.yml`)

When a `v*.*.*` tag is pushed:

1. **Pre-release checks** (parallel):
   - Lefthook CI validation
   - Tag format validation (`v1.2.3` or `v1.2.3-beta.1`)
   - CHANGELOG entry check

2. **Release job**:
   - Full checkout with history (for changelog generation)
   - Run unit tests for fast validation
   - Execute GoReleaser with `--clean`
   - Upload release artifacts

### Snapshot Builds

For non-tagged commits, GoReleaser produces snapshot builds:
```
grctool_0.1.0-SNAPSHOT-abc1234_darwin_arm64.tar.gz
```

### Installation from Release

```bash
# Automated installer (recommended)
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash

# System-wide installation
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash -s -- --system

# Self-update for existing installations
grctool update install
```

## Pre-Commit Hooks (Lefthook)

Lefthook manages all git hooks. Hooks run in parallel for speed.

### Pre-Commit Hooks

| # | Hook | Scope | Behavior |
|---|------|-------|----------|
| 1 | `file-size` | All staged files | Rejects oversized files |
| 2 | `gofmt` | `*.go` | Checks formatting |
| 3 | `goimports` | `*.go` | Checks import organization |
| 4 | `go-vet` | `*.go` | Checks for common mistakes |
| 5 | `build` | `*.go` | Verifies compilation |
| 6 | `test` | `*.go` | Runs short tests on changed packages |
| 7 | `lint` | `*.go` | golangci-lint on changes (non-blocking) |
| 8 | `security` | `*.go` | gosec scan (non-blocking) |
| 9 | `secrets` | All staged files | `detect-secrets.sh` (blocking) |
| 10 | `docs` | `*.go` | Checks for undocumented exports (non-blocking) |
| 11 | `debug-artifacts` | `*.go` | Warns about `fmt.Print`, `TODO`, etc. (non-blocking) |

### Pre-Push Hooks

| Hook | Behavior |
|------|----------|
| `test-full` | Runs full test suite (excluding tugboat package) |
| `coverage` | Checks coverage threshold |

### Commit Message Validation

The `commit-msg` hook runs `scripts/check-commit-msg.sh` to enforce conventional commit format.

### Skipping Hooks

```bash
# Skip specific hook categories
LEFTHOOK_EXCLUDE=test,lint git commit

# Skip all hooks (NOT RECOMMENDED)
git commit --no-verify
```

## CI/CD Build Pipeline

### Pipeline Overview (`ci.yml`)

```
ci-checks ─────┬──> test-matrix ──────┐
               ├──> build-matrix ─────┤
               ├──> coverage ─────────┤
               │                      │
license-check ─┤                      ├──> quality-gates
               │                      │
security-scan ─┘                      │
                                      │
```

### CI Jobs

#### ci-checks
- Installs lefthook, golangci-lint, goimports, gosec
- Runs `LEFTHOOK_CONFIG=lefthook-ci.yml lefthook run ci`

#### test-matrix
- Runs unit tests: `go test -timeout=30s -v ./internal/... ./cmd/... -count=1`
- Runs integration tests: `go test -tags=integration -timeout=2m -v ./internal/... ./test/integration/... -count=1`

#### build-matrix
- Builds `linux/amd64` binary
- Uploads as build artifact

#### coverage
- Generates `coverage.out` with atomic coverage mode
- Checks against 15% threshold (realistic for current state)
- Uploads to Codecov
- Generates HTML coverage report as artifact

#### security-scan
- Runs gosec with JSON and SARIF output
- Uploads SARIF to GitHub Code Scanning
- Runs `detect-secrets.sh` on all tracked files
- Non-blocking (informational only)

#### license-check
- Validates license headers via `scripts/check-license-headers.sh`

#### quality-gates
- Runs after all other jobs complete
- Fails if any of: ci-checks, test-matrix, build-matrix, coverage, or license-check failed
- Security scan is non-blocking

## Quality Gates Before Merge

### Automated Gates (CI)

| Gate | Source | Blocking |
|------|--------|----------|
| Compilation | `go build` | Yes |
| Unit tests | `go test ./internal/... ./cmd/...` | Yes |
| Integration tests | VCR playback tests | Yes |
| Code formatting | `gofmt` | Yes |
| Import organization | `goimports` | Yes |
| Go vet | `go vet` | Yes |
| Linting | `golangci-lint` | Yes (CI) |
| Coverage threshold | 15%+ (increasing) | Yes |
| License headers | Header check script | Yes |
| Security scan | gosec | No (informational) |
| Secret detection | `detect-secrets.sh` | Yes (pre-commit) |

### Manual Gates

| Gate | Requirement |
|------|------------|
| Code review | All changes require peer review |
| Security review | Changes to auth, crypto, or API clients require security-focused review |
| Benchmark check | Performance-sensitive changes should run `make bench-compare` |

## Docker Build

```bash
# Build Docker image
make docker-build
# Equivalent to: docker build -t grctool:$(VERSION) .

# Run in Docker container
make docker-run
# Mounts local configs directory
```

## Cleanup

```bash
# Remove build artifacts
make clean
# Removes: build/, dist/, coverage.out, coverage.html

# Clean AI context cache
make ai-clean
# Removes: .ai-context/
```

## Build Information

```bash
# Show version metadata
make version
# Displays: Version, Build Time, Git Commit

# Show full build info
make info
# Adds: Binary Name, directories, Go version
```

## References

- Makefile: `Makefile`
- GoReleaser config: `.goreleaser.yml`
- Lefthook config: `lefthook.yml`
- CI workflow: `.github/workflows/ci.yml`
- Release workflow: `.github/workflows/release.yml`
- Testing workflow: `.github/workflows/testing.yml`
- golangci-lint config: `.golangci.yml`
- Secret detection: `scripts/detect-secrets.sh`

---

*These build procedures reflect GRCTool's actual Makefile targets, CI pipeline, and release configuration. All commands are copy-pasteable from a checkout of the repository.*
