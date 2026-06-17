# GRCTool Evidence Assembly Screencast

This screencast demonstrates the complete workflow for assembling compliance evidence using GRCTool, combining data from policies, controls, GitHub, and Terraform.

## Files

- **grctool-evidence-demo.cast** - The asciinema recording (13 KB, ~52 lines)
- **demo-script-v2.sh** - The source script used to generate the recording
- **record-demo.sh** - Original demo script (deprecated)
- **demo-script-final.sh** - Intermediate version (deprecated)

## What's Demonstrated

The screencast shows a complete evidence assembly workflow:

### 1. Evidence Discovery (Steps 1-3)
- Check GRCTool version and build information
- List available evidence tasks
- Get detailed information about a specific evidence task (ET-0001: Access Control)

### 2. Relationship Analysis (Steps 4-5)
- Map relationships between evidence tasks, controls, and policies
- View detailed control information (AC1: Access Provisioning and Approval)

### 3. Context Generation (Step 6)
- Generate control-specific summaries for evidence collection
- Save context to structured files for AI-assisted evidence generation

### 4. GitHub Evidence Collection (Steps 7-8)
- Extract repository permissions (collaborators, teams, branch protection)
- Analyze GitHub Actions workflows for CI/CD security controls

### 5. Terraform Infrastructure Scanning (Steps 9-10)
- Scan Terraform configurations for IAM roles and security groups
- Search for encryption-related resources using pattern matching

### 6. Evidence Review (Steps 11-12)
- Show generated evidence files
- Display evidence directory structure

## How to Use

### Play the Recording

```bash
asciinema play grctool-evidence-demo.cast
```

Press `Space` to pause/resume, `Ctrl+C` to exit.

### Upload and Share

```bash
# Upload to asciinema.org for web viewing
asciinema upload grctool-evidence-demo.cast
```

This will provide a URL you can share with others. The recording can be viewed in any web browser without installing asciinema.

### Convert to GIF (For GitHub Embedding)

To embed the demo directly in GitHub README or documentation:

```bash
# Install agg (asciinema GIF generator)
cargo install --git https://github.com/asciinema/agg

# Convert to animated GIF
agg grctool-evidence-demo.cast grctool-evidence-demo.gif

# Optimize the GIF size
# Install gifsicle (optional but recommended)
brew install gifsicle  # macOS
# or: apt install gifsicle  # Ubuntu/Debian

# Optimize
gifsicle -O3 --lossy=80 grctool-evidence-demo.gif -o grctool-evidence-demo-optimized.gif
```

Then embed in GitHub markdown:
```markdown
![GRCTool Demo](grctool-evidence-demo.gif)
```

**Note:** GIFs can be large. Consider:
- Using lower frame rates: `agg --fps 10 grctool-evidence-demo.cast demo.gif`
- Reducing dimensions: `agg --cols 100 --rows 30 grctool-evidence-demo.cast demo.gif`
- Uploading to asciinema.org for the embedded player instead

### Export to SVG

```bash
# Using svg-term
npm install -g svg-term-cli
cat grctool-evidence-demo.cast | svg-term --out grctool-evidence-demo.svg
```

### Embed in Documentation

```html
<!-- Embed on a webpage -->
<script src="https://asciinema.org/a/[ID].js" id="asciicast-[ID]" async></script>
```

## Technical Details

- **Terminal**: 80 columns × 24 rows (standard terminal size)
- **Duration**: ~2-3 minutes
- **Format**: Asciinema v3 format (JSON-based)
- **Shell**: Zsh

## Commands Demonstrated

All commands shown in the screencast use real grctool functionality:

```bash
# Version info
./build/grctool version

# Evidence tasks
./build/grctool tool evidence-task-list
./build/grctool tool evidence-task-details --task-ref ET-0001

# Relationships
./build/grctool tool evidence-relationships --task-ref ET-0001 --depth 2

# Control summaries
./build/grctool tool control-summary-generator --control-id AC1 --task-ref ET-0001

# GitHub evidence
./build/grctool tool github-permissions --repository owner/repo
./build/grctool tool github-workflow-analyzer --analysis-type full

# Terraform scanning
./build/grctool tool terraform-scanner --resource-types aws_iam_role
./build/grctool tool terraform-scanner --pattern 'encrypt|kms'
```

## Recreating the Recording

To regenerate the screencast:

```bash
# Make the script executable
chmod +x demo-script-v2.sh

# Record a new session
asciinema rec -c "./demo-script-v2.sh" --overwrite grctool-evidence-demo.cast
```

## Next Steps After Watching

Try these commands to continue the workflow:

```bash
# Generate evidence for a task
grctool evidence generate ET-0001

# Evaluate evidence quality
grctool evidence evaluate ET-0001

# Submit evidence to Tugboat Logic
grctool evidence submit ET-0001
```

## Notes

- The recording uses a headless mode (no TTY) for consistent playback
- All data shown is real from the 7thsense-ops/isms project
- GitHub and Terraform tools connect to live data sources
- Some commands may show zero results if resources don't exist

## Feedback

For questions or issues with the screencast:
- Open an issue at https://github.com/anthropics/grctool/issues
- Check the main documentation: `grctool --help`
