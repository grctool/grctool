---
title: "Monitoring Setup"
phase: "05-deploy"
category: "monitoring"
tags: ["monitoring", "logging", "zerolog", "observability", "performance", "metrics"]
related: ["deployment-operations", "runbook", "security-monitoring"]
created: 2026-03-17
updated: 2026-03-17
helix_mapping: "05-Deploy monitoring-setup artifact"
---

# Monitoring Setup

## Overview

GRCTool is a CLI application, not a long-running service. Its monitoring strategy centers on structured logging, performance benchmarks, error tracking through exit codes and log output, and CI-driven quality metrics. This document describes the observability infrastructure in place and how operators can diagnose issues in production use.

## Structured Logging Configuration

### Zerolog Foundation

GRCTool uses [zerolog](https://github.com/rs/zerolog) as its structured logging backend, wrapped behind a custom `Logger` interface in `internal/logger/`. This abstraction enables multi-destination logging and consistent field semantics across the entire codebase.

**Key implementation files:**

- `internal/logger/zerolog_logger.go` -- ZerologLogger adapter with full interface implementation
- `internal/logger/multi.go` -- MultiLogger for simultaneous console and file output

### Logger Interface

The `Logger` interface provides structured, type-safe logging with context propagation:

```go
type Logger interface {
    Trace(msg string, fields ...Field)
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    WithFields(fields ...Field) Logger
    WithContext(ctx context.Context) Logger
    WithComponent(component string) Logger
    TraceOperation(operation string) Tracer
    RequestLogger(requestID string) Logger
    DumpJSON(obj interface{}, msg string)
    Timing(operation string) TimingLogger
}
```

### Log Levels

| Level | Use Case | Default Visibility |
|-------|----------|--------------------|
| `trace` | Detailed diagnostic data, field-level operations | Off in production |
| `debug` | Operation steps, checkpoint timing, VCR playback details | Off in production |
| `info` | Normal operations: sync started, evidence generated, auth status | Default level |
| `warn` | Degraded situations: timing abandoned, coverage below threshold | Always visible |
| `error` | Failures: auth failures, API errors, generation failures | Always visible |

### Output Destinations

GRCTool supports three output modes, configured via `.grctool.yaml`:

```yaml
logging:
  level: "info"
  format: "json"       # "json" for structured, "text" for human-readable
  output: "stdout"     # "stdout", "stderr", or "file"
  file_path: ""        # Required when output is "file"
  show_caller: false   # Include source file and line number
```

**Console output (text format):** Uses `zerolog.ConsoleWriter` with RFC3339 timestamps and color support for terminal use.

**File output:** Uses `zerolog/diode` for non-blocking writes with a 1000-message buffer and 10ms flush interval. Dropped messages are reported to stderr.

**JSON output:** Default structured format with fields: `timestamp`, `level`, `component`, `operation`, `duration`, `message`.

### Multi-Logger Configuration

The `CreateMultiLogger` function enables simultaneous logging to console and file:

```go
logger, err := logger.CreateMultiLogger(
    &logger.Config{Level: "info", Format: "text", Output: "stdout"},
    &logger.Config{Level: "debug", Format: "json", Output: "file", FilePath: "/var/log/grctool/grctool.log"},
)
```

This produces human-readable console output for interactive use while capturing full-detail JSON logs for post-hoc analysis.

## Operation Tracing

### TraceOperation Pattern

GRCTool includes a built-in operation tracing facility for tracking multi-step workflows:

```go
tracer := log.TraceOperation("evidence_generation")
tracer.Step("loading_task", logger.String("task_id", "ET-0047"))
tracer.Step("running_tools", logger.Int("tool_count", 3))
tracer.Success(logger.String("output_format", "json"))
// or: tracer.Error(err)
```

Each step logs elapsed time, step count, and custom fields. On completion, the total duration and step count are recorded at `info` level.

### Timing Logger

For fine-grained performance analysis, the `TimingLogger` interface tracks checkpoint durations within an operation:

```go
timing := log.Timing("sync_operation")
timing.Mark("fetch_policies")
timing.Mark("fetch_controls")
timing.Mark("fetch_evidence_tasks")
timing.Complete()
```

The `Complete` call emits a timing breakdown with per-checkpoint `duration_ms` and cumulative `elapsed_ms`.

## Performance Monitoring

### Benchmark Infrastructure

GRCTool maintains a comprehensive benchmark suite, run via Makefile targets:

| Target | Purpose |
|--------|---------|
| `make bench` | Run all benchmarks across tugboat, tools, auth, services, storage packages |
| `make bench-compare` | Compare current results against saved baseline |
| `make bench-profile` | Generate CPU and memory profiles for the tugboat sync benchmark |
| `make bench-save` | Save current results as the new baseline |
| `make bench-memory` | Run memory-focused benchmarks (large file processing, auth allocation, data service) |
| `make bench-report` | Generate a comprehensive comparison report |

Benchmark results are stored in `benchmarks/current.txt` with baselines managed by `scripts/benchmark-compare.sh`.

### CI Performance Tracking

The `testing.yml` workflow includes a benchmarks job that:

1. Downloads the previous baseline artifact
2. Runs the full benchmark suite
3. Compares against baseline using `benchmark-action/github-action-benchmark`
4. Alerts at 110% regression threshold
5. Comments on PRs when regressions are detected

### Profiling

For deep performance analysis:

```bash
# Generate CPU and memory profiles
make bench-profile

# Analyze profiles
go tool pprof benchmarks/profiles/cpu.prof
go tool pprof benchmarks/profiles/mem.prof
```

## Error Tracking

### Structured Error Reporting

All errors flow through the structured logging system with consistent fields:

- `component`: Which package generated the error (e.g., `tugboat`, `auth`, `tools`)
- `operation`: What was being attempted (e.g., `sync`, `evidence_generation`)
- `duration`: How long the operation ran before failing
- `err`: The error message

### CLI Exit Codes

GRCTool uses standard exit codes for scriptability:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (auth failure, API error, invalid config) |
| Non-zero | Tool-specific failures |

### Error Patterns to Monitor

When reviewing logs, watch for:

- **Repeated auth failures**: May indicate expired credentials or Tugboat API changes
- **Sync timeouts**: Network issues or Tugboat API degradation
- **Evidence generation failures**: Missing task data, Claude API errors, or tool failures
- **Cache corruption**: Stale or invalid `.cache/` data causing unexpected behavior

## Health Check Mechanisms

### Configuration Validation

```bash
# Verify configuration is valid
grctool config validate

# Check authentication status
grctool auth status
```

### Installation Verification

The `scripts/install.sh` installer performs post-install verification:

1. Binary exists and is executable
2. Version command responds correctly
3. Auth status check (informational)

### Operational Health Checks

For routine health verification:

```bash
# Check binary version and build info
grctool version

# Verify Tugboat connectivity
grctool auth status

# Test sync with a dry run
grctool sync --verbose
```

## CLI Metrics and Telemetry

### Current State

GRCTool does not currently collect or transmit telemetry data. All monitoring is local and opt-in through log configuration.

### Available Metrics via Logs

When structured JSON logging is enabled, the following metrics can be extracted:

- **Command execution time**: From `timing_completed` log entries
- **Evidence generation success/failure rate**: From `operation_completed` / `operation_failed` entries
- **API call duration**: From operation step timing breakdowns
- **Tool execution counts**: From tool-specific log entries

### Extracting Metrics from Logs

```bash
# Count evidence generation outcomes from JSON logs
grep "operation_completed" /var/log/grctool/grctool.log | jq -r '.operation' | sort | uniq -c

# Extract average sync duration
grep "timing_completed" /var/log/grctool/grctool.log | \
  jq 'select(.operation == "sync_operation") | .total_duration' | \
  awk '{sum+=$1; count++} END {print sum/count "ms average"}'
```

## Alerting Approach

Since GRCTool is a CLI tool (not a service), alerting is handled through CI/CD pipeline status rather than runtime monitoring:

### CI Quality Gates

The `ci.yml` workflow enforces quality gates that serve as the primary alerting mechanism:

| Gate | Condition | Action on Failure |
|------|-----------|-------------------|
| CI Checks | Lefthook CI checks pass | Block merge |
| Test Matrix | Unit + integration tests pass | Block merge |
| Build Matrix | Linux amd64 build succeeds | Block merge |
| Coverage | Coverage meets threshold (currently 15%, target 80%) | Block merge |
| License Check | All files have license headers | Block merge |
| Security Scan | gosec + secret detection | Informational (non-blocking) |

### Scheduled Analysis

The `testing.yml` advanced testing workflow runs weekly (Monday 2 AM UTC) for:

- Deep coverage analysis with critical package checks
- Performance benchmark regression detection
- Mutation testing quality assessment

## Monitoring Checklist

- [x] Structured logging with zerolog and custom Logger interface
- [x] Multi-destination logging (console + file)
- [x] Operation tracing with step-level timing
- [x] Performance benchmarks with baseline comparison
- [x] CI quality gates for automated alerting
- [x] Secret detection in pre-commit hooks
- [x] Security scanning (gosec) in CI
- [x] Coverage tracking and threshold enforcement
- [ ] Runtime telemetry collection (not planned -- CLI tool)
- [ ] External monitoring dashboard (not applicable -- CLI tool)

## References

- `internal/logger/zerolog_logger.go` -- Zerolog adapter implementation
- `internal/logger/multi.go` -- Multi-logger implementation
- `Makefile` -- Benchmark and coverage targets
- `.github/workflows/ci.yml` -- CI quality gates
- `.github/workflows/testing.yml` -- Advanced testing and benchmarks
- `scripts/benchmark-compare.sh` -- Benchmark comparison tooling

---

*This monitoring setup reflects GRCTool's architecture as a CLI tool where observability is achieved through structured logging, CI-driven quality metrics, and performance benchmarks rather than traditional service monitoring.*
