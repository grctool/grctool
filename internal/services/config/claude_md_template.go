// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"bytes"
	"text/template"

	"github.com/grctool/grctool/internal/config"
)

// ClaudeMdTemplate is the template for generating CLAUDE.md for AI assistance
const ClaudeMdTemplate = `# CLAUDE.md - AI Assistant Guide for GRCTool Users

## 🎯 PROJECT OVERVIEW

This project uses **GRCTool** - an automated compliance evidence collection CLI tool.

**What is GRCTool?**
- Automates evidence collection for SOC2, ISO27001, and other compliance frameworks
- Syncs policies, controls, and evidence tasks from Tugboat Logic
- Integrates with infrastructure (Terraform, GitHub, Google Workspace)
- Generates compliance evidence using AI and automated scanning

**Your Role as an AI Assistant:**
Help users navigate their compliance program, generate evidence, and understand their security controls.

## 📁 PROJECT STRUCTURE

This project is organized as follows:

**Data Directory**: {{.DataDir}}
{{if .DocsPath}}- **{{.DocsPath}}/** - Synced data from Tugboat Logic
  - **policies/** - Policy documents and metadata (JSON/Markdown)
  - **controls/** - Security controls and requirements (JSON/Markdown)
  - **evidence_tasks/** - Evidence collection tasks (JSON)
{{end}}{{if .EvidencePath}}- **{{.EvidencePath}}/** - Generated evidence files organized by task
{{end}}{{if .PromptsPath}}- **{{.PromptsPath}}/** - Custom evidence generation prompts
{{end}}{{if .CachePath}}- **{{.CachePath}}/** - Performance cache (can be safely deleted)
{{end}}
## 🔧 COMMON COMMANDS

### Initial Setup
` + "```bash" + `
# 1. Initialize configuration (safe to run multiple times)
grctool init

# 2. Authenticate with Tugboat Logic (browser-based)
grctool auth login

# 3. Verify authentication
grctool auth status
` + "```" + `

### Data Synchronization
` + "```bash" + `
# Sync latest data from Tugboat Logic
grctool sync

# This downloads:
# - Policies (governance documents)
# - Controls (security requirements)
# - Evidence Tasks (what evidence needs to be collected)
` + "```" + `

### Evidence Collection Workflow
` + "```bash" + `
# 1. List all evidence tasks
grctool tool evidence-task-list

# 2. Get details about a specific task
grctool tool evidence-task-details --task-ref ET-0001

# 3. Generate evidence for a task
grctool evidence generate ET-0001

# 4. Review generated evidence
grctool evidence review ET-0001
` + "```" + `

### Tool Discovery
` + "```bash" + `
# List all 29+ available evidence collection tools
grctool tool --help

# Examples of available tools:
grctool tool terraform-security-indexer  # Infrastructure security
grctool tool github-permissions          # Access controls
grctool tool github-workflow-analyzer    # CI/CD security
grctool tool atmos-stack-analyzer        # Multi-env analysis
` + "```" + `

## 🏷️ NAMING CONVENTIONS

Understanding file and reference naming:

- **Evidence Tasks**: ` + "`ET-0001`, `ET-0104`" + ` (4-digit zero-padded)
- **Policies**: ` + "`POL-0001`, `POL-0002`" + ` (4-digit zero-padded)
- **Controls**: ` + "`AC-01`, `CC-01.1`, `SO-19`" + ` (varies by framework)
- **Evidence Files**: ` + "`ET-0001-327992-access_registration.json`" + `

## 📊 KEY CONCEPTS

### Policies
High-level governance documents that define "what" the organization does.
Example: "Access Control Policy", "Data Protection Policy"

### Controls
Specific security requirements that implement policies. Define "how" things are done.
Example: "CC6.8 - Logical access controls restrict access to authorized users"

### Evidence Tasks
Specific evidence that must be collected to prove controls are implemented.
Example: "ET-0047 - GitHub Repository Access Controls - Show team permissions"

### Tools
Automated scanners and analyzers that collect evidence from infrastructure:
- **Terraform Tools** (7 tools) - Infrastructure as Code security
- **GitHub Tools** (6 tools) - Repository access, workflows, security features
- **Google Workspace Tools** - User access, groups, drive permissions
- **Atmos Tools** - Multi-environment stack analysis

## 🔍 COMMON USER QUESTIONS

### "What evidence do I need to collect?"
` + "```bash" + `
grctool tool evidence-task-list
` + "```" + `
This shows all pending evidence tasks with status and assignees.

### "What controls apply to [some system]?"
` + "```bash" + `
# Search through synced controls
grep -r "keyword" {{.DataDir}}/{{.DocsPath}}/controls/

# Or ask me - I can read the control files and explain them
` + "```" + `

### "How do I generate evidence for GitHub access controls?"
` + "```bash" + `
# Use the specialized GitHub permission tool
grctool tool github-permissions --repository owner/repo
` + "```" + `

### "What Terraform security evidence can be collected?"
` + "```bash" + `
# Use the comprehensive indexer for fast queries
grctool tool terraform-security-indexer --query-type control_mapping

# Or use the security analyzer for deep analysis
grctool tool terraform-security-analyzer --security-domain all
` + "```" + `

## 🔐 AUTHENTICATION

GRCTool uses **browser-based authentication** with Tugboat Logic:

` + "```bash" + `
# Login (opens Safari, saves credentials securely)
grctool auth login

# Check status
grctool auth status

# Logout (clears credentials)
grctool auth logout
` + "```" + `

**Note**: Credentials are stored in {{.CacheDir}}/auth/ and are automatically refreshed.

## 🎯 HELPING USERS WITH EVIDENCE

When a user asks for help with evidence collection:

1. **Understand the task**: Read the evidence task JSON file
2. **Identify applicable tools**: Suggest which grctool tools can collect this evidence
3. **Review existing evidence**: Check if evidence already exists
4. **Guide evidence generation**: Walk through tool usage
5. **Help with formatting**: Ensure evidence meets auditor requirements

### Example: Helping with ET-0047 (GitHub Repository Access)

` + "```bash" + `
# 1. Get task details
grctool tool evidence-task-details --task-ref ET-0047

# 2. Check what controls it maps to
# I can read: {{.DataDir}}/{{.DocsPath}}/evidence_tasks/ET-0047-*.json

# 3. Run the appropriate tool
grctool tool github-permissions --repository org/repo --output-format matrix

# 4. Review and format the output for compliance
` + "```" + `

## 📚 GETTING MORE HELP

` + "```bash" + `
# General help
grctool --help

# Command-specific help
grctool sync --help
grctool tool evidence-task-list --help

# Tool-specific help
grctool tool github-permissions --help
` + "```" + `

## ✅ CHECKLIST FOR AI ASSISTANTS

When helping users:
- [ ] Confirm data is synced (` + "`grctool sync`" + `)
- [ ] Verify authentication is valid (` + "`grctool auth status`" + `)
- [ ] Read evidence task files to understand requirements
- [ ] Suggest appropriate tools for evidence collection
- [ ] Help interpret tool output for compliance purposes
- [ ] Ensure evidence is properly documented and formatted
- [ ] Never commit secrets, tokens, or credentials

---

**Configuration Details**
{{if .OrgID}}- Organization ID: {{.OrgID}}{{end}}
- Data Directory: {{.DataDir}}
- Authentication: Browser-based (Safari)
- Tools Available: 29+ evidence collection tools

**Last Updated**: Generated by ` + "`grctool init`" + `
**Regenerate**: Run ` + "`grctool init`" + ` anytime to update this file with current configuration
`

// ClaudeMdData holds the data for template rendering
type ClaudeMdData struct {
	DataDir      string
	DocsPath     string
	EvidencePath string
	PromptsPath  string
	CachePath    string
	CacheDir     string
	OrgID        string
}

// RenderClaudeMd renders the CLAUDE.md template with config values
func RenderClaudeMd(cfg *config.Config) (string, error) {
	// Prepare template data
	data := ClaudeMdData{
		DataDir:      cfg.Storage.DataDir,
		DocsPath:     cfg.Storage.Paths.Docs,
		EvidencePath: cfg.Storage.Paths.Evidence,
		PromptsPath:  cfg.Storage.Paths.Prompts,
		CachePath:    cfg.Storage.Paths.Cache,
		CacheDir:     cfg.Storage.CacheDir,
		OrgID:        cfg.Tugboat.OrgID,
	}

	// Parse and execute template
	tmpl, err := template.New("claude_md").Parse(ClaudeMdTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
