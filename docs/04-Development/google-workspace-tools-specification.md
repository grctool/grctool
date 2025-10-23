# Google Workspace Tools - Technical Specification

**Version:** 1.0
**Date:** 2025-10-23
**Purpose:** Evidence collection from Google Workspace documents for compliance documentation
**Tool Count:** 1 tool (multi-capability)

## Overview

This specification defines the Google Workspace integration tool for automated compliance evidence collection from Google Workspace documents. This tool extracts content, metadata, and configuration from Google Drive, Docs, Sheets, and Forms to support SOC2/ISO27001 compliance requirements related to policy documentation, training materials, access controls, and security questionnaires.

## Tool Suite Architecture

```
┌──────────────────────────────────────────────────────────┐
│   Evidence Collection Request (ET-054, ET-064, etc.)     │
└─────────────────┬────────────────────────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────────────────────────┐
│            Tool Selection Layer (Claude)                  │
│   Determines document type and extraction requirements    │
└─────────────┬────────────────────────────────────────────┘
              │
              ▼
┌──────────────────────────────────────────────────────────┐
│           Google Workspace Tool (Single Entry)            │
│  Dispatches to appropriate Google API based on doc type   │
└──────────────┬───────────────────────────────────────────┘
               │
       ────────┼────────┬─────────┬──────────┐
       │       │        │         │          │
       ▼       ▼        ▼         ▼          ▼
   ┌──────┬──────┬─────────┬────────┐
   │Drive │Docs  │Sheets   │Forms   │
   │API   │API   │API      │API     │
   └──────┴──────┴─────────┴────────┘
            │
            ▼
   ┌─────────────────────────┐
   │  Google Workspace       │
   │  (Service Account Auth) │
   └─────────────────────────┘
```

## Authentication Requirements

The Google Workspace tool uses **service account authentication** with JSON Web Token (JWT) credentials.

### Setup Requirements

1. **Google Cloud Project** with enabled APIs:
   - Google Drive API
   - Google Docs API
   - Google Sheets API
   - Google Forms API

2. **Service Account** with appropriate permissions:
   - Domain-wide delegation (optional, for organization-wide access)
   - Viewer/Reader permissions on target documents/folders

3. **Service Account Credentials JSON** containing:
   - Private key for JWT signing
   - Service account email
   - Project ID and client information

### Authentication Configuration

**Method 1: Environment Variable (Recommended)**
```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
grctool tool google-workspace --document-id <ID> --document-type drive
```

**Method 2: Explicit Path Parameter**
```bash
grctool tool google-workspace \
  --document-id <ID> \
  --document-type docs \
  --credentials-path /secure/google-creds.json
```

**Method 3: Default Locations**
The tool automatically searches these locations if no explicit path is provided:
1. `./google-credentials.json` (current directory)
2. `./service-account.json` (current directory)
3. `$HOME/.config/gcloud/application_default_credentials.json`
4. `$GOOGLE_APPLICATION_CREDENTIALS` (environment variable)

### Required API Scopes

The service account requires the following read-only OAuth scopes:

```
https://www.googleapis.com/auth/drive.readonly
https://www.googleapis.com/auth/documents.readonly
https://www.googleapis.com/auth/spreadsheets.readonly
https://www.googleapis.com/auth/forms.responses.readonly
```

## Tool: Google Workspace (`google-workspace`)

### Purpose
Extract evidence from Google Workspace documents including policies, training materials, security questionnaires, and access control documentation stored in Drive, Docs, Sheets, and Forms.

### Implementation Status
✅ **COMPLETE & REGISTERED** in `internal/tools/google_workspace.go`

### Claude Tool Definition

```json
{
  "name": "google-workspace",
  "description": "Extract evidence from Google Workspace documents including Drive, Docs, Sheets, and Forms",
  "input_schema": {
    "type": "object",
    "properties": {
      "document_id": {
        "type": "string",
        "description": "Google document ID (from URL or share link)"
      },
      "document_type": {
        "type": "string",
        "description": "Type of Google document: drive, docs, sheets, forms",
        "enum": ["drive", "docs", "sheets", "forms"],
        "default": "drive"
      },
      "extraction_rules": {
        "type": "object",
        "description": "Rules for extracting content",
        "properties": {
          "include_metadata": {
            "type": "boolean",
            "description": "Include document metadata (created, modified, editors)",
            "default": true
          },
          "include_revisions": {
            "type": "boolean",
            "description": "Include revision history",
            "default": false
          },
          "sheet_range": {
            "type": "string",
            "description": "For sheets: range to extract (e.g., 'A1:D10', 'Sheet1!A:Z')"
          },
          "search_query": {
            "type": "string",
            "description": "Search query for Drive folder content"
          },
          "max_results": {
            "type": "integer",
            "description": "Maximum number of results to return",
            "minimum": 1,
            "maximum": 100,
            "default": 20
          }
        }
      },
      "credentials_path": {
        "type": "string",
        "description": "Path to Google service account credentials JSON file"
      }
    },
    "required": ["document_id"]
  }
}
```

### CLI Usage

```bash
# Extract from Google Drive folder
grctool tool google-workspace \
  --document-id 1A2B3C4D5E6F7G8H9I0J \
  --document-type drive

# Extract from Google Docs
grctool tool google-workspace \
  --document-id 1K2L3M4N5O6P7Q8R9S0T \
  --document-type docs \
  --include-metadata

# Extract from Google Sheets with specific range
grctool tool google-workspace \
  --document-id 1U2V3W4X5Y6Z7A8B9C0D \
  --document-type sheets \
  --sheet-range "Sheet1!A1:D50"

# Extract from Google Forms
grctool tool google-workspace \
  --document-id 1E2F3G4H5I6J7K8L9M0N \
  --document-type forms

# With explicit credentials path
grctool tool google-workspace \
  --document-id 1O2P3Q4R5S6T7U8V9W0X \
  --document-type drive \
  --credentials-path /secure/gcp-service-account.json
```

## Capabilities by Document Type

### 1. Google Drive (`drive`)

Extract folder contents, file metadata, and perform searches within folders.

**Key Capabilities:**
- List all files and subfolders
- Extract file metadata (name, type, size, dates, owner)
- Search within folder contents
- Retrieve sharing/permission information
- Track recent modifications

**Example Request:**
```json
{
  "document_id": "1A2B3C4D5E6F7G8H9I0J",
  "document_type": "drive",
  "extraction_rules": {
    "include_metadata": true,
    "search_query": "policy OR procedure",
    "max_results": 50
  }
}
```

**Example Output:**
```json
{
  "tool": "google-workspace",
  "document_type": "drive",
  "document_id": "1A2B3C4D5E6F7G8H9I0J",
  "folder_name": "Security Policies",
  "extracted_at": "2025-10-23T14:30:00Z",
  "files": [
    {
      "id": "1B2C3D4E5F6G7H8I9J0K",
      "name": "Information Security Policy.docx",
      "mime_type": "application/vnd.google-apps.document",
      "created_time": "2024-03-15T10:20:00Z",
      "modified_time": "2025-09-10T14:15:00Z",
      "owner": "admin@example.com",
      "size_bytes": 45678,
      "web_view_link": "https://docs.google.com/document/d/1B2C3D4E5F6G7H8I9J0K/view"
    },
    {
      "id": "1C2D3E4F5G6H7I8J9K0L",
      "name": "Access Control Policy.docx",
      "mime_type": "application/vnd.google-apps.document",
      "created_time": "2024-04-20T11:30:00Z",
      "modified_time": "2025-08-22T09:45:00Z",
      "owner": "compliance@example.com",
      "size_bytes": 38245,
      "web_view_link": "https://docs.google.com/document/d/1C2D3E4F5G6H7I8J9K0L/view"
    }
  ],
  "total_files": 2,
  "metadata": {
    "folder_owner": "admin@example.com",
    "folder_created": "2024-01-10T08:00:00Z",
    "folder_modified": "2025-09-10T14:15:00Z"
  }
}
```

**Use Cases:**
- Policy document inventory
- Training materials catalog
- Document version tracking
- Evidence of document management

### 2. Google Docs (`docs`)

Extract text content, structure, and formatting from Google Documents.

**Key Capabilities:**
- Extract full document text content
- Preserve document structure (headings, paragraphs, lists)
- Extract tables and formatted content
- Document metadata (title, author, dates)
- Revision history (optional)

**Example Request:**
```json
{
  "document_id": "1K2L3M4N5O6P7Q8R9S0T",
  "document_type": "docs",
  "extraction_rules": {
    "include_metadata": true,
    "include_revisions": false
  }
}
```

**Example Output:**
```json
{
  "tool": "google-workspace",
  "document_type": "docs",
  "document_id": "1K2L3M4N5O6P7Q8R9S0T",
  "title": "Information Security Policy",
  "extracted_at": "2025-10-23T14:30:00Z",
  "content": {
    "text": "# Information Security Policy\n\n## 1. Purpose\nThis policy establishes...\n\n## 2. Scope\nThis policy applies to...",
    "word_count": 2345,
    "character_count": 14567
  },
  "metadata": {
    "title": "Information Security Policy",
    "created_time": "2024-03-15T10:20:00Z",
    "modified_time": "2025-09-10T14:15:00Z",
    "last_modifying_user": "compliance@example.com",
    "document_id": "1K2L3M4N5O6P7Q8R9S0T",
    "mime_type": "application/vnd.google-apps.document",
    "web_view_link": "https://docs.google.com/document/d/1K2L3M4N5O6P7Q8R9S0T/view"
  }
}
```

**Use Cases:**
- Policy content extraction
- Procedure documentation
- Security standards compliance verification
- Document attestation

### 3. Google Sheets (`sheets`)

Extract structured data, formulas, and formatting from Google Spreadsheets.

**Key Capabilities:**
- Extract cell values and formulas
- Range-based extraction (specific sheets/cells)
- Structured data retrieval
- Column mapping for evidence validation
- Metadata and sheet information

**Example Request:**
```json
{
  "document_id": "1U2V3W4X5Y6Z7A8B9C0D",
  "document_type": "sheets",
  "extraction_rules": {
    "sheet_range": "Sheet1!A1:E100",
    "include_metadata": true
  }
}
```

**Example Output:**
```json
{
  "tool": "google-workspace",
  "document_type": "sheets",
  "document_id": "1U2V3W4X5Y6Z7A8B9C0D",
  "title": "Employee Access Review",
  "extracted_at": "2025-10-23T14:30:00Z",
  "sheets": [
    {
      "sheet_name": "Q3 2025 Review",
      "range": "Sheet1!A1:E100",
      "headers": ["Employee", "System", "Access Level", "Review Date", "Approved By"],
      "data": [
        ["John Doe", "Production DB", "Read-Only", "2025-09-15", "Jane Manager"],
        ["Jane Smith", "Admin Panel", "Full Access", "2025-09-20", "Bob Director"],
        ["Bob Jones", "S3 Buckets", "Read-Write", "2025-09-22", "Jane Manager"]
      ],
      "row_count": 3,
      "column_count": 5
    }
  ],
  "metadata": {
    "title": "Employee Access Review",
    "created_time": "2025-07-01T09:00:00Z",
    "modified_time": "2025-09-22T16:30:00Z",
    "last_modifying_user": "auditor@example.com",
    "web_view_link": "https://docs.google.com/spreadsheets/d/1U2V3W4X5Y6Z7A8B9C0D/view"
  }
}
```

**Use Cases:**
- Access review documentation
- Training completion tracking
- Asset inventories
- Risk assessments
- Security questionnaires

### 4. Google Forms (`forms`)

Extract form structure, questions, and response data from Google Forms.

**Key Capabilities:**
- Extract form questions and structure
- Retrieve form responses
- Question type identification
- Response validation and completeness
- Form configuration and settings

**Example Request:**
```json
{
  "document_id": "1E2F3G4H5I6J7K8L9M0N",
  "document_type": "forms",
  "extraction_rules": {
    "include_metadata": true
  }
}
```

**Example Output:**
```json
{
  "tool": "google-workspace",
  "document_type": "forms",
  "document_id": "1E2F3G4H5I6J7K8L9M0N",
  "title": "Security Awareness Training Quiz",
  "extracted_at": "2025-10-23T14:30:00Z",
  "form_structure": {
    "title": "Security Awareness Training Quiz",
    "description": "Complete this quiz after watching the security training video",
    "questions": [
      {
        "question_id": "q1",
        "question_text": "What is the purpose of multi-factor authentication?",
        "question_type": "MULTIPLE_CHOICE",
        "required": true,
        "options": ["Add convenience", "Increase security", "Both"]
      },
      {
        "question_id": "q2",
        "question_text": "List three indicators of a phishing email",
        "question_type": "PARAGRAPH_TEXT",
        "required": true
      }
    ],
    "total_questions": 2
  },
  "responses": {
    "response_count": 47,
    "completion_rate": "94%",
    "sample_responses": [
      {
        "response_id": "r1",
        "timestamp": "2025-09-15T10:30:00Z",
        "answers": {
          "q1": "Increase security",
          "q2": "Suspicious sender, urgent language, unexpected attachments"
        }
      }
    ]
  },
  "metadata": {
    "title": "Security Awareness Training Quiz",
    "created_time": "2025-01-15T08:00:00Z",
    "modified_time": "2025-09-10T12:00:00Z",
    "owner": "training@example.com",
    "web_view_link": "https://docs.google.com/forms/d/1E2F3G4H5I6J7K8L9M0N/view"
  }
}
```

**Use Cases:**
- Training completion evidence
- Security questionnaire responses
- Incident report forms
- Vendor assessment forms
- Employee acknowledgments

## Evidence Mapping System

The Google Workspace tool supports YAML-based evidence mapping configuration for automatically associating documents with evidence tasks.

### Configuration File: `google_evidence_mappings.yaml`

Located in:
- `{DataDir}/google_evidence_mappings.yaml`
- `./google_evidence_mappings.yaml`
- `./configs/google_evidence_mappings.yaml`

### Mapping Configuration Structure

```yaml
evidence_mappings:
  - task_ref: ET-054
    task_name: "Security Policy Documentation"
    frameworks:
      - soc2
      - iso27001
    priority: high
    document_sources:
      - document_id: "1K2L3M4N5O6P7Q8R9S0T"
        document_type: docs
        document_name: "Information Security Policy"
        validation_rules:
          min_content_length: 1000
          required_keywords: ["security", "policy", "compliance"]
          last_updated_within_days: 365

  - task_ref: ET-064
    task_name: "Access Review Documentation"
    frameworks:
      - soc2
    priority: high
    document_sources:
      - document_id: "1U2V3W4X5Y6Z7A8B9C0D"
        document_type: sheets
        document_name: "Quarterly Access Reviews"
        extraction_rules:
          sheet_range: "Q3 2025!A1:F500"
        validation_rules:
          min_rows: 10
          required_columns: ["Employee", "System", "Review Date", "Approved By"]
          data_freshness_days: 90

  - task_ref: ET-074
    task_name: "Security Training Completion"
    frameworks:
      - soc2
      - iso27001
    priority: medium
    document_sources:
      - document_id: "1E2F3G4H5I6J7K8L9M0N"
        document_type: forms
        document_name: "Security Awareness Quiz"
        validation_rules:
          min_response_count: 40
          required_completion_rate: 0.85
```

### Using Evidence Mappings

When mappings are configured, the tool can automatically:
1. Load task-specific document IDs
2. Apply extraction rules from configuration
3. Validate extracted data against requirements
4. Generate compliance-ready evidence packages

## Output Format

All Google Workspace tool outputs follow a standardized JSON structure:

```json
{
  "tool": "google-workspace",
  "document_type": "drive|docs|sheets|forms",
  "document_id": "string",
  "title": "string",
  "extracted_at": "ISO8601 timestamp",
  "content": {
    // Type-specific content structure
  },
  "metadata": {
    "title": "string",
    "created_time": "ISO8601 timestamp",
    "modified_time": "ISO8601 timestamp",
    "last_modifying_user": "email",
    "owner": "email",
    "web_view_link": "URL",
    "mime_type": "string"
  },
  "evidence_source": {
    "type": "google_workspace",
    "document_id": "string",
    "document_type": "string",
    "extracted_at": "ISO8601 timestamp",
    "url": "string"
  }
}
```

## Security Considerations

### Credential Security

1. **Never commit credentials** to version control
2. **Use restrictive file permissions**: `chmod 600 /path/to/credentials.json`
3. **Store credentials outside repository**: Use secure credential management
4. **Rotate service account keys** regularly (recommended: every 90 days)
5. **Use environment variables** for credential paths in CI/CD

### Access Control

1. **Principle of Least Privilege**: Grant service account minimal required permissions
2. **Read-Only Scopes**: Use `.readonly` scopes wherever possible
3. **Domain-Wide Delegation**: Only enable if organization-wide access is required
4. **Document Sharing**: Ensure service account has viewer access to target documents

### Data Privacy

1. **Redact Sensitive Data**: Remove PII from evidence reports if not required
2. **Audit Logging**: Enable Google Workspace audit logs for service account activity
3. **Access Reviews**: Regularly review service account permissions
4. **Secure Storage**: Encrypt evidence files at rest

## Performance Considerations

### Caching Strategy

The tool implements a 5-minute TTL cache for:
- Document metadata
- Folder listings
- Sheet data (static ranges)
- Form structures

### Rate Limiting

Google API quotas:
- **Drive API**: 20,000 requests/100 seconds
- **Docs API**: 500 requests/100 seconds
- **Sheets API**: 500 requests/100 seconds per user
- **Forms API**: 300 requests/minute

**Best Practices:**
- Batch document requests where possible
- Use incremental evidence collection
- Implement exponential backoff for retries
- Monitor quota usage in GCP console

### Large Document Handling

- **Documents**: Paginate large docs (>10MB)
- **Sheets**: Extract specific ranges instead of entire spreadsheet
- **Drive Folders**: Use `max_results` parameter to limit items
- **Forms**: Limit response count for large surveys

## Error Handling

### Common Errors

**Authentication Errors:**
```json
{
  "error": "failed to initialize Google client: credentials not found",
  "solution": "Set GOOGLE_APPLICATION_CREDENTIALS or use --credentials-path flag"
}
```

**Permission Errors:**
```json
{
  "error": "failed to access document: 403 Forbidden",
  "solution": "Ensure service account has viewer/reader permission on the document"
}
```

**Document Not Found:**
```json
{
  "error": "failed to extract from drive: document not found (404)",
  "solution": "Verify document ID and ensure it's shared with the service account"
}
```

**API Quota Exceeded:**
```json
{
  "error": "rate limit exceeded for Google Sheets API",
  "solution": "Implement retry with exponential backoff or reduce request frequency"
}
```

## Testing

### Unit Tests
Located in `internal/tools/google_workspace_unit_test.go`
- Validation logic testing
- Parameter parsing tests
- Error handling tests

### Integration Tests
Located in multiple test files with `//go:build e2e` tag:
- `google_workspace_test.go` - End-to-end API tests
- `google_workspace_comprehensive_test.go` - Full coverage tests
- `google_workspace_edge_cases_test.go` - Edge cases and error conditions
- `google_workspace_performance_test.go` - Performance and concurrency tests

### Running Tests

```bash
# Unit tests only (no API calls)
go test ./internal/tools -run TestGoogleWorkspace -short

# Integration tests (requires credentials)
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/test-credentials.json
go test ./internal/tools -tags=e2e -run TestGoogleWorkspace

# Performance tests
go test ./internal/tools -tags=e2e -run TestGoogleWorkspacePerformance -v
```

## Compliance Framework Mappings

### SOC 2 Trust Service Criteria

**CC1.4 - Commitment to Competence**
- Training completion tracking (Forms)
- Training materials documentation (Drive/Docs)

**CC3.1 - Risk Assessment Process**
- Risk assessment documentation (Sheets)
- Risk registers and tracking (Sheets)

**CC6.1 - Logical Access Controls**
- Access review documentation (Sheets)
- Access request forms (Forms)

**CC6.8 - Data Loss Prevention**
- Data classification policies (Docs)
- DLP policy documentation (Drive/Docs)

**CC7.2 - System Monitoring**
- Incident response procedures (Docs)
- Security monitoring procedures (Drive/Docs)

### ISO 27001 Annex A Controls

**A.5.1 - Policies for Information Security**
- Security policy documentation (Docs)
- Policy management system (Drive)

**A.6.1 - Internal Organization**
- Role definitions and responsibilities (Docs/Sheets)
- Organizational structure documentation (Drive)

**A.7.2 - Awareness, Education, and Training**
- Training materials (Drive/Docs)
- Training completion records (Forms/Sheets)

**A.9.2 - User Access Management**
- Access review documentation (Sheets)
- Access request workflows (Forms)

## References

- [[google-workspace-setup]] - Setup and authentication guide
- [[data-formats]] - Evidence data schemas
- [[naming-conventions]] - File naming standards
- [[evidence-collection-workflow]] - Evidence workflow documentation

---

*This specification documents the Google Workspace integration as implemented in GRCTool v1.0. For setup instructions, see the [Google Workspace Setup Guide](../01-User-Guide/google-workspace-setup.md).*
