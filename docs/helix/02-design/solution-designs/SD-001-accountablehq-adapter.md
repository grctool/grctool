---
title: "SD-001: AccountableHQ Integration Adapter"
phase: "02-design"
category: "solution-design"
tags: ["accountablehq", "adapter", "integration", "policy-sync", "hexagonal-architecture"]
related: ["adr-006", "adr-010", "data-design", "contracts", "FEAT-001"]
created: 2026-03-17
updated: 2026-03-17
---

# SD-001: AccountableHQ Integration Adapter

## Overview

This document describes the solution design for a bidirectional integration adapter between GRCTool and AccountableHQ's policy repository. The adapter follows the hexagonal architecture pattern (ADR-006) and implements the standard import/export/sync adapter interface defined by the system of record architecture (ADR-010). It integrates with the master index as the canonical data store and supports configurable conflict resolution, audit logging, and scheduled execution.

---

## Architecture

### Hexagonal Placement

The AccountableHQ adapter is an **infrastructure adapter** (driven side) in GRCTool's hexagonal architecture. It implements `interfaces.SyncProvider` from SD-004/ADR-011.

```
                    +----------------------------+
                    |       Domain Core          |
                    |                            |
                    |  ProviderRegistry          |
                    |  SyncOrchestrator          |
                    |  ConflictResolver          |
                    |                            |
                    +------+----------+----------+
                           |          |
                    [Port] |          | [Port]
                           |          |
              +------------+--+  +----+---------------+
              | DataProvider  |  | SyncProvider        |
              | (interface)   |  | (extends DataProv.) |
              +------+--------+  +----+----------------+
                     |                |
          +----------+----------------+-----------+
          | AccountableHQSyncProvider             |
          | (internal/providers/accountablehq/)   |
          +---------------------------------------+
                     |
                     v
              [AccountableHQ REST API]
              (via AccountableHQClient interface)
```

### Port Interfaces

> **NOTE: These interfaces are superseded by the canonical `DataProvider`/`SyncProvider` interfaces defined in SD-004 and ADR-011. This adapter will implement SD-004's `SyncProvider` interface. The interfaces shown here are retained for historical context but will not be implemented as described.**

The adapter implements three port interfaces, consistent with the integration point contracts defined in the data design:

```go
// Port: ImportAdapter - Pull artifacts from an external system into the master index
type ImportAdapter interface {
    // Import retrieves all policies (or a filtered subset) from the external system.
    // Returns imported artifacts with their external IDs for master index mapping.
    Import(ctx context.Context, opts ImportOptions) (*ImportResult, error)

    // ImportOne retrieves a single policy by external ID.
    ImportOne(ctx context.Context, externalID string) (*PolicyArtifact, error)

    // ListRemote lists available policies in the external system without importing.
    ListRemote(ctx context.Context, opts ListOptions) ([]RemotePolicySummary, error)
}

// Port: ExportAdapter - Push artifacts from the master index to an external system
type ExportAdapter interface {
    // Export pushes one or more policies to the external system.
    // Returns the resulting external IDs and versions.
    Export(ctx context.Context, policies []PolicyArtifact, opts ExportOptions) (*ExportResult, error)

    // ExportOne pushes a single policy to the external system.
    ExportOne(ctx context.Context, policy PolicyArtifact) (*ExportReceipt, error)

    // Delete removes a policy from the external system (tombstone support).
    Delete(ctx context.Context, externalID string) error
}

// Port: SyncAdapter - Bidirectional reconciliation with conflict resolution
type SyncAdapter interface {
    ImportAdapter
    ExportAdapter

    // Sync performs bidirectional reconciliation between the master index and the external system.
    Sync(ctx context.Context, opts SyncOptions) (*SyncResult, error)

    // DetectConflicts compares local and remote state and returns conflicts without resolving them.
    DetectConflicts(ctx context.Context) ([]Conflict, error)

    // ResolveConflict applies a resolution to a specific conflict.
    ResolveConflict(ctx context.Context, conflict Conflict, resolution ConflictResolution) error
}
```

---

## AccountableHQ API Integration Points

### API Surface

The adapter interacts with AccountableHQ through its REST API. The following endpoints are required:

| Operation | HTTP Method | Endpoint (assumed) | Purpose |
|-----------|-------------|-------------------|---------|
| List policies | GET | `/api/v1/policies` | Retrieve all policies with pagination |
| Get policy | GET | `/api/v1/policies/{id}` | Retrieve a single policy by ID |
| Create policy | POST | `/api/v1/policies` | Create a new policy in AccountableHQ |
| Update policy | PUT/PATCH | `/api/v1/policies/{id}` | Update an existing policy |
| Delete policy | DELETE | `/api/v1/policies/{id}` | Remove a policy (soft delete) |
| Get policy version | GET | `/api/v1/policies/{id}/versions` | Retrieve version history |
| Get policy diff | GET | `/api/v1/policies/{id}/diff` | Compare versions |

**Note**: Actual endpoints will be confirmed during AccountableHQ API discovery. The adapter abstracts these behind the port interface, so endpoint changes only affect the adapter implementation.

### API Client

```go
// AccountableHQClient handles HTTP communication with AccountableHQ's API.
// Wraps authentication, rate limiting, and response parsing.
type AccountableHQClient struct {
    baseURL    string
    httpClient *http.Client
    auth       AuthProvider
    rateLimiter *RateLimiter
    logger     zerolog.Logger
}
```

---

## Data Model Mapping

### AccountableHQ Policy to GRCTool Policy Entity

| AccountableHQ Field | GRCTool Field | Transformation |
|---------------------|---------------|----------------|
| `id` | `external_ids.accountablehq` | Stored as external ID in master index |
| `title` / `name` | `name` | Direct mapping |
| `content` / `body` | Policy document (Markdown file) | Stored as `.md` file in `docs/policies/` |
| `status` | `metadata.status` | Map to GRCTool status enum (draft, active, archived) |
| `owner` | `metadata.owner` | Direct mapping |
| `version` | `metadata.version` | Direct mapping |
| `approved_at` | `metadata.approved_at` | ISO-8601 timestamp normalization |
| `approved_by` | `metadata.approved_by` | Direct mapping |
| `created_at` | `created_at` | ISO-8601 timestamp normalization |
| `updated_at` | `updated_at` | ISO-8601 timestamp normalization |
| `tags` / `categories` | `metadata.tags` | Direct mapping |
| (no GRCTool equivalent) | `grctool_id` (POL-NNNN) | Assigned by master index on first import |
| (unmapped fields) | `metadata.extensions.accountablehq` | Preserved as extension metadata to prevent data loss |

### Content Normalization

Policy content must be normalized during mapping to ensure consistent round-trip fidelity:

- HTML content from AccountableHQ is converted to Markdown for local storage
- Markdown from GRCTool is converted to AccountableHQ's expected format on export
- Template variables (`{{organization.name}}`) are preserved as-is during sync; interpolation happens at render time, not at sync time
- Content hashing uses the normalized form to avoid false-positive conflict detection

---

## Sync Algorithm

### Overview

The sync algorithm follows a three-phase approach: **detect changes**, **resolve conflicts**, **apply changes**.

### Phase 1: Change Detection

```
For each policy in (master index UNION remote policies):

  1. Match local and remote by external ID mapping
  2. Compare content_hash and updated_at timestamps
  3. Classify as:
     - LOCAL_ONLY:    exists in master index but not in AccountableHQ
     - REMOTE_ONLY:   exists in AccountableHQ but not in master index
     - UNCHANGED:     content hash matches on both sides
     - LOCAL_CHANGED: local content hash differs from last-synced hash; remote unchanged
     - REMOTE_CHANGED: remote content hash differs from last-synced hash; local unchanged
     - CONFLICT:      both sides changed since last sync
     - TOMBSTONE:     marked as deleted on one side
```

### Phase 2: Conflict Resolution

Conflicts are resolved according to the configured policy (per-entity or global):

| Policy | Behavior |
|--------|----------|
| `local_wins` | GRCTool version overwrites AccountableHQ |
| `remote_wins` | AccountableHQ version overwrites GRCTool |
| `newest_wins` | The version with the most recent `updated_at` is kept |
| `manual` | Conflict is flagged; sync skips the policy until resolved |

For `manual` conflicts, the sync produces a conflict file:

> **NOTE: Per SD-004 and ADR-011, the master index is implemented as the existing `StorageService` enriched with `ExternalIDs` and `SyncMetadata` fields. No separate `.index/` directory is needed. Conflict files will be stored under the data directory in a location determined during implementation.**

```yaml
# .index/conflicts/POL-0012.yaml
policy_id: "POL-0012"
external_id: "accountablehq:policy-abc-123"
detected_at: "2026-03-17T14:30:00Z"
local_version:
  content_hash: "sha256:abc..."
  updated_at: "2026-03-17T14:00:00Z"
  updated_by: "erik"
remote_version:
  content_hash: "sha256:def..."
  updated_at: "2026-03-17T14:15:00Z"
  updated_by: "jane@accountablehq"
status: "unresolved"
```

### Phase 3: Apply Changes

| Classification | Action |
|----------------|--------|
| LOCAL_ONLY | Export to AccountableHQ (create) |
| REMOTE_ONLY | Import to master index (create with new POL-NNNN ID) |
| LOCAL_CHANGED | Export to AccountableHQ (update) |
| REMOTE_CHANGED | Import to master index (update local copy) |
| CONFLICT (resolved) | Apply resolution decision |
| CONFLICT (manual, unresolved) | Skip; record in conflict log |
| TOMBSTONE (local deleted) | Delete from AccountableHQ; remove from master index |
| TOMBSTONE (remote deleted) | Mark as deleted in master index; preserve in local archive |
| UNCHANGED | No action |

### Tombstone Handling

Deletes are tracked as tombstones to prevent re-creation during sync:

```yaml
# In master index registry
POL-0012:
  grctool_id: "POL-0012"
  deleted: true
  deleted_at: "2026-03-17T15:00:00Z"
  deleted_by: "erik"
  tombstone_ttl: "90d"  # Tombstone retained for 90 days before purge
```

When a tombstone is encountered on one side, the sync propagates the delete to the other side rather than treating the missing entry as a new artifact.

---

## Configuration

### .grctool.yaml Section

> **NOTE: The `integrations` config key proposed below is a schema extension to `.grctool.yaml`. Config schema extension is tracked as a separate implementation task and will be validated against the existing `Config` struct in `internal/config/config.go`.**

```yaml
integrations:
  accountablehq:
    enabled: true
    base_url: "https://api.accountablehq.com"

    # Authentication (choose one)
    auth:
      method: "api_key"       # "api_key" or "oauth2"
      api_key: "${ACCOUNTABLEHQ_API_KEY}"  # Environment variable reference

      # OAuth2 settings (if method is "oauth2")
      # oauth2:
      #   client_id: "${ACCOUNTABLEHQ_CLIENT_ID}"
      #   client_secret: "${ACCOUNTABLEHQ_CLIENT_SECRET}"
      #   token_url: "https://auth.accountablehq.com/oauth/token"
      #   scopes: ["policies:read", "policies:write"]

    # Sync settings
    sync:
      direction: "bidirectional"   # "import_only", "export_only", "bidirectional"
      conflict_policy: "newest_wins"  # "local_wins", "remote_wins", "manual", "newest_wins"
      batch_size: 50                  # Policies per API request batch
      concurrency: 3                  # Parallel API requests
      tombstone_ttl: "90d"            # How long to retain delete tombstones

    # Schedule (requires FEAT-003 task scheduler)
    schedule:
      enabled: false
      cron: "0 */6 * * *"       # Every 6 hours
      on_conflict: "newest_wins" # Override conflict policy for scheduled runs (no "manual" allowed)
      notify_on_failure: true

    # Rate limiting
    rate_limit:
      requests_per_second: 10
      burst: 20

    # Field mapping overrides (optional)
    field_mapping:
      status_map:
        "published": "active"
        "draft": "draft"
        "retired": "archived"
```

---

## Authentication

### Option 1: API Key

The simpler integration path. An API key is provisioned in AccountableHQ and stored as an environment variable.

```
Authorization: Bearer <api_key>
```

- API key is read from environment variable (`ACCOUNTABLEHQ_API_KEY`) or `.grctool.yaml` (with environment variable substitution)
- Key is never logged or stored in plaintext in config files committed to git
- Key rotation requires updating the environment variable

### Option 2: OAuth2 Client Credentials

For organizations requiring short-lived tokens and finer-grained scopes.

```
1. Client sends client_id + client_secret to token_url
2. Token endpoint returns access_token (+ refresh_token if supported)
3. Adapter caches access_token and refreshes before expiry
4. All API calls use: Authorization: Bearer <access_token>
```

- Token refresh is automatic and transparent to the sync logic
- Token cache stored in `.cache/accountablehq_token.json` (excluded from git)
- Scopes requested match the minimum required for policy CRUD operations

### Auth Provider Interface

```go
// Implements the AuthProvider port from ADR-006
type AccountableHQAuthProvider struct {
    method  string  // "api_key" or "oauth2"
    config  AuthConfig
    token   *CachedToken
}

func (a *AccountableHQAuthProvider) Authenticate(ctx context.Context) (*AuthResult, error)
func (a *AccountableHQAuthProvider) Refresh(ctx context.Context) (*AuthResult, error)
func (a *AccountableHQAuthProvider) IsValid() bool
```

---

## Error Handling

### Retry Strategy

| Error Type | Retry | Strategy |
|------------|-------|----------|
| Network timeout | Yes | Exponential backoff: 1s, 2s, 4s, 8s, max 3 retries |
| HTTP 429 (rate limited) | Yes | Respect `Retry-After` header; fall back to exponential backoff |
| HTTP 5xx (server error) | Yes | Exponential backoff, max 3 retries |
| HTTP 401 (unauthorized) | Once | Refresh auth token, retry once; fail if still 401 |
| HTTP 4xx (client error, non-401/429) | No | Fail immediately; log error with request context |
| Content validation error | No | Fail for the individual policy; continue batch |

### Partial Sync

When individual policies fail during a batch sync:

- The sync continues processing remaining policies
- Failed policies are recorded in the sync result with error details
- The overall sync is marked as `partial_success` rather than `failure`
- A subsequent sync retries previously failed policies

### Rollback

For critical failures mid-sync (e.g., master index write failure):

- All pending changes are written to a staging area before being committed to the master index
- If the commit fails, the staging area is preserved for manual recovery
- The master index is never left in a partially-updated state
- A recovery command (`grctool sync recover`) can replay or discard staged changes

### Sync Result Structure

```go
type SyncResult struct {
    StartedAt     time.Time
    CompletedAt   time.Time
    Direction     string // "import", "export", "bidirectional"
    Status        string // "success", "partial_success", "failure"
    Imported      []SyncEntry
    Exported      []SyncEntry
    Conflicts     []Conflict
    Errors        []SyncError
    AuditLogEntry string // Path to audit log entry file
}
```

---

## Scheduler Integration

The AccountableHQ adapter integrates with the GRCTool task scheduler (FEAT-003) for automated sync:

### Registration

```go
// During adapter initialization, register a sync task with the scheduler
scheduler.Register(Task{
    Name:     "accountablehq-policy-sync",
    Schedule: config.Integrations.AccountableHQ.Schedule.Cron,
    Enabled:  config.Integrations.AccountableHQ.Schedule.Enabled,
    Execute: func(ctx context.Context) error {
        opts := SyncOptions{
            Direction:      config.Integrations.AccountableHQ.Sync.Direction,
            ConflictPolicy: config.Integrations.AccountableHQ.Schedule.OnConflict,
        }
        result, err := adapter.Sync(ctx, opts)
        if err != nil {
            return fmt.Errorf("scheduled accountablehq sync failed: %w", err)
        }
        if result.Status == "partial_success" {
            logger.Warn().Int("errors", len(result.Errors)).Msg("scheduled sync completed with errors")
        }
        return nil
    },
})
```

### Constraints

- Scheduled sync must not use `manual` conflict policy (no interactive prompts)
- A distributed lock (file-based, `.cache/accountablehq_sync.lock`) prevents overlapping sync runs
- Lock has a configurable TTL (default 30 minutes) to handle stale locks from crashed processes
- Scheduler respects the adapter's rate limit configuration

---

## Audit Trail

Every sync operation produces an append-only audit entry:

```yaml
# {data_dir}/.audit/sync/2026-03-17T143000Z-accountablehq.yaml
operation_id: "sync-20260317-143000-abc123"
timestamp: "2026-03-17T14:30:00Z"
integration: "accountablehq"
direction: "bidirectional"
trigger: "manual"  # or "scheduled"
operator: "erik"
duration_ms: 4200
status: "success"
summary:
  imported: 3
  exported: 1
  conflicts_detected: 1
  conflicts_resolved: 1
  unchanged: 45
  errors: 0
changes:
  - policy_id: "POL-0012"
    action: "imported"
    direction: "inbound"
    fields_changed: ["content", "updated_at"]
    content_hash_before: "sha256:abc..."
    content_hash_after: "sha256:def..."
  - policy_id: "POL-0008"
    action: "exported"
    direction: "outbound"
    fields_changed: ["title", "content"]
    content_hash_before: "sha256:ghi..."
    content_hash_after: "sha256:jkl..."
  - policy_id: "POL-0015"
    action: "conflict_resolved"
    direction: "bidirectional"
    resolution: "newest_wins"
    winner: "remote"
```

---

## Package Structure

```
internal/
+-- adapters/
|   +-- accountablehq/
|   |   +-- client.go          # HTTP client for AccountableHQ API
|   |   +-- client_test.go
|   |   +-- auth.go            # Auth provider (API key + OAuth2)
|   |   +-- auth_test.go
|   |   +-- import.go          # ImportAdapter implementation
|   |   +-- import_test.go
|   |   +-- export.go          # ExportAdapter implementation
|   |   +-- export_test.go
|   |   +-- sync.go            # SyncAdapter implementation (orchestrates import + export)
|   |   +-- sync_test.go
|   |   +-- mapper.go          # Data model mapping (AccountableHQ <-> GRCTool)
|   |   +-- mapper_test.go
|   |   +-- config.go          # Configuration parsing for accountablehq section
|   +-- ports/
|       +-- integration.go     # Port interfaces (ImportAdapter, ExportAdapter, SyncAdapter)
+-- sync/
    +-- orchestrator.go        # Sync orchestrator (change detection, conflict resolution)
    +-- conflict.go            # Conflict detection and resolution logic
    +-- audit.go               # Audit trail writer
```

---

## Testing Strategy

| Level | Approach | Coverage |
|-------|----------|----------|
| Unit | Mock AccountableHQ API responses; test mapper, conflict logic, auth provider | All data transformations, conflict policies, error paths |
| Integration | VCR cassettes (ADR-005) recording real AccountableHQ API interactions | Full import/export/sync flows against recorded API responses |
| Contract | Validate adapter satisfies port interface contracts | Compile-time interface satisfaction + behavioral contract tests |
| End-to-end | Sync against a staging AccountableHQ environment | Round-trip fidelity, conflict detection, audit trail completeness |

---

## Open Questions

| Question | Context | Owner |
|----------|---------|-------|
| What is AccountableHQ's exact API schema for policies? | Adapter design assumes a RESTful CRUD API; actual endpoints need confirmation | Engineering |
| Does AccountableHQ support webhooks for real-time change notification? | Could enable event-driven sync instead of polling | Engineering |
| What are AccountableHQ's rate limits? | Affects batch_size and concurrency configuration defaults | Engineering |
| Does AccountableHQ support content versioning natively? | Affects conflict detection and audit trail implementation | Engineering |
| What AccountableHQ policy fields are required vs. optional? | Affects export validation and error handling | Engineering |

---

## References

- [FEAT-001: AccountableHQ Bidirectional Policy Sync](/home/erik/Projects/grctool/docs/helix/01-frame/features/FEAT-001-accountablehq-policy-sync.md)
- [ADR-006: Hexagonal Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-006-hexagonal-architecture-ports-and-adapters)
- [ADR-010: System of Record Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-010-system-of-record-architecture)
- [Data Design: Integration Point Contracts](/home/erik/Projects/grctool/docs/helix/02-design/data-design/data-design.md#integration-point-contracts)
- [Interface Contracts](/home/erik/Projects/grctool/docs/helix/02-design/contracts/contracts.md)
