---
title: "Secure Coding Guidelines"
phase: "04-build"
category: "security"
tags: ["security", "coding-guidelines", "go", "input-validation", "credentials", "tls", "path-traversal"]
related: ["security-tests", "development-practices", "testing-strategy"]
created: 2026-03-17
updated: 2026-03-17
helix_mapping: "04-Build / Secure Coding"
---

# Secure Coding Guidelines

## Overview

This document establishes secure coding practices for GRCTool. GRCTool handles authentication tokens, compliance data, and communicates with external APIs (GitHub, Tugboat Logic, Google Workspace). These guidelines address the specific threat surface of a CLI tool that processes sensitive compliance evidence and stores credentials locally.

## Core Principles

1. **Fail Secure**: On error, default to the most restrictive behavior. Never expose partial data on failure.
2. **Zero Trust Inputs**: Validate all inputs at the boundary, including CLI arguments, config file values, and API responses.
3. **Least Privilege**: Request only the permissions needed. Store only the data required.
4. **Defense in Depth**: Layer multiple controls -- validation, sanitization, encoding, and output filtering.

## Go-Specific Security Patterns

### Import Organization

Always separate standard library, third-party, and internal imports. This makes it easy to audit third-party dependencies:

```go
import (
    "context"
    "fmt"

    "github.com/spf13/cobra"

    "github.com/grctool/grctool/internal/models"
)
```

### Use `crypto/rand` for Security-Sensitive Random Values

```go
// WRONG: math/rand is predictable
import "math/rand"
token := rand.Intn(1000000)

// CORRECT: crypto/rand for tokens, nonces, identifiers
import "crypto/rand"
b := make([]byte, 32)
_, err := rand.Read(b)
```

gosec flags use of `math/rand` in security contexts.

### Avoid Unsafe Package

Never use `unsafe` in application code. If a dependency requires it, document the justification.

### Context Propagation

Always pass `context.Context` through the call chain. This enables:
- Timeout enforcement for network operations
- Cancellation propagation
- Request-scoped logging

```go
func (s *Service) CollectEvidence(ctx context.Context, taskRef string) (*Evidence, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    return s.client.Fetch(ctx, taskRef)
}
```

## Input Validation Patterns

### Task Reference Validation

All evidence task references must match the `ET-NNNN` format. Validation occurs at the boundary (CLI command handler or API entry point).

```go
// Pattern from internal/services/validation
var validTaskRefPattern = regexp.MustCompile(`^ET-\d{4}$`)

func ValidateTaskRef(taskRef string) error {
    if !validTaskRefPattern.MatchString(taskRef) {
        return fmt.Errorf("invalid task reference format: expected ET-NNNN, got %q", taskRef)
    }
    return nil
}
```

**Key rules**:
- Validate format before any file system or network operation
- Reject lowercase (`et-0001`), missing prefix (`0001`), and excessively long values
- Never concatenate unvalidated task refs into file paths

### Window Format Validation

Evidence windows must match `YYYY-QN` (quarterly) or `YYYY-MM-DD` (date) format.

[NOT YET IMPLEMENTED] A `ValidateWindow()` function does not currently exist in the codebase. The following is the proposed pattern for when it is implemented:

```go
var (
    quarterlyPattern = regexp.MustCompile(`^\d{4}-Q[1-4]$`)
    datePattern      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func ValidateWindow(window string) error {
    if !quarterlyPattern.MatchString(window) && !datePattern.MatchString(window) {
        return fmt.Errorf("invalid window format: expected YYYY-QN or YYYY-MM-DD, got %q", window)
    }
    return nil
}
```

### File Extension Validation

Only accept known-safe evidence file extensions:

```go
var allowedExtensions = map[string]bool{
    ".md": true, ".csv": true, ".json": true, ".pdf": true,
    ".txt": true, ".yaml": true, ".yml": true,
}

func ValidateFileExtension(filename string) error {
    ext := strings.ToLower(filepath.Ext(filename))
    if !allowedExtensions[ext] {
        return fmt.Errorf("unsupported file extension: %s", ext)
    }
    return nil
}
```

### Boundary Testing

Always test edge cases in validation:
- Empty strings
- Strings at maximum length
- Unicode and multi-byte characters
- Null bytes embedded in strings
- Strings containing path separators

## Error Handling

### No Sensitive Data in Error Messages

Error messages may appear in logs, terminal output, or CI artifacts. Never include:
- Tokens or credentials
- File system paths that reveal deployment structure
- Internal IP addresses or hostnames
- Stack traces in user-facing errors

```go
// WRONG: leaks token value
return fmt.Errorf("authentication failed with token %s", token)

// WRONG: leaks internal path
return fmt.Errorf("config not found at %s", filepath.Join(homeDir, ".grctool.yaml"))

// CORRECT: generic message with safe context
return fmt.Errorf("authentication failed: invalid or expired GitHub token")

// CORRECT: wrap with context but not sensitive details
return fmt.Errorf("failed to load configuration: %w", err)
```

### Error Wrapping

Use `%w` for wrapping to preserve error chains. Use `errors.Is()` and `errors.As()` for checking:

```go
if err := validateTask(taskID); err != nil {
    return fmt.Errorf("failed to validate task %s: %w", taskID, err)
}

// Caller can check:
if errors.Is(err, ErrTaskNotFound) {
    // handle not found
}
```

### Custom Error Types

Define sentinel errors for common conditions:

```go
var (
    ErrTaskNotFound     = errors.New("evidence task not found")
    ErrInvalidFormat    = errors.New("invalid format")
    ErrPermissionDenied = errors.New("permission denied")
)
```

Use typed errors when callers need to extract details:

```go
type ValidationError struct {
    Field   string
    Value   interface{}
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}
```

### Fail Secure on Errors

When an operation fails, do not return partial or default data that could mislead:

```go
// WRONG: returns empty evidence that could be submitted
func CollectEvidence(ctx context.Context) (*Evidence, error) {
    result, err := fetch(ctx)
    if err != nil {
        return &Evidence{}, nil  // silent failure
    }
    return result, nil
}

// CORRECT: propagate the error
func CollectEvidence(ctx context.Context) (*Evidence, error) {
    result, err := fetch(ctx)
    if err != nil {
        return nil, fmt.Errorf("evidence collection failed: %w", err)
    }
    return result, nil
}
```

## Credential Management

### Token Storage

- Store tokens in files with `0600` permissions (owner read/write only)
- Use `os.OpenFile` with explicit mode, not `os.Create` (which uses `0666`)
- Store credentials in a dedicated directory (e.g., `auth/`) that is `.gitignore`-d

```go
func writeToken(path string, token []byte) error {
    return os.WriteFile(path, token, 0600)
}
```

### Token Usage

- Never log tokens at any log level
- Clear tokens from memory when no longer needed (where Go's garbage collector permits)

[NOT YET IMPLEMENTED] A `maskToken()` function does not currently exist in the codebase. The code avoids logging token values entirely rather than masking them. If a masking function is added in the future, it should show only the first 4 characters:

```go
// Proposed pattern (not yet implemented):
func maskToken(token string) string {
    if len(token) <= 4 {
        return "****"
    }
    return token[:4] + "****"
}
```

### VCR Cassette Credential Redaction

When recording HTTP interactions, the VCR system must redact sensitive headers:

```go
config := &vcr.Config{
    SanitizeHeaders: true,
    RedactHeaders:   []string{"authorization", "cookie", "x-api-key"},
}
```

Always verify that cassettes do not contain credentials before committing. The `detect-secrets.sh` script scans for this.

### Environment Variables

- Read tokens from environment variables or secure config, never hardcode
- Document required environment variables and their purpose
- Validate token format before use

## HTTP Client Security

### TLS Enforcement

All outbound connections must use HTTPS. Do not disable certificate verification:

```go
// WRONG: disables TLS verification
client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    },
}

// CORRECT: use default TLS verification
client := &http.Client{
    Timeout: 30 * time.Second,
}
```

### Timeouts

Always configure timeouts on HTTP clients to prevent hanging connections:

```go
client := &http.Client{
    Timeout: 30 * time.Second,
}
```

For finer control, use context-based timeouts:

```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
```

### User-Agent

Set a descriptive User-Agent header so API providers can identify the client:

```go
req.Header.Set("User-Agent", fmt.Sprintf("grctool/%s", version))
```

### Response Body Limits

Limit response body reading to prevent memory exhaustion:

```go
body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB max
```

### Redirect Policy

For endpoints handling authentication, limit or disable redirect following:

```go
client := &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        if len(via) >= 3 {
            return fmt.Errorf("too many redirects")
        }
        return nil
    },
}
```

## File System Security

### Path Traversal Prevention

Never construct file paths by directly concatenating user input. Always validate and sanitize.

GRCTool implements path safety validation via `Validator.ValidatePathSafety()` in `internal/tools/validator.go`. This method:

1. Rejects empty paths
2. Cleans and normalizes the path with `filepath.Clean()`
3. Rejects path traversal attempts (paths containing `..`)
4. Rejects absolute paths (only relative paths under the data directory are allowed)
5. Verifies the resolved path stays within the configured data directory boundary

```go
// Actual implementation: internal/tools/validator.go
func (v *Validator) ValidatePathSafety(path string) (*ValidationResult, error) {
    // ...
    cleanPath := filepath.Clean(path)

    // Check for path traversal attempts
    if strings.Contains(cleanPath, "..") {
        // returns ValidationResult with Valid=false, rule="no_traversal"
    }

    // Check for absolute paths when relative expected
    if filepath.IsAbs(cleanPath) {
        // returns ValidationResult with Valid=false, rule="relative_only"
    }

    // Ensure path would be under data directory
    if v.dataDir != "" {
        // validates joined path stays within data directory boundary
    }
    // ...
}
```

### File Permissions

| File Type | Permissions | Rationale |
|-----------|-------------|-----------|
| Credentials / tokens | `0600` | Owner read/write only |
| Configuration files | `0644` | Owner write, world read |
| Directories | `0755` | Owner write, world traverse |
| Evidence files | `0644` | Shared read access |
| Generated binaries | `0755` | Executable |

### Temporary Files

Use Go's built-in temp facilities which handle cleanup:

```go
// Preferred: auto-cleanup via testing
tmpDir := t.TempDir()

// Production code: explicit cleanup
tmpFile, err := os.CreateTemp("", "grctool-*.json")
if err != nil {
    return err
}
defer os.Remove(tmpFile.Name())
defer tmpFile.Close()
```

## Template Injection Prevention

GRCTool uses template variables (e.g., `{{organization.name}}`) in policy documents. Prevent template injection:

### Safe Template Rendering

- Use Go's `text/template` or `html/template` with explicit variable maps
- Never pass user input as a template string
- Pre-define all allowed template variables

```go
// WRONG: user input as template
tmpl, _ := template.New("").Parse(userInput)

// CORRECT: user input as data, not template
tmpl, _ := template.New("").Parse("Organization: {{.Name}}")
tmpl.Execute(w, map[string]string{"Name": orgName})
```

### Interpolation Boundaries

The interpolation system (configured in `.grctool.yaml`) should:
- Only substitute known variable paths (`organization.name`, etc.)
- Reject variable names containing path separators or shell metacharacters
- Log substitutions at debug level for auditability

## Dependency Management

### go.sum Verification

The `go.sum` file provides cryptographic verification of dependencies. Always:

```bash
# Verify checksums after any dependency change
go mod verify

# Tidy and download
go mod tidy
go mod download
```

### Dependency Updates

- Review dependency changes in PRs (check `go.sum` diffs)
- Run `govulncheck ./...` before and after updates
- Prefer well-maintained dependencies with security track records
- Pin versions in `go.mod` (Go modules do this by default)

### Supply Chain Protections

- GoReleaser builds with `CGO_ENABLED=0` for static binaries (no C library dependencies)
- Build with `-trimpath` to remove local file system paths from binaries
- Release checksums are SHA-256 signed

## Logging Security

### What to Log

- Authentication events (login, logout, token refresh -- without token values)
- Authorization failures
- Evidence submission events
- Configuration changes
- Errors (with safe context)

### What NOT to Log

- Token values (avoid logging them entirely; `maskToken()` is [NOT YET IMPLEMENTED])
- Full API responses containing user data
- File contents of evidence
- Password or credential values
- Full file paths that reveal system structure

### Structured Logging

Use the project's structured logger (`internal/logger`) with typed fields:

```go
logger.Info("evidence submitted",
    logger.String("task_ref", taskRef),
    logger.String("window", window),
    logger.Int("file_count", len(files)),
)
```

## Code Review Security Checklist

Use this checklist when reviewing PRs that touch security-sensitive code:

### Authentication and Authorization
- [ ] Tokens are validated before use
- [ ] No tokens appear in log output, error messages, or VCR cassettes
- [ ] Auth providers correctly report status
- [ ] Failed authentication returns informative but safe error messages

### Input Handling
- [ ] All CLI arguments are validated at the command handler level
- [ ] File paths are sanitized against traversal
- [ ] Task references match `ET-NNNN` format
- [ ] Window values match expected patterns

### Network
- [ ] All HTTP clients have timeouts configured
- [ ] TLS verification is not disabled
- [ ] Response bodies are size-limited
- [ ] User-Agent is set

### Data Handling
- [ ] Sensitive data is not written to logs
- [ ] File permissions are restrictive for credential files (`0600`)
- [ ] Temporary files are cleaned up
- [ ] VCR cassettes redact sensitive headers

### Dependencies
- [ ] New dependencies are justified and reviewed
- [ ] `go.sum` changes match `go.mod` changes
- [ ] No known vulnerabilities (`govulncheck`)
- [ ] No use of `unsafe` package

### Error Handling
- [ ] Errors are wrapped with context (`%w`)
- [ ] Error messages do not contain sensitive data
- [ ] Failures default to secure state (no partial data returned)
- [ ] All error paths are tested

## Static Analysis Configuration

### gosec

Configured in `.golangci.yml`:

```yaml
gosec:
  severity: high
  confidence: high
  excludes:
    - G104  # Handled by errcheck
    - G304  # File path provided by user is acceptable for CLI tool
  config:
    global:
      audit: true
```

G304 (file inclusion) is excluded because GRCTool is a CLI tool where user-provided file paths are expected behavior. This is a documented and accepted risk.

### golangci-lint Security Linters

The following security-relevant linters are enabled:

| Linter | Purpose |
|--------|---------|
| `govet` | Detects common mistakes (all checks enabled except `fieldalignment`) |
| `staticcheck` | Comprehensive static analysis |
| `gosec` | Security-focused analysis |
| `gocritic` | Includes diagnostic and performance checks |

## References

- Development Practices: `docs/helix/04-build/implementation-plan/development-practices.md`
- Security Tests: `docs/helix/03-test/security-tests/security-tests.md`
- golangci-lint Configuration: `.golangci.yml`
- Secret Detection: `scripts/detect-secrets.sh`
- VCR Implementation: `internal/vcr/`
- Auth Providers: `internal/auth/`
- Evidence Validation: `internal/services/validation/`

---

*These guidelines are specific to GRCTool's Go codebase, threat surface, and toolchain. Code examples reference actual project patterns and configuration.*
