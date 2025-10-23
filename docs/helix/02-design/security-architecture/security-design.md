---
title: "Security Architecture and Implementation"
phase: "02-design"
category: "security"
tags: ["security", "authentication", "secrets", "vulnerability", "compliance"]
related: ["system-architecture", "api-design", "security-requirements"]
created: 2025-01-10
updated: 2025-01-10
helix_mapping: "Consolidated from 04-Development/security.md"
---

# Security Architecture and Implementation

## Overview

Security is fundamental to GRCTool's design, given its role in SOC2 compliance and handling of sensitive organizational data. This document outlines the comprehensive security architecture, implementation patterns, and security controls throughout the system.

## Security Design Principles

### Core Security Principles
1. **Zero Trust**: Assume no implicit trust; verify everything
2. **Least Privilege**: Grant minimum necessary permissions
3. **Defense in Depth**: Multiple layers of security controls
4. **Fail Secure**: Default to secure state on failures
5. **Security by Design**: Build security in from the start

### Threat Model

#### Assets to Protect
- **Authentication Credentials**: API tokens, session cookies
- **Evidence Data**: SOC2 compliance information
- **Configuration Data**: System configuration and secrets
- **Source Code**: Application logic and algorithms

#### Threat Vectors
- **Credential Theft**: Exposed API keys or tokens
- **Code Injection**: Malicious input in templates or commands
- **Supply Chain**: Compromised dependencies
- **Information Disclosure**: Sensitive data in logs or outputs
- **Privilege Escalation**: Unauthorized access to system resources

## Authentication and Authorization Architecture

### Authentication Providers

#### GitHub Authentication
```go
type GitHubAuthProvider struct {
    token     string
    client    *http.Client
    cache     *AuthCache
    validator TokenValidator
}

func (g *GitHubAuthProvider) Authenticate(ctx context.Context) (*AuthResult, error) {
    // Validate token format
    if !g.validator.IsValidFormat(g.token) {
        return nil, ErrInvalidTokenFormat
    }

    // Check cache first
    if cached := g.cache.Get(g.token); cached != nil {
        if !cached.IsExpired() {
            return cached, nil
        }
    }

    // Validate with GitHub API
    req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", "token "+g.token)
    req.Header.Set("User-Agent", "GRCTool/1.0")

    resp, err := g.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, ErrAuthenticationFailed
    }

    // Parse response and cache result
    result := &AuthResult{
        Provider:    "github",
        Token:       g.token,
        ValidUntil:  time.Now().Add(time.Hour),
        Permissions: extractPermissions(resp),
    }

    g.cache.Set(g.token, result)
    return result, nil
}
```

#### Tugboat Authentication
```go
type TugboatAuthProvider struct {
    baseURL string
    client  *http.Client
    cache   *AuthCache
}

func (t *TugboatAuthProvider) ExtractTokenFromBrowser() (string, error) {
    // Browser-based authentication to avoid storing API keys
    // Uses Safari cookie extraction on macOS

    cmd := exec.Command("osascript", "-e", `
        tell application "Safari"
            repeat with w in windows
                repeat with t in tabs of w
                    if URL of t contains "tugboatlogic.com" then
                        return do JavaScript "document.cookie" in t
                    end if
                end repeat
            end repeat
        end tell
    `)

    output, err := cmd.Output()
    if err != nil {
        return "", ErrBrowserAuthFailed
    }

    // Extract bearer token from cookies
    cookies := parseCookieString(string(output))
    token := extractBearerToken(cookies)

    if token == "" {
        return "", ErrTokenNotFound
    }

    return token, nil
}
```

### Token Management

#### Secure Token Storage
```go
type SecureTokenStore struct {
    keyring Keyring
    cache   map[string]TokenInfo
    mutex   sync.RWMutex
}

func (s *SecureTokenStore) Store(provider, token string) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    // Encrypt token before storage
    encrypted, err := s.keyring.Encrypt([]byte(token))
    if err != nil {
        return err
    }

    // Store in system keyring
    key := fmt.Sprintf("grctool.%s.token", provider)
    err = s.keyring.Set(key, string(encrypted))
    if err != nil {
        return err
    }

    // Update cache
    s.cache[provider] = TokenInfo{
        Provider:  provider,
        ExpiresAt: time.Now().Add(24 * time.Hour),
        Encrypted: true,
    }

    return nil
}

func (s *SecureTokenStore) Retrieve(provider string) (string, error) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()

    key := fmt.Sprintf("grctool.%s.token", provider)
    encrypted, err := s.keyring.Get(key)
    if err != nil {
        return "", err
    }

    // Decrypt token
    decrypted, err := s.keyring.Decrypt([]byte(encrypted))
    if err != nil {
        return "", err
    }

    return string(decrypted), nil
}
```

#### Token Validation
```go
type TokenValidator interface {
    IsValidFormat(token string) bool
    ValidateToken(ctx context.Context, token string) (*ValidationResult, error)
    GetTokenExpiry(token string) (time.Time, error)
}

type GitHubTokenValidator struct {
    client *http.Client
}

func (g *GitHubTokenValidator) IsValidFormat(token string) bool {
    // GitHub personal access tokens: ghp_[A-Za-z0-9]{36}
    // GitHub app tokens: ghs_[A-Za-z0-9]{36}
    patterns := []string{
        `^ghp_[A-Za-z0-9]{36}$`,
        `^ghs_[A-Za-z0-9]{36}$`,
    }

    for _, pattern := range patterns {
        if matched, _ := regexp.MatchString(pattern, token); matched {
            return true
        }
    }

    return false
}
```

## Secret Management Architecture

### Environment Variable Security

#### Secure Configuration Pattern
```yaml
# .grctool.yaml - Configuration with environment variable substitution
tugboat:
  base_url: "${TUGBOAT_BASE_URL}"
  org_id: "${TUGBOAT_ORG_ID}"
  timeout: 30s

evidence:
  claude:
    api_key: "${CLAUDE_API_KEY}"
    model: "claude-3-sonnet-20240229"

github:
  token: "${GITHUB_TOKEN}"

# Secrets are NEVER stored in configuration files
# Use environment variables or secure secret stores
```

#### Environment Variable Validation
```go
func ValidateEnvironmentVariables() error {
    required := []string{
        "CLAUDE_API_KEY",
        "TUGBOAT_BASE_URL",
    }

    optional := []string{
        "GITHUB_TOKEN",
        "TUGBOAT_ORG_ID",
    }

    var missing []string
    for _, env := range required {
        if os.Getenv(env) == "" {
            missing = append(missing, env)
        }
    }

    if len(missing) > 0 {
        return fmt.Errorf("required environment variables missing: %v", missing)
    }

    // Validate optional variables are set if related features are enabled
    for _, env := range optional {
        if value := os.Getenv(env); value != "" {
            if err := validateEnvironmentValue(env, value); err != nil {
                return fmt.Errorf("invalid %s: %v", env, err)
            }
        }
    }

    return nil
}
```

### Secret Detection and Prevention

#### Pre-commit Secret Detection
```bash
#!/bin/bash
# scripts/detect-secrets.sh

set -e

echo "Scanning for secrets in staged files..."

# Secret patterns to detect
PATTERNS=(
    # API Keys
    "[Aa]pi[_-]?[Kk]ey['\"]?\s*[:=]\s*['\"][A-Za-z0-9]{20,}['\"]"
    "[Aa]pi[_-]?[Tt]oken['\"]?\s*[:=]\s*['\"][A-Za-z0-9]{20,}['\"]"

    # GitHub tokens
    "ghp_[A-Za-z0-9]{36}"
    "ghs_[A-Za-z0-9]{36}"

    # AWS credentials
    "AKIA[0-9A-Z]{16}"
    "[A-Za-z0-9/+=]{40}"

    # Private keys
    "-----BEGIN [A-Z]+ PRIVATE KEY-----"

    # Generic secrets
    "[Pp]assword['\"]?\s*[:=]\s*['\"][^'\"]{8,}['\"]"
    "[Ss]ecret['\"]?\s*[:=]\s*['\"][^'\"]{8,}['\"]"
    "[Tt]oken['\"]?\s*[:=]\s*['\"][A-Za-z0-9]{16,}['\"]"
)

# Allowed patterns (test data, examples, placeholders)
ALLOWED_PATTERNS=(
    "password.*test"
    "api[_-]?key.*example"
    "YOUR_.*_HERE"
    "REPLACE_WITH_"
    "\\$\\{.*\\}"  # Environment variable placeholders
    "{{.*}}"       # Template placeholders
)

# Get staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM)

if [[ -z "$STAGED_FILES" ]]; then
    echo "No staged files to scan"
    exit 0
fi

FOUND_SECRETS=false

for file in $STAGED_FILES; do
    # Skip binary files
    if file "$file" | grep -q "binary"; then
        continue
    fi

    # Skip certain file types
    case "$file" in
        *.png|*.jpg|*.jpeg|*.gif|*.pdf|*.zip|*.tar.gz) continue ;;
    esac

    echo "Scanning: $file"

    for pattern in "${PATTERNS[@]}"; do
        if grep -qE "$pattern" "$file"; then
            # Check if it's an allowed pattern
            ALLOWED=false
            for allowed_pattern in "${ALLOWED_PATTERNS[@]}"; do
                if grep -E "$pattern" "$file" | grep -qE "$allowed_pattern"; then
                    ALLOWED=true
                    break
                fi
            done

            if [[ "$ALLOWED" == false ]]; then
                echo "❌ POTENTIAL SECRET DETECTED in $file:"
                grep -nE "$pattern" "$file" | sed 's/^/    /'
                FOUND_SECRETS=true
            fi
        fi
    done
done

if [[ "$FOUND_SECRETS" == true ]]; then
    echo ""
    echo "❌ Potential secrets detected in staged files!"
    echo "Please review and remove any actual secrets before committing."
    echo ""
    echo "If these are false positives (test data, examples), you can:"
    echo "1. Add them to ALLOWED_PATTERNS in scripts/detect-secrets.sh"
    echo "2. Use SKIP_SECRETS=true git commit to bypass this check"
    echo ""
    exit 1
fi

echo "✅ No secrets detected in staged files"
```

### Credential Rotation

#### Token Rotation Procedure
```go
type TokenRotator struct {
    providers map[string]AuthProvider
    store     SecureTokenStore
    scheduler *cron.Cron
}

func (t *TokenRotator) ScheduleRotation(provider string, interval time.Duration) error {
    cronExpr := fmt.Sprintf("@every %s", interval.String())

    _, err := t.scheduler.AddFunc(cronExpr, func() {
        if err := t.rotateToken(provider); err != nil {
            log.Error().
                Err(err).
                Str("provider", provider).
                Msg("Token rotation failed")
        }
    })

    return err
}

func (t *TokenRotator) rotateToken(provider string) error {
    log.Info().
        Str("provider", provider).
        Msg("Starting token rotation")

    // Get current token
    oldToken, err := t.store.Retrieve(provider)
    if err != nil {
        return err
    }

    // Generate or retrieve new token
    authProvider := t.providers[provider]
    newAuth, err := authProvider.Authenticate(context.Background())
    if err != nil {
        return err
    }

    // Store new token
    if err := t.store.Store(provider, newAuth.Token); err != nil {
        return err
    }

    // Verify new token works
    if err := authProvider.ValidateToken(newAuth.Token); err != nil {
        // Rollback on failure
        t.store.Store(provider, oldToken)
        return err
    }

    log.Info().
        Str("provider", provider).
        Msg("Token rotation completed successfully")

    return nil
}
```

## Input Validation and Sanitization

### Command Injection Prevention
```go
func sanitizeCommand(input string) (string, error) {
    // Whitelist allowed characters for command arguments
    allowedChars := regexp.MustCompile(`^[a-zA-Z0-9\-_./]+$`)

    if !allowedChars.MatchString(input) {
        return "", ErrInvalidInput
    }

    // Check for command injection patterns
    dangerous := []string{
        ";", "&&", "||", "|", "`", "$(",
        "../", "./", "/bin/", "/usr/bin/",
        "rm ", "del ", "format ", "shutdown",
    }

    for _, pattern := range dangerous {
        if strings.Contains(strings.ToLower(input), pattern) {
            return "", ErrDangerousInput
        }
    }

    return input, nil
}

func ExecuteSafeCommand(command string, args ...string) (string, error) {
    // Validate command is in allowlist
    allowedCommands := map[string]bool{
        "git":     true,
        "terraform": true,
        "kubectl": true,
    }

    if !allowedCommands[command] {
        return "", ErrCommandNotAllowed
    }

    // Sanitize all arguments
    sanitizedArgs := make([]string, len(args))
    for i, arg := range args {
        sanitized, err := sanitizeCommand(arg)
        if err != nil {
            return "", fmt.Errorf("invalid argument %d: %v", i, err)
        }
        sanitizedArgs[i] = sanitized
    }

    // Execute with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, command, sanitizedArgs...)
    output, err := cmd.Output()

    return string(output), err
}
```

### Template Injection Prevention
```go
func SafeTemplateExecution(templateStr string, data interface{}) (string, error) {
    // Create template with security functions
    tmpl := template.New("safe").Funcs(template.FuncMap{
        "html":   template.HTMLEscapeString,
        "js":     template.JSEscapeString,
        "url":    template.URLQueryEscaper,
        "safe":   func(s string) template.HTML { return template.HTML(s) },
    })

    // Parse template
    parsed, err := tmpl.Parse(templateStr)
    if err != nil {
        return "", err
    }

    // Execute with limited context
    var buf bytes.Buffer
    err = parsed.Execute(&buf, data)
    if err != nil {
        return "", err
    }

    return buf.String(), nil
}
```

### Data Validation
```go
type Validator interface {
    Validate(input interface{}) error
}

type EvidenceTaskValidator struct{}

func (e *EvidenceTaskValidator) Validate(input interface{}) error {
    task, ok := input.(*models.EvidenceTask)
    if !ok {
        return ErrInvalidInputType
    }

    // Validate ID format
    if !regexp.MustCompile(`^ET-\d{4}$`).MatchString(task.ID) {
        return ErrInvalidTaskID
    }

    // Validate name length and characters
    if len(task.Name) == 0 || len(task.Name) > 255 {
        return ErrInvalidTaskName
    }

    if !regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,]+$`).MatchString(task.Name) {
        return ErrInvalidTaskNameChars
    }

    // Validate status
    validStatuses := []string{"pending", "in_progress", "completed", "failed"}
    if !contains(validStatuses, task.Status) {
        return ErrInvalidTaskStatus
    }

    return nil
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

## Vulnerability Management

### Dependency Scanning
```bash
#!/bin/bash
# scripts/vulnerability-scan.sh

set -e

echo "Running vulnerability scan..."

# Update vulnerability database
echo "Updating vulnerability database..."
go install golang.org/x/vuln/cmd/govulncheck@latest

# Scan Go modules for vulnerabilities
echo "Scanning Go modules..."
if ! govulncheck ./...; then
    echo "❌ Vulnerabilities found in Go modules"
    exit 1
fi

# Scan Docker images if Dockerfile exists
if [[ -f "Dockerfile" ]]; then
    echo "Scanning Docker image..."
    if command -v trivy >/dev/null 2>&1; then
        docker build -t grctool:security-scan .
        trivy image --severity HIGH,CRITICAL grctool:security-scan
    else
        echo "⚠️  Trivy not found, skipping Docker scan"
    fi
fi

# Check for outdated dependencies
echo "Checking for outdated dependencies..."
go list -u -m all | grep -v "indirect" | while read -r line; do
    if echo "$line" | grep -q "=>"; then
        echo "⚠️  Replace directive found: $line"
    fi
done

echo "✅ Vulnerability scan completed"
```

### Security Audit Process
```bash
#!/bin/bash
# scripts/security-audit.sh

set -e

REPORT_FILE="security-audit-$(date +%Y%m%d).txt"

echo "Running comprehensive security audit..."
echo "Report will be saved to: $REPORT_FILE"

{
    echo "GRCTool Security Audit Report"
    echo "Generated: $(date)"
    echo "Version: $(git describe --tags --always)"
    echo "=================================="
    echo ""

    echo "1. Dependency Vulnerabilities:"
    echo "------------------------------"
    govulncheck ./... || echo "Vulnerabilities found - review required"
    echo ""

    echo "2. Code Security Issues:"
    echo "------------------------"
    gosec -fmt json ./... | jq -r '.Issues[] | "\(.severity): \(.details) (\(.file):\(.line))"' || echo "No security issues found"
    echo ""

    echo "3. Hardcoded Secrets Check:"
    echo "---------------------------"
    ./scripts/detect-secrets.sh || echo "Potential secrets found - review required"
    echo ""

    echo "4. File Permissions Audit:"
    echo "---------------------------"
    find . -type f -perm /o+w | head -10
    echo ""

    echo "5. Configuration Security:"
    echo "-------------------------"
    grep -r "http://" configs/ || echo "No insecure HTTP URLs found"
    grep -r "password" configs/ | grep -v "example" || echo "No hardcoded passwords found"
    echo ""

    echo "6. Network Security:"
    echo "-------------------"
    grep -r "localhost" . --include="*.go" | grep -v "_test.go" | head -5 || echo "No localhost references in production code"
    echo ""

    echo "7. Authentication Security:"
    echo "--------------------------"
    grep -r "token" . --include="*.go" | grep -E "(log|print)" | head -5 || echo "No token logging found"

} > "$REPORT_FILE"

echo "✅ Security audit completed"
echo "Report saved to: $REPORT_FILE"

# Check if any critical issues were found
if grep -q "Vulnerabilities found\|Potential secrets found" "$REPORT_FILE"; then
    echo "⚠️  Critical security issues found - review report"
    exit 1
fi
```

## Secure Development Lifecycle

### Security Review Checklist

#### Code Review Security Checklist
- [ ] No hardcoded secrets, passwords, or API keys
- [ ] All user inputs are validated and sanitized
- [ ] Error messages don't expose sensitive information
- [ ] Authentication and authorization are properly implemented
- [ ] Cryptographic operations use secure algorithms
- [ ] File operations use safe paths (no directory traversal)
- [ ] Command execution is properly sanitized
- [ ] Logging doesn't expose sensitive data
- [ ] Network communications use secure protocols
- [ ] Dependencies are up-to-date and vulnerability-free

#### Deployment Security Checklist
- [ ] Secrets are managed through secure secret stores
- [ ] Services run with minimal privileges
- [ ] Network access is restricted to necessary ports
- [ ] SSL/TLS is properly configured
- [ ] Security headers are implemented
- [ ] Audit logging is enabled
- [ ] Backups are encrypted
- [ ] Monitoring and alerting are configured
- [ ] Incident response procedures are documented
- [ ] Security updates are automatically applied

## Incident Response

### Security Incident Procedure
```bash
#!/bin/bash
# scripts/security-incident-response.sh

INCIDENT_TYPE=${1}
SEVERITY=${2:-medium}

echo "SECURITY INCIDENT RESPONSE INITIATED"
echo "Type: $INCIDENT_TYPE"
echo "Severity: $SEVERITY"
echo "Time: $(date)"

case $INCIDENT_TYPE in
    "credential-compromise")
        echo "Credential compromise detected:"
        echo "1. Immediately rotate all affected credentials"
        echo "2. Audit access logs for unauthorized activity"
        echo "3. Update all systems using compromised credentials"
        echo "4. Document incident and remediation steps"
        ;;
    "vulnerability-exploit")
        echo "Vulnerability exploitation detected:"
        echo "1. Isolate affected systems"
        echo "2. Apply security patches immediately"
        echo "3. Review logs for signs of compromise"
        echo "4. Notify stakeholders"
        ;;
    "data-breach")
        echo "Data breach detected:"
        echo "1. Contain the breach"
        echo "2. Assess scope of data exposure"
        echo "3. Notify legal and compliance teams"
        echo "4. Prepare breach notification"
        ;;
esac

# Generate incident report template
cat > "incident-report-$(date +%Y%m%d-%H%M%S).md" << EOF
# Security Incident Report

**Incident ID:** INC-$(date +%Y%m%d-%H%M%S)
**Date:** $(date)
**Type:** $INCIDENT_TYPE
**Severity:** $SEVERITY

## Summary


## Timeline


## Impact Assessment


## Root Cause Analysis


## Remediation Actions


## Lessons Learned


## Follow-up Actions

EOF

echo "Incident report template created"
```

## Security Guidelines for Developers

### Security Best Practices
1. **Never commit secrets**: Use environment variables or secret stores
2. **Validate all inputs**: Assume all input is malicious
3. **Use secure defaults**: Fail secure, not fail open
4. **Keep dependencies updated**: Regularly update and scan dependencies
5. **Follow least privilege**: Grant minimum necessary permissions
6. **Use secure communication**: Always use HTTPS/TLS
7. **Log security events**: But never log sensitive data
8. **Handle errors securely**: Don't expose system information
9. **Test security controls**: Include security tests in test suites
10. **Stay informed**: Keep up with security best practices

## References

- [[system-architecture]] - Overall system design and security integration
- [[api-design]] - Secure API design patterns
- [[security-requirements]] - Security requirements from Frame phase
- [[testing-strategy]] - Security testing approaches

---

*This security architecture ensures GRCTool meets the highest standards for handling sensitive compliance data while maintaining usability and performance.*