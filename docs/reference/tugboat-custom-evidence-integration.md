# Setting Up Custom Evidence Integrations

**Source:** [OneTrust Certification Automation Support](https://support.tugboatlogic.com/hc/en-us/articles/360049620392-Setting-Up-Custom-Evidence-Integrations)
**Last Updated:** February 01, 2023
**Author:** Becky Carnegie

> **Note:** Certification Automation can support creating the API Key, Username, Password and URL. Certification Automation cannot support writing code or API calls to send data to the URL. Users can access our Custom API Marketplace (https://community.tugboatlogic.com/categories/custom-api-marketplace) to ask or engage our community. Users can also find examples of custom evidence integrations on GitHub here: https://github.com/PolicyHarbor/evidence-collectors

## Purpose of Document

The Setting Up Custom Evidence Integrations document allows you to customize and add your own integrations to upload specific evidence to a Certification Automation evidence task. This upload is made to the Certification Automation server using HTTP Basic Authorization and an API Key.

Many HTTP clients (e.g. written in Python, Ruby, PHP, C, C#) can be used to upload evidence. Evidence can be uploaded using cURL, a common command-line tool as demonstrated by the following:

```bash
curl -v --user <provided-username>:<given-password> \
 -H "X-API-KEY: <given-x-api-key>" \
 -F "collected=<date-of-evidence>" -F "file=@<local_filename_csv>;type=text/csv" \
 <given-collector-url>
```

Fields provided by Certification Automation are marked as **given**. The HTTP client is capable of POSTing Content-Type multipart/form-data according to RFC 2388.

### Supported File Types

**Supported:**
```
txt, csv, odt, ods, xls, json, pdf, png, gif, jpg, jpeg
```

**Not Supported (file extensions):**
```
html, htm, js, exe, php5, pht, phtml, shtml, asa, cer, asax, swf, xap
```

**Not Supported (MIME types):**
```
html, xhtml+xml, javascript, dosexec, shellscript
```

**Max file upload size:** 20MB

## Document Outline

This document contains three parts:

1. [Part 1: Generate an HTTP Header](#part-1-generate-an-http-header)
2. [Part 2: Generate an Evidence URL](#part-2-generate-an-evidence-url)
3. [Part 3: Build a Command Line](#part-3-build-a-command-line)

## Part 1: Generate an HTTP Header

1. Select **Custom Integrations**.
2. Click the **plus** to add a new integration.
3. Enter an **Account Name**.
4. Enter a **Description** (optional).
5. Click **Save and Continue**.
6. Enter a **Username**.
7. Click **Generate Password**.
8. Copy the **Username**, **Password** and **X-API-KEY** for later use, or click **Download JSON** and save the resulting file.

> **Note:** The **Username**, **Password** and **X-API-KEY** cannot be recovered again later.

## Part 2: Generate an Evidence URL

1. Click the **plus** to configure a new evidence service.
2. Select the scope from the **Filter by Scope** drop-down menu.
3. Select the Evidence Task from the **Select Evidence Task** drop-down menu.
4. Click **Save & Complete**.
5. Click **Copy URL** (located next to your new evidence service).
6. Save the URL so you can use it in your command script later.

## Part 3: Build a Command Line

1. Use the **HTTP Header** (created in Part 1) and URL (created in Part 2) to build a POST API Request to upload an evidence file. Include the following in your request:

   - **An evidence file:** This is the file that will be attached to your Evidence Task. Select a filename with the correct mime type, such as text/csv in the example below.
   - **Evidence collection date:** This is the date the evidence was generated, and it should be set as an ISO8601 date (yyyy-mm-dd).

Once you have the URL you can add it to your API request. Once you have the collector URL in place, you should see something similar to the following if written in cURL command line:

```bash
curl -v --user jamie:XBwgsKLNRwSM2KjVaYc67FJMspeTzjBW \
-H "X-API-KEY: 762bbc8a-0363-11eb-ae9b-0e8ffbd46778-org-id-11455" \
-F "collected=2020-09-30" -F "file=@evidence.csv;type=text/csv" \
https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/
```

## Example Integration

Here's a complete example showing all the pieces together:

```bash
# Credentials from Part 1 (HTTP Header)
USERNAME="jamie"
PASSWORD="XBwgsKLNRwSM2KjVaYc67FJMspeTzjBW"
API_KEY="762bbc8a-0363-11eb-ae9b-0e8ffbd46778-org-id-11455"

# Collector URL from Part 2
COLLECTOR_URL="https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/"

# Evidence file details
EVIDENCE_FILE="evidence.csv"
COLLECTION_DATE="2020-09-30"

# Submit evidence
curl -v --user "${USERNAME}:${PASSWORD}" \
  -H "X-API-KEY: ${API_KEY}" \
  -F "collected=${COLLECTION_DATE}" \
  -F "file=@${EVIDENCE_FILE};type=text/csv" \
  "${COLLECTOR_URL}"
```

## GRCTool Implementation

GRCTool implements this API through the `grctool evidence submit` command. For setup instructions, see the [Evidence Submission](../../CLAUDE.md#-evidence-submission) section in CLAUDE.md.

### Configuration

Add to `.grctool.yaml`:

```yaml
tugboat:
  username: "your-username"      # From Part 1
  password: "your-password"      # From Part 1
  collector_urls:
    "ET-0001": "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/"
    "ET-0047": "https://openapi.tugboatlogic.com/api/v0/evidence/collector/806/"
```

Set environment variable:

```bash
export TUGBOAT_API_KEY="your-x-api-key-from-part-1"
```

### Usage

```bash
# Submit evidence for a task
grctool evidence submit ET-0001 --window 2025-Q4 --notes "Quarterly review"
```

## Related Resources

- [Custom API Marketplace](https://community.tugboatlogic.com/categories/custom-api-marketplace)
- [Example Evidence Collectors](https://github.com/PolicyHarbor/evidence-collectors)
- [API Security Assurance Document](https://support.tugboatlogic.com/hc/en-us/articles/4401926231444)
