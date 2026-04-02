---
title: "Architecture Decision Records Index"
phase: "02-design"
category: "adr"
tags: ["architecture", "decisions", "design", "rationale"]
related: ["system-design", "security-design", "requirements"]
created: 2025-01-10
updated: 2026-03-17
helix_mapping: "Backfilled from codebase evidence"
---

# Architecture Decision Records

This document captures the significant architectural decisions made during GRCTool's design and development. Each ADR records the context, decision, alternatives considered, and consequences.

## ADR Summary

| ADR | Title | Status | Confidence | Date |
|-----|-------|--------|------------|------|
| ADR-001 | Go as implementation language | Accepted | High | 2025-01-10 |
| ADR-002 | Cobra CLI framework | Accepted | High | 2025-01-10 |
| ADR-003 | Browser-based authentication over form-based | Accepted | High | 2025-01-10 |
| ADR-004 | JSON-based local storage for evidence data | Accepted | High | 2025-01-10 |
| ADR-005 | VCR testing pattern for API integration tests | Accepted | High | 2025-01-10 |
| ADR-006 | Hexagonal architecture (ports and adapters) | Accepted | High | 2025-01-10 |
| ADR-007 | Template variable interpolation for organization customization | Accepted | Med | 2025-01-10 |
| ADR-008 | zerolog for structured logging | Accepted | High | 2025-01-10 |
| ADR-009 | Agentic workflow architecture for evidence coordination | Accepted | Med | 2025-01-10 |
| ADR-010 | System of record architecture | Accepted | Med | 2026-03-17 |
| ADR-011 | Universal document provider framework | Accepted | Med | 2026-03-17 |

---

## ADR-001: Go as Implementation Language

| Date | Status | Deciders | Related | Confidence |
|------|--------|----------|---------|------------|
| 2025-01-10 | Accepted | Engineering Team | ADR-002, ADR-008 | High |

### Context

| Aspect | Description |
|--------|-------------|
| Problem | Selecting a language for a CLI compliance tool that must integrate with REST APIs, parse Terraform HCL, interact with the filesystem, and produce reliable builds for distribution |
| Current State | Greenfield project with no existing codebase constraints |
| Requirements | Single-binary distribution, strong concurrency for parallel tool execution, static typing for reliability, first-class CLI ecosystem, ability to parse HCL natively |

### Decision

We will use Go (1.24+) as the implementation language for GRCTool.

**Key Points**: Single-binary compilation | Native HCL parsing via hashicorp/hcl | Strong CLI ecosystem (Cobra, Viper)

### Alternatives

| Option | Pros | Cons | Evaluation |
|--------|------|------|------------|
| Python | Rapid prototyping, rich library ecosystem, widely known | Packaging complexity (virtualenvs, wheels), runtime dependency, slower execution, no native HCL parsing | Rejected: distribution complexity and runtime dependency unacceptable for CLI tool |
| Rust | Maximum performance, memory safety, single binary | Steeper learning curve, slower compilation, smaller CLI ecosystem, no first-party HCL library | Rejected: learning curve overhead not justified for this domain; CLI ecosystem less mature |
| TypeScript/Node | Rapid development, easy JSON handling, large npm ecosystem | Runtime dependency (Node.js), packaging complexity, less suited for CLI tooling | Rejected: runtime dependency and performance characteristics unsuitable |
| **Go** | Single binary, native HCL parsing, Cobra/Viper ecosystem, goroutines for concurrency, fast compilation | Verbose error handling, limited generics (improving), no sum types | **Selected: best balance of distribution simplicity, ecosystem fit, and HCL support** |

### Consequences

| Type | Impact |
|------|--------|
| Positive | Zero-dependency distribution as a single binary; native HCL/Terraform parsing via hashicorp/hcl/v2; proven CLI framework ecosystem (Cobra + Viper); goroutine-based concurrency for parallel tool execution; fast compilation and test cycles |
| Negative | Verbose error handling increases code volume; generics limitations require some interface-based workarounds; smaller pool of Go developers compared to Python/JS |
| Neutral | Team must maintain Go toolchain (currently go1.24.12); dependency management via go.mod is straightforward |

### Implementation Impact

| Aspect | Assessment |
|--------|------------|
| Effort | Low - Go is well-suited for this domain |
| Skills | Go proficiency required; team has existing experience |
| Performance | Excellent for CLI workloads; sub-second command execution |
| Scalability | Goroutines handle concurrent evidence collection well |
| Security | Strong type system reduces class of bugs; govulncheck for dependency scanning |

### Validation

| Success Metric | Review Trigger |
|----------------|----------------|
| CLI commands complete in < 5 seconds for typical operations | Performance degrades below acceptable thresholds |
| Single binary < 50MB distributable | Binary size grows beyond acceptable limits |
| go.mod dependency count stays manageable (< 30 direct) | Dependency sprawl indicates architecture issues |

---

## ADR-002: Cobra CLI Framework

| Date | Status | Deciders | Related | Confidence |
|------|--------|----------|---------|------------|
| 2025-01-10 | Accepted | Engineering Team | ADR-001 | High |

### Context

| Aspect | Description |
|--------|-------------|
| Problem | GRCTool requires a structured CLI with subcommands (auth, sync, evidence, tool), persistent flags, help generation, and shell completion |
| Current State | Go selected as language; need a CLI framework that supports complex command hierarchies |
| Requirements | Nested subcommands, flag parsing with validation, automatic help/usage generation, shell completion, Viper integration for configuration |

### Decision

We will use Cobra (`github.com/spf13/cobra`) as the CLI framework, paired with Viper (`github.com/spf13/viper`) for configuration management and pflag (`github.com/spf13/pflag`) for POSIX-compliant flag parsing.

**Key Points**: Industry-standard Go CLI framework | Native Viper integration for config | Supports 30+ tool subcommands cleanly

### Alternatives

| Option | Pros | Cons | Evaluation |
|--------|------|------|------------|
| Standard library flag | No dependencies, simple | No subcommand support, no help generation, no shell completion, no config file integration | Rejected: insufficient for complex CLI with 30+ subcommands |
| urfave/cli | Simpler API, good for basic CLIs | Less mature subcommand support, no native Viper integration, smaller ecosystem | Rejected: Viper integration and subcommand depth favor Cobra |
| Kong | Struct-based CLI definition, clean API | Smaller community, no Viper integration, less battle-tested at scale | Rejected: Cobra's ecosystem maturity and Viper pairing are decisive |
| **Cobra + Viper + pflag** | De facto Go CLI standard, native config integration, shell completion, excellent subcommand support | Larger dependency footprint than stdlib, Cobra's code generation is opinionated | **Selected: proven at scale (kubectl, hugo, gh), native Viper integration** |

### Consequences

| Type | Impact |
|------|--------|
| Positive | Automatic help text and usage generation for all 30+ tools; YAML/env/flag configuration precedence via Viper; shell completion for bash/zsh; consistent command pattern across auth, sync, evidence, and tool subcommands |
| Negative | Cobra adds ~3 direct dependencies; RunE pattern requires consistent error propagation; Viper's global state can complicate testing |
| Neutral | Configuration precedence (flags > env > file > defaults) is handled automatically by Viper |

### Validation

| Success Metric | Review Trigger |
|----------------|----------------|
| All 30+ tool commands register cleanly under `grctool tool` | Command hierarchy becomes unwieldy or slow to initialize |
| `grctool --help` output is clear and complete | Users report confusion about available commands |

---

## ADR-003: Browser-Based Authentication (Safari) over Form-Based

| Date | Status | Deciders | Related | Confidence |
|------|--------|----------|---------|------------|
| 2025-01-10 | Accepted | Engineering Team | ADR-006 | High |

### Context

| Aspect | Description |
|--------|-------------|
| Problem | Tugboat Logic does not expose a public API key provisioning mechanism; authentication requires session cookies from an authenticated browser session |
| Current State | Tugboat Logic uses browser-based sessions with cookies; no documented OAuth or API key endpoints |
| Requirements | Authenticate with Tugboat Logic API without storing long-lived credentials in config files; support the existing browser-based login flow; minimize credential exposure risk |

### Decision

We will use browser-based authentication that extracts session cookies from Safari on macOS using AppleScript, rather than implementing form-based credential submission.

**Key Points**: No credentials stored in config files | Leverages existing browser session | Safari cookie extraction via AppleScript on macOS

### Alternatives

| Option | Pros | Cons | Evaluation |
|--------|------|------|------------|
| Form-based login (username/password in config) | Simple implementation, cross-platform | Credentials stored in plaintext config, password rotation breaks automation, violates zero-trust principles | Rejected: storing credentials in config is unacceptable for a compliance tool |
| OAuth 2.0 flow | Industry standard, token-based, refreshable | Tugboat Logic does not expose OAuth endpoints; would require API changes from vendor | Rejected: not supported by Tugboat Logic API |
| API key provisioning | Simple, long-lived token | Tugboat Logic does not provide API key management; would require vendor support | Rejected: not available from vendor |
| **Browser-based (Safari cookie extraction)** | No stored credentials, leverages existing SSO/MFA, browser handles auth complexity | macOS-only (Safari AppleScript), requires Accessibility permissions, session cookies expire | **Selected: aligns with zero-trust principles; no credential storage** |

### Consequences

| Type | Impact |
|------|--------|
| Positive | No passwords or API keys stored in config files; leverages enterprise SSO/MFA already configured in browser; session tokens auto-refresh when user is logged into Tugboat; aligns with compliance tool security expectations |
| Negative | macOS-only limitation (Safari AppleScript); requires macOS Accessibility permissions; sessions expire and require re-authentication; not suitable for CI/CD without alternative auth path |
| Neutral | Auth provider interface (`AuthProvider`) allows adding cross-platform providers later; `NoAuthProvider` exists for tools that work offline |

### Risks

| Risk | Prob | Impact | Mitigation |
|------|------|--------|------------|
| macOS-only limits adoption | Medium | Medium | Document manual cookie extraction; plan cross-platform auth provider |
| Safari API changes break extraction | Low | High | AppleScript API is stable; VCR tests cover auth flow |
| Session expiry during long operations | Medium | Low | Auth status check before operations; clear error messages for re-auth |

### Validation

| Success Metric | Review Trigger |
|----------------|----------------|
| Auth flow completes in < 10 seconds | Users report frequent auth failures |
| No credentials appear in config files or logs | Security audit finds credential exposure |
| Auth provider interface supports future providers | Cross-platform requirement becomes critical |

---

## ADR-004: JSON-Based Local Storage for Evidence Data

| Date | Status | Deciders | Related | Confidence |
|------|--------|----------|---------|------------|
| 2025-01-10 | Accepted | Engineering Team | ADR-006 | High |

### Context

| Aspect | Description |
|--------|-------------|
| Problem | GRCTool needs to persist synced policies, controls, evidence tasks, generated evidence, and submission state locally for offline access and audit traceability |
| Current State | No existing data store; compliance data comes from Tugboat Logic API in JSON format |
| Requirements | Human-readable storage for audit review; version-controllable via git; no external database dependency; support for structured queries via filesystem patterns; offline-first operation |

### Decision

We will use JSON files as the primary storage format, organized in a structured directory hierarchy with deterministic naming conventions, supplemented by YAML for configuration and submission state.

**Key Points**: JSON for domain data (policies, controls, tasks) | YAML for configuration and state | Deterministic file naming (ET-0001-327992-name.json) | Git-friendly for audit trails

### Alternatives

| Option | Pros | Cons | Evaluation |
|--------|------|------|------------|
| SQLite | Structured queries, ACID transactions, single file | Not human-readable, harder to diff in git, requires database driver dependency | Rejected: human readability and git integration are primary requirements |
| PostgreSQL | Full SQL, concurrent access, robust querying | External dependency, deployment complexity, overkill for single-user CLI | Rejected: violates no-external-dependency requirement |
| BoltDB/bbolt | Embedded Go key-value store, single file, no external deps | Binary format not human-readable, no git diffing, limited query capability | Rejected: human readability requirement |
| **JSON files + YAML config** | Human-readable, git-diffable, no dependencies, matches API format, offline-first | No ACID transactions, filesystem-limited query performance, potential for large file counts | **Selected: audit transparency and version control integration are decisive** |

### Consequences

| Type | Impact |
|------|--------|
| Positive | Evidence files are directly reviewable by auditors; git history provides a complete audit trail; no database setup or migration needed; data format matches Tugboat API responses; offline access to all synced data |
| Negative | No transactional guarantees for multi-file operations; query performance degrades with large file counts (mitigated by caching); file naming conventions must be strictly enforced; potential for filesystem-level corruption |
| Neutral | Storage paths are configurable via `StoragePaths` in config; `.cache/` directory provides performance optimization layer |

### Implementation Impact

| Aspect | Assessment |
|--------|------------|
| Effort | Low - JSON marshaling is native Go |
| Skills | Standard Go file I/O and JSON handling |
| Performance | Sub-100ms queries via caching layer; raw filesystem access for cold queries |
| Scalability | Suitable for hundreds of evidence tasks; thousands would need indexing or database migration |
| Security | File permissions (600) for sensitive data; .gitignore excludes credentials |

### Validation

| Success Metric | Review Trigger |
|----------------|----------------|
| Read operations complete in < 100ms with cache | File count exceeds 10,000 and queries slow down |
| All evidence files are valid JSON parseable by standard tools | File corruption detected |
| Git diff of synced data is meaningful and reviewable | Data format changes make diffs unreadable |

---

## ADR-005: VCR Testing Pattern for API Integration Tests

| Date | Status | Deciders | Related | Confidence |
|------|--------|----------|---------|------------|
| 2025-01-10 | Accepted | Engineering Team | ADR-003, ADR-004 | High |

### Context

| Aspect | Description |
|--------|-------------|
| Problem | Integration tests for Tugboat Logic and GitHub APIs require authenticated HTTP requests; running these against live APIs in CI/CD is unreliable due to rate limits, credential management, and network dependencies |
| Current State | Multiple external API integrations (Tugboat, GitHub, Google Workspace, Claude) that need test coverage |
| Requirements | Reproducible tests without live API access; fast test execution in CI/CD; no credential requirements in test environments; ability to record real interactions for test fixtures |

### Decision

We will implement a custom VCR (Video Cassette Recorder) testing infrastructure (`internal/vcr/`) that records HTTP interactions to YAML cassette files and replays them during test execution.

**Key Points**: Custom VCR implementation in `internal/vcr/` | YAML cassette format for readability | Four modes: record, playback, record_once, off | Sensitive data scrubbed from cassettes

### Alternatives

| Option | Pros | Cons | Evaluation |
|--------|------|------|------------|
| Live API testing only | Tests real behavior exactly | Requires credentials in CI, rate-limited, flaky, slow, network-dependent | Rejected: fundamentally incompatible with reliable CI/CD |
| Manual mock/stub creation | Full control over test data, no recording needed | Tedious to maintain, may diverge from actual API behavior, doesn't catch API changes | Rejected: maintenance burden and drift risk too high with 4+ API integrations |
| go-vcr library (dnaeon/go-vcr) | Established library, community maintained | Less control over cassette format, may not handle all Tugboat API patterns, external dependency | Rejected: custom implementation needed for Tugboat-specific patterns and sensitive data scrubbing |
| **Custom VCR implementation** | Full control over recording/playback, integrated sensitive data scrubbing, YAML cassettes for readability, mode-based operation | Must maintain custom code, potential for bugs in replay logic | **Selected: custom implementation provides necessary control for compliance-sensitive testing** |

### Consequences

| Type | Impact |
|------|--------|
| Positive | Tests run without network access or credentials; sub-second test execution for API integration tests; reproducible across environments; cassettes serve as API documentation; sensitive data automatically scrubbed |
| Negative | Custom VCR code must be maintained; cassettes may become stale if APIs change; initial recording requires real credentials; cassette files increase repository size |
| Neutral | VCR mode controlled via `VCR_MODE` environment variable; cassettes stored in `test/integration/fixtures/` |

### Validation

| Success Metric | Review Trigger |
|----------------|----------------|
| Integration tests pass without network access | API changes cause cassette mismatches |
| Integration test suite completes in < 30 seconds | Test execution time grows significantly |
| No credentials present in cassette files | Security audit finds leaked credentials in test fixtures |

---

## ADR-006: Hexagonal Architecture (Ports and Adapters)

| Date | Status | Deciders | Related | Confidence |
|------|--------|----------|---------|------------|
| 2025-01-10 | Accepted | Engineering Team | ADR-001, ADR-004 | High |

### Context

| Aspect | Description |
|--------|-------------|
| Problem | GRCTool integrates with multiple external systems (Tugboat Logic, GitHub, Google Workspace, Claude AI) and must support swapping implementations, adding new tools, and testing in isolation |
| Current State | Greenfield architecture design with known need for 4+ external API integrations and 30+ evidence collection tools |
| Requirements | External system implementations must be swappable (testing, vendor changes); new tools must be addable without modifying core logic; domain models must be independent of infrastructure concerns; clear separation for a growing tool registry |

### Decision

We will adopt hexagonal architecture (ports and adapters) where domain logic defines interfaces (ports) and infrastructure provides implementations (adapters).

**Key Points**: Ports defined as Go interfaces in domain/service packages | Adapters implement ports in `internal/adapters/`, `internal/tugboat/`, `internal/tools/` | Tool registry pattern for 30+ evidence collection tools | Factory pattern for service construction

### Alternatives

| Option | Pros | Cons | Evaluation |
|--------|------|------|------------|
| Layered architecture (traditional N-tier) | Simple, well-understood, easy to start | Tight coupling between layers, difficult to swap implementations, testing requires full stack | Rejected: need to swap adapters for testing and support multiple tool implementations |
| Clean architecture (strict onion) | Strong dependency rule, very pure domain | Over-engineering for a CLI tool, too many abstraction layers, Go idioms favor simplicity | Rejected: excessive abstraction for current project size |
| Microservices | Independent deployment, technology flexibility | Massive over-engineering for a CLI tool, operational complexity | Rejected: entirely inappropriate for single-binary CLI |
| **Hexagonal (ports and adapters)** | Clear interface contracts, easy testing with mocks, swap implementations freely, natural fit for tool registry pattern | Requires discipline to maintain boundaries, more interfaces than strictly necessary | **Selected: best balance of testability, extensibility, and Go idiom alignment** |

### Consequences

| Type | Impact |
|------|--------|
| Positive | Each external integration (Tugboat, GitHub, Claude) is testable via mock adapters; new evidence tools implement `Tool` interface and register in the registry; auth providers implement `AuthProvider` interface and are swappable; domain models (`internal/models/`) have no infrastructure dependencies |
| Negative | Interface proliferation if not managed carefully; Go's implicit interface satisfaction can make dependencies less visible; some boilerplate for adapter wiring |
| Neutral | Go's implicit interface implementation naturally supports ports without explicit declaration; `internal/` package visibility enforces boundaries |

### Implementation Impact

| Aspect | Assessment |
|--------|------------|
| Effort | Medium - interfaces and registries need careful initial design |
| Skills | Understanding of dependency inversion and Go interfaces |
| Performance | Negligible overhead; interface dispatch is fast in Go |
| Scalability | Scales well to 30+ tools and additional adapters |
| Security | Clear boundaries help isolate security-sensitive code (auth, credential handling) |

### Validation

| Success Metric | Review Trigger |
|----------------|----------------|
| New tools can be added by implementing Tool interface + registering | Adding a tool requires modifying core packages |
| Integration tests use mock adapters without infrastructure setup | Tests require live services to run |
| Auth providers are swappable without changing consuming code | Auth changes cascade through multiple packages |

---

## ADR-007: Template Variable Interpolation for Organization Customization

| Date | Status | Deciders | Related | Confidence |
|------|--------|----------|---------|------------|
| 2025-01-10 | Accepted | Engineering Team | ADR-004 | Med |

### Context

| Aspect | Description |
|--------|-------------|
| Problem | Policy and control documents synced from Tugboat Logic contain template placeholders (e.g., `{{organization.name}}`) that must be replaced with organization-specific values when generating evidence or markdown output |
| Current State | 408+ template variable occurrences across 40 policy documents; organization name appears throughout compliance documentation |
| Requirements | Replace template variables consistently across all generated output; support nested variable structures; detect circular references; configurable via `.grctool.yaml` |

### Decision

We will implement a custom template variable interpolation system configured in `.grctool.yaml` under the `interpolation` key, supporting nested variable structures with dot notation flattening and circular reference detection.

**Key Points**: `InterpolationConfig` with nested variable maps | Dot-notation flattening (`organization.name`) | Circular reference detection via DFS | Enabled by default with sensible defaults

### Alternatives

| Option | Pros | Cons | Evaluation |
|--------|------|------|------------|
| Go text/template | Standard library, full template language, well-documented | Over-powered for simple substitution, security risks with arbitrary template execution, different syntax from Tugboat's `{{var}}` format | Rejected: security risk of arbitrary template execution in compliance context |
| Simple string replacement (strings.ReplaceAll) | Trivial to implement, no dependencies | No nested variable support, no circular reference detection, fragile with overlapping variable names | Rejected: insufficient for nested variables and safety requirements |
| Environment variable only substitution | Standard pattern, no config needed | Limited to flat key-value pairs, no nested structure, OS-dependent | Rejected: does not support the nested `organization.name` pattern from Tugboat |
| **Custom interpolation with dot-notation** | Supports nested variables, circular reference detection, configurable defaults, matches Tugboat's placeholder format | Custom code to maintain, limited to string substitution (not full template logic) | **Selected: right level of power for the use case with safety guarantees** |

### Consequences

| Type | Impact |
|------|--------|
| Positive | Consistent organization branding across all 40+ policy documents; configurable per-deployment without modifying synced data; circular reference detection prevents infinite loops; `GetFlatVariables()` provides simple key-value access for consumers |
| Negative | Custom implementation to maintain; limited to string substitution (no conditionals or loops); must keep variable names synchronized with Tugboat's template format |
| Neutral | Default organization name ("Seventh Sense") is set when no variables are configured; interpolation is always enabled |

### Validation

| Success Metric | Review Trigger |
|----------------|----------------|
| All 408+ template occurrences are correctly replaced | New template patterns appear that are not handled |
| No circular reference panics in production | Circular reference detection fails for edge case |
| Variable configuration is intuitive in `.grctool.yaml` | Users report confusion about variable syntax |

---

## ADR-008: zerolog for Structured Logging

| Date | Status | Deciders | Related | Confidence |
|------|--------|----------|---------|------------|
| 2025-01-10 | Accepted | Engineering Team | ADR-001, ADR-006 | High |

### Context

| Aspect | Description |
|--------|-------------|
| Problem | A compliance tool handling sensitive data needs structured logging with field-level redaction, URL sanitization, and configurable output levels; standard library `log` package is insufficient |
| Current State | Multiple output targets needed (console stderr, file); sensitive fields (tokens, passwords, cookies) must be automatically redacted; API URLs contain auth parameters that must be sanitized |
| Requirements | Structured key-value logging; field redaction for sensitive data; URL sanitization; configurable log levels per output target; JSON and text format support; minimal allocation overhead for hot paths |

### Decision

We will use zerolog (`github.com/rs/zerolog`) as the structured logging library, wrapped in a custom logger package (`internal/logger/`) that adds redaction, URL sanitization, and multi-target output configuration.

**Key Points**: zerolog for zero-allocation structured logging | Custom wrapper for redaction and sanitization | Multi-target output (console + file) via `LoggingConfig` | Default redact fields: password, token, key, secret, api_key, cookie

### Alternatives

| Option | Pros | Cons | Evaluation |
|--------|------|------|------------|
| Standard library log | No dependencies, simple | No structured logging, no levels, no field redaction, no JSON output | Rejected: completely insufficient for compliance tool requirements |
| logrus | Popular, structured, well-known | Maintenance mode, higher allocation overhead than zerolog, reflection-based | Rejected: maintenance mode and performance overhead |
| zap (uber-go/zap) | High performance, structured, well-maintained | More complex API, heavier dependency tree, sugared logger adds cognitive load | Rejected: zerolog is simpler and sufficient; zap's complexity not justified |
| slog (Go 1.21+) | Standard library, structured, no external deps | Limited redaction support, newer/less battle-tested for custom output targets, requires Go 1.21+ | Rejected: insufficient redaction and sanitization capabilities out of the box |
| **zerolog** | Zero-allocation JSON logging, chainable API, minimal dependency, excellent performance | Less feature-rich than zap for advanced use cases, requires custom wrapper for redaction | **Selected: performance + simplicity + minimal dependencies** |

### Consequences

| Type | Impact |
|------|--------|
| Positive | Zero-allocation logging on hot paths; automatic redaction of sensitive fields (password, token, key, secret, api_key, cookie); URL sanitization strips auth parameters from logged URLs; configurable per-target log levels (warn for console, info for file) |
| Negative | Custom wrapper code in `internal/logger/` must be maintained; zerolog's chainable API style differs from traditional logging patterns; buffered output requires explicit flush management |
| Neutral | Default configuration provides console (stderr, warn level) and file output (info level) with 5-second flush intervals and 100-entry buffers |

### Validation

| Success Metric | Review Trigger |
|----------------|----------------|
| No sensitive data (tokens, passwords) appears in log output | Security audit finds credential leakage in logs |
| Log output is parseable as structured data (JSON mode) | Log analysis tools cannot parse output |
| Logging adds < 1% overhead to command execution | Profiling shows logging as performance bottleneck |

---

## ADR-009: Agentic Workflow Architecture for Evidence Coordination

| Date | Status | Deciders | Related | Confidence |
|------|--------|----------|---------|------------|
| 2025-01-10 | Accepted | Engineering Team | ADR-006, ADR-004 | Med |

### Context

| Aspect | Description |
|--------|-------------|
| Problem | Evidence collection requires coordinating multiple tools (Terraform, GitHub, Google Workspace), assembling context from policies and controls, generating evidence via Claude AI, and managing the full lifecycle from intake through auditor handoff |
| Current State | 30+ evidence collection tools available; evidence tasks have complex relationships to controls and policies; generation requires multi-step orchestration with AI |
| Requirements | Coordinate multiple tools for a single evidence task; assemble rich context (task + controls + policies + security mappings) for AI generation; manage evidence lifecycle states (no_evidence -> generated -> validated -> submitted -> accepted); support batch operations across multiple tasks |

### Decision

We will implement an agentic workflow architecture where the evidence orchestrator coordinates tool execution, context assembly, AI generation, and lifecycle management through a predictable pipeline: intake, collect, review, package, respond.

**Key Points**: Evidence orchestrator coordinates tool selection and execution | `EvidenceDataPackage` pattern assembles context for AI consumption | `EvidenceTaskState` tracks lifecycle across local and remote state | `SubmissionBatch` enables coordinated multi-task submissions

### Alternatives

| Option | Pros | Cons | Evaluation |
|--------|------|------|------------|
| Manual tool-by-tool execution | Simple, no orchestration code, user has full control | Tedious for 100+ evidence tasks, no context assembly, no lifecycle tracking, error-prone | Rejected: does not scale to real compliance workloads |
| Static pipeline (fixed tool sequence) | Predictable, easy to implement | Cannot adapt to different evidence task requirements, wastes time running irrelevant tools | Rejected: evidence tasks require different tool combinations |
| Event-driven workflow (pub/sub) | Loosely coupled, extensible, handles complex flows | Over-engineering for CLI tool, operational complexity, harder to debug, requires message broker | Rejected: unnecessary infrastructure complexity for single-user CLI |
| **Agentic orchestration** | Adapts tool selection to task requirements, assembles context intelligently, manages lifecycle state, supports batch operations | More complex orchestration code, AI generation quality varies, state management across local and remote | **Selected: necessary complexity for real compliance workflow automation** |

### Consequences

| Type | Impact |
|------|--------|
| Positive | Single command (`evidence generate ET-0001`) triggers full workflow: task analysis, relationship mapping, tool orchestration, AI generation, output formatting; `EvidenceContext` provides comprehensive context to Claude AI; `StateCache` enables efficient status tracking across all tasks; batch submissions coordinate multi-task evidence delivery |
| Negative | Orchestration logic is complex and must handle partial failures; AI generation quality depends on context assembly quality; state synchronization between local and Tugboat requires careful management; `AutomationCapability` heuristics may miscategorize tasks |
| Neutral | Workflow stages (intake, collect, review, package, respond) map to CLI subcommands; evidence lifecycle states are persisted in `.state/evidence_state.yaml` |

### Risks

| Risk | Prob | Impact | Mitigation |
|------|------|--------|------------|
| AI-generated evidence rejected by auditors | Medium | High | Include reasoning and source attribution; human review step before submission |
| Tool orchestration failures leave partial state | Medium | Medium | Idempotent operations; state recovery on restart; clear error messages |
| Context assembly produces irrelevant information | Low | Medium | Focused summaries via `control-summary-generator` and `policy-summary-generator` tools |

### Dependencies

- **Technical**: Claude API for evidence generation; Tugboat API for submission; Go concurrency primitives for parallel tool execution
- **Decisions**: ADR-006 (hexagonal architecture enables tool interface), ADR-004 (JSON storage for evidence persistence)

### Validation

| Success Metric | Review Trigger |
|----------------|----------------|
| >= 80% of generated evidence accepted after single review | Acceptance rate drops below threshold |
| Evidence generation completes in < 5 minutes per task | Generation time exceeds acceptable limits |
| Batch submissions process all tasks without manual intervention | Batch failures require manual recovery |

---

## ADR-010: System of Record Architecture

| Date | Status | Deciders | Related | Confidence |
|------|--------|----------|---------|------------|
| 2026-03-17 | Accepted | Engineering Team | ADR-004, ADR-006, ADR-009 | Med |

### Context

| Aspect | Description |
|--------|-------------|
| Problem | GRCTool currently operates as a data aggregator: it syncs policies, controls, and evidence tasks from Tugboat Logic and stores them locally as cached copies. Tugboat Logic is the source of truth. As GRCTool matures and organizations depend on it for evidence generation, workflow orchestration, and multi-platform integration, this aggregator model creates friction: data ownership is ambiguous, integrations are unidirectional (pull-only), and adding new compliance platforms requires replicating the Tugboat-centric sync model. |
| Current State | All compliance artifacts (policies, controls, evidence tasks) originate from Tugboat Logic via `grctool sync`. GRCTool assigns local reference IDs (ET-NNNN, POL-NNNN) but defers to Tugboat numeric IDs as the canonical identifiers. No local editing of compliance artifacts is supported. Integration is one-way: Tugboat -> GRCTool for data, GRCTool -> Tugboat for evidence submission. |
| Requirements | GRCTool must become the canonical source for compliance artifacts; external systems must integrate bidirectionally; organizations must own their data locally with selective sync to cloud platforms; new compliance platforms must be addable without architectural changes. |

### Decision

GRCTool will become the **system of record** for compliance data. Policies, controls, control mappings, and evidence tasks will be first-class entities managed by GRCTool with a local master index as the canonical registry. External systems (Tugboat Logic, GitHub, Terraform, Google Workspace, future platforms) will integrate through bidirectional adapter interfaces — import, export, and sync — rather than serving as the upstream source of truth.

**Key Points**: Master index with GRCTool-native identifiers | Bidirectional integration adapters (import/export/sync) | Configurable conflict resolution (local_wins, remote_wins, manual, newest_wins) | Tugboat Logic becomes one integration target, not THE source | Phased migration from aggregator to system of record

### Alternatives

| Option | Pros | Cons | Evaluation |
|--------|------|------|------------|
| Keep Tugboat as source of truth | No migration needed, simple model, single source | Platform lock-in, unidirectional only, no local editing, adding new platforms requires separate sync models | Rejected: does not scale to multi-platform integration or local data sovereignty |
| Shared ownership (no single source) | Flexible, no migration needed | Ambiguous data ownership, conflict resolution is undefined, difficult to reason about consistency | Rejected: ambiguity is unacceptable for compliance data that must have a clear audit trail |
| External database as system of record (PostgreSQL, etc.) | Mature tooling, SQL queries, ACID transactions | Violates offline-first and no-external-dependency principles (ADR-004), not git-friendly, deployment complexity | Rejected: contradicts core design principles |
| **GRCTool master index as system of record** | Clear data ownership, local-first, git-friendly, supports bidirectional integration with any platform, no vendor lock-in | Requires master index implementation, conflict resolution logic, migration tooling for existing Tugboat-sourced data | **Selected: aligns with product vision, enables multi-platform integration, preserves data sovereignty** |

### Consequences

| Type | Impact |
|------|--------|
| Positive | Clear data ownership: GRCTool is authoritative for all compliance artifacts. Enables bidirectional integration with any compliance platform, not just Tugboat. Organizations own their data locally and can switch platforms without data loss. Master index provides a stable foundation for cross-framework control mappings and multi-platform evidence collection. |
| Negative | Requires implementing a master index registry and migration tooling. Bidirectional sync introduces conflict resolution complexity. Existing workflows that assume Tugboat-as-source must be updated during migration phases. |
| Neutral | File-based storage model (ADR-004) is preserved; the master index is stored as YAML files in the `.index/` directory. Hexagonal architecture (ADR-006) naturally supports the adapter-based integration model. **NOTE: Per SD-004 and ADR-011, the master index is implemented as the existing `StorageService` enriched with `ExternalIDs` and `SyncMetadata` fields. No separate `.index/` directory is needed.** |

### Implementation Impact

| Aspect | Assessment |
|--------|------------|
| Effort | High - master index, adapter interfaces, conflict resolution, and migration tooling |
| Skills | Existing Go and architecture skills; conflict resolution patterns are well-documented |
| Performance | Minimal overhead; master index is a lightweight YAML registry |
| Scalability | Scales to multiple integration targets without architectural changes |
| Security | Local data sovereignty reduces cloud exposure; selective sync gives organizations control over what data leaves their environment |

### Risks

| Risk | Prob | Impact | Mitigation |
|------|------|--------|------------|
| Migration disrupts existing Tugboat workflows | Medium | Medium | Phased migration (shadow index -> dual-write -> GRCTool-as-source); backward compatibility at each phase |
| Conflict resolution produces incorrect results | Medium | High | Default to `manual` conflict policy; surface conflicts clearly to users; comprehensive test coverage for sync scenarios |
| Master index grows stale if sync is not run | Low | Medium | Health checks and staleness warnings; configurable sync schedules |
| Integration adapter complexity grows with each new platform | Medium | Medium | Standard adapter interface (ADR-006) constrains complexity; shared test infrastructure for adapter validation |

### Dependencies

- **Technical**: ADR-004 (file-based storage provides the storage foundation), ADR-006 (hexagonal architecture provides the adapter pattern), ADR-009 (agentic workflows operate on master index data)
- **Migration**: Existing Tugboat-synced data must be importable into the master index without data loss

### Validation

| Success Metric | Review Trigger |
|----------------|----------------|
| All compliance artifacts have GRCTool-native IDs in the master index | Artifacts found without canonical IDs |
| Bidirectional sync with Tugboat Logic operates without data loss | Sync produces data inconsistencies or silent data loss |
| New integration target can be added by implementing adapter interface only | Adding a platform requires modifying core logic |
| Migration from Tugboat-as-source completes without manual data repair | Migration requires manual intervention for > 5% of artifacts |

---

## ADR-011: Universal Document Provider Framework

| Date | Status | Deciders | Related | Confidence |
|------|--------|----------|---------|------------|
| 2026-03-17 | Accepted | Engineering Team | ADR-006, ADR-010 | Med |

**Implementation status (2026-04-02):** This ADR's decision has been partially
implemented. All domain entity IDs are unified as `string`. `ExternalIDs` and
`SyncMetadata` fields exist on all domain entities. `DataProvider`,
`SyncProvider`, and `ProviderRegistry` interfaces are defined and tested. The
Tugboat adapter is refactored as `TugboatDataProvider`. Remaining gaps:
ContentHash computation, CLI index surface, ProviderInfo metadata.
Status upgraded from "Proposed" to "Accepted" to reflect that the architectural
decision is committed and partially shipped.

### Context

| Aspect | Description |
|--------|-------------|
| Problem | GRCTool's domain model originally had inconsistent ID types: `Policy.ID` was `string`, while `Control.ID` and `EvidenceTask.ID` were `int`. Tugboat Logic was the only data source, hardwired through `SyncService`. The system-of-record vision (ADR-010) required pluggable providers, but no provider abstraction existed. |
| Current State | As of 2026-04-02, the core provider framework is implemented: all domain IDs are `string`, `ExternalIDs` and `SyncMetadata` fields exist on all entities, `DataProvider`/`SyncProvider`/`ProviderRegistry` interfaces are defined in `internal/interfaces/provider.go`, and the Tugboat adapter implements `DataProvider`. The `SyncService` uses the provider registry alongside a direct `tugboat.Client` reference. ContentHash computation and some CLI surfaces remain unimplemented. |
| Requirements | Pluggable data providers that return domain entities; consistent ID types across all domain entities; external ID tracking per provider; backward-compatible migration path for existing Tugboat-synced data; conflict detection for bidirectional sync scenarios |

### Decision

We will introduce a layered Universal Document Provider Framework:

1. **Fix ID types**: Change `Control.ID` and `EvidenceTask.ID` from `int` to `string` across the domain model. This is a breaking change that aligns all entity IDs to `string`, matching `Policy.ID` which already uses string.

2. **Add external ID tracking**: Add `ExternalIDs map[string]string` and `SyncMetadata *SyncMetadata` fields to `Policy`, `Control`, and `EvidenceTask` structs. `ExternalIDs` maps provider names to their native IDs (e.g., `{"tugboat": "12345"}`). `SyncMetadata` tracks per-provider sync timestamps and content hashes.

3. **Define `DataProvider` interface**: A read-only interface for retrieving compliance entities from external sources, with methods returning domain entities: `ListPolicies`, `ListControls`, `ListEvidenceTasks`, `GetPolicy`, `GetControl`, `GetEvidenceTask`, plus `TestConnection` and `Name`.

4. **Define `SyncProvider` interface**: Extends `DataProvider` with write-back operations (`PushPolicy`, `PushControl`, `PushEvidenceTask`, `DeletePolicy`, `DeleteControl`, `DeleteEvidenceTask`) and `DetectChanges` for conflict detection.

5. **Implement `ProviderRegistry`**: A registry that maps provider names to `DataProvider` or `SyncProvider` instances, manages provider lifecycle (initialization, health checks, shutdown).

6. **Refactor Tugboat as `DataProvider`**: Wrap the existing `tugboat.Client` and `TugboatToDomain` adapter into a `TugboatDataProvider` struct that implements the `DataProvider` interface. This is the proof-of-concept provider.

7. **Master index is the existing `StorageService`**: The local file storage, already implemented in `internal/storage/`, serves as the master index. No new `.index/` directory is needed. The master index is enriched with `ExternalIDs` and `SyncMetadata` fields on domain entities, stored in the existing JSON files.

**Key Points**: Consistent string IDs across all entities | ExternalIDs map for multi-provider tracking | DataProvider/SyncProvider interface hierarchy | ProviderRegistry for lifecycle management | Existing StorageService as master index | Tugboat refactored as first DataProvider

### Alternatives

| Option | Pros | Cons | Evaluation |
|--------|------|------|------------|
| Keep Tugboat hardwired, add each new provider ad-hoc | No breaking changes, fastest short-term path | Each new provider duplicates sync orchestration; no consistent interface; impossible to test providers uniformly | Rejected: does not scale; violates ADR-006 hexagonal architecture principles |
| Use generic `map[string]interface{}` for all entities | Maximum flexibility, no schema changes needed | Loses type safety; Go's type system provides no compile-time guarantees; every consumer must type-assert | Rejected: unacceptable for a compliance tool where data integrity is critical |
| Introduce a full ORM/database layer (SQLite, bbolt) | Structured queries, ACID transactions, proper indexing | Contradicts ADR-004 (JSON-based storage); not human-readable; not git-diffable; adds external dependency | Rejected: violates core design principles |
| **Universal Document Provider Framework** | Consistent interface contract; type-safe domain entities; existing storage as master index; incremental migration; testable via mock providers | Breaking change to Control.ID and EvidenceTask.ID; adapter code must be updated; requires careful migration | **Selected: right level of abstraction for multi-provider architecture without over-engineering** |

### Consequences

| Type | Impact |
|------|--------|
| Positive | All domain entities use consistent `string` IDs, eliminating `strconv.Itoa` / `fmt.Sprintf("%d", ...)` scattered through adapter and service code. ExternalIDs enable tracking the same entity across multiple providers. DataProvider interface enables clean testing with mock providers. ProviderRegistry supports runtime provider discovery. Existing StorageService requires no structural changes — just richer entity fields. |
| Negative | Breaking change: `Control.ID int` → `string` and `EvidenceTask.ID int` → `string` affects 33+ files including adapters, services, formatters, tools, storage, commands, and tests. `EvidenceRecord.TaskID int` must also change to `string`. All `logger.Int("control_id", ...)` calls must become `logger.String("control_id", ...)`. Existing JSON data on disk needs migration (numeric IDs → string IDs). |
| Neutral | The `SyncService` struct will be refactored to accept a `DataProvider` instead of `*tugboat.Client`, but its orchestration logic (fetch list → fetch details → convert → save) remains the same pattern. |

### Implementation Impact

| Aspect | Assessment |
|--------|------------|
| Effort | High — ID type change touches 33+ files; provider interfaces and registry are medium effort; Tugboat refactoring is medium effort |
| Skills | Standard Go interface design; familiarity with existing adapter pattern |
| Performance | Negligible — interface dispatch overhead; ExternalIDs/SyncMetadata add small JSON payload |
| Scalability | Each new provider implements DataProvider interface; no core changes needed |
| Security | Provider credentials managed per-provider; no cross-provider credential leakage |

### Risks

| Risk | Prob | Impact | Mitigation |
|------|------|--------|------------|
| ID type migration breaks existing stored data | High | High | Write a migration tool that reads existing JSON files and converts numeric IDs to strings; run as part of upgrade |
| Breaking change scope is larger than estimated | Medium | Medium | Comprehensive `grep` for all `Control.ID` and `EvidenceTask.ID` usage before starting; incremental PR strategy |
| Provider interface is too narrow or too wide | Medium | Medium | Start with Tugboat as reference implementation; iterate interface based on second provider (AccountableHQ from SD-001) |

### Dependencies

- **Technical**: ADR-006 (hexagonal architecture provides the port/adapter pattern), ADR-010 (system of record establishes the vision this ADR implements)
- **Sequencing**: ID type migration must happen before provider interface implementation; Tugboat DataProvider is proof-of-concept before adding AccountableHQ

### Validation

| Success Metric | Review Trigger |
|----------------|----------------|
| All domain entity IDs are `string` type with no `int` ID fields remaining | New entity added with `int` ID |
| Tugboat provider implements DataProvider and passes existing integration tests | Integration tests fail after refactoring |
| Second provider (AccountableHQ) can be added by implementing DataProvider only | Adding a provider requires modifying core sync logic |
| ExternalIDs correctly track entity origins across providers | Entity provenance is lost during sync |

---

## References

- [System Architecture](/home/erik/Projects/grctool/docs/helix/02-design/architecture/system-design.md)
- [Security Design](/home/erik/Projects/grctool/docs/helix/02-design/security-architecture/security-design.md)
- [Product Requirements](/home/erik/Projects/grctool/docs/helix/01-frame/prd/requirements.md)
- [Architecture Overview](/home/erik/Projects/grctool/docs/ARCHITECTURE.md)
