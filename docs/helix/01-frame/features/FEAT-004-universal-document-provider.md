---
title: "FEAT-004: Universal Document Provider Framework"
phase: "01-frame"
category: "features"
tags: ["provider-framework", "domain-model", "master-index", "sync", "architecture"]
related: ["adr-010", "adr-006", "FEAT-001", "FEAT-002", "FEAT-003"]
status: "Proposed"
priority: "P0"
created: 2026-03-17
updated: 2026-03-17
---

# FEAT-004: Universal Document Provider Framework

## Status

| Field | Value |
|-------|-------|
| Status | Proposed |
| Priority | P0 |
| Owner | TBD |
| Target Phase | Phase 1 (Foundation) |

---

## Problem Statement

GRCTool has no universal abstraction for compliance data sources. The Tugboat Logic integration is hardwired: the `SyncService` directly orchestrates Tugboat API calls through a bespoke `TugboatToDomain` adapter, the `StorageService` interface exposes entity-specific `Save`/`Get`/`GetAll` methods with no provider awareness, and the domain model itself has inconsistent ID types (`Policy.ID` is `string`, while `Control.ID` and `EvidenceTask.ID` are `int`). No mechanism exists for multi-provider sync, external ID tracking, or conflict resolution.

This matters because three committed features depend on a provider abstraction that does not exist:

- **FEAT-001** (AccountableHQ Bidirectional Policy Sync) needs a `SyncProvider` interface with conflict detection and write-back capability.
- **FEAT-002** (Google Drive Bidirectional Sync) needs a `DataProvider` interface to read/write compliance documents from Drive.
- **FEAT-003** (Audit Lifecycle Scheduler) needs a provider registry to route scheduled sync operations across multiple providers.

Without this foundational framework, each integration will reinvent its own adapter pattern, ID mapping, and sync state tracking -- producing inconsistent behavior, duplicated logic, and a codebase that resists extension.

---

## Scope

### In Scope

1. **Unified entity ID model** -- Standardize all domain entity IDs to a single type (string), resolving the `Policy.ID: string` vs `Control.ID: int` / `EvidenceTask.ID: int` inconsistency. Existing numeric IDs become external identifiers.
2. **ExternalIDs and SyncMetadata on domain entities** -- Add `ExternalIDs map[string]string` (keyed by provider name) and `SyncMetadata` struct (last sync time, content hash, sync state, conflict state) to `Policy`, `Control`, and `EvidenceTask`.
3. **DataProvider interface** -- Universal read contract for compliance data sources: `ListPolicies`, `GetPolicy`, `ListControls`, `GetControl`, `ListEvidenceTasks`, `GetEvidenceTask`, with provider identity and capability reporting.
4. **SyncProvider interface** -- Extends `DataProvider` with `PushPolicy`, `PushControl`, `PushEvidenceTask`, `DeletePolicy`, `DeleteControl`, `DeleteEvidenceTask`, plus `DetectChanges` for bidirectional reconciliation.
5. **Provider registry** -- Manages registration, lookup, and lifecycle of multiple `DataProvider`/`SyncProvider` instances. Routes sync operations to the correct provider by name. Reports provider health and capabilities.
6. **Master index** -- Canonical local registry that assigns GRCTool-native IDs (POL-NNNN, CTRL-NNNN, ET-NNNN) independent of any external system. Maps native IDs to external IDs across all registered providers. Supports lookup in both directions. (Per SD-004/ADR-011, implemented as the existing `StorageService` enriched with `ExternalIDs` and `SyncMetadata` fields.)
7. **Refactored Tugboat adapter** -- Rewrite the existing `TugboatToDomain` adapter and `SyncService` to implement `DataProvider` (read-only, matching Tugboat's current capabilities), proving the interface design works against a real integration.

### Out of Scope

- AccountableHQ adapter implementation (FEAT-001)
- Google Drive adapter implementation (FEAT-002)
- Scheduler integration and cron-based sync (FEAT-003)
- Real-time push/webhook-based sync
- Multi-tenant provider configurations
- UI or dashboard for provider management

---

## User Stories

### US-001: Consistent ID Types Across Domain Entities

**As a** platform engineer,
**I want** a consistent ID type across all domain entities
**so that** adapters and storage layers don't need type-switching hacks to handle Policy (string) vs Control/EvidenceTask (int) IDs.

**Acceptance Criteria**:

- All domain entities (`Policy`, `Control`, `EvidenceTask`) use `string` as the type for their primary `ID` field
- Existing Tugboat numeric IDs are preserved as external identifiers, not lost during migration
- All `StorageService` methods accept and return the unified ID type without requiring callers to convert between `int` and `string`
- A migration path is documented for existing serialized data (JSON files on disk) that contain numeric IDs
- The `Relationship` struct in `common.go` (which already uses `string` for `SourceID`/`TargetID`) continues to work without changes
- Unit tests verify round-trip serialization of entities with the new ID type

**Dependencies**: ADR-006 (hexagonal architecture boundary between domain and adapters)

### US-002: External ID Tracking on Domain Entities

**As a** compliance manager,
**I want** each policy, control, and evidence task to track which external systems it exists in
**so that** I can trace data provenance and know where each artifact originated or is synchronized to.

**Acceptance Criteria**:

- `Policy`, `Control`, and `EvidenceTask` structs include an `ExternalIDs` field of type `map[string]string`, keyed by provider name (e.g., `{"tugboat": "12345", "accountablehq": "pol-abc"}`)
- External IDs are persisted to disk alongside the entity and survive serialization round-trips
- `ExternalIDs` is populated automatically during provider sync operations (import/export)
- The existing Tugboat adapter populates `ExternalIDs["tugboat"]` with the Tugboat numeric ID during sync
- A lookup function exists to find an entity by provider name and external ID (e.g., "find the Policy where `ExternalIDs["tugboat"] == "42"`)
- External IDs are displayed in CLI output when running `grctool tool evidence-task-details` or equivalent commands

**Dependencies**: ADR-010 (system of record architecture defines external ID semantics)

### US-003: DataProvider Interface for Read-Only Data Sources

**As a** platform engineer,
**I want** a `DataProvider` interface that defines the universal read contract for compliance data sources
**so that** new integrations can be added without modifying core storage or sync logic.

**Acceptance Criteria**:

- A `DataProvider` interface is defined in `internal/interfaces/` (or `internal/domain/`) with methods: `Name() string`, `Capabilities() ProviderCapabilities`, `ListPolicies(ctx, filter) ([]Policy, error)`, `GetPolicy(ctx, externalID) (*Policy, error)`, and equivalent methods for Controls and EvidenceTasks
- `ProviderCapabilities` reports which entity types the provider supports and whether it is read-only or read-write
- The interface uses `context.Context` for cancellation and timeout support
- The interface returns domain objects (not provider-specific DTOs), with the adapter responsible for conversion
- At least one concrete implementation exists (the refactored Tugboat adapter) that passes all interface compliance tests
- An interface compliance test suite (table-driven) can verify any `DataProvider` implementation

**Dependencies**: ADR-006 (adapter pattern), ADR-010 (provider model)

### US-004: SyncProvider Interface for Bidirectional Sync

**As a** platform engineer,
**I want** a `SyncProvider` interface that extends `DataProvider` with write operations and conflict detection
**so that** bidirectional integrations (AccountableHQ, GDrive) have a standard contract for write-back and reconciliation.

**Acceptance Criteria**:

- `SyncProvider` embeds `DataProvider` and adds: `PushPolicy(ctx, policy) error`, `PushControl(ctx, control) error`, `PushEvidenceTask(ctx, task) error`, `DeletePolicy(ctx, externalID) error`, `DeleteControl(ctx, externalID) error`, `DeleteEvidenceTask(ctx, externalID) error`
- `SyncProvider` includes `DetectChanges(ctx, since) (*ChangeSet, error)` for detecting entities that changed since a given time
- `SyncProvider` includes `ResolveConflict(ctx, conflict, resolution) error` for applying a conflict resolution decision
- Conflict resolution strategies are enumerated: `LocalWins`, `RemoteWins`, `NewestWins`, `Manual` (consistent with ADR-010)
- Write operations are idempotent: writing the same entity twice with no changes produces no side effects
- No concrete `SyncProvider` implementation is required in FEAT-004 (Tugboat is read-only); the interface is validated via a mock implementation in tests

**Dependencies**: ADR-010 (conflict resolution model), ADR-006 (port/adapter separation)

### US-005: Master Index with GRCTool-Native IDs

**As a** compliance manager,
**I want** a master index that assigns GRCTool-native IDs independent of any external system
**so that** I have a stable, human-readable identifier for every compliance artifact regardless of which providers come and go.

**Acceptance Criteria**:

- A master index is stored on disk in the data directory, in a git-friendly format (JSON or YAML). **NOTE: Per SD-004 and ADR-011, the master index is implemented as the existing `StorageService` enriched with `ExternalIDs` and `SyncMetadata` fields. No separate `.index/` directory is needed.**
- The master index assigns sequential, human-readable IDs: `POL-NNNN` for policies, `CTRL-NNNN` for controls, `ET-NNNN` for evidence tasks. **NOTE: Existing controls in the codebase use framework reference codes as identifiers (e.g., `CC-06.1`, `AC-01`, `SO-19`) rather than a GRCTool-native sequential format. The relationship between GRCTool-native IDs (`CTRL-NNNN`) and framework reference IDs needs clarification per grct-3gl.5.**
- Each index entry maps the native ID to zero or more external IDs (one per provider)
- The index supports bidirectional lookup: native ID to external IDs, and external ID (by provider) to native ID
- The index is append-only for ID assignment: once a native ID is assigned, it is never reassigned to a different entity
- Re-running sync with any provider does not create duplicate index entries; existing mappings are updated in place
- The index file can be committed to git and merged without conflicts under normal operation (no concurrent writes to the same entry)
- `grctool index list` (or equivalent) displays the index with filtering by entity type and provider

**Dependencies**: ADR-010 (master index specification)

### US-006: Provider Registry for Multi-Provider Management

**As a** platform engineer,
**I want** a provider registry that manages multiple active providers and routes sync operations
**so that** adding a new integration is a matter of registering a provider instance, not modifying orchestration code.

**Acceptance Criteria**:

- A `ProviderRegistry` struct (or interface) supports `Register(provider DataProvider) error`, `Get(name string) (DataProvider, error)`, `List() []ProviderInfo`, and `Remove(name string) error`
- Registration validates that no two providers share the same name
- `ProviderInfo` includes: name, capabilities (entity types, read/write), health status, last sync time
- The registry distinguishes between `DataProvider` (read-only) and `SyncProvider` (read-write) registrations
- The `SyncService` is refactored to iterate over registered providers rather than hard-coding Tugboat
- Provider configuration is declarative in `.grctool.yaml` (provider name, type, connection parameters)
- The registry is safe for concurrent access (multiple goroutines can read provider state)

**Dependencies**: ADR-006 (hexagonal architecture, port registry pattern)

### US-007: Sync Metadata on Every Entity

**As a** compliance manager,
**I want** sync metadata on every entity (last sync time, content hash, conflict state)
**so that** I can audit data freshness and detect stale or conflicting records.

**Acceptance Criteria**:

- A `SyncMetadata` struct is added to the domain package with per-provider map fields: `LastSyncTime map[string]time.Time` (keyed by provider name, e.g., `"tugboat"`, `"accountablehq"`), `ContentHash map[string]string` (content hash per provider for change detection), and `ConflictState string` (using constants: `""` for none, `"local_modified"`, `"remote_modified"`, `"both_modified"`, `"resolved"`)
- `Policy`, `Control`, and `EvidenceTask` each reference `SyncMetadata` via a pointer field (`*SyncMetadata`)
- `ContentHash` entries are computed deterministically from entity content (excluding metadata fields) using a stable hashing algorithm (e.g., SHA-256 of canonical JSON), stored per provider
- `LastSyncTime` entries are updated per provider during sync operations, enabling independent tracking of sync state across multiple providers
- `ConflictState` is updated automatically during sync operations: set to `""` (none) on successful sync, `"both_modified"` when both local and remote have changed, `"local_modified"` or `"remote_modified"` for one-sided changes, `"resolved"` after conflict resolution
- Sync metadata is persisted to disk alongside the entity (in the entity's JSON file) and displayed in CLI output
- `grctool sync status` (or equivalent) reports entities grouped by conflict state, highlighting conflicts and stale records
- Content hash comparison is the primary mechanism for detecting changes (not timestamp-only comparison)

**Dependencies**: ADR-010 (sync state model and conflict detection)

### US-008: Tugboat Adapter Refactored to DataProvider Interface

**As a** platform engineer,
**I want** the existing Tugboat sync refactored to implement the new `DataProvider` interface
**so that** we have proof the provider pattern works against a real integration before building AccountableHQ and GDrive adapters.

**Acceptance Criteria**:

- A `TugboatProvider` struct implements the `DataProvider` interface, wrapping the existing `tugboat.Client`
- The existing `TugboatToDomain` adapter logic (in `internal/adapters/tugboat.go`) is moved into the `TugboatProvider` implementation
- The `SyncService` no longer directly imports or references `tugboat.Client`; it interacts exclusively through the `DataProvider` interface via the provider registry
- All existing sync behavior is preserved: auto-pagination, detail embeds, the three entity types (policies, controls, evidence tasks)
- The `TugboatProvider` populates `ExternalIDs["tugboat"]` on every entity it returns
- The `TugboatProvider` computes and sets `SyncMetadata.ContentHash` on every entity it returns
- Existing integration tests and sync tests continue to pass without modification (or with minimal adapter changes)
- The `TugboatProvider` is registered by default in the provider registry when Tugboat credentials are configured

**Dependencies**: ADR-006 (adapter pattern), ADR-010 (provider model), US-003 (DataProvider interface), US-006 (provider registry)

---

## Dependencies

| Dependency | Type | Status | Notes |
|------------|------|--------|-------|
| ADR-010: System of Record Architecture | Architecture | Accepted | Defines master index, provider model, conflict resolution strategies, and sync state semantics |
| ADR-006: Hexagonal Architecture | Architecture | Accepted | Defines port/adapter pattern that DataProvider and SyncProvider implement |
| Domain model (internal/domain/) | Technical | Exists | Must be modified: unify ID types, add ExternalIDs and SyncMetadata |
| StorageService (internal/interfaces/storage.go) | Technical | Exists | Must be extended to support master index operations and provider-aware queries |
| Tugboat adapter (internal/adapters/tugboat.go) | Technical | Exists | Must be refactored to implement DataProvider |
| SyncService (internal/services/sync.go) | Technical | Exists | Must be refactored to use provider registry instead of direct Tugboat coupling |

---

## Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| ID type migration breaks existing serialized data | Medium | High | Provide a `grctool migrate` command that rewrites on-disk JSON files; support both ID formats during a transition period; comprehensive before/after test fixtures |
| DataProvider interface is too rigid for future providers | Medium | High | Design the interface around the three core entity types but include an `Extensions() map[string]interface{}` escape hatch; validate against AccountableHQ and GDrive requirements before finalizing |
| Master index file conflicts in git during team collaboration | Low | Medium | Use line-per-entry JSONL format or YAML with sorted keys to minimize merge conflicts; document merge strategy |
| Content hash instability across Go versions or serialization changes | Low | High | Pin canonical JSON serialization (sorted keys, no HTML escaping, deterministic float rendering); hash algorithm is versioned in the index |
| Performance degradation with large provider registries or index files | Low | Medium | Lazy-load provider connections; memory-map the index file for large installations; benchmark with 1000+ entities |
| Tugboat adapter refactoring introduces regressions in existing sync | Medium | Medium | Run existing integration test suite against the refactored adapter in CI before merging; feature-flag the new code path with fallback to legacy |

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| ID type consistency | 100% of domain entities use string IDs | Static analysis / compile-time verification |
| DataProvider compliance | Tugboat adapter passes full interface compliance test suite | Automated test suite with zero failures |
| Master index completeness | Every synced entity has a native ID and at least one external ID mapping | Index entry count matches entity count after sync |
| Zero sync regression | All existing sync integration tests pass after refactoring | CI test suite, zero failures |
| Provider registration | New provider can be registered with < 50 lines of boilerplate | Code review of a mock provider implementation |
| Sync metadata coverage | 100% of synced entities have ContentHash and LastSyncedAt populated | Post-sync validation query |

---

## References

- [ADR-010: System of Record Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-010-system-of-record-architecture)
- [ADR-006: Hexagonal Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-006-hexagonal-architecture)
- [FEAT-001: AccountableHQ Bidirectional Policy Sync](/home/erik/Projects/grctool/docs/helix/01-frame/features/FEAT-001-accountablehq-policy-sync.md)
- [FEAT-002: Google Drive Bidirectional Sync](/home/erik/Projects/grctool/docs/helix/01-frame/features/FEAT-002-gdrive-bidirectional-sync.md)
- [FEAT-003: Audit Lifecycle Scheduler](/home/erik/Projects/grctool/docs/helix/01-frame/features/FEAT-003-audit-lifecycle-scheduler.md)
- [PRD Requirements: System of Record Vision](/home/erik/Projects/grctool/docs/helix/01-frame/prd/requirements.md)
- [Domain Model: Policy](/home/erik/Projects/grctool/internal/domain/policy.go) -- `ID: string`
- [Domain Model: Control](/home/erik/Projects/grctool/internal/domain/control.go) -- `ID: int` (inconsistency)
- [Domain Model: EvidenceTask](/home/erik/Projects/grctool/internal/domain/evidence.go) -- `ID: int` (inconsistency)
- [StorageService Interface](/home/erik/Projects/grctool/internal/interfaces/storage.go)
