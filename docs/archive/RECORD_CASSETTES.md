# Recording VCR Cassettes

This guide shows you how to record VCR cassettes for integration tests that make GitHub API calls.

## Prerequisites

You need a GitHub personal access token with `repo` scope.

Create one at: https://github.com/settings/tokens

## Step 1: Set your GitHub token

```bash
export GITHUB_TOKEN="ghp_your_token_here"
```

## Step 2: Record the cassettes

Run each test in record mode:

```bash
# Record evidence_task_validation cassette
VCR_MODE=record go test -v -tags=integration -timeout=5m -run "TestEvidenceTaskValidation" ./test/integration/

# Record tools_integration cassette
VCR_MODE=record go test -v -tags=integration -timeout=5m -run "TestToolsIntegration" ./test/integration/
```

## Step 3: Verify the cassettes were created

```bash
ls -lh test/vcr_cassettes/evidence_task_validation.yaml
ls -lh test/vcr_cassettes/tools_integration.yaml
```

## Step 4: Run all tests to verify

```bash
go test -v -tags=integration ./test/integration/...
```

## What happens during recording?

- VCR makes **real GitHub API calls** using your `GITHUB_TOKEN`
- All HTTP requests and responses are recorded to YAML files
- Sensitive headers (Authorization, tokens) are automatically redacted
- The cassettes are saved to `test/vcr_cassettes/`

## Troubleshooting

### "Bad credentials" error
Make sure your `GITHUB_TOKEN` is set and valid:
```bash
echo $GITHUB_TOKEN
curl -H "Authorization: Bearer $GITHUB_TOKEN" https://api.github.com/user
```

### Cassette not being recorded
Check the test output for the log message:
```
🎬 VCR RECORD MODE - Recording cassette: evidence_task_validation.yaml
```

If you see `⚠️ VCR OFF` instead, the environment variable may not be set correctly.

### Need to re-record a cassette
Just delete the old cassette and record again:
```bash
rm test/vcr_cassettes/evidence_task_validation.yaml
VCR_MODE=record go test -v -tags=integration -run "TestEvidenceTaskValidation" ./test/integration/
```
