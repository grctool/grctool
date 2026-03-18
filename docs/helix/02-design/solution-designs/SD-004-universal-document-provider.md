---
title: "SD-004: Universal Document Provider Framework"
phase: "02-design"
category: "solution-design"
tags: ["provider", "adapter", "multi-source", "domain-model", "sync", "hexagonal-architecture"]
related: ["adr-006", "adr-010", "adr-011", "SD-001", "data-design"]
created: 2026-03-17
updated: 2026-03-17
---

# SD-004: Universal Document Provider Framework

## Overview

This document describes the solution design for the Universal Document Provider Framework in GRCTool. The framework introduces a pluggable provider abstraction for compliance data sources, fixes inconsistent domain model ID types, adds external ID tracking for multi-provider scenarios, and refactors the existing Tugboat integration as the first provider implementation. It implements the system-of-record vision from ADR-010 and the provider framework decision from ADR-011.

---

## Architecture

### Hexagonal Placement

The provider framework introduces new **port interfaces** (`DataProvider`, `SyncProvider`) in the domain layer. Each external compliance platform implements these ports as infrastructure adapters. The `ProviderRegistry` is a domain service that manages provider lifecycle. The existing `StorageService` remains the canonical local store (master index).

```
                    +------------------------------------+
                    |          Domain Core               |
                    |                                    |
                    |  DataProvider (port interface)      |
                    |  SyncProvider (port interface)      |
                    |  ProviderRegistry (domain service)  |
                    |  StorageService (master index)      |
                    |  SyncOrchestrator                  |
                    |                                    |
                    +------+----------+----------+-------+
                           |          |          |
                    [Port] |   [Port] |   [Port] |
                           |          |          |
              +------------+--+  +----+----+  +--+-----------+
              | DataProvider  |  | Storage  |  | SyncProvider |
              | (interface)   |  | Service  |  | (interface)  |
              +------+--------+  +---------+  +------+-------+
                     |                               |
          +----------+----------+         +----------+-----------+
          | TugboatDataProvider |         | Future: AccountableHQ|
          | (proof of concept)  |         | SyncProvider         |
          +---------------------+         +----------------------+
                     |                               |
                     v                               v
              [Tugboat Logic API]          [AccountableHQ API]
```

---

## Domain Model Changes

### ID Type Unification

The core breaking change: all domain entity IDs become `string`.

**Current state** (from `internal/domain/`):

| Entity | Field | Current Type | New Type |
|--------|-------|-------------|----------|
| `Policy` | `ID` | `string` | `string` (no change) |
| `Control` | `ID` | `int` | `string` |
| `EvidenceTask` | `ID` | `int` | `string` |
| `EvidenceRecord` | `TaskID` | `int` | `string` |

**Changes to `internal/domain/control.go`:**

```go
// Before
type Control struct {
    ID                int        `json:"id"`
    // ...
}

// After
type Control struct {
    ID                string     `json:"id"`
    ExternalIDs       map[string]string  `json:"external_ids,omitempty"`
    SyncMetadata      *SyncMetadata      `json:"sync_metadata,omitempty"`
    // ... (all other fields unchanged)
}
```

**Changes to `internal/domain/evidence.go`:**

```go
// Before
type EvidenceTask struct {
    ID                 int        `json:"id"`
    // ...
}

// After
type EvidenceTask struct {
    ID                 string     `json:"id"`
    ExternalIDs        map[string]string  `json:"external_ids,omitempty"`
    SyncMetadata       *SyncMetadata      `json:"sync_metadata,omitempty"`
    // ... (all other fields unchanged)
}

// Before
type EvidenceRecord struct {
    // ...
    TaskID      int                    `json:"task_id"`
    // ...
}

// After
type EvidenceRecord struct {
    // ...
    TaskID      string                 `json:"task_id"`
    // ...
}
```

**Changes to `internal/domain/policy.go`:**

```go
// Before
type Policy struct {
    ID          string    `json:"id"`
    // ...
}

// After
type Policy struct {
    ID           string                `json:"id"`
    ExternalIDs  map[string]string     `json:"external_ids,omitempty"`
    SyncMetadata *SyncMetadata         `json:"sync_metadata,omitempty"`
    // ... (all other fields unchanged)
}
```

### New SyncMetadata Struct

Add to `internal/domain/sync_metadata.go` (new file):

```go
package domain

import (
    "time"
)

// SyncMetadata tracks synchronization state per provider for a domain entity.
type SyncMetadata struct {
    // LastSyncTime records the last successful sync time per provider.
    // Key is provider name (e.g., "tugboat", "accountablehq").
    LastSyncTime map[string]time.Time `json:"last_sync_time,omitempty"`

    // ContentHash stores a hash of the entity content per provider,
    // used for change detection without fetching full content.
    ContentHash map[string]string `json:"content_hash,omitempty"`

    // ConflictState indicates whether this entity has unresolved conflicts.
    // Empty string means no conflict.
    ConflictState string `json:"conflict_state,omitempty"`
}

// ConflictState constants
const (
    ConflictStateNone          = ""
    ConflictStateLocalModified = "local_modified"
    ConflictStateRemoteModified = "remote_modified"
    ConflictStateBothModified  = "both_modified"
    ConflictStateResolved      = "resolved"
)
```

---

## Interface Definitions

**Target state.** These interfaces define the design target for `internal/interfaces/provider.go`. The file already exists with `DataProvider`, `SyncProvider`, `EvidenceSubmitter`, and `RelationshipQuerier` partially implemented. Methods marked **(NEW)** below do not yet exist in code and must be added.

### DataProvider

```go
package interfaces

import (
    "context"
    "time"

    "github.com/grctool/grctool/internal/domain"
)

// ListOptions controls pagination and filtering for list operations.
// Fields map to what the Tugboat API actually supports (page-based pagination,
// framework filtering). Other providers may ignore unsupported fields.
type ListOptions struct {
    Page      int    `json:"page,omitempty"`
    PageSize  int    `json:"page_size,omitempty"`
    Framework string `json:"framework,omitempty"`
    Status    string `json:"status,omitempty"`
    Category  string `json:"category,omitempty"`
}

// ProviderCapabilities reports which entity types a provider supports
// and whether it offers read-only or read-write access.
// Callers use this to skip entity types a provider cannot serve
// (e.g., AccountableHQ supports policies only).
type ProviderCapabilities struct {
    SupportsPolicies      bool `json:"supports_policies"`
    SupportsControls      bool `json:"supports_controls"`
    SupportsEvidenceTasks bool `json:"supports_evidence_tasks"`
    SupportsWrite         bool `json:"supports_write"`
    SupportsChangeDetect  bool `json:"supports_change_detect"`
}

// DataProvider defines read-only access to compliance entities from an external source.
// This is the minimal interface that every compliance platform must implement.
type DataProvider interface {
    // Name returns the unique identifier for this provider (e.g., "tugboat", "accountablehq").
    Name() string

    // (NEW) Capabilities reports which entity types and operations this provider supports.
    // Callers must check capabilities before calling entity-specific methods;
    // calling an unsupported method returns an error.
    Capabilities() ProviderCapabilities

    // TestConnection verifies that the provider is reachable and authenticated.
    TestConnection(ctx context.Context) error

    // ListPolicies returns a page of policies and the total count.
    ListPolicies(ctx context.Context, opts ListOptions) ([]domain.Policy, int, error)

    // ListControls returns a page of controls and the total count.
    ListControls(ctx context.Context, opts ListOptions) ([]domain.Control, int, error)

    // ListEvidenceTasks returns a page of evidence tasks and the total count.
    ListEvidenceTasks(ctx context.Context, opts ListOptions) ([]domain.EvidenceTask, int, error)

    // GetPolicy retrieves a single policy by its provider-native ID.
    GetPolicy(ctx context.Context, id string) (*domain.Policy, error)

    // GetControl retrieves a single control by its provider-native ID.
    GetControl(ctx context.Context, id string) (*domain.Control, error)

    // GetEvidenceTask retrieves a single evidence task by its provider-native ID.
    GetEvidenceTask(ctx context.Context, id string) (*domain.EvidenceTask, error)
}
```

### SyncProvider

```go
// SyncProvider extends DataProvider with bidirectional sync capabilities.
// Not all providers need to implement this — read-only providers implement
// DataProvider only.
type SyncProvider interface {
    DataProvider

    // PushPolicy creates or updates a policy in the remote system.
    PushPolicy(ctx context.Context, policy *domain.Policy) error

    // PushControl creates or updates a control in the remote system.
    PushControl(ctx context.Context, control *domain.Control) error

    // PushEvidenceTask creates or updates an evidence task in the remote system.
    PushEvidenceTask(ctx context.Context, task *domain.EvidenceTask) error

    // DeletePolicy removes a policy from the remote system.
    DeletePolicy(ctx context.Context, id string) error

    // DeleteControl removes a control from the remote system.
    DeleteControl(ctx context.Context, id string) error

    // DeleteEvidenceTask removes an evidence task from the remote system.
    DeleteEvidenceTask(ctx context.Context, id string) error

    // DetectChanges returns entities that changed since the given time.
    DetectChanges(ctx context.Context, since time.Time) (*ChangeSet, error)

    // (NEW) ResolveConflict applies a conflict resolution decision to an entity.
    // Strategies: LocalWins, RemoteWins, NewestWins, Manual.
    // Manual strategy marks the conflict as requiring human review.
    ResolveConflict(ctx context.Context, conflict Conflict, resolution ConflictResolution) error
}

// ConflictResolution enumerates strategies for resolving sync conflicts.
type ConflictResolution string

const (
    ConflictResolutionLocalWins  ConflictResolution = "local_wins"
    ConflictResolutionRemoteWins ConflictResolution = "remote_wins"
    ConflictResolutionNewestWins ConflictResolution = "newest_wins"
    ConflictResolutionManual     ConflictResolution = "manual"
)

// Conflict represents a detected conflict between local and remote state.
type Conflict struct {
    EntityType string    `json:"entity_type"` // "policy", "control", "evidence_task"
    EntityID   string    `json:"entity_id"`   // GRCTool-native ID
    Provider   string    `json:"provider"`
    LocalHash  string    `json:"local_hash"`
    RemoteHash string    `json:"remote_hash"`
    DetectedAt time.Time `json:"detected_at"`
}
```

### Optional Capability Interfaces

Not all providers support evidence submission or relationship queries. These
capabilities are expressed as separate interfaces. Callers must type-assert
before use:

```go
if es, ok := provider.(EvidenceSubmitter); ok {
    err := es.SubmitEvidence(ctx, taskID, file, meta)
}
```

#### EvidenceSubmitter

```go
// EvidenceSubmitter is an optional interface for providers that support
// uploading evidence and managing attachments. Only providers connected
// to platforms with evidence intake (e.g., Tugboat custom collectors)
// implement this interface.
type EvidenceSubmitter interface {
    // SubmitEvidence uploads evidence for a task.
    SubmitEvidence(ctx context.Context, taskID string, file io.Reader, meta SubmissionMetadata) error

    // ListAttachments returns evidence attachments for a task.
    ListAttachments(ctx context.Context, taskID string, opts ListOptions) ([]Attachment, int, error)

    // DownloadAttachment returns a reader for an attachment's content.
    DownloadAttachment(ctx context.Context, attachmentID string) (io.ReadCloser, string, error)
}

// SubmissionMetadata provides context for an evidence submission.
type SubmissionMetadata struct {
    CollectedDate string            `json:"collected_date"`
    Notes         string            `json:"notes,omitempty"`
    Window        string            `json:"window,omitempty"`
    ContentType   string            `json:"content_type,omitempty"`
    Filename      string            `json:"filename,omitempty"`
    Metadata      map[string]string `json:"metadata,omitempty"`
}

// Attachment represents an evidence attachment in an upstream platform.
type Attachment struct {
    ID            string `json:"id"`
    TaskID        string `json:"task_id"`
    Filename      string `json:"filename"`
    MimeType      string `json:"mime_type"`
    CollectedDate string `json:"collected_date"`
}
```

#### RelationshipQuerier

```go
// RelationshipQuerier is an optional interface for providers that support
// cross-entity relationship queries. Not all providers have relationship
// data (e.g., AccountableHQ manages policies only).
type RelationshipQuerier interface {
    // GetEvidenceTasksByControl returns evidence tasks linked to a control.
    GetEvidenceTasksByControl(ctx context.Context, controlID string) ([]domain.EvidenceTask, error)

    // GetControlsByPolicy returns controls implementing a policy.
    GetControlsByPolicy(ctx context.Context, policyID string) ([]domain.Control, error)

    // GetPoliciesByControl returns policies that a control implements.
    GetPoliciesByControl(ctx context.Context, controlID string) ([]domain.Policy, error)
}
```

### ChangeSet

```go
// ChangeSet represents entities that changed in a remote provider since a given time.
// Matches the existing implementation in internal/interfaces/provider.go.
type ChangeSet struct {
    Provider   string        `json:"provider"`
    Since      time.Time     `json:"since"`
    DetectedAt time.Time     `json:"detected_at"`
    Changes    []ChangeEntry `json:"changes,omitempty"`
}

// ChangeEntry represents a single changed entity.
type ChangeEntry struct {
    EntityType string    `json:"entity_type"` // "policy", "control", "evidence_task"
    EntityID   string    `json:"entity_id"`
    ChangeType string    `json:"change_type"` // "created", "updated", "deleted"
    Hash       string    `json:"hash,omitempty"`
    ModifiedAt time.Time `json:"modified_at"`
}
```

### ProviderRegistry

```go
// ProviderInfo summarizes a registered provider's state.
// (NEW) Not yet in code.
type ProviderInfo struct {
    Name         string               `json:"name"`
    Capabilities ProviderCapabilities `json:"capabilities"`
    Healthy      bool                 `json:"healthy"`
    LastSyncTime *time.Time           `json:"last_sync_time,omitempty"`
}
```

### ProviderRegistry

The registry is currently a concrete struct in `internal/providers/registry.go`.
**Target**: Extract a `ProviderRegistry` interface into `internal/interfaces/provider.go`
so that services depend on the interface, not the concrete type.

**Current signatures** (in `internal/providers/registry.go`):

```go
type ProviderRegistry struct { /* ... */ }

func NewProviderRegistry() *ProviderRegistry
func (r *ProviderRegistry) Register(provider interfaces.DataProvider) error
func (r *ProviderRegistry) Get(name string) (interfaces.DataProvider, error)
func (r *ProviderRegistry) GetSyncProvider(name string) (interfaces.SyncProvider, error)
func (r *ProviderRegistry) List() []string
func (r *ProviderRegistry) ListSyncProviders() []string
func (r *ProviderRegistry) Remove(name string)
func (r *ProviderRegistry) Count() int
func (r *ProviderRegistry) Has(name string) bool
func (r *ProviderRegistry) HealthCheck(ctx context.Context) map[string]error
```

---

## ProviderRegistry Implementation

The registry is implemented in `internal/providers/registry.go` as a concrete struct.
See the current signatures above. The implementation is thread-safe (uses `sync.RWMutex`)
and snapshots the provider map before calling `TestConnection` in `HealthCheck` to avoid
holding the read lock during potentially slow network calls.

---

## Tugboat Refactoring Plan

### Current Architecture

```
SyncService (internal/services/sync.go)
    ├── *tugboat.Client        (direct dependency)
    ├── *adapters.TugboatToDomain  (direct dependency)
    ├── *storage.Storage       (direct dependency)
    └── orchestration methods: syncPolicies(), syncControls(), syncEvidenceTasks()
```

### Target Architecture

```
SyncService (internal/services/sync.go)
    ├── *providers.ProviderRegistry  (abstracted dependency)
    ├── *storage.Storage             (storage dependency)
    └── orchestration methods: syncPoliciesFromProvider(), syncControlsFromProvider(), ...
```

### Existing File: `internal/providers/tugboat/provider.go`

`TugboatDataProvider` **already exists** and implements `DataProvider`, `EvidenceSubmitter`, and `RelationshipQuerier`. Constructor signature:

```go
func NewTugboatDataProvider(
    client *tugboat.Client,
    adapter *adapters.TugboatToDomain,
    orgID string,
    log logger.Logger,
) *TugboatDataProvider
```

The key change in each method: after converting via `p.adapter.ConvertControl(apiControl)`, the provider sets `ExternalIDs = map[string]string{"tugboat": strconv.Itoa(originalID)}` to preserve the Tugboat numeric ID as an external reference, while the domain `ID` field becomes the string representation.

### Files Modified for Tugboat Refactoring

| File | Change |
|------|--------|
| `internal/adapters/tugboat_provider.go` | **New file** — TugboatDataProvider implementing DataProvider |
| `internal/adapters/tugboat.go` | No changes — existing adapter remains as-is, used internally by TugboatDataProvider |
| `internal/services/sync.go` | Replace `*tugboat.Client` + `*adapters.TugboatToDomain` fields with `interfaces.ProviderRegistry`; refactor `syncPolicies()` etc. to call provider methods |
| `internal/services/sync.go` → `NewSyncService()` | Accept `ProviderRegistry` instead of `*tugboat.Client`; look up "tugboat" provider from registry |

---

## Migration Path

### Phase 1: ID Type Migration (Breaking Change)

Change `Control.ID` and `EvidenceTask.ID` from `int` to `string`. This is the prerequisite for everything else.

**Adapter changes** (`internal/adapters/tugboat.go`):

```go
// Before (line 137)
return domain.Control{
    ID: c.ID,  // int assigned directly
    ...
}

// After
return domain.Control{
    ID: strconv.Itoa(c.ID),  // converted to string
    ...
}
```

Same pattern for `ConvertEvidenceTask` (line 273):

```go
// Before
domainTask := domain.EvidenceTask{
    ID: task.ID,  // int assigned directly
    ...
}

// After
domainTask := domain.EvidenceTask{
    ID: strconv.Itoa(task.ID),  // converted to string
    ...
}
```

**Storage migration**: Existing JSON files on disk have numeric `"id"` fields for controls and evidence tasks. **NOTE: Go's `encoding/json.Unmarshal` will NOT decode a JSON number into a `string` field -- it returns an error.** A custom `UnmarshalJSON` method or a wrapper type is needed to handle `"id": 12345` → `string("12345")`. The codebase already has an `IntOrString` type in `internal/tugboat/models/types.go` that handles exactly this case (unmarshalling both JSON strings and JSON numbers into a string value). The migration should either: (a) use a similar `IntOrString` approach on the domain `ID` fields during a transition period, or (b) run a one-time data migration tool (`grctool migrate`) that rewrites on-disk JSON files to use string-quoted IDs (e.g., `"id": "12345"`).

### Phase 2: Add ExternalIDs and SyncMetadata Fields

Add the new fields to all three domain structs. Existing JSON files without these fields will deserialize correctly (the fields are `omitempty` and pointer/map types default to nil/empty).

### Phase 3: Introduce Provider Interfaces and Registry

Add `internal/interfaces/provider.go` with `DataProvider`, `SyncProvider`, `ChangeSet`, `ListOptions`, and `ProviderRegistry` interfaces. Add `internal/services/provider_registry.go` with the concrete implementation.

### Phase 4: Implement TugboatDataProvider

Create `internal/adapters/tugboat_provider.go`. This wraps the existing client and adapter — no changes to `internal/tugboat/client.go` or `internal/adapters/tugboat.go`.

### Phase 5: Refactor SyncService

Modify `internal/services/sync.go`:

```go
// Before
type SyncService struct {
    tugboatClient         *tugboat.Client
    adapter               *adapters.TugboatToDomain
    storage               *storage.Storage
    // ...
}

// After
type SyncService struct {
    providerRegistry      interfaces.ProviderRegistry
    storage               *storage.Storage
    // ... (formatters, registry, etc. remain)
}
```

The `SyncAll` method changes from directly calling `s.tugboatClient.GetAllPolicies()` to:

```go
func (s *SyncService) syncPolicies(ctx context.Context, providerName string, stats *SyncStats) error {
    provider := s.providerRegistry.Get(providerName)
    if provider == nil {
        return fmt.Errorf("provider %q not registered", providerName)
    }

    policies, total, err := provider.ListPolicies(ctx, interfaces.ListOptions{})
    if err != nil {
        return fmt.Errorf("failed to list policies from %s: %w", providerName, err)
    }
    stats.Total = total

    for _, policy := range policies {
        // Enrich with detailed data via provider.GetPolicy()
        detailed, err := provider.GetPolicy(ctx, policy.ExternalIDs[providerName])
        if err != nil {
            stats.Errors++
            continue
        }
        if err := s.storage.SavePolicy(detailed); err != nil {
            stats.Errors++
            continue
        }
        stats.Synced++
    }
    return nil
}
```

---

## Master Index

The existing `StorageService` (implemented in `internal/storage/`) **is** the master index. This corrects the over-engineering suggested in ADR-010 which mentioned a separate `.index/` directory.

**Why no separate index is needed:**

1. The `StorageService` already provides `GetAllPolicies()`, `GetAllControls()`, `GetAllEvidenceTasks()` — this is the entity registry.
2. Adding `ExternalIDs` to each entity's JSON file means the storage layer already tracks which providers know about each entity.
3. Adding `SyncMetadata` to each entity's JSON file means the storage layer already tracks sync state per provider.
4. The existing file naming convention (`ET-0001-327992-name.json`) already embeds the Tugboat ID — the `ExternalIDs` field formalizes this.

**What the master index provides (via enriched StorageService):**

- **Entity lookup by GRCTool ID**: `GetPolicy("POL-0001")` — already exists
- **Entity lookup by external ID**: `GetPolicyByExternalID("tugboat", "12345")` — new method on StorageService, implemented as a scan-and-filter (acceptable performance for hundreds of entities per ADR-004)
- **Sync state per provider**: Read `SyncMetadata.LastSyncTime["tugboat"]` from the entity JSON
- **Conflict detection**: Check `SyncMetadata.ConflictState` on each entity
- **Cross-provider entity correlation**: Match entities across providers via `ExternalIDs` map

**New StorageService methods** (added to `internal/interfaces/storage.go`):

```go
// Added to StorageService interface
GetPolicyByExternalID(provider, externalID string) (*domain.Policy, error)
GetControlByExternalID(provider, externalID string) (*domain.Control, error)
GetEvidenceTaskByExternalID(provider, externalID string) (*domain.EvidenceTask, error)
```

---

## Impact Analysis

### Files That Must Change When Control.ID and EvidenceTask.ID Become String

This is the comprehensive list based on grep analysis of the codebase:

**Domain layer** (`internal/domain/`):

| File | Change |
|------|--------|
| `internal/domain/control.go` | `ID int` → `ID string`; add `ExternalIDs`, `SyncMetadata` fields |
| `internal/domain/evidence.go` | `EvidenceTask.ID int` → `string`; `EvidenceRecord.TaskID int` → `string`; add `ExternalIDs`, `SyncMetadata` fields |
| `internal/domain/policy.go` | Add `ExternalIDs`, `SyncMetadata` fields (ID already string) |
| `internal/domain/control_reference_test.go` | Update test fixtures: `ID: 123` → `ID: "123"` |

**Adapters** (`internal/adapters/`):

| File | Change |
|------|--------|
| `internal/adapters/tugboat.go` | `ConvertControl`: `ID: c.ID` → `ID: strconv.Itoa(c.ID)`; `ConvertEvidenceTask`: `ID: task.ID` → `ID: strconv.Itoa(task.ID)`; `convertEmbeddedControl`: `control.ID = int(id)` → `control.ID = strconv.Itoa(int(id))`; `convertEmbeddedEvidenceTask`: same pattern; all `control.ID != 0` checks → `control.ID != ""`; all `task.ID != 0` checks → `task.ID != ""` |
| `internal/adapters/tugboat_test.go` | Update test assertions: numeric ID expectations → string |

**Services** (`internal/services/`):

| File | Change |
|------|--------|
| `internal/services/sync.go` | `strconv.Itoa(apiControl.ID)` already used for API calls (no change there); `logger.Int("control_id", ...)` → `logger.String("control_id", ...)`; `logger.Int("task_id", ...)` → `logger.String("task_id", ...)`; `strconv.Itoa(domainTask.ID)` calls for storage lookups → direct string use |
| `internal/services/evidence.go` | `task.ID` int usage → string |
| `internal/services/evidence_evaluator.go` | `task.ID` int usage → string |
| `internal/services/evidence/service.go` | `task.ID` int usage → string |
| `internal/services/evidence_summaries_test.go` | Test fixtures with numeric IDs → string |
| `internal/services/submission/service.go` | `task.ID` int usage → string |
| `internal/services/validation/service.go` | `task.ID` int usage → string |

**Storage** (`internal/storage/`):

| File | Change |
|------|--------|
| `internal/storage/unified_storage.go` | `GetEvidenceRecordsByTaskID(taskID int)` → `GetEvidenceRecordsByTaskID(taskID string)` |
| `internal/storage/local_store.go` | Same method signature change |

**Interfaces** (`internal/interfaces/`):

| File | Change |
|------|--------|
| `internal/interfaces/storage.go` | `GetEvidenceRecordsByTaskID(taskID int)` → `GetEvidenceRecordsByTaskID(taskID string)` |
| `internal/interfaces/services.go` | `GetEvidenceRecordsByTaskID(taskID int)` → `GetEvidenceRecordsByTaskID(taskID string)` |

**Tools** (`internal/tools/`):

| File | Change |
|------|--------|
| `internal/tools/evidence_task_details.go` | `task.ID` int formatting → string |
| `internal/tools/evidence_relationships.go` | `task.ID` int formatting → string |
| `internal/tools/evidence_generator.go` | `task.ID` int formatting → string |
| `internal/tools/evidence_validator.go` | `task.ID` int formatting → string |
| `internal/tools/evidence_writer.go` | `task.ID` int formatting → string |
| `internal/tools/collection_plan.go` | `task.ID` int formatting → string |
| `internal/tools/prompt_assembler.go` | `task.ID` int formatting → string |
| `internal/tools/policy_summary_generator.go` | `control.ID` int formatting → string |
| `internal/tools/control_summary_generator.go` | `control.ID` int formatting → string |

**Formatters** (`internal/formatters/`):

| File | Change |
|------|--------|
| `internal/formatters/control.go` | `control.ID` int formatting → string |
| `internal/formatters/evidence_task.go` | `task.ID` int formatting → string |
| `internal/formatters/evidence_task_test.go` | Test fixtures with numeric IDs → string |

**Registry** (`internal/registry/`):

| File | Change |
|------|--------|
| `internal/registry/evidence_task_registry.go` | `task.ID` int usage → string |

**Commands** (`cmd/`):

| File | Change |
|------|--------|
| `cmd/control.go` | `control.ID` int formatting → string |
| `cmd/evidence.go` | `task.ID` int formatting → string |
| `cmd/evidence_setup.go` | `task.ID` int formatting → string |
| `cmd/evidence_migrate.go` | `task.ID` int formatting → string |
| `cmd/quick_validation_test.go` | Test fixtures with numeric IDs → string |

**Test helpers** (`test/`):

| File | Change |
|------|--------|
| `test/helpers/builders.go` | Builder functions that set `ID: int` → `ID: string` |
| `test/helpers/builders_test.go` | Test assertions for numeric IDs → string |

**Tugboat client tests** (`internal/tugboat/`):

| File | Change |
|------|--------|
| `internal/tugboat/client_test.go` | Assertions on `control.ID` and `task.ID` types |
| `internal/tugboat/final_relationship_test.go` | Same |

### Total Impact: 33+ files across 10 packages

---

## New Files Summary

| File | Purpose |
|------|---------|
| `internal/domain/sync_metadata.go` | `SyncMetadata` struct and conflict state constants |
| `internal/interfaces/provider.go` | (EXISTS) Add `ProviderCapabilities`, `Capabilities()`, `Conflict`, `ConflictResolution`, `ResolveConflict()`, `ProviderInfo` |
| `internal/providers/registry.go` | (EXISTS) Concrete `ProviderRegistry` — extract interface into `internal/interfaces/` |
| `internal/providers/tugboat/provider.go` | (EXISTS) `TugboatDataProvider` implementing `DataProvider`, `EvidenceSubmitter`, `RelationshipQuerier` |

---

## Testing Strategy

1. **Contract tests for DataProvider**: Table-driven test suite (`DataProviderContractSuite`) verifying `Name()`, `Capabilities()`, `TestConnection()`, and all List/Get methods. Every provider implementation must pass this suite. Existing suite at `internal/providers/contract_test.go` must be extended with `Capabilities()` tests.

2. **Contract tests for EvidenceSubmitter**: Existing suite at `internal/providers/evidence_submitter_contract_test.go` covers `SubmitEvidence()`, `ListAttachments()`, `DownloadAttachment()`. Type-assertion gating already tested.

3. **Contract tests for RelationshipQuerier**: Existing suite at `internal/providers/relationship_contract_test.go` covers cross-entity queries. Type-assertion gating already tested.

4. **Unit tests for TugboatDataProvider**: Use existing VCR cassettes (ADR-005) to verify that the provider returns correctly typed domain entities with `ExternalIDs` populated and `Capabilities()` returns correct flags.

5. **Integration tests for ProviderRegistry**: Register multiple mock providers, verify lookup, health check, `Remove()`, and lifecycle management.

6. **Migration test**: Load existing JSON files with numeric IDs, verify they deserialize correctly into `string` ID fields.

7. **Mock DataProvider for sync tests**: Replace the current `*tugboat.Client` mock pattern with a mock `DataProvider`, simplifying test setup.

8. **Submission routing test**: Verify that `SubmissionService` resolves `EvidenceSubmitter` from the provider registry via type-assertion, not through a direct `tugboat.Client` reference.

---

## Open Questions

1. **Pagination strategy**: The Tugboat API uses page-based pagination. Should `ListOptions` also support cursor-based pagination for providers that prefer it?

2. **Bulk vs. detail fetch**: Current sync fetches a list then detail for each entity. Should `DataProvider` have a `ListDetailedPolicies` method, or should providers internally optimize?

3. **Conflict resolution workflow**: The `SyncProvider.ResolveConflict()` method and `ConflictResolution` strategies are now defined in this document. The remaining open question is the **user-facing workflow**: how does `grctool sync` surface conflicts to the user, and what CLI commands or interactive prompts invoke `ResolveConflict()`? This needs a separate UX design.

---

## Submission Service Routing

### Current State

`SubmissionService` (`internal/services/submission/service.go`) holds a direct `*tugboat.Client` reference and calls `tugboatClient.SubmitEvidence()` to upload evidence. This bypasses the provider abstraction.

### Target State

`SubmissionService` resolves the target provider from the `ProviderRegistry`, type-asserts for `EvidenceSubmitter`, and calls `SubmitEvidence()` through the interface:

```go
type SubmissionService struct {
    registry      *providers.ProviderRegistry
    storage       *storage.Storage
    validator     *validation.EvidenceValidationService
    collectorURLs map[string]string
}

func (s *SubmissionService) Submit(ctx context.Context, taskRef string, providerName string) error {
    provider, err := s.registry.Get(providerName)
    if err != nil {
        return fmt.Errorf("provider %q not registered: %w", providerName, err)
    }
    submitter, ok := provider.(interfaces.EvidenceSubmitter)
    if !ok {
        return fmt.Errorf("provider %q does not support evidence submission", providerName)
    }
    // ... prepare file, metadata ...
    return submitter.SubmitEvidence(ctx, taskID, file, meta)
}
```

The `TugboatDataProvider` already implements `EvidenceSubmitter` with contract tests passing. The wiring change is: replace the `*tugboat.Client` field with `ProviderRegistry`, and resolve the submitter at call time.

### Files Changed

| File | Change |
|------|--------|
| `internal/services/submission/service.go` | Replace `tugboatClient *tugboat.Client` with `registry ProviderRegistry`; resolve submitter via type-assertion |
| `cmd/evidence.go` | Pass registry to `NewSubmissionService()` instead of `tugboat.Client` |

---

## Tool Client Abstraction

### Current State

GitHub tools (`internal/tools/github/client.go`) and the Tugboat sync wrapper each independently construct auth providers and HTTP clients. Six GitHub tools instantiate their own `GitHubAuthProvider`. VCR transport wrapping happens per-client. VCR tests are disabled (`//go:build disabled`) because the config plumbing changed.

### Target State

Auth providers and HTTP clients are constructed once during tool initialization (`internal/tools/registry_init.go`) and injected into tools:

```go
// In registry_init.go
func InitializeToolRegistry(cfg *config.Config, log logger.Logger) {
    // Construct shared auth providers once
    githubAuth := auth.NewGitHubAuthProvider(cfg)
    tugboatAuth := auth.NewTugboatAuthProvider(cfg)

    // Construct shared HTTP clients with VCR transport
    githubClient := github.NewGitHubClient(cfg, log, githubAuth)

    // Register tools with injected dependencies
    RegisterTool(github.NewPermissionsTool(githubClient, log))
    RegisterTool(github.NewWorkflowsTool(githubClient, log))
    // ...
}
```

This enables:
- VCR recording for all GitHub tool calls (single transport wrapping point)
- `auth status` can report GitHub auth by querying the shared provider
- Tools no longer reach into config or environment for credentials

### Files Changed

| File | Change |
|------|--------|
| `internal/tools/registry_init.go` | Construct shared auth providers and HTTP clients; pass to tool constructors |
| `internal/tools/github/client.go` | Accept auth provider and HTTP client as constructor params instead of building internally |
| `internal/tools/github_*.go` (6 files) | Remove per-tool auth provider construction; accept shared client |
| `cmd/auth.go` | Query shared auth providers for `auth status` instead of checking config fields directly |

---

## References

- [ADR-006: Hexagonal Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-006)
- [ADR-010: System of Record Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-010)
- [ADR-011: Universal Document Provider Framework](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-011)
- [SD-001: AccountableHQ Integration Adapter](/home/erik/Projects/grctool/docs/helix/02-design/solution-designs/SD-001-accountablehq-adapter.md)
