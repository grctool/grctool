# GRCTool Architecture

This document describes the architecture, design patterns, and implementation details of GRCTool.

## Table of Contents

- [Overview](#overview)
- [System Architecture](#system-architecture)
- [Directory Structure](#directory-structure)
- [Core Components](#core-components)
- [Design Patterns](#design-patterns)
- [Data Flow](#data-flow)
- [Extension Points](#extension-points)
- [Testing Strategy](#testing-strategy)
- [Security Considerations](#security-considerations)

## Overview

GRCTool is a CLI application for automating compliance evidence collection through Tugboat Logic integration. It follows a clean architecture pattern with clear separation between CLI, business logic, and external integrations.

### Key Characteristics

- **Language**: Go 1.21+
- **Architecture**: Hexagonal (Ports & Adapters)
- **CLI Framework**: Cobra
- **Storage**: JSON-based local storage with caching
- **API Integration**: REST clients for Tugboat Logic, GitHub, Google Workspace
- **AI Integration**: Claude API for evidence generation
- **Testing**: Three-tier strategy (unit, integration, functional)

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Interface                            │
│                    (cmd/ - Cobra Commands)                       │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                    Application Core                              │
│                                                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐            │
│  │  Services   │  │ Orchestrator│  │   Registry   │            │
│  │             │  │             │  │              │            │
│  └──────┬──────┘  └──────┬──────┘  └──────┬───────┘            │
│         │                │                 │                     │
│  ┌──────▼────────────────▼─────────────────▼───────┐            │
│  │            Domain Models & Logic                 │            │
│  │     (Policies, Controls, Evidence Tasks)         │            │
│  └──────────────────────┬───────────────────────────┘            │
└─────────────────────────┼───────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
┌───────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐
│   Adapters   │  │   Storage   │  │    Tools    │
│              │  │             │  │             │
│ - Tugboat    │  │ - JSON      │  │ - Terraform │
│ - GitHub     │  │ - Cache     │  │ - GitHub    │
│ - Claude AI  │  │ - Templates │  │ - Google WS │
│ - Google WS  │  │             │  │             │
└──────────────┘  └─────────────┘  └─────────────┘
```

### Layers

1. **CLI Layer** (`cmd/`) - User-facing commands
2. **Application Layer** (`internal/services/`, `internal/orchestrator/`) - Business logic
3. **Domain Layer** (`internal/domain/`, `internal/models/`) - Core entities
4. **Infrastructure Layer** (`internal/adapters/`, `internal/storage/`, `internal/tools/`) - External integrations

## Directory Structure

```
grctool/
├── cmd/                          # CLI command implementations
│   ├── auth.go                  # Authentication commands
│   ├── config.go                # Configuration management
│   ├── sync.go                  # Data synchronization
│   ├── evidence.go              # Evidence management
│   ├── policy.go                # Policy operations
│   ├── control.go               # Control operations
│   └── tool_*.go                # Evidence collection tool commands
│
├── internal/                     # Private application code
│   │
│   ├── adapters/                # External service adapters
│   │   └── Interfaces for external systems
│   │
│   ├── appcontext/              # Application context management
│   │   └── Shared application state
│   │
│   ├── auth/                    # Authentication
│   │   ├── browser_auth.go     # Safari cookie extraction (macOS)
│   │   └── token_manager.go    # Credential storage
│   │
│   ├── config/                  # Configuration management
│   │   ├── config.go           # Configuration loading and validation
│   │   └── defaults.go         # Default values
│   │
│   ├── domain/                  # Domain models
│   │   ├── policy.go           # Policy entity
│   │   ├── control.go          # Control entity
│   │   └── evidence_task.go    # Evidence task entity
│   │
│   ├── formatters/              # Output formatters
│   │   ├── csv.go              # CSV output
│   │   ├── markdown.go         # Markdown output
│   │   └── json.go             # JSON output
│   │
│   ├── logger/                  # Logging infrastructure
│   │   └── logger.go           # Structured logging
│   │
│   ├── models/                  # Data models (DTOs)
│   │   └── API request/response structures
│   │
│   ├── orchestrator/            # Application orchestration
│   │   └── Coordinates services and workflows
│   │
│   ├── registry/                # Tool and entity registration
│   │   └── evidence_task_registry.go  # Evidence task registry
│   │
│   ├── services/                # Business services
│   │   ├── config/             # Config service
│   │   ├── evidence/           # Evidence generation service
│   │   ├── submission/         # Evidence submission service
│   │   └── validation/         # Validation service
│   │
│   ├── storage/                 # Data persistence
│   │   ├── json_store.go       # JSON file storage
│   │   ├── cache.go            # Performance cache
│   │   └── docs/               # Document storage logic
│   │
│   ├── templates/               # Templates
│   │   └── prompts/            # Claude AI prompts
│   │
│   ├── tools/                   # Evidence collection tools
│   │   ├── interface.go        # Tool interface
│   │   ├── registry.go         # Tool registry
│   │   ├── terraform/          # Terraform tools
│   │   │   ├── analyzer.go
│   │   │   ├── indexer.go
│   │   │   └── snippets.go
│   │   ├── github/             # GitHub tools
│   │   │   ├── permissions.go
│   │   │   ├── workflows.go
│   │   │   └── security.go
│   │   └── types/              # Tool type definitions
│   │
│   ├── transport/               # HTTP transport layer
│   │   └── HTTP clients and middleware
│   │
│   ├── tugboat/                 # Tugboat Logic API client
│   │   ├── client.go           # API client
│   │   ├── auth.go             # Authentication
│   │   └── models/             # API models
│   │
│   ├── utils/                   # Utility functions
│   │   └── Common helpers
│   │
│   └── vcr/                     # VCR testing infrastructure
│       ├── vcr.go              # HTTP recording/playback
│       ├── cassette.go         # Cassette management
│       └── matcher.go          # Request matching
│
├── test/                        # Test suites
│   ├── integration/            # Integration tests (VCR-based)
│   │   ├── tugboat_test.go
│   │   ├── github_test.go
│   │   └── fixtures/           # VCR cassettes
│   │
│   └── functional/             # Functional CLI tests
│       ├── cli_test.go
│       └── workflows_test.go
│
├── configs/                     # Configuration examples
│   └── example.yaml
│
├── docs/                        # Documentation
│   ├── ARCHITECTURE.md         # This file
│   ├── 01-User-Guide/
│   ├── 02-Features/
│   └── 04-Development/
│
└── scripts/                     # Build and utility scripts
    └── install.sh
```

## Core Components

### 1. CLI Layer (`cmd/`)

Built with [Cobra](https://github.com/spf13/cobra), the CLI provides user-facing commands.

**Key Commands:**
- `auth` - Authentication management
- `sync` - Synchronize data from Tugboat Logic
- `evidence` - Evidence management (list, generate, submit)
- `policy` - Policy operations
- `control` - Control operations
- `tool` - Evidence collection tool execution

**Command Pattern:**
```go
var syncCmd = &cobra.Command{
    Use:   "sync",
    Short: "Sync data from Tugboat Logic",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. Load configuration
        // 2. Initialize services
        // 3. Execute business logic
        // 4. Format and display output
        return nil
    },
}
```

### 2. Services Layer (`internal/services/`)

Services contain business logic and orchestrate workflows.

**Key Services:**
- **Config Service** - Configuration loading and validation
- **Evidence Service** - Evidence generation and analysis
- **Submission Service** - Evidence submission to Tugboat Logic
- **Validation Service** - Data validation

**Service Pattern:**
```go
type EvidenceService struct {
    storage     Storage
    tugboat     TugboatClient
    claude      ClaudeClient
    toolRegistry ToolRegistry
}

func (s *EvidenceService) GenerateEvidence(taskID string) error {
    // Business logic here
}
```

### 3. Domain Layer (`internal/domain/`, `internal/models/`)

Domain models represent core business entities.

**Key Entities:**
- `Policy` - Governance documents
- `Control` - Security controls
- `EvidenceTask` - Evidence collection tasks
- `Evidence` - Generated evidence

**Domain Model Example:**
```go
type EvidenceTask struct {
    ID              string
    ReferenceID     string
    Name            string
    Description     string
    Status          string
    AssignedTo      string
    Controls        []Control
    Policies        []Policy
    CollectionType  string
}
```

### 4. Storage Layer (`internal/storage/`)

Manages data persistence using JSON files with caching.

**Features:**
- JSON-based storage for offline access
- Performance caching (sub-100ms queries)
- Configurable storage paths
- Type-safe storage interfaces

**Storage Pattern:**
```go
type Storage interface {
    Save(path string, data interface{}) error
    Load(path string, dest interface{}) error
    List(pattern string) ([]string, error)
}
```

### 5. Tool Layer (`internal/tools/`)

Evidence collection tools implement a common interface.

**Tool Categories:**
- **Terraform Tools** (7 tools) - Infrastructure security analysis
- **GitHub Tools** (6 tools) - Repository access and security
- **Google Workspace Tools** - User and access management
- **Utility Tools** - Evidence assembly, validation

**Tool Interface:**
```go
type Tool interface {
    Name() string
    Description() string
    Execute(context.Context, ToolInput) (ToolOutput, error)
}
```

### 6. Adapters Layer (`internal/adapters/`, `internal/tugboat/`)

Adapters implement integrations with external systems.

**Key Adapters:**
- **Tugboat Client** - Tugboat Logic REST API
- **GitHub Client** - GitHub REST and GraphQL APIs
- **Claude Client** - Anthropic Claude API
- **Google Workspace Client** - Google Admin SDK

## Design Patterns

### Hexagonal Architecture (Ports & Adapters)

The application uses hexagonal architecture to separate:
- **Ports** - Interfaces defining contracts (`internal/interfaces/`)
- **Adapters** - Implementations for external systems (`internal/adapters/`)

Benefits:
- Easy to test (mock external dependencies)
- Easy to swap implementations
- Clear separation of concerns

### Registry Pattern

Tools and entities are registered in central registries:

```go
// Tool Registry
var toolRegistry = map[string]Tool{
    "terraform-analyzer": NewTerraformAnalyzer(),
    "github-permissions": NewGitHubPermissions(),
    // ... more tools
}

func GetTool(name string) (Tool, error) {
    tool, ok := toolRegistry[name]
    if !ok {
        return nil, fmt.Errorf("tool not found: %s", name)
    }
    return tool, nil
}
```

### Factory Pattern

Used for creating instances with dependencies:

```go
type ServiceFactory struct {
    config  *Config
    storage Storage
}

func (f *ServiceFactory) NewEvidenceService() *EvidenceService {
    return &EvidenceService{
        storage: f.storage,
        tugboat: f.NewTugboatClient(),
        claude:  f.NewClaudeClient(),
    }
}
```

### Repository Pattern

Storage layer implements repository pattern:

```go
type PolicyRepository interface {
    Save(policy *Policy) error
    FindByID(id string) (*Policy, error)
    FindAll() ([]*Policy, error)
}
```

### VCR Pattern (Testing)

Integration tests use VCR (Video Cassette Recorder) to record and replay HTTP interactions:

```go
func TestTugboatClient(t *testing.T) {
    // VCR intercepts HTTP calls
    client := setupTestClient(t)

    // First run: records to cassette
    // Subsequent runs: replays from cassette
    policies, err := client.GetPolicies(ctx)
    assert.NoError(t, err)
}
```

**Benefits:**
- No live API calls in CI/CD
- Fast test execution
- Reproducible tests
- Rate limit friendly

## Data Flow

### Synchronization Flow

```
User Command
    │
    ▼
CLI (sync command)
    │
    ▼
Sync Service
    │
    ├──► Tugboat Client ──► REST API
    │        │
    │        ▼
    │    API Response
    │        │
    │        ▼
    └──► Storage ──► JSON Files
              │
              ▼
        Local Cache
```

### Evidence Generation Flow

```
User Command (evidence generate ET-0001)
    │
    ▼
CLI (evidence command)
    │
    ▼
Evidence Service
    │
    ├──► Load Evidence Task ──► Storage
    │
    ├──► Identify Tools ──► Tool Registry
    │
    ├──► Execute Tools ──► Terraform/GitHub/etc.
    │        │
    │        ▼
    │    Tool Output
    │        │
    │        ▼
    ├──► Assemble Prompt ──► Prompt Templates
    │
    ├──► Generate Evidence ──► Claude API
    │        │
    │        ▼
    │    AI-Generated Evidence
    │        │
    │        ▼
    └──► Save Evidence ──► Storage
              │
              ▼
        Evidence File (JSON/Markdown)
```

### Authentication Flow

```
User Command (auth login)
    │
    ▼
CLI (auth command)
    │
    ▼
Auth Service
    │
    ▼
Browser Auth (macOS)
    │
    ├──► Open Safari
    │
    ├──► User Logs In
    │
    ├──► Extract Cookies (AppleScript)
    │        │
    │        ▼
    │    Session Cookies
    │        │
    │        ▼
    └──► Save Credentials ──► Storage (~/.grctool/auth/)
              │
              ▼
        Encrypted Credentials
```

## Extension Points

### Adding a New Tool

1. **Create tool implementation:**
   ```go
   // internal/tools/newtool/newtool.go
   type NewTool struct {
       config *Config
   }

   func (t *NewTool) Execute(ctx context.Context, input ToolInput) (ToolOutput, error) {
       // Implementation
   }
   ```

2. **Register in tool registry:**
   ```go
   // internal/tools/registry.go
   func init() {
       RegisterTool("newtool", NewNewTool())
   }
   ```

3. **Add CLI command:**
   ```go
   // cmd/tool_newtool.go
   var newToolCmd = &cobra.Command{
       Use:   "newtool",
       Short: "Execute new tool",
       RunE:  runNewTool,
   }
   ```

4. **Write tests:**
   - Unit tests for business logic
   - Integration tests with VCR
   - Functional tests for CLI

### Adding a New Formatter

1. **Implement Formatter interface:**
   ```go
   // internal/formatters/xml.go
   type XMLFormatter struct{}

   func (f *XMLFormatter) Format(data interface{}) (string, error) {
       // Implementation
   }
   ```

2. **Register formatter:**
   ```go
   func init() {
       RegisterFormatter("xml", &XMLFormatter{})
   }
   ```

### Adding a New Service

1. **Define service interface:**
   ```go
   // internal/services/newservice/interface.go
   type NewService interface {
       DoSomething(ctx context.Context) error
   }
   ```

2. **Implement service:**
   ```go
   // internal/services/newservice/service.go
   type newService struct {
       storage Storage
   }
   ```

3. **Wire dependencies:**
   ```go
   // cmd/root.go
   func initServices() {
       newSvc := newservice.New(storage)
       // Use service
   }
   ```

## Testing Strategy

### Three-Tier Testing

GRCTool uses a comprehensive three-tier testing strategy:

#### 1. Unit Tests
- **Location**: Alongside code (`*_test.go`)
- **Speed**: Very fast (2-3 seconds)
- **Scope**: Single function/method
- **Dependencies**: Mocked
- **Run**: `make test-unit`

**Example:**
```go
func TestParseConfig(t *testing.T) {
    cfg, err := ParseConfig("testdata/config.yaml")
    assert.NoError(t, err)
    assert.Equal(t, "https://api.tugboatlogic.com", cfg.BaseURL)
}
```

#### 2. Integration Tests
- **Location**: `test/integration/`
- **Tag**: `//go:build integration`
- **Speed**: Fast (VCR playback)
- **Scope**: Multiple components with external APIs
- **Dependencies**: VCR cassettes (no live APIs)
- **Run**: `make test-integration`

**Example:**
```go
//go:build integration

func TestTugboatClient_GetPolicies(t *testing.T) {
    client := setupTestClient(t)  // Uses VCR
    policies, err := client.GetPolicies(ctx)
    assert.NoError(t, err)
    assert.NotEmpty(t, policies)
}
```

#### 3. Functional Tests
- **Location**: `test/functional/`
- **Tag**: `//go:build functional`
- **Speed**: Moderate (CLI invocation)
- **Scope**: End-to-end CLI workflows
- **Dependencies**: Built binary
- **Run**: `make test-functional`

**Example:**
```go
//go:build functional

func TestCLI_Sync(t *testing.T) {
    output := runCLI(t, "sync", "--policies")
    assert.Contains(t, output, "Synced 15 policies")
}
```

### VCR Testing Infrastructure

VCR (Video Cassette Recorder) records and replays HTTP interactions:

**Modes:**
- `record` - Record requests to cassettes
- `playback` - Play back from cassettes (default)
- `record_once` - Record if cassette doesn't exist
- `off` - Disable VCR (live requests)

**Environment Variables:**
```bash
VCR_MODE=record make test-integration      # Record new cassettes
VCR_MODE=playback make test-integration    # Playback (default)
```

**Cassette Structure:**
```yaml
version: 1
interactions:
  - request:
      method: GET
      url: https://api.tugboatlogic.com/policies
    response:
      status_code: 200
      body: {...}
```

**Benefits:**
- No API rate limiting in tests
- Fast test execution
- Reproducible tests
- Works in CI/CD without credentials

### Coverage Goals

- **Core business logic**: 80%+
- **Utilities and helpers**: 60%+
- **Critical security functions**: 100%

Check coverage:
```bash
make test-coverage
```

## Security Considerations

### Authentication Security

- **Browser-based auth** uses macOS accessibility permissions
- Session cookies stored in `~/.grctool/auth/`
- Credentials not encrypted at rest (rely on OS permissions)
- File permissions set to `600` (user read/write only)

### API Security

- **HTTPS only** (enforced)
- **SSL certificate validation** (default)
- **Rate limiting** to prevent account lockout
- **Token rotation** recommended quarterly

### Data Security

- **Evidence files** may contain sensitive infrastructure info
- **`.gitignore`** excludes credentials and evidence
- **VCR cassettes** scrubbed of sensitive data
- **Logs** sanitized (no secrets logged)

### Code Security

- **gosec** security scanner (part of linting)
- **Dependency scanning** via Dependabot
- **Static analysis** via golangci-lint
- **License headers** enforced (Apache 2.0)

## Performance Optimizations

### Caching Strategy

- **JSON cache** for frequently accessed data
- **Tool indexer cache** for Terraform analysis (sub-100ms queries)
- **Template cache** for prompt assembly
- **TTL-based invalidation** (configurable)

### Concurrent Execution

- **Parallel tool execution** for independent tools
- **Worker pools** for batch operations
- **Context-based cancellation** for timeout handling

### Resource Management

- **Lazy loading** of large data structures
- **Streaming** for large file operations
- **Connection pooling** for HTTP clients
- **Graceful shutdown** with cleanup

## Configuration Management

Configuration uses YAML with environment variable overrides:

**Precedence (highest to lowest):**
1. Command-line flags
2. Environment variables
3. Configuration file (`.grctool.yaml`)
4. Default values

**Example:**
```yaml
tugboat:
  base_url: "${TUGBOAT_BASE_URL:-https://api.tugboatlogic.com}"
  org_id: "${TUGBOAT_ORG_ID}"

evidence:
  claude:
    api_key: "${CLAUDE_API_KEY}"
    model: "claude-3-5-sonnet-20241022"
```

## Error Handling

### Error Wrapping

Errors are wrapped with context:
```go
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}
```

### Error Types

- **Domain errors** - Custom error types for business logic
- **API errors** - HTTP status codes with details
- **Validation errors** - User input validation
- **System errors** - File system, network, etc.

### Logging Levels

- **DEBUG** - Detailed debugging information
- **INFO** - General informational messages
- **WARN** - Warning messages (non-critical)
- **ERROR** - Error messages (critical)

## Future Enhancements

### Planned Improvements

- **Plugin system** for third-party tools
- **REST API** for programmatic access
- **Web UI** for evidence review
- **Evidence templates** for common patterns
- **Multi-tenant support** for SaaS deployment
- **Webhook integration** for event-driven workflows
- **Evidence versioning** with diff support
- **Compliance dashboard** with metrics

### Architecture Evolution

As the project grows, consider:
- **Microservices** for scalability
- **Event sourcing** for audit trails
- **GraphQL API** for flexible queries
- **Database backend** (PostgreSQL) for larger datasets
- **Message queue** (RabbitMQ/Kafka) for async processing

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines on extending the architecture.

## References

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [VCR Testing](https://github.com/dnaeon/go-vcr)
