# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Complete evidence submission system for Tugboat Logic API integration
- Terraform git hash tracking to evidence reports for version control
- VCR recording for GitHub integration tests with rate limiting support
- Enhanced evidence task analysis with comprehensive derivation and mapping
- GitHub Search API rate limiting and `gh auth` integration for authentication
- OS-appropriate default log file paths for cross-platform compatibility
- Markdown document generation for evidence tasks
- Smart tab completion for task, policy, and control reference IDs
- Clickable Tugboat URLs in evidence task list for easier navigation
- Auto-extraction of organization ID during authentication flow
- Assignee and AEC status embeds to evidence task synchronization

### Changed
- Overhauled CLAUDE.md for improved user-focused AI assistance
- Migrated GitHub tools to environment-based VCR configuration
- Completed VCR config migration to environment-based approach for better testing
- Reorganized storage paths with configurable type/format structure
- Simplified config initialization for improved user experience
- Standardized evidence task reference IDs with collection type column

### Fixed
- Framework code JSON mapping for Tugboat API compatibility
- Organization ID inclusion in evidence task URLs for correct navigation
- Integration test failures with VCR playback support
- Integration test expectations to match VCR cassette data

### Chore
- Added `.gitignore` entries for integration test generated artifacts
- Added VCR regeneration script and test documentation to `.gitignore`
- Updated integration test fixtures with latest run data
- Applied Go fmt formatting improvements across codebase
- Added test expectation fix utility script

### Testing
- Enhanced testing infrastructure and development documentation
- Updated integration test data and expectations for accuracy
- Relaxed integration test expectations to match VCR cassette data

## [0.1.0] - Initial Development

### Added
- Browser-based Safari authentication for macOS
- Tugboat Logic API client with OAuth2 support
- Policy, control, and evidence task synchronization
- JSON-based local data storage with caching
- Claude AI integration for evidence generation
- 29 evidence collection tools:
  - 7 Terraform tools (security analyzer, indexer, snippet extractor, etc.)
  - 6 GitHub tools (permissions, workflows, reviews, security features)
  - Google Workspace integration tools
  - Evidence assembly and utility tools
- CLI interface with Cobra framework
- Comprehensive logging with configurable levels
- Configuration management with YAML support
- Three-tier testing strategy (unit, integration, functional)
- VCR-based integration testing infrastructure
- GitHub Actions CI/CD pipeline
- Security control mapping for SOC 2 and ISO 27001
- Output formatters (CSV, Markdown, JSON)
- High-performance indexing with caching
- Tab completion for Bash, Zsh, and Fish shells

### Developer Features
- golangci-lint configuration with comprehensive rules
- Makefile with build, test, and development targets
- Hot reload development mode with Air
- Mutation testing support
- Benchmark testing infrastructure
- Code coverage reporting
- Security scanning with gosec
- Left-hook Git hooks for quality checks

---

## Release Naming Convention

Starting with version 1.0.0, releases will follow semantic versioning:

- **MAJOR** version: Incompatible API changes
- **MINOR** version: New functionality in a backwards-compatible manner
- **PATCH** version: Backwards-compatible bug fixes

## How to Update This Changelog

When contributing, please add your changes to the `[Unreleased]` section under the appropriate category:

- **Added** - New features
- **Changed** - Changes in existing functionality
- **Deprecated** - Soon-to-be removed features
- **Removed** - Removed features
- **Fixed** - Bug fixes
- **Security** - Security vulnerability fixes

Example:
```markdown
### Added
- New evidence collection tool for AWS CloudTrail analysis
```

## Links

- [Compare Unreleased Changes](https://github.com/grctool/grctool/compare/v0.1.0...HEAD)

---

**Note**: This project is currently in active development (pre-1.0). The API and features may change between releases. Once we reach version 1.0.0, we will maintain backwards compatibility according to semantic versioning principles.
