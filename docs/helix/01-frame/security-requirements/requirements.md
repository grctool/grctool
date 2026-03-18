---
title: "Security Requirements"
phase: "01-frame"
category: "security"
tags: ["security", "requirements", "authentication", "authorization", "data-protection", "compliance"]
related: ["product-requirements", "compliance-requirements", "threat-model", "security-architecture"]
created: 2025-01-10
updated: 2026-03-17
helix_mapping: "Frame phase security requirements - drives Design phase security architecture"
---

# Security Requirements

**Project**: GRCTool
**Version**: 1.0
**Date**: 2026-03-17
**Owner**: Engineering Team
**Security Champion**: Security Engineering Lead

---

## Overview

This document defines the security requirements for GRCTool, an agentic compliance evidence collection CLI tool. Because GRCTool handles sensitive compliance data, authentication credentials, and integrates with third-party APIs (Tugboat Logic, GitHub, Google Workspace, Claude AI), security is a foundational concern. These requirements inform all design decisions, testing strategies, and operational procedures throughout the development lifecycle.

## Security User Stories

### Authentication and Identity Management

**SEC-001: Browser-Based Tugboat Authentication**
- **As a** compliance manager
- **I want** secure browser-based authentication with Tugboat Logic via Safari
- **So that** I can authenticate using native macOS capabilities (Touch ID, Face ID, Keychain) without storing long-lived API keys
- **Acceptance Criteria**:
  - [ ] Authentication uses Safari AppleScript integration to extract session cookies
  - [ ] Bearer tokens are extracted from base64-encoded JSON cookie data
  - [ ] Authentication flow validates that the browser is on the correct domain (tugboatlogic.com) before extracting cookies
  - [ ] Authentication times out after a configurable period (default: 5 minutes)
  - [ ] Retry logic allows up to 3 attempts for token extraction
  - [ ] Clear error messages guide users through Safari Developer settings when JavaScript automation is not enabled

**SEC-002: GitHub Authentication via CLI**
- **As a** security engineer
- **I want** GitHub authentication that leverages the `gh` CLI token when available
- **So that** I do not need to manually manage or store GitHub personal access tokens
- **Acceptance Criteria**:
  - [ ] GitHub token is automatically retrieved from `gh auth token` when no token is configured
  - [ ] Token format is validated against known GitHub token prefixes (ghp_, ghs_, gho_, ghu_, ghr_)
  - [ ] Token is populated in both `auth.github.token` and `evidence.tools.github.api_token` for backward compatibility
  - [ ] Fallback to environment variable (`GITHUB_TOKEN`) when `gh` CLI is unavailable
  - [ ] Invalid or expired tokens produce actionable error messages

**SEC-003: Token Lifecycle Management**
- **As a** system administrator
- **I want** authentication tokens to have defined lifetimes and expiration handling
- **So that** stale credentials do not persist indefinitely
- **Acceptance Criteria**:
  - [ ] Tugboat authentication credentials default to 30-minute expiration
  - [ ] `AuthStatus` tracks `LastValidated` and `ExpiresAt` timestamps
  - [ ] Expired credentials trigger re-authentication prompts rather than silent failures
  - [ ] `grctool auth status` displays current authentication state, token presence, and validity
  - [ ] `grctool auth logout` clears all cached authentication data

**SEC-004: Authentication Provider Interface**
- **As a** developer extending GRCTool
- **I want** a consistent authentication provider interface
- **So that** new integrations follow the same security patterns
- **Acceptance Criteria**:
  - [ ] All authentication providers implement the `AuthProvider` interface (Name, IsAuthRequired, GetStatus, Authenticate, ValidateAuth, ClearAuth)
  - [ ] Tools that do not require authentication use `NoAuthProvider` with explicit `dataSource` labeling
  - [ ] Authentication state is queryable without triggering re-authentication
  - [ ] Context-based cancellation is supported for all authentication operations

### Authorization and Access Control

**SEC-005: Least Privilege API Access**
- **As a** security engineer
- **I want** API integrations to request only the minimum required permissions
- **So that** a compromised token limits the blast radius
- **Acceptance Criteria**:
  - [ ] GitHub token permissions are validated and scoped to read-only repository access where possible
  - [ ] Tugboat API access is scoped to evidence collection and submission operations
  - [ ] Google Workspace credentials are scoped to read-only Drive access
  - [ ] Tool execution is constrained to an allowlist of safe commands (git, terraform, kubectl)

**SEC-006: Command Execution Allowlisting**
- **As a** system
- **I must** restrict which external commands can be executed
- **So that** command injection attacks cannot execute arbitrary system commands
- **Acceptance Criteria**:
  - [ ] Only allowlisted commands can be executed via the `grctool-run` tool
  - [ ] All command arguments are validated against a character allowlist (`[a-zA-Z0-9\-_./]`)
  - [ ] Dangerous patterns (`;`, `&&`, `||`, `|`, backticks, `$(`, `../`) are rejected
  - [ ] Command execution uses `context.WithTimeout` with a 30-second default

**SEC-007: Configuration Access Control**
- **As a** system
- **I must** validate configuration structure and reject unknown keys
- **So that** configuration injection or misconfiguration is detected early
- **Acceptance Criteria**:
  - [ ] Unknown top-level configuration keys produce warnings on stderr
  - [ ] Misplaced configuration sections (e.g., `evidence.terraform` instead of `evidence.tools.terraform`) produce corrective guidance
  - [ ] Authentication mode is validated to known values (`form`, `browser`)
  - [ ] File paths in configuration are resolved relative to the config file directory

### Data Protection

**SEC-008: Credential Storage Security**
- **As a** system
- **I must** protect stored credentials from unauthorized access
- **So that** authentication secrets are not exposed through configuration files or logs
- **Acceptance Criteria**:
  - [ ] API keys are stored via environment variables (`CLAUDE_API_KEY`, `GITHUB_TOKEN`, `TUGBOAT_API_KEY`), never in configuration files
  - [ ] Configuration files use `${ENV_VAR}` placeholder syntax for secrets
  - [ ] The `auth.cache_dir` stores cached authentication data with restricted file permissions
  - [ ] The Tugboat API password supports environment variable substitution
  - [ ] Collector URLs are stored in `.grctool.yaml` and excluded from version control via `.gitignore`

**SEC-009: Evidence Data Protection**
- **As a** compliance manager
- **I want** evidence data to be stored securely on the local filesystem
- **So that** sensitive compliance information is protected from unauthorized access
- **Acceptance Criteria**:
  - [ ] Evidence output files are written to configured directories with appropriate filesystem permissions
  - [ ] Evidence file paths are validated to prevent directory traversal
  - [ ] Cache directories (`.cache/`) can be safely deleted without losing primary data
  - [ ] Evidence data includes metadata (timestamps, sources) for audit traceability

**SEC-010: Transit Encryption**
- **As a** system
- **I must** encrypt all data in transit
- **So that** sensitive data cannot be intercepted during API communication
- **Acceptance Criteria**:
  - [ ] All Tugboat Logic API communication uses HTTPS
  - [ ] All GitHub API communication uses HTTPS
  - [ ] All Google Workspace API communication uses HTTPS with OAuth2
  - [ ] All Claude AI API communication uses HTTPS
  - [ ] HTTP client configurations enforce TLS and reject insecure connections

### Input Validation and Injection Prevention

**SEC-011: CLI Input Validation**
- **As a** system
- **I must** validate all CLI inputs before processing
- **So that** malformed or malicious input cannot compromise system integrity
- **Acceptance Criteria**:
  - [ ] Evidence task IDs are validated against the pattern `ET-\d{4}`
  - [ ] Task names are validated for length (1-255 characters) and allowed characters (`[a-zA-Z0-9\s\-_.,]`)
  - [ ] Task status values are validated against an enumerated set (`pending`, `in_progress`, `completed`, `failed`)
  - [ ] File paths are sanitized to prevent path traversal attacks
  - [ ] URL inputs are validated for scheme and domain

**SEC-012: Template Injection Prevention**
- **As a** system
- **I must** safely process template variables in policy and control documents
- **So that** template processing cannot be exploited for code execution
- **Acceptance Criteria**:
  - [ ] Template interpolation is limited to `{{variable.name}}` syntax with a known variable registry
  - [ ] Circular references in template variables are detected and rejected (DFS-based detection implemented)
  - [ ] Template values are HTML-escaped when generating output for web contexts
  - [ ] Only configured variables in `interpolation.variables` are substituted

**SEC-013: API Response Validation**
- **As a** system
- **I must** validate data received from external APIs
- **So that** malicious or malformed API responses do not compromise the application
- **Acceptance Criteria**:
  - [ ] JSON responses are unmarshalled into typed structs, not arbitrary maps
  - [ ] Response sizes are bounded to prevent memory exhaustion
  - [ ] API error responses are handled without exposing internal system details
  - [ ] Timeout and retry logic prevents hanging on unresponsive APIs

### Audit and Logging

**SEC-014: Structured Security Logging**
- **As a** security analyst
- **I want** comprehensive structured logging of security-relevant events
- **So that** security incidents can be detected and investigated
- **Acceptance Criteria**:
  - [ ] All authentication events (login, logout, token validation, failures) are logged
  - [ ] API request/response logging is configurable via `log_api_requests` and `log_api_responses`
  - [ ] Log output supports both text and JSON formats
  - [ ] Console logger defaults to `warn` level; file logger defaults to `info` level
  - [ ] Caller information is included in file logs (`show_caller: true`)

**SEC-015: Credential Redaction in Logs**
- **As a** system
- **I must** never log sensitive credential data
- **So that** credentials cannot be leaked through log files or console output
- **Implementation Note**: `[NOT YET IMPLEMENTED]` The `RedactionHook` in `internal/logger/zerolog_logger.go` is a stub -- the `Run` method is a no-op. Log-level field redaction is not yet active; redaction currently only occurs at the tool output layer (`RedactSensitiveData` in `internal/tools/output.go`). The acceptance criteria below describe the target state.
- **Acceptance Criteria**:
  - [ ] Default redaction fields include: `password`, `token`, `key`, `secret`, `api_key`, `cookie`
  - [ ] URL sanitization is enabled by default (`sanitize_urls: true`) to strip query parameters containing secrets
  - [ ] Log buffer size is configurable (default: 100 entries) with configurable flush intervals (default: 5s)
  - [ ] Bearer tokens are never included in log output
  - [ ] Error messages from authentication failures do not include token values

**SEC-016: Evidence Audit Trail**
- **As an** auditor
- **I want** complete traceability of evidence generation activities
- **So that** I can verify evidence integrity and the collection process
- **Acceptance Criteria**:
  - [ ] Evidence files include generation timestamps and source attribution
  - [ ] Evidence file naming follows the convention `ET-XXXX-{tugboat_id}-{description}.json`
  - [ ] Configuration changes are detectable through version-controlled `.grctool.yaml`
  - [ ] Sync operations produce structured output documenting what was downloaded

### Dependency and Supply Chain Security

**SEC-017: Go Module Vulnerability Scanning**
- **As a** developer
- **I want** automated dependency vulnerability scanning
- **So that** known vulnerabilities in third-party libraries are detected before deployment
- **Acceptance Criteria**:
  - [ ] `govulncheck ./...` runs as part of the CI/CD pipeline
  - [ ] All direct dependencies are pinned to specific versions in `go.mod`
  - [ ] Go toolchain version is pinned (`go 1.24.0`, `toolchain go1.24.12`)
  - [ ] Indirect dependencies are tracked in `go.sum` for integrity verification
  - [ ] Dependency updates are reviewed for security implications

**SEC-018: Secret Detection in Source Code**
- **As a** developer
- **I want** pre-commit secret detection
- **So that** credentials are never accidentally committed to version control
- **Acceptance Criteria**:
  - [ ] `scripts/detect-secrets.sh` scans staged files for API keys, tokens, private keys, passwords, and connection strings
  - [ ] Patterns cover AWS (AKIA), GitHub (ghp_, ghs_, gho_, ghu_, ghr_), Google Cloud (AIza), Slack (xox), Stripe (sk_/pk_), and generic secrets
  - [ ] Allowed patterns exclude test data, placeholders (`${...}`, `{{...}}`), and example credentials
  - [ ] Binary files and specified extensions (.md, .sum, .mod, .lock, images) are skipped
  - [ ] Vendor, node_modules, .git, and testdata directories are excluded
  - [ ] Exit code 1 blocks commits when potential secrets are found

### Compliance Alignment

**SEC-019: SOC 2 Trust Services Alignment**
- **As a** compliance manager
- **I want** GRCTool itself to meet SOC 2 Trust Services Criteria
- **So that** the tool used for evidence collection does not become a compliance liability
- **Acceptance Criteria**:
  - [ ] **CC6.1 (Access Control)**: Authentication and authorization controls are implemented for all API integrations
  - [ ] **CC6.6 (System Boundaries)**: Trust boundaries are defined between CLI, APIs, and local filesystem
  - [ ] **CC6.8 (Logical Access)**: Token-based authentication restricts access to authorized users
  - [ ] **CC7.1 (Monitoring)**: Structured logging captures security-relevant events
  - [ ] **CC8.1 (Change Management)**: Version-controlled configuration and dependency management

**SEC-020: ISO 27001 Control Alignment**
- **As a** compliance manager
- **I want** GRCTool security controls to map to ISO 27001 Annex A
- **So that** the tool supports organizations pursuing ISO certification
- **Acceptance Criteria**:
  - [ ] **A.9 (Access Control)**: Role-based and token-based access controls implemented
  - [ ] **A.10 (Cryptography)**: TLS for all API communication; environment variable encryption for secrets
  - [ ] **A.12 (Operations Security)**: Logging, monitoring, and vulnerability management processes defined
  - [ ] **A.14 (System Development)**: Secure development lifecycle with secret detection and code review
  - [ ] **A.18 (Compliance)**: Compliance requirements documented and mapped to controls

### System-of-Record Security Requirements

**SEC-SOR-001: Master Data Access Control**
- **Status**: Planned
- **As a** compliance manager
- **I want** only authorized processes and users to modify canonical compliance data in the master index
- **So that** the integrity of the system of record is protected from unauthorized or accidental changes
- **Acceptance Criteria**:
  - [ ] Write access to the master index is restricted to authenticated and authorized GRCTool processes
  - [ ] Direct filesystem modifications to master index files are detectable via integrity checks (checksums or git-based change detection)
  - [ ] Integration adapters require explicit authorization before writing to the master index
  - [ ] Read-only access modes are available for reporting and audit queries
  - [ ] Access control policies are configurable per artifact type (policies, controls, evidence tasks)

**SEC-SOR-002: Bidirectional Sync Integrity**
- **Status**: Planned
- **As a** security engineer
- **I want** conflict detection and resolution for data flowing between GRCTool and external systems
- **So that** bidirectional sync does not silently overwrite or corrupt compliance data
- **Acceptance Criteria**:
  - [ ] Conflicts between local master index and remote platform state are detected before any write operation
  - [ ] Conflict resolution strategy is configurable per integration target (local_wins, remote_wins, manual, newest_wins)
  - [ ] All conflict resolutions are logged with the chosen strategy, affected artifacts, and resulting state
  - [ ] Manual conflict resolution presents a clear diff of local vs. remote state to the user
  - [ ] Sync operations are atomic at the artifact level -- partial sync failures do not leave the master index in an inconsistent state

**SEC-SOR-003: Integration Adapter Authentication**
- **Status**: Planned
- **As a** system
- **I must** require each integration adapter to authenticate independently
- **So that** a compromised adapter cannot affect other integrations or the master index beyond its scope
- **Acceptance Criteria**:
  - [ ] Each integration adapter (Tugboat, GitHub, Google Workspace, future platforms) maintains its own authentication credentials and session state
  - [ ] Adapter credentials are scoped to the minimum permissions required for that specific integration
  - [ ] Compromise of one adapter's credentials does not grant access to other adapters or the master index directly
  - [ ] Adapter authentication failures are isolated -- one adapter's auth failure does not block other adapters from operating
  - [ ] Adapter credential rotation can be performed independently without affecting other integrations

**SEC-SOR-004: Audit Trail for Master Data Changes**
- **Status**: Planned
- **As an** auditor
- **I want** every modification to canonical compliance data logged with actor, timestamp, and change details
- **So that** the full history of the system of record is traceable and verifiable
- **Acceptance Criteria**:
  - [ ] Every create, update, and delete operation on master index artifacts produces an audit log entry
  - [ ] Audit log entries include: actor identity (user or adapter), timestamp (UTC), operation type, artifact identifier, and a summary of changes (before/after values or diff)
  - [ ] Audit logs are stored in a tamper-evident format (append-only log or git-backed history)
  - [ ] Audit logs are queryable by artifact, actor, time range, and operation type
  - [ ] Audit trail data is retained for a configurable period (default: 7 years, aligned with SOC 2 retention requirements)

**SEC-SOR-005: Data Sovereignty Controls**
- **Status**: Planned
- **As a** compliance manager
- **I want** configurable data residency and export restrictions for compliance data
- **So that** the organization maintains control over where compliance data is stored and transmitted
- **Acceptance Criteria**:
  - [ ] Organizations can configure which compliance artifacts are eligible for sync to external platforms
  - [ ] Export restrictions can be applied per artifact type, sensitivity classification, or integration target
  - [ ] Data residency configuration specifies allowed storage locations (local-only, specific cloud regions)
  - [ ] Sync operations respect export restrictions and reject attempts to transmit restricted data
  - [ ] Data sovereignty configuration is auditable and version-controlled

**SEC-SOR-006: Backup and Recovery**
- **Status**: Planned
- **As a** system administrator
- **I want** the master index to support point-in-time recovery
- **So that** compliance data can be restored to a known-good state after data corruption, accidental deletion, or security incidents
- **Acceptance Criteria**:
  - [ ] Master index supports point-in-time recovery via git-based snapshots or dedicated backup mechanism
  - [ ] Backup frequency is configurable (default: every sync operation creates a recoverable checkpoint)
  - [ ] Recovery procedures are documented and tested, with a target recovery time objective (RTO) of under 1 hour
  - [ ] Recovery does not require re-syncing from external platforms (local backups are self-contained)
  - [ ] Backup integrity is verifiable via checksums or signature validation

## Security Risk Assessment

### High-Risk Areas

1. **Credential Exposure**: Authentication tokens for Tugboat, GitHub, Google, and Claude AI could be exposed through misconfigured logging, insecure storage, or source code commits. Mitigated by environment variable storage, log redaction, and pre-commit secret detection.

2. **API Integration Compromise**: Compromised API tokens could allow unauthorized access to compliance data or infrastructure information. Mitigated by least-privilege scoping, token expiration, and validation.

3. **Command Injection**: CLI inputs processed as shell commands could allow arbitrary code execution. Mitigated by input validation, command allowlisting, and argument sanitization.

4. **Supply Chain Attack**: Compromised Go dependencies could introduce vulnerabilities. Mitigated by version pinning, `go.sum` integrity checks, and `govulncheck` scanning.

### Risk Tolerance Levels

- **Critical**: Zero tolerance -- must be addressed before any release. Includes credential exposure in logs, command injection, and authentication bypass.
- **High**: Low tolerance -- must have mitigation plan before release. Includes API token compromise and supply chain vulnerabilities.
- **Medium**: Moderate tolerance -- mitigation plan required within 30 days. Includes missing audit trail entries and configuration validation gaps.
- **Low**: Higher tolerance -- mitigation plan required within 90 days. Includes cross-platform authentication limitations and optional feature security hardening.

## Security Architecture Requirements

### Application Security
- [ ] Secure development lifecycle followed with pre-commit hooks
- [ ] Static analysis via `gosec` for Go security issues
- [ ] Dependency vulnerability scanning via `govulncheck`
- [ ] Secret detection via `scripts/detect-secrets.sh`
- [ ] Input validation on all user-facing interfaces

### Infrastructure Security
- [ ] Configuration files validated on load with unknown key detection
- [ ] File permissions enforced on credential cache directories
- [ ] Path traversal prevention on all file operations
- [ ] Environment variable substitution for all secrets

## Security Testing Requirements

### Testing Types Required
- [ ] Secret detection scanning (pre-commit)
- [ ] Dependency vulnerability assessment (CI/CD)
- [ ] Authentication flow testing (unit and integration)
- [ ] Input validation testing (unit tests with malicious inputs)
- [ ] Command injection testing (unit tests)
- [ ] Template injection testing (unit tests)
- [ ] Configuration validation testing (unit tests)

### Testing Frequency
- **Pre-commit**: Secret detection on every commit
- **Per-build**: `govulncheck` and `gosec` in CI/CD
- **Quarterly**: Dependency audit and update review
- **Annually**: Comprehensive security architecture review

## Non-Functional Security Requirements

### Performance Requirements
- [ ] Authentication flow completes within 5 minutes (user interaction time)
- [ ] Token validation completes within 2 seconds
- [ ] Log redaction adds less than 1% overhead to logging operations
- [ ] Secret detection script completes within 10 seconds for typical changesets

### Availability Requirements
- [ ] Offline operation supported for local-only tools (NoAuthProvider)
- [ ] Authentication failures produce clear recovery guidance
- [ ] Cached credentials allow operation without re-authentication within expiration window

## Assumptions and Dependencies

### Assumptions
- macOS is the primary deployment platform (Safari authentication is macOS-only)
- Users have the `gh` CLI installed and authenticated for GitHub integration
- Environment variables are secured by the host operating system
- The local filesystem provides adequate access control for evidence data

### Dependencies
- Safari browser with "Allow JavaScript from Apple Events" enabled for Tugboat authentication
- `gh` CLI for automated GitHub token retrieval
- Go toolchain (1.24.x) for build and vulnerability scanning
- `govulncheck` and `gosec` for security analysis

### Constraints
- Browser-based authentication limits adoption to macOS users (cross-platform auth planned for future release)
- CLI-only interface means no web session management concerns
- Local-first architecture means data protection relies on filesystem permissions rather than application-level encryption at rest

## Phase Gate Requirements

To proceed to the Design phase, the following must be complete:
- [x] All security requirements reviewed and documented
- [x] Compliance requirements mapped to security controls
- [x] Security risk assessment completed
- [x] Authentication architecture defined
- [x] Secret management approach established
- [x] Dependency security process defined

---

**Document Control**
- **Version**: 1.0
- **Last Updated**: 2026-03-17
- **Next Review Date**: 2026-06-17
- **Change History**: Initial creation based on codebase analysis and template alignment
