---
title: "Security Test Specifications"
phase: "03-test"
category: "security-testing"
tags: ["security", "testing", "authentication", "authorization", "injection", "credentials", "sast"]
related: ["testing-strategy", "security-architecture", "threat-model"]
created: 2026-03-17
updated: 2026-03-17
helix_mapping: "03-Test / Security Tests"
---

# Security Test Specifications

## Overview

This document defines the security test suite for GRCTool. Every test specification maps to a concrete threat surface in the application: authentication tokens, credential storage, input handling, API communication, dependency supply chain, and pre-commit secret detection. Tests are organized by category and include implementation guidance using GRCTool's existing tooling (gosec, govulncheck, golangci-lint, and the custom `detect-secrets.sh` script).

## Security Test Categories

### 1. Authentication Testing

GRCTool authenticates against two external services (GitHub and Tugboat Logic) and exposes a `NoAuthProvider` for tools that operate without credentials. Authentication tests validate the full lifecycle of token acquisition, validation, and revocation.

#### SEC-AUTH-001: Token Validation on Provider Initialization

**Objective**: Verify that auth providers reject missing, malformed, and expired tokens at construction time or first use.

**Test Cases**:

| Test ID | Scenario | Input | Expected Result | Pass Criteria |
|---------|----------|-------|-----------------|---------------|
| SEC-AUTH-001a | Empty GitHub token | `""` | `Authenticate()` returns error | Error message does not leak internal state |
| SEC-AUTH-001b | Empty Tugboat token | `""` | `Authenticate()` returns error | `status.Authenticated == false` |
| SEC-AUTH-001c | Malformed GitHub PAT | `"not-a-ghp-token"` | Validation fails | No network request is made |
| SEC-AUTH-001d | Expired Tugboat session | Expired cookie | `ValidateAuth()` returns error | Error instructs user to re-authenticate |

**Implementation Pattern**:
```go
func TestGitHubAuthProvider_RejectsEmptyToken(t *testing.T) {
    cacheDir := t.TempDir()
    provider := NewGitHubAuthProvider("", cacheDir, &mockLogger{})

    ctx := context.Background()
    status := provider.GetStatus(ctx)

    assert.False(t, status.Authenticated)
    assert.False(t, status.TokenPresent)

    err := provider.Authenticate(ctx)
    assert.Error(t, err)
    assert.NotContains(t, err.Error(), cacheDir) // No path leakage
}
```

#### SEC-AUTH-002: Token Format Checking

**Objective**: Verify that token format is validated before any network call is made.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-AUTH-002a | GitHub token missing `ghp_` prefix | Rejected locally |
| SEC-AUTH-002b | GitHub token with correct prefix but wrong length | Rejected locally |
| SEC-AUTH-002c | Tugboat base URL with non-HTTPS scheme | Rejected or warned |

#### SEC-AUTH-003: Authentication Status Accuracy

**Objective**: Verify `GetStatus()` returns accurate state across the provider lifecycle.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-AUTH-003a | Before authentication | `Authenticated: false` |
| SEC-AUTH-003b | After successful authentication | `Authenticated: true`, correct `Provider` and `Source` |
| SEC-AUTH-003c | After `ClearAuth()` | `Authenticated: false`, `TokenPresent: false` |

#### SEC-AUTH-004: NoAuthProvider Security Boundary

**Objective**: Verify that `NoAuthProvider` cannot be used to bypass authentication for tools that require it.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-AUTH-004a | `NoAuthProvider.IsAuthRequired()` | Returns `false` |
| SEC-AUTH-004b | Tool requiring auth receives NoAuthProvider | Tool rejects the provider or errors before making API calls |

### 2. Authorization Testing

#### SEC-AUTHZ-001: Evidence Task Access Control

**Objective**: Verify that evidence operations respect task ownership and scope.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-AUTHZ-001a | Access evidence for valid task ref (ET-0001) | Succeeds |
| SEC-AUTHZ-001b | Access evidence for non-existent task ref (ET-9999) | Returns descriptive error |
| SEC-AUTHZ-001c | Malformed task ref (lowercase `et-0001`) | Validation rejects input |
| SEC-AUTHZ-001d | Task ref with path traversal (`ET-../../etc/passwd`) | Rejected by naming validation |

#### SEC-AUTHZ-002: Collector URL Validation

**Objective**: Verify that collector URLs are validated before use in submission.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-AUTHZ-002a | Valid Tugboat collector URL | Accepted |
| SEC-AUTHZ-002b | Non-HTTPS collector URL | Rejected |
| SEC-AUTHZ-002c | URL pointing to non-Tugboat domain | Rejected or warned |
| SEC-AUTHZ-002d | URL with embedded credentials | Rejected |

### 3. Input Validation Testing

#### SEC-INPUT-001: Evidence Task Reference Validation

**Objective**: Verify that all entry points validate task references against the `ET-NNNN` format.

**Implementation Reference**: `internal/services/validation/evidence_validation_test.go` -- `TestValidationRules_ValidTaskRef`

| Test ID | Scenario | Input | Expected Result |
|---------|----------|-------|-----------------|
| SEC-INPUT-001a | Valid format | `ET-0001` | Passes |
| SEC-INPUT-001b | Missing prefix | `0001` | Fails |
| SEC-INPUT-001c | Lowercase prefix | `et-0001` | Fails |
| SEC-INPUT-001d | Path injection | `ET-0001/../../secret` | Fails |
| SEC-INPUT-001e | Shell metacharacters | `ET-0001; rm -rf /` | Fails |
| SEC-INPUT-001f | Null bytes | `ET-0001\x00malicious` | Fails |

#### SEC-INPUT-002: Window Format Validation

**Implementation Reference**: `TestValidationRules_WindowFormat` [NOT YET IMPLEMENTED - `ValidateWindow()` validation function does not exist in the codebase]

| Test ID | Scenario | Input | Expected Result |
|---------|----------|-------|-----------------|
| SEC-INPUT-002a | Valid quarterly | `2025-Q4` | Passes [NOT YET IMPLEMENTED - validation function does not exist] |
| SEC-INPUT-002b | Valid date | `2025-10-22` | Passes [NOT YET IMPLEMENTED - validation function does not exist] |
| SEC-INPUT-002c | Reversed format | `Q4-2025` | Warning [NOT YET IMPLEMENTED - validation function does not exist] |
| SEC-INPUT-002d | Path injection in window | `2025-Q4/../../../etc` | Rejected [NOT YET IMPLEMENTED - validation function does not exist] |

#### SEC-INPUT-003: Terraform Content Parsing Safety

**Objective**: Verify that the Terraform HCL parser handles malicious or malformed input safely.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-INPUT-003a | Extremely large HCL file (>100MB) | Parser returns error or terminates within timeout |
| SEC-INPUT-003b | Deeply nested HCL blocks (1000+ levels) | No stack overflow; graceful error |
| SEC-INPUT-003c | HCL with embedded null bytes | Handled without crash |
| SEC-INPUT-003d | Binary file passed as `.tf` | Parser returns parse error |

#### SEC-INPUT-004: File Extension Validation

**Implementation Reference**: `TestValidationRules_ValidFileExtensions`

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-INPUT-004a | Allowed extensions (.md, .csv, .json, .pdf) | Passes |
| SEC-INPUT-004b | Executable extension (.exe) | Warning |
| SEC-INPUT-004c | Double extension (.md.exe) | Warning |
| SEC-INPUT-004d | No extension | Warning |

### 4. Credential Handling Testing

#### SEC-CRED-001: No Plaintext Credentials in Logs

**Objective**: Verify that tokens, passwords, and session cookies never appear in log output.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-CRED-001a | GitHub token used in API call | Token not present in any log level |
| SEC-CRED-001b | Tugboat session cookie in request | Cookie value not in logs |
| SEC-CRED-001c | Authentication error with token context | Error message omits token entirely [NOT YET IMPLEMENTED - no `maskToken()` function exists; tokens are omitted from logs rather than masked] |

**Implementation Pattern**:
```go
func TestNoTokenLeakageInLogs(t *testing.T) {
    var logBuf bytes.Buffer
    // Configure logger to write to buffer
    logger := newTestLogger(&logBuf)

    token := "ghp_abc123secretvalue456"
    provider := NewGitHubAuthProvider(token, t.TempDir(), logger)

    ctx := context.Background()
    _ = provider.Authenticate(ctx)

    logOutput := logBuf.String()
    assert.NotContains(t, logOutput, token)
    assert.NotContains(t, logOutput, "abc123secretvalue456")
}
```

#### SEC-CRED-002: VCR Cassette Credential Redaction

**Objective**: Verify that recorded VCR cassettes redact all sensitive headers and tokens.

**Implementation Reference**: `internal/vcr/vcr_test.go` -- `TestVCR_RecordAndPlayback`

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-CRED-002a | Authorization header in recording | Replaced with `[REDACTED]` in cassette |
| SEC-CRED-002b | API key in query parameter | Redacted or removed |
| SEC-CRED-002c | Cookie header with session token | Redacted in cassette |
| SEC-CRED-002d | Cassette replayed after redaction | Response still valid |

#### SEC-CRED-003: Credential Storage Security

**Objective**: Verify that stored credentials use appropriate file permissions and are not world-readable.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-CRED-003a | Auth cache file permissions | File mode is 0600 or more restrictive |
| SEC-CRED-003b | Config file with collector URLs | No embedded credentials in `.grctool.yaml` |
| SEC-CRED-003c | Temporary files during auth flow | Cleaned up after use |

### 5. API Security Testing

#### SEC-API-001: TLS Enforcement

**Objective**: Verify that all outbound HTTP connections use TLS.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-API-001a | Tugboat API base URL with HTTPS | Connection succeeds |
| SEC-API-001b | Tugboat API base URL with HTTP | Connection refused or upgraded |
| SEC-API-001c | GitHub API calls | Always use HTTPS |
| SEC-API-001d | TLS certificate validation | Invalid certificates are rejected |

#### SEC-API-002: HTTP Client Configuration

**Objective**: Verify that HTTP clients are configured with appropriate security defaults.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-API-002a | Request timeout | Configured (default 30s) |
| SEC-API-002b | User-Agent header | Set to identifiable value (not empty) |
| SEC-API-002c | Redirect following | Limited or disabled for sensitive endpoints |
| SEC-API-002d | Response body size limit | Bounded to prevent memory exhaustion |

#### SEC-API-003: Rate Limiting Behavior

**Objective**: Verify that GRCTool handles rate-limited responses correctly.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-API-003a | GitHub 403 with rate limit headers | Retry with backoff |
| SEC-API-003b | Tugboat 429 response | Respect `Retry-After` header |
| SEC-API-003c | Sustained rate limiting | Fail gracefully with informative error |

### 6. Dependency Vulnerability Testing

#### SEC-DEP-001: govulncheck Integration

**Objective**: Verify that known vulnerabilities in dependencies are detected.

**Execution**:
```bash
# Run as part of CI or local security scan
make vulnerability-check
# Equivalent to: govulncheck ./...
```

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-DEP-001a | Clean dependency tree | No known vulnerabilities reported |
| SEC-DEP-001b | Dependency with known CVE | govulncheck reports the vulnerability with severity |
| SEC-DEP-001c | Transitive dependency vulnerability | Detected and reported |

#### SEC-DEP-002: gosec Static Analysis

**Objective**: Verify that static analysis catches common security anti-patterns in Go code.

**Execution**:
```bash
# Run as part of CI
make security-scan
# Equivalent to: gosec ./...
```

**Configuration Reference**: `.golangci.yml` gosec settings -- severity `high`, confidence `high`, excludes G104 and G304.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-DEP-002a | Hardcoded credentials in source | Flagged by gosec |
| SEC-DEP-002b | Use of `math/rand` for security | Flagged (should use `crypto/rand`) |
| SEC-DEP-002c | Unvalidated file paths (G304) | Excluded for CLI tool (documented exception) |
| SEC-DEP-002d | SQL injection patterns | Flagged if applicable |

#### SEC-DEP-003: go.sum Integrity

**Objective**: Verify that `go.sum` checksums are validated and tamper-evident.

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-DEP-003a | `go mod verify` passes | All module checksums match |
| SEC-DEP-003b | Modified `go.sum` entry | `go mod verify` fails |
| SEC-DEP-003c | Missing `go.sum` entry | `go mod tidy` detects and fails build |

### 7. Pre-Commit Secret Detection Testing

#### SEC-SECRET-001: detect-secrets.sh Coverage

**Objective**: Verify that the secret detection script catches known secret patterns and respects the allowlist.

**Implementation Reference**: `scripts/detect-secrets.sh`

| Test ID | Scenario | Pattern | Expected Result |
|---------|----------|---------|-----------------|
| SEC-SECRET-001a | AWS access key | `AKIA[0-9A-Z]{16}` | Detected |
| SEC-SECRET-001b | GitHub PAT | `ghp_[a-zA-Z0-9]{36}` | Detected |
| SEC-SECRET-001c | Private key header | `-----BEGIN RSA PRIVATE KEY` | Detected |
| SEC-SECRET-001d | Generic API key | `api_key = "abc123..."` (32+ chars) | Detected |
| SEC-SECRET-001e | Database connection string | `postgres://user:pass@host/db` | Detected |
| SEC-SECRET-001f | Test/example credential | `password = "test"` | Allowed (false positive exclusion) |
| SEC-SECRET-001g | Template placeholder | `${API_KEY}` | Allowed |
| SEC-SECRET-001h | Binary file | Compiled binary | Skipped |
| SEC-SECRET-001i | Markdown file | `.md` extension | Skipped |

#### SEC-SECRET-002: Lefthook Integration

**Objective**: Verify that secret detection runs as part of the pre-commit hook chain.

**Implementation Reference**: `lefthook.yml` -- command `secrets`

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| SEC-SECRET-002a | Staged file with secret | Pre-commit hook fails, commit blocked |
| SEC-SECRET-002b | Staged file without secrets | Pre-commit hook passes |
| SEC-SECRET-002c | Secret in non-scanned extension | Hook passes (skipped file) |

## Automated Security Testing Pipeline

### Static Application Security Testing (SAST)

```yaml
# CI integration via .github/workflows/ci.yml
sast_tools:
  gosec:
    execution: "make security-scan"
    output: "gosec-results.json, gosec-results.sarif"
    ci_job: "security-scan"
    fail_on: "informational only (non-blocking)"
  golangci-lint:
    execution: "make lint"
    security_linters: ["gosec", "govet", "staticcheck"]
    ci_job: "ci-checks"
    fail_on: "high severity"
  govulncheck:
    execution: "make vulnerability-check"
    ci_job: "manual / weekly"
    fail_on: "any known vulnerability"
```

### Dependency Scanning

```yaml
dependency_scanning:
  tool: "govulncheck"
  execution: "govulncheck ./..."
  frequency: "on every CI run + weekly scheduled"
  go_sum_verification: "go mod verify"
  threshold: "any known vulnerability"
```

### Secret Detection

```yaml
secret_detection:
  tool: "scripts/detect-secrets.sh"
  trigger: "pre-commit (lefthook), CI (security-scan job)"
  patterns: 31  # AWS, GitHub, GCP, private keys, passwords, DB strings, Slack, Stripe, Square, Twilio, generic secrets, etc.
  allowlist: 11  # Test credentials, placeholders, templates, example keys
  skip_extensions: [".md", ".txt", ".sum", ".mod", ".lock", ".pdf", ".png", ".jpg", ".svg"]
  skip_directories: ["vendor", "node_modules", ".git", "testdata", "test/fixtures"]
```

## Security Test Execution

### Local Development

```bash
# Run full security scan
make security-scan

# Run vulnerability check
make vulnerability-check

# Run secret detection on all tracked files
./scripts/detect-secrets.sh $(git ls-files)

# Run golangci-lint with security linters
make lint
```

### CI/CD Pipeline

The CI pipeline (`ci.yml`) runs security tests in the `security-scan` job:

1. **gosec** scan with JSON and SARIF output
2. **SARIF upload** to GitHub Code Scanning
3. **Secret detection** on all tracked files
4. Results are uploaded as artifacts for review

The `quality-gates` job aggregates results but treats security scan failures as non-blocking warnings to avoid disrupting development flow for informational findings.

## Security Test Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| SAST findings (high severity) | 0 | gosec JSON output |
| Known dependency vulnerabilities | 0 | govulncheck output |
| Secret detection false negatives | 0 | Periodic manual review of allowlist |
| VCR cassette credential leaks | 0 | Automated scan of cassette directory |
| Auth provider test coverage | 90%+ | `go test -cover ./internal/auth/...` |
| Input validation test coverage | 90%+ | `go test -cover ./internal/services/validation/...` |

## References

- Testing Strategy: `docs/helix/03-test/test-plan/testing-strategy.md`
- Development Practices: `docs/helix/04-build/implementation-plan/development-practices.md`
- Secret Detection Script: `scripts/detect-secrets.sh`
- Lefthook Configuration: `lefthook.yml`
- golangci-lint Configuration: `.golangci.yml`
- CI Security Job: `.github/workflows/ci.yml` (security-scan)

---

*This document specifies security tests grounded in GRCTool's actual codebase, tooling, and CI pipeline. Tests reference real implementation files and configuration to ensure they are immediately actionable.*
