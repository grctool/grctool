# Evidence Submission Process

> Complete guide for submitting evidence to Tugboat Logic

---

**Generated**: 2025-10-27 16:56:21 EDT  
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
# 1. Validate first
grctool evidence validate ET-0047 --window 2025-Q4

# 2. Preview submission (dry-run)
grctool evidence submit ET-0047 --window 2025-Q4 --dry-run

# 3. Submit
grctool evidence submit ET-0047 \
  --window 2025-Q4 \
  --notes "Q4 quarterly review"
```

### Submit Manual Evidence

GRCTool can submit ANY files in the evidence folder:

```bash
# Place files in folder
cp ~/Documents/policy.pdf data/evidence/ET-0001_Policy/2025-Q4/
cp ~/Documents/training.xlsx data/evidence/ET-0001_Policy/2025-Q4/

# Submit
grctool evidence submit ET-0001 --window 2025-Q4 --skip-validation
```

### Bulk Submission

```bash
# Submit all validated evidence
for task in ET-0047 ET-0048 ET-0049; do
  grctool evidence submit $task --window 2025-Q4
done
```

---

## Supported File Types

**Supported**: txt, csv, json, pdf, png, gif, jpg, jpeg, md, doc, docx, xls, xlsx, odt, ods  
**Max Size**: 20MB per file  
**Not Supported**: html, htm, js, exe, php  

---

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

### Error: file extension not supported
**Solution**: Check file type is in supported list  

### Error: file size exceeds maximum
**Solution**: File must be under 20MB  

### Error: 401 Unauthorized
**Solution**: Check username/password are correct  

### Error: 403 Forbidden
**Solution**: Verify API key is correct and has permissions  

---

**Next Steps**: After submission, monitor status with `grctool status` and wait for auditor review.
