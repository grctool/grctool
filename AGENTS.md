# AGENTS.md — Engineering Guidelines and Best Practices

This document consolidates our coding, testing, security, and operational best practices for contributors and AI coding agents working in this repository.

Scope
- Language: Go 1.23+
- App: grctool — a Go CLI integrating with Tugboat Logic for compliance automation and evidence generation
- Domains: API clients, local JSON storage, CLI commands (Cobra), AI assistance (Claude), Terraform parsing

Core Principles
- Prefer simple, minimal, composable solutions
- Design for change; small modules with clear interfaces
- Strong typing, explicit errors with context, and predictable behavior
- Least privilege and secure-by-default for credentials and data
- Deterministic dev/test environments (VCR recordings for HTTP)

Project Conventions
- CLI: Cobra commands under grctool/cmd; business logic lives in internal/* packages
- Packages: internal/{auth, claude, config, domain, formatters, logger, markdown, models, orchestrator, registry, services, storage, tools, transport, tugboat, utils, vcr}
- Config: YAML via Viper; environment variable substitution supported
- Storage: Local JSON data_dir and cache_dir; deterministic filenames: {Ref}_{ID}_{Sanitized}.json
- Evidence: Prompts, generated outputs, and metadata live under grctool/evidence

Tools & Evidence Collection
- All evidence collection tools are self-documenting via CLI help system
- Tool discovery: `./bin/grctool tool --help` lists all available tools
- Tool help: `./bin/grctool tool <name> --help` shows detailed usage for specific tools
- Tools output structured JSON with evidence source metadata for auditability
- Common parameters: `--task-ref` (evidence task reference), `--output json`, `--quiet`
- Tool categories: evidence tasks, GitHub analysis, Terraform scanning, Google Workspace, storage operations
- Example usage: `./bin/grctool tool evidence-task-list` or `./bin/grctool tool evidence-task-details --task-ref ET-0001`

Coding Standards (Go)
- Formatting: go fmt; run make fmt and make lint before commit
- Naming:
  - Types/structs: descriptive without prefixes (PolicyFormatter, SyncService)
  - Constructors: NewXxx (e.g., NewPolicyFormatter())
  - Avoid “Legacy/Old” prefixes; remove/replace old code instead
- Errors:
  - Wrap with context: fmt.Errorf("context: %w", err)
  - Avoid panics outside of main(); return errors
  - Define sentinel errors for expected conditions
- Interfaces:
  - Define minimal, consumer-driven interfaces in the consumer package
  - Accept interfaces, return concrete types where practical
- Concurrency:
  - Pass context.Context; respect cancellation
  - Avoid goroutine leaks; use errgroup where helpful
  - Guard shared state; prefer immutable data
- Logging:
  - Use internal/logger with structured fields, not fmt.Println
  - Redact secrets; never log tokens, cookies, API keys
  - Include operation, component, duration_ms; use Debug for verbose traces

Configuration & Secrets
- Config in YAML; environment substitution for secrets (e.g., ${CLAUDE_API_KEY})
- Do not commit credentials; use env vars or OS keychain where available
- Browser-based auth for Tugboat Logic (macOS Safari flow), no Tugboat API keys supported.
- Validate config on startup; fail fast with actionable errors

Authentication Flow (Tugboat)
- grctool auth login → opens Safari, user logs in, tool captures cookies and extracts bearer.
- Credentials saved to .grctool.yaml; grctool auth status validates; grctool auth logout clears.
- Expected unauthenticated error example:
  $ make sync
  Error: connection test failed: failed to extract bearer token: no cookie header provided

HTTP, Retries, and Rate Limiting
- Centralize HTTP in internal/tugboat client
- Retries with backoff on transient errors (5xx, timeouts, rate limit)
- Respect configured rate limits; add jitter
- Sanitize/record only non-sensitive request metadata in logs

Testing Strategy
We use a comprehensive 4-tier test organization that separates authentication requirements:

1. **Unit Tests** (`make test`, `make test-unit`)
   - Fast, deterministic, no external dependencies
   - Pure function and interface-based testing for tool logic, config, formatters, etc.
   - Prefer: 1) Pure functions, 2) Interface + stubs, 3) VCR recordings, 4) Mocks only as last resort
   - Build tag: `//go:build !e2e` (excludes auth-dependent tests)
   - Target: ~2-3 second execution for fast feedback

2. **Integration Tests** (`make test-integration`)
   - Local data dependencies, tool orchestration testing
   - TestFullWorkflow validates complete evidence pipeline
   - Cross-tool validation and performance testing
   - No external API calls, works with local fixtures

3. **Functional Tests** (`make test-functional`)
   - Built binary required, CLI end-to-end testing
   - Build tag: `//go:build functional`
   - Real filesystem operations, configuration loading
   - Tests complete application through CLI interface

4. **End-to-End Tests** (`make test-e2e`)
   - Authentication required (GitHub, Tugboat APIs)
   - Build tag: `//go:build e2e`
   - Environment variables: GITHUB_TOKEN, TUGBOAT_* configs
   - Properly skip when authentication unavailable

**Recommended Commands:**
- `make test-no-auth` - All core tests without authentication (CI-friendly)
- `make test-all` - Complete test suite including authenticated tests
- `make ci` - CI checks without authentication requirements
- `make ci-with-auth` - Full CI with authentication tests

**VCR Framework:**
- Record and playback HTTP to eliminate live API in tests
- Modes: off | record | playback | record_once
- Cassettes in grctool/internal/tugboat/testdata/vcr_cassettes/
- Headers redacted: authorization, cookie, api_key/token/password

**Coverage:** make test-coverage for unit tests, make test-coverage-all for full suite

## Testing Best Practices

### Testing Philosophy

We follow the **Classical/Detroit School of TDD**, which emphasizes:

- **State-based testing**: Test what the system does, not how it does it
- **Real implementations**: Use actual objects and minimal test doubles
- **Behavioral verification**: Assert on outcomes and side effects, not interactions
- **Test-driven design**: Let tests drive better code organization and interfaces

**Mock Avoidance Hierarchy** (prefer higher options):
1. **Pure functions** - No dependencies, directly testable
2. **Interfaces with stubs** - Simple implementations for testing
3. **VCR recordings** - Captured real HTTP interactions
4. **Test fixtures** - Static data in testdata/ folders
5. **Mocks** - Only as absolute last resort with documented justification

This approach aligns with Go's philosophy of simplicity and produces more maintainable, refactor-friendly tests.

### Prefer Pure Functions
- Extract business logic into pure functions that return values based on inputs
- Test pure functions directly without any dependencies
- **Refactoring Pattern**: Separate pure core logic from impure I/O shell
- Example: `ParseRepositoryPermissions(data []byte) (Permissions, error)`

**Before (impure function):**
```go
func (s *Service) ProcessEvidence(taskID string) error {
    data, err := s.db.GetEvidence(taskID)     // I/O dependency
    if err != nil { return err }
    
    // Business logic buried with I/O
    score := 0.0
    for _, evidence := range data {
        if evidence.Complete { score += evidence.Weight }
    }
    
    return s.db.SaveScore(taskID, score)      // I/O dependency
}
```

**After (pure function extracted):**
```go
// Pure function - easily testable
func CalculateEvidenceScore(evidence []Evidence) float64 {
    score := 0.0
    for _, e := range evidence {
        if e.Complete { score += e.Weight }
    }
    return score
}

// Thin I/O shell
func (s *Service) ProcessEvidence(taskID string) error {
    data, err := s.db.GetEvidence(taskID)
    if err != nil { return err }
    
    score := CalculateEvidenceScore(data)    // Pure function call
    return s.db.SaveScore(taskID, score)
}
```

### Consumer-Defined Interfaces

Follow Go's principle: **"Accept interfaces, return structs"**

- **Define interfaces where they're used**, not where they're implemented
- **Keep interfaces small and focused** (1-3 methods ideal)
- **Let consumers specify exactly what they need**

```go
// ❌ Producer-defined interface (anti-pattern)
// In database/store.go
type Store interface {
    GetUser(id string) (User, error)
    CreateUser(u User) error
    UpdateUser(u User) error
    DeleteUser(id string) error
    ListUsers() ([]User, error)
    // ... many methods consumers don't need
}

// ✅ Consumer-defined interface (preferred)
// In evidence/processor.go
type EvidenceStore interface {
    GetEvidence(taskID string) ([]Evidence, error)  // Only what this consumer needs
}

type Processor struct {
    store EvidenceStore  // Interface, not concrete type
}
```

### Use Interfaces with Stubs
- Define minimal interfaces for dependencies
- Create simple stub implementations for testing
- Avoid mock frameworks unless absolutely necessary

### VCR for External APIs
- Record real HTTP interactions once
- Replay in tests for deterministic results
- No need to mock HTTP clients

### Table-Driven Tests

Use table-driven tests for multiple similar test cases. This pattern reduces duplication and makes it easy to add new test cases.

**Prefer Maps over Slices:**
- Better IDE navigation (can collapse individual test cases)
- Undefined iteration order helps catch test dependencies
- Clear separation of test name from fixtures

```go
func TestCalculateEvidenceScore(t *testing.T) {
    tests := map[string]struct {
        evidence []Evidence
        want     float64
    }{
        "no evidence": {
            evidence: []Evidence{},
            want:     0.0,
        },
        "single complete evidence": {
            evidence: []Evidence{{Complete: true, Weight: 1.0}},
            want:     1.0,
        },
        "mixed complete and incomplete": {
            evidence: []Evidence{
                {Complete: true, Weight: 0.8},
                {Complete: false, Weight: 0.5},
                {Complete: true, Weight: 0.2},
            },
            want: 1.0, // 0.8 + 0.2
        },
        "all incomplete": {
            evidence: []Evidence{
                {Complete: false, Weight: 1.0},
                {Complete: false, Weight: 0.5},
            },
            want: 0.0,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            got := CalculateEvidenceScore(tt.evidence)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

**Functional Builder Pattern** for complex test fixtures:
```go
type EvidenceBuilder struct {
    evidence Evidence
}

func NewEvidence() *EvidenceBuilder {
    return &EvidenceBuilder{
        evidence: Evidence{
            Weight:     1.0,      // Sensible defaults
            Complete:   false,
            Timestamp:  time.Now(),
        },
    }
}

func (b *EvidenceBuilder) WithWeight(w float64) *EvidenceBuilder {
    b.evidence.Weight = w
    return b
}

func (b *EvidenceBuilder) Complete() *EvidenceBuilder {
    b.evidence.Complete = true
    return b
}

func (b *EvidenceBuilder) Build() Evidence {
    return b.evidence
}

// Usage in tests
tests := map[string]struct {
    evidence []Evidence
    want     float64
}{
    "builder pattern example": {
        evidence: []Evidence{
            NewEvidence().WithWeight(0.8).Complete().Build(),
            NewEvidence().WithWeight(0.2).Build(), // incomplete
        },
        want: 0.8,
    },
}
```

### Test Fixtures and testdata/ Pattern

For large or complex test data, use the `testdata/` folder pattern:

```
internal/formatters/
├── formatter.go
├── formatter_test.go
└── testdata/
    ├── evidence_task.json          # Input fixture
    ├── evidence_task.golden        # Expected output
    ├── complex_policy.json
    └── complex_policy.golden
```

**File-driven tests** for complex inputs/outputs:
```go
func TestFormatEvidenceTask(t *testing.T) {
    testFiles, err := filepath.Glob("testdata/*.json")
    require.NoError(t, err)

    for _, testFile := range testFiles {
        name := strings.TrimSuffix(filepath.Base(testFile), ".json")
        t.Run(name, func(t *testing.T) {
            // Read input
            input, err := os.ReadFile(testFile)
            require.NoError(t, err)
            
            var task EvidenceTask
            require.NoError(t, json.Unmarshal(input, &task))

            // Execute function
            result := FormatEvidenceTask(task)

            // Compare with golden file
            goldenFile := strings.Replace(testFile, ".json", ".golden", 1)
            if os.Getenv("UPDATE_GOLDEN") == "true" {
                os.WriteFile(goldenFile, []byte(result), 0644)
                return
            }

            expected, err := os.ReadFile(goldenFile)
            require.NoError(t, err)
            assert.Equal(t, string(expected), result)
        })
    }
}
```

**Update golden files:** `UPDATE_GOLDEN=true go test ./...`

### When Mocks Might Be Acceptable

While we prefer the alternatives above, there are rare cases where mocks provide value:

1. **Time-dependent behavior** - Testing timeouts, retries, or time-based logic
2. **Error simulation** - Testing error paths that are hard to trigger naturally
3. **Third-party SDK limitations** - When external libraries don't provide testable interfaces
4. **Performance-critical paths** - When real dependencies are prohibitively slow
5. **Behavioral verification** - Ensuring specific interactions occur (use sparingly)

**When using mocks, always document the justification:**
```go
// TestRetryBehavior uses a mock because we need to verify
// that exactly 3 retry attempts are made within the timeout period.
// Real HTTP clients would make this test flaky and slow.
func TestRetryBehavior(t *testing.T) {
    mockClient := &MockHTTPClient{}
    mockClient.On("Do", mock.Anything).Return(nil, errors.New("server error")).Times(3)
    
    service := NewService(mockClient)
    err := service.FetchData(context.Background())
    
    assert.Error(t, err)
    mockClient.AssertExpectations(t)
}
```

**Signs you might be over-mocking:**
- Test setup is longer than the actual test
- Mocks break frequently when refactoring
- You're mocking internal implementation details
- The mock is more complex than a real implementation would be

### Refactoring Guidelines for Testability

**Dependency Injection Patterns:**

```go
// ❌ Hard to test - dependencies created internally
type Service struct {
    client *http.Client
}

func NewService() *Service {
    return &Service{
        client: &http.Client{Timeout: 30 * time.Second},
    }
}

// ✅ Easy to test - dependencies injected
type Service struct {
    client HTTPClient  // Interface
}

type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}

func NewService(client HTTPClient) *Service {
    return &Service{client: client}
}

// Production code
func main() {
    httpClient := &http.Client{Timeout: 30 * time.Second}
    service := NewService(httpClient)
}

// Test code
func TestService(t *testing.T) {
    stubClient := &StubHTTPClient{
        responses: map[string]*http.Response{
            "GET /api/data": {StatusCode: 200, Body: ...},
        },
    }
    service := NewService(stubClient)
    // ... test with predictable responses
}
```

**Functional Options Pattern** for complex configuration:
```go
type ServiceConfig struct {
    timeout    time.Duration
    retries    int
    baseURL    string
    client     HTTPClient
}

type ServiceOption func(*ServiceConfig)

func WithTimeout(d time.Duration) ServiceOption {
    return func(c *ServiceConfig) { c.timeout = d }
}

func WithClient(client HTTPClient) ServiceOption {
    return func(c *ServiceConfig) { c.client = client }
}

func NewService(opts ...ServiceOption) *Service {
    config := &ServiceConfig{
        timeout: 30 * time.Second,  // defaults
        retries: 3,
        baseURL: "https://api.example.com",
        client:  &http.Client{},
    }
    
    for _, opt := range opts {
        opt(config)
    }
    
    return &Service{config: config}
}

// Easy to test with custom client
func TestService(t *testing.T) {
    service := NewService(
        WithTimeout(time.Second),
        WithClient(&StubHTTPClient{}),
    )
    // ... test
}
```

### Test Code Organization
```go
// Pure function - testable without any setup
func CalculateComplianceScore(evidence []Evidence) float64 {
    // Pure calculation logic
}

// Interface for dependencies
type DataSource interface {
    GetEvidence(taskID string) ([]Evidence, error)
}

// Stub implementation for tests
type StubDataSource struct {
    Evidence map[string][]Evidence
}

func (s *StubDataSource) GetEvidence(taskID string) ([]Evidence, error) {
    if evidence, exists := s.Evidence[taskID]; exists {
        return evidence, nil
    }
    return nil, fmt.Errorf("task %s not found", taskID)
}

// Tool uses interface
type ComplianceTool struct {
    source DataSource  // Interface, not concrete type
}
```

CLI UX & DX
- Clear, consistent command verbs and flags; helpful --help
- Idempotent operations; safe re-runs
- Human-friendly errors and summary output; machine-friendly JSON when needed
- Default sensible paths; allow overrides via flags and config

Security Best Practices
- Principle of least privilege in all external integrations
- Redact secrets in logs; central redaction list in config where applicable
- Evidence outputs should be reviewed prior to submission
- Regularly run make security-scan and make vulnerability-check
- SBOM and dependency hygiene recommended for releases

Performance & Reliability
- Bound timeouts for all I/O
- Stream large payloads; avoid loading entire files when unnecessary
- Cache raw API responses in cache_dir when helpful (for debug/VCR)
- Measure and log durations of key operations

Documentation Practices
- Keep README and consolidated docs current with code
- Update TECHNICAL_ARCHITECTURE.md upon notable architectural changes
- Document new commands under grctool/README.md and cross-link

Release & Versioning (Recommended)
- Semantic Versioning (MAJOR.MINOR.PATCH)
- Tag releases; include changelog of features, fixes, security updates
- Provide platform builds via make build-all (artifacts to dist/)

Commit & PR Conventions
- Conventional Commits for messages: feat:, fix:, docs:, refactor:, test:, chore:
- Small, atomic PRs with clear scope and tests; avoid mixing refactors with features
- Link issues and include a brief rationale in the PR description
- Require at least one review; address comments before merge

Static Analysis & CI Gates
- Run: make fmt lint vet test-no-auth before pushing; CI runs make ci (deps, fmt, vet, lint, test-no-auth, security-scan)
- Use make test-no-auth for fast development feedback (2-3 seconds)
- Use make test-all for comprehensive validation when authentication available
- Prefer staticcheck if available in the toolchain; treat warnings as opportunities to improve
- Maintain go.mod/go.sum hygiene; avoid unused deps; run go mod tidy as part of make deps
- SBOM/vulnerability checks before release: make vulnerability-check; consider cyclonedx-gomod

AI Agent-Specific Guidance
- Do not introduce new architectural patterns without prior consolidation
- Prefer extending existing happy paths and modules
- Add/adjust tests alongside code changes following 4-tier organization
- For new HTTP endpoints, add VCR coverage plans and security redaction rules
- Keep changes minimal and reversible unless explicitly directed otherwise
- Use `make test-no-auth` for fast feedback during development
- Tag authentication-dependent tests with `//go:build e2e`
- Ensure new tests can run without external API dependencies where possible

Issue Tracking with git-bug
- We use git-bug for distributed issue tracking (no GitHub issues)
- List issues: `git-bug bug` or `git-bug bug status:open`
- Create issue: `git-bug bug new --title "Title" --message "Description"`
- View issue: `git-bug bug show <id>`
- Close issue: `git-bug bug close <id>`
- Add comment: `git-bug bug comment <id> --message "Comment"`
- Issues are stored in git objects and sync with the repository
- Use labels for categorization: `git-bug bug new --title "Title" --label type:refactor`

Checklist Before Opening a PR
- [ ] Code compiles (make build)
- [ ] Core tests pass without authentication (make test-no-auth)
- [ ] Unit tests pass locally (make test or make test-unit)
- [ ] Integration tests pass (make test-integration)
- [ ] Lint and fmt pass (make fmt lint)
- [ ] Security scans pass (make security-scan, make vulnerability-check)
- [ ] Docs updated (README, TECHNICAL_ARCHITECTURE, DEVELOPMENT_PLAN if applicable)
- [ ] No secrets or credentials in code, tests, or docs
- [ ] Authentication-dependent tests properly tagged with //go:build e2e
- [ ] New tests follow the 4-tier organization (unit/integration/functional/e2e)

**Testing-Specific Checklist:**
- [ ] Business logic extracted into pure functions where possible
- [ ] Interfaces defined by consumers, not producers ("accept interfaces, return structs")
- [ ] Table-driven tests used for multiple similar test cases (prefer maps over slices)
- [ ] Test fixtures placed in testdata/ folders when appropriate
- [ ] Any mocks have documented justification for why alternatives weren't suitable
- [ ] Tests focus on behavior (what the code does) not implementation (how it does it)
- [ ] New dependencies are injected rather than created internally
- [ ] Complex test setups use functional builder pattern for readability



---

Structured Logging — How to use
- Always prefer internal/logger over fmt.Print* in internal packages. CLI commands may use cmd.Print* for user-facing messages, but diagnostics go through the logger.
- Initialize logger once per command path using config.Logging; create component/operation child loggers in hot paths.
- Include consistent fields: component, operation, duration_ms when measuring, resource_id/type when applicable.
- Redact secrets: authorization, cookie, api_key/token/password never appear in logs. Use logger’s built-in redaction where provided and avoid logging entire sensitive payloads.
- Context: where a context.Context is available, pass it through; derive request IDs or correlation data at the command boundary if supported.
- CLI vs Internal:
  - CLI: Human messages via cmd.Print*, diagnostics via logger.
  - Internal packages: Do not write directly to stdout/stderr. Use logger for diagnostics. For user-facing text routed from services, use appcontext output helpers (appcontext.Printf) that respect command writers.
- Examples (illustrative):
  - log := logger.WithFields(logger.Field{"component","tugboat"}, logger.Field{"operation","sync_policies"})
  - log.Info("request", logger.Field{"method","GET"}, logger.Field{"path","/api/policies"})
  - start := time.Now(); defer func(){ log.Info("completed", logger.Field{"duration_ms", time.Since(start).Milliseconds()}) }()

VCR Framework — How to use
- Purpose: Record HTTP interactions to “cassettes” for deterministic test playback; avoid live HTTP in unit tests/CI.
- Modes: off | record | playback | record_once.
  - playback is the CI default; tests must pass with no live HTTP.
  - record_once records only when a cassette is missing, else plays back.
- Environment:
  - VCR_MODE=playback make test    # Recommended default
  - VCR_MODE=record_once make test  # Devs recording new coverage
  - VCR_MODE=record make test       # Force re-record (use sparingly)
- Cassette location: grctool/internal/tugboat/testdata/vcr_cassettes/ (deterministic names based on method/path/query-hash).
- Redaction: authorization and cookie headers are redacted; api_key/token/password are sanitized. Review cassettes before commit.
- Test pattern:
  - Construct vcr.Config from env (default playback when unset) and inject into clients.
  - Prefer table-driven tests; parallelize where safe.
- Expectations:
  - Never rely on live Tugboat API in CI.
  - When adding new client endpoints, add cassettes and tests concurrently.

Greenfield Development Stance — Refactoring & Tests
- Modify in place: When changing behavior or fixing bugs, update the existing types and functions. Do not create duplicate files, _new variants, or rename old code to *_legacy. Do the hard work of refactoring now.
- Update tests with changes: Any change in behavior must be reflected in tests in the same PR. Keep unit tests deterministic; expand VCR coverage for new HTTP paths.
- Forward momentum: Prefer removing obsolete branches/flags over preserving legacy pathways. Avoid suggesting that we “skip the hard thing” or delete it without replacement—solve it properly.
- Backwards compatibility: Only maintain compatibility when explicitly required by product constraints. Otherwise, simplify and move forward.
- Review checklist:
  - Does this change alter public behavior? If yes, adjust tests and docs.
  - Are there any newly introduced *_legacy or *_new files? If so, refactor instead.
  - Are logs structured and secrets redacted? Are user messages separated from diagnostics?
  - If HTTP behavior changed, did we add/update cassettes and keep CI in playback?

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
