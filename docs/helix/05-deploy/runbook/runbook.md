---
title: "Operational Runbook"
phase: "05-deploy"
category: "operations"
tags: ["runbook", "operations", "troubleshooting", "installation", "maintenance"]
related: ["deployment-operations", "monitoring-setup", "security-monitoring"]
created: 2026-03-17
updated: 2026-03-17
helix_mapping: "05-Deploy runbook artifact"
---

# GRCTool Operational Runbook

**Last Updated**: 2026-03-17
**Maintained By**: GRCTool Core Team

## Service Overview

### Description

GRCTool is a CLI application that automates compliance evidence collection for SOC2, ISO27001, and other governance frameworks. It integrates with Tugboat Logic for policy/control synchronization, Claude AI for evidence generation, and infrastructure tools (Terraform, GitHub, Google Workspace) for automated evidence extraction.

### Architecture

```
User (CLI) --> grctool binary
                |
                +---> Tugboat Logic API (sync, submit)
                +---> Claude AI API (evidence generation)
                +---> GitHub API (permissions, workflows, security)
                +---> Terraform configs (local file analysis)
                +---> Google Workspace API (docs, drive)
                |
                +---> Local storage (docs/, evidence/, .cache/)
                +---> Configuration (.grctool.yaml)
                +---> Authentication (auth/ directory)
```

### Dependencies

- **Tugboat Logic**: Policy, control, and evidence task synchronization
- **Claude AI (Anthropic)**: Evidence generation and analysis
- **GitHub API**: Repository access controls, workflow analysis, security features
- **Google Workspace API**: Document and drive evidence extraction
- **Go runtime**: Version 1.24.12 (embedded in binary -- no runtime dependency for end users)

## Installation Procedures

### Automated Installation (Recommended)

```bash
# Install to ~/.local/bin (no sudo required)
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash

# Install system-wide to /usr/local/bin (requires sudo)
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash -s -- --system

# Install a specific version
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash -s -- --version v0.1.0
```

The installer automatically:
1. Detects OS (Linux, macOS) and architecture (amd64, arm64)
2. Downloads the appropriate release archive from GitHub
3. Verifies SHA256 checksum
4. Installs the binary
5. Updates PATH in shell configuration if needed

### Manual Installation

```bash
# Download for your platform (example: macOS Apple Silicon)
curl -L https://github.com/grctool/grctool/releases/download/v0.1.0/grctool_0.1.0_darwin_arm64.tar.gz | tar xz

# Verify checksum
curl -L https://github.com/grctool/grctool/releases/download/v0.1.0/checksums.txt -o checksums.txt
sha256sum -c checksums.txt

# Move to PATH
sudo mv grctool /usr/local/bin/
chmod +x /usr/local/bin/grctool
```

### Building from Source

```bash
git clone https://github.com/grctool/grctool.git
cd grctool
make build
# Binary at build/grctool

# Or install to ~/.local/bin (runs unit tests first)
make install
```

### Installation Verification

```bash
# Check version and build info
grctool version

# Verify basic functionality
grctool --help
```

### Uninstallation

```bash
# Using the installer script
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash -s -- --uninstall

# Manual removal
rm -f ~/.local/bin/grctool
# or
sudo rm -f /usr/local/bin/grctool
```

## Initial Setup

### Step 1: Initialize Configuration

```bash
grctool init
```

This creates:
- `.grctool.yaml` -- Main configuration file
- `CLAUDE.md` -- AI assistant context file
- Required directory structure for data and evidence storage

The `init` command is safe to run multiple times and will not overwrite existing configuration.

### Step 2: Authenticate with Tugboat Logic

```bash
grctool auth login
```

This opens a browser-based authentication flow (Safari on macOS) that:
1. Opens the Tugboat Logic login page
2. User completes authentication in the browser
3. Credentials are extracted and stored securely in the `auth/` directory

### Step 3: Verify Authentication

```bash
grctool auth status
```

Expected output confirms:
- Authentication token is valid
- Organization ID is correct
- Token expiration status

### Step 4: Initial Data Sync

```bash
grctool sync --verbose
```

Downloads from Tugboat Logic:
- Policies (governance documents) to `docs/policies/`
- Controls (security requirements) to `docs/controls/`
- Evidence tasks (collection requirements) to `docs/evidence_tasks/`

## Common Operations

### Data Synchronization

```bash
# Standard sync
grctool sync

# Verbose sync with timing details
grctool sync --verbose

# Validate integrity and consistency of synchronized data
grctool sync validate
# Checks that local data is consistent with remote data, validates relationships
# between entities, and identifies data integrity issues.
# Supports flags: --policies, --controls, --evidence, --framework, --json

# Show sync summary and statistics
grctool sync summary
# Displays last sync time, sync status, data freshness, and counts
# of policies, controls, and evidence tasks.
```

**Expected behavior**: Downloads latest policies, controls, and evidence tasks. Existing files are updated; no data is deleted.

**Typical duration**: 10-30 seconds depending on network and data volume.

### Evidence Collection Workflow

```bash
# 1. List all evidence tasks
grctool tool evidence-task-list

# 2. Get details for a specific task
grctool tool evidence-task-details --task-ref ET-0047

# 3. Generate evidence
grctool evidence generate ET-0047

# 4. Evaluate generated evidence quality
grctool evidence evaluate ET-0047

# 5. Review evidence (human-readable assessment)
grctool evidence review ET-0047

# 6. Configure collector URL for submission (one-time per task)
grctool evidence setup ET-0047 --collector-url "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/"

# 7. Submit evidence to Tugboat Logic
grctool evidence submit ET-0047
```

### Batch Evidence Generation

```bash
# Generate evidence for all tasks
grctool evidence generate --all

# Generate context/prompts only (no AI calls)
grctool evidence generate --all --context-only
```

### Tool Discovery and Usage

```bash
# List all 30 available tools
grctool tool --help

# Run specific infrastructure tools
grctool tool terraform-security-indexer
grctool tool github-permissions --repository org/repo
grctool tool github-workflow-analyzer
grctool tool google-workspace
```

## Troubleshooting

### Authentication Issues

#### Symptom: `grctool auth status` reports expired or invalid token

**Diagnosis:**
```bash
grctool auth status
```

**Resolution:**
```bash
# Re-authenticate
grctool auth logout
grctool auth login
grctool auth status
```

#### Symptom: Browser does not open during `auth login`

**Cause**: Browser-based authentication currently requires macOS with Safari.

**Resolution:**
1. Verify you are on macOS
2. Ensure Safari is available and not blocked
3. Check that no VPN or proxy is blocking Tugboat Logic URLs
4. As a workaround, manually authenticate through the Tugboat Logic web UI and configure credentials in `.grctool.yaml`

#### Symptom: Authentication works but API calls fail

**Diagnosis:**
```bash
# Check environment variables
env | grep -E "(CLAUDE|TUGBOAT|GITHUB)"

# Verify configuration
grctool config validate
```

**Resolution:**
- Ensure `CLAUDE_API_KEY` is set for evidence generation
- Ensure `GITHUB_TOKEN` is set for GitHub tool operations
- Verify Tugboat base URL in `.grctool.yaml`

### Sync Failures

#### Symptom: Sync times out or returns network errors

**Diagnosis:**
```bash
# Run with debug logging
GRCTOOL_LOG_LEVEL=debug grctool sync --verbose
```

**Resolution:**
1. Check network connectivity to `app.tugboatlogic.com`
2. Verify authentication is still valid: `grctool auth status`
3. Check for Tugboat Logic service status
4. Retry -- transient network issues are common
5. If persistent, check rate limits (configured at 10 req/s by default)

#### Symptom: Sync completes but data appears stale

**Resolution:**
1. Clear the cache: `rm -rf docs/.cache/`
2. Re-run sync: `grctool sync --verbose`
3. Verify file timestamps in `docs/policies/`, `docs/controls/`, `docs/evidence_tasks/`

### Evidence Generation Failures

#### Symptom: Evidence generation fails with Claude API errors

**Diagnosis:**
```bash
GRCTOOL_LOG_LEVEL=debug grctool evidence generate ET-0047
```

**Resolution:**
1. Verify `CLAUDE_API_KEY` environment variable is set and valid
2. Check API key has sufficient quota
3. Verify the evidence task exists: `grctool tool evidence-task-details --task-ref ET-0047`
4. Check that dependent data has been synced: `grctool sync`

#### Symptom: Evidence quality is low or incomplete

**Resolution:**
1. Ensure latest data is synced: `grctool sync`
2. Review task requirements: `grctool tool evidence-task-details --task-ref ET-XXXX`
3. Run evaluation: `grctool evidence evaluate ET-XXXX`
4. Re-generate with fresh context: `grctool evidence generate ET-XXXX`

### GitHub Tool Failures

#### Symptom: GitHub tools return authentication errors

**Resolution:**
```bash
# Verify GitHub token
echo $GITHUB_TOKEN | head -c 10

# Test token directly
curl -H "Authorization: Bearer $GITHUB_TOKEN" https://api.github.com/user

# If using gh CLI
gh auth status
export GITHUB_TOKEN=$(gh auth token)
```

#### Symptom: GitHub rate limiting errors

**Resolution:**
1. Wait for rate limit window to reset (typically 1 hour)
2. Reduce concurrent tool execution
3. Use VCR recordings for testing instead of live API calls

## Cache Management

### Cache Location

GRCTool stores cache files in `docs/.cache/` (configurable via `.grctool.yaml` `storage.cache_dir`).

### Clearing the Cache

```bash
# Remove all cached data
rm -rf docs/.cache/

# The cache is automatically rebuilt on next operation
```

### When to Clear the Cache

- After upgrading GRCTool to a new version
- When sync data appears stale despite re-syncing
- After changing configuration that affects data processing
- When troubleshooting unexpected behavior

The cache is safe to delete at any time. It will be regenerated automatically.

## Configuration Management

### Configuration File Location

Primary configuration: `.grctool.yaml` in the project root.

### Key Configuration Sections

```yaml
tugboat:
  base_url: "https://app.tugboatlogic.com"
  org_id: "your-org-id"
  timeout: 30s
  rate_limit: 10

evidence:
  claude:
    api_key: "${CLAUDE_API_KEY}"   # Use environment variable
    model: "claude-3-sonnet-20240229"
    timeout: 60s

storage:
  data_dir: "./docs"
  cache_dir: "./docs/.cache"

logging:
  level: "info"
  format: "json"
  output: "stdout"

interpolation:
  enabled: true
  variables:
    organization:
      name: "Your Organization"
```

### Environment Variables

| Variable | Purpose | Required |
|----------|---------|----------|
| `CLAUDE_API_KEY` | Anthropic API key for evidence generation | Yes (for evidence generation) |
| `GITHUB_TOKEN` | GitHub API access token | Yes (for GitHub tools) |
| `GRCTOOL_LOG_LEVEL` | Override log level (trace/debug/info/warn/error) | No |
| `GRCTOOL_ENV` | Environment name (development/staging/production) | No |

### Configuration Validation

```bash
grctool config validate
```

## Upgrade Procedures

### Using the Self-Update Command

```bash
grctool update install
```

### Using the Installer Script

```bash
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash
```

The installer will overwrite the existing binary while preserving configuration.

### Manual Upgrade

1. Download the new version from GitHub Releases
2. Verify checksum
3. Replace the existing binary
4. Verify: `grctool version`
5. Clear cache if recommended by release notes: `rm -rf docs/.cache/`

### Post-Upgrade Verification

```bash
# Verify new version
grctool version

# Verify configuration compatibility
grctool config validate

# Verify authentication
grctool auth status

# Test sync
grctool sync --verbose
```

## Recovery Procedures

### Recovering from Corrupted Configuration

```bash
# Back up current config
cp .grctool.yaml .grctool.yaml.backup

# Re-initialize
grctool init

# Restore custom settings from backup
# Compare .grctool.yaml with .grctool.yaml.backup and merge
```

### Recovering from Corrupted Data

```bash
# Clear all synced data and cache
rm -rf docs/policies/ docs/controls/ docs/evidence_tasks/ docs/.cache/

# Re-sync from Tugboat Logic
grctool sync --verbose
```

Evidence files in `evidence/` are generated artifacts and can be regenerated:

```bash
grctool evidence generate ET-XXXX
```

### Recovering from Authentication Loss

```bash
grctool auth logout
grctool auth login
grctool auth status
```

### Disaster Recovery: Complete Reinstallation

```bash
# 1. Save configuration
cp .grctool.yaml /tmp/grctool-config-backup.yaml

# 2. Reinstall
curl -fsSL https://raw.githubusercontent.com/grctool/grctool/main/scripts/install.sh | bash

# 3. Restore configuration
cp /tmp/grctool-config-backup.yaml .grctool.yaml

# 4. Re-authenticate
grctool auth login

# 5. Re-sync data
grctool sync --verbose

# 6. Verify
grctool auth status
grctool tool evidence-task-list
```

## Release Process

### How Releases Are Built

Releases are automated via GoReleaser (`.goreleaser.yml`) and triggered by pushing a version tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The `release.yml` GitHub Actions workflow:
1. Runs unit tests for validation
2. Invokes GoReleaser to build binaries for linux/darwin on amd64/arm64
3. Generates SHA256 checksums
4. Creates a GitHub Release with binaries, checksums, and auto-generated changelog
5. Publishes installation instructions in the release notes

### Version Format

GRCTool follows semantic versioning:
- `v1.0.0` -- Major release (breaking changes)
- `v1.1.0` -- Minor release (new features)
- `v1.1.1` -- Patch release (bug fixes)
- `v1.2.0-rc.1` -- Release candidate
- `v1.2.0-beta.1` -- Beta

## Appendix: Important Commands Reference

```bash
# Authentication
grctool auth login          # Browser-based login
grctool auth status         # Check auth status
grctool auth logout         # Clear credentials

# Data management
grctool sync                # Sync from Tugboat Logic
grctool sync validate       # Validate synced data integrity
grctool sync summary        # Show sync summary and statistics
grctool init                # Initialize configuration

# Evidence workflow
grctool evidence generate ET-XXXX     # Generate evidence
grctool evidence evaluate ET-XXXX     # Evaluate quality
grctool evidence review ET-XXXX       # Human-readable review
grctool evidence submit ET-XXXX       # Submit to Tugboat

# Tool operations
grctool tool evidence-task-list                    # List tasks
grctool tool evidence-task-details --task-ref ET-XXXX  # Task details
grctool tool github-permissions --repository org/repo  # GitHub permissions
grctool tool terraform-security-indexer            # Terraform analysis

# Diagnostics
grctool version             # Version info
grctool config validate     # Validate configuration
grctool --help              # General help
```

## Appendix: External Dependencies and Status

| Service | Purpose | Status Page |
|---------|---------|-------------|
| Tugboat Logic | Policy/control sync, evidence submission | Contact vendor |
| Anthropic (Claude) | AI evidence generation | https://status.anthropic.com |
| GitHub API | Repository analysis tools | https://www.githubstatus.com |
| Google Workspace | Document evidence extraction | https://www.google.com/appsstatus |

---

*Keep this runbook updated as GRCTool evolves. Last review: 2026-03-17*
