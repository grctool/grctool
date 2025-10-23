# Google Workspace Setup Guide

**Purpose:** Step-by-step guide to configure Google Workspace authentication for GRCTool evidence collection

**Estimated Time:** 20-30 minutes

**Prerequisites:**
- Access to Google Cloud Console (console.cloud.google.com)
- Google Workspace administrator access (for domain-wide delegation, if needed)
- Permissions to create Google Cloud projects and service accounts

---

## Table of Contents

1. [Overview](#overview)
2. [Step 1: Create Google Cloud Project](#step-1-create-google-cloud-project)
3. [Step 2: Enable Required APIs](#step-2-enable-required-apis)
4. [Step 3: Create Service Account](#step-3-create-service-account)
5. [Step 4: Configure Service Account Permissions](#step-4-configure-service-account-permissions)
6. [Step 5: Generate and Download Credentials](#step-5-generate-and-download-credentials)
7. [Step 6: Configure GRCTool](#step-6-configure-grctool)
8. [Step 7: Test the Integration](#step-7-test-the-integration)
9. [Troubleshooting](#troubleshooting)
10. [Security Best Practices](#security-best-practices)

---

## Overview

GRCTool uses **service account authentication** to access Google Workspace documents. This method:
- ✅ Provides programmatic access without user interaction
- ✅ Uses read-only scopes for security
- ✅ Works in automated/CI-CD environments
- ✅ Supports organization-wide access (with domain-wide delegation)

### Authentication Flow

```
┌──────────────┐
│   GRCTool    │
└──────┬───────┘
       │ 1. Reads service account credentials
       │    (JSON file with private key)
       ▼
┌──────────────────┐
│  Google OAuth2   │
│  Token Endpoint  │◄─── 2. JWT signed with private key
└──────┬───────────┘
       │ 3. Returns access token
       ▼
┌──────────────────┐
│ Google Workspace │
│      APIs        │◄─── 4. API requests with access token
│ (Drive/Docs/     │
│  Sheets/Forms)   │
└──────────────────┘
```

---

## Step 1: Create Google Cloud Project

### 1.1 Navigate to Google Cloud Console

1. Open your browser and go to: https://console.cloud.google.com
2. Sign in with your Google account
3. If you have multiple organizations, select the appropriate one

### 1.2 Create a New Project

1. Click the **project dropdown** in the top navigation bar
2. Click **"NEW PROJECT"** button
3. Fill in project details:
   - **Project name:** `grctool-compliance` (or your preferred name)
   - **Organization:** Select your organization (if applicable)
   - **Location:** Select your organization or leave as default
4. Click **"CREATE"**
5. Wait for project creation (takes 10-30 seconds)
6. Select the new project from the project dropdown

**Screenshot Reference:**
```
Google Cloud Console > Project Dropdown > NEW PROJECT

Project Details:
  Project name: grctool-compliance
  Project ID:   grctool-compliance-1234 (auto-generated)
  Organization: example.com
  Location:     example.com
```

---

## Step 2: Enable Required APIs

### 2.1 Navigate to API Library

1. In the Google Cloud Console, open the **hamburger menu** (☰) in the top-left
2. Navigate to **"APIs & Services"** > **"Library"**
3. You'll see the API Library with thousands of available APIs

### 2.2 Enable Google Drive API

1. In the API Library search box, type: `Google Drive API`
2. Click on **"Google Drive API"** from the results
3. Click the **"ENABLE"** button
4. Wait for the API to be enabled (~5 seconds)

### 2.3 Enable Google Docs API

1. Click the **back arrow** or navigate back to **API Library**
2. Search for: `Google Docs API`
3. Click on **"Google Docs API"**
4. Click **"ENABLE"**

### 2.4 Enable Google Sheets API

1. Navigate back to **API Library**
2. Search for: `Google Sheets API`
3. Click on **"Google Sheets API"**
4. Click **"ENABLE"**

### 2.5 Enable Google Forms API

1. Navigate back to **API Library**
2. Search for: `Google Forms API`
3. Click on **"Google Forms API"**
4. Click **"ENABLE"**

### 2.6 Verify Enabled APIs

1. Navigate to **"APIs & Services"** > **"Enabled APIs & services"**
2. You should see all four APIs listed:
   - ✅ Google Drive API
   - ✅ Google Docs API
   - ✅ Google Sheets API
   - ✅ Google Forms API

**Note:** API enablement is immediate and free. No billing required unless you exceed generous free quotas (unlikely for compliance evidence collection).

---

## Step 3: Create Service Account

### 3.1 Navigate to Service Accounts

1. In Google Cloud Console, go to **hamburger menu** (☰)
2. Navigate to **"IAM & Admin"** > **"Service Accounts"**
3. Ensure your project (`grctool-compliance`) is selected in the project dropdown

### 3.2 Create New Service Account

1. Click **"+ CREATE SERVICE ACCOUNT"** at the top
2. Fill in service account details:

   **Step 1: Service account details**
   - **Service account name:** `grctool-evidence-collector`
   - **Service account ID:** `grctool-evidence-collector` (auto-filled)
   - **Service account description:** `Service account for GRCTool to collect compliance evidence from Google Workspace documents`
   - Click **"CREATE AND CONTINUE"**

   **Step 2: Grant this service account access to project (optional)**
   - **Role:** Leave blank (not needed for document access)
   - Click **"CONTINUE"**

   **Step 3: Grant users access to this service account (optional)**
   - Leave blank
   - Click **"DONE"**

### 3.3 Locate Your Service Account

You should now see your service account in the list:
```
grctool-evidence-collector@grctool-compliance-1234.iam.gserviceaccount.com
```

**Save this email address** - you'll need it to share documents with the service account.

---

## Step 4: Configure Service Account Permissions

### 4.1 Share Documents with Service Account

The service account needs access to the Google Workspace documents you want to extract evidence from.

**Method 1: Share Individual Documents**

1. Open the Google Drive folder or document you want to access
2. Click the **"Share"** button
3. Enter the service account email: `grctool-evidence-collector@grctool-compliance-1234.iam.gserviceaccount.com`
4. Set permission to **"Viewer"** (read-only)
5. Click **"Send"** (uncheck "Notify people" to avoid notification emails)

**Method 2: Share an Entire Folder**

1. In Google Drive, right-click on the folder containing your compliance documents
2. Select **"Share"**
3. Add the service account email with **"Viewer"** permissions
4. This grants access to all files in the folder and subfolders

**Method 3: Domain-Wide Delegation (Advanced)**

For organization-wide access to all documents, you can enable domain-wide delegation:

1. In Google Cloud Console, go to **"IAM & Admin"** > **"Service Accounts"**
2. Click on your service account
3. Click **"SHOW ADVANCED SETTINGS"** (if available) or go to the **"Details"** tab
4. Under "Domain-wide delegation", click **"Enable G Suite Domain-wide Delegation"**
5. Note the **Client ID** (numeric ID)
6. Go to Google Workspace Admin Console: https://admin.google.com
7. Navigate to **Security** > **API Controls** > **Domain-wide Delegation**
8. Click **"Add new"**
9. Enter the Client ID and the following OAuth scopes:
   ```
   https://www.googleapis.com/auth/drive.readonly,https://www.googleapis.com/auth/documents.readonly,https://www.googleapis.com/auth/spreadsheets.readonly,https://www.googleapis.com/auth/forms.responses.readonly
   ```
10. Click **"Authorize"**

⚠️ **Warning:** Domain-wide delegation grants access to ALL documents in your organization. Only use this if you need broad access and have proper security controls.

---

## Step 5: Generate and Download Credentials

### 5.1 Create Service Account Key

1. In Google Cloud Console, navigate to **"IAM & Admin"** > **"Service Accounts"**
2. Find your service account: `grctool-evidence-collector@...`
3. Click the **three-dot menu** (⋮) on the right
4. Select **"Manage keys"**
5. Click **"ADD KEY"** > **"Create new key"**
6. Select **"JSON"** as the key type
7. Click **"CREATE"**

### 5.2 Download Credentials File

- Your browser will automatically download a JSON file
- The file will be named something like: `grctool-compliance-1234-abc123def456.json`
- **This file contains your private key** - treat it like a password!

### 5.3 Secure the Credentials File

**Immediately after downloading:**

1. **Move the file to a secure location:**
   ```bash
   # Example: Move to a secure directory
   mkdir -p ~/.config/grctool
   mv ~/Downloads/grctool-compliance-*.json ~/.config/grctool/google-credentials.json
   ```

2. **Set restrictive file permissions:**
   ```bash
   chmod 600 ~/.config/grctool/google-credentials.json
   ```

3. **Verify permissions:**
   ```bash
   ls -la ~/.config/grctool/google-credentials.json
   # Should show: -rw------- (owner read/write only)
   ```

4. **Add to .gitignore** (if in a git repository):
   ```bash
   echo "google-credentials.json" >> .gitignore
   echo "*-credentials.json" >> .gitignore
   ```

### 5.4 Understand the Credentials File Structure

Your credentials file contains:

```json
{
  "type": "service_account",
  "project_id": "grctool-compliance-1234",
  "private_key_id": "abc123def456...",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
  "client_email": "grctool-evidence-collector@grctool-compliance-1234.iam.gserviceaccount.com",
  "client_id": "123456789012345678901",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/...",
  "universe_domain": "googleapis.com"
}
```

**Critical Fields:**
- `private_key`: RSA private key used to sign JWT tokens (KEEP SECRET!)
- `client_email`: Service account email (use this to share documents)
- `project_id`: Your GCP project identifier

---

## Step 6: Configure GRCTool

### 6.1 Set Environment Variable (Recommended)

The simplest method is to set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable:

**macOS/Linux (Bash/Zsh):**
```bash
# Add to ~/.bashrc, ~/.zshrc, or ~/.profile
export GOOGLE_APPLICATION_CREDENTIALS="$HOME/.config/grctool/google-credentials.json"

# Apply immediately
source ~/.bashrc  # or ~/.zshrc
```

**macOS/Linux (Fish shell):**
```fish
# Add to ~/.config/fish/config.fish
set -Ux GOOGLE_APPLICATION_CREDENTIALS "$HOME/.config/grctool/google-credentials.json"
```

**Windows (PowerShell):**
```powershell
# Set user environment variable
[System.Environment]::SetEnvironmentVariable(
    "GOOGLE_APPLICATION_CREDENTIALS",
    "$env:USERPROFILE\.config\grctool\google-credentials.json",
    [System.EnvironmentVariableTarget]::User
)
```

### 6.2 Verify Environment Variable

```bash
echo $GOOGLE_APPLICATION_CREDENTIALS
# Should output: /home/user/.config/grctool/google-credentials.json
```

### 6.3 Alternative: Use Explicit Path in Commands

You can also specify the credentials path in each command:

```bash
grctool tool google-workspace \
  --document-id 1A2B3C4D5E6F7G8H9I0J \
  --document-type drive \
  --credentials-path ~/.config/grctool/google-credentials.json
```

---

## Step 7: Test the Integration

### 7.1 Prepare a Test Document

1. Create a simple Google Doc or use an existing one
2. Share it with your service account email (Viewer permission)
3. Note the **document ID** from the URL:
   ```
   https://docs.google.com/document/d/1K2L3M4N5O6P7Q8R9S0T/edit
                                      └──────────────────┘
                                         Document ID
   ```

### 7.2 Run a Test Command

**Test with Google Docs:**
```bash
grctool tool google-workspace \
  --document-id 1K2L3M4N5O6P7Q8R9S0T \
  --document-type docs
```

**Expected Output:**
```json
{
  "tool": "google-workspace",
  "document_type": "docs",
  "document_id": "1K2L3M4N5O6P7Q8R9S0T",
  "title": "Test Document",
  "extracted_at": "2025-10-23T14:30:00Z",
  "content": {
    "text": "This is a test document...",
    "word_count": 42
  },
  "metadata": {
    "title": "Test Document",
    "created_time": "2025-10-20T10:00:00Z",
    "modified_time": "2025-10-23T12:00:00Z",
    "web_view_link": "https://docs.google.com/document/d/1K2L3M4N5O6P7Q8R9S0T/view"
  }
}
```

### 7.3 Test with Different Document Types

**Test Google Drive Folder:**
```bash
grctool tool google-workspace \
  --document-id 1A2B3C4D5E6F7G8H9I0J \
  --document-type drive
```

**Test Google Sheets:**
```bash
grctool tool google-workspace \
  --document-id 1U2V3W4X5Y6Z7A8B9C0D \
  --document-type sheets \
  --sheet-range "Sheet1!A1:D10"
```

**Test Google Forms:**
```bash
grctool tool google-workspace \
  --document-id 1E2F3G4H5I6J7K8L9M0N \
  --document-type forms
```

---

## Troubleshooting

### Error: "credentials not found"

**Error Message:**
```
Error: google credentials not found. Set credentials_path parameter or GOOGLE_APPLICATION_CREDENTIALS environment variable
```

**Solutions:**
1. Verify environment variable is set: `echo $GOOGLE_APPLICATION_CREDENTIALS`
2. Check file exists: `ls -la $GOOGLE_APPLICATION_CREDENTIALS`
3. Try explicit path: `--credentials-path /full/path/to/credentials.json`
4. Restart your terminal to load new environment variables

### Error: "403 Forbidden" or "Permission Denied"

**Error Message:**
```
Error: failed to access document: 403 Forbidden
```

**Solutions:**
1. **Check document is shared:** Verify the service account email has Viewer permission
2. **Verify service account email:** Use the exact email from the credentials file
3. **Check API is enabled:** Ensure Google Drive/Docs/Sheets/Forms APIs are enabled in GCP
4. **Wait for propagation:** Sharing changes can take 1-2 minutes to propagate

### Error: "404 Not Found"

**Error Message:**
```
Error: failed to extract from drive: document not found (404)
```

**Solutions:**
1. **Verify document ID:** Check the ID is correct (no extra characters)
2. **Check document type:** Ensure `--document-type` matches the actual document type
3. **Confirm document exists:** Open the document in your browser to verify it's accessible
4. **Check document is not deleted:** Verify the document isn't in Google Drive trash

### Error: "Invalid JWT Signature"

**Error Message:**
```
Error: failed to initialize Google client: invalid JWT signature
```

**Solutions:**
1. **Re-download credentials:** The credentials file may be corrupted
2. **Check file permissions:** Ensure the file is readable (`chmod 600`)
3. **Verify file format:** Open the file and ensure it's valid JSON
4. **Check for modifications:** Don't manually edit the credentials file

### Error: "Rate Limit Exceeded"

**Error Message:**
```
Error: rate limit exceeded for Google Drive API
```

**Solutions:**
1. **Wait and retry:** Google APIs have quotas; wait a few minutes
2. **Implement backoff:** Add delays between requests
3. **Check quota limits:** Review quota usage in GCP Console > APIs & Services > Dashboard
4. **Request quota increase:** If needed, request higher quotas from Google

### Error: "API Not Enabled"

**Error Message:**
```
Error: Google Drive API has not been used in project grctool-compliance-1234 before or it is disabled
```

**Solutions:**
1. Go to GCP Console > APIs & Services > Library
2. Search for the API mentioned in the error
3. Click "Enable"
4. Wait 1-2 minutes and retry

### Authentication Test Script

Create a test script to diagnose issues:

```bash
#!/bin/bash
# google-workspace-test.sh

echo "=== Google Workspace Authentication Test ==="
echo

echo "1. Checking environment variable..."
if [ -z "$GOOGLE_APPLICATION_CREDENTIALS" ]; then
    echo "❌ GOOGLE_APPLICATION_CREDENTIALS not set"
    exit 1
else
    echo "✅ GOOGLE_APPLICATION_CREDENTIALS = $GOOGLE_APPLICATION_CREDENTIALS"
fi

echo
echo "2. Checking credentials file exists..."
if [ ! -f "$GOOGLE_APPLICATION_CREDENTIALS" ]; then
    echo "❌ Credentials file not found"
    exit 1
else
    echo "✅ Credentials file exists"
fi

echo
echo "3. Checking file permissions..."
PERMS=$(stat -c %a "$GOOGLE_APPLICATION_CREDENTIALS" 2>/dev/null || stat -f %A "$GOOGLE_APPLICATION_CREDENTIALS")
if [ "$PERMS" != "600" ]; then
    echo "⚠️  File permissions are $PERMS (should be 600)"
    echo "    Run: chmod 600 $GOOGLE_APPLICATION_CREDENTIALS"
else
    echo "✅ File permissions correct (600)"
fi

echo
echo "4. Checking JSON validity..."
if jq empty "$GOOGLE_APPLICATION_CREDENTIALS" 2>/dev/null; then
    echo "✅ JSON is valid"
else
    echo "❌ JSON is invalid or jq not installed"
fi

echo
echo "5. Extracting service account email..."
EMAIL=$(jq -r .client_email "$GOOGLE_APPLICATION_CREDENTIALS" 2>/dev/null)
if [ -n "$EMAIL" ]; then
    echo "✅ Service account: $EMAIL"
    echo
    echo "   Use this email to share documents in Google Drive!"
else
    echo "❌ Could not extract service account email"
fi

echo
echo "=== Test Complete ==="
```

Make it executable and run:
```bash
chmod +x google-workspace-test.sh
./google-workspace-test.sh
```

---

## Security Best Practices

### Credential Management

1. **Never commit credentials to version control**
   ```bash
   # Add to .gitignore
   *-credentials.json
   google-credentials.json
   service-account*.json
   ```

2. **Use restrictive file permissions**
   ```bash
   chmod 600 /path/to/credentials.json
   ```

3. **Store credentials outside repository**
   ```bash
   # Good: ~/.config/grctool/credentials.json
   # Bad:  ./credentials.json (in project directory)
   ```

4. **Rotate keys regularly**
   - Recommended: Every 90 days
   - Create new key before deleting old one
   - Test with new key before removing old key

5. **Use different service accounts per environment**
   - Development: `grctool-dev@project.iam.gserviceaccount.com`
   - Production: `grctool-prod@project.iam.gserviceaccount.com`

### Access Control

1. **Principle of Least Privilege**
   - Grant service account access only to required documents
   - Use Viewer/Reader permissions (never Editor unless required)
   - Avoid domain-wide delegation unless necessary

2. **Regular Access Reviews**
   - Quarterly review of service account permissions
   - Remove access to documents no longer needed
   - Audit service account usage in Google Workspace Admin

3. **Monitoring and Auditing**
   - Enable Google Workspace audit logs
   - Review service account activity regularly
   - Set up alerts for unusual access patterns

### Operational Security

1. **Secure CI/CD Secrets**
   ```yaml
   # GitHub Actions example
   - name: Collect Evidence
     env:
       GOOGLE_APPLICATION_CREDENTIALS: ${{ secrets.GOOGLE_CREDENTIALS }}
     run: |
       echo "$GOOGLE_APPLICATION_CREDENTIALS" > /tmp/creds.json
       grctool tool google-workspace --credentials-path /tmp/creds.json ...
       rm /tmp/creds.json
   ```

2. **Use Secret Management Tools**
   - HashiCorp Vault
   - Google Secret Manager
   - AWS Secrets Manager
   - Azure Key Vault

3. **Implement Key Rotation**
   ```bash
   # Example rotation script
   # 1. Create new key
   gcloud iam service-accounts keys create new-key.json \
     --iam-account=grctool@project.iam.gserviceaccount.com

   # 2. Test with new key
   export GOOGLE_APPLICATION_CREDENTIALS=./new-key.json
   grctool tool google-workspace --document-id <test-id> --document-type docs

   # 3. If successful, delete old key
   gcloud iam service-accounts keys delete OLD_KEY_ID \
     --iam-account=grctool@project.iam.gserviceaccount.com
   ```

4. **Encrypt Credentials at Rest**
   ```bash
   # Example: Encrypt with GPG
   gpg --symmetric --cipher-algo AES256 google-credentials.json

   # Decrypt when needed
   gpg --decrypt google-credentials.json.gpg > google-credentials.json
   ```

---

## Next Steps

### Integration with Evidence Collection

Now that authentication is configured, you can:

1. **Identify compliance documents** in Google Workspace
2. **Share documents** with the service account
3. **Extract evidence** using GRCTool
4. **Automate evidence collection** in CI/CD pipelines

### Example Evidence Collection Workflow

```bash
# 1. Extract policy documents from Drive folder
grctool tool google-workspace \
  --document-id 1A2B3C4D5E6F7G8H9I0J \
  --document-type drive \
  > evidence/policies-inventory.json

# 2. Extract security policy content
grctool tool google-workspace \
  --document-id 1K2L3M4N5O6P7Q8R9S0T \
  --document-type docs \
  > evidence/information-security-policy.json

# 3. Extract access review spreadsheet
grctool tool google-workspace \
  --document-id 1U2V3W4X5Y6Z7A8B9C0D \
  --document-type sheets \
  --sheet-range "Q3 2025!A1:F500" \
  > evidence/access-review-q3-2025.json

# 4. Extract training quiz responses
grctool tool google-workspace \
  --document-id 1E2F3G4H5I6J7K8L9M0N \
  --document-type forms \
  > evidence/security-training-responses.json
```

### Additional Resources

- **Technical Specification:** `docs/04-Development/google-workspace-tools-specification.md`
- **CLI Reference:** `docs/reference/cli-commands.md` (Google Workspace section)
- **Evidence Mapping:** Configure `google_evidence_mappings.yaml` for automatic evidence collection
- **API Documentation:** https://developers.google.com/workspace

### Support

If you encounter issues not covered in this guide:

1. Check the [Troubleshooting](#troubleshooting) section above
2. Review Google Workspace API documentation
3. Check GRCTool logs: `tail -f grctool.log`
4. Enable verbose logging: `grctool --verbose --log-level debug tool google-workspace ...`

---

**Document Version:** 1.0
**Last Updated:** 2025-10-23
**Maintained By:** GRCTool Project
