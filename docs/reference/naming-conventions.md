---
title: "Naming Conventions and Standards"
type: "reference"
category: "naming-conventions"
tags: ["naming", "standards", "reference-ids", "file-naming", "conventions"]
related: ["[[data-formats]]", "[[cli-commands]]", "[[api-documentation]]"]
created: 2025-09-10
modified: 2025-09-10
status: "active"
---

# Naming Conventions and Standards

## Overview

This document establishes the standardized naming conventions for reference IDs, file names, directories, and other identifiers used throughout GRCTool. Consistent naming ensures predictable automation, reliable file operations, and clear communication across all system components.

## Reference ID Standards

### Evidence Task References

**Format**: `ET-NNNN` (4-digit zero-padded)

**Examples**:
- `ET-0001` - First evidence task
- `ET-0101` - 101st evidence task  
- `ET-0999` - 999th evidence task

**Input Normalization**:
```bash
# All these inputs normalize to ET-0001
ET1 → ET-0001
ET-1 → ET-0001
ET 1 → ET-0001
et-0001 → ET-0001 (case insensitive)
328001 → ET-0001 (via Tugboat ID lookup)
```

**Validation Rules**:
- Must start with `ET-` prefix
- Must have exactly 4 digits after prefix
- Leading zeros required for consistency
- Maximum supported: ET-9999 (expandable if needed)

**Usage Examples**:
```bash
# CLI usage accepts flexible input
grctool tool evidence-task-details --task-ref ET1
grctool tool evidence-task-details --task-ref ET-0001
grctool tool evidence-task-details --task-ref 328001

# All normalize to ET-0001 internally
```

### Policy References

**Format**: `POL-NNNN` (4-digit zero-padded)

**Examples**:
- `POL-0001` - Information Security Policy
- `POL-0002` - Access Control Policy
- `POL-0015` - Data Classification Policy

**Input Normalization**:
```bash
POL1 → POL-0001
POL-1 → POL-0001
pol-0001 → POL-0001
94641 → POL-0001 (via Tugboat ID lookup)
```

**Validation Rules**:
- Must start with `POL-` prefix
- Must have exactly 4 digits after prefix
- Sequential numbering preferred
- Range: POL-0001 to POL-9999

### Control References

Control references vary by compliance framework due to established industry standards.

#### ISO 27001 Controls

**Format**: `XX-NN` (2-character category, 2-digit number)

**Examples**:
- `AC-01` - Access Control Policy
- `AC-02` - Account Management
- `SC-13` - Cryptographic Protection
- `AU-02` - Audit Events

**Categories**:
- `AC` - Access Control
- `AU` - Audit and Accountability
- `CA` - Security Assessment and Authorization
- `CM` - Configuration Management
- `CP` - Contingency Planning
- `IA` - Identification and Authentication
- `IR` - Incident Response
- `MA` - Maintenance
- `MP` - Media Protection
- `PE` - Physical and Environmental Protection
- `PL` - Planning
- `PS` - Personnel Security
- `RA` - Risk Assessment
- `SA` - System and Services Acquisition
- `SC` - System and Communications Protection
- `SI` - System and Information Integrity

#### SOC 2 Controls

**Format**: `CC-NN.N` (Common Criteria format)

**Examples**:
- `CC-1.1` - Control Environment - Demonstrates Commitment
- `CC-6.1` - Logical and Physical Access Controls
- `CC-8.1` - Change Management Controls

**File Name Format**: `CC-NN_N` (underscores instead of periods for filesystem compatibility)
- `CC-01_1-778805-control_environment.json`

**JSON Format**: `CC-NN.N` (periods preserved in data)
```json
{
  "reference_id": "CC-1.1",
  "file_reference_id": "CC-01_1"
}
```

**Categories**:
- `CC-1.x` - Control Environment
- `CC-2.x` - Communication and Information
- `CC-3.x` - Risk Assessment
- `CC-4.x` - Monitoring Activities  
- `CC-5.x` - Control Activities
- `CC-6.x` - Logical and Physical Access Controls
- `CC-7.x` - System Operations
- `CC-8.x` - Change Management
- `CC-9.x` - Risk Mitigation

### Organization-Specific References

**Format**: Flexible based on organizational needs

**Examples**:
- `CUST-0001` - Custom Control 1
- `PROC-0001` - Custom Procedure 1
- `STD-0001` - Technical Standard 1

## File Naming Conventions

### Core Data Files

**Pattern**: `<type>-<reference_id>-<tugboat_id>-<sanitized_name>.json`

**Components**:
- `type`: Document type (ET, POL, AC, CC, etc.)
- `reference_id`: Normalized reference ID 
- `tugboat_id`: Tugboat Logic numeric ID
- `sanitized_name`: Filesystem-safe descriptive name

**Examples**:
```
ET-0001-328001-access_control_documentation.json
POL-0001-94641-information_security_policy.json
AC-01-778771-access_control_policy.json
CC-01_1-778805-control_environment_design.json
```

### Evidence Files

**Pattern**: `evidence-<task_ref>-<timestamp>-v<version>.json`

**Components**:
- `evidence`: Fixed prefix
- `task_ref`: Evidence task reference (ET-NNNN)
- `timestamp`: ISO 8601 compact format (YYYYMMDDTHHMMSSZ)
- `version`: Version number (v1, v2, etc.)

**Examples**:
```
evidence-ET-0001-20250910T143000Z-v1.json
evidence-ET-0045-20250910T150000Z-v2.json
evidence-ET-0078-20250910T160000Z-v1.json
```

### Tool Output Files

**Pattern**: `<tool_name>-<task_ref>-<timestamp>.json`

**Examples**:
```
terraform-scanner-ET-0011-20250910T143000Z.json
github-permissions-ET-0001-20250910T143000Z.json
google-workspace-ET-0090-20250910T143000Z.json
```

### Backup and Archive Files

**Pattern**: `<original_name>.backup.<timestamp>`

**Examples**:
```
ET-0001-328001-access_control_documentation.json.backup.20250910T143000Z
evidence-ET-0001-20250910T143000Z-v1.json.backup.20250910T150000Z
```

## Directory Naming Standards

### Primary Directory Structure

```
grctool/                          # Project root (kebab-case)
├── cmd/                         # CLI commands (lowercase)
├── internal/                    # Internal packages (lowercase)
│   ├── auth/                   # Authentication (lowercase)
│   ├── config/                 # Configuration (lowercase) 
│   ├── services/               # Business services (lowercase)
│   ├── tools/                  # Evidence collection tools (lowercase)
│   └── tugboat/                # Tugboat client (lowercase)
└── docs/                       # Documentation (lowercase)

../docs/                         # Document storage (separate repo)
├── evidence_tasks/             # Evidence task files (snake_case)
├── policies/                   # Policy documents (lowercase)
├── controls/                   # Control definitions (lowercase)
└── evidence_prompts/           # Generated prompts (snake_case)
```

### Evidence Storage Structure

**Current Format (as of 2025-10-28)**:
```
evidence/                                           # Evidence root
├── TaskName_ET-0001_328001/                       # {Name}_{ET-Ref}_{TugboatID}
│   └── 2025-Q4/                                   # Collection window
│       ├── 01_evidence.md                         # Working evidence files
│       ├── .generation/                           # Generation metadata
│       ├── .submitted/                            # Submitted files
│       └── archive/                               # Synced from Tugboat
├── Another_Task_ET-0002_328002/
└── summaries/                                     # Aggregated summaries (lowercase)
    ├── daily/                                     # Daily summaries (lowercase)
    ├── weekly/                                    # Weekly summaries (lowercase)
    └── audit-ready/                               # Audit packages (kebab-case)
```

**Directory Naming**: Evidence task directories are named with the task name first (human-readable), followed by the ET reference (for quick lookup), and the Tugboat ID (canonical identifier).

### Tool-Specific Directories

```
internal/tools/                  # Tool implementations
├── terraform/                  # Tool category (lowercase)
│   ├── scanner.go             # Implementation files (lowercase)
│   ├── hcl_parser.go          # Snake case for multi-word (Go convention)
│   └── types.go               # Supporting files (lowercase)
├── github/                     # Tool category (lowercase)
│   ├── permissions.go         # Implementation files (lowercase)
│   ├── security_features.go   # Snake case for multi-word
│   └── workflow_analyzer.go   # Snake case for multi-word
└── google_workspace/           # Tool category (snake_case for multi-word)
    ├── client.go              # Implementation files (lowercase)
    └── document_analyzer.go   # Snake case for multi-word
```

## Name Sanitization Rules

### Filesystem Safety

**Prohibited Characters** (replaced with underscores):
- Forward slash: `/` → `_`
- Backslash: `\` → `_`
- Colon: `:` → `_`
- Asterisk: `*` → `_`
- Question mark: `?` → `_`
- Quote marks: `"` `'` → `_`
- Less/greater than: `<` `>` → `_`
- Pipe: `|` → `_`
- Space: ` ` → `_`

**Sanitization Examples**:
```
"Access Control: User Management" → "access_control_user_management"
"Network Security (DMZ)" → "network_security_dmz"
"Policy Review - Q4 2024" → "policy_review_q4_2024"
```

**Sanitization Function**:
```go
func SanitizeFileName(name string) string {
    // Convert to lowercase
    name = strings.ToLower(name)
    
    // Replace prohibited characters with underscores
    prohibited := []string{"/", "\\", ":", "*", "?", "\"", "'", "<", ">", "|", " "}
    for _, char := range prohibited {
        name = strings.ReplaceAll(name, char, "_")
    }
    
    // Remove multiple consecutive underscores
    re := regexp.MustCompile(`_+`)
    name = re.ReplaceAllString(name, "_")
    
    // Trim leading/trailing underscores
    name = strings.Trim(name, "_")
    
    // Limit length
    if len(name) > 100 {
        name = name[:100]
    }
    
    return name
}
```

### Case Conventions

**Reference IDs**: UPPERCASE with hyphens
- `ET-0001`, `POL-0001`, `AC-01`

**File Names**: lowercase with underscores  
- `access_control_documentation.json`
- `information_security_policy.json`

**Directory Names**: lowercase, prefer single words
- `evidence_tasks/`, `policies/`, `controls/`

**Go Package Names**: lowercase, single word preferred
- `auth`, `config`, `tugboat` (not `tugboat_logic`)

**Go File Names**: lowercase with underscores for multi-word
- `evidence_task.go`, `policy_service.go`

## Tool Naming Standards

### Tool Names

**Format**: kebab-case for CLI, snake_case for internal Go packages

**CLI Tool Names**:
```bash
terraform-scanner              # Infrastructure analysis
github-permissions            # Access control analysis
google-workspace             # Document management
evidence-task-details        # Evidence task information
```

**Go Package Names**:
```go
internal/tools/terraform_scanner
internal/tools/github_permissions
internal/tools/google_workspace
internal/tools/evidence_task_details
```

**Tool Registry Names** (internal):
```go
const (
    ToolTerraformScanner      = "terraform-scanner"
    ToolGitHubPermissions     = "github-permissions"
    ToolGoogleWorkspace       = "google-workspace"
    ToolEvidenceTaskDetails   = "evidence-task-details"
)
```

### Tool Categories

**Infrastructure Analysis**:
- `terraform-scanner`
- `terraform-hcl-parser`

**Access Control**:
- `github-permissions`
- `github-deployment-access`
- `identity-evidence-collector` (planned)

**Security Analysis**:
- `github-security-features`
- `github-workflow-analyzer`
- `security-ops-evidence-collector` (planned)

**Document Management**:
- `google-workspace`
- `evidence-generator`
- `policy-summary-generator`

**Utility Tools**:
- `storage-read`
- `storage-write`
- `name-generator`

## Configuration Naming

### YAML Configuration Keys

**Format**: snake_case for consistency with YAML conventions

```yaml
# Correct naming
storage:
  data_dir: "./docs"
  cache_dir: "./cache"
  local_data_dir: "./local"

logging:
  log_level: "info"
  file_logging: true
  structured_format: true

tools:
  terraform_scanner:
    max_files: 1000
    timeout_seconds: 300
  
  github_permissions:
    include_archived: false
    rate_limit_requests: 5000
```

### Environment Variables

**Format**: UPPERCASE with underscores, prefixed with `GRCTOOL_`

```bash
# Authentication
GRCTOOL_TUGBOAT_ORG_ID=12345
GRCTOOL_TUGBOAT_COOKIE="session_cookie_value"

# Storage
GRCTOOL_DATA_DIR="/custom/data/path"
GRCTOOL_CACHE_DIR="/custom/cache/path"

# Logging
GRCTOOL_LOG_LEVEL=debug
GRCTOOL_LOG_FILE="/var/log/grctool.log"

# Tool-specific
GRCTOOL_TERRAFORM_TIMEOUT=600
GRCTOOL_GITHUB_TOKEN="github_pat_..."
```

## JSON Field Naming

### API Responses

**Format**: snake_case for consistency with JSON conventions

```json
{
  "evidence_task": {
    "reference_id": "ET-0001",
    "tugboat_id": 328001,
    "task_name": "Access Control Documentation",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2025-09-10T14:30:00Z",
    "due_date": "2025-12-31T23:59:59Z",
    "automation_level": "high",
    "quality_score": 95
  }
}
```

### Tool Metadata

```json
{
  "meta": {
    "correlation_id": "uuid-v4-string",
    "task_ref": "ET-0001",
    "tool_name": "terraform-scanner",
    "tool_version": "1.2.0",
    "execution_timestamp": "2025-09-10T14:30:00Z",
    "duration_ms": 5234,
    "auth_status": "authenticated",
    "data_source": "file_system",
    "schema_version": "1.0"
  }
}
```

## Validation and Normalization

### Reference ID Validation

```go
type ReferenceIDValidator struct {
    patterns map[string]*regexp.Regexp
}

func NewReferenceIDValidator() *ReferenceIDValidator {
    return &ReferenceIDValidator{
        patterns: map[string]*regexp.Regexp{
            "evidence_task": regexp.MustCompile(`^ET-[0-9]{4}$`),
            "policy":        regexp.MustCompile(`^POL-[0-9]{4}$`),
            "control_iso":   regexp.MustCompile(`^[A-Z]{2}-[0-9]{2}$`),
            "control_soc2":  regexp.MustCompile(`^CC-[0-9]{1,2}\.[0-9]$`),
        },
    }
}

func (v *ReferenceIDValidator) Validate(refType, refID string) error {
    pattern, exists := v.patterns[refType]
    if !exists {
        return fmt.Errorf("unknown reference type: %s", refType)
    }
    
    if !pattern.MatchString(refID) {
        return fmt.Errorf("invalid reference ID format: %s", refID)
    }
    
    return nil
}
```

### Input Normalization Functions

```go
// NormalizeEvidenceTaskRef normalizes various evidence task reference formats
func NormalizeEvidenceTaskRef(input string) (string, error) {
    input = strings.TrimSpace(strings.ToUpper(input))
    
    // Handle Tugboat ID lookup
    if matched, _ := regexp.MatchString(`^[0-9]+$`, input); matched {
        tugboatID, _ := strconv.Atoi(input)
        return LookupEvidenceTaskByTugboatID(tugboatID)
    }
    
    // Handle various ET formats
    patterns := []struct {
        regex *regexp.Regexp
        replacement string
    }{
        {regexp.MustCompile(`^ET-?([0-9]{1,4})$`), "ET-${1}"},
        {regexp.MustCompile(`^ET\s+([0-9]{1,4})$`), "ET-${1}"},
    }
    
    for _, pattern := range patterns {
        if pattern.regex.MatchString(input) {
            normalized := pattern.regex.ReplaceAllString(input, pattern.replacement)
            // Ensure 4-digit padding
            parts := strings.Split(normalized, "-")
            if len(parts) == 2 {
                num, _ := strconv.Atoi(parts[1])
                return fmt.Sprintf("ET-%04d", num), nil
            }
        }
    }
    
    return "", fmt.Errorf("invalid evidence task reference: %s", input)
}
```

## CLI Command Naming

### Command Structure

**Format**: noun-verb or descriptive-noun

```bash
# Core commands (verbs)
grctool auth login
grctool sync --full
grctool config validate

# Evidence commands (noun-verb)
grctool evidence list
grctool evidence generate
grctool evidence validate

# Tool commands (descriptive-noun)
grctool tool terraform-scanner
grctool tool github-permissions
grctool tool evidence-task-details
```

### Flag Naming

**Format**: kebab-case for multi-word, prefer full words

```bash
# Correct flag naming
--task-ref ET-0001              # Clear, descriptive
--output-dir ./evidence         # Full words preferred
--include-collaborators         # Boolean flags descriptive
--max-files 1000               # Clear parameter naming

# Avoid abbreviated flags in main interface
--repository myorg/myrepo       # Not --repo
--framework soc2               # Not --fw
```

## Error Code Naming

### Standard Error Codes

**Format**: SCREAMING_SNAKE_CASE with descriptive prefixes

```go
const (
    // Authentication errors
    ErrAuthRequired        = "AUTH_REQUIRED"
    ErrAuthExpired        = "AUTH_EXPIRED"
    ErrAuthInvalid        = "AUTH_INVALID"
    
    // Validation errors
    ErrInvalidInput       = "INVALID_INPUT"
    ErrInvalidTaskRef     = "INVALID_TASK_REF"
    ErrInvalidPath        = "INVALID_PATH"
    
    // Resource errors
    ErrNotFound           = "NOT_FOUND"
    ErrAlreadyExists      = "ALREADY_EXISTS"
    ErrPermissionDenied   = "PERMISSION_DENIED"
    
    // System errors
    ErrInternalError      = "INTERNAL_ERROR"
    ErrServiceUnavailable = "SERVICE_UNAVAILABLE"
    ErrTimeoutExceeded    = "TIMEOUT_EXCEEDED"
)
```

## Migration and Compatibility

### Backward Compatibility Rules

1. **Reference ID Changes**: Must maintain lookup compatibility for 12 months
2. **File Name Changes**: Must support both old and new formats during transition
3. **API Changes**: Must version API responses and support previous version
4. **CLI Changes**: Must maintain command aliases for deprecated formats

### Migration Utilities

```go
// Support legacy reference formats during transition
type LegacyReferenceMapper struct {
    mappings map[string]string
}

func (m *LegacyReferenceMapper) MapLegacyReference(legacy string) (string, error) {
    if modern, exists := m.mappings[legacy]; exists {
        return modern, nil
    }
    
    // Try automatic normalization
    return NormalizeEvidenceTaskRef(legacy)
}
```

## Best Practices Summary

### Reference IDs
- ✅ Use consistent zero-padding: `ET-0001` not `ET-1`
- ✅ Maintain case consistency: `ET-0001` not `et-0001`
- ✅ Support flexible input, normalize to standard format
- ✅ Include framework-specific considerations (SOC 2 periods vs underscores)

### File Names
- ✅ Use descriptive, sanitized names
- ✅ Include all identifying information (type, ref, ID, name)
- ✅ Maintain consistent component ordering
- ✅ Limit length for filesystem compatibility

### Directories
- ✅ Use lowercase for directory names
- ✅ Prefer single words, use snake_case for multi-word
- ✅ Group related files logically
- ✅ Maintain consistent structure across environments

### Tools and Commands
- ✅ Use kebab-case for CLI commands and tool names
- ✅ Use snake_case for internal Go packages
- ✅ Choose descriptive names over abbreviations
- ✅ Maintain consistent verb-noun or noun-verb patterns

## References

- [[data-formats]] - JSON schemas and data structure formats
- [[cli-commands]] - Command-line interface usage and examples
- [[api-documentation]] - Internal API naming and conventions
- [[glossary]] - Terms and definitions for naming consistency
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) - Go naming conventions
- [RFC 3986](https://tools.ietf.org/html/rfc3986) - URI syntax and naming
- [ISO 8601](https://www.iso.org/iso-8601-date-and-time-format.html) - Date and time format standards

---

*These conventions are enforced through automated validation and linting. For updates, see the [[helix/06-iterate/roadmap-feedback|Naming Standards Working Group]] ADRs.*