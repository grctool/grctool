---
title: "Data Formats and Schemas"
type: "reference"
category: "data-formats"
tags: ["json", "schemas", "data-structures", "formats", "api"]
related: ["[[api-documentation]]", "[[cli-commands]]", "[[naming-conventions]]"]
created: 2025-09-10
modified: 2025-09-10
status: "active"
---

# Data Formats and Schemas

## Overview

This document defines the standard data formats, JSON schemas, and data structures used throughout GRCTool. All data exchanges, file storage, and API responses follow these standardized formats to ensure consistency and interoperability.

## Core Data Structures

### Evidence Task Format

**File Pattern**: `ET-{NNNN}-{tugboat_id}-{sanitized_name}.json`
**Example**: `ET-0001-328001-access_control_documentation.json`

```json
{
  "$schema": "https://grctool.example.com/schemas/evidence-task/v1.0.json",
  "id": 328001,
  "reference_id": "ET-0001",
  "name": "Access Control Documentation",
  "description": "Document and evidence user access controls and permissions",
  "guidance": "Collect evidence of user access provisioning, reviews, and termination processes",
  "framework": ["soc2", "iso27001"],
  "category": "access_control",
  "priority": "high",
  "status": "pending",
  "automation_level": "high",
  "required_sources": ["github", "identity_management", "manual"],
  "controls": [
    {
      "id": "AC-01",
      "framework": "iso27001",
      "name": "Access Control Policy"
    },
    {
      "id": "CC6.1", 
      "framework": "soc2",
      "name": "Logical and Physical Access Controls"
    }
  ],
  "evidence_requirements": {
    "completeness_threshold": 0.95,
    "accuracy_threshold": 0.98,
    "review_required": false,
    "retention_period": "3_years"
  },
  "collection_metadata": {
    "estimated_effort_hours": 2,
    "complexity": "medium",
    "frequency": "quarterly",
    "automation_coverage": 0.90
  },
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2025-09-10T14:30:00Z",
  "due_date": "2025-12-31T23:59:59Z",
  "assigned_to": "compliance@company.com"
}
```

### Control Format

**File Pattern**: `{control_id}-{tugboat_id}-{sanitized_name}.json`
**Examples**: 
- `AC-01-778771-access_control_policy.json`
- `CC-01_1-778805-control_environment_design.json`

```json
{
  "$schema": "https://grctool.example.com/schemas/control/v1.0.json",
  "id": 778771,
  "reference_id": "AC-01",
  "framework": "iso27001",
  "category": "access_control",
  "name": "Access Control Policy",
  "description": "Information security policy for access control management",
  "control_type": "preventive",
  "implementation_status": "implemented",
  "frequency": "continuous",
  "owner": "security@company.com",
  "reviewers": ["compliance@company.com", "audit@company.com"],
  "implementation_details": {
    "control_activities": [
      "Access request approval workflow",
      "Regular access reviews",
      "Automated provisioning/deprovisioning"
    ],
    "evidence_sources": ["github-permissions", "identity-management", "policy-documents"],
    "monitoring": {
      "automated": true,
      "frequency": "daily",
      "metrics": ["access_requests", "approval_time", "review_completeness"]
    }
  },
  "relationships": {
    "related_controls": ["AC-02", "AC-03", "AC-05"],
    "supporting_policies": ["POL-0001"],
    "evidence_tasks": ["ET-0001", "ET-0003", "ET-0004"]
  },
  "compliance_mapping": {
    "soc2": ["CC6.1", "CC6.2"],
    "iso27001": ["A.9.1.1", "A.9.2.1"],
    "nist": ["AC-1", "AC-2"]
  },
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2025-09-10T14:30:00Z",
  "review_date": "2025-12-31T00:00:00Z"
}
```

### Policy Format

**File Pattern**: `POL-{NNNN}-{tugboat_id}-{sanitized_name}.json`
**Example**: `POL-0001-94641-information_security_policy.json`

```json
{
  "$schema": "https://grctool.example.com/schemas/policy/v1.0.json",
  "id": 94641,
  "reference_id": "POL-0001",
  "name": "Information Security Policy",
  "description": "Comprehensive information security policy document",
  "version": "2.1.0",
  "status": "approved",
  "category": "information_security",
  "owner": "ciso@company.com",
  "reviewers": ["legal@company.com", "compliance@company.com"],
  "approved_by": "ceo@company.com",
  "approved_date": "2024-12-01T00:00:00Z",
  "effective_date": "2024-12-15T00:00:00Z",
  "review_cycle": "annual",
  "next_review_date": "2025-12-01T00:00:00Z",
  "document_metadata": {
    "document_type": "policy",
    "classification": "internal",
    "retention_period": "7_years",
    "distribution": "all_employees",
    "training_required": true
  },
  "content": {
    "abstract": "This policy establishes the framework for information security...",
    "sections": [
      {
        "number": "1",
        "title": "Purpose and Scope",
        "content": "This policy applies to all employees, contractors..."
      },
      {
        "number": "2", 
        "title": "Responsibilities",
        "content": "All users are responsible for maintaining..."
      }
    ],
    "references": [
      "ISO/IEC 27001:2022",
      "NIST Cybersecurity Framework",
      "SOC 2 Trust Service Criteria"
    ]
  },
  "compliance_mapping": {
    "controls": ["AC-01", "SC-01", "SA-01"],
    "frameworks": ["soc2", "iso27001", "nist"],
    "evidence_tasks": ["ET-0090", "ET-0091", "ET-0093"]
  },
  "change_history": [
    {
      "version": "2.1.0",
      "date": "2024-12-01T00:00:00Z",
      "author": "compliance@company.com",
      "summary": "Updated access control requirements per ISO 27001:2022",
      "changes": ["Section 3.2", "Appendix A"]
    }
  ],
  "created_at": "2023-01-15T10:30:00Z",
  "updated_at": "2024-12-01T00:00:00Z"
}
```

### Evidence Record Format

**File Pattern**: `evidence-{task_ref}-{timestamp}-{version}.json`
**Example**: `evidence-ET-0001-20250910T143000Z-v1.json`

```json
{
  "$schema": "https://grctool.example.com/schemas/evidence-record/v1.0.json",
  "id": "uuid-v4-string",
  "task_id": 328001,
  "task_reference_id": "ET-0001",
  "title": "Access Control Evidence - Q4 2024",
  "content_type": "markdown",
  "content": "# Access Control Evidence\n\n## Summary\n...",
  "quality_score": 95,
  "completeness": 0.98,
  "accuracy": 0.99,
  "validation_status": "validated",
  "sources": [
    {
      "type": "automated",
      "tool": "github-permissions",
      "version": "1.2.0",
      "location": "https://github.com/myorg/myrepo",
      "timestamp": "2025-09-10T14:30:00Z",
      "checksum": "sha256:abc123...",
      "metadata": {
        "repository": "myorg/myrepo",
        "collaborators_count": 25,
        "teams_count": 5,
        "admin_users": 3
      }
    },
    {
      "type": "automated",
      "tool": "terraform-scanner", 
      "version": "1.1.0",
      "location": "./infrastructure",
      "timestamp": "2025-09-10T14:25:00Z",
      "checksum": "sha256:def456...",
      "metadata": {
        "files_scanned": 45,
        "resources_analyzed": 156,
        "security_issues": 0
      }
    }
  ],
  "metadata": {
    "framework": "soc2",
    "controls": ["CC6.1", "CC6.2"],
    "collection_time_ms": 15234,
    "automation_level": "high",
    "review_required": false,
    "tags": ["access_control", "quarterly_review", "automated"],
    "correlation_id": "uuid-v4-string",
    "audit_period": {
      "start_date": "2024-10-01T00:00:00Z",
      "end_date": "2024-12-31T23:59:59Z"
    }
  },
  "validation": {
    "rules_applied": [
      "completeness_check",
      "accuracy_validation",
      "cross_reference_check"
    ],
    "validation_date": "2025-09-10T14:35:00Z",
    "validator": "system",
    "issues": [],
    "recommendations": [
      "Consider adding manual verification for edge cases"
    ]
  },
  "version": 1,
  "created_at": "2025-09-10T14:30:00Z",
  "created_by": "system",
  "file_path": "./evidence/ET-0001/evidence-ET-0001-20250910T143000Z-v1.json"
}
```

## Tool Output Formats

### Standard Tool Envelope

All tools return consistent JSON envelopes with this structure:

```json
{
  "$schema": "https://grctool.example.com/schemas/tool-envelope/v1.0.json",
  "success": true,
  "content": "Tool-specific output content",
  "meta": {
    "correlation_id": "uuid-v4-string",
    "task_ref": "ET-0001",
    "tool": "terraform-scanner",
    "version": "1.2.0",
    "timestamp": "2025-09-10T14:30:00Z",
    "duration_ms": 5234,
    "auth_status": "authenticated",
    "data_source": "file_system",
    "schema_version": "1.0"
  },
  "error": null
}
```

### Error Format

```json
{
  "success": false,
  "content": "",
  "meta": {
    "correlation_id": "uuid-v4-string",
    "tool": "terraform-scanner",
    "version": "1.2.0",
    "timestamp": "2025-09-10T14:30:00Z",
    "duration_ms": 1234,
    "schema_version": "1.0"
  },
  "error": {
    "code": "INVALID_PATH",
    "message": "Terraform configuration path not found",
    "details": {
      "path": "./invalid-path",
      "suggestions": ["./infrastructure", "./terraform"]
    },
    "correlation_id": "uuid-v4-string"
  }
}
```

### Infrastructure Analysis Output (Terraform Scanner)

```json
{
  "success": true,
  "content": "Infrastructure security analysis complete. Found 156 resources across 45 files.",
  "meta": {
    "correlation_id": "uuid-v4",
    "task_ref": "ET-0011",
    "tool": "terraform-scanner",
    "version": "1.2.0",
    "timestamp": "2025-09-10T14:30:00Z",
    "duration_ms": 8765,
    "data_source": "file_system",
    "schema_version": "1.0"
  },
  "analysis": {
    "summary": {
      "files_scanned": 45,
      "resources_total": 156,
      "resources_analyzed": 156,
      "security_issues": 2,
      "compliance_score": 0.95
    },
    "security_findings": [
      {
        "severity": "medium",
        "type": "encryption",
        "resource": "aws_s3_bucket.data",
        "file": "./infrastructure/storage.tf",
        "line": 15,
        "message": "S3 bucket does not have encryption enabled",
        "recommendation": "Add server_side_encryption_configuration block",
        "control_references": ["AC-01", "SC-13"]
      }
    ],
    "compliance_analysis": {
      "soc2": {
        "applicable_controls": ["CC6.1", "CC6.7"],
        "compliance_percentage": 0.95,
        "findings": 1
      },
      "iso27001": {
        "applicable_controls": ["A.10.1.1", "A.13.1.1"],
        "compliance_percentage": 0.97,
        "findings": 1
      }
    },
    "resource_inventory": [
      {
        "type": "aws_security_group",
        "count": 12,
        "compliant": 11,
        "issues": ["sg-web allows 0.0.0.0/0 on port 22"]
      },
      {
        "type": "aws_s3_bucket", 
        "count": 8,
        "compliant": 7,
        "issues": ["bucket 'data' missing encryption"]
      }
    ]
  }
}
```

### Access Control Analysis Output (GitHub Permissions)

```json
{
  "success": true,
  "content": "Access control analysis complete for myorg/myrepo",
  "meta": {
    "correlation_id": "uuid-v4",
    "task_ref": "ET-0001",
    "tool": "github-permissions",
    "version": "1.1.0",
    "timestamp": "2025-09-10T14:30:00Z",
    "duration_ms": 3456,
    "auth_status": "authenticated",
    "data_source": "github_api",
    "schema_version": "1.0"
  },
  "analysis": {
    "repository": {
      "name": "myorg/myrepo",
      "visibility": "private",
      "default_branch": "main",
      "branch_protection": true
    },
    "access_summary": {
      "total_users": 25,
      "admin_users": 3,
      "write_users": 8,
      "read_users": 14,
      "teams": 5
    },
    "users": [
      {
        "login": "alice.admin",
        "permission": "admin",
        "type": "direct",
        "source": "collaborator",
        "added_date": "2024-01-15T00:00:00Z",
        "last_activity": "2025-09-10T10:00:00Z"
      },
      {
        "login": "bob.developer", 
        "permission": "write",
        "type": "team",
        "source": "backend-team",
        "added_date": "2024-03-01T00:00:00Z",
        "last_activity": "2025-09-09T16:30:00Z"
      }
    ],
    "teams": [
      {
        "name": "backend-team",
        "permission": "write", 
        "members": 5,
        "description": "Backend development team"
      }
    ],
    "security_features": {
      "branch_protection": {
        "enabled": true,
        "require_pr_reviews": 2,
        "dismiss_stale_reviews": true,
        "require_code_owner_reviews": true
      },
      "security_scanning": {
        "code_scanning": true,
        "secret_scanning": true,
        "dependency_scanning": true
      }
    },
    "compliance_assessment": {
      "access_control_score": 0.92,
      "findings": [
        {
          "type": "excessive_permissions",
          "severity": "low", 
          "description": "3 users have admin access",
          "recommendation": "Review admin access necessity"
        }
      ]
    }
  }
}
```

## Configuration File Formats

### Main Configuration (.grctool.yaml)

```yaml
# GRCTool Configuration
version: "1.0"

# Authentication configuration
auth:
  tugboat:
    # Browser-based authentication (no API keys)
    org_id: 12345
    browser: "auto"  # auto, chrome, firefox, safari
    timeout: "5m"

# Storage configuration  
storage:
  data_dir: "./docs"
  cache_dir: "./docs/.cache"
  local_data_dir: "./data"
  
# Logging configuration
logging:
  level: "info"
  file:
    enabled: true
    path: "grctool.log"
    level: "trace"
    max_size_mb: 100
    max_backups: 5
  structured: true
  redaction:
    - "authorization"
    - "cookie"
    - "api_key"
    - "token"
    - "password"
    - "secret"

# Tool-specific configuration
tools:
  terraform_scanner:
    max_files: 1000
    timeout: "300s"
    parallel: true
    max_workers: 4
    
  github_permissions:
    timeout: "60s"
    include_archived: false
    rate_limit:
      requests_per_hour: 5000
      
  google_workspace:
    timeout: "120s"
    max_documents: 500
    include_metadata: true

# VCR (HTTP recording) configuration
vcr:
  mode: "playbook"  # off, record, playbook, record_once
  cassette_dir: "./internal/tugboat/testdata/vcr_cassettes"
  redaction:
    headers:
      - "Authorization"
      - "Cookie"
    query_params:
      - "api_key"
      - "token"

# Evidence collection configuration
evidence:
  output_dir: "./evidence"
  formats: ["json", "markdown"]
  validation:
    min_quality_score: 85
    require_sources: true
  retention:
    days: 2555  # 7 years
  batch:
    size: 10
    parallel: true
    max_workers: 4

# Framework-specific configuration
frameworks:
  soc2:
    type: "type_ii"
    criteria: ["security", "availability", "confidentiality"]
  iso27001:
    version: "2022"
    scope: "information_security"

# Network and performance configuration
network:
  timeout: "30s"
  retries: 3
  retry_delay: "5s"
  
performance:
  max_concurrent_operations: 10
  memory_limit: "2GB"
  temp_dir: "/tmp/grctool"
```

## Data Validation Schemas

### JSON Schema Definitions

**Evidence Task Schema** (`schemas/evidence-task/v1.0.json`):
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://grctool.example.com/schemas/evidence-task/v1.0.json",
  "title": "Evidence Task",
  "description": "Evidence collection task definition",
  "type": "object",
  "required": ["id", "reference_id", "name", "framework", "status"],
  "properties": {
    "id": {
      "type": "integer",
      "minimum": 1,
      "description": "Tugboat Logic ID"
    },
    "reference_id": {
      "type": "string",
      "pattern": "^ET-[0-9]{4}$",
      "description": "Evidence task reference (ET-0001)"
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 200,
      "description": "Human-readable task name"
    },
    "framework": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": ["soc2", "iso27001", "nist", "pci", "hipaa"]
      },
      "minItems": 1,
      "description": "Applicable compliance frameworks"
    },
    "status": {
      "type": "string",
      "enum": ["pending", "in_progress", "completed", "overdue"],
      "description": "Current task status"
    },
    "automation_level": {
      "type": "string", 
      "enum": ["full", "high", "medium", "low", "manual"],
      "description": "Level of automation available"
    },
    "priority": {
      "type": "string",
      "enum": ["critical", "high", "medium", "low"],
      "description": "Task priority level"
    },
    "required_sources": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": ["terraform", "github", "google_workspace", "identity_management", "manual", "logs", "monitoring"]
      },
      "description": "Required evidence sources"
    },
    "controls": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "framework", "name"],
        "properties": {
          "id": {"type": "string"},
          "framework": {"type": "string"},
          "name": {"type": "string"}
        }
      }
    },
    "created_at": {
      "type": "string",
      "format": "date-time"
    },
    "updated_at": {
      "type": "string", 
      "format": "date-time"
    },
    "due_date": {
      "type": "string",
      "format": "date-time"
    }
  }
}
```

**Tool Envelope Schema** (`schemas/tool-envelope/v1.0.json`):
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://grctool.example.com/schemas/tool-envelope/v1.0.json",
  "title": "Tool Output Envelope",
  "description": "Standard envelope for all tool outputs",
  "type": "object",
  "required": ["success", "content", "meta"],
  "properties": {
    "success": {
      "type": "boolean",
      "description": "Whether the tool execution succeeded"
    },
    "content": {
      "type": "string",
      "description": "Tool output content"
    },
    "meta": {
      "type": "object",
      "required": ["correlation_id", "tool", "version", "timestamp", "duration_ms", "schema_version"],
      "properties": {
        "correlation_id": {
          "type": "string",
          "format": "uuid",
          "description": "Unique request correlation ID"
        },
        "task_ref": {
          "type": "string",
          "pattern": "^ET-[0-9]{4}$",
          "description": "Associated evidence task reference"
        },
        "tool": {
          "type": "string",
          "description": "Tool name"
        },
        "version": {
          "type": "string",
          "pattern": "^[0-9]+\\.[0-9]+\\.[0-9]+$",
          "description": "Tool version (semver)"
        },
        "timestamp": {
          "type": "string",
          "format": "date-time",
          "description": "Execution timestamp"
        },
        "duration_ms": {
          "type": "integer",
          "minimum": 0,
          "description": "Execution duration in milliseconds"
        },
        "auth_status": {
          "type": "string",
          "enum": ["authenticated", "unauthenticated", "expired", "error"],
          "description": "Authentication status for auth-required tools"
        },
        "data_source": {
          "type": "string",
          "enum": ["file_system", "github_api", "tugboat_api", "google_api", "aws_api", "cached"],
          "description": "Primary data source used"
        },
        "schema_version": {
          "type": "string",
          "description": "Schema version for this envelope"
        }
      }
    },
    "error": {
      "type": ["object", "null"],
      "properties": {
        "code": {
          "type": "string",
          "description": "Error code"
        },
        "message": {
          "type": "string",
          "description": "Human-readable error message"
        },
        "details": {
          "type": "object",
          "description": "Additional error context"
        },
        "correlation_id": {
          "type": "string",
          "format": "uuid"
        }
      }
    }
  }
}
```

## File Naming Conventions

### Standardized Reference ID Format

**Evidence Tasks**: `ET-NNNN` (4-digit zero-padded)
- Examples: `ET-0001`, `ET-0104`
- Input normalization: `ET1` → `ET-0001`, `ET-101` → `ET-0101`

**Policies**: `POL-NNNN` (4-digit zero-padded)  
- Examples: `POL-0001`, `POL-0002`
- Input normalization: `POL1` → `POL-0001`

**Controls**: Format varies by framework
- ISO 27001: `AC-NN`, `SC-NN` (2-digit zero-padded)
- SOC 2: `CC-NN_N` (underscores for subsections in filenames, periods in JSON)
- Examples: `AC-01`, `CC-01_1` (filename) / `CC-01.1` (JSON)

### File Naming Patterns

**General Pattern**: `<type>-<ref_id>-<tugboat_id>-<short_name>.json`

**Examples**:
- `ET-0001-327992-access_registration.json`
- `POL-0001-94641-information_security.json`
- `AC-01-778771-access_provisioning.json`
- `CC-01_1-778805-control_environment.json`

**Evidence Files**: `evidence-<task_ref>-<timestamp>-v<version>.json`
- `evidence-ET-0001-20250910T143000Z-v1.json`

**Tool Output Files**: `<tool>-<task_ref>-<timestamp>.json`
- `terraform-scanner-ET-0011-20250910T143000Z.json`
- `github-permissions-ET-0001-20250910T143000Z.json`

## Data Validation Rules

### Required Field Validation

**Evidence Tasks:**
- `id`: Must be positive integer
- `reference_id`: Must match pattern `ET-[0-9]{4}`
- `name`: 1-200 characters, non-empty
- `framework`: At least one valid framework
- `status`: Must be valid enum value

**Evidence Records:**
- `task_id`: Must correspond to existing evidence task
- `quality_score`: Integer 0-100
- `completeness`: Float 0.0-1.0
- `accuracy`: Float 0.0-1.0
- `sources`: At least one evidence source required

### Business Logic Validation

**Cross-Reference Validation:**
- Evidence tasks must reference valid controls
- Controls must reference valid policies
- Evidence records must reference valid tasks

**Completeness Validation:**
- High-priority evidence tasks require multiple sources
- Critical controls require automated evidence collection
- Manual evidence tasks require review approval

**Quality Validation:**
- Evidence quality score minimum thresholds by task priority
- Automated evidence must include tool version and timestamp
- Source checksums required for integrity verification

## Export and Import Formats

### CSV Export Format

**Evidence Task Export**:
```csv
Reference ID,Name,Status,Framework,Priority,Automation Level,Due Date,Assigned To
ET-0001,Access Control Documentation,pending,soc2;iso27001,high,high,2025-12-31,compliance@company.com
ET-0002,Network Security Analysis,completed,soc2,medium,full,2025-11-30,security@company.com
```

**Control Export**:
```csv
Reference ID,Name,Framework,Category,Status,Owner,Evidence Tasks
AC-01,Access Control Policy,iso27001,access_control,implemented,security@company.com,ET-0001;ET-0003
CC-6.1,Logical Access Controls,soc2,access_control,implemented,security@company.com,ET-0001;ET-0004
```

### Audit Report Format

**Executive Summary JSON**:
```json
{
  "report_metadata": {
    "generated_at": "2025-09-10T14:30:00Z",
    "period": {
      "start": "2025-01-01T00:00:00Z",
      "end": "2025-12-31T23:59:59Z"
    },
    "framework": "soc2",
    "report_type": "executive_summary"
  },
  "compliance_summary": {
    "overall_score": 0.94,
    "evidence_tasks": {
      "total": 105,
      "completed": 98,
      "pending": 5,
      "overdue": 2,
      "automation_rate": 0.90
    },
    "controls": {
      "total": 64,
      "implemented": 62,
      "in_progress": 2,
      "not_applicable": 0
    }
  },
  "findings": [
    {
      "severity": "medium",
      "category": "evidence_collection",
      "description": "2 evidence tasks are overdue",
      "tasks": ["ET-0045", "ET-0078"],
      "recommendation": "Prioritize completion of overdue evidence tasks"
    }
  ],
  "recommendations": [
    "Implement automated vendor management evidence collection",
    "Enhance manual process documentation for physical security"
  ]
}
```

## API Response Formats

### Paginated Response Format

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 25,
    "total_pages": 5,
    "total_records": 105,
    "has_next": true,
    "has_previous": false
  },
  "meta": {
    "request_id": "uuid-v4",
    "timestamp": "2025-09-10T14:30:00Z",
    "duration_ms": 234
  }
}
```

### Batch Operation Response Format

```json
{
  "batch_id": "uuid-v4",
  "status": "completed",
  "started_at": "2025-09-10T14:30:00Z",
  "completed_at": "2025-09-10T14:45:00Z",
  "duration_ms": 900000,
  "results": {
    "total_items": 50,
    "successful": 47,
    "failed": 3,
    "skipped": 0
  },
  "errors": [
    {
      "item_id": "ET-0045",
      "error": "authentication_required",
      "message": "GitHub authentication required for repository access"
    }
  ]
}
```

## Version Management

### Schema Versioning Strategy

- **Major Version**: Breaking changes to data structure
- **Minor Version**: Backward-compatible additions
- **Patch Version**: Non-structural fixes and clarifications

**Version Headers**: All JSON documents include `$schema` field with version reference
**Migration Support**: Tools support multiple schema versions with automatic upgrade paths
**Deprecation Policy**: Old schema versions supported for 12 months after replacement

### Data Migration Format

```json
{
  "migration": {
    "from_version": "1.0.0",
    "to_version": "1.1.0",
    "migration_date": "2025-09-10T14:30:00Z",
    "changes": [
      {
        "type": "field_added",
        "field": "automation_level",
        "default_value": "medium"
      },
      {
        "type": "field_renamed",
        "old_field": "task_status",
        "new_field": "status"
      }
    ]
  }
}
```

## References

- [[api-documentation]] - Internal API interfaces and service contracts
- [[cli-commands]] - Command-line interface and output formats
- [[naming-conventions]] - Reference ID standards and file naming patterns
- [[glossary]] - Terms and definitions used in data structures
- [JSON Schema Specification](https://json-schema.org/specification.html)
- [ISO 8601 Date Format](https://www.iso.org/iso-8601-date-and-time-format.html)

---

*This documentation is automatically validated against actual data structures in the codebase. Last updated: 2025-09-10*