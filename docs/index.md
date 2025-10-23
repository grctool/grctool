---
title: GRCTool Documentation
tags: [overview, documentation, index, navigation, grctool, compliance, security, cli, helix]
created: 2025-01-10
updated: 2025-01-10
category: overview
search: grctool documentation index navigation guide helix
---

# GRCTool Documentation

Welcome to the comprehensive documentation for **GRCTool** - a CLI application for managing security program compliance through Tugboat Logic integration.

> **📢 Documentation Restructured**: This documentation has been reorganized using the HELIX methodology to better align with the software development lifecycle. See [[HELIX-MIGRATION]] for details.

## What is GRCTool?

GRCTool automates the collection, analysis, and generation of evidence for security compliance frameworks like SOC 2, ISO 27001, and others. It bridges the gap between your infrastructure and compliance requirements by intelligently mapping policies, controls, and evidence tasks.

## Quick Start

**First time using GRCTool?**
- [[reference/cli-commands|🚀 CLI Reference]] - Essential commands and workflows
- [[helix/01-frame/user-stories|👥 User Stories]] - See how GRCTool helps different roles
- [[helix/05-deploy/deployment-operations|📦 Installation]] - Get up and running

**Common Tasks:**
- [[helix/01-frame/compliance-requirements|📋 Compliance Requirements]] - SOC2, ISO27001 frameworks
- [[helix/02-design/system-architecture|🏗️ Architecture]] - Technical system design
- [[helix/03-test/testing-strategy|🧪 Testing]] - Quality assurance approach
- [[helix/04-build/development-practices|⚡ Development]] - Coding standards and practices

## HELIX Documentation Structure

This documentation follows the **HELIX software development lifecycle methodology**, organizing content around six key phases:

### 🎯 01-Frame (Requirements & Vision)
*Define what we're building and why*

- **[[helix/01-frame/product-requirements|Product Requirements]]** - Vision, goals, target audience
- **[[helix/01-frame/user-stories|User Stories]]** - Personas, requirements, acceptance criteria
- **[[helix/01-frame/compliance-requirements|Compliance Requirements]]** - SOC2, ISO27001, security frameworks

### 🏗️ 02-Design (Architecture & Planning)
*Plan how we'll build it*

- **[[helix/02-design/system-architecture|System Architecture]]** - Technical design, patterns, components
- **[[helix/02-design/security-architecture|Security Architecture]]** - Security design, authentication, threat modeling

### 🧪 03-Test (Quality Assurance)
*Ensure it works correctly and securely*

- **[[helix/03-test/testing-strategy|Testing Strategy]]** - Comprehensive testing approach, VCR, mutation testing

### ⚡ 04-Build (Development)
*Write and maintain the code*

- **[[helix/04-build/development-practices|Development Practices]]** - Coding standards, security practices, quality guidelines

### 🚀 05-Deploy (Operations)
*Release and operate the system*

- **[[helix/05-deploy/deployment-operations|Deployment & Operations]]** - Deployment, monitoring, maintenance, performance

### 🔄 06-Iterate (Feedback & Improvement)
*Learn and improve continuously*

- **[[helix/06-iterate/roadmap-feedback|Roadmap & Feedback]]** - Product roadmap, feedback collection, continuous improvement

## Reference Documentation

Essential reference materials that support all HELIX phases:

### 📖 Technical Reference
- **[[reference/api-documentation|API Documentation]]** - REST API and CLI reference
- **[[reference/cli-commands|CLI Commands]]** - Complete command reference with examples
- **[[reference/data-formats|Data Formats]]** - JSON schemas and data structures
- **[[reference/naming-conventions|Naming Conventions]]** - Consistent naming standards
- **[[reference/glossary|Glossary]]** - Terminology and definitions

### 🔧 Operations Reference
*Operational procedures are integrated into the HELIX Deploy phase: [[helix/05-deploy/deployment-operations|Deployment & Operations]]*

### 📊 Strategy Reference
*Strategic planning documentation is integrated into the HELIX Frame and Iterate phases: [[helix/01-frame/product-requirements|Product Requirements]] and [[helix/06-iterate/roadmap-feedback|Roadmap & Feedback]]*

## Key Features

- **🔐 Automated Browser Authentication** - Safari-based login with automatic cookie extraction (macOS)
- **📊 Data Synchronization** - Download policies, controls, and evidence tasks via REST API
- **🤖 AI-Powered Evidence Generation** - Uses Claude AI to intelligently generate compliance evidence
- **🔍 Evidence Analysis** - Maps relationships between evidence tasks, controls, and policies
- **🛡️ Security Control Mapping** - Automated mapping of infrastructure to compliance controls
- **📄 Multiple Output Formats** - Generate evidence in CSV or Markdown formats
- **💾 Local Data Storage** - JSON-based storage for offline access and analysis

## Architecture Overview

```
grctool/
├── cmd/                    # CLI commands
├── internal/
│   ├── auth/              # Browser authentication
│   ├── claude/            # Claude AI integration
│   ├── config/            # Configuration management
│   ├── domain/            # Domain models
│   ├── evidence/          # Evidence generation
│   ├── formatters/        # Output formatters
│   ├── storage/           # Local data storage
│   ├── tools/             # Evidence collection tools
│   └── tugboat/           # Tugboat Logic API client
├── configs/               # Configuration templates
└── docs/                  # Documentation (HELIX structure)
    ├── helix/            # HELIX methodology phases (01-frame through 06-iterate)
    ├── reference/        # Technical reference materials
    ├── assets/           # Diagrams, templates, and supporting materials
    └── controls/         # Compliance control definitions
```

## Document Categories & User Types

### By HELIX Phase
- **Frame**: Product vision, user requirements, compliance needs
- **Design**: Architecture, security design, API specifications
- **Test**: Testing strategy, quality assurance, security testing
- **Build**: Development practices, coding standards, security guidelines
- **Deploy**: Operations, deployment, monitoring, maintenance
- **Iterate**: Roadmap, feedback, continuous improvement

### By User Type
- **Compliance Managers**: Frame (requirements), Iterate (feedback), Reference (compliance)
- **Security Engineers**: Frame (security requirements), Design (security architecture), Build (security practices)
- **DevOps Engineers**: Design (system architecture), Deploy (operations), Reference (CLI commands)
- **Developers**: Design (architecture), Test (testing), Build (practices), Reference (API docs)
- **Product Managers**: Frame (vision), Iterate (roadmap), Strategy (business metrics)
- **Auditors**: Frame (compliance), Reference (evidence formats), Operations (audit procedures)

## Quick Commands Reference

```bash
# Build and run
go build -o bin/grctool
./bin/grctool --help

# Core operations
./bin/grctool sync                                    # Sync from Tugboat API
./bin/grctool tool evidence-task-details --task-ref ET-0001
./bin/grctool tool name-generator --document-type evidence --reference-id ET-0001

# Authentication
./bin/grctool auth login                              # Browser-based login
./bin/grctool auth status                             # Check authentication status

# Evidence generation
./bin/grctool generate evidence --task ET-0001       # Generate evidence for task
./bin/grctool list policies                          # List available policies
./bin/grctool list controls                          # List available controls
```

## Navigation Guide

### For New Users
1. Start with **[[helix/01-frame/product-requirements|Product Requirements]]** to understand GRCTool's purpose
2. Review **[[helix/01-frame/user-stories|User Stories]]** to find your role and use cases
3. Follow **[[reference/cli-commands|CLI Commands]]** for hands-on getting started
4. Reference **[[operations/troubleshooting|Troubleshooting]]** when issues arise

### For Developers
1. **[[helix/02-design/system-architecture|System Architecture]]** - Understand the technical design
2. **[[helix/04-build/development-practices|Development Practices]]** - Follow coding standards
3. **[[helix/03-test/testing-strategy|Testing Strategy]]** - Implement quality assurance
4. **[[reference/api-documentation|API Documentation]]** - Technical reference

### For Compliance Teams
1. **[[helix/01-frame/compliance-requirements|Compliance Requirements]]** - Framework requirements
2. **[[helix/02-design/security-architecture|Security Architecture]]** - Security controls design
3. **[[helix/06-iterate/roadmap-feedback|Roadmap]]** - Future compliance capabilities
4. **[[reference/data-formats|Data Formats]]** - Evidence and report structures

### For Operations Teams
1. **[[helix/05-deploy/deployment-operations|Deployment & Operations]]** - Complete operational guide
2. **[[reference/cli-commands|CLI Commands]]** - Command reference for automation
3. **[[helix/06-iterate/roadmap-feedback|Roadmap & Feedback]]** - Continuous improvement

## Support and Community

- **Issues & Bugs**: Use GitHub Issues for bug reports and feature requests
- **Development**: See **[[helix/04-build/development-practices|Development Practices]]** for contribution guidelines
- **Architecture**: Detailed technical design in **[[helix/02-design/system-architecture|System Architecture]]**
- **Security**: Security guidelines in **[[helix/02-design/security-architecture|Security Architecture]]**

## Search Tags

Use these tags to find related documentation:
`grctool`, `compliance`, `security`, `cli`, `tugboat`, `evidence`, `soc2`, `iso27001`, `automation`, `ai`, `google-workspace`, `terraform`, `github`, `api`, `authentication`, `monitoring`, `deployment`, `testing`, `architecture`, `helix`

## Document Navigation

This documentation uses **wikilinks** (`[[document-name]]`) for seamless navigation in Obsidian, Foam, or compatible markdown viewers. All internal references use the format `[[path/document-name|Display Text]]` for optimal linking.

**Navigation Tips:**
- **HELIX Structure**: Follow the six phases for comprehensive understanding
- **Reference First**: Use reference docs for quick answers
- **Phase Context**: Understand how each phase contributes to the whole
- **Cross-References**: Follow links between related phases
- **Search Tags**: Use tags to find content across phases

---

## Migration Information

This documentation was restructured on **January 10, 2025** using the HELIX methodology. See **[[HELIX-MIGRATION]]** for:
- Complete mapping of old to new content locations
- Migration rationale and benefits
- Rollback procedures if needed
- Cross-reference updates

*This documentation is maintained alongside the codebase to ensure accuracy and completeness. The HELIX structure supports both development workflows and compliance requirements.*