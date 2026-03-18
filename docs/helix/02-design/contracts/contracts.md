---
title: "Interface Contracts"
phase: "02-design"
category: "contracts"
tags: ["contracts", "interfaces", "api", "cli", "tools"]
related: ["system-design", "adr-index", "data-design"]
created: 2025-01-10
updated: 2025-01-10
helix_mapping: "Backfilled from codebase evidence"
---

# Interface Contracts

This document defines the external and internal interface contracts for GRCTool. These contracts specify inputs, outputs, error handling, and behavioral guarantees for each major system boundary.

## Contract Summary

| Contract ID | Name | Type | Status |
|-------------|------|------|--------|
| API-001 | CLI Command Interface | CLI | Approved |
| API-002 | Tool Interface | Library | Approved |
| API-003 | Tugboat API Interface | REST | Approved |
| API-004 | Storage Interface | Library | Approved |
| API-005 | Configuration Interface | Library | Approved |
| API-006 | Authentication Interface | Library | Approved |

---

## API-001: CLI Command Interface

**Type**: CLI
**Status**: Approved
**Version**: 1.0.0

### Command Structure

```bash
$ grctool [global-flags] <command> [subcommand] [command-flags] [arguments]
```

### Global Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--config` | `-c` | string | `.grctool.yaml` | Path to configuration file |
| `--verbose` | `-v` | bool | false | Enable verbose output |
| `--debug` | | bool | false | Enable debug logging |
| `--quiet` | `-q` | bool | false | Suppress non-essential output |
| `--format` | `-f` | string | `text` | Output format (text, json) |

### Commands

#### Command: `auth login`

**Purpose**: Authenticate with Tugboat Logic via browser-based flow
**Usage**: `$ grctool auth login`

**Input**: None (interactive browser flow)

**Output**:
- Success: Confirmation message to stdout
- Error: Error description to stderr

**Exit Codes**:
- `0`: Authentication successful
- `1`: Authentication failed (browser extraction error)
- `2`: Safari not available or accessibility permissions denied

**Example**:
```bash
$ grctool auth login
Opening Safari for Tugboat Logic authentication...
Authentication successful. Credentials saved.

$ grctool auth status
Provider: tugboat
Status: authenticated
Token: present (valid)
Source: browser
```

#### Command: `auth status`

**Purpose**: Display current authentication state for all providers
**Usage**: `$ grctool auth status [--provider NAME]`

**Output Schema** (JSON format):
```json
{
  "providers": [
    {
      "provider": "string",
      "authenticated": "boolean",
      "token_present": "boolean",
      "token_valid": "boolean",
      "last_validated": "string (ISO-8601, optional)",
      "expires_at": "string (ISO-8601, optional)",
      "source": "string (local|cache|api)"
    }
  ]
}
```

**Exit Codes**:
- `0`: At least one provider authenticated
- `1`: No providers authenticated

#### Command: `sync`

**Purpose**: Synchronize policies, controls, and evidence tasks from Tugboat Logic
**Usage**: `$ grctool sync [--policies] [--controls] [--evidence-tasks]`

**Options**:
- `--policies`: Sync only policies
- `--controls`: Sync only controls
- `--evidence-tasks`: Sync only evidence tasks
- (no flags): Sync all resource types

**Output**:
- Success: Summary of synced resources to stdout
- Progress: Status updates to stderr

**Exit Codes**:
- `0`: Sync completed successfully
- `1`: Partial sync failure (some resources failed)
- `2`: Authentication failure
- `3`: Network/API error

**Example**:
```bash
$ grctool sync
Syncing policies... 40 synced
Syncing controls... 85 synced
Syncing evidence tasks... 104 synced
Sync completed successfully.
```

#### Command: `tool <tool-name>`

**Purpose**: Execute an evidence collection tool
**Usage**: `$ grctool tool <tool-name> [tool-specific-flags]`

**Output**:
- Success: Tool output to stdout (format depends on tool)
- Error: Error details to stderr

**Exit Codes**:
- `0`: Tool executed successfully
- `1`: Tool execution error
- `2`: Invalid tool name or configuration
- `3`: Authentication required but not available

**Example**:
```bash
$ grctool tool evidence-task-list --format json
[
  {"id": 327992, "name": "GitHub Repository Access Controls", "reference_id": "ET-0047", "status": "pending"}
]

$ grctool tool github-permissions --repository org/repo --output-format matrix
```

#### Command: `evidence generate <task-ref>`

**Purpose**: Generate evidence for a specific evidence task
**Usage**: `$ grctool evidence generate <task-ref> [--format csv|markdown] [--tools TOOLS]`

**Options**:
- `--format`: Output format (csv, markdown). Default: csv
- `--tools`: Comma-separated list of tools to use
- `--include-reasoning`: Include AI reasoning in output

**Output**:
- Success: Generated evidence summary to stdout; files written to evidence directory
- Error: Error details to stderr

**Exit Codes**:
- `0`: Evidence generated successfully
- `1`: Generation failed
- `2`: Task not found
- `3`: Tool execution error

#### Command: `evidence submit <task-ref>`

**Purpose**: Submit evidence to Tugboat Logic
**Usage**: `$ grctool evidence submit <task-ref> [--window WINDOW] [--dry-run]`

**Options**:
- `--window`: Collection window (e.g., 2025-Q4). Default: current quarter
- `--dry-run`: Validate without submitting

**Exit Codes**:
- `0`: Submission accepted
- `1`: Submission rejected or failed
- `2`: Validation failed (use `--dry-run` to preview)
- `3`: Collector URL not configured

#### Command: `evidence setup <task-ref>`

**Purpose**: Configure collector URL for evidence submission
**Usage**: `$ grctool evidence setup <task-ref> --collector-url URL [--dry-run]`

**Exit Codes**:
- `0`: Setup successful
- `1`: Invalid URL or configuration error

---

## API-002: Tool Interface

**Type**: Library (Go Interface)
**Status**: Approved
**Version**: 1.0.0

### Interface Definition

Source: `internal/tools/interface.go`

```go
// Tool defines the interface for evidence collection tools
type Tool interface {
    // GetClaudeToolDefinition returns the tool definition for Claude
    GetClaudeToolDefinition() models.ClaudeTool

    // Execute runs the tool with the given parameters
    // Returns: result string, evidence source (if applicable), error
    Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error)

    // Name returns the tool name
    Name() string

    // Description returns the tool description
    Description() string
}
```

Parameters are passed as a generic `map[string]interface{}` rather than a typed struct. Each tool extracts its own expected keys from the map. The return tuple provides a result string (typically JSON), an optional `*models.EvidenceSource` for provenance tracking, and an error.

### ClaudeTool Definition

Source: `internal/models/evidence.go`

```go
// ClaudeTool represents a tool available to Claude
type ClaudeTool struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema map[string]interface{} `json:"input_schema"`
}
```

### EvidenceSource Contract

Source: `internal/models/evidence.go`

```go
// EvidenceSource tracks what sources were used to generate evidence
type EvidenceSource struct {
    Type        string                 `json:"type"`      // terraform, github, google_docs
    Resource    string                 `json:"resource"`  // file path, issue number, doc ID
    Content     string                 `json:"content"`   // extracted content
    Relevance   float64                `json:"relevance"` // 0.0-1.0 relevance score
    ExtractedAt time.Time              `json:"extracted_at"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"` // Additional source-specific data
}
```

### ToolOutput Envelope (JSON wire format)

Source: `internal/tools/output.go`

All tool outputs are wrapped in a standardized JSON envelope before being written to stdout:

```go
// ToolOutput represents the standardized JSON envelope for all tool outputs
type ToolOutput struct {
    OK    bool        `json:"ok"`
    Data  interface{} `json:"data,omitempty"`
    Error *ToolError  `json:"error,omitempty"`
    Meta  ToolMeta    `json:"meta"`
}

// ToolError provides detailed error information with correlation tracking
type ToolError struct {
    Code          string                 `json:"code"`
    Message       string                 `json:"message"`
    Details       map[string]interface{} `json:"details,omitempty"`
    CorrelationID string                 `json:"correlation_id"`
    Timestamp     time.Time              `json:"timestamp"`
}

// ToolMeta contains metadata about the operation
type ToolMeta struct {
    CorrelationID string      `json:"correlation_id"`
    Timestamp     time.Time   `json:"timestamp"`
    DurationMS    int64       `json:"duration_ms"`
    TaskRef       string      `json:"task_ref,omitempty"`
    Tool          string      `json:"tool,omitempty"`
    Version       string      `json:"version,omitempty"`
    AuthStatus    *AuthStatus `json:"auth_status,omitempty"`
    DataSource    string      `json:"data_source,omitempty"`
}
```

Common error codes: `VALIDATION_ERROR`, `EXECUTION_ERROR`, `PATH_SAFETY_ERROR`, `TASK_NOT_FOUND`, `UNAUTHORIZED`, `INTERNAL_ERROR`, `CONFIG_ERROR`, `SERVICE_ERROR`, `NETWORK_ERROR`.

### Registration Contract

Source: `internal/tools/registry.go`

Tools must be registered in the global `Registry` at initialization. The registry provides lookup, execution, and Claude tool definition export:

```go
// RegisterTool registers a tool in the global registry.
// The tool's Name() is used as the key. Returns an error on duplicate registration.
func RegisterTool(tool Tool) error

// GetTool retrieves a registered tool by name.
// Returns an error if the tool is not found.
func GetTool(name string) (Tool, error)

// ExecuteTool runs a tool from the global registry by name.
func ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (string, *models.EvidenceSource, error)

// GetAllClaudeToolDefinitions returns Claude tool definitions for all registered tools.
func GetAllClaudeToolDefinitions() []models.ClaudeTool
```

### Tool Categories

| Category | Tools | Auth Required |
|----------|-------|---------------|
| Evidence Analysis | evidence-task-list, evidence-task-details, evidence-relationships, prompt-assembler | No (local data) |
| Terraform | terraform-security-indexer, terraform-security-analyzer, terraform-hcl-parser, terraform-evidence-query, terraform_analyzer, terraform_snippets | No (local files) |
| GitHub | github-permissions, github-workflow-analyzer, github-review-analyzer, github-security-features, github-enhanced, github-searcher | Yes (GitHub token) |
| Google Workspace | google-workspace | Yes (Google credentials) |
| Evidence Management | evidence-generator, evidence-validator, evidence-writer, storage-read, storage-write | Varies |

### Error Contract

Tools must return descriptive errors that include:
- The tool name
- The operation that failed
- Actionable guidance for the user

```go
// Example error format
fmt.Errorf("github-permissions: failed to fetch repository %s: %w (ensure GITHUB_TOKEN is set)", repo, err)
```

---

## API-003: Tugboat API Interface

**Type**: REST Client
**Status**: Approved
**Version**: 1.0.0

### Base URL

```
https://openapi.tugboatlogic.com/api/v0
```

### Authentication

- **Method**: Cookie-based session authentication (extracted via browser)
- **Header**: `Cookie: <session-cookies>` or `Authorization: Bearer <token>`
- **Fallback**: HTTP Basic Auth for evidence submission endpoint (username/password)

### Endpoints

#### GET /policies

**Purpose**: List all organization policies
**Authentication**: Required (session cookie)

**Request**:
```http
GET /api/v0/policies?page=1&page_size=100 HTTP/1.1
Host: openapi.tugboatlogic.com
Cookie: <session-cookies>
```

**Response Success (200)**:
```json
{
  "results": [
    {
      "id": 123,
      "name": "Access Control Policy",
      "description": "...",
      "status": "active"
    }
  ],
  "count": 40,
  "next": "https://openapi.tugboatlogic.com/api/v0/policies?page=2",
  "previous": null
}
```

**Pagination**: Cursor-based via `next`/`previous` URLs. Client must follow `next` until null.

#### GET /controls

**Purpose**: List all organization controls with framework mappings
**Authentication**: Required (session cookie)

**Response Success (200)**: Same pagination pattern as policies. Each control includes `framework_codes` array with SOC2/ISO27001 code mappings.

#### GET /evidence-tasks

**Purpose**: List all evidence collection tasks
**Authentication**: Required (session cookie)

**Response Success (200)**: Same pagination pattern. Each task includes `master_evidence_id`, `collection_interval`, and `completed` status.

#### POST /evidence/collector/{collector_id}/

**Purpose**: Submit evidence for a specific task via custom collector
**Authentication**: Required (HTTP Basic Auth)
**Content-Type**: multipart/form-data

**Request**:
```http
POST /api/v0/evidence/collector/805/ HTTP/1.1
Host: openapi.tugboatlogic.com
Authorization: Basic <base64(username:password)>
Content-Type: multipart/form-data

--boundary
Content-Disposition: form-data; name="file"; filename="evidence.md"
Content-Type: text/markdown

<evidence content>
--boundary--
```

**Response Success (200)**:
```json
{
  "submission_id": "sub_abc123",
  "status": "accepted",
  "received_at": "2025-01-10T12:00:00Z"
}
```

### Error Response Format

```json
{
  "detail": "Authentication credentials were not provided.",
  "status_code": 401
}
```

### Rate Limiting

| Endpoint | Rate Limit | Strategy |
|----------|-----------|----------|
| GET endpoints | 10 requests/second | Configurable via `tugboat.rate_limit` |
| POST evidence | 5 requests/second | Sequential submission recommended |

### Client Contract

Source: `internal/tugboat/client.go`, `internal/tugboat/policies.go`, `internal/tugboat/controls.go`, `internal/tugboat/evidence.go`, `internal/tugboat/evidence_submission.go`, `internal/tugboat/submissions.go`

The Tugboat client is a concrete `*Client` struct (no formal Go interface). It is created via `NewClient(cfg, vcrConfig)` and exposes the following public methods:

```go
// Construction and lifecycle
func NewClient(cfg *config.TugboatConfig, vcrConfig *vcr.Config) *Client
func (c *Client) Close()
func (c *Client) TestConnection(ctx context.Context) error

// Policies
func (c *Client) GetPolicies(ctx context.Context, opts *PolicyListOptions) ([]models.Policy, error)
func (c *Client) GetAllPolicies(ctx context.Context, org string, framework string) ([]models.Policy, error)
func (c *Client) GetPolicyDetails(ctx context.Context, policyID string) (*models.PolicyDetails, error)

// Controls
func (c *Client) GetControls(ctx context.Context, opts *ControlListOptions) ([]models.Control, error)
func (c *Client) GetAllControls(ctx context.Context, org string, framework string) ([]models.Control, error)
func (c *Client) GetControlDetails(ctx context.Context, controlID string) (*models.ControlDetails, error)
func (c *Client) GetControlDetailsWithEvidenceEmbeds(ctx context.Context, controlID string) (*models.ControlDetails, error)

// Evidence tasks
func (c *Client) GetEvidenceTasks(ctx context.Context, opts *EvidenceTaskListOptions) ([]models.EvidenceTask, error)
func (c *Client) GetAllEvidenceTasks(ctx context.Context, org string, framework string) ([]models.EvidenceTask, error)
func (c *Client) GetEvidenceTaskDetails(ctx context.Context, taskID string) (*models.EvidenceTaskDetails, error)
func (c *Client) GetEvidenceTasksByControl(ctx context.Context, controlID string, org string) ([]models.EvidenceTask, error)

// Evidence submission (Custom Evidence Integration API -- HTTP Basic Auth + X-API-KEY)
func (c *Client) SubmitEvidence(ctx context.Context, req *SubmitEvidenceRequest) (*SubmitEvidenceResponse, error)

// Evidence attachments (browsing existing submissions)
func (c *Client) GetEvidenceAttachments(ctx context.Context, opts *models.EvidenceAttachmentListOptions) ([]models.EvidenceAttachment, error)
func (c *Client) GetAllEvidenceAttachments(ctx context.Context, taskID int, observationPeriod string) ([]models.EvidenceAttachment, error)
func (c *Client) GetEvidenceAttachmentsByTask(ctx context.Context, taskID int) ([]models.EvidenceAttachment, error)
func (c *Client) GetEvidenceAttachmentsByTaskAndWindow(ctx context.Context, taskID int, startDate, endDate string) ([]models.EvidenceAttachment, error)
func (c *Client) GetAttachmentDownloadURL(ctx context.Context, attachmentID int) (*models.AttachmentDownloadResponse, error)
func (c *Client) DownloadAttachment(ctx context.Context, attachmentID int, destPath string) error
```

The `SubmitEvidenceRequest` and `SubmitEvidenceResponse` types:

```go
type SubmitEvidenceRequest struct {
    CollectorURL  string    // Full collector URL
    FilePath      string    // Path to the evidence file to upload
    CollectedDate time.Time // Date the evidence was collected
    ContentType   string    // MIME type of the file
}

type SubmitEvidenceResponse struct {
    Success    bool      `json:"success"`
    Message    string    `json:"message,omitempty"`
    ReceivedAt time.Time `json:"received_at"`
}
```

---

## API-004: Storage Interface

**Type**: Library (Go Interface)
**Status**: Approved
**Version**: 1.0.0

### Interface Definition

Source: `internal/interfaces/storage.go`

```go
// StorageService provides storage operations for the application data
type StorageService interface {
    // Policy operations
    SavePolicy(policy *domain.Policy) error
    GetPolicy(id string) (*domain.Policy, error)
    GetPolicyByReferenceAndID(referenceID, numericID string) (*domain.Policy, error)
    GetAllPolicies() ([]domain.Policy, error)
    GetPolicySummary() (*domain.PolicySummary, error)

    // Control operations
    SaveControl(control *domain.Control) error
    GetControl(id string) (*domain.Control, error)
    GetControlByReferenceAndID(referenceID, numericID string) (*domain.Control, error)
    GetAllControls() ([]domain.Control, error)
    GetControlSummary() (*domain.ControlSummary, error)

    // Evidence task operations
    SaveEvidenceTask(task *domain.EvidenceTask) error
    GetEvidenceTask(id string) (*domain.EvidenceTask, error)
    GetEvidenceTaskByReferenceAndID(referenceID, numericID string) (*domain.EvidenceTask, error)
    GetAllEvidenceTasks() ([]domain.EvidenceTask, error)
    GetEvidenceTaskSummary() (*domain.EvidenceTaskSummary, error)

    // Evidence record operations
    SaveEvidenceRecord(record *domain.EvidenceRecord) error
    GetEvidenceRecord(id string) (*domain.EvidenceRecord, error)
    GetEvidenceRecordsByTaskID(taskID int) ([]domain.EvidenceRecord, error)

    // Statistics and metadata
    GetStats() (map[string]interface{}, error)
    SetSyncTime(syncType string, syncTime time.Time) error
    GetSyncTime(syncType string) (time.Time, error)

    // Utility operations
    Clear() error
}
```

Related interfaces in the same file:

```go
// CacheService provides caching capabilities for performance optimization
type CacheService interface {
    Set(key string, value interface{}, expiration time.Duration) error
    Get(key string, target interface{}) error
    Delete(key string) error
    Clear() error
    Exists(key string) bool
    SetToolResult(toolName string, params map[string]interface{}, result interface{}, expiration time.Duration) error
    GetToolResult(toolName string, params map[string]interface{}, target interface{}) error
    SetSummary(summaryType, id string, summary interface{}, expiration time.Duration) error
    GetSummary(summaryType, id string, target interface{}) error
    GetSize() int64
    GetStats() map[string]interface{}
}

// FileService provides low-level file operations
type FileService interface {
    Save(category, id string, data interface{}) error
    Load(category, id string, target interface{}) error
    Exists(category, id string) bool
    Delete(category, id string) error
    List(category string) ([]string, error)
    CreateDirectory(path string) error
    DeleteDirectory(path string) error
    ListDirectories(basePath string) ([]string, error)
    GetFullPath(category, id string) string
    GetSize() (int64, error)
}

// LocalDataStore provides offline-first data access without external dependencies
type LocalDataStore interface {
    StorageService
    IsDataAvailable() bool
    GetDataSources() []string
    ValidateDataIntegrity() error
    SetFallbackEnabled(enabled bool)
    IsFallbackEnabled() bool
    ImportData(sourcePath string) error
    ExportData(targetPath string) error
    GetLastDataUpdate() (time.Time, error)
}
```

### File Naming Conventions

| Entity | Pattern | Example |
|--------|---------|---------|
| Evidence Task (synced) | `ET-{NNN}.json` | `ET-001.json` |
| Policy (synced) | `POL-{NNN}.json` | `POL-001.json` |
| Control (synced) | `{code}.json` | `AC-01.json`, `CC1_1.json` |
| Generated Evidence | `{NN}_{source}_{description}.md` | `01_terraform_iam_roles.md` |
| Submission State | `submission.yaml` | `submission.yaml` |
| Generation Metadata | `metadata.yaml` | `metadata.yaml` |

### Directory Layout

```
{data_dir}/
  docs/
    policies/
      json/                          # Policy JSON from Tugboat
      markdown/                      # Policy markdown renderings
    controls/
      json/                          # Control JSON from Tugboat
      markdown/                      # Control markdown renderings
    evidence_tasks/
      json/                          # Evidence task JSON from Tugboat
      markdown/                      # Evidence task markdown renderings
    evidence_prompts/                # Generated prompts for AI
  evidence/
    ET-0001/                         # Evidence organized by task ref
      2025-Q4/                       # By collection window
        01_terraform_iam_roles.md
        02_github_permissions.md
        .generation/
          metadata.yaml              # Generation metadata
        .submission/
          submission.yaml            # Submission state
  .cache/
    prompts/                         # Cached prompt assemblies
    summaries/                       # Cached AI summaries
    tool_outputs/                    # Cached tool execution results
    relationships/                   # Cached relationship mappings
    validations/                     # Cached validation results
  .state/
    evidence_state.yaml              # Aggregated evidence state cache
```

### Path Resolution

All paths in storage are resolved relative to `storage.data_dir` from configuration. The `StoragePaths` structure provides configurable subdirectory overrides with sensible defaults (see `WithDefaults()` method).

### File Permissions

| File Type | Permission | Rationale |
|-----------|-----------|-----------|
| Configuration (.grctool.yaml) | 0644 | Readable by processes, writable by owner |
| Auth credentials | 0600 | Owner read/write only |
| Evidence files | 0644 | Readable for audit review |
| Cache files | 0644 | Deletable without consequence |

---

## API-005: Configuration Interface

**Type**: Library
**Status**: Approved
**Version**: 1.0.0

### Configuration File Schema

The primary configuration file is `.grctool.yaml` located in the project root.

```yaml
# Top-level configuration structure
tugboat:                    # Tugboat Logic API settings
  base_url: string          # Required. API base URL
  org_id: string            # Organization ID
  timeout: duration         # Default: 30s
  rate_limit: int           # Default: 10 requests/sec
  auth_mode: string         # "form" or "browser"
  cookie_header: string     # Browser auth cookie (env var substitution supported)
  bearer_token: string      # Direct bearer token (env var substitution supported)
  log_api_requests: bool    # Log HTTP request details
  log_api_responses: bool   # Log HTTP response details
  username: string          # HTTP Basic Auth for evidence submission
  password: string          # HTTP Basic Auth (env var substitution supported)
  collector_urls:            # Evidence task ref -> collector URL mapping
    ET-0001: string         # e.g., "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/"

evidence:
  generation:
    output_dir: string      # Default: "evidence/generated"
    prompt_dir: string      # Default: "evidence/prompts"
    include_reasoning: bool # Include AI reasoning in output
    max_tool_calls: int     # Default: 50
    default_format: string  # "csv" or "markdown". Default: "csv"
  tools:
    terraform:
      enabled: bool
      scan_paths: [string]
      include_patterns: [string]  # Default: ["*.tf", "*.tfvars"]
      exclude_patterns: [string]  # Default: ["*.secret", ".terraform/**"]
    github:
      enabled: bool
      api_token: string     # Env var substitution supported; falls back to gh CLI
      repository: string    # owner/repo format
      max_issues: int       # Default: 100
      rate_limit: int       # Default: 30 (GitHub Search API limit)
    google_docs:
      enabled: bool
      credentials_file: string  # Path to Google service account JSON
      shared_drive_id: string
  quality:
    min_sources: int        # Default: 2
    require_reasoning: bool
    min_completeness_score: float  # 0.0-1.0. Default: 0.7
    min_quality_score: float       # 0.0-1.0. Default: 0.8
  terraform:
    atmos_path: string      # Path to Atmos stack configs
    repo_path: string       # Path to terraform repo for git hash

storage:
  data_dir: string          # Default: "./data"
  local_data_dir: string    # Default: "./local_data"
  cache_dir: string         # Default: "./.cache"
  paths:                    # Customizable subdirectory paths (all optional)
    docs: string
    evidence: string
    cache: string
    prompts: string
    policies_json: string
    policies_markdown: string
    controls_json: string
    controls_markdown: string
    evidence_tasks_json: string
    evidence_tasks_markdown: string
    evidence_prompts: string

logging:
  loggers:
    console:
      enabled: bool         # Default: true
      level: string         # Default: "warn"
      format: string        # "text" or "json". Default: "text"
      output: string        # "stdout", "stderr", "file". Default: "stderr"
      sanitize_urls: bool   # Default: true
      redact_fields: [string]  # Default: [password, token, key, secret, api_key, cookie]
    file:
      enabled: bool         # Default: true
      level: string         # Default: "info"
      format: string        # Default: "text"
      output: string        # Default: "file"
      file_path: string     # Default: platform-specific log path

interpolation:
  enabled: bool             # Default: true (always enabled)
  variables:                # Nested variable map
    organization:
      name: string          # Default: "Seventh Sense"

auth:
  cache_dir: string         # Default: "{cache_dir}/auth"
  github:
    token: string           # Env var substitution supported
  tugboat:
    bearer_token: string    # Env var substitution supported
```

### Configuration Precedence

Configuration values are resolved in this order (highest priority first):

1. **Command-line flags** (e.g., `--format csv`)
2. **Environment variables** (e.g., `GITHUB_TOKEN`)
3. **Configuration file** (`.grctool.yaml`)
4. **Default values** (hardcoded in `Validate()`)

### Environment Variable Substitution

Configuration values starting with `${` and ending with `}` are treated as environment variable references:

```yaml
github:
  api_token: "${GITHUB_TOKEN}"      # Resolved from GITHUB_TOKEN env var
tugboat:
  password: "${TUGBOAT_API_KEY}"    # Resolved from TUGBOAT_API_KEY env var
```

If the environment variable is not set, the value is cleared (empty string) for optional fields.

### Loading Contract

```go
// Load reads configuration from Viper, processes environment variables,
// resolves relative paths, and validates all fields.
// Returns an error if required fields are missing or validation fails.
func Load() (*Config, error)

// LoadWithoutValidation reads configuration without validation.
// Useful for template rendering or partial configuration inspection.
func LoadWithoutValidation() (*Config, error)

// Validate checks all configuration fields for correctness.
// Sets default values for optional fields that are empty.
// Returns an error describing the first validation failure.
func (c *Config) Validate() error
```

### Validation Rules

| Field | Rule | Default |
|-------|------|---------|
| `tugboat.base_url` | Required, non-empty | None (must be set) |
| `tugboat.auth_mode` | Must be "form" or "browser" if set | Empty (optional) |
| `tugboat.timeout` | Must be > 0 | 30s |
| `tugboat.rate_limit` | Must be > 0 | 10 |
| `evidence.generation.default_format` | Must be "csv" or "markdown" | "csv" |
| `evidence.generation.max_tool_calls` | Must be > 0 | 50 |
| `evidence.quality.min_completeness_score` | Must be 0.0-1.0 | 0.7 |
| `evidence.quality.min_quality_score` | Must be 0.0-1.0 | 0.8 |
| `evidence.terraform.atmos_path` | Must exist on filesystem if set | Empty |
| `evidence.tools.google_docs.credentials_file` | Must exist if Google Docs enabled | Empty |

---

## API-006: Authentication Interface

**Type**: Library (Go Interface)
**Status**: Approved
**Version**: 1.0.0

### Interface Definition

```go
// AuthProvider defines the contract for tool-specific authentication providers.
// Each external integration (GitHub, Tugboat, Google Workspace) has its own provider.
type AuthProvider interface {
    // Name returns the provider identifier (e.g., "github", "tugboat").
    Name() string

    // IsAuthRequired returns true if this provider requires authentication.
    // Returns false for tools that work with local data (e.g., Terraform file scanning).
    IsAuthRequired() bool

    // GetStatus returns the current authentication state without side effects.
    // Must not trigger re-authentication.
    GetStatus(ctx context.Context) *AuthStatus

    // Authenticate performs the authentication process.
    // For browser-based auth, this opens Safari and extracts cookies.
    // For token-based auth, this validates the configured token.
    Authenticate(ctx context.Context) error

    // ValidateAuth checks whether current credentials are still valid.
    // May make an API call to verify token validity.
    ValidateAuth(ctx context.Context) error

    // ClearAuth removes cached credentials.
    ClearAuth() error
}
```

### AuthStatus Contract

```go
type AuthStatus struct {
    Authenticated bool       // Whether the provider is successfully authenticated
    Provider      string     // Provider identifier
    TokenPresent  bool       // Whether a credential is configured (may be invalid)
    TokenValid    bool       // Whether the credential has been validated recently
    LastValidated *time.Time // When validation last occurred
    ExpiresAt     *time.Time // When the credential expires (if known)
    CacheUsed     bool       // Whether cached credentials were used
    Error         string     // Error message if authentication failed
    Source        string     // Data source: "local", "cache", "api", "browser"
}
```

### Provider Implementations

| Provider | Auth Method | Credential Storage | Token Refresh |
|----------|-----------|-------------------|---------------|
| Tugboat | Browser cookie extraction (Safari) | Config file (cookie_header) | Manual re-login |
| GitHub | Personal access token or `gh` CLI | Config file or `gh auth token` | Token rotation |
| Google Workspace | Service account JSON | Credentials file on disk | OAuth2 auto-refresh |
| NoAuth (local tools) | None | None | N/A |

### NoAuthProvider Contract

For tools that operate on local data (Terraform file scanning, local document analysis), the `NoAuthProvider` satisfies the `AuthProvider` interface with no-op implementations:

```go
// NewNoAuthProvider creates a provider for offline/local tools.
// Always returns Authenticated: true with Source: "local".
func NewNoAuthProvider(name string, source string) AuthProvider
```

### Error Handling

Authentication errors must be categorized:

| Error | Condition | User Action |
|-------|-----------|-------------|
| `ErrBrowserAuthFailed` | Safari not accessible or cookies not found | Run `grctool auth login` with Safari open to Tugboat |
| `ErrTokenNotFound` | No token configured and `gh` CLI not available | Set `GITHUB_TOKEN` or run `gh auth login` |
| `ErrTokenExpired` | Session cookie or token has expired | Re-authenticate with `grctool auth login` |
| `ErrAuthenticationFailed` | Token is present but API rejects it | Verify credentials; may need token rotation |

---

## Cross-Cutting Contracts

### Error Response Format

All interfaces return errors following Go conventions with context wrapping:

```go
// Error wrapping pattern used throughout GRCTool
fmt.Errorf("component: operation failed: %w", underlyingError)

// Example
fmt.Errorf("tugboat: failed to sync policies (page %d): %w", page, err)
```

### Context and Cancellation

All interfaces that perform I/O accept `context.Context` as the first parameter and must respect cancellation:

```go
// All long-running operations must check context
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // proceed with operation
}
```

### Backwards Compatibility

- All changes to CLI flags must be additive (no removal of existing flags)
- JSON output schemas are append-only (new fields may be added, existing fields never removed)
- Tool interface methods must not change signature; new capabilities are added via new interface methods
- Configuration file changes are backward-compatible; new keys have defaults

---

## References

- [System Architecture](/home/erik/Projects/grctool/docs/helix/02-design/architecture/system-design.md)
- [ADR Index](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md)
- [Data Design](/home/erik/Projects/grctool/docs/helix/02-design/data-design/data-design.md)
- [Tool Interface Source](/home/erik/Projects/grctool/internal/tools/interface.go)
- [Tool Output Source](/home/erik/Projects/grctool/internal/tools/output.go)
- [Tool Registry Source](/home/erik/Projects/grctool/internal/tools/registry.go)
- [Storage Interface Source](/home/erik/Projects/grctool/internal/interfaces/storage.go)
- [Auth Provider Source](/home/erik/Projects/grctool/internal/auth/provider.go)
- [Tugboat Client Source](/home/erik/Projects/grctool/internal/tugboat/client.go)
- [Config Source](/home/erik/Projects/grctool/internal/config/config.go)
