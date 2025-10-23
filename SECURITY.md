# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

As GRCTool is currently in active development (pre-1.0), we support only the latest release version. Once we reach version 1.0, we will maintain security updates for the current major version and the previous major version.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

We take the security of GRCTool seriously. If you believe you have found a security vulnerability, please report it to us as described below.

### How to Report

**GitHub Security Advisory**: For critical vulnerabilities, please use [GitHub's private security reporting](https://github.com/grctool/grctool/security/advisories/new)

**GitHub Issues**: For non-critical security concerns, you may open an issue at https://github.com/grctool/grctool/issues with the "security" label

Please include the following information in your report:

1. **Description** of the vulnerability
2. **Steps to reproduce** the issue
3. **Potential impact** of the vulnerability
4. **Suggested fix** (if you have one)
5. **Your contact information** for follow-up questions

### What to Expect

When you report a vulnerability, we will:

1. **Acknowledge receipt** within 48 hours (2 business days)
2. **Provide an initial assessment** within 5 business days
3. **Keep you informed** of our progress
4. **Credit you** in the security advisory (unless you prefer to remain anonymous)

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 5 business days
- **Fix Timeline**: Depends on severity
  - Critical: 7-14 days
  - High: 14-30 days
  - Medium: 30-90 days
  - Low: 90+ days or next release

### Disclosure Policy

We follow responsible disclosure practices:

1. **Private disclosure** to maintainers first
2. **Coordinated disclosure** after a fix is available
3. **Public disclosure** in release notes and security advisories
4. **CVE assignment** for significant vulnerabilities

We request that you:
- Give us reasonable time to address the vulnerability before public disclosure
- Make a good faith effort to avoid privacy violations, data destruction, and service disruption
- Do not exploit the vulnerability beyond what is necessary to demonstrate it

## Security Best Practices

### For Users

When using GRCTool, follow these security best practices:

#### Credential Management

- **Never commit** `.grctool.yaml` with real credentials to version control
- Use **environment variables** for sensitive data:
  ```bash
  export CLAUDE_API_KEY="your-key"
  export TUGBOAT_ORG_ID="your-org-id"
  ```
- Store authentication cookies securely (automatically handled by the tool)
- **Rotate API keys** regularly (at least quarterly)
- Use **separate credentials** for development and production

#### File Permissions

Ensure sensitive files have appropriate permissions:
```bash
chmod 600 .grctool.yaml          # Config file
chmod 700 ~/.grctool/auth/       # Auth directory
```

#### Network Security

- Use **HTTPS only** for API connections (enforced by default)
- Verify SSL certificates (default behavior)
- Be cautious with proxy settings
- Review API rate limiting to prevent account lockout

#### Evidence Handling

- **Review generated evidence** before submitting to Tugboat Logic
- Be aware that evidence may contain **sensitive infrastructure information**
- Use `.gitignore` to exclude evidence files from version control
- Implement **access controls** on evidence directories

### For Contributors

#### Secure Development

- **Never commit** secrets, tokens, or credentials
- Use **VCR cassettes** for integration tests (scrub sensitive data)
- Run `make security-scan` before submitting PRs
- Follow secure coding practices in `.golangci.yml`

#### Dependencies

- Keep dependencies up to date
- Review dependency security advisories
- Use `go mod verify` to check module integrity
- Audit new dependencies before adding them

#### Code Review

Security-sensitive areas requiring extra review:
- Authentication mechanisms (`internal/auth/`)
- API client code (`internal/tugboat/`)
- File system operations (`internal/storage/`)
- Credential handling (`internal/config/`)
- External command execution (if any)

## Known Security Considerations

### macOS Safari Authentication

The browser-based authentication feature:
- Uses Safari's cookie storage via AppleScript
- Requires user consent and accessibility permissions
- Stores cookies in encrypted format
- Only supports macOS (by design)

**Risks**:
- Requires granting terminal accessibility permissions
- Cookies are extracted from Safari's local storage
- Cookies are stored in plaintext in `~/.grctool/auth/`

**Mitigations**:
- User must explicitly run `auth login`
- Permissions can be revoked at any time
- Cookies expire according to Tugboat Logic's session policy
- Use `auth logout` to clear stored credentials

### API Key Storage

Claude API keys and Tugboat credentials:
- Stored in `.grctool.yaml` (YAML format)
- Can be provided via environment variables (recommended)
- Not encrypted at rest (rely on OS file permissions)

**Recommendations**:
- Use environment variables for production
- Set restrictive file permissions (600)
- Never commit configuration files with real keys
- Rotate keys regularly

### Evidence Data

Generated evidence may contain:
- Infrastructure configuration details
- Access control information
- Repository and file names
- User and group information
- Security control implementations

**Recommendations**:
- Review evidence before submission
- Use `.gitignore` for evidence directories
- Implement access controls on data directories
- Consider evidence data sensitivity in your security classification

## Security Scanning

### Automated Scans

We run the following security scans:

- **gosec**: Go security checker (part of `make lint`)
- **golangci-lint**: Multiple security-focused linters
- **Dependency scanning**: Via GitHub Dependabot
- **Static analysis**: Via golangci-lint rules

Run security scans locally:
```bash
make lint              # Includes gosec
make security-scan     # Dedicated security checks
make vulnerability-check  # Check for known vulnerabilities
```

### Vulnerability Management

We monitor:
- GitHub Security Advisories
- Go vulnerability database
- Dependency vulnerability reports
- Community security disclosures

## Compliance Considerations

GRCTool is designed for security compliance:

- **Audit trails**: All operations are logged
- **Data integrity**: Evidence includes timestamps and hashes
- **Access control**: Relies on Tugboat Logic's authentication
- **Encryption in transit**: HTTPS for all API calls
- **Data retention**: Users control local data storage

## Security Updates

Security updates are released as:
1. **Patch releases** for minor issues
2. **Hotfix releases** for critical vulnerabilities
3. **Security advisories** via GitHub Security Advisories

Subscribe to releases to be notified of security updates:
- Watch the repository on GitHub
- Subscribe to the releases RSS feed
- Follow security advisories

## Questions

If you have questions about this security policy, contact:

- **Security issues**: Use [GitHub Security Advisories](https://github.com/grctool/grctool/security/advisories/new) for private disclosure
- **General questions**: Open an issue at https://github.com/grctool/grctool/issues
- **Public discussion**: GitHub Discussions

## Acknowledgments

We appreciate responsible security researchers who help keep GRCTool and its users safe. Security researchers who report valid vulnerabilities will be acknowledged in:

- Security advisories (with permission)
- CHANGELOG.md
- Project documentation

Thank you for helping keep GRCTool secure!
