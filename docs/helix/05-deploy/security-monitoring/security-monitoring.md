---
title: "Security Monitoring"
phase: "05-deploy"
category: "security"
tags: ["security", "monitoring", "secrets", "vulnerability", "scanning", "incident-response"]
related: ["monitoring-setup", "runbook", "deployment-operations"]
created: 2026-03-17
updated: 2026-03-17
helix_mapping: "05-Deploy security-monitoring artifact"
---

# Security Monitoring

## Overview

GRCTool handles sensitive compliance data, API credentials, and authentication tokens. This document describes the security monitoring controls in place across the development lifecycle, CI/CD pipeline, and operational use. As a CLI tool that processes compliance evidence, GRCTool must practice the same security hygiene it helps organizations demonstrate.

## Pre-Commit Secret Detection

### Lefthook Integration

Secret detection runs automatically on every commit through the Lefthook pre-commit hook configuration (`lefthook.yml`):

```yaml
pre-commit:
  commands:
    secrets:
      run: scripts/detect-secrets.sh {staged_files}
```

### Detection Capabilities

The `scripts/detect-secrets.sh` script scans for:

**API Keys and Tokens:**
- Generic API keys and access tokens (32+ character patterns)
- Bearer token patterns
- AWS access key IDs (`AKIA...`) and secret access keys
- Google Cloud API keys (`AIza...`)
- GitHub personal access tokens (`ghp_`, `gho_`, `ghu_`, `ghs_`, `ghr_`)
- Slack tokens (`xox[baprs]-...`)
- Stripe keys (`sk_test_`, `sk_live_`, `pk_test_`, `pk_live_`)

**Private Keys:**
- RSA, DSA, EC, OpenSSH, and PGP private key headers

**Credentials:**
- Password assignments in code
- Database connection strings (PostgreSQL, MySQL, MongoDB, Redis)

**Client Secrets:**
- OAuth client secrets
- Generic secret assignments

### False Positive Management

The script maintains an allow-list for known safe patterns:

- Test/example credentials (`test`, `example`, `sample`, `mock`, `placeholder`)
- Environment variable placeholders (`${...}`)
- Template placeholders (`{{...}}`)
- Documentation examples

### Skipped Files

The scanner automatically skips:
- Binary files
- Markdown and text documentation (`.md`, `.txt`)
- Go module files (`.sum`, `.mod`)
- Media files (`.png`, `.jpg`, `.svg`)
- Vendor and test fixture directories

### Handling Detection Failures

When secrets are detected:

1. The commit is blocked
2. File names and line numbers are reported
3. The developer must either:
   - Remove the actual secret and use environment variables
   - Add a pattern to `ALLOWED_PATTERNS` if it is a false positive
   - Add the file to `.gitignore` if it legitimately contains secrets

Bypassing with `--no-verify` is documented but explicitly discouraged.

## Dependency Vulnerability Scanning

### govulncheck

GRCTool uses `govulncheck` for Go dependency vulnerability detection:

```bash
# Run vulnerability check
make vulnerability-check

# Direct invocation
govulncheck ./...
```

The `vulnerability-check` Makefile target automatically installs `govulncheck` if not present.

### What govulncheck Detects

- Known vulnerabilities in Go standard library
- Vulnerabilities in direct and transitive dependencies
- Only reports vulnerabilities in code paths actually used by GRCTool (reducing false positives)

### CI/CD Status

govulncheck is available locally via `make vulnerability-check` but is **not** currently in the automated CI pipeline. The `ci.yml` workflow does not include govulncheck, and `testing.yml` (which could run it weekly) is currently disabled.

### Recommended Schedule

- Before every release (part of `release-prep` target)
- Weekly as part of maintenance routine (run `make vulnerability-check` manually)
- After any `go.sum` changes (dependency updates)

## Security Scanning in CI/CD

### gosec Static Analysis

The CI pipeline (`ci.yml`) runs gosec on every push to `main` and every pull request:

```yaml
security-scan:
  steps:
    - name: Run gosec security scan
      run: |
        gosec -fmt json -out gosec-results.json ./... || true
        gosec -fmt sarif -out gosec-results.sarif -severity medium ./... || true
```

**Capabilities:**
- SQL injection detection
- Hardcoded credential detection
- Insecure cryptographic usage
- File permission issues
- Integer overflow risks
- HTTP security issues

**Output:**
- JSON results uploaded as CI artifacts
- SARIF format uploaded to GitHub Code Scanning (via CodeQL integration)
- Results are currently informational (non-blocking) to avoid false-positive friction

### golangci-lint Security Rules

The linter configuration includes security-focused checks:

```yaml
# lefthook.yml pre-commit
lint:
  run: golangci-lint run --new-from-rev=HEAD~1
```

golangci-lint aggregates multiple security-relevant linters including `gosec`, `govet`, and `staticcheck` which catch common security issues.

### Secret Detection in CI

The CI security scan job also runs secret detection:

```yaml
- name: Run secret detection
  run: ./scripts/detect-secrets.sh $(git ls-files)
```

This scans the entire repository, not just staged files, catching secrets that might have been committed with `--no-verify`.

### License Header Verification

The `license-check` CI job ensures all source files carry the Apache 2.0 license header, preventing accidental inclusion of code with incompatible licenses:

```yaml
license-check:
  steps:
    - name: Check License Headers
      run: ./scripts/check-license-headers.sh
```

## Authentication Failure Monitoring

### Detecting Authentication Issues

Authentication failures are surfaced through:

1. **CLI exit codes**: Non-zero exit when auth fails
2. **Structured logs**: Error-level log entries with `component: auth`
3. **`grctool auth status` command**: Reports token validity and expiration

### Common Authentication Failure Patterns

| Pattern | Likely Cause | Resolution |
|---------|-------------|------------|
| Token expired | Session timeout | `grctool auth logout && grctool auth login` |
| Invalid credentials | Password changed in Tugboat | Re-authenticate via browser |
| Network timeout on auth | Tugboat API unreachable | Check network, try again later |
| Cookie extraction failure | Safari automation issue | Update macOS, check Safari permissions |

### Monitoring Authentication Health

For automated environments, wrap auth checks in monitoring scripts:

```bash
#!/bin/bash
if ! grctool auth status > /dev/null 2>&1; then
    echo "ALERT: GRCTool authentication has failed"
    # Send notification
    exit 1
fi
```

## API Credential Rotation Procedures

### Credentials Managed by GRCTool

| Credential | Storage | Rotation Frequency |
|------------|---------|-------------------|
| Tugboat Logic auth token | `auth/` directory | Session-based, re-authenticate when expired |
| Claude API key | `CLAUDE_API_KEY` env var | Per Anthropic policy (rotate on exposure) |
| GitHub token | `GITHUB_TOKEN` env var | Every 90 days or on exposure |
| Google Workspace credentials | OAuth tokens | Per Google policy |

### Rotation Procedure

**Claude API Key:**
1. Generate new key at https://console.anthropic.com
2. Update `CLAUDE_API_KEY` environment variable
3. Verify: `grctool evidence generate ET-0001` (test with one task)
4. Revoke old key

**GitHub Token:**
1. Generate new token at https://github.com/settings/tokens
2. Update `GITHUB_TOKEN` environment variable
3. Verify: `grctool tool github-permissions --repository org/repo`
4. Revoke old token

**Tugboat Logic:**
1. Run `grctool auth logout`
2. Run `grctool auth login` (browser-based re-authentication)
3. Verify: `grctool auth status`

### Credential Storage Guidelines

- Never store credentials in `.grctool.yaml` directly
- Use environment variables for API keys
- The `auth/` directory should be in `.gitignore`
- Credentials should not appear in log output (zerolog redaction hook is in place)

## Incident Response for Credential Exposure

### If a Secret Is Committed to Git

**Immediate Actions (within 15 minutes):**

1. **Revoke the exposed credential immediately**
   - Rotate the API key/token at the provider
   - Do not wait for the PR to be merged or the commit to be reverted

2. **Remove from git history**
   ```bash
   # Use git-filter-repo (preferred over filter-branch)
   pip install git-filter-repo
   git filter-repo --invert-paths --path <file-with-secret>
   ```

3. **Force-push the cleaned history**
   ```bash
   git push --force-with-lease origin main
   ```

4. **Verify removal**
   ```bash
   git log --all --full-history -- <file-with-secret>
   scripts/detect-secrets.sh $(git ls-files)
   ```

### If a Secret Is Found in CI Logs

1. Delete the CI run logs if possible
2. Rotate the exposed credential
3. Review log configuration to ensure secrets are not echoed
4. Add the secret pattern to the detect-secrets allow-list if it was a configuration issue

### If a Secret Is Found in Evidence Output

1. Remove the evidence file containing the secret
2. Re-generate evidence with proper redaction
3. If evidence was submitted to Tugboat, contact Tugboat Logic support
4. Review tool configuration for data redaction settings

### Post-Incident Review

After any credential exposure:

1. Document the incident (what was exposed, for how long, potential impact)
2. Verify the credential has been rotated
3. Check for unauthorized usage of the credential during the exposure window
4. Update detection patterns if the secret type was not caught by `detect-secrets.sh`
5. Add a test case to prevent recurrence

## Security Monitoring Checklist

### Development Phase
- [x] Pre-commit secret detection via Lefthook
- [x] gosec static security analysis in pre-commit hooks
- [x] golangci-lint with security-focused rules
- [x] Debug artifact detection (fmt.Print, log.Print, panic)

### CI/CD Phase
- [x] gosec scan with SARIF output to GitHub Code Scanning
- [x] Full-repository secret detection scan
- [x] License header verification
- [ ] Dependency vulnerability scanning (govulncheck) -- available locally via `make vulnerability-check` but NOT currently in the automated CI pipeline (`ci.yml` does not run govulncheck; `testing.yml` is disabled)
- [x] Quality gates requiring all security checks to pass

### Runtime Phase
- [x] Authentication status verification command
- [x] Environment variable-based credential management
- [x] Structured logging with potential for secret redaction
- [x] No telemetry or data transmission without user action

### Operational Phase
- [ ] Scheduled dependency vulnerability scans (weekly via testing.yml)
- [ ] Credential rotation reminders (manual process)
- [ ] Incident response playbook (documented above)

## Security Metrics

| Metric | How to Measure | Target |
|--------|---------------|--------|
| Secret detection false positive rate | Review blocked commits over time | < 5% |
| gosec findings (medium+) | CI artifact `gosec-results.json` | 0 critical, < 5 medium |
| Known dependency vulnerabilities | `govulncheck ./...` output | 0 critical, 0 high |
| Time to rotate exposed credential | Incident response logs | < 15 minutes |
| License header compliance | `scripts/check-license-headers.sh` exit code | 100% |

## References

- `scripts/detect-secrets.sh` -- Secret detection implementation
- `lefthook.yml` -- Pre-commit hook configuration
- `.github/workflows/ci.yml` -- CI security scanning
- `Makefile` -- `security-scan` and `vulnerability-check` targets
- `internal/logger/zerolog_logger.go` -- Logging with RedactionHook
- `.goreleaser.yml` -- Release build configuration (no secrets embedded)

---

*Security monitoring is a continuous process. This document should be updated whenever new credential types are introduced or detection capabilities are enhanced.*
