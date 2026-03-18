---
title: "SD-002: Google Drive Sync Adapter"
phase: "02-design"
category: "solution-design"
status: "Proposed"
tags: ["google-drive", "sync-adapter", "bidirectional", "integration"]
related: ["FEAT-002", "adr-010", "adr-006", "data-design", "google-workspace"]
created: 2026-03-17
updated: 2026-03-17
---

# SD-002: Google Drive Sync Adapter

## Overview

This solution design extends GRCTool's Google Workspace integration with write-back capability, enabling bidirectional synchronization between GRCTool's master index and Google Drive. The existing `GoogleWorkspaceTool` (`internal/tools/google_workspace.go`) remains unchanged as a read-only evidence extraction tool. A new `GDriveSyncAdapter` implements the standard adapter interface (ADR-006) to handle outbound publishing, inbound change detection, and bidirectional reconciliation.

### Design Principles

- **Additive, not invasive**: The existing read-only `GoogleWorkspaceTool` is untouched; the sync adapter is a separate component.
- **Master index is authoritative**: GRCTool's master index (ADR-010) is the source of truth. Google Drive holds a derived, synchronized copy.
- **Selective sync**: Only explicitly configured artifacts are synced. Default is opt-in.
- **Conflict-aware**: The adapter detects and surfaces conflicts rather than silently overwriting.

## Architecture

### Component Placement

```
internal/
+-- adapters/
|   +-- gdrive/
|   |   +-- adapter.go          # GDriveSyncAdapter (implements SyncAdapter interface)
|   |   +-- converter.go        # Markdown <-> Google Docs structural content
|   |   +-- sheets.go           # Control matrix <-> Google Sheets conversion
|   |   +-- revision.go         # Revision tracking and change detection
|   |   +-- ratelimit.go        # Google API rate limiter
|   |   +-- config.go           # Drive sync configuration parsing
|   |   +-- adapter_test.go     # Unit tests
|   |   +-- converter_test.go   # Round-trip fidelity tests
+-- tools/
|   +-- google_workspace.go     # UNCHANGED - existing read-only evidence tool
```

### Adapter Interface

> **NOTE: These interfaces are superseded by the canonical `DataProvider`/`SyncProvider` interfaces defined in SD-004 and ADR-011. This adapter will implement SD-004's `SyncProvider` interface. The interfaces shown here are retained for historical context but will not be implemented as described.**

The `GDriveSyncAdapter` implements the standard adapter interface defined by the hexagonal architecture (ADR-006):

```go
// SyncAdapter defines bidirectional sync with an external system.
// This interface is shared across all integration adapters.
type SyncAdapter interface {
    // Import pulls changes from the external system into the master index.
    Import(ctx context.Context, opts ImportOptions) (*SyncResult, error)

    // Export pushes changes from the master index to the external system.
    Export(ctx context.Context, opts ExportOptions) (*SyncResult, error)

    // Sync performs bidirectional reconciliation.
    Sync(ctx context.Context, opts SyncOptions) (*SyncResult, error)

    // Status returns the current sync state for all tracked artifacts.
    Status(ctx context.Context) (*SyncStatus, error)
}
```

### Integration with Existing Code

```
                    +---------------------+
                    |    Master Index      |
                    | (.index/ YAML files) |
                    +----------+----------+
                               |
              +----------------+----------------+
              |                                 |
    +---------v----------+           +----------v---------+
    | GoogleWorkspaceTool |           | GDriveSyncAdapter   |
    | (read-only evidence)|           | (bidirectional sync) |
    | internal/tools/     |           | internal/adapters/   |
    +---------------------+           +---------------------+
              |                                 |
              v                                 v
    +-------------------+           +--------------------+
    | Google Workspace   |           | Google Drive/Docs/  |
    | APIs (read scopes) |           | Sheets APIs (r/w)   |
    +-------------------+           +--------------------+
```

The two components share the same Google Cloud project and service account credentials but use different OAuth scopes. The existing tool uses read-only scopes; the sync adapter requires read-write scopes for Drive, Docs, and Sheets.

## Data Model Mapping

### Policy to Google Doc

| GRCTool Field | Google Doc Representation |
|---------------|--------------------------|
| Policy title | Document title |
| Policy body (Markdown) | Docs API structural content (paragraphs, headings, lists, tables) |
| Policy metadata (ID, status, version) | Document properties (custom properties on the Doc) |
| Template variables | Interpolated before publishing (e.g., `{{organization.name}}` resolved) |
| Last modified timestamp | Tracked in master index sync state, compared against Drive revision |

Markdown elements and their Docs API mappings:

| Markdown Element | Docs API Element |
|------------------|------------------|
| `# Heading 1` | `Paragraph` with `HEADING_1` named style |
| `## Heading 2` | `Paragraph` with `HEADING_2` named style |
| `**bold**` | `TextRun` with `bold: true` |
| `*italic*` | `TextRun` with `italic: true` |
| `` `code` `` | `TextRun` with `weightedFontFamily: "Courier New"` |
| `- list item` | `Paragraph` with `bullet` preset |
| `1. ordered` | `Paragraph` with `numberedList` preset |
| `| table |` | `Table` element with `TableRow` and `TableCell` children |
| `> blockquote` | `Paragraph` with increased `indentStart` |
| `---` | `SectionBreak` or `HorizontalRule` |
| Code blocks | `Paragraph` with monospace font and background shading |

### Control Matrix to Google Sheet

Controls are exported as a Google Sheet with the following structure:

| Column | Source | Notes |
|--------|--------|-------|
| A: Control ID | `control.grctool_id` | e.g., CC6.8 |
| B: Title | `control.title` | Short description |
| C: Description | `control.description` | Full control text |
| D: Status | `control.status` | Implemented / Partial / Not Implemented |
| E: Mapped Policies | `control.policy_ids[]` | Comma-separated policy IDs |
| F: Evidence Tasks | `control.evidence_task_ids[]` | Comma-separated ET IDs |
| G+: Framework columns | `control.framework_mappings` | One column per framework (SOC2, ISO27001, etc.) |

Conditional formatting rules:
- Status "Implemented" -- green background (#C6EFCE)
- Status "Partially Implemented" -- yellow background (#FFEB9C)
- Status "Not Implemented" -- red background (#FFC7CE)

### Evidence Task List to Google Sheet

| Column | Source |
|--------|--------|
| A: Task ID | `evidence_task.grctool_id` |
| B: Name | `evidence_task.name` |
| C: Status | `evidence_task.lifecycle_state` |
| D: Assignee | `evidence_task.assignee` |
| E: Due Date | `evidence_task.due_date` |
| F: Mapped Controls | `evidence_task.control_ids[]` |
| G: Last Evidence Date | `evidence_task.last_evidence_generated` |
| H: Automation | `evidence_task.automation_capability` |

## Drive Folder Structure

The sync adapter creates a folder hierarchy in Google Drive mirroring GRCTool's data layout:

```
{root_folder}/                              # Configurable root folder ID
+-- Policies/                               # Published policy documents
|   +-- POL-0001 - Access Control Policy    # Google Doc
|   +-- POL-0002 - Data Protection Policy   # Google Doc
|   +-- ...
+-- Controls/                               # Control matrix sheets
|   +-- Control Matrix                      # Google Sheet (all controls)
|   +-- SOC2 Controls                       # Google Sheet (SOC2-filtered view)
+-- Evidence/                               # Evidence task tracking
|   +-- Evidence Task Tracker               # Google Sheet (all tasks)
+-- Auditor/                                # Read-only auditor access folder
|   +-- (symlinks or copies of published artifacts for external sharing)
```

Folder and document IDs are stored in the master index integrations registry:

> **NOTE: Per SD-004 and ADR-011, the master index is implemented as the existing `StorageService` enriched with `ExternalIDs` and `SyncMetadata` fields. No separate `.index/` directory is needed. The mapping data shown below will be stored as part of entity `SyncMetadata` and provider-specific configuration.**

```yaml
# .index/integrations.yaml
gdrive:
  root_folder_id: "1ABC...XYZ"
  mappings:
    POL-0001:
      doc_id: "1DEF...UVW"
      last_synced_revision: "rev-47"
      last_sync_time: "2026-03-17T10:30:00Z"
      sync_direction: "bidirectional"
    POL-0002:
      doc_id: "1GHI...RST"
      last_synced_revision: "rev-12"
      last_sync_time: "2026-03-17T10:30:00Z"
      sync_direction: "outbound_only"
  sheets:
    control_matrix:
      sheet_id: "1JKL...OPQ"
      last_sync_time: "2026-03-17T10:30:00Z"
    evidence_tasks:
      sheet_id: "1MNO...LMN"
      last_sync_time: "2026-03-17T10:30:00Z"
```

## Sync Algorithm

### Change Detection

The sync adapter uses a dual-tracking approach to detect changes:

**Remote (Google Drive) changes:**
- Google Drive API `revisions.list` returns revision history for each document.
- Each sync stores the `revisionId` of the last-synced version.
- On the next sync, the adapter compares the current head revision against the stored revision.
- If they differ, the remote document has changed.

**Local (GRCTool) changes:**
- Local policy files are tracked by content hash (SHA-256 of the Markdown content).
- Each sync stores the hash of the content that was last exported.
- On the next sync, the adapter computes the current hash and compares.
- If they differ, the local file has changed.

### Sync States

For each tracked artifact, the sync algorithm determines one of four states:

| Local Changed | Remote Changed | State | Action |
|---------------|----------------|-------|--------|
| No | No | `in_sync` | No action |
| Yes | No | `local_ahead` | Export local changes to Drive |
| No | Yes | `remote_ahead` | Import remote changes to local |
| Yes | Yes | `conflict` | Apply conflict resolution policy |

### Sync Cycle

A full bidirectional sync cycle proceeds as follows:

```
1. Load sync state from master index (.index/integrations.yaml)
2. For each tracked artifact:
   a. Compute local content hash
   b. Fetch current Drive revision ID via Drive API
   c. Compare against stored state to determine sync state
   d. If in_sync: skip
   e. If local_ahead: convert Markdown -> Docs API content, update Doc
   f. If remote_ahead: fetch Doc content, convert Docs -> Markdown, write local file
   g. If conflict: apply conflict resolution policy (see below)
   h. Update stored sync state (hash, revision ID, timestamp)
3. Sync control matrix Sheet (outbound only; Sheets are not editable externally)
4. Sync evidence task Sheet (outbound only)
5. Write updated sync state to master index
6. Report sync summary (synced, conflicts, errors)
```

### Conflict Resolution

Conflict resolution follows the policy configured in `.grctool.yaml`:

| Policy | Behavior |
|--------|----------|
| `manual` (default) | Halt sync for the conflicted artifact; write both versions to a `.conflicts/` directory with diff; require user to resolve via `grctool drive resolve` |
| `local_wins` | Overwrite the Drive document with the local version |
| `remote_wins` | Overwrite the local file with the Drive document content |
| `newest_wins` | Compare local file mtime against Drive revision timestamp; keep the newer version |

Conflict artifacts written to `.conflicts/`:

```
.conflicts/
+-- POL-0001/
|   +-- local.md            # Local version
|   +-- remote.md           # Remote version (converted from Docs)
|   +-- diff.patch          # Unified diff between local and remote
|   +-- metadata.yaml       # Conflict metadata (timestamps, revisions)
```

## Configuration

The sync adapter is configured in `.grctool.yaml` under the `gdrive` key:

> **NOTE: The `gdrive` config key proposed below is a schema extension to `.grctool.yaml`. Config schema extension is tracked as a separate implementation task and will be validated against the existing `Config` struct in `internal/config/config.go`.**

```yaml
gdrive:
  enabled: false                    # Must be explicitly enabled
  root_folder_id: "1ABC...XYZ"     # Google Drive folder ID for sync root
  credentials_path: ""              # Overrides GOOGLE_APPLICATION_CREDENTIALS if set

  sync:
    direction: "bidirectional"      # bidirectional | outbound_only | inbound_only
    conflict_policy: "manual"       # manual | local_wins | remote_wins | newest_wins
    schedule: ""                    # Cron expression for scheduled sync (requires FEAT-003)

  scope:
    policies:
      include: ["*"]               # Glob patterns for policy IDs to sync
      exclude: ["POL-DRAFT-*"]     # Glob patterns to exclude
      tags_include: []             # Sync only policies with these tags
      tags_exclude: ["draft", "internal"]  # Exclude policies with these tags
    controls:
      enabled: true                # Export control matrix to Sheets
    evidence_tasks:
      enabled: true                # Export evidence task tracker to Sheets

  auditor:
    enabled: false                 # Enable auditor folder
    folder_id: ""                  # Separate folder ID for auditor access
    share_with: []                 # Email addresses to share with (Viewer)
    exclude_tags: ["draft", "internal", "sensitive"]

  rate_limit:
    drive_requests_per_day: 10000  # Below the 12,000/day quota (safety margin)
    docs_requests_per_minute: 250  # Below the 300/min quota (safety margin)
    sheets_requests_per_minute: 50 # Conservative limit for Sheets API
    backoff_base_ms: 1000          # Base delay for exponential backoff
    backoff_max_ms: 60000          # Maximum backoff delay
```

## Authentication

The sync adapter reuses the existing service account authentication pattern from `google_workspace.go`. The key difference is the OAuth scopes required:

| Component | Scopes |
|-----------|--------|
| GoogleWorkspaceTool (existing, read-only) | `drive.readonly`, `documents.readonly`, `spreadsheets.readonly`, `forms.responses.readonly` |
| GDriveSyncAdapter (new, read-write) | `drive`, `documents`, `spreadsheets` |

The adapter initializes Google API clients using the same credential loading pattern:

```go
func (a *GDriveSyncAdapter) initClients(ctx context.Context) error {
    credPath := a.config.GDrive.CredentialsPath
    if credPath == "" {
        credPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
    }

    credJSON, err := os.ReadFile(credPath)
    if err != nil {
        return fmt.Errorf("read credentials: %w", err)
    }

    // Read-write scopes for sync adapter
    scopes := []string{
        "https://www.googleapis.com/auth/drive",
        "https://www.googleapis.com/auth/documents",
        "https://www.googleapis.com/auth/spreadsheets",
    }

    conf, err := google.JWTConfigFromJSON(credJSON, scopes...)
    if err != nil {
        return fmt.Errorf("parse credentials: %w", err)
    }

    httpClient := conf.Client(ctx)
    // Initialize Drive, Docs, Sheets services from httpClient...
}
```

Existing installations that only use the read-only tool are unaffected. The sync adapter's expanded scopes are only requested when `gdrive.enabled: true` is set in configuration.

## Format Conversion: Markdown to Google Docs API

### Conversion Strategy

The converter (`converter.go`) operates on the Google Docs API v1 structural content model. Rather than using an intermediate HTML representation (which loses fidelity), the converter parses Markdown directly into Docs API `Request` objects for `batchUpdate`.

The conversion pipeline:

```
Markdown source
    |
    v
Goldmark AST (Go Markdown parser)
    |
    v
Docs API Request[] (InsertText, UpdateParagraphStyle, etc.)
    |
    v
docs.BatchUpdateDocumentRequest
```

For inbound conversion (Docs to Markdown):

```
docs.Document (structural content)
    |
    v
Walk StructuralElement tree
    |
    v
Emit Markdown tokens based on paragraph style and text formatting
    |
    v
Markdown source
```

### Supported Markdown Subset

The initial implementation supports a defined subset to ensure reliable round-trip fidelity:

| Feature | Outbound (MD -> Docs) | Inbound (Docs -> MD) | Round-trip |
|---------|----------------------|---------------------|------------|
| Headings (H1-H6) | Yes | Yes | Yes |
| Bold, italic | Yes | Yes | Yes |
| Inline code | Yes | Yes | Yes |
| Unordered lists | Yes | Yes | Yes |
| Ordered lists | Yes | Yes | Yes |
| Tables | Yes | Yes | Partial (formatting may simplify) |
| Code blocks | Yes | Yes | Partial (language hint may be lost) |
| Blockquotes | Yes | Yes | Yes |
| Horizontal rules | Yes | Yes | Yes |
| Links | Yes | Yes | Yes |
| Images | No (v1) | No (v1) | No |
| Nested lists (> 2 levels) | Yes | Partial | Partial |

Items marked "Partial" or "No" are documented as known limitations. Round-trip fidelity tests validate the "Yes" items.

## Rate Limiting

The adapter implements a token-bucket rate limiter that respects Google API quotas:

| API | Quota | Adapter Limit (safety margin) |
|-----|-------|-------------------------------|
| Google Drive API | 12,000 requests/day | 10,000 requests/day |
| Google Docs API | 300 requests/minute | 250 requests/minute |
| Google Sheets API | 100 requests/minute | 50 requests/minute |

Implementation details:
- Token bucket per API with configurable refill rate.
- Exponential backoff with jitter when a `429 Too Many Requests` response is received.
- Backoff starts at `backoff_base_ms` (default 1s) and caps at `backoff_max_ms` (default 60s).
- Sync progress is reported to the user during throttled operations so long-running syncs are not mistaken for hangs.
- Daily quota tracking persists across sync runs via `.cache/gdrive_quota.yaml`.

## Scheduler Integration

The sync adapter exposes a `SchedulableTask` that hooks into the task scheduler (FEAT-003):

```go
// GDriveSyncTask implements the SchedulableTask interface for
// periodic Drive synchronization.
type GDriveSyncTask struct {
    adapter *GDriveSyncAdapter
}

func (t *GDriveSyncTask) Name() string { return "gdrive-sync" }
func (t *GDriveSyncTask) Schedule() string { return t.adapter.config.GDrive.Sync.Schedule }
func (t *GDriveSyncTask) Execute(ctx context.Context) error {
    _, err := t.adapter.Sync(ctx, SyncOptions{})
    return err
}
```

When FEAT-003 is not yet available, the sync runs on-demand only via `grctool drive sync`. The cron schedule in configuration is stored but inactive until the scheduler is implemented.

## CLI Commands

The adapter adds the following commands under `grctool drive`:

| Command | Description |
|---------|-------------|
| `grctool drive sync` | Full bidirectional sync (or as configured by `sync.direction`) |
| `grctool drive sync --direction outbound` | Export local changes to Drive only |
| `grctool drive sync --direction inbound` | Import Drive changes to local only |
| `grctool drive sync --dry-run` | Preview sync actions without making changes |
| `grctool drive status` | Show sync state for all tracked artifacts |
| `grctool drive resolve POL-0001` | Resolve a conflict for a specific artifact |
| `grctool drive resolve --all --policy local_wins` | Bulk resolve all conflicts |
| `grctool drive init` | Create Drive folder structure and initial export |

## Error Handling

| Error Scenario | Behavior |
|----------------|----------|
| Credentials missing or invalid | Clear error message directing user to setup guide; sync aborts |
| Drive folder not found | Error with folder ID; suggest running `grctool drive init` |
| Document permission denied | Skip document, log warning, continue sync for remaining artifacts |
| Rate limit exceeded (429) | Exponential backoff with jitter; retry up to 5 times; then skip and report |
| Network error mid-sync | Save partial sync state; next run resumes from last successful artifact |
| Docs API content conversion failure | Skip artifact, log error with details, continue sync |
| Conflict detected | Follow configured conflict policy; default to `manual` (halt and write conflict artifacts) |

## Testing Strategy

| Test Type | Coverage | Approach |
|-----------|----------|----------|
| Unit tests | Markdown <-> Docs conversion, sync state logic, conflict detection | Standard Go tests with table-driven cases |
| Round-trip fidelity | Markdown -> Docs -> Markdown produces equivalent output | Golden file tests with representative policy documents |
| Integration tests | Full sync cycle against Google APIs | VCR cassettes (ADR-005) recording real API interactions |
| Rate limit tests | Backoff and quota enforcement | Mock HTTP client with 429 responses |
| Regression tests | Existing GoogleWorkspaceTool unchanged | Existing test suite must pass without modification |

## Implementation Phases

| Phase | Scope | Milestone |
|-------|-------|-----------|
| Phase 1: Outbound export | Publish policies as Docs, controls as Sheets; one-way push from GRCTool to Drive | MVP: `grctool drive sync --direction outbound` |
| Phase 2: Change detection | Detect remote changes via revision tracking; detect local changes via content hashing | Sync status reporting: `grctool drive status` |
| Phase 3: Inbound sync | Import Docs changes back to local Markdown; conflict detection and resolution | Full bidirectional: `grctool drive sync` |
| Phase 4: Auditor folder | Configurable read-only folder with selective sharing | Auditor access: `grctool drive sync` populates auditor folder |
| Phase 5: Scheduler | Hook into FEAT-003 task scheduler for cron-based sync | Automated sync on schedule |

## References

- [FEAT-002: Google Drive Bidirectional Sync](/home/erik/Projects/grctool/docs/helix/01-frame/features/FEAT-002-gdrive-bidirectional-sync.md)
- [ADR-010: System of Record Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-010-system-of-record-architecture)
- [ADR-006: Hexagonal Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-006-hexagonal-architecture-ports-and-adapters)
- [Data Design](/home/erik/Projects/grctool/docs/helix/02-design/data-design/data-design.md)
- [Google Workspace Tool](/home/erik/Projects/grctool/internal/tools/google_workspace.go)
- [Google Workspace Setup Guide](/home/erik/Projects/grctool/docs/01-User-Guide/google-workspace-setup.md)
- [Google Docs API Reference](https://developers.google.com/docs/api/reference/rest)
- [Google Drive API Reference](https://developers.google.com/drive/api/reference/rest/v3)
- [Google Sheets API Reference](https://developers.google.com/sheets/api/reference/rest)
