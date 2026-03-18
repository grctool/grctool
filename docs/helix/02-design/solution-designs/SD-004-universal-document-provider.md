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

All interfaces are defined in `internal/interfaces/provider.go` (new file).

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

// DataProvider defines read-only access to compliance entities from an external source.
// This is the minimal interface that every compliance platform must implement.
type DataProvider interface {
    // Name returns the unique identifier for this provider (e.g., "tugboat", "accountablehq").
    Name() string

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

    // TestConnection verifies that the provider is reachable and authenticated.
    TestConnection(ctx context.Context) error
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
}
```

### ChangeSet

```go
// ChangeSet represents entities that changed in a remote provider since a given time.
type ChangeSet struct {
    // Provider name that produced this changeset.
    Provider string `json:"provider"`

    // Since is the timestamp used to detect changes.
    Since time.Time `json:"since"`

    // DetectedAt is when this changeset was computed.
    DetectedAt time.Time `json:"detected_at"`

    // Changed entities by type.
    Policies      []ChangeEntry `json:"policies,omitempty"`
    Controls      []ChangeEntry `json:"controls,omitempty"`
    EvidenceTasks []ChangeEntry `json:"evidence_tasks,omitempty"`
}

// ChangeEntry represents a single changed entity.
type ChangeEntry struct {
    // ID is the provider-native ID of the changed entity.
    ID string `json:"id"`

    // ChangeType indicates what happened: "created", "updated", "deleted".
    ChangeType string `json:"change_type"`

    // ChangedAt is when the change occurred in the remote system.
    ChangedAt time.Time `json:"changed_at"`

    // ContentHash is the new content hash (empty for deletes).
    ContentHash string `json:"content_hash,omitempty"`
}
```

### ProviderRegistry

```go
// ProviderRegistry manages the lifecycle of data and sync providers.
// Defined in internal/interfaces/provider.go, implemented in internal/services/provider_registry.go.
type ProviderRegistry interface {
    // Register adds a provider to the registry. Errors if a provider with the same name exists.
    Register(provider DataProvider) error

    // Get returns a provider by name. Returns nil if not found.
    Get(name string) DataProvider

    // GetSync returns a provider as SyncProvider if it implements bidirectional sync.
    // Returns nil if the provider is read-only or not found.
    GetSync(name string) SyncProvider

    // List returns the names of all registered providers.
    List() []string

    // HealthCheck tests connectivity for all registered providers.
    HealthCheck(ctx context.Context) map[string]error

    // Close shuts down all providers gracefully.
    Close() error
}
```

---

## ProviderRegistry Implementation

The registry is implemented in `internal/services/provider_registry.go` (new file):

```go
package services

import (
    "context"
    "fmt"
    "sync"

    "github.com/grctool/grctool/internal/interfaces"
)

type providerRegistry struct {
    mu        sync.RWMutex
    providers map[string]interfaces.DataProvider
}

func NewProviderRegistry() interfaces.ProviderRegistry {
    return &providerRegistry{
        providers: make(map[string]interfaces.DataProvider),
    }
}

func (r *providerRegistry) Register(provider interfaces.DataProvider) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    name := provider.Name()
    if _, exists := r.providers[name]; exists {
        return fmt.Errorf("provider %q already registered", name)
    }
    r.providers[name] = provider
    return nil
}

func (r *providerRegistry) Get(name string) interfaces.DataProvider {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.providers[name]
}

func (r *providerRegistry) GetSync(name string) interfaces.SyncProvider {
    r.mu.RLock()
    defer r.mu.RUnlock()
    if p, ok := r.providers[name]; ok {
        if sp, ok := p.(interfaces.SyncProvider); ok {
            return sp
        }
    }
    return nil
}

func (r *providerRegistry) List() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    names := make([]string, 0, len(r.providers))
    for name := range r.providers {
        names = append(names, name)
    }
    return names
}

func (r *providerRegistry) HealthCheck(ctx context.Context) map[string]error {
    r.mu.RLock()
    defer r.mu.RUnlock()
    results := make(map[string]error, len(r.providers))
    for name, provider := range r.providers {
        results[name] = provider.TestConnection(ctx)
    }
    return results
}

func (r *providerRegistry) Close() error {
    // Future: call Close() on providers that implement io.Closer
    return nil
}
```

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
    ├── interfaces.ProviderRegistry  (abstracted dependency)
    ├── interfaces.StorageService    (abstracted dependency)
    └── orchestration methods: syncFromProvider(providerName)

TugboatDataProvider (internal/adapters/tugboat_provider.go) — NEW
    ├── *tugboat.Client         (encapsulated)
    ├── *adapters.TugboatToDomain (encapsulated)
    └── implements: interfaces.DataProvider
```

### New File: `internal/adapters/tugboat_provider.go`

This wraps the existing `tugboat.Client` and `TugboatToDomain` adapter into a `DataProvider` implementation:

```go
package adapters

import (
    "context"
    "strconv"

    "github.com/grctool/grctool/internal/domain"
    "github.com/grctool/grctool/internal/interfaces"
    "github.com/grctool/grctool/internal/tugboat"
)

// TugboatDataProvider implements interfaces.DataProvider by wrapping
// the existing tugboat.Client and TugboatToDomain adapter.
type TugboatDataProvider struct {
    client  *tugboat.Client
    adapter *TugboatToDomain
    orgID   string
}

func NewTugboatDataProvider(client *tugboat.Client, orgID string) *TugboatDataProvider {
    return &TugboatDataProvider{
        client:  client,
        adapter: NewTugboatToDomain(),
        orgID:   orgID,
    }
}

func (p *TugboatDataProvider) Name() string { return "tugboat" }

func (p *TugboatDataProvider) ListPolicies(ctx context.Context, opts interfaces.ListOptions) ([]domain.Policy, int, error) {
    apiPolicies, err := p.client.GetAllPolicies(ctx, p.orgID, opts.Framework)
    if err != nil {
        return nil, 0, err
    }

    policies := make([]domain.Policy, 0, len(apiPolicies))
    for _, apiPolicy := range apiPolicies {
        dp := p.adapter.ConvertPolicy(apiPolicy)
        dp.ExternalIDs = map[string]string{"tugboat": dp.ID}
        policies = append(policies, dp)
    }
    return policies, len(policies), nil
}

// ListControls, ListEvidenceTasks, GetPolicy, GetControl, GetEvidenceTask
// follow the same pattern: call tugboat.Client method, convert via adapter,
// set ExternalIDs, return domain entity.

func (p *TugboatDataProvider) TestConnection(ctx context.Context) error {
    return p.client.TestConnection(ctx)
}
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
| `internal/interfaces/provider.go` | `DataProvider`, `SyncProvider`, `ProviderRegistry`, `ListOptions`, `ChangeSet`, `ChangeEntry` |
| `internal/services/provider_registry.go` | Concrete `ProviderRegistry` implementation |
| `internal/adapters/tugboat_provider.go` | `TugboatDataProvider` implementing `DataProvider` |

---

## Testing Strategy

1. **Unit tests for TugboatDataProvider**: Use existing VCR cassettes (ADR-005) to verify that the provider returns correctly typed domain entities with `ExternalIDs` populated.

2. **Integration tests for ProviderRegistry**: Register multiple mock providers, verify lookup, health check, and lifecycle management.

3. **Migration test**: Load existing JSON files with numeric IDs, verify they deserialize correctly into `string` ID fields.

4. **Mock DataProvider for sync tests**: Replace the current `*tugboat.Client` mock pattern with a mock `DataProvider`, simplifying test setup.

---

## Open Questions

1. **Pagination strategy**: The Tugboat API uses page-based pagination. Should `ListOptions` also support cursor-based pagination for providers that prefer it?

2. **Bulk vs. detail fetch**: Current sync fetches a list then detail for each entity. Should `DataProvider` have a `ListDetailedPolicies` method, or should providers internally optimize?

3. **Conflict resolution UI**: ADR-010 mentions configurable conflict resolution (`local_wins`, `remote_wins`, `manual`, `newest_wins`). The `SyncMetadata.ConflictState` field supports detection, but the resolution workflow needs a separate design.

---

## References

- [ADR-006: Hexagonal Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-006)
- [ADR-010: System of Record Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-010)
- [ADR-011: Universal Document Provider Framework](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-011)
- [SD-001: AccountableHQ Integration Adapter](/home/erik/Projects/grctool/docs/helix/02-design/solution-designs/SD-001-accountablehq-adapter.md)
