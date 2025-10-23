# Lefthook Migration - Git Hooks Modernization

## Overview

We've migrated from custom bash pre-commit hooks to **Lefthook**, a fast and powerful Git hooks manager written in Go. This provides unified, performant, and maintainable git hooks for both local development and CI/CD.

## Key Benefits

### ✅ **Unified Tooling**
- Same tool and configuration for local pre-commit and CI/CD
- No more drift between local hooks and CI checks
- Consistent behavior across all environments

### ⚡ **Performance**
- Written in Go - native performance for Go projects
- Parallel execution by default
- Faster than bash scripts with better error handling

### 🔧 **Maintainability**
- YAML configuration instead of complex bash scripts
- Easy to understand and modify
- Better error messages and debugging

### 🎯 **Flexibility**
- Easy to skip specific checks: `LEFTHOOK_EXCLUDE=test,lint git commit`
- Different configurations for different environments
- Simple to add new checks

## Files Created/Modified

### New Configuration Files
- `lefthook.yml` - Main configuration for local pre-commit hooks
- `lefthook-ci.yml` - CI-specific configuration for GitHub Actions
- `scripts/check-file-size.sh` - File size validation script
- `scripts/check-commit-msg.sh` - Commit message validation script

### Updated Files
- `.github/workflows/ci.yml` - New comprehensive CI pipeline using lefthook
- `.github/workflows/testing.yml` - Updated for advanced testing features
- `CLAUDE.md` - Updated documentation with lefthook commands

### Backup Files
- `.git/hooks/pre-commit.old` - Original bash pre-commit hook (backup)
- `.git/hooks/pre-commit.backup` - Additional backup

## Migration Summary

### What Changed
1. **Pre-commit Hook**: Replaced 295-line bash script with structured YAML configuration
2. **CI/CD**: New GitHub Actions workflow using lefthook for consistency
3. **Script Organization**: Extracted reusable scripts to `scripts/` directory
4. **Documentation**: Updated CLAUDE.md with new workflows

### What Stayed the Same
- All existing checks are preserved (formatting, linting, tests, security, secrets)
- Same quality gates and thresholds
- Same skip mechanisms (environment variables)
- Same error reporting and logging

## Usage

### Local Development
```bash
# Hooks run automatically on git commit
git add .
git commit -m "feat: implement new feature"

# Run hooks manually
lefthook run pre-commit

# Skip specific checks
LEFTHOOK_EXCLUDE=test,lint git commit
```

### CI/CD Testing
```bash
# Test CI configuration locally
LEFTHOOK_CONFIG=lefthook-ci.yml lefthook run ci

# Run specific CI check groups
LEFTHOOK_CONFIG=lefthook-ci.yml lefthook run performance
LEFTHOOK_CONFIG=lefthook-ci.yml lefthook run quality
```

### Configuration Management
```bash
# Validate configuration
lefthook validate

# See all configured hooks
lefthook dump

# Reinstall hooks if needed
lefthook install
```

## Checks Included

### Pre-commit (Local)
- ✅ File size limits
- ✅ Go formatting (gofmt)
- ✅ Import organization (goimports)
- ✅ Go vet analysis
- ✅ Build verification
- ✅ Unit tests (short mode)
- ✅ Linting (golangci-lint)
- ✅ Security scanning (gosec)
- ✅ Secret detection
- ✅ Documentation check
- ✅ Debug artifacts detection

### CI Pipeline
- ✅ All pre-commit checks (on all files)
- ✅ Full test suite
- ✅ Integration tests
- ✅ Coverage analysis
- ✅ Multi-platform builds
- ✅ Security SARIF reports
- ✅ Quality gates

### Pre-push (Local)
- ✅ Full test suite
- ✅ Coverage validation

## Troubleshooting

### Common Issues

**Hook not running**: Ensure lefthook is installed
```bash
lefthook install
```

**Configuration errors**: Validate your changes
```bash
lefthook validate
```

**Missing tools**: Install required tools
```bash
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
```

**Skip problematic checks temporarily**:
```bash
LEFTHOOK_EXCLUDE=security,test git commit
```

### Rollback (if needed)
To temporarily rollback to the old system:
```bash
cp .git/hooks/pre-commit.old .git/hooks/pre-commit
```

## Future Enhancements

With lefthook in place, we can easily:
- Add new code quality checks
- Implement pre-push hooks for additional validation
- Create environment-specific configurations
- Add commit message validation rules
- Integrate with additional security tools

## Resources

- [Lefthook Documentation](https://lefthook.dev/)
- [Lefthook GitHub Repository](https://github.com/evilmartians/lefthook)
- [Configuration Examples](https://github.com/evilmartians/lefthook/tree/master/docs)

---

*This migration improves developer experience while maintaining code quality and security standards.*