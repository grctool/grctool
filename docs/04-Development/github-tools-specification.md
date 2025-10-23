# GitHub Analysis Tools - Technical Specification

**Version:** 1.0
**Date:** 2025-01-10
**Purpose:** Evidence collection for GitHub repository security, change management, and access controls
**Tool Count:** 6 tools

## Overview

This specification defines the GitHub analysis toolchain for automated compliance evidence collection. These tools analyze GitHub repositories, workflows, permissions, and security features to extract evidence for SOC2/ISO27001 compliance requirements related to change management, access control, and security scanning.

## Tool Suite Architecture

```
┌─────────────────────────────────────────────────────────┐
│    Evidence Collection Request (ET-047, ET-071, etc.)   │
└─────────────────┬───────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────┐
│              Tool Selection Layer (Claude)               │
│   Selects appropriate tools based on task requirements   │
└─────────────┬───────────────────────────────────────────┘
              │
              ├──────┬──────┬──────┬──────┬──────┐
              ▼      ▼      ▼      ▼      ▼      ▼
         ┌────────┬────────┬────────┬────────┬────────┐
         │Searcher│Perms   │Deploy  │Security│Workflow│
         │        │        │        │Features│Analyzer│
         └────────┴────────┴────────┴────────┴────────┘
                           │
                           ▼
              ┌─────────────────────────┐
              │   GitHub REST API v3    │
              │  (Authenticated Access) │
              └─────────────────────────┘
```

## Authentication Requirements

All GitHub tools require authentication via `GITHUB_TOKEN` environment variable:

```yaml
# .grctool.yaml
github:
  token: "${GITHUB_TOKEN}"  # Personal access token or GitHub App token
  cache_enabled: true
  cache_ttl: "1h"
```

**Required Token Permissions:**
- `repo` - Full control of private repositories
- `read:org` - Read organization membership (for org-level analysis)
- `workflow` - Update GitHub Action workflows (for workflow analysis)

## Tool 1: GitHub Searcher (`github-searcher`)

### Purpose
Search GitHub repositories for security-related evidence including commits, workflows, issues, and pull requests. Provides filtered search with date ranges and result limiting.

### Implementation Status
✅ **COMPLETE & REGISTERED** in `internal/tools/github_searcher.go`

### Claude Tool Definition

```json
{
  "name": "github-searcher",
  "description": "Search GitHub repositories for security evidence including commits, workflows, issues, and PRs with date filtering and caching.",
  "input_schema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Search query for GitHub content (uses GitHub search syntax)"
      },
      "search_type": {
        "type": "string",
        "description": "Type of search to perform",
        "enum": ["commit", "workflow", "issue", "pr", "all"],
        "default": "all"
      },
      "since": {
        "type": "string",
        "description": "Date filter - only results since this date (YYYY-MM-DD format)"
      },
      "limit": {
        "type": "integer",
        "description": "Maximum number of results to return",
        "default": 50
      },
      "labels": {
        "type": "array",
        "description": "Filter by GitHub labels (for issues and PRs)",
        "items": {"type": "string"}
      },
      "use_cache": {
        "type": "boolean",
        "description": "Use cached API results when available",
        "default": true
      }
    },
    "required": ["query"]
  }
}
```

### Capabilities

#### 1. Commit Search
Search commit history for security-related changes:

**Example Request:**
```json
{
  "query": "security OR vulnerability OR CVE",
  "search_type": "commit",
  "since": "2024-01-01",
  "limit": 100
}
```

**Example Output:**
```json
{
  "search_type": "commit",
  "total_results": 47,
  "results": [
    {
      "sha": "abc123def456",
      "message": "fix: address security vulnerability in authentication",
      "author": "developer@example.com",
      "date": "2024-03-15T14:30:00Z",
      "url": "https://github.com/org/repo/commit/abc123def456",
      "files_changed": 3,
      "additions": 15,
      "deletions": 8,
      "excerpt": "...updated authentication middleware to prevent..."
    }
  ]
}
```

**Use Cases:**
- Track security fixes and patches
- Document vulnerability remediation
- Evidence of timely security updates

#### 2. Workflow Search
Find GitHub Actions workflows related to security:

**Example Request:**
```json
{
  "query": "security scan OR code analysis OR SAST",
  "search_type": "workflow",
  "limit": 50
}
```

**Use Cases:**
- Document security scanning implementations
- Verify CI/CD security controls
- Evidence of automated security testing

#### 3. Issue and PR Search
Search for security-related issues and pull requests:

**Example Request:**
```json
{
  "query": "security",
  "search_type": "issue",
  "labels": ["security", "vulnerability"],
  "since": "2024-01-01"
}
```

**Use Cases:**
- Track security issue resolution
- Document security review process
- Evidence of security incident handling

### SOC2 Control Mapping

| Control | Evidence Use Case |
|---------|------------------|
| CC7.2 | Commit history shows security monitoring and updates |
| CC8.1 | Issue tracking shows change management process |
| CC7.4 | Security incidents documented and tracked |

---

## Tool 2: GitHub Permissions Analyzer (`github-permissions`)

### Purpose
Extract comprehensive repository access controls and permissions for SOC2 audit evidence. Analyzes collaborators, team access, branch protection, and deployment environment controls.

### Implementation Status
✅ **COMPLETE & REGISTERED** in `internal/tools/github_permissions.go`

### Claude Tool Definition

```json
{
  "name": "github-permissions",
  "description": "Extract comprehensive repository access controls including collaborators, teams, branch protection, deployment rules, and organization members.",
  "input_schema": {
    "type": "object",
    "properties": {
      "repository": {
        "type": "string",
        "description": "Repository in format 'owner/repo' (e.g., 'octocat/Hello-World')"
      },
      "output_format": {
        "type": "string",
        "description": "Output format for permissions data",
        "enum": ["detailed", "matrix", "summary"],
        "default": "detailed"
      },
      "include_org_members": {
        "type": "boolean",
        "description": "Include organization member information if available",
        "default": true
      },
      "use_cache": {
        "type": "boolean",
        "description": "Use cached API results when available",
        "default": true
      }
    },
    "required": ["repository"]
  }
}
```

### Capabilities

#### 1. Direct Collaborator Analysis
List all repository collaborators with permission levels:

**Example Output (Detailed Format):**
```json
{
  "repository": "organization/application",
  "collaborators": [
    {
      "login": "developer1",
      "name": "John Developer",
      "permission": "write",
      "role_name": "Developer",
      "added_date": "2023-06-15T10:00:00Z",
      "type": "direct"
    },
    {
      "login": "admin1",
      "name": "Jane Admin",
      "permission": "admin",
      "role_name": "Repository Admin",
      "added_date": "2023-01-10T08:00:00Z",
      "type": "direct"
    }
  ]
}
```

#### 2. Team-Based Access
Analyze team permissions with member lists:

**Example Output:**
```json
{
  "teams": [
    {
      "name": "backend-team",
      "permission": "write",
      "members": [
        {"login": "dev1", "role": "member"},
        {"login": "dev2", "role": "member"}
      ],
      "member_count": 2
    },
    {
      "name": "security-team",
      "permission": "admin",
      "members": [
        {"login": "security1", "role": "maintainer"}
      ],
      "member_count": 1
    }
  ]
}
```

#### 3. Branch Protection Rules
Extract branch protection configurations:

**Example Output:**
```json
{
  "branch_protection": [
    {
      "branch": "main",
      "protection_rules": {
        "required_reviews": {
          "required_approving_review_count": 2,
          "dismiss_stale_reviews": true,
          "require_code_owner_reviews": true
        },
        "required_status_checks": {
          "strict": true,
          "contexts": ["security-scan", "unit-tests"]
        },
        "enforce_admins": true,
        "restrictions": {
          "users": [],
          "teams": ["senior-developers"],
          "apps": []
        }
      }
    }
  ]
}
```

#### 4. Permission Matrix (Matrix Format)
Consolidated view of all access:

**Example Output (Matrix Format):**
```markdown
# Repository Access Control Matrix
**Repository:** organization/application
**Generated:** 2025-01-10T15:30:00Z

## User Permissions

| User | Direct Access | Team Access | Effective Permission | Branch Write | Admin |
|------|--------------|-------------|---------------------|--------------|-------|
| developer1 | write | backend-team (write) | write | ✅ | ❌ |
| admin1 | admin | security-team (admin) | admin | ✅ | ✅ |
| reviewer1 | - | review-team (triage) | triage | ❌ | ❌ |

## Branch Protection Summary

| Branch | Required Reviews | Status Checks | Admin Enforcement |
|--------|-----------------|---------------|-------------------|
| main | 2 reviewers | ✅ security-scan, unit-tests | ✅ Enabled |
| develop | 1 reviewer | ✅ unit-tests | ❌ Not enforced |
```

### SOC2 Control Mapping

| Control | Evidence Use Case |
|---------|------------------|
| CC6.1 | Logical access controls - repository permissions |
| CC6.2 | User authentication and authorization |
| CC6.3 | Least privilege access - permission levels |
| CC8.1 | Change management - branch protection rules |

### Evidence Task Mapping

| Evidence Task | Description | Tool Capability |
|--------------|-------------|-----------------|
| ET-047 | Repository Access Controls | Direct collaborators, team access, permission matrix |
| ET-071 | Change Management Controls | Branch protection, required reviews |

---

## Tool 3: GitHub Deployment Access Analyzer (`github-deployment-access`)

### Purpose
Extract deployment environment access controls and protection rules for SOC2 audit evidence. Analyzes environment-specific protections, required reviewers, and deployment authorization.

### Implementation Status
✅ **COMPLETE & REGISTERED** in `internal/tools/github_deployment_access.go`

### Claude Tool Definition

```json
{
  "name": "github-deployment-access",
  "description": "Extract deployment environment protection rules including required reviewers, wait timers, and branch restrictions for SOC2 deployment control evidence.",
  "input_schema": {
    "type": "object",
    "properties": {
      "repository": {
        "type": "string",
        "description": "Repository in format 'owner/repo'"
      },
      "environment": {
        "type": "string",
        "description": "Specific environment name to analyze (optional - empty for all)"
      },
      "output_format": {
        "type": "string",
        "enum": ["detailed", "matrix", "summary"],
        "default": "detailed"
      },
      "include_branch_rules": {
        "type": "boolean",
        "description": "Include branch protection rules affecting deployments",
        "default": true
      }
    },
    "required": ["repository"]
  }
}
```

### Capabilities

#### 1. Environment Protection Rules
Analyze deployment environment configurations:

**Example Output:**
```json
{
  "environments": [
    {
      "name": "production",
      "protection_rules": [
        {
          "type": "required_reviewers",
          "reviewers": {
            "users": [{"login": "deployer1"}],
            "teams": [{"slug": "platform-team"}]
          }
        },
        {
          "type": "wait_timer",
          "wait_timer": 300
        },
        {
          "type": "branch_policy",
          "deployment_branch_policy": {
            "protected_branches": true,
            "custom_branch_policies": false
          }
        }
      ],
      "deployment_count_last_30d": 47
    },
    {
      "name": "staging",
      "protection_rules": [
        {
          "type": "required_reviewers",
          "reviewers": {
            "teams": [{"slug": "backend-team"}]
          }
        }
      ],
      "deployment_count_last_30d": 156
    }
  ]
}
```

#### 2. Deployment Authorization Matrix
Who can deploy to which environments:

**Example Output (Matrix Format):**
```markdown
# Deployment Authorization Matrix
**Repository:** organization/application

| Environment | Required Reviewers | Wait Timer | Deployment Branch | Protected |
|-------------|-------------------|------------|-------------------|-----------|
| production | platform-team (3 members) | 5 minutes | main only | ✅ Yes |
| staging | backend-team (8 members) | None | develop, main | ✅ Yes |
| development | None | None | Any | ❌ No |

## Authorized Deployers

### Production Environment
- **Teams:** platform-team
  - deployer1 (maintainer)
  - deployer2 (member)
  - deployer3 (member)
- **Wait Timer:** 5 minutes (prevents accidental deployments)
- **Branch Restriction:** Only `main` branch

### Staging Environment
- **Teams:** backend-team
  - 8 members with deployment access
- **Branch Restriction:** `develop` or `main` branches
```

#### 3. Environment Protection Coverage
Analyze which environments have adequate protection:

**Example Output (Summary Format):**
```json
{
  "total_environments": 3,
  "protected_environments": 2,
  "unprotected_environments": 1,
  "protection_summary": {
    "production": {
      "status": "fully_protected",
      "controls": ["required_reviewers", "wait_timer", "branch_policy"],
      "compliance": "SOC2_COMPLIANT"
    },
    "staging": {
      "status": "partially_protected",
      "controls": ["required_reviewers"],
      "missing_controls": ["wait_timer"],
      "compliance": "REVIEW_RECOMMENDED"
    },
    "development": {
      "status": "unprotected",
      "controls": [],
      "compliance": "ACCEPTABLE_FOR_DEV"
    }
  }
}
```

### SOC2 Control Mapping

| Control | Evidence Use Case |
|---------|------------------|
| CC8.1 | Change management - deployment approval process |
| CC7.2 | System monitoring - deployment controls |
| CC6.1 | Access controls - environment-specific authorization |

### Evidence Task Mapping

| Evidence Task | Description | Tool Capability |
|--------------|-------------|-----------------|
| ET-071 | Deployment Controls | Environment protection rules, required reviewers |
| ET-047 | Access Controls | Deployment authorization matrix |

---

## Tool 4: GitHub Security Features Analyzer (`github-security-features`)

### Purpose
Extract repository security feature configuration for SOC2 audit evidence. Analyzes vulnerability alerts, secret scanning, code scanning, and security policy presence.

### Implementation Status
✅ **COMPLETE & REGISTERED** in `internal/tools/github_security_features.go`

### Claude Tool Definition

```json
{
  "name": "github-security-features",
  "description": "Extract repository security feature configuration including vulnerability alerts, secret scanning, code scanning, and security policies.",
  "input_schema": {
    "type": "object",
    "properties": {
      "repository": {
        "type": "string",
        "description": "Repository in format 'owner/repo'"
      },
      "output_format": {
        "type": "string",
        "enum": ["detailed", "matrix", "summary"],
        "default": "detailed"
      },
      "include_policy_analysis": {
        "type": "boolean",
        "description": "Include security policy analysis (SECURITY.md)",
        "default": true
      },
      "include_compliance_mapping": {
        "type": "boolean",
        "description": "Include SOC2/compliance framework mapping",
        "default": false
      }
    },
    "required": ["repository"]
  }
}
```

### Capabilities

#### 1. Security Feature Enablement
Check which security features are enabled:

**Example Output:**
```json
{
  "repository": "organization/application",
  "security_features": {
    "vulnerability_alerts": {
      "enabled": true,
      "last_updated": "2024-12-15T10:00:00Z"
    },
    "automated_security_fixes": {
      "enabled": true,
      "dependabot_enabled": true
    },
    "secret_scanning": {
      "enabled": true,
      "status": "active",
      "alerts_count": 0
    },
    "code_scanning": {
      "enabled": true,
      "tools": ["CodeQL", "Semgrep"],
      "last_analysis": "2025-01-10T08:00:00Z",
      "alerts_count": 3
    },
    "dependency_graph": {
      "enabled": true
    },
    "security_policy": {
      "exists": true,
      "file_path": "SECURITY.md",
      "last_updated": "2024-06-01T00:00:00Z"
    }
  }
}
```

#### 2. Security Feature Matrix
Visual overview of security posture:

**Example Output (Matrix Format):**
```markdown
# Security Feature Enablement Matrix
**Repository:** organization/application
**Analysis Date:** 2025-01-10

| Feature | Status | Configuration | SOC2 Control |
|---------|--------|---------------|--------------|
| Vulnerability Alerts | ✅ Enabled | Auto-updates: Yes | CC7.1 |
| Dependabot | ✅ Enabled | Auto-merge: Safe updates | CC7.1 |
| Secret Scanning | ✅ Enabled | Push protection: Yes | CC6.8 |
| Code Scanning | ✅ Enabled | CodeQL, Semgrep | CC7.1 |
| Security Policy | ✅ Present | SECURITY.md | CC7.4 |
| Branch Protection | ✅ Enabled | 2 required reviews | CC8.1 |

## Security Alerts Summary
- **Code Scanning:** 3 alerts (2 medium, 1 low)
- **Secret Scanning:** 0 alerts
- **Dependabot:** 12 open PRs (8 low priority)

## Compliance Status
✅ **SOC2 COMPLIANT** - All recommended security features enabled
```

#### 3. Security Policy Analysis
Extract and analyze SECURITY.md content:

**Example Output:**
```json
{
  "security_policy": {
    "exists": true,
    "file_path": "SECURITY.md",
    "sections": {
      "reporting_vulnerabilities": true,
      "supported_versions": true,
      "disclosure_policy": true,
      "contact_information": true
    },
    "contact_email": "security@example.com",
    "response_time_sla": "48 hours",
    "content_excerpt": "## Reporting Security Issues\n\nPlease report security vulnerabilities to security@example.com..."
  }
}
```

### SOC2 Control Mapping

| Control | Evidence Use Case |
|---------|------------------|
| CC7.1 | Security monitoring - automated vulnerability detection |
| CC7.4 | Security incident response - security policy presence |
| CC6.8 | Data security - secret scanning |
| CC8.1 | Change management - code scanning in CI/CD |

---

## Tool 5: GitHub Workflow Analyzer (`github-workflow-analyzer`)

### Purpose
Analyze GitHub Actions workflows for comprehensive SOC2 evidence of CI/CD security, security scanning workflows, deployment controls, and compliance with security requirements.

### Implementation Status
✅ **COMPLETE & REGISTERED** in `internal/tools/github_workflows.go`

### Claude Tool Definition

```json
{
  "name": "github-workflow-analyzer",
  "description": "Analyze GitHub Actions workflows for CI/CD security evidence including security scanning, deployment controls, and approval requirements.",
  "input_schema": {
    "type": "object",
    "properties": {
      "analysis_type": {
        "type": "string",
        "description": "Type of workflow analysis",
        "enum": ["security", "deployment", "approval", "full"],
        "default": "full"
      },
      "filter_workflows": {
        "type": "array",
        "description": "Filter workflows by name patterns (e.g., '*security*', '*deploy*')",
        "items": {"type": "string"}
      },
      "check_branch_protection": {
        "type": "boolean",
        "description": "Check branch protection rules and approval requirements",
        "default": true
      },
      "include_content": {
        "type": "boolean",
        "description": "Include full workflow file content in results",
        "default": false
      },
      "use_cache": {
        "type": "boolean",
        "description": "Use cached results when available",
        "default": true
      }
    },
    "required": []
  }
}
```

### Capabilities

#### 1. Security Scanning Workflow Detection
Identify security scanning implementations:

**Example Output:**
```json
{
  "security_workflows": [
    {
      "name": "CodeQL Security Analysis",
      "file_path": ".github/workflows/codeql-analysis.yml",
      "triggers": ["push", "pull_request", "schedule"],
      "schedule": "0 0 * * 1",
      "security_tools": ["CodeQL"],
      "languages": ["javascript", "python", "go"],
      "runs_on_branches": ["main", "develop"],
      "last_run": "2025-01-10T06:00:00Z",
      "status": "success"
    },
    {
      "name": "Dependency Scanning",
      "file_path": ".github/workflows/dependency-check.yml",
      "triggers": ["pull_request"],
      "security_tools": ["npm audit", "go mod"],
      "last_run": "2025-01-10T14:00:00Z",
      "status": "success"
    }
  ]
}
```

#### 2. Deployment Workflow Analysis
Analyze deployment processes and controls:

**Example Output:**
```json
{
  "deployment_workflows": [
    {
      "name": "Production Deploy",
      "file_path": ".github/workflows/deploy-production.yml",
      "environment": "production",
      "triggers": ["workflow_dispatch"],
      "manual_approval_required": true,
      "required_reviewers": ["platform-team"],
      "deployment_steps": [
        "security-scan",
        "build",
        "integration-tests",
        "deploy",
        "smoke-tests"
      ],
      "secrets_used": [
        "AWS_ACCESS_KEY_ID",
        "AWS_SECRET_ACCESS_KEY",
        "DEPLOY_TOKEN"
      ],
      "deployment_frequency_30d": 23
    }
  ]
}
```

#### 3. Required Status Checks
Map workflows to branch protection requirements:

**Example Output:**
```markdown
# Workflow Compliance Matrix
**Repository:** organization/application

| Workflow | Purpose | Branch Protection | Required for Merge | Last Run |
|----------|---------|-------------------|-------------------|----------|
| security-scan | CodeQL analysis | ✅ Required on main | Yes | Success (1h ago) |
| unit-tests | Test coverage | ✅ Required on main | Yes | Success (30m ago) |
| lint | Code quality | ✅ Required on main | Yes | Success (30m ago) |
| deploy-staging | Auto-deploy | ❌ Not required | No | Success (2d ago) |

## Security Scanning Coverage
- **SAST:** CodeQL (JavaScript, Python, Go)
- **Dependency Scanning:** npm audit, go mod
- **Secret Scanning:** GitHub native (enabled)
- **Container Scanning:** Trivy (on Docker builds)

## Deployment Controls
- **Production:** Manual approval required (platform-team)
- **Staging:** Automatic on develop branch merge
- **Development:** Automatic on feature branch push
```

#### 4. Workflow Security Analysis
Identify security issues in workflows:

**Example Output:**
```json
{
  "security_findings": [
    {
      "workflow": "deploy-production.yml",
      "severity": "medium",
      "issue": "Secrets used in workflow steps without environment isolation",
      "recommendation": "Use environment secrets instead of repository secrets for production deploys",
      "line": 45
    },
    {
      "workflow": "pr-checks.yml",
      "severity": "low",
      "issue": "Workflow runs on pull_request_target with code checkout",
      "recommendation": "Use pull_request trigger for untrusted code execution",
      "line": 12
    }
  ]
}
```

### SOC2 Control Mapping

| Control | Evidence Use Case |
|---------|------------------|
| CC7.1 | System security - automated security scanning |
| CC8.1 | Change management - CI/CD pipeline controls |
| CC7.2 | System monitoring - workflow execution logs |
| CC6.6 | Logical security - deployment approval requirements |

### Evidence Task Mapping

| Evidence Task | Description | Tool Capability |
|--------------|-------------|-----------------|
| ET-071 | CI/CD Security Controls | Security scanning workflows, deployment approvals |
| ET-047 | Access Controls | Deployment workflow permissions |

---

## Tool 6: GitHub Review Analyzer (`github-review-analyzer`)

### Purpose
Analyze GitHub pull request reviews for comprehensive SOC2 evidence of change management processes, code review compliance, and security-related PR handling.

### Implementation Status
✅ **COMPLETE & REGISTERED** in `internal/tools/github_reviews.go`

### Claude Tool Definition

```json
{
  "name": "github-review-analyzer",
  "description": "Analyze pull request reviews for SOC2 evidence including review compliance, security PR handling, and change management metrics.",
  "input_schema": {
    "type": "object",
    "properties": {
      "analysis_period": {
        "type": "string",
        "description": "Time period for analysis",
        "enum": ["30d", "90d", "180d", "1y"],
        "default": "90d"
      },
      "state": {
        "type": "string",
        "description": "PR state to analyze",
        "enum": ["open", "closed", "merged", "all"],
        "default": "all"
      },
      "include_security_prs": {
        "type": "boolean",
        "description": "Focus on security-related pull requests",
        "default": true
      },
      "check_compliance": {
        "type": "boolean",
        "description": "Check compliance with review policies",
        "default": true
      },
      "detailed_metrics": {
        "type": "boolean",
        "description": "Include detailed reviewer statistics and patterns",
        "default": true
      },
      "max_prs": {
        "type": "integer",
        "description": "Maximum number of PRs to analyze",
        "default": 200
      },
      "use_cache": {
        "type": "boolean",
        "description": "Use cached results when available",
        "default": true
      }
    },
    "required": []
  }
}
```

### Capabilities

#### 1. Review Compliance Analysis
Check adherence to review policies:

**Example Output:**
```json
{
  "analysis_period": "90d",
  "total_prs": 187,
  "compliance_summary": {
    "total_merged": 165,
    "reviewed_before_merge": 163,
    "compliance_rate": 0.988,
    "average_reviews_per_pr": 2.3,
    "policy_violations": 2
  },
  "review_requirements": {
    "minimum_reviews": 2,
    "require_code_owner_review": true,
    "dismiss_stale_reviews": true
  },
  "violations": [
    {
      "pr_number": 1234,
      "title": "Hotfix: Critical production bug",
      "merged_without_reviews": true,
      "reason": "Emergency deployment",
      "merged_by": "admin1"
    }
  ]
}
```

#### 2. Security PR Tracking
Analyze security-related pull requests:

**Example Output:**
```json
{
  "security_prs": {
    "total": 23,
    "by_type": {
      "vulnerability_fix": 12,
      "dependency_update": 8,
      "security_enhancement": 3
    },
    "average_time_to_review": "4.2 hours",
    "average_time_to_merge": "18.5 hours",
    "review_compliance": 1.0,
    "examples": [
      {
        "pr_number": 1456,
        "title": "Security: Update lodash to fix CVE-2024-12345",
        "created": "2025-01-05T10:00:00Z",
        "merged": "2025-01-05T16:30:00Z",
        "time_to_merge_hours": 6.5,
        "reviews": [
          {"reviewer": "security-team-member", "state": "APPROVED"}
        ],
        "labels": ["security", "dependencies"]
      }
    ]
  }
}
```

#### 3. Reviewer Statistics
Analyze review participation and quality:

**Example Output:**
```markdown
# Pull Request Review Metrics (90 days)
**Repository:** organization/application
**Period:** 2024-10-10 to 2025-01-10

## Review Compliance
- **Total PRs Merged:** 165
- **PRs with Required Reviews:** 163 (98.8%)
- **Average Reviews per PR:** 2.3
- **Compliance Rate:** ✅ 98.8% (target: 95%)

## Top Reviewers
| Reviewer | PRs Reviewed | Avg Review Time | Approval Rate |
|----------|-------------|-----------------|---------------|
| senior-dev1 | 87 | 3.2 hours | 94% |
| senior-dev2 | 76 | 4.1 hours | 91% |
| tech-lead | 45 | 2.8 hours | 98% |

## Security PR Handling
- **Security PRs:** 23
- **Average Time to Review:** 4.2 hours
- **Average Time to Merge:** 18.5 hours
- **Review Compliance:** ✅ 100%

## Change Management Evidence
- **Review Policy:** 2 required approvals, code owner review required
- **Stale Review Dismissal:** Enabled
- **Admin Override:** 2 instances (documented emergencies)
```

#### 4. Approval Timeline Analysis
Track time from PR creation to approval:

**Example Output:**
```json
{
  "timeline_metrics": {
    "median_time_to_first_review": "2.5 hours",
    "median_time_to_approval": "6.8 hours",
    "median_time_to_merge": "12.3 hours",
    "prs_reviewed_within_sla": {
      "24h": 145,
      "48h": 162,
      "72h": 165
    },
    "sla_compliance_24h": 0.878
  }
}
```

### SOC2 Control Mapping

| Control | Evidence Use Case |
|---------|------------------|
| CC8.1 | Change management - code review process |
| CC7.2 | System monitoring - review timeline tracking |
| CC7.4 | Security incidents - security PR handling |
| CC6.1 | Access controls - reviewer authorization |

### Evidence Task Mapping

| Evidence Task | Description | Tool Capability |
|--------------|-------------|-----------------|
| ET-071 | Change Management Process | Review compliance, approval workflows |
| ET-047 | Access Controls | Reviewer permissions, approval authority |

---

## Tool Selection Guide

### By Evidence Task

#### ET-047: Repository Access Controls
**Recommended Tools:**
1. **github-permissions** - Direct collaborators, team access
2. **github-deployment-access** - Environment-specific permissions
3. **github-security-features** - Security feature access controls

#### ET-071: Change Management Controls
**Recommended Tools:**
1. **github-review-analyzer** - PR review process compliance
2. **github-workflow-analyzer** - CI/CD pipeline controls
3. **github-permissions** - Branch protection rules

### By SOC2 Control

#### CC6.1: Logical Access Security
- **github-permissions**: Repository access matrix
- **github-deployment-access**: Deployment authorization

#### CC7.1: System Security Monitoring
- **github-workflow-analyzer**: Security scanning workflows
- **github-security-features**: Automated vulnerability detection

#### CC8.1: Change Management
- **github-review-analyzer**: Code review compliance
- **github-workflow-analyzer**: Deployment approval workflows
- **github-permissions**: Branch protection enforcement

---

## Performance Considerations

### API Rate Limiting

GitHub REST API has rate limits:
- **Authenticated:** 5,000 requests/hour
- **Search API:** 30 requests/minute

**Mitigation Strategies:**
1. **Caching:** All tools support result caching (default 1 hour)
2. **Batch Requests:** Tools minimize API calls through efficient batching
3. **Conditional Requests:** Use ETags for unchanged data
4. **Pagination:** Intelligent pagination for large result sets

### Tool Performance

| Tool | API Calls (typical) | Cache Duration | Execution Time |
|------|-------------------|----------------|----------------|
| github-searcher | 1-5 | 1 hour | 2-5s |
| github-permissions | 5-10 | 1 hour | 3-8s |
| github-deployment-access | 3-7 | 1 hour | 2-6s |
| github-security-features | 4-8 | 1 hour | 2-5s |
| github-workflow-analyzer | 10-30 | 30 min | 5-15s |
| github-review-analyzer | 20-100 | 1 hour | 10-30s |

---

## Configuration

### Basic Configuration

```yaml
# .grctool.yaml
github:
  token: "${GITHUB_TOKEN}"
  default_org: "your-organization"
  cache:
    enabled: true
    ttl: "1h"
    directory: ".cache/github"
  api:
    base_url: "https://api.github.com"
    timeout: "30s"
    retry_max: 3
```

### Environment Variables

```bash
# Required
export GITHUB_TOKEN="ghp_your_personal_access_token"

# Optional
export GITHUB_API_URL="https://api.github.com"  # For GitHub Enterprise
export GITHUB_ORG="your-default-organization"
```

---

## Error Handling

### Common Errors

**1. Authentication Failure**
```json
{
  "error": "github_auth_failed",
  "message": "GitHub API authentication failed: invalid token",
  "recommendation": "Check GITHUB_TOKEN environment variable"
}
```

**2. Rate Limit Exceeded**
```json
{
  "error": "rate_limit_exceeded",
  "message": "GitHub API rate limit exceeded",
  "rate_limit": {
    "limit": 5000,
    "remaining": 0,
    "reset_at": "2025-01-10T16:00:00Z"
  },
  "recommendation": "Wait until rate limit resets or enable caching"
}
```

**3. Repository Not Found**
```json
{
  "error": "repository_not_found",
  "message": "Repository 'owner/repo' not found or not accessible",
  "recommendation": "Check repository name and token permissions"
}
```

---

## Testing

### Unit Tests
- API response parsing
- Permission matrix generation
- Compliance checking logic
- Date range filtering

### Integration Tests
- GitHub API mocking with VCR
- Multi-repository analysis
- Rate limit handling
- Cache invalidation

### Test Fixtures

Located in `internal/tools/github/testdata/`:
```
testdata/
├── repositories/
│   ├── public-repo.json
│   └── private-repo.json
├── workflows/
│   ├── security-scan.yml
│   └── deploy.yml
└── vcr/
    ├── permissions_test.yaml
    └── workflows_test.yaml
```

---

## Future Enhancements

### Planned Features

1. **Advanced Analytics**
   - Repository activity heatmaps
   - Security posture scoring
   - Trend analysis over time

2. **GitHub Enterprise Support**
   - Self-hosted GitHub instance support
   - Custom API endpoints
   - Enterprise-specific features

3. **Additional Integrations**
   - GitHub Advanced Security features
   - GitHub Insights data
   - Third-party security tools integration

4. **Automated Remediation**
   - Suggest security improvements
   - Generate compliance reports
   - Auto-fix common issues

---

## References

### External Dependencies

- **GitHub REST API:** `github.com/google/go-github/v57/github`
- **OAuth2:** `golang.org/x/oauth2`

### Related Documentation

- [GitHub REST API Documentation](https://docs.github.com/en/rest)
- [GitHub Security Best Practices](https://docs.github.com/en/code-security)
- [SOC2 Trust Services Criteria](https://www.aicpa.org/soc)

### Internal Documentation

- `docs/01-User-Guide/github-integration.md`
- `docs/04-Development/testing-guide.md`
- `docs/06-Compliance/SOC2-controls.md`

---

**Version History:**
- v1.0 (2025-01-10): Initial specification for GitHub tools suite
