# Terraform Analysis Tools - Technical Specification

**Version:** 1.1
**Date:** 2025-01-10
**Purpose:** Evidence collection for infrastructure security configuration (ET-001 and related tasks)
**Tool Count:** 7 tools (4 base + 3 advanced)

## Overview

This specification defines the Terraform analysis toolchain for automated compliance evidence collection. These tools scan Infrastructure-as-Code (Terraform) configurations to extract security-relevant settings for SOC2/ISO27001 compliance.

## Tool Suite Architecture

```
┌─────────────────────────────────────────────────────────┐
│          Evidence Collection Request (ET-001)            │
└─────────────────┬───────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────┐
│              Tool Selection Layer (Claude)               │
│   Selects appropriate tools based on task requirements   │
└─────────────┬───────────────────────────────────────────┘
              │
              ├──────────────┬──────────────┬──────────────┐
              ▼              ▼              ▼              ▼
       ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐
       │Terraform │   │   HCL    │   │ Security │   │  Query   │
       │ Scanner  │   │  Parser  │   │ Analyzer │   │Interface │
       └──────────┘   └──────────┘   └──────────┘   └──────────┘
              │              │              │              │
              └──────────────┴──────────────┴──────────────┘
                            ▼
                  ┌────────────────────┐
                  │  Evidence Assembly │
                  │   (Claude + Tools) │
                  └────────────────────┘
```

## Tool 1: Terraform Scanner (`terraform-scanner`)

### Purpose
Fast, pattern-based scanning of Terraform files for specific resource types and configurations. Optimized for quick filtering and initial resource discovery.

### Implementation Status
✅ **Implemented** as `terraform_analyzer` in `internal/tools/terraform.go`

### Claude Tool Definition

```json
{
  "name": "terraform-scanner",
  "description": "Scans Terraform configuration files for specific resource types, security controls, and compliance patterns. Provides fast filtering and CSV/markdown reports.",
  "input_schema": {
    "type": "object",
    "properties": {
      "analysis_type": {
        "type": "string",
        "description": "Type of analysis to perform",
        "enum": [
          "security_controls",
          "resource_types",
          "compliance_check",
          "security_issues",
          "modules",
          "data_sources",
          "locals"
        ],
        "default": "security_controls"
      },
      "resource_types": {
        "type": "array",
        "description": "Resource types to scan for (e.g., aws_iam_role, aws_s3_bucket). Leave empty to scan all.",
        "items": {"type": "string"}
      },
      "security_controls": {
        "type": "array",
        "description": "Security controls to find evidence for (e.g., encryption, access_control, logging)",
        "items": {"type": "string"}
      },
      "control_codes": {
        "type": "array",
        "description": "SOC2 control codes to filter by (e.g., CC6.1, CC6.8)",
        "items": {"type": "string"}
      },
      "file_patterns": {
        "type": "array",
        "description": "File patterns to filter analysis (e.g., security.tf, main.tf)",
        "items": {"type": "string"}
      },
      "standards": {
        "type": "array",
        "description": "Compliance standards to check (e.g., SOC2, PCI)",
        "items": {"type": "string"}
      },
      "output_format": {
        "type": "string",
        "description": "Output format",
        "enum": ["csv", "markdown"],
        "default": "csv"
      }
    },
    "required": ["analysis_type"]
  }
}
```

### Capabilities

#### 1. Resource Type Scanning
- Pattern-based regex scanning for Terraform resource blocks
- Supports glob patterns (e.g., `aws_*`, `azurerm_security_*`)
- Returns file path, line range, resource name, and basic configuration

**Example Request:**
```json
{
  "analysis_type": "resource_types",
  "resource_types": ["aws_security_group", "aws_network_acl", "aws_vpc"],
  "output_format": "csv"
}
```

**Example Output:**
```csv
Resource Type,Resource Name,File Path,Line Range,Security Controls,Key Configuration
aws_security_group,web_server_sg,terraform/network.tf,45-72,CC6.6;CC7.1,vpc_id=vpc-main;ingress=...
aws_vpc,main,terraform/network.tf,15-35,CC6.6;CC7.1,cidr_block=10.0.0.0/16;enable_dns_support=true
```

#### 2. Security Control Mapping
- Maps resources to SOC2 control codes (CC6.1, CC6.8, etc.)
- Filters results by requested control codes
- Identifies security-relevant configurations

**SOC2 Control Mapping:**
| Resource Type | SOC2 Controls | Evidence Use Case |
|--------------|---------------|-------------------|
| `aws_iam_role` | CC6.1, CC6.3 | Access control, least privilege |
| `aws_kms_key` | CC6.8 | Encryption at rest |
| `aws_security_group` | CC6.6, CC7.1 | Network security, firewall rules |
| `aws_s3_bucket` | CC6.8, CC7.2 | Data protection, logging |
| `aws_cloudtrail` | CC7.2, CC7.4 | Audit logging, monitoring |
| `aws_autoscaling_group` | SO2 | System availability, resilience |

**Example Request:**
```json
{
  "analysis_type": "security_controls",
  "control_codes": ["CC6.8"],
  "output_format": "markdown"
}
```

#### 3. Compliance Checks
- Filters resources by compliance standard (SOC2, PCI, ISO27001)
- Identifies security-relevant patterns
- Flags potential compliance gaps

**Example Request:**
```json
{
  "analysis_type": "compliance_check",
  "standards": ["SOC2"],
  "output_format": "csv"
}
```

#### 4. Security Issue Detection
- Identifies overly permissive configurations (0.0.0.0/0 rules)
- Detects missing encryption settings
- Flags public access configurations

**Detection Patterns:**
- CIDR blocks with `0.0.0.0/0` on sensitive ports
- IAM actions/resources with `*` wildcards
- Public access flags set to `true`
- Missing encryption configurations

### Performance Characteristics
- **Speed:** Fast (regex-based, no AST parsing)
- **Coverage:** Pattern-based (may miss complex references)
- **Best For:** Initial resource discovery, control mapping, quick scans
- **Limitations:** Cannot parse complex HCL expressions or interpolations

### Configuration

From `.grctool.yaml`:
```yaml
evidence:
  tools:
    terraform:
      enabled: true
      scan_paths:
        - "../infrastructure/terraform"
        - "../terraform"
      include_patterns:
        - "*.tf"
        - "*.tfvars"
      exclude_patterns:
        - "*.tfstate"
        - ".terraform/*"
        - "**/modules/**"  # Optional: exclude third-party modules
```

### Output Format

**CSV Format:**
```csv
Resource Type,Resource Name,File Path,Line Range,Security Controls,Key Configuration
```

**Markdown Format:**
```markdown
# Terraform Security Configuration Evidence

Generated: 2025-10-07T15:30:00Z
Total Resources: 47

## aws_security_group

### web_server_sg

**File:** `terraform/network.tf` (lines 45-72)

**Security Controls:** `CC6.6`, `CC7.1`

**Configuration:**
- **vpc_id:** `vpc-main`
- **ingress:** `[{from_port=443, to_port=443, protocol=tcp, cidr_blocks=[0.0.0.0/0]}]`
```

### Evidence Source Metadata

```json
{
  "type": "terraform_analyzer",
  "resource": "Analyzed 47 Terraform resources (security_controls)",
  "relevance": 0.85,
  "extracted_at": "2025-10-07T15:30:00Z",
  "metadata": {
    "analysis_type": "security_controls",
    "resource_count": 47,
    "scan_paths": ["../infrastructure/terraform"],
    "format": "csv",
    "security_controls": ["encryption"],
    "file_patterns": [],
    "standards": []
  }
}
```

---

## Tool 2: HCL Parser (`terraform-hcl-parser`)

### Purpose
Deep HCL syntax parsing with full AST analysis. Extracts complete resource configurations, resolves expressions, and analyzes infrastructure topology.

### Implementation Status
✅ **Implemented** in `internal/tools/terraform/hcl.go` (needs tool registration)

### Claude Tool Definition

```json
{
  "name": "terraform-hcl-parser",
  "description": "Comprehensive HCL parser for Terraform configurations. Provides deep syntax analysis, expression evaluation, dependency mapping, and infrastructure topology visualization.",
  "input_schema": {
    "type": "object",
    "properties": {
      "scan_paths": {
        "type": "array",
        "description": "Paths to Terraform directories to parse",
        "items": {"type": "string"}
      },
      "include_modules": {
        "type": "boolean",
        "description": "Include module calls in analysis",
        "default": true
      },
      "analyze_security": {
        "type": "boolean",
        "description": "Perform security configuration analysis",
        "default": true
      },
      "analyze_ha": {
        "type": "boolean",
        "description": "Analyze high availability and multi-AZ configurations",
        "default": true
      },
      "control_mapping": {
        "type": "array",
        "description": "SOC2 control codes to map resources to",
        "items": {"type": "string"}
      },
      "include_diagnostics": {
        "type": "boolean",
        "description": "Include HCL parsing diagnostics in output",
        "default": false
      },
      "output_format": {
        "type": "string",
        "description": "Output format",
        "enum": ["detailed", "summary", "security-only"],
        "default": "detailed"
      }
    },
    "required": []
  }
}
```

### Capabilities

#### 1. Full HCL AST Parsing
- Uses HashiCorp HCL parser library
- Parses all Terraform block types:
  - `resource` blocks
  - `data` sources
  - `variable` declarations
  - `output` definitions
  - `locals` blocks
  - `module` calls
  - `provider` configurations
- Evaluates expressions and interpolations
- Resolves variable references

#### 2. Infrastructure Topology Mapping
- Builds resource dependency graph
- Identifies VPC → Subnet → Instance relationships
- Maps security group attachments
- Traces encryption key usage

**Example Topology Output:**
```json
{
  "infrastructure_map": {
    "vpcs": [
      {
        "id": "aws_vpc.main",
        "cidr": "10.0.0.0/16",
        "subnets": ["aws_subnet.public_1a", "aws_subnet.public_1b"],
        "security_groups": ["aws_security_group.web", "aws_security_group.db"]
      }
    ],
    "encryption_summary": {
      "at_rest_encryption": {
        "enabled": true,
        "kms_keys": ["aws_kms_key.main"],
        "encrypted_resources": ["aws_s3_bucket.data", "aws_rds_cluster.main"]
      },
      "in_transit_encryption": {
        "enabled": true,
        "tls_version": "1.2",
        "certificates": ["aws_acm_certificate.web"]
      }
    },
    "ha_analysis": {
      "multi_az_resources": ["aws_rds_cluster.main", "aws_lb.web"],
      "autoscaling_groups": ["aws_autoscaling_group.web"],
      "load_balancers": ["aws_lb.web"]
    }
  }
}
```

#### 3. Multi-AZ Analysis
- Detects `multi_az = true` settings
- Analyzes `availability_zones` configurations
- Identifies subnet spread patterns
- Maps load balancer zone configurations

**Multi-AZ Detection Patterns:**
```hcl
# Explicit multi-AZ
resource "aws_rds_instance" "db" {
  multi_az = true  # ← Detected
}

# Subnet list spanning zones
resource "aws_lb" "web" {
  subnets = [
    aws_subnet.public_1a.id,  # us-east-1a
    aws_subnet.public_1b.id   # us-east-1b  ← Multi-AZ detected
  ]
}

# Dynamic availability zone reference
resource "aws_subnet" "public" {
  count = length(data.aws_availability_zones.available.names)
  availability_zone = data.aws_availability_zones.available.names[count.index]
  # ← Multi-AZ detected from count pattern
}
```

#### 4. Encryption Configuration Extraction
- Identifies KMS key usage
- Maps encryption settings by resource
- Analyzes encryption at rest vs. in transit
- Detects missing encryption

**Encryption Evidence:**
```json
{
  "encryption_configs": [
    {
      "resource_type": "aws_s3_bucket",
      "resource_name": "data_bucket",
      "kms_key_id": "aws_kms_key.main",
      "encryption_enabled": true,
      "encryption_type": "AES256",
      "file_path": "terraform/storage.tf",
      "line_range": "34-52"
    }
  ]
}
```

#### 5. Dependency Analysis
- Extracts `depends_on` relationships
- Infers implicit dependencies from references
- Builds resource creation order
- Identifies circular dependencies

### Performance Characteristics
- **Speed:** Slower (full AST parsing)
- **Coverage:** Complete (handles all HCL syntax)
- **Best For:** Detailed configuration extraction, topology mapping, HA analysis
- **Limitations:** Requires valid HCL syntax

### Output Format

**Detailed JSON:**
```json
{
  "parsed_at": "2025-10-07T15:30:00Z",
  "tool_version": "1.0.0",
  "modules": [
    {
      "file_path": "terraform/main.tf",
      "resources": [...],
      "data_sources": [...],
      "variables": [...],
      "outputs": [...]
    }
  ],
  "infrastructure_map": {...},
  "dependencies": [...],
  "security_findings": [...],
  "parse_summary": {
    "files_processed": 12,
    "total_resources": 87,
    "module_counts": 3,
    "security_findings": 5,
    "ha_findings": {
      "multi_az_resources": 8,
      "single_az_resources": 12
    }
  }
}
```

---

## Tool 3: Security Analyzer (`terraform-security-analyzer`)

### Purpose
Comprehensive security configuration analysis with SOC2 control mapping, compliance gap detection, and security finding categorization.

### Implementation Status
✅ **Implemented** in `internal/tools/terraform/security.go`

### Claude Tool Definition

```json
{
  "name": "terraform-security-analyzer",
  "description": "Comprehensive security configuration analyzer for Terraform manifests with SOC2 control mapping, compliance gap detection, and remediation recommendations.",
  "input_schema": {
    "type": "object",
    "properties": {
      "security_domain": {
        "type": "string",
        "description": "Security domain to focus on",
        "enum": ["encryption", "iam", "network", "backup", "monitoring", "all"],
        "default": "all"
      },
      "soc2_controls": {
        "type": "array",
        "description": "Specific SOC2 controls to find evidence for",
        "items": {"type": "string"}
      },
      "evidence_tasks": {
        "type": "array",
        "description": "Evidence task IDs to address (e.g., ET-001, ET-021, ET-023)",
        "items": {"type": "string"}
      },
      "include_compliance_gaps": {
        "type": "boolean",
        "description": "Include analysis of potential compliance gaps and recommendations",
        "default": true
      },
      "extract_sensitive_configs": {
        "type": "boolean",
        "description": "Extract detailed security configurations (excludes actual secrets)",
        "default": true
      },
      "output_format": {
        "type": "string",
        "description": "Output format",
        "enum": ["detailed_json", "summary_markdown", "compliance_csv"],
        "default": "detailed_json"
      }
    },
    "required": []
  }
}
```

### Capabilities

#### 1. Security Domain Analysis

**Encryption Domain:**
- KMS key configurations
- S3 bucket encryption settings
- RDS/database encryption
- EBS volume encryption
- Encryption in transit (TLS/SSL)
- Certificate management

**IAM Domain:**
- Role definitions
- Policy documents
- Least privilege analysis
- Wildcard permission detection
- Assume role policies
- MFA requirements

**Network Domain:**
- VPC configurations
- Security group rules
- Network ACLs
- Subnet isolation
- Internet gateway exposure
- NAT gateway configurations

**Backup Domain:**
- Backup retention policies
- S3 versioning
- RDS snapshots
- Point-in-time recovery

**Monitoring Domain:**
- CloudTrail logging
- CloudWatch configurations
- Log retention
- Security event monitoring
- GuardDuty enablement

#### 2. Evidence Task Mapping

Maps resources to specific evidence tasks:

| Resource Type | Evidence Tasks | Purpose |
|--------------|----------------|---------|
| `aws_s3_bucket` + encryption | ET-023 | Data Encryption at Rest |
| `aws_kms_key` | ET-023 | Key Management Evidence |
| `aws_lb_listener` + SSL | ET-021 | Encryption in Transit |
| `aws_acm_certificate` | ET-021 | TLS Certificate Management |
| `aws_iam_role` | ET-047 | Access Control Configuration |
| `aws_security_group` | ET-071 | Firewall Configurations |
| `aws_vpc` + multi-AZ | ET-103 | High Availability Setup |

**Example Request for ET-001:**
```json
{
  "security_domain": "all",
  "evidence_tasks": ["ET-001"],
  "soc2_controls": ["CC6.1", "CC6.6", "CC6.8"],
  "include_compliance_gaps": true,
  "output_format": "summary_markdown"
}
```

#### 3. Security Finding Detection

**High Severity Findings:**
- S3 buckets without encryption
- RDS instances without storage encryption
- Security groups with 0.0.0.0/0 on sensitive ports
- IAM policies with wildcard actions on wildcard resources
- No CloudTrail logging

**Medium Severity Findings:**
- IAM policies with overly broad permissions
- Security groups allowing all traffic within VPC
- Missing MFA requirements
- Short backup retention periods

**Low Severity Findings:**
- Missing resource tags
- Inconsistent naming conventions
- Potential optimization opportunities

**Example Finding:**
```json
{
  "type": "encryption",
  "severity": "high",
  "description": "S3 bucket does not have server-side encryption configured",
  "recommendation": "Enable server-side encryption for S3 bucket",
  "soc2_controls": ["CC6.8"],
  "resource_type": "aws_s3_bucket",
  "resource_name": "data_bucket",
  "file_path": "terraform/storage.tf",
  "line_range": "12-25"
}
```

#### 4. Compliance Gap Analysis

Identifies missing security configurations:

**Encryption Gaps:**
```json
{
  "type": "encryption",
  "severity": "high",
  "description": "No encryption configurations found in Terraform manifests",
  "soc2_controls": ["CC6.8"],
  "evidence_tasks": ["ET-021", "ET-023"],
  "recommendations": [
    "Implement KMS key management for encryption at rest",
    "Configure SSL/TLS for encryption in transit",
    "Enable S3 bucket encryption",
    "Enable RDS encryption"
  ]
}
```

**IAM Gaps:**
```json
{
  "type": "iam",
  "severity": "medium",
  "description": "Limited IAM configurations found - may indicate insufficient access controls",
  "soc2_controls": ["CC6.1", "CC6.3"],
  "evidence_tasks": ["ET-047"],
  "recommendations": [
    "Implement comprehensive IAM role-based access control",
    "Use IAM policies with least privilege principles",
    "Configure multi-factor authentication requirements"
  ]
}
```

#### 5. SOC2 Control Evidence Mapping

**For ET-001 (Infrastructure Security Configuration Evidence):**

Maps to control **CC6.1** (Logical and Physical Access Controls):
- IAM roles with least privilege
- Security group configurations
- Network access restrictions

**Example Output:**
```json
{
  "soc2_control_mapping": {
    "CC6.1": [
      {
        "resource_type": "aws_iam_role",
        "resource_name": "application_role",
        "file_path": "terraform/iam.tf",
        "line_range": "45-78",
        "configuration": {
          "assume_role_policy": "...",
          "managed_policy_arns": ["arn:aws:iam::aws:policy/ReadOnlyAccess"]
        },
        "security_findings": []
      }
    ],
    "CC6.6": [
      {
        "resource_type": "aws_security_group",
        "resource_name": "web_server_sg",
        "file_path": "terraform/network.tf",
        "line_range": "23-56",
        "configuration": {
          "ingress_rules": [...]
        },
        "security_findings": [
          {
            "type": "network",
            "severity": "high",
            "description": "Security group allows unrestricted ingress (0.0.0.0/0)"
          }
        ]
      }
    ]
  }
}
```

### Output Formats

#### Detailed JSON
Complete analysis with all resources, findings, and mappings.

#### Summary Markdown
Executive summary with key findings and recommendations:

```markdown
# Terraform Security Configuration Analysis

**Analysis Date:** 2025-10-07T15:30:00Z
**Security Domain:** all
**Files Analyzed:** 12
**Security Resources Found:** 47

## Executive Summary

- **Encryption Configurations:** 8
- **IAM Configurations:** 12
- **Network Configurations:** 15
- **Backup Configurations:** 3
- **Monitoring Configurations:** 5
- **Compliance Gaps:** 2

## SOC2 Control Mapping

### CC6.1 (12 resources)
- **application_role** (aws_iam_role) - `terraform/iam.tf`
- **database_role** (aws_iam_role) - `terraform/iam.tf`
...

### CC6.8 (8 resources)
- **data_bucket** (aws_s3_bucket) - `terraform/storage.tf`
- **main_kms_key** (aws_kms_key) - `terraform/encryption.tf`
...

## Compliance Gaps

### Monitoring (medium severity)
**Description:** No monitoring/logging configurations found

**Recommendations:**
- Enable CloudTrail for API logging
- Configure CloudWatch for monitoring and alerting
- Set up log retention policies
- Implement security event monitoring
```

#### Compliance CSV
Tabular format for spreadsheet analysis:

```csv
Resource Type,Resource Name,File Path,Line Range,SOC2 Controls,Evidence Tasks,Security Findings,Compliance Status
aws_s3_bucket,data_bucket,terraform/storage.tf,12-25,CC6.8;CC7.2,ET-023,encryption:high,NON_COMPLIANT
aws_kms_key,main,terraform/encryption.tf,5-18,CC6.8,ET-023,,COMPLIANT
aws_iam_role,app_role,terraform/iam.tf,45-78,CC6.1;CC6.3,ET-047,iam:medium,PARTIALLY_COMPLIANT
```

---

## Tool 4: Query Interface (`terraform-query-interface`)

### Purpose
Alternative/advanced tool for flexible querying of Terraform configurations using a domain-specific query language.

### Implementation Status
✅ **COMPLETE & REGISTERED** in `internal/tools/terraform/query_interface.go`

### Claude Tool Definition

```json
{
  "name": "terraform-query-interface",
  "description": "Advanced query interface for Terraform configurations. Supports flexible filtering, aggregation, and custom queries using a simplified query syntax.",
  "input_schema": {
    "type": "object",
    "properties": {
      "query_type": {
        "type": "string",
        "description": "Type of query to execute",
        "enum": ["resource", "security", "compliance", "topology", "custom"],
        "default": "resource"
      },
      "filters": {
        "type": "object",
        "description": "Filter criteria for resources",
        "properties": {
          "resource_type": {"type": "string"},
          "has_tag": {"type": "string"},
          "in_vpc": {"type": "string"},
          "control_codes": {"type": "array", "items": {"type": "string"}},
          "severity": {"type": "string"}
        }
      },
      "aggregations": {
        "type": "array",
        "description": "Aggregation functions to apply",
        "items": {
          "type": "string",
          "enum": ["count", "group_by_type", "group_by_control", "summarize"]
        }
      },
      "output_format": {
        "type": "string",
        "enum": ["json", "csv", "markdown"],
        "default": "json"
      }
    },
    "required": ["query_type"]
  }
}
```

### Use Cases

**Example: Find all resources in a specific VPC**
```json
{
  "query_type": "resource",
  "filters": {
    "in_vpc": "aws_vpc.main"
  },
  "aggregations": ["group_by_type"]
}
```

**Example: Count security findings by severity**
```json
{
  "query_type": "security",
  "aggregations": ["summarize"],
  "filters": {
    "severity": "high"
  }
}
```

---

## Tool 5: Terraform Snippets Extractor (`terraform-snippets`)

### Purpose
Extract relevant Terraform code snippets for specific SOC2 controls and security patterns. Enables precise code citation in compliance evidence by returning exact file locations and context for security configurations.

### Implementation Status
✅ **COMPLETE & REGISTERED** in `internal/tools/terraform/snippets.go`

### Claude Tool Definition

```json
{
  "name": "terraform-snippets",
  "description": "Extract relevant Terraform code snippets for specific SOC2 controls and compliance patterns. Returns exact file locations with context for security configurations.",
  "input_schema": {
    "type": "object",
    "properties": {
      "control_codes": {
        "type": "array",
        "description": "SOC2 control codes to find snippets for (e.g., CC6.1, CC6.8)",
        "items": {"type": "string"}
      },
      "security_patterns": {
        "type": "array",
        "description": "Security patterns to search for (e.g., encryption, access_control, logging)",
        "items": {"type": "string"}
      },
      "resource_types": {
        "type": "array",
        "description": "Specific Terraform resource types to extract (e.g., aws_kms_key, aws_security_group)",
        "items": {"type": "string"}
      },
      "include_context": {
        "type": "boolean",
        "description": "Include surrounding code context (before/after lines)",
        "default": true
      },
      "context_lines": {
        "type": "integer",
        "description": "Number of context lines to include before and after match",
        "default": 5
      },
      "output_format": {
        "type": "string",
        "description": "Output format for snippets",
        "enum": ["markdown", "json", "plain"],
        "default": "markdown"
      }
    },
    "required": []
  }
}
```

### Capabilities

#### 1. Control-Based Snippet Extraction
Extract code snippets mapped to specific SOC2 controls:

**Example Request:**
```json
{
  "control_codes": ["CC6.8"],
  "include_context": true,
  "context_lines": 3,
  "output_format": "markdown"
}
```

**Example Output:**
````markdown
### CC6.8: Encryption at Rest

#### aws_kms_key.main
**File:** `terraform/encryption.tf` (lines 5-18)

```hcl
# Context (lines 2-4)
# KMS encryption configuration for data at rest
# Complies with SOC2 CC6.8 requirements

# Matched snippet (lines 5-18)
resource "aws_kms_key" "main" {
  description             = "Primary encryption key for data at rest"
  deletion_window_in_days = 10
  enable_key_rotation     = true

  tags = {
    Name        = "primary-kms-key"
    Compliance  = "SOC2-CC6.8"
    Environment = "production"
  }
}

# Context (lines 19-21)
resource "aws_kms_alias" "main" {
  name          = "alias/primary-key"
  target_key_id = aws_kms_key.main.key_id
```
````

#### 2. Pattern-Based Extraction
Search for security patterns across configurations:

**Example Request:**
```json
{
  "security_patterns": ["encryption", "multi_az"],
  "include_context": true,
  "output_format": "json"
}
```

**Example Output:**
```json
{
  "snippets": [
    {
      "pattern": "encryption",
      "resource_type": "aws_s3_bucket",
      "resource_name": "data_bucket",
      "file_path": "terraform/storage.tf",
      "line_start": 34,
      "line_end": 52,
      "code": "resource \"aws_s3_bucket\" \"data_bucket\" {\n  bucket = \"company-data-bucket\"\n  \n  server_side_encryption_configuration {\n    rule {\n      apply_server_side_encryption_by_default {\n        sse_algorithm     = \"AES256\"\n        kms_master_key_id = aws_kms_key.main.arn\n      }\n    }\n  }\n}",
      "context_before": "# S3 bucket configuration\n# Encrypted storage for sensitive data",
      "context_after": "\nresource \"aws_s3_bucket_public_access_block\" \"data_bucket\" {",
      "controls": ["CC6.8", "CC7.2"]
    }
  ]
}
```

#### 3. Resource Type Filtering
Extract specific resource types for focused analysis:

**Example Request:**
```json
{
  "resource_types": ["aws_security_group", "aws_network_acl"],
  "output_format": "plain"
}
```

#### 4. Compliance Citation Support
Generates properly formatted citations for audit evidence:

**Citation Format:**
```markdown
**Evidence Source:** Terraform Infrastructure Configuration
**File:** `terraform/network.tf` (lines 45-72)
**Control:** CC6.6 (Network Security Controls)
**Resource:** aws_security_group.web_server_sg

[Code snippet follows...]
```

### Use Cases

**1. Audit Evidence Generation**
- Extract exact code snippets for compliance documentation
- Provide auditors with precise file locations
- Show configuration context and intent

**2. Security Review**
- Quick extraction of all encryption configurations
- Identify access control patterns
- Review network security settings

**3. Change Documentation**
- Document security-relevant configurations
- Track changes to compliance controls
- Generate change management evidence

### Performance Characteristics
- **Speed:** Fast (file-based extraction with caching)
- **Precision:** Exact line numbers and context
- **Best For:** Evidence documentation, audit citations, security reviews
- **Limitations:** Requires indexed files for optimal performance

### Integration with Other Tools

Works best in combination with:
- **terraform-security-indexer**: Fast lookup of snippet locations
- **terraform-security-analyzer**: Identify which snippets need extraction
- **evidence-writer**: Assemble snippets into evidence documents

**Workflow Example:**
```python
# 1. Use indexer to find relevant resources
index_results = terraform_security_indexer.query({
    "control_codes": ["CC6.8"]
})

# 2. Extract snippets for those resources
snippets = terraform_snippets.extract({
    "resource_types": index_results.resource_types,
    "control_codes": ["CC6.8"],
    "include_context": true
})

# 3. Assemble into evidence document
evidence_writer.write({
    "snippets": snippets,
    "task_ref": "ET-023"
})
```

---

## Tool 6: Terraform Security Indexer (`terraform-security-indexer`)

### Purpose
Index-first architecture for fast compliance queries across large Terraform codebases. Pre-processes and indexes security attributes for sub-100ms query times, with persistent caching and automatic invalidation on file changes.

### Implementation Status
✅ **COMPLETE & REGISTERED** in `internal/tools/terraform/indexer.go`

### Claude Tool Definition

```json
{
  "name": "terraform-security-indexer",
  "description": "High-performance security indexer for Terraform configurations. Pre-processes and indexes security attributes for fast compliance queries with persistent caching.",
  "input_schema": {
    "type": "object",
    "properties": {
      "query_type": {
        "type": "string",
        "description": "Type of query to execute",
        "enum": ["control_mapping", "resource_lookup", "security_summary", "compliance_status", "rebuild_index"],
        "default": "control_mapping"
      },
      "control_codes": {
        "type": "array",
        "description": "SOC2 control codes to query",
        "items": {"type": "string"}
      },
      "resource_types": {
        "type": "array",
        "description": "Filter by resource types",
        "items": {"type": "string"}
      },
      "security_domains": {
        "type": "array",
        "description": "Security domains to query (encryption, iam, network, backup, monitoring)",
        "items": {"type": "string"}
      },
      "force_rebuild": {
        "type": "boolean",
        "description": "Force index rebuild even if cache is valid",
        "default": false
      },
      "output_format": {
        "type": "string",
        "enum": ["json", "summary", "csv"],
        "default": "json"
      }
    },
    "required": ["query_type"]
  }
}
```

### Architecture

#### Index Structure
```json
{
  "index_version": "1.0",
  "created_at": "2025-01-10T15:30:00Z",
  "source_paths": ["../infrastructure/terraform"],
  "file_count": 47,
  "resource_count": 312,
  "index": {
    "by_control": {
      "CC6.1": [
        {
          "resource_type": "aws_iam_role",
          "resource_name": "application_role",
          "file_path": "terraform/iam.tf",
          "line_range": [45, 78],
          "security_score": 0.95,
          "attributes": {
            "has_mfa": false,
            "least_privilege": true,
            "policy_count": 2
          }
        }
      ]
    },
    "by_resource_type": {
      "aws_kms_key": [...]
    },
    "by_security_domain": {
      "encryption": [...]
    },
    "by_file": {
      "terraform/iam.tf": {
        "last_modified": "2025-01-10T12:00:00Z",
        "resource_count": 12,
        "controls": ["CC6.1", "CC6.3"]
      }
    }
  },
  "metadata": {
    "total_index_size": 524288,
    "compression": "gzip",
    "checksum": "sha256:abc123..."
  }
}
```

#### Persistent Storage
- **Location:** `.cache/terraform_security_index.json.gz`
- **Format:** Gzip-compressed JSON
- **Size:** ~500KB for 300+ resources (10:1 compression ratio)
- **Invalidation:** Automatic on file modification time changes

### Capabilities

#### 1. Fast Control Mapping
Query resources by SOC2 control code with sub-100ms response:

**Example Request:**
```json
{
  "query_type": "control_mapping",
  "control_codes": ["CC6.8", "CC7.2"]
}
```

**Performance:**
- **Cold start** (rebuild index): 2-5 seconds for 300 resources
- **Warm cache**: <100ms query time
- **Index size**: ~50KB per 100 resources (compressed)

#### 2. Resource Lookup
Fast lookup of specific resource types:

**Example Request:**
```json
{
  "query_type": "resource_lookup",
  "resource_types": ["aws_kms_key", "aws_s3_bucket"]
}
```

#### 3. Security Summary
Aggregate statistics across security domains:

**Example Request:**
```json
{
  "query_type": "security_summary",
  "output_format": "summary"
}
```

**Example Output:**
```markdown
# Terraform Security Index Summary

**Last Updated:** 2025-01-10T15:30:00Z
**Files Indexed:** 47
**Resources Indexed:** 312

## Security Domain Coverage

| Domain | Resource Count | Control Coverage |
|--------|---------------|------------------|
| Encryption | 45 | CC6.8, CC7.2 |
| IAM | 78 | CC6.1, CC6.3 |
| Network | 123 | CC6.6, CC7.1 |
| Backup | 23 | SO2, A1.2 |
| Monitoring | 43 | CC7.2, CC7.4 |

## SOC2 Control Mapping

| Control | Resource Count | Coverage |
|---------|---------------|----------|
| CC6.1 | 78 | 89% |
| CC6.6 | 123 | 95% |
| CC6.8 | 45 | 78% |
| CC7.2 | 66 | 82% |
```

#### 4. Compliance Status
Check compliance status with automatic gap detection:

**Example Request:**
```json
{
  "query_type": "compliance_status",
  "control_codes": ["CC6.1", "CC6.6", "CC6.8"]
}
```

#### 5. Index Management
Rebuild index on demand:

**Example Request:**
```json
{
  "query_type": "rebuild_index",
  "force_rebuild": true
}
```

### Index Invalidation

Automatic invalidation triggers:
- File modification time changes
- New files added to scan paths
- File deletions
- Configuration changes

**Invalidation Strategy:**
```go
// Per-file invalidation - only rebuild changed files
if fileModTime.After(indexedModTime) {
    rebuildFileIndex(filePath)
}

// Full index rebuild on structural changes
if len(currentFiles) != len(indexedFiles) {
    rebuildFullIndex()
}
```

### Performance Benchmarks

| Operation | Cold Start | Warm Cache |
|-----------|-----------|------------|
| Index build (300 resources) | 2.5s | - |
| Control query (CC6.8) | 3.2s | 45ms |
| Resource lookup | 2.8s | 32ms |
| Security summary | 3.5s | 78ms |
| Compliance status | 3.8s | 95ms |

**Memory Usage:**
- Index in memory: ~2MB (uncompressed)
- Index on disk: ~500KB (gzip compressed)
- Peak during build: ~8MB

### Integration with Query Interface

The indexer provides the backend for the query interface:

```python
# Query interface uses indexer for fast lookups
query_results = terraform_query_interface.query({
    "query_type": "security",
    "filters": {
        "control_codes": ["CC6.8"]
    }
})

# Behind the scenes:
# 1. Query interface validates request
# 2. Indexer checks cache validity
# 3. If valid, returns indexed data (<100ms)
# 4. If invalid, rebuilds index (2-5s) then returns data
```

### Configuration

```yaml
# .grctool.yaml
evidence:
  tools:
    terraform:
      indexer:
        enabled: true
        cache_dir: ".cache"
        cache_ttl: "24h"
        auto_rebuild: true
        compression: true
        scan_paths:
          - "../infrastructure/terraform"
        exclude_patterns:
          - "*.tfstate"
          - ".terraform/*"
```

### Use Cases

**1. Large Codebase Analysis**
- 500+ Terraform files
- Need consistent <100ms query times
- Frequent compliance queries

**2. CI/CD Integration**
- Pre-build compliance checks
- Fast security validation
- Automated control mapping

**3. Evidence Generation**
- Rapid resource discovery
- Control-based evidence assembly
- Compliance status reporting

### Future Enhancements

**Planned Features:**
1. Incremental index updates (only changed files)
2. Multi-version index support (git branch comparison)
3. Index merge for multi-repository analysis
4. Custom attribute indexing
5. Index export/import for sharing

---

## Tool 7: Atmos Stack Analyzer (`atmos-stack-analyzer`)

### Purpose
Multi-environment Terraform analysis using Atmos stack configurations. Analyzes configuration drift between environments, validates security compliance across dev/staging/prod, and ensures consistent security controls.

### Implementation Status
✅ **COMPLETE & REGISTERED** in `internal/tools/terraform/atmos.go`

### Claude Tool Definition

```json
{
  "name": "atmos-stack-analyzer",
  "description": "Analyzes Atmos stack configurations and multi-environment Terraform deployments for security compliance, configuration drift, and cross-environment consistency.",
  "input_schema": {
    "type": "object",
    "properties": {
      "analysis_mode": {
        "type": "string",
        "description": "Analysis mode to execute",
        "enum": ["security", "drift", "compliance", "full"],
        "default": "full"
      },
      "environments": {
        "type": "array",
        "description": "Specific environments to analyze (e.g., dev, staging, prod). Empty for all.",
        "items": {"type": "string"}
      },
      "stack_names": {
        "type": "array",
        "description": "Specific stack names to analyze (e.g., vpc, app, db). Empty for all.",
        "items": {"type": "string"}
      },
      "compliance_frameworks": {
        "type": "array",
        "description": "Compliance frameworks to check (SOC2, ISO27001, PCI)",
        "items": {"type": "string"}
      },
      "security_focus": {
        "type": "string",
        "description": "Security domain focus",
        "enum": ["encryption", "network", "iam", "monitoring", "backup", "all"],
        "default": "all"
      },
      "include_drift_analysis": {
        "type": "boolean",
        "description": "Include configuration drift analysis between environments",
        "default": true
      },
      "output_format": {
        "type": "string",
        "enum": ["detailed_json", "summary_markdown", "compliance_csv", "drift_report"],
        "default": "detailed_json"
      }
    },
    "required": []
  }
}
```

### Capabilities

#### 1. Multi-Environment Drift Detection
Identify configuration differences between environments:

**Example Request:**
```json
{
  "analysis_mode": "drift",
  "environments": ["dev", "staging", "prod"],
  "stack_names": ["vpc", "security"],
  "output_format": "drift_report"
}
```

**Example Output:**
```markdown
# Atmos Configuration Drift Report

**Analysis Date:** 2025-01-10T15:30:00Z
**Environments:** dev, staging, prod
**Stacks Analyzed:** vpc, security

## Critical Drifts (Security Impact)

### vpc/security_group - Production Missing Encryption
**Severity:** HIGH
**Environments Affected:** prod
**Drift Type:** Security Configuration Missing

**Dev/Staging Configuration:**
```hcl
ingress {
  from_port   = 443
  to_port     = 443
  protocol    = "tcp"
  cidr_blocks = ["10.0.0.0/8"]
}
```

**Production Configuration:**
```hcl
ingress {
  from_port   = 443
  to_port     = 443
  protocol    = "tcp"
  cidr_blocks = ["0.0.0.0/0"]  # ⚠️ More permissive than dev/staging
}
```

**Recommendation:** Align production security group rules with dev/staging
**SOC2 Impact:** CC6.6 (Network Security Controls)
```

#### 2. Cross-Environment Security Validation
Ensure security controls are consistent:

**Example Request:**
```json
{
  "analysis_mode": "security",
  "environments": ["dev", "staging", "prod"],
  "security_focus": "encryption",
  "output_format": "summary_markdown"
}
```

**Example Output:**
```markdown
# Multi-Environment Security Analysis

## Encryption at Rest (CC6.8)

| Stack | Dev | Staging | Prod | Status |
|-------|-----|---------|------|--------|
| rds | ✅ AES256 | ✅ AES256 | ✅ AES256 | Consistent |
| s3 | ✅ KMS | ✅ KMS | ⚠️ None | **DRIFT DETECTED** |
| ebs | ✅ KMS | ✅ KMS | ✅ KMS | Consistent |

**Issues Found:** 1
**Recommendation:** Enable S3 encryption in production to match dev/staging
```

#### 3. Compliance Framework Mapping
Map Atmos stacks to compliance controls:

**Example Request:**
```json
{
  "analysis_mode": "compliance",
  "compliance_frameworks": ["SOC2", "ISO27001"],
  "environments": ["prod"],
  "output_format": "compliance_csv"
}
```

**Example Output:**
```csv
Framework,Control,Stack,Environment,Resource,Status,Evidence
SOC2,CC6.1,iam,prod,aws_iam_role.app,COMPLIANT,Least privilege configured
SOC2,CC6.6,vpc,prod,aws_security_group.web,NON_COMPLIANT,Allows 0.0.0.0/0
SOC2,CC6.8,rds,prod,aws_db_instance.main,COMPLIANT,Encryption enabled
ISO27001,A.9.2.1,iam,prod,aws_iam_policy.access,COMPLIANT,Access control policy
```

#### 4. Environment Consistency Checks
Validate that production has equal or stricter security than lower environments:

**Validation Rules:**
- Encryption: prod >= staging >= dev
- Network restrictions: prod <= staging <= dev (more restrictive)
- Backup retention: prod >= staging >= dev
- Logging: prod >= staging >= dev

#### 5. Stack Dependency Analysis
Map dependencies between Atmos stacks:

**Example:**
```markdown
# Stack Dependencies

app (prod)
├── depends on: vpc (prod)
├── depends on: rds (prod)
└── depends on: kms (prod)

vpc (prod)
└── depends on: network-base (prod)

rds (prod)
├── depends on: vpc (prod)
└── depends on: kms (prod)
```

### Atmos-Specific Features

#### 1. Component Analysis
Analyze Atmos component configurations:

```yaml
# atmos/stacks/dev/vpc.yaml
components:
  terraform:
    vpc:
      vars:
        cidr_block: "10.0.0.0/16"
        enable_dns_hostnames: true
        encryption_enabled: true  # ← Analyzed for security
```

#### 2. Variable Inheritance Tracking
Track variable overrides across stack hierarchy:

```markdown
# Variable Inheritance for `encryption_enabled`

Global Default: true
├── Overridden in: stacks/prod/rds.yaml (true)
├── Overridden in: stacks/staging/rds.yaml (true)
└── ⚠️ Not set in: stacks/dev/rds.yaml (defaults to false)
```

#### 3. Backend Configuration Validation
Ensure secure backend configurations:

```markdown
# Backend Security Analysis

| Environment | State Encryption | State Locking | Backup Enabled |
|-------------|-----------------|---------------|----------------|
| dev | ✅ KMS | ✅ DynamoDB | ⚠️ No |
| staging | ✅ KMS | ✅ DynamoDB | ✅ Yes |
| prod | ✅ KMS | ✅ DynamoDB | ✅ Yes |
```

### Use Cases

**1. Pre-Deployment Validation**
- Ensure production configuration is secure
- Verify no drift from approved baseline
- Validate compliance before deployment

**2. Security Audit Evidence**
- Document multi-environment security consistency
- Prove security controls across all environments
- Generate compliance reports

**3. Configuration Management**
- Track environment-specific overrides
- Identify security gaps between environments
- Ensure consistent security posture

### Performance Characteristics
- **Speed:** Medium (processes multiple environments)
- **Coverage:** Complete across all Atmos stacks
- **Best For:** Multi-environment deployments, Atmos users
- **Limitations:** Requires Atmos stack structure

---

## Tool Selection Guide for ET-001

### Recommended Tool Combination

For **ET-001 (Infrastructure Security Configuration Evidence)**, use:

1. **terraform-scanner** - Initial resource discovery
   - Quick identification of security-relevant resources
   - Filter by control codes (CC6.1, CC6.6, CC6.8)
   - Generate initial CSV of resources

2. **terraform-security-analyzer** - Comprehensive security analysis
   - Deep security configuration extraction
   - Compliance gap detection
   - SOC2 control mapping
   - Security findings categorization

3. **terraform-hcl-parser** - Infrastructure topology (optional)
   - Multi-AZ configuration analysis
   - Encryption topology mapping
   - Dependency visualization

### Execution Workflow

```python
# Step 1: Initial resource discovery
scanner_results = terraform_scanner.execute({
    "analysis_type": "security_controls",
    "control_codes": ["CC6.1", "CC6.6", "CC6.8"],
    "output_format": "csv"
})

# Step 2: Comprehensive security analysis
security_results = terraform_security_analyzer.execute({
    "security_domain": "all",
    "evidence_tasks": ["ET-001"],
    "soc2_controls": ["CC6.1", "CC6.6", "CC6.8"],
    "include_compliance_gaps": true,
    "output_format": "summary_markdown"
})

# Step 3: Multi-AZ and HA analysis (if needed)
hcl_results = terraform_hcl_parser.execute({
    "analyze_ha": true,
    "analyze_security": true,
    "control_mapping": ["CC6.6", "CC7.1"],
    "output_format": "summary"
})

# Step 4: Assemble evidence with Claude
evidence = claude.assemble_evidence(
    task="ET-001",
    sources=[scanner_results, security_results, hcl_results],
    requirements=ET001_REQUIREMENTS
)
```

---

## Performance Comparison

| Tool | Speed | Accuracy | Coverage | Best For |
|------|-------|----------|----------|----------|
| **terraform-scanner** | ⚡⚡⚡ Fast | ⭐⭐ Good | ⭐⭐ Pattern-based | Quick filtering, control mapping |
| **terraform-hcl-parser** | ⚡ Slow | ⭐⭐⭐ Excellent | ⭐⭐⭐ Complete | Detailed configs, topology |
| **terraform-security-analyzer** | ⚡⚡ Medium | ⭐⭐⭐ Excellent | ⭐⭐⭐ Complete | Security analysis, compliance |
| **terraform-query-interface** | ⚡⚡ Medium | ⭐⭐⭐ Excellent | ⭐⭐⭐ Flexible | Custom queries, advanced filtering |
| **terraform-snippets** | ⚡⚡⚡ Fast | ⭐⭐⭐ Exact | ⭐⭐ Targeted | Evidence citations, code extraction |
| **terraform-security-indexer** | ⚡⚡⚡ Very Fast* | ⭐⭐⭐ Excellent | ⭐⭐⭐ Complete | Large codebases, repeated queries |
| **atmos-stack-analyzer** | ⚡⚡ Medium | ⭐⭐⭐ Excellent | ⭐⭐⭐ Multi-env | Drift detection, Atmos deployments |

*Sub-100ms query time with warm cache; 2-5s cold start for index build

---

## Configuration Requirements

### Terraform File Structure

Tools expect standard Terraform project structure:

```
infrastructure/
├── terraform/
│   ├── main.tf
│   ├── variables.tf
│   ├── outputs.tf
│   ├── network.tf
│   ├── security.tf
│   ├── storage.tf
│   ├── iam.tf
│   └── modules/
│       ├── vpc/
│       ├── rds/
│       └── eks/
```

### Scan Path Configuration

```yaml
# .grctool.yaml
evidence:
  tools:
    terraform:
      enabled: true
      scan_paths:
        - "../infrastructure/terraform"
        - "./terraform"
      include_patterns:
        - "*.tf"
        - "*.tfvars"
      exclude_patterns:
        - "*.tfstate"
        - "*.tfstate.backup"
        - ".terraform/*"
        - ".terraform.lock.hcl"
```

---

## Error Handling

### Common Errors

**1. Invalid HCL Syntax**
```json
{
  "error": "HCL parsing failed",
  "file": "terraform/main.tf",
  "line": 45,
  "diagnostics": ["Expected closing brace"]
}
```

**Mitigation:** Use `terraform-scanner` (regex-based) as fallback

**2. Missing Configuration Files**
```json
{
  "error": "No Terraform files found",
  "scan_paths": ["../infrastructure/terraform"],
  "recommendation": "Check scan_paths configuration"
}
```

**3. Permission Denied**
```json
{
  "error": "Failed to read file",
  "file": "terraform/secrets.tf",
  "recommendation": "Check file permissions"
}
```

---

## Security Considerations

### Secret Handling

All tools implement secret redaction:

```json
{
  "configuration": {
    "api_key": "[REDACTED]",
    "password": "[REDACTED]",
    "kms_key_id": "arn:aws:kms:us-east-1:123456789012:key/abc-123"  // ✅ ARNs are preserved
  }
}
```

**Redacted Fields:**
- `password`
- `secret`
- `token`
- `api_key`
- `private_key`

**Preserved Fields:**
- Resource ARNs
- KMS key IDs (identifiers only, not actual keys)
- IAM role ARNs
- Security group IDs

### File Size Limits

- Maximum file size: 10 MB
- Maximum total scan: 100 MB
- Large files trigger streaming parse

---

## Testing Strategy

### Unit Tests
- Resource pattern matching
- Control code mapping
- Security finding detection
- HCL expression evaluation

### Integration Tests
- End-to-end scanning of fixture Terraform configs
- Validation against expected evidence output
- Performance benchmarking

### Test Fixtures

Located in `test/fixtures/terraform/`:
```
test/fixtures/terraform/
├── valid/
│   ├── encryption.tf          # Encryption evidence
│   ├── iam.tf                  # IAM configurations
│   ├── network.tf              # Security groups, VPCs
│   └── multi_az.tf             # HA configurations
├── invalid/
│   ├── missing_encryption.tf   # Compliance gaps
│   └── overly_permissive.tf    # Security findings
└── edge_cases/
    ├── complex_expressions.tf  # HCL parser stress test
    └── circular_deps.tf        # Dependency analysis
```

---

## Future Enhancements

### Planned Features

1. **Terraform Cloud/Enterprise Integration**
   - Query Terraform Cloud workspaces
   - Analyze remote state
   - Compare plan vs. actual

2. **Drift Detection**
   - Compare Terraform config vs. AWS actual state
   - Identify unmanaged resources
   - Detect configuration drift

3. **Cost Analysis Integration**
   - Estimate infrastructure costs
   - Identify optimization opportunities
   - Security vs. cost tradeoffs

4. **Policy-as-Code Validation**
   - Integrate with OPA (Open Policy Agent)
   - Custom policy enforcement
   - Pre-deployment validation

5. **Terraform Plan Analysis**
   - Analyze `terraform plan` output
   - Detect risky changes before apply
   - Security impact assessment

---

## References

### External Dependencies

- **HCL Parser:** `github.com/hashicorp/hcl/v2`
- **Cty Types:** `github.com/zclconf/go-cty`

### Related Documentation

- [Terraform Best Practices](https://www.terraform-best-practices.com/)
- [AWS Security Best Practices](https://docs.aws.amazon.com/security/)
- [SOC2 Trust Services Criteria](https://www.aicpa.org/soc)

### Internal Documentation

- `docs/01-User-Guide/evidence-collection-workflow.md`
- `docs/04-Development/testing-guide.md`
- `docs/04-Development/coding-standards.md`

---

**Version History:**
- v1.0 (2025-10-07): Initial specification for ET-001 support
