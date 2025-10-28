# Evidence Submission Process

> Complete guide for submitting evidence to Tugboat Logic

---

**Generated**: 2025-10-28 10:33:19 EDT
**GRCTool Version**: dev
**Documentation Version**: dev

---

## Overview

Evidence submission uploads files to Tugboat Logic via the Custom Evidence Integration API. This document describes the setup, submission workflow, and troubleshooting.

---

## Setup Requirements

### 1. Generate Credentials in Tugboat UI

Navigate to: **Integrations** > **Custom Integrations**

1. Click **+** to add new integration
2. Enter account name and description
3. Click **Generate Password**
4. **IMPORTANT**: Save Username, Password, and X-API-KEY (cannot be recovered)

### 2. Generate Collector URLs

For each evidence task:
1. Click **+** to configure evidence service
2. Select scope and evidence task
3. Click **Copy URL** - this is your collector URL
4. Repeat for each task

### 3. Configure GRCTool

Add to `.grctool.yaml`:
```yaml
tugboat:
  username: "your-username"
  password: "your-password"
  collector_urls:
    "ET-0001": "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/"
    "ET-0047": "https://openapi.tugboatlogic.com/api/v0/evidence/collector/806/"
    # Add more task -> URL mappings
```

### 4. Set API Key Environment Variable

```bash
# Direct
export TUGBOAT_API_KEY="your-x-api-key-from-step-1"

# OR use 1Password
op run --env-file=".env.tugboat" -- grctool evidence submit ET-0001
```

---

## Submission Workflow

### Submit Evidence for Single Task

```bash
# 1. (Optional) Evaluate first
grctool evidence evaluate ET-0047 --window 2025-Q4

# 2. Preview submission (dry-run)
grctool evidence submit ET-0047 --window 2025-Q4 --dry-run

# 3. Submit (files auto-move from root to .submitted/)
grctool evidence submit ET-0047 \
  --window 2025-Q4 \
  --notes "Q4 quarterly review"
```

**What Happens**:
1. Command reads files from root directory (e.g., `2025-Q4/01_evidence.md`)
2. Files are uploaded to Tugboat Logic
3. **Files automatically move from root to `.submitted/`**
4. Submission metadata is saved to `.submitted/.submission/submission.yaml`
5. Root directory is now empty and ready for next collection

### Submit Manual Evidence

GRCTool can submit ANY files in the root directory:

```bash
# Place files in root directory (NOT in subdirectories)
cp ~/Documents/policy.pdf data/evidence/ET-0001_Policy/2025-Q4/
cp ~/Documents/training.xlsx data/evidence/ET-0001_Policy/2025-Q4/

# Submit (evaluates if needed, or skip evaluation)
grctool evidence submit ET-0001 --window 2025-Q4 --skip-evaluation
```

**Important**: Only files in the root directory are submitted. Files in `.submitted/`, `archive/`, or other subdirectories are NOT submitted.

### Bulk Submission

```bash
# Submit all evaluated evidence
for task in ET-0047 ET-0048 ET-0049; do
  grctool evidence submit $task --window 2025-Q4
done
```

### Checking Already-Submitted Files

```bash
# Check if files are already submitted (prevents resubmission)
ls data/evidence/ET-0047_*/2025-Q4/.submitted/

# Check submission status
cat data/evidence/ET-0047_*/2025-Q4/.submitted/.submission/submission.yaml
```

### Resubmitting Evidence

If you need to resubmit evidence:

```bash
# 1. Move files back from .submitted/ to root
mv data/evidence/ET-0047_*/2025-Q4/.submitted/*.md data/evidence/ET-0047_*/2025-Q4/

# 2. Make necessary changes
# Edit files in root directory

# 3. Resubmit (files move back to .submitted/)
grctool evidence submit ET-0047 --window 2025-Q4
```

---

## Supported File Types

**Supported**: txt, csv, json, pdf, png, gif, jpg, jpeg, md, doc, docx, xls, xlsx, odt, ods
**Max Size**: 20MB per file
**Not Supported**: html, htm, js, exe, php

---

## Understanding Evidence Directories

### Root Directory (Working Area)

**Location**: `evidence/ET-XXXX_TaskName/2025-Q4/`

**Purpose**: Active working directory for evidence generation and review

**What Gets Submitted**: ALL files in root directory (excluding subdirectories)

**After Submission**: Files automatically move to `.submitted/`

### .submitted/ Directory (Hidden)

**Location**: `evidence/ET-XXXX_TaskName/2025-Q4/.submitted/`

**Purpose**: Stores evidence after successful submission to prevent resubmission

**Contents**:
- Evidence files that were uploaded
- `.submission/submission.yaml` with tracking metadata

**Important**: Files here are NOT submitted again when you run submit command

### archive/ Directory

**Location**: `evidence/ET-XXXX_TaskName/2025-Q4/archive/`

**Purpose**: Evidence synced FROM Tugboat Logic (auditor-approved versions)

**Contents**:
- Evidence files downloaded from Tugboat
- `.submission/submission.yaml` with Tugboat metadata

**Important**: This is read-only reference material, NOT uploaded by submit command

### Key Differences

| Directory | Purpose | Submitted? | Source |
|-----------|---------|------------|--------|
| **Root** | Working area | YES (then moved) | You create |
| **.submitted/** | Uploaded files | NO (already submitted) | Moved from root after upload |
| **archive/** | Synced evidence | NO (from Tugboat) | Downloaded via sync |

## Troubleshooting

### Error: TUGBOAT_API_KEY not set
**Solution**: Export the environment variable
```bash
export TUGBOAT_API_KEY="your-key"
```

### Error: credentials not configured
**Solution**: Add username/password to .grctool.yaml

### Error: collector URL not configured
**Solution**: Add task mapping to tugboat.collector_urls

### Error: no files found in root directory
**Solution**: Evidence files must be in root directory, not in subdirectories
```bash
# Check for files in root
ls data/evidence/ET-0047_*/2025-Q4/*.md

# NOT in subdirectories
ls data/evidence/ET-0047_*/2025-Q4/.submitted/  # Already submitted
ls data/evidence/ET-0047_*/2025-Q4/archive/     # Synced from Tugboat
```

### Error: file extension not supported
**Solution**: Check file type is in supported list

### Error: file size exceeds maximum
**Solution**: File must be under 20MB

### Error: 401 Unauthorized
**Solution**: Check username/password are correct

### Error: 403 Forbidden
**Solution**: Verify API key is correct and has permissions

### Warning: Files already submitted
**Solution**: Files are already in `.submitted/` directory. To resubmit:
```bash
# Move files back to root
mv data/evidence/ET-XXXX_*/2025-Q4/.submitted/*.md data/evidence/ET-XXXX_*/2025-Q4/
```

---

**Next Steps**:
- After submission, files are in `.submitted/` directory
- Monitor status with `grctool status`
- Sync approved evidence from Tugboat with `grctool sync` (appears in `archive/`)
