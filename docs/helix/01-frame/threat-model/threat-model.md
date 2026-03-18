---
title: "Threat Model"
phase: "01-frame"
category: "security"
tags: ["threat-model", "stride", "security", "risk-assessment", "attack-surface"]
related: ["security-requirements", "security-architecture", "compliance-requirements"]
created: 2025-01-10
updated: 2026-03-17
helix_mapping: "Frame phase threat modeling - drives Design phase security architecture decisions"
---

# Threat Model

**Project**: GRCTool
**Version**: 1.0
**Date**: 2026-03-17
**Threat Modeling Team**: Security Engineering, Platform Engineering
**Review Date**: 2026-06-17

---

## Executive Summary

**System Overview**: GRCTool is a CLI-based compliance evidence collection tool that integrates with Tugboat Logic, GitHub, Google Workspace, and Claude AI to automate SOC 2 and ISO 27001 audit evidence generation. It runs locally on macOS workstations and communicates with external APIs over HTTPS.

**Key Assets**: Authentication credentials (Tugboat bearer tokens, GitHub tokens, Claude API keys), compliance evidence data, the master index (canonical registry of all compliance artifacts), configuration files with collector URLs and integration settings, and API integration channels.

**Primary Threats**: Credential exposure through logs or source code; command injection via CLI inputs; supply chain compromise through Go dependencies; unauthorized evidence tampering; information disclosure through error messages or cached data; master index corruption or poisoning with amplified blast radius as system of record; bidirectional sync conflicts across integration targets; integration adapter compromise.

**Risk Level**: Medium-High -- GRCTool handles sensitive credentials and compliance data. As it evolves into the system of record for compliance artifacts, the blast radius of data integrity and availability threats increases compared to the previous aggregator model.

## System Description

### System Boundaries

**In Scope**:
- GRCTool CLI application (Go binary)
- Configuration files (`.grctool.yaml`)
- Authentication subsystem (Safari browser auth, GitHub CLI auth, environment variables)
- Local data storage (evidence files, cache, synced policies/controls/tasks)
- API integrations (Tugboat Logic, GitHub, Google Workspace, Claude AI)
- Pre-commit security tooling (secret detection)

**Out of Scope**:
- Tugboat Logic platform security (external SaaS)
- GitHub platform security (external SaaS)
- Google Workspace platform security (external SaaS)
- Anthropic Claude AI platform security (external SaaS)
- macOS operating system security
- Network infrastructure between user workstation and API endpoints

**Trust Boundaries**:
1. **User Workstation <-> GRCTool CLI**: User input crosses into the application
2. **GRCTool CLI <-> Local Filesystem**: Application reads/writes evidence, config, and cache files
3. **GRCTool CLI <-> Tugboat Logic API**: Authenticated HTTPS communication for data sync and evidence submission
4. **GRCTool CLI <-> GitHub API**: Token-authenticated HTTPS communication for repository analysis
5. **GRCTool CLI <-> Google Workspace API**: OAuth2-authenticated HTTPS communication for document access
6. **GRCTool CLI <-> Claude AI API**: API key-authenticated HTTPS communication for evidence generation
7. **GRCTool CLI <-> Safari Browser**: AppleScript automation for cookie extraction
8. **GRCTool CLI <-> External Commands**: Controlled execution of allowlisted commands (git, terraform, kubectl)
9. **GRCTool CLI <-> Master Index**: Application reads/writes the canonical compliance artifact registry; integrity and consistency are critical
10. **Master Index <-> Integration Adapters**: Bidirectional data flow between the authoritative master index and external platform adapters (import/export/sync); conflict resolution occurs at this boundary

### System Components

1. **CLI Command Layer** (cobra/pflag): Parses user input, validates arguments, routes to subcommands. Trust level: untrusted input.
2. **Authentication Subsystem** (`internal/auth`): Manages credentials for Tugboat (Safari cookie extraction), GitHub (gh CLI / env var), and Google Workspace (OAuth2). Trust level: sensitive credential handling.
3. **Configuration Loader** (`internal/config`): Reads `.grctool.yaml`, processes environment variable substitution, validates structure. Trust level: locally trusted but potentially user-modified.
4. **API Clients**: HTTP clients for Tugboat, GitHub, Google, and Claude APIs. Trust level: authenticated outbound connections over TLS.
5. **Evidence Engine**: Generates, validates, evaluates, and submits compliance evidence. Trust level: processes data from multiple sources.
6. **Tool Subsystem**: 30+ evidence collection tools including Terraform analyzers, GitHub scanners, and document readers. Trust level: executes with application privileges.
7. **Storage Layer**: Reads and writes JSON, Markdown, and YAML files to local filesystem. Trust level: locally trusted.
8. **Logging Subsystem** (`internal/logger`): Structured logging with redaction. Trust level: must not leak credentials.

### Data Flows

**External Data Sources**:
- Tugboat Logic API: Policies, controls, evidence tasks (JSON)
- GitHub API: Repository metadata, permissions, workflows, security features
- Google Workspace API: Documents, spreadsheets, drive contents
- Claude AI API: Generated evidence content and analysis
- Safari Browser: Session cookies and bearer tokens
- `gh` CLI: GitHub authentication tokens

**Internal Data Processing**:
- Configuration loading and environment variable substitution
- Template variable interpolation in policy/control documents
- Evidence generation combining multiple data sources
- Evidence validation and quality scoring
- Relationship mapping between evidence tasks, controls, and policies

**External Data Destinations**:
- Tugboat Logic Collector API: Evidence submission via HTTP Basic Auth
- Local filesystem: Evidence files, cache, logs

## Assets and Protection Goals

### Data Assets

| Asset | Classification | Confidentiality | Integrity | Availability | Owner |
|---|---|---|---|---|---|
| Tugboat bearer token | Critical | Critical | High | High | Auth Subsystem |
| GitHub API token | Critical | Critical | High | High | Auth Subsystem |
| Claude API key | Critical | Critical | High | High | Environment |
| Google Workspace OAuth credentials | Critical | Critical | High | Medium | Config / Credentials file |
| Tugboat Basic Auth password | Critical | Critical | High | Medium | Environment / Config |
| Safari session cookies | Sensitive | High | Medium | Low | Auth Subsystem |
| Compliance evidence data | Sensitive | High | Critical | High | Storage Layer |
| Synced policies and controls | Internal | Medium | High | Medium | Storage Layer |
| Configuration file (.grctool.yaml) | Internal | Medium | High | High | Local Filesystem |
| Collector URLs | Internal | Medium | High | Medium | Config |
| Application logs | Internal | Medium (after redaction) | Medium | Low | Logging Subsystem |
| Master index (`.index/`) | Critical | High | Critical | Critical | Storage Layer |
| Integration adapter state | Sensitive | Medium | High | High | Adapter Subsystem |
| Cache data | Low | Low | Low | Low | Storage Layer |

### System Assets

| Asset | Description | Criticality | Dependencies |
|---|---|---|---|
| Authentication subsystem | Safari auth, GitHub auth, token management | Critical | Safari, gh CLI, macOS |
| Tugboat API client | Sync and submission of compliance data | High | Tugboat bearer token, network |
| GitHub API client | Repository security analysis | High | GitHub token, network |
| Evidence generation engine | AI-powered evidence creation | High | Claude API key, tool subsystem |
| Configuration system | Application settings and secrets | High | Viper, local filesystem |
| Secret detection tooling | Pre-commit security scanning | Medium | Git hooks, bash |
| Master index registry | Canonical registry of all compliance artifacts (policies, controls, mappings, evidence tasks) | Critical | Local filesystem, YAML storage |
| Integration adapter subsystem | Bidirectional adapters for import, export, and sync with external platforms | High | Master index, external API clients, conflict resolution logic |

## STRIDE Threat Analysis

### Spoofing Identity

| Threat | Description | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|
| TM-S-001 | Attacker steals Tugboat bearer token from cached credentials or logs to impersonate a legitimate user against the Tugboat API | 4 (Major) | 2 (Low) | 8 (Low) | Log redaction for token/cookie fields; 24-hour token expiration; auth cache in restricted directory |
| TM-S-002 | Attacker obtains GitHub token from environment variables or `gh` CLI to access repository data | 4 (Major) | 2 (Low) | 8 (Low) | Token format validation (ghp_, ghs_ prefixes); leveraging `gh` CLI's own credential storage; avoid storing tokens in config files |
| TM-S-003 | Malicious AppleScript execution during Safari auth flow extracts cookies from a different domain | 3 (Moderate) | 1 (Very Low) | 3 (Very Low) | Domain validation -- Safari script checks URL contains "tugboatlogic.com" before extracting cookies |
| TM-S-004 | Attacker spoofs the Tugboat API endpoint via DNS poisoning or config manipulation to harvest credentials | 4 (Major) | 1 (Very Low) | 4 (Very Low) | HTTPS with TLS validation; base URL configured in version-controlled config; URL validation |

### Tampering with Data

| Threat | Description | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|
| TM-T-001 | Attacker modifies `.grctool.yaml` to redirect evidence submission to a malicious collector URL | 4 (Major) | 2 (Low) | 8 (Low) | Config file under version control; collector URL validation; config structure validation on load |
| TM-T-002 | Attacker modifies generated evidence files on disk before submission | 4 (Major) | 2 (Low) | 8 (Low) | Evidence includes generation timestamps and source metadata; review workflow before submission |
| TM-T-003 | Attacker tampers with synced policy/control documents to alter compliance evidence context | 3 (Moderate) | 2 (Low) | 6 (Low) | Data directory under version control; sync from authoritative Tugboat source; re-sync capability |
| TM-T-004 | Malicious template variable injection via `{{...}}` syntax in interpolation configuration | 3 (Moderate) | 1 (Very Low) | 3 (Very Low) | Circular reference detection in template variables; substitution limited to configured variables only; DFS-based cycle detection |

### Repudiation

| Threat | Description | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|
| TM-R-001 | User denies generating or submitting specific evidence, with no audit trail to prove otherwise | 3 (Moderate) | 3 (Medium) | 9 (Low) | Evidence files include generation timestamps; structured logging of evidence operations; file-based logs with caller information |
| TM-R-002 | User denies modifying configuration that led to incorrect evidence generation | 2 (Minor) | 3 (Medium) | 6 (Low) | Configuration file under git version control; validate config structure on load; log config file path used |

### Information Disclosure

| Threat | Description | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|
| TM-I-001 | Credentials leaked in application logs (bearer tokens, API keys, cookies) | 5 (Catastrophic) | 3 (Medium) | 15 (High) | Default redaction of password/token/key/secret/api_key/cookie fields; URL sanitization enabled by default; sensitive fields excluded from structured log output |
| TM-I-002 | Secrets committed to source code repository | 5 (Catastrophic) | 2 (Low) | 10 (Medium) | `scripts/detect-secrets.sh` pre-commit hook; comprehensive pattern matching for API keys, tokens, private keys, passwords, connection strings; allowed patterns for test data |
| TM-I-003 | Error messages expose internal system paths, API endpoints, or token fragments | 3 (Moderate) | 3 (Medium) | 9 (Low) | Error wrapping with `fmt.Errorf` and `%w`; authentication errors return typed errors (ErrAuthenticationFailed) without token data |
| TM-I-004 | Cached evidence or auth data accessible to other users on shared workstation | 3 (Moderate) | 2 (Low) | 6 (Low) | Auth cache directory with restricted permissions; cache is deletable without data loss; evidence in user-owned directories |
| TM-I-005 | Claude AI API receives sensitive compliance data that may be retained by the AI provider | 3 (Moderate) | 3 (Medium) | 9 (Low) | Use enterprise AI agreements with data handling terms; minimize sensitive data in prompts; document data flows to AI service |

### Denial of Service

| Threat | Description | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|
| TM-D-001 | API rate limiting causes evidence generation to fail during batch operations | 3 (Moderate) | 4 (High) | 12 (Medium) | Configurable rate limits (Tugboat: 10 req/min default; GitHub Search: 30 req/min default); retry logic; resumable batch operations |
| TM-D-002 | Malformed or excessively large API responses exhaust local memory | 2 (Minor) | 2 (Low) | 4 (Very Low) | HTTP client timeouts (default: 30 seconds); response size validation; context-based cancellation |
| TM-D-003 | Runaway template interpolation or circular variable references cause infinite loops | 2 (Minor) | 1 (Very Low) | 2 (Very Low) | DFS-based circular reference detection implemented in `validateInterpolationVariables`; config validation runs at load time |

### Elevation of Privilege

| Threat | Description | Impact | Likelihood | Risk | Mitigation |
|---|---|---|---|---|---|
| TM-E-001 | Command injection through CLI arguments allows arbitrary system command execution | 5 (Catastrophic) | 2 (Low) | 10 (Medium) | Command allowlisting (git, terraform, kubectl only); argument character validation (`[a-zA-Z0-9\-_./]`); dangerous pattern rejection (`;`, `&&`, `||`, pipes, backticks, `$(`) |
| TM-E-002 | Path traversal in evidence task IDs or file names allows reading/writing arbitrary files | 4 (Major) | 2 (Low) | 8 (Low) | Task ID validation (`ET-\d{4}`); file paths resolved relative to configured data directory; path sanitization for directory traversal (`../`) |
| TM-E-003 | Compromised Go dependency introduces malicious code executed with application privileges | 5 (Catastrophic) | 1 (Very Low) | 5 (Low) | Go module version pinning; `go.sum` integrity verification; `govulncheck` in CI/CD; regular dependency audits |

## System-of-Record Threat Analysis

As GRCTool evolves from a compliance data aggregator into the **system of record** (see ADR-010), new threat categories emerge. The master index becomes the canonical registry for all compliance artifacts, and bidirectional integration adapters replace the previous pull-only sync model. These changes amplify the blast radius of data integrity threats and introduce new attack surfaces around sync, adapters, and data sovereignty.

### TM-SOR-001: Bidirectional Sync Conflicts

| Attribute | Detail |
|---|---|
| **Threat ID** | TM-SOR-001 |
| **Title** | Bidirectional Sync Conflicts |
| **Description** | When GRCTool becomes the canonical source, conflicting updates from multiple sources (Tugboat Logic, CLI edits, future integrations) could create inconsistent compliance state. Concurrent modifications to the same artifact through different integration paths may result in silent data loss, stale data overwriting newer changes, or divergent artifact versions across platforms. |
| **Attack Vector** | Concurrent updates to the same compliance artifact from two or more sources (e.g., a user edits a control locally while Tugboat Logic pushes an update via sync). Race conditions in conflict resolution logic could be exploited by timing attacks or simply occur through normal multi-user workflows. |
| **Impact** | 4 (Major) -- Inconsistent compliance state across platforms; evidence generated from stale or conflicting data may be inaccurate; audit findings based on incorrect artifact versions; potential regulatory non-compliance. |
| **Likelihood** | 4 (High) -- Bidirectional sync with multiple active sources makes conflicts a near-certainty in operational use. |
| **Risk Score** | 16 (High) |
| **Severity** | High |
| **Mitigations (Planned)** | Configurable conflict resolution policies (local_wins, remote_wins, manual, newest_wins) per ADR-010; vector clock or timestamp-based change tracking; conflict detection surfaced to user before resolution; comprehensive audit log of all sync operations and conflict resolutions; dry-run mode for sync operations. |
| **Status** | Planned |

### TM-SOR-002: Master Index Poisoning

| Attribute | Detail |
|---|---|
| **Threat ID** | TM-SOR-002 |
| **Title** | Master Index Poisoning |
| **Description** | Malicious or incorrect data written to the master index propagates to all downstream consumers and integration targets. As the system of record, corrupted data has an amplified blast radius -- every integration adapter, evidence generation workflow, and compliance report draws from the master index. A single poisoned artifact can cascade into incorrect evidence, failed audits, and regulatory exposure. |
| **Attack Vector** | Direct filesystem modification of `.index/` YAML files; compromised import adapter injecting malicious artifacts during sync; exploitation of insufficient input validation on artifact creation/update; malformed data from an external platform accepted without schema validation. |
| **Impact** | 5 (Catastrophic) -- Corrupted canonical data propagates to all integration targets and downstream evidence; potential for widespread compliance failures across frameworks; recovery requires identifying and reverting all poisoned artifacts and their downstream effects. |
| **Likelihood** | 2 (Low) -- Requires local filesystem access or a compromised integration adapter; git version control provides some protection. |
| **Risk Score** | 10 (Medium) |
| **Severity** | High |
| **Mitigations (Planned)** | Schema validation on all master index writes (import, create, update); integrity checksums for master index artifacts; git-based version control for full audit trail and rollback capability; input sanitization on all adapter imports; periodic integrity verification against known-good snapshots; write-ahead logging for multi-artifact operations. |
| **Status** | Planned |

### TM-SOR-003: Data Sovereignty and Residency

| Attribute | Detail |
|---|---|
| **Threat ID** | TM-SOR-003 |
| **Title** | Data Sovereignty and Residency |
| **Description** | Local-first storage of compliance data raises questions about data residency, backup, and regulatory compliance. As the system of record, the master index contains the canonical copy of all compliance artifacts. Organizations operating under GDPR, data localization requirements, or industry-specific regulations must ensure that compliance data storage and transfer practices meet jurisdictional requirements. |
| **Attack Vector** | Inadvertent sync of compliance data to cloud platforms in restricted jurisdictions; lack of encryption at rest exposing data to unauthorized local access; insufficient backup practices leading to data loss of the canonical compliance record; cross-border data transfer through integration adapters without organizational awareness. |
| **Impact** | 3 (Moderate) -- Regulatory fines or audit findings for data residency violations; data loss of canonical compliance records if backups are insufficient; organizational liability for improper cross-border data transfers. |
| **Likelihood** | 3 (Medium) -- Data residency requirements are common in regulated industries; local-first storage shifts residency responsibility to the organization. |
| **Risk Score** | 9 (Low) |
| **Severity** | Medium |
| **Mitigations (Planned)** | Document data residency implications in deployment guide; support configurable sync policies that restrict which data flows to which integration targets; recommend macOS FileVault or equivalent disk encryption for at-rest protection; provide backup guidance for the master index directory; integration adapter configuration includes data classification and transfer restrictions. |
| **Status** | Planned |

### TM-SOR-004: Integration Adapter Compromise

| Attribute | Detail |
|---|---|
| **Threat ID** | TM-SOR-004 |
| **Title** | Integration Adapter Compromise |
| **Description** | A compromised or malicious integration adapter could exfiltrate compliance data to unauthorized systems or inject false data into the master index. As GRCTool moves to a plugin-based integration architecture, each adapter becomes a potential attack vector with read and write access to the canonical compliance registry. |
| **Attack Vector** | Supply chain compromise of an adapter dependency; malicious third-party adapter distributed through unofficial channels; compromised adapter credentials used to establish unauthorized data flows; adapter code modified post-installation to exfiltrate data or inject artifacts. |
| **Impact** | 5 (Catastrophic) -- Exfiltration of complete compliance data (policies, controls, evidence) to unauthorized parties; injection of false compliance artifacts that propagate through evidence generation; potential for undetected long-term data theft. |
| **Likelihood** | 2 (Low) -- Requires either supply chain compromise or social engineering to install a malicious adapter; plugin architecture is not yet implemented. |
| **Risk Score** | 10 (Medium) |
| **Severity** | High |
| **Mitigations (Planned)** | Adapter interface enforces principle of least privilege (adapters declare required permissions); adapter provenance verification (code signing or checksum validation); adapter sandbox limiting filesystem and network access to declared integration targets; audit logging of all adapter operations (reads, writes, network calls); adapter allowlisting in configuration; review process for third-party adapters. |
| **Status** | Planned |

### TM-SOR-005: Cascade Failure from Master Index Unavailability

| Attribute | Detail |
|---|---|
| **Threat ID** | TM-SOR-005 |
| **Title** | Cascade Failure from Master Index Unavailability |
| **Description** | If the master index becomes corrupted or unavailable, all dependent integrations and evidence workflows are blocked. As the single source of truth, the master index is a single point of failure for the entire compliance program. Filesystem corruption, accidental deletion, or failed migrations could render the index unusable. |
| **Attack Vector** | Filesystem corruption (disk failure, OS crash during write); accidental deletion of `.index/` directory; failed schema migration leaving the index in an inconsistent state; git merge conflicts corrupting YAML index files; disk space exhaustion preventing index writes. |
| **Impact** | 4 (Major) -- All evidence generation, sync operations, and compliance reporting blocked until index is restored; potential data loss if backups are insufficient; recovery time depends on index size and backup freshness. |
| **Likelihood** | 2 (Low) -- Filesystem corruption is uncommon on modern systems; git provides inherent backup; accidental deletion is mitigable with standard practices. |
| **Risk Score** | 8 (Low) |
| **Severity** | Medium |
| **Mitigations (Planned)** | Git version control provides inherent backup and rollback for all index files; `grctool index verify` command to validate index integrity on demand; write-ahead logging for multi-file index operations; graceful degradation -- read-only mode when index writes fail; automated index rebuild from synced data sources as a recovery path; backup recommendations in operational documentation. |
| **Status** | Planned |

### TM-SOR-006: Unauthorized Master Data Modification

| Attribute | Detail |
|---|---|
| **Threat ID** | TM-SOR-006 |
| **Title** | Unauthorized Master Data Modification |
| **Description** | Without proper access controls on the master index, any CLI user or automated process could modify canonical compliance data. In a system-of-record model, unauthorized modifications to the master index directly alter the organization's official compliance posture -- unlike the aggregator model where local data could be re-synced from Tugboat. |
| **Attack Vector** | Direct filesystem editing of `.index/` YAML files by unauthorized users on a shared workstation; automated scripts or CI/CD processes writing to the master index without authorization; malicious modification of compliance artifacts to weaken security controls or fabricate evidence task completion. |
| **Impact** | 5 (Catastrophic) -- Unauthorized changes to canonical compliance data; fabricated or weakened controls may pass through evidence generation undetected; audit integrity compromised if modifications are not traceable. |
| **Likelihood** | 3 (Medium) -- Local filesystem access controls are the primary barrier; shared workstations or permissive file permissions increase likelihood. |
| **Risk Score** | 15 (High) |
| **Severity** | High |
| **Mitigations (Planned)** | Restrictive filesystem permissions (600/700) on master index directory; git commit signing for index modifications to ensure attribution; `grctool index audit` command to detect unauthorized changes by comparing working tree to last signed commit; index write operations require explicit CLI confirmation; structured audit log of all index modifications with user identity and timestamp; read-only mode configurable for environments where index should not be modified. |
| **Status** | Planned |

## Attack Trees

### Attack Tree 1: Credential Theft

```
Steal Authentication Credentials
+-- Extract from Logs
|   +-- Bearer token in application log [TM-I-001]
|   +-- API key in structured log output [TM-I-001]
|   +-- Cookie data in debug output [TM-I-001]
+-- Extract from Source Code
|   +-- Hardcoded token in committed file [TM-I-002]
|   +-- API key in configuration committed to git [TM-I-002]
|   +-- Credentials in test fixtures [TM-I-002]
+-- Extract from Local Storage
|   +-- Read auth cache directory [TM-I-004]
|   +-- Read .grctool.yaml with inline credentials [TM-I-004]
|   +-- Read environment variables from process [TM-S-002]
+-- Intercept in Transit
    +-- MITM on API communication [TM-S-004]
    +-- DNS spoofing to redirect API calls [TM-S-004]
```

### Attack Tree 2: Evidence Manipulation

```
Tamper with Compliance Evidence
+-- Modify Evidence Files
|   +-- Direct filesystem modification [TM-T-002]
|   +-- Redirect output to attacker-controlled path [TM-E-002]
+-- Modify Evidence Sources
|   +-- Alter synced policy/control documents [TM-T-003]
|   +-- Inject malicious template variables [TM-T-004]
|   +-- Modify configuration to use wrong data sources [TM-T-001]
+-- Redirect Evidence Submission
    +-- Change collector URL in config [TM-T-001]
    +-- Spoof Tugboat API endpoint [TM-S-004]
```

### Attack Tree 3: Arbitrary Code Execution

```
Execute Arbitrary Code
+-- Command Injection
|   +-- Inject shell metacharacters in CLI arguments [TM-E-001]
|   +-- Inject commands via evidence task names [TM-E-001]
|   +-- Exploit external command execution [TM-E-001]
+-- Path Traversal
|   +-- Traverse to system files via task ID manipulation [TM-E-002]
|   +-- Write to arbitrary paths via evidence output [TM-E-002]
+-- Supply Chain
    +-- Compromised Go module dependency [TM-E-003]
    +-- Malicious code in build toolchain [TM-E-003]
```

### Attack Tree 4: System-of-Record Compromise

```
Compromise Master Index Integrity
+-- Poison Master Index Data
|   +-- Direct filesystem modification of .index/ YAML files [TM-SOR-002, TM-SOR-006]
|   +-- Compromised integration adapter injects false artifacts [TM-SOR-004]
|   +-- Malformed external data accepted without validation [TM-SOR-002]
+-- Create Inconsistent Compliance State
|   +-- Exploit bidirectional sync race conditions [TM-SOR-001]
|   +-- Force conflict resolution to overwrite correct data [TM-SOR-001]
|   +-- Modify artifacts during migration from aggregator model [TM-SOR-001]
+-- Exfiltrate Compliance Data
|   +-- Compromised adapter sends data to unauthorized target [TM-SOR-004]
|   +-- Sync configured to push data to restricted jurisdiction [TM-SOR-003]
+-- Deny Access to Compliance Data
|   +-- Corrupt or delete master index files [TM-SOR-005]
|   +-- Trigger failed schema migration [TM-SOR-005]
|   +-- Exhaust disk space to prevent index writes [TM-SOR-005]
```

## Risk Assessment Matrix

### Risk Scoring
- **Impact**: 1 (Minimal) - 5 (Catastrophic)
- **Likelihood**: 1 (Very Low) - 5 (Very High)
- **Risk Score**: Impact x Likelihood

### Risk Levels
- **Critical (20-25)**: Immediate action required
- **High (15-19)**: Action required within 30 days
- **Medium (10-14)**: Action required within 90 days
- **Low (5-9)**: Monitor and plan mitigation
- **Very Low (1-4)**: Accept risk or implement if cost-effective

### Top Risks Identified

| Risk ID | Threat | Impact | Likelihood | Risk Score | Priority |
|---|---|---|---|---|---|
| TM-SOR-001 | Bidirectional sync conflicts create inconsistent compliance state | 4 | 4 | 16 | High |
| TM-I-001 | Credential leakage in application logs | 5 | 3 | 15 | High |
| TM-SOR-006 | Unauthorized master data modification | 5 | 3 | 15 | High |
| TM-D-001 | API rate limiting disrupts batch evidence generation | 3 | 4 | 12 | Medium |
| TM-I-002 | Secrets committed to source code | 5 | 2 | 10 | Medium |
| TM-E-001 | Command injection via CLI arguments | 5 | 2 | 10 | Medium |
| TM-SOR-002 | Master index poisoning with amplified blast radius | 5 | 2 | 10 | Medium |
| TM-SOR-004 | Integration adapter compromise (exfiltration or injection) | 5 | 2 | 10 | Medium |
| TM-I-005 | Sensitive data sent to Claude AI API | 3 | 3 | 9 | Low |
| TM-R-001 | Insufficient audit trail for evidence generation | 3 | 3 | 9 | Low |
| TM-SOR-003 | Data sovereignty and residency compliance | 3 | 3 | 9 | Low |
| TM-T-001 | Configuration tampering redirects evidence submission | 4 | 2 | 8 | Low |
| TM-S-001 | Tugboat bearer token theft | 4 | 2 | 8 | Low |
| TM-S-002 | GitHub token theft | 4 | 2 | 8 | Low |
| TM-T-002 | Evidence file tampering on disk | 4 | 2 | 8 | Low |
| TM-SOR-005 | Cascade failure from master index unavailability | 4 | 2 | 8 | Low |

## Mitigation Strategies

### Immediate Actions (High Risk)

1. **TM-I-001 -- Credential Log Redaction**
   - Default redaction of sensitive fields: password, token, key, secret, api_key, cookie
   - URL sanitization enabled by default to strip query parameters
   - Dedicated `LoggerConfig.RedactFields` configuration with sensible defaults
   - **Status**: Implemented in `internal/config/config.go` (DefaultLoggingConfig)
   - **Owner**: Security Engineering
   - **Residual Risk**: Custom log statements outside the structured logger may bypass redaction

### Medium-Term Actions (Medium Risk)

2. **TM-D-001 -- API Rate Limit Management**
   - Configurable rate limits per API (Tugboat: `rate_limit`, GitHub: `rate_limit`)
   - HTTP client timeouts (default: 30 seconds)
   - **Status**: Implemented in configuration; retry logic in evidence generation
   - **Owner**: Platform Engineering

3. **TM-I-002 -- Pre-Commit Secret Detection**
   - Comprehensive `scripts/detect-secrets.sh` scanning 20+ secret patterns
   - Covers AWS, GitHub, Google Cloud, Slack, Stripe, private keys, passwords, connection strings
   - Allowed patterns for test data and placeholders
   - **Status**: Implemented and available as pre-commit hook
   - **Owner**: Development Team

4. **TM-E-001 -- Command Injection Prevention**
   - Command allowlisting restricts execution to git, terraform, kubectl
   - Argument character validation with allowlist pattern
   - Dangerous metacharacter detection and rejection
   - Context-based timeout (30 seconds)
   - **Status**: Implemented in security architecture design
   - **Owner**: Security Engineering

### Long-Term Actions (Low Risk)

5. **TM-I-005 -- AI Data Handling**
   - Document data flows to Claude AI service
   - Establish enterprise AI agreement with Anthropic for data handling
   - Minimize sensitive data in evidence generation prompts
   - **Status**: Planning
   - **Owner**: Compliance Team
   - **Timeline**: 90 days

6. **TM-R-001 -- Enhanced Audit Trail**
   - Implement structured evidence generation audit logs
   - Add evidence integrity checksums
   - Track user actions with timestamps and operation context
   - **Status**: Partially implemented (timestamps in evidence files)
   - **Owner**: Product Engineering
   - **Timeline**: 120 days

### System-of-Record Actions (Planned)

7. **TM-SOR-001 -- Bidirectional Sync Conflict Resolution**
   - Implement configurable conflict resolution policies (local_wins, remote_wins, manual, newest_wins)
   - Add vector clock or timestamp-based change tracking for all master index artifacts
   - Surface conflicts to users before automatic resolution; require explicit confirmation for destructive resolutions
   - Provide dry-run mode for all sync operations to preview changes
   - Comprehensive audit log of sync operations and conflict resolutions
   - **Status**: Planned (depends on ADR-010 system-of-record implementation)
   - **Owner**: Platform Engineering
   - **Timeline**: Aligned with system-of-record migration phases

8. **TM-SOR-002 -- Master Index Integrity Protection**
   - Schema validation on all master index writes (import, create, update operations)
   - Integrity checksums for master index artifacts
   - Git-based version control provides audit trail and rollback capability
   - Input sanitization and schema validation on all adapter imports
   - Periodic integrity verification against known-good snapshots
   - Write-ahead logging for multi-artifact operations to prevent partial writes
   - **Status**: Planned
   - **Owner**: Security Engineering
   - **Timeline**: Aligned with master index implementation

9. **TM-SOR-004 -- Integration Adapter Security**
   - Adapter interface enforces principle of least privilege (declared permissions model)
   - Adapter provenance verification (code signing or checksum validation)
   - Audit logging of all adapter operations (reads, writes, network calls)
   - Adapter allowlisting in configuration to prevent unauthorized adapters
   - Review process for third-party adapter adoption
   - **Status**: Planned
   - **Owner**: Security Engineering
   - **Timeline**: Aligned with plugin-based integration architecture

10. **TM-SOR-006 -- Master Index Access Control**
    - Restrictive filesystem permissions (600/700) on master index directory
    - Git commit signing for index modifications to ensure attribution
    - `grctool index audit` command to detect unauthorized changes
    - Structured audit log of all index modifications with user identity and timestamp
    - Read-only mode configurable for environments where index should not be modified
    - **Status**: Planned
    - **Owner**: Security Engineering
    - **Timeline**: Aligned with system-of-record migration phases

## Security Controls Mapping

### Preventive Controls
- **Authentication**: Browser-based Tugboat auth with domain validation; GitHub token format validation; environment variable-based secret storage
- **Authorization**: Command allowlisting; argument sanitization; file path validation; master index access controls with restrictive filesystem permissions (planned)
- **Input Validation**: Task ID regex validation; template variable cycle detection; configuration structure validation; schema validation on all master index writes (planned)
- **Encryption**: HTTPS/TLS for all API communication; no plaintext credential storage in config files
- **Data Integrity**: Master index integrity checksums; write-ahead logging for multi-artifact operations; adapter provenance verification (all planned)

### Detective Controls
- **Logging**: Structured logging with zerolog; configurable log levels; file and console outputs; audit logging of all adapter and index operations (planned)
- **Secret Detection**: Pre-commit scanning with `scripts/detect-secrets.sh`; pattern-based detection across 20+ secret types
- **Vulnerability Scanning**: `govulncheck` for Go dependency vulnerabilities
- **Configuration Validation**: Unknown key warnings; misplaced section detection; required field validation
- **Index Integrity Monitoring**: `grctool index verify` for on-demand integrity checks; `grctool index audit` for unauthorized change detection; periodic integrity verification against known-good snapshots (all planned)
- **Sync Conflict Detection**: Conflict detection surfaced to users before resolution; dry-run mode for sync operations (planned)

### Corrective Controls
- **Token Rotation**: 24-hour default expiration on Tugboat credentials; re-authentication flow
- **Authentication Recovery**: Clear error messages with step-by-step remediation guidance for Safari automation permissions
- **Cache Invalidation**: Cache directories safely deletable; re-sync from authoritative sources
- **Incident Response**: Security incident procedures documented for credential compromise, vulnerability exploitation, and data breach scenarios
- **Index Recovery**: Git-based rollback for master index corruption; automated index rebuild from synced data sources; graceful degradation to read-only mode on write failures (planned)
- **Conflict Recovery**: Configurable conflict resolution with manual override; full audit trail of sync resolutions for post-incident analysis (planned)

## Assumptions and Dependencies

### Assumptions
- macOS provides adequate filesystem-level access control for credential and evidence storage
- HTTPS/TLS provides sufficient protection for data in transit to API endpoints
- Users operate GRCTool on personal or organizationally managed workstations (not shared public systems)
- External API providers (Tugboat, GitHub, Google, Anthropic) maintain their own security postures
- The `gh` CLI securely manages its own credential storage

### Dependencies
- Safari browser and AppleScript automation for Tugboat authentication
- GitHub CLI (`gh`) for automated token retrieval
- Go module ecosystem for dependency integrity (`go.sum`)
- `govulncheck` for vulnerability detection in Go dependencies
- macOS Keychain and biometric authentication for browser-based login

### Constraints
- macOS-only Safari authentication limits cross-platform adoption
- CLI-only interface constrains threat surface but limits user base
- Local-first architecture means data protection relies on filesystem permissions
- No application-level encryption at rest for evidence files (relies on macOS FileVault)

## Threat Model Validation

### Review Checklist
- [x] All system components identified and trust levels assigned
- [x] Trust boundaries clearly defined (10 boundaries, including master index and integration adapters)
- [x] All data flows documented (sources, internal, destinations)
- [x] STRIDE analysis completed for each threat category
- [x] System-of-record threat analysis completed (6 SOR-specific threats)
- [x] Risk assessment completed with impact and likelihood scores
- [x] Mitigation strategies defined for all medium and higher risks
- [x] Security controls mapped (preventive, detective, corrective)
- [x] Assumptions and dependencies documented

### Validation Questions
1. **Completeness**: Have we identified all relevant threats? -- Yes, systematic STRIDE analysis covers all major categories, supplemented by system-of-record-specific threat analysis (TM-SOR-001 through TM-SOR-006) addressing master index integrity, sync conflicts, adapter security, and data sovereignty.
2. **Accuracy**: Are the risk ratings appropriate? -- Yes, ratings reflect the local CLI context where most threats require local access or social engineering. System-of-record threats are rated higher due to the amplified blast radius of canonical data corruption.
3. **Feasibility**: Are the mitigations practical to implement? -- Yes, most critical mitigations are already implemented (log redaction, secret detection, input validation). System-of-record mitigations are planned and aligned with the phased ADR-010 migration.
4. **Coverage**: Do the controls address all identified threats? -- Yes, with residual risks documented for edge cases. System-of-record controls are planned and will be implemented alongside the master index and adapter architecture.
5. **Prioritization**: Are we focusing on the highest risks first? -- Yes, credential leakage (TM-I-001) and bidirectional sync conflicts (TM-SOR-001) are the top priorities. Existing controls address TM-I-001; SOR mitigations are planned for the system-of-record implementation phase.

## Maintenance and Updates

### Review Schedule
- **Major Review**: Annually or after significant system changes (new API integrations, authentication changes)
- **Minor Review**: Quarterly to assess new threats and dependency vulnerabilities
- **Ad-hoc Review**: After security incidents, new threat intelligence, or audit findings

### Update Triggers
- New external API integration added
- Authentication mechanism changes
- New evidence collection tools introduced
- Security incident involving GRCTool or its dependencies
- Changes to compliance framework requirements (SOC 2, ISO 27001)
- Go dependency vulnerability discoveries
- System-of-record migration phase transitions (shadow index, dual-write, GRCTool-as-source)
- New integration adapter added or third-party adapter adopted

---

**Document Control**
- **Threat Model ID**: TM-GRCTool-1.0
- **Version**: 1.0
- **Last Updated**: 2026-03-17
- **Next Review Date**: 2026-06-17
- **Change History**: Initial creation based on codebase analysis, STRIDE methodology, and template alignment; 2026-03-17: Added system-of-record threat analysis (TM-SOR-001 through TM-SOR-006) covering master index integrity, bidirectional sync conflicts, data sovereignty, integration adapter security, cascade failure, and unauthorized data modification per ADR-010
