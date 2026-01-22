---
title: "Deployment and Operations"
phase: "05-deploy"
category: "operations"
tags: ["deployment", "operations", "monitoring", "maintenance", "performance"]
related: ["monitoring-alerting", "performance-optimization", "maintenance-procedures"]
created: 2025-01-10
updated: 2025-01-10
helix_mapping: "Consolidated from 05-Operations/ deployment, monitoring, maintenance"
---

# Deployment and Operations

## Overview

This document covers the complete deployment and operational lifecycle of GRCTool, including build procedures, deployment strategies, monitoring, maintenance, and performance optimization. As a CLI tool, GRCTool follows a distribution-based deployment model rather than traditional service deployment.

## Build and Release Process

### Build Pipeline

#### Local Development Build
```bash
# Quick development build
make build

# Build with version information
make build-release

# Cross-platform builds
make build-all-platforms

# Build with optimizations
go build -ldflags="-s -w" -o bin/grctool
```

#### CI/CD Build Process
```bash
# Full CI pipeline
name: Build and Release

on:
  push:
    tags: ['v*']
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24.12'

      - name: Run tests
        run: make test-ci

      - name: Security scan
        run: make security-scan

      - name: Coverage check
        run: make coverage-check

  build:
    needs: test
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: [linux, darwin, windows]
        arch: [amd64, arm64]

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4

      - name: Build binary
        run: |
          GOOS=${{ matrix.os }} GOARCH=${{ matrix.arch }} \
          go build -ldflags="-s -w -X main.version=${{ github.ref_name }}" \
          -o dist/grctool-${{ matrix.os }}-${{ matrix.arch }}

      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: grctool-${{ matrix.os }}-${{ matrix.arch }}
          path: dist/grctool-*

  release:
    needs: build
    runs-on: ubuntu-latest
    if: startsWith(github.ref, 'refs/tags/')

    steps:
      - name: Download artifacts
        uses: actions/download-artifact@v3

      - name: Create release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            grctool-*/grctool-*
          generate_release_notes: true
```

### Release Management

#### Version Strategy
```bash
# Semantic versioning (SemVer)
v1.0.0    # Major release with breaking changes
v1.1.0    # Minor release with new features
v1.1.1    # Patch release with bug fixes

# Pre-release versions
v1.2.0-rc.1    # Release candidate
v1.2.0-beta.1  # Beta version
v1.2.0-alpha.1 # Alpha version
```

#### Release Checklist
- [ ] All tests pass (unit, integration, functional)
- [ ] Security scan completed with no critical issues
- [ ] Code coverage meets threshold (≥80%)
- [ ] Documentation updated for new features
- [ ] Changelog updated with user-facing changes
- [ ] Version numbers updated in code and documentation
- [ ] Release notes prepared
- [ ] Backwards compatibility verified
- [ ] Performance benchmarks run and verified

## Distribution and Installation

### Distribution Methods

#### GitHub Releases
```bash
# Download latest release
curl -s https://api.github.com/repos/yourorg/grctool/releases/latest | \
  grep "browser_download_url.*linux-amd64" | \
  cut -d '"' -f 4 | \
  wget -qi -

# Make executable
chmod +x grctool-linux-amd64
sudo mv grctool-linux-amd64 /usr/local/bin/grctool
```

#### Package Managers
```bash
# Homebrew (macOS)
brew tap yourorg/grctool
brew install grctool

# Chocolatey (Windows)
choco install grctool

# APT (Ubuntu/Debian)
wget -qO- https://packagecloud.io/yourorg/grctool/gpgkey | sudo apt-key add -
echo "deb https://packagecloud.io/yourorg/grctool/ubuntu/ focal main" | sudo tee /etc/apt/sources.list.d/grctool.list
sudo apt-get update
sudo apt-get install grctool

# YUM (RHEL/CentOS)
sudo yum install -y https://packagecloud.io/yourorg/grctool/packages/el/7/grctool-1.0.0-1.x86_64.rpm
```

#### Container Distribution
```dockerfile
# Dockerfile for containerized distribution
FROM golang:1.24.12-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o grctool .

FROM alpine:latest
RUN apk --no-cache add ca-certificates git
WORKDIR /root/

COPY --from=builder /app/grctool .
COPY configs/grctool.yaml.example .grctool.yaml

CMD ["./grctool"]
```

### Installation Verification
```bash
# Installation test script
#!/bin/bash
set -e

echo "Testing GRCTool installation..."

# Verify binary exists and is executable
if ! command -v grctool &> /dev/null; then
    echo "❌ grctool command not found"
    exit 1
fi

# Check version
version=$(grctool version --short)
echo "✅ GRCTool version: $version"

# Test basic functionality
if grctool auth status > /dev/null 2>&1; then
    echo "✅ Authentication check passed"
else
    echo "⚠️  Authentication not configured (expected for new installations)"
fi

# Test configuration loading
if grctool config validate > /dev/null 2>&1; then
    echo "✅ Configuration validation passed"
else
    echo "⚠️  Configuration validation failed (check .grctool.yaml)"
fi

echo "✅ Installation verification completed"
```

## Configuration Management

### Configuration Files

#### Default Configuration Structure
```yaml
# .grctool.yaml - Main configuration file
tugboat:
  base_url: "${TUGBOAT_BASE_URL}"
  org_id: "${TUGBOAT_ORG_ID}"
  timeout: 30s
  rate_limit: 10
  auth_mode: "browser"

evidence:
  claude:
    api_key: "${CLAUDE_API_KEY}"
    model: "claude-3-sonnet-20240229"
    timeout: 60s
  tools:
    terraform:
      enabled: true
      scan_paths: ["./terraform", "./infra"]
    github:
      enabled: true
      token: "${GITHUB_TOKEN}"

storage:
  data_dir: "./docs"
  cache_dir: "./docs/.cache"

logging:
  level: "info"
  format: "json"
  output: "stdout"

performance:
  max_concurrent_tools: 5
  cache_ttl: "1h"
  request_timeout: "30s"
```

#### Environment-Specific Configurations
```bash
# Development environment
export GRCTOOL_ENV=development
export GRCTOOL_LOG_LEVEL=debug
export GRCTOOL_DATA_DIR=./dev-data

# Staging environment
export GRCTOOL_ENV=staging
export GRCTOOL_LOG_LEVEL=info
export GRCTOOL_DATA_DIR=/opt/grctool/staging-data

# Production environment
export GRCTOOL_ENV=production
export GRCTOOL_LOG_LEVEL=warn
export GRCTOOL_DATA_DIR=/opt/grctool/data
```

### Configuration Validation
```bash
# Configuration validation script
#!/bin/bash

echo "Validating GRCTool configuration..."

# Check configuration file exists
if [[ ! -f ".grctool.yaml" ]]; then
    echo "❌ Configuration file .grctool.yaml not found"
    exit 1
fi

# Validate configuration syntax
if ! grctool config validate; then
    echo "❌ Configuration validation failed"
    exit 1
fi

# Check required environment variables
required_vars=(
    "CLAUDE_API_KEY"
    "TUGBOAT_BASE_URL"
)

missing_vars=()
for var in "${required_vars[@]}"; do
    if [[ -z "${!var}" ]]; then
        missing_vars+=("$var")
    fi
done

if [[ ${#missing_vars[@]} -gt 0 ]]; then
    echo "❌ Missing required environment variables: ${missing_vars[*]}"
    exit 1
fi

# Check optional variables
optional_vars=(
    "GITHUB_TOKEN"
    "TUGBOAT_ORG_ID"
)

for var in "${optional_vars[@]}"; do
    if [[ -z "${!var}" ]]; then
        echo "⚠️  Optional environment variable not set: $var"
    fi
done

echo "✅ Configuration validation completed"
```

## Monitoring and Observability

### Application Metrics

#### Performance Metrics
```go
// metrics.go - Application metrics collection
type Metrics struct {
    EvidenceGenerationDuration prometheus.HistogramVec
    EvidenceGenerationCount    prometheus.CounterVec
    APIRequestDuration         prometheus.HistogramVec
    APIRequestCount           prometheus.CounterVec
    CacheHitRate              prometheus.GaugeVec
}

func NewMetrics() *Metrics {
    return &Metrics{
        EvidenceGenerationDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "grctool_evidence_generation_duration_seconds",
                Help: "Time spent generating evidence",
            },
            []string{"task_id", "tool", "status"},
        ),
        EvidenceGenerationCount: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "grctool_evidence_generation_total",
                Help: "Total number of evidence generation operations",
            },
            []string{"task_id", "tool", "status"},
        ),
    }
}
```

#### Health Checks
```go
// health.go - Health check implementation
type HealthChecker struct {
    storage   storage.Repository
    tugboat   *tugboat.Client
    claude    *claude.Client
}

func (h *HealthChecker) Check(ctx context.Context) map[string]HealthStatus {
    checks := map[string]HealthStatus{
        "storage":   h.checkStorage(ctx),
        "tugboat":   h.checkTugboat(ctx),
        "claude":    h.checkClaude(ctx),
        "config":    h.checkConfig(ctx),
    }

    return checks
}

func (h *HealthChecker) checkStorage(ctx context.Context) HealthStatus {
    if err := h.storage.HealthCheck(ctx); err != nil {
        return HealthStatus{
            Status:  "unhealthy",
            Message: err.Error(),
        }
    }

    return HealthStatus{
        Status:  "healthy",
        Message: "Storage accessible",
    }
}
```

### Logging Strategy

#### Structured Logging
```go
// logging.go - Structured logging implementation
func setupLogging(config *Config) {
    zerolog.TimeFieldFormat = time.RFC3339

    var output io.Writer = os.Stdout
    if config.Logging.Output == "file" {
        file, err := os.OpenFile(config.Logging.File,
            os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            log.Fatal().Err(err).Msg("Failed to open log file")
        }
        output = file
    }

    log.Logger = zerolog.New(output).With().Timestamp().Logger()

    level, err := zerolog.ParseLevel(config.Logging.Level)
    if err != nil {
        level = zerolog.InfoLevel
    }
    zerolog.SetGlobalLevel(level)
}

// Usage in application code
func generateEvidence(ctx context.Context, taskID string) error {
    start := time.Now()

    log.Info().
        Str("task_id", taskID).
        Str("operation", "evidence_generation").
        Msg("Starting evidence generation")

    defer func() {
        log.Info().
            Str("task_id", taskID).
            Dur("duration", time.Since(start)).
            Msg("Evidence generation completed")
    }()

    // Implementation
    return nil
}
```

#### Log Aggregation
```bash
# Log aggregation with ELK stack
# filebeat.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/grctool/*.log
  json.keys_under_root: true
  json.add_error_key: true

output.elasticsearch:
  hosts: ["localhost:9200"]
  index: "grctool-logs-%{+yyyy.MM.dd}"

# logstash.conf
input {
  beats {
    port => 5044
  }
}

filter {
  if [fields][app] == "grctool" {
    json {
      source => "message"
    }

    mutate {
      remove_field => ["message"]
    }
  }
}

output {
  elasticsearch {
    hosts => ["elasticsearch:9200"]
    index => "grctool-logs-%{+YYYY.MM.dd}"
  }
}
```

### Alerting

#### Critical Alerts
```yaml
# alertmanager.yml
route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'grctool-alerts'

receivers:
- name: 'grctool-alerts'
  email_configs:
  - to: 'ops-team@company.com'
    subject: 'GRCTool Alert: {{ .GroupLabels.alertname }}'
    body: |
      {{ range .Alerts }}
      Alert: {{ .Annotations.summary }}
      Description: {{ .Annotations.description }}
      {{ end }}

# prometheus rules
groups:
- name: grctool
  rules:
  - alert: EvidenceGenerationFailure
    expr: rate(grctool_evidence_generation_total{status="failed"}[5m]) > 0.1
    for: 5m
    annotations:
      summary: "High evidence generation failure rate"
      description: "Evidence generation failure rate is {{ $value }} per second"

  - alert: APIHighLatency
    expr: histogram_quantile(0.95, rate(grctool_api_request_duration_seconds_bucket[5m])) > 5
    for: 10m
    annotations:
      summary: "High API latency detected"
      description: "95th percentile latency is {{ $value }} seconds"
```

## Performance Optimization

### Performance Monitoring

#### Continuous Benchmarking
```bash
# benchmark.sh - Continuous performance monitoring
#!/bin/bash

set -e

echo "Running GRCTool performance benchmarks..."

# Create benchmark results directory
mkdir -p benchmarks/$(date +%Y%m%d)
RESULT_FILE="benchmarks/$(date +%Y%m%d)/benchmark-$(date +%H%M%S).txt"

# Run benchmarks
go test -bench=. -benchmem ./... > "$RESULT_FILE"

# Compare with baseline if it exists
if [[ -f "benchmarks/baseline.txt" ]]; then
    echo "Comparing with baseline..."
    benchcmp benchmarks/baseline.txt "$RESULT_FILE"
fi

# Check for performance regressions
if python3 scripts/check-performance-regression.py "$RESULT_FILE"; then
    echo "✅ No performance regressions detected"
else
    echo "❌ Performance regression detected!"
    exit 1
fi
```

#### Resource Usage Monitoring
```go
// monitoring.go - Resource usage monitoring
func monitorResourceUsage() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)

        log.Info().
            Uint64("alloc_mb", bToMb(m.Alloc)).
            Uint64("total_alloc_mb", bToMb(m.TotalAlloc)).
            Uint64("sys_mb", bToMb(m.Sys)).
            Uint32("num_gc", m.NumGC).
            Msg("Memory stats")

        // Trigger garbage collection if memory usage is high
        if m.Alloc > 100*1024*1024 { // 100MB
            runtime.GC()
        }
    }
}

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}
```

### Optimization Strategies

#### Memory Optimization
```go
// Use object pools for frequently allocated objects
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 1024)
    },
}

func processData(data []byte) error {
    buffer := bufferPool.Get().([]byte)
    defer bufferPool.Put(buffer[:0])

    // Process data using pooled buffer
    return nil
}

// Stream large files instead of loading into memory
func processLargeFile(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line

    for scanner.Scan() {
        if err := processLine(scanner.Bytes()); err != nil {
            return err
        }
    }

    return scanner.Err()
}
```

#### Concurrency Optimization
```go
// Optimize concurrent operations with worker pools
func processConcurrently(items []Item, maxWorkers int) error {
    semaphore := make(chan struct{}, maxWorkers)
    errChan := make(chan error, len(items))
    var wg sync.WaitGroup

    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()

            semaphore <- struct{}{} // Acquire
            defer func() { <-semaphore }() // Release

            if err := processItem(item); err != nil {
                errChan <- err
            }
        }(item)
    }

    wg.Wait()
    close(errChan)

    // Check for errors
    for err := range errChan {
        if err != nil {
            return err
        }
    }

    return nil
}
```

## Maintenance Procedures

### Regular Maintenance Tasks

#### Automated Maintenance
```bash
# maintenance.sh - Regular maintenance tasks
#!/bin/bash

set -e

echo "Starting GRCTool maintenance..."

# Clean old cache files (older than 7 days)
echo "Cleaning cache files..."
find ~/.grctool/cache -type f -mtime +7 -delete

# Rotate log files
echo "Rotating log files..."
if [[ -f "/var/log/grctool/grctool.log" ]]; then
    logrotate /etc/logrotate.d/grctool
fi

# Update dependencies
echo "Checking for security updates..."
go list -u -m all | grep -v "indirect" | while read -r line; do
    if echo "$line" | grep -q "=>"; then
        echo "Replace directive found: $line"
    fi
done

# Run security scan
echo "Running security scan..."
govulncheck ./...

# Generate maintenance report
cat > "/tmp/grctool-maintenance-$(date +%Y%m%d).txt" << EOF
GRCTool Maintenance Report
Date: $(date)
Version: $(grctool version --short)

Cache cleanup: Complete
Log rotation: Complete
Security scan: $(govulncheck ./... &>/dev/null && echo "Passed" || echo "Issues found")
Disk usage: $(du -sh ~/.grctool 2>/dev/null || echo "N/A")
EOF

echo "✅ Maintenance completed"
```

#### Manual Maintenance Procedures

1. **Monthly Security Review**
   - Review and rotate API keys
   - Update dependencies
   - Run comprehensive security scan
   - Review access logs for anomalies

2. **Quarterly Performance Review**
   - Analyze performance metrics
   - Review and optimize slow operations
   - Update performance baselines
   - Capacity planning review

3. **Annual Security Audit**
   - Comprehensive security assessment
   - Penetration testing
   - Code security review
   - Compliance verification

### Troubleshooting

#### Common Issues and Solutions

1. **Authentication Failures**
   ```bash
   # Check token validity
   grctool auth status

   # Re-authenticate
   grctool auth logout
   grctool auth login

   # Verify environment variables
   env | grep -E "(CLAUDE|TUGBOAT|GITHUB)"
   ```

2. **Performance Issues**
   ```bash
   # Enable debug logging
   export GRCTOOL_LOG_LEVEL=debug

   # Run with profiling
   grctool --profile-cpu=cpu.prof --profile-mem=mem.prof evidence generate ET-0001

   # Analyze profiles
   go tool pprof cpu.prof
   go tool pprof mem.prof
   ```

3. **Storage Issues**
   ```bash
   # Check disk space
   df -h

   # Verify permissions
   ls -la ~/.grctool/

   # Test storage access
   grctool config validate --check-storage
   ```

### Disaster Recovery

#### Backup Procedures
```bash
# backup.sh - Data backup procedures
#!/bin/bash

BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/grctool/$BACKUP_DATE"

mkdir -p "$BACKUP_DIR"

# Backup configuration
cp ~/.grctool/config/* "$BACKUP_DIR/"

# Backup data directory
if [[ -d "docs" ]]; then
    tar -czf "$BACKUP_DIR/data.tar.gz" docs/
fi

# Backup logs
if [[ -d "/var/log/grctool" ]]; then
    tar -czf "$BACKUP_DIR/logs.tar.gz" /var/log/grctool/
fi

# Create backup metadata
cat > "$BACKUP_DIR/metadata.json" << EOF
{
  "backup_date": "$BACKUP_DATE",
  "grctool_version": "$(grctool version --short)",
  "system_info": "$(uname -a)",
  "backup_size": "$(du -sh $BACKUP_DIR | cut -f1)"
}
EOF

echo "✅ Backup completed: $BACKUP_DIR"
```

#### Recovery Procedures
```bash
# restore.sh - Data recovery procedures
#!/bin/bash

BACKUP_DIR=${1:-"/backup/grctool/latest"}

if [[ ! -d "$BACKUP_DIR" ]]; then
    echo "❌ Backup directory not found: $BACKUP_DIR"
    exit 1
fi

echo "Restoring from backup: $BACKUP_DIR"

# Restore configuration
if [[ -f "$BACKUP_DIR/config.tar.gz" ]]; then
    tar -xzf "$BACKUP_DIR/config.tar.gz" -C ~/.grctool/
fi

# Restore data
if [[ -f "$BACKUP_DIR/data.tar.gz" ]]; then
    tar -xzf "$BACKUP_DIR/data.tar.gz"
fi

# Verify restoration
grctool config validate

echo "✅ Restoration completed"
```

## Production Deployment Checklists

### Pre-Deployment Checklist

#### Security Validation
- [ ] **Code Security Scan**
  ```bash
  # Run comprehensive security analysis
  make security-scan
  gosec -fmt json -out gosec-report.json ./...
  nancy sleuth --exclude-vulnerability-file .nancy-ignore
  ```
  - Zero critical vulnerabilities
  - All high-severity issues resolved or documented
  - Dependencies updated to latest secure versions

- [ ] **Credential Management**
  ```bash
  # Verify no secrets in code
  git secrets --scan-history
  truffleHog --regex --entropy=False .
  ```
  - No hardcoded secrets in source code
  - All API keys using environment variables
  - Credential rotation schedule documented

- [ ] **Authentication Setup**
  ```bash
  # Test authentication flow
  grctool auth logout
  grctool auth login
  grctool auth status
  ```
  - Browser-based authentication working
  - Token refresh mechanisms functional
  - Multi-factor authentication configured

#### Quality Assurance
- [ ] **Test Coverage Verification**
  ```bash
  # Verify test coverage meets requirements
  make coverage-check
  make test-all
  ```
  - Overall coverage ≥ 80%
  - Critical packages ≥ 90% coverage
  - All tests passing in CI/CD

- [ ] **Performance Validation**
  ```bash
  # Run performance benchmarks
  make bench
  make bench-compare
  ```
  - No performance regressions vs baseline
  - Memory usage within acceptable limits
  - API response times under thresholds

- [ ] **Integration Testing**
  ```bash
  # Test external integrations
  make test-integration
  VCR_MODE=record go test ./test/integration/...
  ```
  - Tugboat API integration functional
  - Claude AI integration working
  - GitHub API access validated

#### Compliance Validation
- [ ] **Evidence Collection Testing**
  ```bash
  # Test evidence generation pipeline
  grctool sync
  grctool evidence generate ET-0001
  grctool evidence validate --strict
  ```
  - Evidence generation successful
  - Data quality meets standards
  - Audit trail complete

- [ ] **Data Protection Verification**
  ```bash
  # Test data protection measures
  grctool evidence generate ET-0001 --sensitive-data-test
  ```
  - Sensitive data properly redacted
  - Encryption working for required fields
  - Data retention policies enforced

### Deployment Execution Checklist

#### Build and Package
- [ ] **Version Management**
  ```bash
  # Tag release version
  git tag v1.2.0
  git push origin v1.2.0

  # Verify version embedding
  grctool version
  ```
  - Semantic version tag applied
  - Version information embedded in binary
  - Release notes prepared

- [ ] **Multi-Platform Build**
  ```bash
  # Build for all target platforms
  make build-all-platforms

  # Verify binary integrity
  sha256sum dist/grctool-*
  ```
  - Linux (amd64, arm64) builds successful
  - macOS (amd64, arm64) builds successful
  - Windows (amd64) build successful
  - Binary signatures generated

- [ ] **Package Creation**
  ```bash
  # Create distribution packages
  make package-deb
  make package-rpm
  make package-homebrew
  ```
  - Debian packages created and tested
  - RPM packages created and tested
  - Homebrew formula updated
  - Package dependencies verified

#### Distribution
- [ ] **GitHub Release**
  ```bash
  # Create GitHub release
  gh release create v1.2.0 dist/grctool-* \
    --title "GRCTool v1.2.0" \
    --notes-file RELEASE_NOTES.md
  ```
  - Release created with all artifacts
  - Release notes comprehensive
  - Download links functional

- [ ] **Package Repository Updates**
  ```bash
  # Update package repositories
  packagecloud push myorg/grctool/ubuntu/focal grctool_1.2.0_amd64.deb
  packagecloud push myorg/grctool/el/7 grctool-1.2.0-1.x86_64.rpm
  ```
  - APT repository updated
  - YUM repository updated
  - Package metadata refreshed

- [ ] **Container Distribution**
  ```bash
  # Build and push container images
  docker build -t grctool:v1.2.0 .
  docker tag grctool:v1.2.0 ghcr.io/myorg/grctool:v1.2.0
  docker push ghcr.io/myorg/grctool:v1.2.0
  ```
  - Container images built
  - Images pushed to registry
  - Security scan passed

### Post-Deployment Validation

#### Installation Testing
- [ ] **Fresh Installation Test**
  ```bash
  # Test installation from packages
  sudo apt update && sudo apt install grctool
  grctool version
  grctool config init
  ```
  - Package installation successful
  - Basic functionality working
  - Configuration initialization working

- [ ] **Upgrade Testing**
  ```bash
  # Test upgrade from previous version
  grctool version  # Check current version
  sudo apt upgrade grctool
  grctool version  # Verify new version
  grctool config validate  # Check config compatibility
  ```
  - Upgrade process smooth
  - Configuration preserved
  - Data migration successful

- [ ] **Cross-Platform Verification**
  ```bash
  # Test on different platforms
  # Linux
  ./grctool-linux-amd64 version

  # macOS
  ./grctool-darwin-amd64 version

  # Windows
  grctool-windows-amd64.exe version
  ```
  - All platforms functional
  - Platform-specific features working
  - File permissions correct

#### Functional Validation
- [ ] **Core Functionality Test**
  ```bash
  # Test complete workflow
  grctool auth login
  grctool sync
  grctool evidence generate ET-0001
  grctool evidence review ET-0001
  ```
  - Authentication working
  - Data synchronization successful
  - Evidence generation functional
  - Output quality acceptable

- [ ] **Performance Monitoring**
  ```bash
  # Monitor performance metrics
  time grctool sync
  grctool evidence generate ET-0001 --profile
  ```
  - Performance within expected ranges
  - No memory leaks detected
  - Resource usage acceptable

- [ ] **Error Handling Test**
  ```bash
  # Test error scenarios
  grctool evidence generate INVALID-TASK
  grctool sync --endpoint=https://invalid-url
  ```
  - Error messages clear and actionable
  - Graceful degradation working
  - No sensitive data in error logs

#### Security Validation
- [ ] **Security Posture Check**
  ```bash
  # Verify security measures
  grctool auth status --verbose
  grctool config validate --security-check
  ```
  - Secure defaults applied
  - Encryption working properly
  - Access controls functional

- [ ] **Audit Trail Verification**
  ```bash
  # Check audit logging
  grctool evidence generate ET-0001
  tail -f ~/.grctool/logs/audit.log
  ```
  - All operations logged
  - Audit trail complete
  - Log integrity maintained

### Production Environment Setup

#### Enterprise Deployment Configuration
```yaml
# production.grctool.yaml
tugboat:
  base_url: "https://app.tugboatlogic.com"
  timeout: 60s
  rate_limit: 5
  retry_attempts: 3
  retry_backoff: "exponential"

evidence:
  claude:
    model: "claude-3-sonnet-20240229"
    timeout: 120s
    max_tokens: 4000
  tools:
    terraform:
      enabled: true
      scan_paths: ["/opt/terraform/prod"]
      timeout: 300s
    github:
      enabled: true
      rate_limit: 30
      timeout: 60s

storage:
  data_dir: "/opt/grctool/data"
  cache_dir: "/opt/grctool/cache"
  backup_enabled: true
  backup_schedule: "0 2 * * *"  # Daily at 2 AM

logging:
  level: "info"
  format: "json"
  output: "file"
  file: "/var/log/grctool/grctool.log"
  rotation:
    max_size: "100MB"
    max_files: 10
    max_age: 30

security:
  credential_rotation: "30d"
  session_timeout: "8h"
  max_concurrent_operations: 3
  encryption_at_rest: true

monitoring:
  metrics_enabled: true
  metrics_port: 9090
  health_check_port: 8080
  tracing_enabled: true
  prometheus_endpoint: "/metrics"
```

#### High-Availability Setup
```bash
#!/bin/bash
# ha-setup.sh - High availability deployment setup

set -e

# Create dedicated user
sudo useradd -r -s /bin/false -d /opt/grctool grctool

# Setup directory structure
sudo mkdir -p /opt/grctool/{bin,data,cache,logs,config}
sudo chown -R grctool:grctool /opt/grctool
sudo chmod 750 /opt/grctool

# Install binary
sudo cp grctool-linux-amd64 /opt/grctool/bin/grctool
sudo chown grctool:grctool /opt/grctool/bin/grctool
sudo chmod 755 /opt/grctool/bin/grctool

# Setup configuration
sudo cp production.grctool.yaml /opt/grctool/config/.grctool.yaml
sudo chown grctool:grctool /opt/grctool/config/.grctool.yaml
sudo chmod 600 /opt/grctool/config/.grctool.yaml

# Create systemd service
sudo tee /etc/systemd/system/grctool.service > /dev/null << EOF
[Unit]
Description=GRCTool Evidence Collection Service
After=network.target
Wants=network.target

[Service]
Type=simple
User=grctool
Group=grctool
WorkingDirectory=/opt/grctool
Environment=GRCTOOL_CONFIG=/opt/grctool/config/.grctool.yaml
ExecStart=/opt/grctool/bin/grctool daemon
Restart=always
RestartSec=10
TimeoutStopSec=30
KillMode=mixed

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/grctool/data /opt/grctool/cache /opt/grctool/logs
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictRealtime=true
RestrictSUIDSGID=true

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable grctool
sudo systemctl start grctool

# Setup log rotation
sudo tee /etc/logrotate.d/grctool > /dev/null << EOF
/var/log/grctool/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 grctool grctool
    postrotate
        systemctl reload grctool
    endscript
}
EOF

# Setup monitoring
sudo tee /opt/grctool/monitor.sh > /dev/null << 'EOF'
#!/bin/bash
# Health check script

SERVICE="grctool"
LOG_FILE="/var/log/grctool/health.log"

if systemctl is-active --quiet $SERVICE; then
    # Service is running, check application health
    if curl -sf http://localhost:8080/health > /dev/null; then
        echo "$(date): $SERVICE is healthy" >> $LOG_FILE
        exit 0
    else
        echo "$(date): $SERVICE health check failed" >> $LOG_FILE
        exit 1
    fi
else
    echo "$(date): $SERVICE is not running" >> $LOG_FILE
    exit 2
fi
EOF

sudo chmod +x /opt/grctool/monitor.sh

# Setup cron for health monitoring
echo "*/5 * * * * /opt/grctool/monitor.sh" | sudo crontab -u grctool -

echo "✅ High availability setup completed"
```

#### Compliance Environment Configuration
```bash
#!/bin/bash
# compliance-setup.sh - Compliance-specific configuration

set -e

# Setup audit logging
sudo mkdir -p /var/log/grctool/audit
sudo chown grctool:grctool /var/log/grctool/audit
sudo chmod 750 /var/log/grctool/audit

# Configure auditd for GRCTool operations
sudo tee -a /etc/audit/rules.d/grctool.rules > /dev/null << EOF
# GRCTool audit rules
-w /opt/grctool/bin/grctool -p x -k grctool_execution
-w /opt/grctool/config/.grctool.yaml -p rwa -k grctool_config
-w /opt/grctool/data -p rwa -k grctool_data
-a always,exit -F arch=b64 -S execve -F exe=/opt/grctool/bin/grctool -k grctool_commands
EOF

sudo service auditd restart

# Setup encrypted storage for sensitive evidence
sudo cryptsetup luksFormat /dev/sdb1
sudo cryptsetup luksOpen /dev/sdb1 grctool-encrypted
sudo mkfs.ext4 /dev/mapper/grctool-encrypted
sudo mkdir -p /opt/grctool/secure-storage
sudo mount /dev/mapper/grctool-encrypted /opt/grctool/secure-storage
sudo chown grctool:grctool /opt/grctool/secure-storage

# Add to fstab for automatic mounting
echo "/dev/mapper/grctool-encrypted /opt/grctool/secure-storage ext4 defaults 0 2" | sudo tee -a /etc/fstab

# Setup backup encryption keys
sudo mkdir -p /opt/grctool/keys
sudo openssl rand -base64 32 > /opt/grctool/keys/backup.key
sudo chown grctool:grctool /opt/grctool/keys/backup.key
sudo chmod 600 /opt/grctool/keys/backup.key

# Configure compliance monitoring
sudo tee /opt/grctool/compliance-monitor.sh > /dev/null << 'EOF'
#!/bin/bash
# Compliance monitoring script

COMPLIANCE_LOG="/var/log/grctool/compliance.log"
ERROR_THRESHOLD=5

# Check for authentication failures
AUTH_FAILURES=$(grep -c "authentication failed" /var/log/grctool/grctool.log || echo 0)
if [ $AUTH_FAILURES -gt $ERROR_THRESHOLD ]; then
    echo "$(date): HIGH - $AUTH_FAILURES authentication failures detected" >> $COMPLIANCE_LOG
    # Send alert
    curl -X POST "$SLACK_WEBHOOK" -d "{\"text\": \"GRCTool: High authentication failure rate detected\"}" || true
fi

# Check for unauthorized access attempts
UNAUTH_ACCESS=$(grep -c "permission denied" /var/log/grctool/grctool.log || echo 0)
if [ $UNAUTH_ACCESS -gt 0 ]; then
    echo "$(date): MEDIUM - $UNAUTH_ACCESS unauthorized access attempts" >> $COMPLIANCE_LOG
fi

# Check evidence generation failures
EVIDENCE_FAILURES=$(grep -c "evidence generation failed" /var/log/grctool/grctool.log || echo 0)
if [ $EVIDENCE_FAILURES -gt $ERROR_THRESHOLD ]; then
    echo "$(date): HIGH - $EVIDENCE_FAILURES evidence generation failures" >> $COMPLIANCE_LOG
fi

# Check data integrity
if ! grctool evidence validate --all --quiet; then
    echo "$(date): CRITICAL - Data integrity check failed" >> $COMPLIANCE_LOG
    # Send critical alert
    curl -X POST "$PAGERDUTY_WEBHOOK" -d "{\"message\": \"GRCTool: Data integrity check failed\"}" || true
fi

echo "$(date): Compliance monitoring completed" >> $COMPLIANCE_LOG
EOF

sudo chmod +x /opt/grctool/compliance-monitor.sh

# Setup compliance monitoring cron
echo "0 */6 * * * /opt/grctool/compliance-monitor.sh" | sudo crontab -u grctool -

echo "✅ Compliance environment setup completed"
```

### Rollback Procedures

#### Automated Rollback
```bash
#!/bin/bash
# rollback.sh - Automated rollback procedures

set -e

PREVIOUS_VERSION=${1:-""}
BACKUP_DIR="/opt/grctool/backups"
CURRENT_VERSION=$(grctool version --short)

if [[ -z "$PREVIOUS_VERSION" ]]; then
    echo "Usage: $0 <previous_version>"
    echo "Available versions:"
    ls $BACKUP_DIR/
    exit 1
fi

echo "Rolling back from $CURRENT_VERSION to $PREVIOUS_VERSION"

# Stop service
sudo systemctl stop grctool

# Backup current state
sudo cp -r /opt/grctool/bin "/opt/grctool/backups/rollback-from-$CURRENT_VERSION-$(date +%Y%m%d_%H%M%S)"

# Restore previous version
if [[ -d "$BACKUP_DIR/$PREVIOUS_VERSION" ]]; then
    sudo cp -r "$BACKUP_DIR/$PREVIOUS_VERSION/bin"/* /opt/grctool/bin/
    sudo cp -r "$BACKUP_DIR/$PREVIOUS_VERSION/config"/* /opt/grctool/config/
    sudo chown -R grctool:grctool /opt/grctool/bin /opt/grctool/config
else
    echo "❌ Backup for version $PREVIOUS_VERSION not found"
    exit 1
fi

# Verify rollback
if /opt/grctool/bin/grctool version --short | grep -q "$PREVIOUS_VERSION"; then
    echo "✅ Rollback successful"

    # Start service
    sudo systemctl start grctool

    # Verify service health
    sleep 10
    if sudo systemctl is-active --quiet grctool; then
        echo "✅ Service started successfully"
    else
        echo "❌ Service failed to start after rollback"
        sudo systemctl status grctool
        exit 1
    fi
else
    echo "❌ Rollback verification failed"
    exit 1
fi

echo "✅ Rollback to $PREVIOUS_VERSION completed successfully"
```

#### Emergency Procedures
```bash
#!/bin/bash
# emergency-stop.sh - Emergency shutdown procedures

echo "🚨 Emergency shutdown initiated"

# Stop all GRCTool services immediately
sudo systemctl stop grctool
sudo pkill -f grctool

# Disable automatic restart
sudo systemctl disable grctool

# Secure sensitive data
sudo chmod 000 /opt/grctool/config/.grctool.yaml
sudo chmod 000 /opt/grctool/keys/

# Create incident log
echo "$(date): Emergency shutdown executed" >> /var/log/grctool/incidents.log
echo "Reason: ${1:-'Not specified'}" >> /var/log/grctool/incidents.log
echo "Operator: $(whoami)" >> /var/log/grctool/incidents.log

# Send emergency notification
curl -X POST "$EMERGENCY_WEBHOOK" -d "{\"text\": \"🚨 GRCTool emergency shutdown executed. Reason: ${1:-'Not specified'}\"}" || true

echo "✅ Emergency shutdown completed"
echo "To restore: sudo systemctl enable grctool && sudo systemctl start grctool"
```

## References

- [[monitoring-alerting]] - Detailed monitoring and alerting setup
- [[performance-optimization]] - Performance tuning and optimization
- [[maintenance-procedures]] - Comprehensive maintenance guidelines
- [[security-architecture]] - Security operational procedures
- [[compliance-deployment-guide]] - Compliance-specific deployment patterns
- [[high-availability-setup]] - Enterprise HA configuration
- [[disaster-recovery-procedures]] - Business continuity planning

---

*This comprehensive deployment and operations guide with production checklists ensures GRCTool can be reliably deployed, monitored, and maintained in enterprise environments while meeting strict compliance, security, and availability requirements.*
