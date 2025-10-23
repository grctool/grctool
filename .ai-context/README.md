# AI Context Cache

This directory contains generated context data optimized for AI agent consumption. The context helps AI agents understand the codebase structure, dependencies, and architectural patterns.

## Generated Files

- **`codebase-summary.json`** - High-level overview with metrics, package structure, and architectural patterns
- **`interfaces.json`** - Comprehensive mapping of Go interfaces and their method signatures
- **`dependencies.json`** - Module and package dependency graph with centrality metrics
- **`recent-changes.json`** - Recent file modifications with semantic context (future)

## Generation

Context is generated using the grctool CLI:

```bash
# Generate all context
make ai-context

# Generate individual context types
./bin/grctool tool context summary
./bin/grctool tool context interfaces  
./bin/grctool tool context deps

# Show codebase statistics
make ai-stats
```

## Usage for AI Agents

These context files provide:

1. **Structural Understanding** - Package organization and file counts
2. **Interface Contracts** - Available interfaces and their method signatures  
3. **Dependency Relationships** - How packages depend on each other
4. **Architectural Patterns** - Layered structure and core modules
5. **Testing Strategy** - 4-tier test organization with VCR framework

## Context Refresh

Context should be regenerated when:
- New packages or interfaces are added
- Major architectural changes occur
- Dependency updates happen
- Before onboarding new AI agents

The context generation is fast (typically <5 seconds) and can be run frequently.

## Cache Management

- **Clean cache**: `make ai-clean`
- **Selective refresh**: Use individual context commands
- **Automated refresh**: Include `make ai-context` in CI/CD pipelines