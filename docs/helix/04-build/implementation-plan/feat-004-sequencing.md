---
title: "FEAT-004 Implementation Sequencing Plan"
phase: "04-build"
category: "implementation-plan"
tags: ["feat-004", "provider-framework", "sequencing", "migration"]
created: 2026-03-18
updated: 2026-03-18
---

# FEAT-004: Universal Document Provider Framework — Implementation Sequencing

## Overview

This document captures the actual implementation sequence for FEAT-004, based on the dependency graph defined in beads epic grct-3gl. FEAT-004 is the P0 foundation that FEAT-001 (AccountableHQ), FEAT-002 (GDrive), and FEAT-003 (Lifecycle/Scheduler) depend on.

## Phase Ordering

```
FEAT-004 (Foundation) → FEAT-003 (Lifecycle, partial parallel)
                       → FEAT-001 (AccountableHQ, after FEAT-004)
                       → FEAT-002 (GDrive, after FEAT-004)
```

FEAT-003's lifecycle state machines and scheduler engine can be built in parallel with FEAT-004's later phases. FEAT-001 and FEAT-002 require the provider framework to be complete.

## Implementation Phases (Completed)

### Phase 1: ID Type Migration (grct-3gl.6, grct-3gl.7)

**Status**: Done

- Changed Control.ID and EvidenceTask.ID from `int` to `string`
- Added custom UnmarshalJSON for backward-compatible JSON loading
- Created `grctool migrate` tool for on-disk normalization
- 96 files changed, all tests passing

**Rollback**: Run `grctool migrate` in reverse is not needed — custom UnmarshalJSON handles both formats indefinitely.

### Phase 2: Domain Model Extension (grct-3gl.8, grct-3gl.5)

**Status**: Done

- Added ExternalIDs and SyncMetadata to Policy, Control, EvidenceTask
- Resolved control ID format: framework codes (CC-06.1) are canonical
- All fields use `omitempty` — backward compatible

**Testing gate**: ID migration regression suite (44 tests) in `test/integration/id_migration_test.go`.

### Phase 3: Provider Interfaces (grct-3gl.9)

**Status**: Done

- Defined DataProvider (read-only) and SyncProvider (read-write) in `internal/interfaces/provider.go`
- Supporting types: ListOptions, ChangeSet, ChangeEntry
- Reusable StubDataProvider in testhelpers

**Testing gate**: DataProvider contract test suite (15 tests per provider) in `internal/providers/contract_test.go`.

### Phase 4: Provider Infrastructure (grct-3gl.10, grct-3gl.11, grct-3gl.14)

**Status**: Done

- ProviderRegistry: thread-safe registration, lookup, health checks
- TugboatDataProvider: wraps existing client as DataProvider
- Config schema: ProvidersConfig, SchedulesConfig, LifecycleConfig added

**Testing gate**: Registry lifecycle tests (100% coverage), Tugboat contract compliance.

### Phase 5: SyncService Refactoring (grct-3gl.12)

**Status**: Done

- SyncService accepts ProviderRegistry, iterates registered providers
- Backward compatible: existing constructor auto-creates registry with Tugboat
- Statistics aggregated across providers

**Testing gate**: 10 regression tests in `internal/services/sync_regression_test.go`.

### Phase 6: Storage Extension (grct-3gl.13)

**Status**: Done

- GetByExternalID methods on StorageService, LocalDataStore, StubStorageService
- Linear scan with nil-map safety

### Phase 7: Cross-Cutting Components (grct-3gl.15, grct-3gl.16, grct-3gl.17)

**Status**: Done

- Conflict detection and resolution framework (4 policies)
- Sync orchestrator (detect → resolve → apply cycle)
- Unified audit trail (YAML, append-only, per-run files)

## Migration Strategy

Per ADR-010, the migration from Tugboat-as-source to GRCTool-as-source follows three phases:

### Shadow Index (Current)

GRCTool syncs from Tugboat and stores locally with ExternalIDs tracking. Tugboat remains the source of truth. The provider framework is in place but only the TugboatDataProvider is active.

### Dual-Write (When FEAT-001/002 adapters are built)

GRCTool syncs bidirectionally with Tugboat and new providers. Conflict resolution determines which version wins. The sync orchestrator coordinates the detect→resolve→apply cycle.

### GRCTool-as-Source (Future)

GRCTool's local storage is authoritative. External systems receive updates from GRCTool. Tugboat becomes an export target, not a source.

**Trigger conditions for each phase transition**: Number of active SyncProviders > 1, conflict resolution tested in production, audit trail shows consistent sync behavior.

## Quality Gates

Each phase required:
1. All existing tests passing (zero regressions)
2. New tests for new code (contract tests, regression tests)
3. Coverage maintained or improved (16% → 44.6%)
4. `go build ./...` and `go vet ./...` clean

## Risk Mitigations Applied

| Risk | Mitigation |
|------|-----------|
| ID type breaking change | Custom UnmarshalJSON + migration tool |
| Backward compatibility | `omitempty` on all new fields, old constructor preserved |
| Provider interface rigidity | Minimal interface (6 read methods + 7 write methods) |
| Sync regression | 10-test regression suite with golden file comparison |
| Conflict data loss | Configurable policy (local_wins, remote_wins, manual) |
