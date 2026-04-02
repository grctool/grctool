---
title: "FEAT-006: Browser-Based Authentication"
phase: "01-frame"
category: "features"
tags: ["authentication", "safari", "tugboat", "browser", "credentials", "backfill"]
related: ["FEAT-005", "FEAT-007", "prd"]
status: "Implemented"
priority: "P0"
created: 2026-04-01
updated: 2026-04-01
backfill: true
---

# FEAT-006: Browser-Based Authentication

## Status

| Field | Value |
|-------|-------|
| Status | Implemented |
| Priority | P0 |
| Implementation | `internal/auth/`, `cmd/auth.go` |

---

## Overview

GRCTool authenticates with Tugboat Logic using browser-based authentication on macOS. Rather than requiring users to manage API keys or OAuth flows manually, it automates Safari to capture authentication cookies after the user completes their normal login flow — supporting Touch ID, Face ID, 1Password, and any other browser-native authentication method.

## Problem Statement

Tugboat Logic does not provide a public API key or OAuth client credentials flow. Users authenticate via a web browser with session cookies. Compliance engineers need a way to authenticate GRCTool against the Tugboat API without manually extracting cookies, while supporting their existing authentication methods (SSO, MFA, password managers). The authentication mechanism must store credentials securely, detect expiration, and provide clear status feedback.

---

## Requirements

### Functional Requirements

- FR-001: Open Safari to the Tugboat Logic login page on `grctool auth login`
- FR-002: Wait for user to complete authentication (with configurable timeout, default 5 minutes)
- FR-003: Extract authentication cookies from Safari via JavaScript execution
- FR-004: Extract bearer token from authentication cookies
- FR-005: Validate extracted credentials via API test call before persisting
- FR-006: Detect organization ID from authenticated API response
- FR-007: Store credentials (cookie header, bearer token, expiration, org ID) in `.grctool.yaml`
- FR-008: Display authentication status via `grctool auth status`
- FR-009: Clear stored credentials via `grctool auth logout`
- FR-010: Support manual cookie extraction as fallback when automation fails

### Non-Functional Requirements

- **Performance**: Authentication flow completes within the user's login time + 5 seconds of extraction
- **Security**: Credentials stored in `.grctool.yaml` with restricted file permissions; never logged or displayed in plaintext; redacted in `config show` output
- **Reliability**: Credential validation before storage prevents saving expired or invalid tokens
- **Usability**: Single command (`grctool auth login`) handles the entire flow; status messages guide the user through browser interaction

---

## User Stories

### US-061: Authenticate with Tugboat Logic [FEAT-006]

**As a** compliance engineer,
**I want to** authenticate GRCTool with Tugboat Logic via my browser
**so that** I can use my existing SSO/MFA/password manager without managing API keys.

**Acceptance Criteria:**

- [x] `grctool auth login` opens Safari to the Tugboat Logic login page
- [x] User completes authentication using any browser-supported method
- [x] GRCTool automatically captures cookies and extracts bearer token
- [x] Credentials are validated via an API call before being stored
- [x] Success message displays authenticated username and organization

### US-062: Check Authentication Status [FEAT-006]

**As a** compliance engineer,
**I want to** verify my authentication status before running sync or evidence commands
**so that** I know whether I need to re-authenticate.

**Acceptance Criteria:**

- [x] `grctool auth status` shows current authentication state (authenticated/expired/not authenticated)
- [x] Displays organization ID and username when authenticated
- [x] Shows credential expiration time when available
- [x] Provides actionable guidance when not authenticated

### US-063: Logout and Clear Credentials [FEAT-006]

**As a** security-conscious user,
**I want to** clear stored credentials when I'm done
**so that** my authentication tokens are not persisted unnecessarily.

**Acceptance Criteria:**

- [x] `grctool auth logout` removes cookie header, bearer token, and expiration from config
- [x] Confirms successful logout
- [x] Subsequent commands requiring auth fail with a clear re-authentication message

### US-064: GitHub Authentication Fallback [FEAT-006]

**As a** DevOps engineer,
**I want** GitHub tools to work with my existing `gh` CLI authentication
**so that** I don't need to configure a separate API token.

**Acceptance Criteria:**

- [x] GitHub tools check for API token in config first
- [x] Fall back to `gh` CLI authentication if no token configured
- [x] Fall back to `GITHUB_TOKEN` environment variable as last resort
- [x] Zero-configuration GitHub access when `gh` is already authenticated

---

## Edge Cases and Error Handling

- Safari not available (non-macOS): Clear error message suggesting manual cookie extraction
- User closes browser before completing login: Timeout after configurable duration with retry instructions
- Tugboat session expires during long operation: API calls fail with clear re-authentication message
- Invalid or expired cookies captured: Validation step rejects credentials before storage
- Organization ID cannot be detected: Prompt user to configure `tugboat.org_id` manually

---

## Success Metrics

| Metric | Target | Status |
|--------|--------|--------|
| Auth flow completion | Single command, < 2 minutes typical | Met |
| Credential validation | 100% of stored credentials verified before persistence | Met |
| Platform support | macOS with Safari (primary) | Met |
| Fallback coverage | Manual extraction documented | Met |

---

## Constraints and Assumptions

### Constraints
- **Technical**: Browser automation requires macOS with Safari; AppleScript-based
- **Platform**: No Windows or Linux browser automation currently supported

### Assumptions
- Users have Safari installed and can access Tugboat Logic via browser
- Tugboat Logic's cookie-based authentication model remains stable
- `gh` CLI is commonly installed in DevOps environments

---

## Dependencies

- **Safari**: macOS browser for automated authentication
- **Tugboat Logic**: Web application authentication endpoint
- **`gh` CLI** (optional): GitHub CLI for fallback authentication
- **FEAT-005**: Configuration system for credential storage

---

## Out of Scope

- Cross-platform browser automation (Chrome, Firefox)
- OAuth 2.0 / OIDC authentication flows
- API key management for Tugboat Logic
- Multi-user credential management
- Credential refresh/rotation automation

---

## Traceability

### Related Artifacts
- **Parent PRD Section**: Core Capabilities — Automated Browser Authentication
- **Implementation**: `internal/auth/`, `cmd/auth.go`

### Feature Dependencies
- **Depends On**: FEAT-005 (configuration storage)
- **Depended By**: FEAT-007 (sync requires authentication), FEAT-010 (submission requires authentication)

---
*Backfill spec: documents functionality that is already implemented in the codebase.*
