# Security Program Manager Makefile

# Build variables
BINARY_NAME=grctool
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# Directories
BUILD_DIR=build
DIST_DIR=dist

.PHONY: help build build-all clean test test-integration test-all test-race test-coverage lint fmt vet deps install run dev
.PHONY: coverage-report coverage-check coverage-badge coverage-critical coverage-monitor
.PHONY: test-e2e test-e2e-github test-e2e-tugboat test-e2e-audit test-e2e-performance test-e2e-config test-e2e-quick test-e2e-comprehensive test-all-comprehensive
.PHONY: mutation-test mutation-report mutation-quick mutation-dry-run mutation-baseline
.PHONY: bench bench-compare bench-profile bench-save bench-memory bench-report

help: ## Show this help message
	@echo "Security Program Manager - Available commands:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

fmt: ## Format Go code
	go fmt ./...

vet: ## Vet Go code
	go vet ./...

lint: ## Run golangci-lint
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

##@ Building

build: deps ## Build the binary for current platform
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

build-test: deps ## Build the binary specifically for testing
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

build-all: deps ## Build binaries for all platforms
	@mkdir -p $(DIST_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe .

install: test-unit build ## Run unit tests, build, and install to ~/.local/bin
	@mkdir -p ~/.local/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) ~/.local/bin/
	@echo "✅ Installed $(BINARY_NAME) to ~/.local/bin/"

##@ Testing

# Fast unit tests (2-3 seconds) - no auth required
test-unit: ## Run fast unit tests with no external dependencies
	go test -timeout=30s -v ./internal/... ./cmd/... -count=1

# Integration tests - VCR recordings, no live APIs, tests auth logic
test-integration: ## Run integration tests with VCR recordings
	VCR_MODE=playback go test -tags=integration -timeout=2m -v ./internal/... ./test/integration/... -count=1

# Functional tests - CLI testing with built binary
test-functional: build-test ## Run functional CLI tests with built binary
	go test -tags="functional" -timeout=5m -v ./test/functional/... -count=1

test-functional-evidence: build-test ## Run evidence collection functional tests
	go test -tags="functional" -timeout=5m -v ./test/functional/... -run TestEvidenceCollection -count=1

test-functional-workflows: build-test ## Run CLI workflow functional tests
	go test -tags="functional" -timeout=5m -v ./test/functional/... -run TestCompleteAuditWorkflow -count=1

test-functional-errors: build-test ## Run CLI error handling functional tests
	go test -tags="functional" -timeout=5m -v ./test/functional/... -run TestCLI_.*Error -count=1

test-functional-performance: build-test ## Run performance functional tests
	go test -tags="functional" -timeout=10m -v ./test/functional/... -run Test.*Performance -count=1

# E2E tests - real APIs with authentication
test-e2e: build ## Run end-to-end tests with real APIs (requires auth)
	go test -tags="e2e" -timeout=10m -v ./test/e2e/... -count=1

test-e2e-github: build ## Run GitHub API E2E tests only
	go test -tags="e2e" -timeout=10m -v ./test/e2e/github_auth_test.go -count=1

test-e2e-tugboat: build ## Run Tugboat API E2E tests only
	go test -tags="e2e" -timeout=10m -v ./test/e2e/tugboat_auth_test.go -count=1

test-e2e-audit: build ## Run complete audit scenario E2E tests
	go test -tags="e2e" -timeout=15m -v ./test/e2e/audit_scenario_e2e_test.go -count=1

test-e2e-performance: build ## Run performance E2E tests
	go test -tags="e2e" -timeout=20m -v ./test/e2e/performance_e2e_test.go -count=1

test-e2e-config: ## Validate E2E test environment configuration
	go test -tags="e2e" -timeout=5m -v ./test/e2e/config_test.go -count=1

test-e2e-quick: build ## Run quick E2E tests (basic functionality only)
	go test -tags="e2e" -timeout=5m -v ./test/e2e/... -run "TestGitHubPermissions_RealAPI|TestTugboatAuthentication_Status" -count=1

test-e2e-comprehensive: build ## Run comprehensive E2E tests with all optional categories
	@echo "Running comprehensive E2E tests with all categories enabled..."
	LARGE_REPO_TEST=1 TEST_MULTIPLE_REPOS=1 TEST_CONCURRENCY=1 TEST_BULK_SYNC=1 \
	TEST_MEMORY_USAGE=1 TEST_TOOL_SCALABILITY=1 TEST_DATA_PROCESSING_SPEED=1 \
	TEST_RATE_LIMITS=1 TEST_TIMEOUTS=1 TEST_RATE_LIMIT_PERFORMANCE=1 \
	TEST_QUARTERLY_REVIEW=1 TEST_DATA_CONSISTENCY=1 TEST_DATA_INTEGRITY=1 \
	go test -tags="e2e" -timeout=30m -v ./test/e2e/... -count=1

# Fast feedback for development (per AGENTS.md)
test-no-auth: test-unit test-integration ## Run all tests without authentication
	@echo "✅ All tests without authentication passed"

# Complete test suite
test-all: test-unit test-integration test-functional test-e2e ## Run complete test suite
	@echo "✅ Complete test suite passed"

test-all-comprehensive: test-unit test-integration test-functional test-e2e-comprehensive ## Run complete test suite with all E2E categories
	@echo "✅ Complete comprehensive test suite passed"

# Coverage reporting
test-coverage: ## Generate coverage report for unit tests
	go test -coverprofile=coverage.out -tags="!e2e,!functional" ./internal/... ./cmd/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-all: ## Generate coverage report for all tests
	go test -coverprofile=coverage-all.out ./...
	go tool cover -html=coverage-all.out -o coverage-all.html
	@echo "Complete coverage report generated: coverage-all.html"

# Enhanced Coverage Analysis
coverage-report: ## Generate detailed HTML coverage report with analysis
	@echo "🔍 Generating comprehensive coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"
	@echo "📊 Overall coverage:" && go tool cover -func=coverage.out | tail -1

coverage-check: ## Check if coverage meets minimum 80% threshold
	@echo "🔍 Checking coverage thresholds..."
	@go test -coverprofile=coverage.out ./... > /dev/null 2>&1 || true
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "📊 Overall coverage: $${COVERAGE}%"; \
	if [ $$(echo "$${COVERAGE} < 80" | bc -l 2>/dev/null || echo "1") -eq 1 ]; then \
		echo "❌ Coverage ($${COVERAGE}%) is below 80% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage meets 80% threshold"; \
	fi

coverage-badge: ## Generate coverage badge for README
	@echo "🏷️  Generating coverage badge..."
	@mkdir -p docs/coverage
	@go test -coverprofile=coverage.out ./... > /dev/null 2>&1 || true
	python3 scripts/generate-badge.py --coverage-file coverage.out --output docs/coverage/badge.svg
	@echo "✅ Coverage badge generated: docs/coverage/badge.svg"

coverage-critical: ## Check coverage for critical packages only
	@echo "🚨 Checking critical package coverage..."
	@./scripts/coverage-monitor.sh || (echo "❌ Critical package coverage check failed" && exit 1)

coverage-monitor: coverage-critical ## Run comprehensive coverage monitoring and analysis
	@echo "📈 Running comprehensive coverage analysis..."
	@echo "📊 Critical packages analysis complete - see above for details"

# VCR recording modes (manual - requires real credentials)
test-record: ## Record new VCR cassettes for integration tests (requires gh auth)
	@echo "🔑 Getting GitHub token from gh auth..."
	@if ! command -v gh >/dev/null 2>&1; then \
		echo "❌ Error: gh CLI not found. Install it with: brew install gh"; \
		exit 1; \
	fi
	@if ! gh auth status >/dev/null 2>&1; then \
		echo "❌ Error: gh auth not configured. Run: gh auth login"; \
		exit 1; \
	fi
	GITHUB_TOKEN=$$(gh auth token) VCR_MODE=record go test ./test/integration/...

test-record-missing: ## Record only missing VCR cassettes (skips existing) - SLOW to avoid rate limits
	@echo "🔑 Getting GitHub token from gh auth..."
	@if ! command -v gh >/dev/null 2>&1; then \
		echo "❌ Error: gh CLI not found. Install it with: brew install gh"; \
		exit 1; \
	fi
	@if ! gh auth status >/dev/null 2>&1; then \
		echo "❌ Error: gh auth not configured. Run: gh auth login"; \
		exit 1; \
	fi
	@echo "📼 Recording only missing cassettes (existing cassettes will be reused)..."
	@echo "⏱️  Using VERY slow rate limiting (5 req/min) to avoid GitHub's anti-scraping detection..."
	@echo "⏳ This will take ~10 minutes to complete. Be patient - you only do this once!"
	GITHUB_TOKEN=$$(gh auth token) GITHUB_RATE_LIMIT=5 VCR_MODE=record_once go test ./test/integration/...

test-playback: ## Playback existing VCR cassettes (same as test-integration)
	VCR_MODE=playback go test ./test/integration/...

# Mutation Testing with Gremlins
mutation-test: deps ## Run full mutation testing on all packages using Gremlins
	@echo "Running mutation testing with Gremlins on all packages..."
	@which gremlins > /dev/null || (echo "Installing Gremlins..." && go install github.com/go-gremlins/gremlins/cmd/gremlins@latest)
	./scripts/gremlins-mutation-test.sh

mutation-report: ## Generate HTML mutation report from existing results
	@echo "Generating mutation testing reports..."
	./scripts/gremlins-mutation-test.sh --report-only

mutation-quick: ## Quick mutation test on critical and standard packages only
	@echo "Running quick mutation testing on critical packages..."
	@which gremlins > /dev/null || (echo "Installing Gremlins..." && go install github.com/go-gremlins/gremlins/cmd/gremlins@latest)
	./scripts/gremlins-mutation-test.sh --quick

mutation-dry-run: ## Analyze mutations without running tests (fast)
	@echo "Running mutation analysis (dry-run)..."
	@which gremlins > /dev/null || (echo "Installing Gremlins..." && go install github.com/go-gremlins/gremlins/cmd/gremlins@latest)
	./scripts/gremlins-mutation-test.sh --dry-run

mutation-baseline: deps ## Establish baseline mutation scores for all packages
	@echo "Establishing mutation testing baseline..."
	@mkdir -p mutation-reports
	@which gremlins > /dev/null || (echo "Installing Gremlins..." && go install github.com/go-gremlins/gremlins/cmd/gremlins@latest)
	./scripts/gremlins-mutation-test.sh > mutation-reports/baseline-$(shell date +%Y%m%d-%H%M%S).log 2>&1 || true
	@echo "Baseline established. Check mutation-reports/ for results."

# Legacy test commands (for backward compatibility)
test: test-no-auth ## Run unit and integration tests (no authentication required) - LEGACY

test-auth: test-e2e ## Run only tests that require authentication - LEGACY

test-race: ## Run unit tests with race detection
	go test -short -v -race ./...

# CI targets
ci: deps fmt vet lint test-no-auth security-scan ## Run CI checks without authentication
	@echo "✅ CI checks passed"

ci-with-auth: ci test-functional test-e2e ## Run complete CI with authentication
	@echo "✅ Complete CI with authentication passed"

##@ Running

run: build ## Build and run the application
	./$(BUILD_DIR)/$(BINARY_NAME)

config-init: build ## Initialize configuration file
	./$(BUILD_DIR)/$(BINARY_NAME) config init

config-example: build ## Create example configuration
	./$(BUILD_DIR)/$(BINARY_NAME) config init --output=configs/example.yaml --force

##@ Docker

docker-build: ## Build Docker image
	docker build -t grctool:$(VERSION) .

docker-run: docker-build ## Run in Docker container
	docker run --rm -it \
		-v $(PWD)/configs:/app/configs \
		grctool:$(VERSION)

##@ Cleanup

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR) $(DIST_DIR) coverage.out coverage.html


##@ Documentation

docs: ## Generate documentation
	@echo "Generating documentation..."
	@mkdir -p docs
	go doc -all > docs/api.txt
	@echo "Documentation generated in docs/"

##@ Security

security-scan: ## Run security scan with gosec
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...

vulnerability-check: ## Check for known vulnerabilities
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

##@ CI/CD

ci: deps fmt vet lint test-no-auth security-scan ## Run all CI checks (no authentication required)

ci-with-auth: deps fmt vet lint test-all security-scan ## Run all CI checks including authenticated tests

ci-build: ci build-all ## Run CI checks and build all platforms

release-prep: clean ci-build ## Prepare release artifacts
	@echo "Release artifacts ready in $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/

##@ Run usage

auth: build ## Example: Run sync command
	./$(BUILD_DIR)/$(BINARY_NAME) auth login

sync: build ## Example: Run sync command
	./$(BUILD_DIR)/$(BINARY_NAME) sync --verbose

evidence-list: build ## Example: List evidence tasks
	./$(BUILD_DIR)/$(BINARY_NAME) evidence list --status pending

prompts: build ## Example: Generate evidence assembly contexts for all tasks
	./$(BUILD_DIR)/$(BINARY_NAME) evidence generate --all --context-only

evidence: build ## Example: Generate evidence for all tasks
	./$(BUILD_DIR)/$(BINARY_NAME) evidence generate --all

terraform-scan: build ## Example: Scan Terraform configurations
	./$(BUILD_DIR)/$(BINARY_NAME) terraform scan --verbose

##@ AI Development

ai-context: build ## Generate AI agent context cache
	@echo "Generating AI agent context..."
	@mkdir -p .ai-context
	./$(BUILD_DIR)/$(BINARY_NAME) tool context all --output-dir=.ai-context --quiet
	@echo "✓ AI context cache generated in .ai-context/"

ai-context-summary: build ## Generate codebase summary for AI agents
	./$(BUILD_DIR)/$(BINARY_NAME) tool context summary --pretty

ai-context-interfaces: build ## Generate interface mapping for AI agents
	./$(BUILD_DIR)/$(BINARY_NAME) tool context interfaces --pretty

ai-context-deps: build ## Generate dependency graph for AI agents
	./$(BUILD_DIR)/$(BINARY_NAME) tool context deps --pretty

ai-review: fmt lint vet test-no-auth ## Pre-flight checks for AI code development
	@echo "Running AI code review checks..."
	@git status --porcelain | head -10 || true
	@git log --oneline -5 || true
	@echo "✓ Code ready for AI agent development"

ai-diff: ## Show recent changes for AI context
	@echo "Recent file changes:"
	@git diff --name-only HEAD~3..HEAD 2>/dev/null | head -10 || echo "No git history available"
	@echo ""
	@echo "Modified Go files:"
	@find . -name "*.go" -newer .git/COMMIT_EDITMSG 2>/dev/null | head -10 || echo "No recently modified Go files"

ai-stats: build ## Show codebase statistics for AI agents
	@echo "Codebase Statistics:"
	@echo "Go files: $(shell find . -name '*.go' -not -path './vendor/*' | wc -l)"
	@echo "Test files: $(shell find . -name '*_test.go' -not -path './vendor/*' | wc -l)"
	@echo "Packages: $(shell find ./internal -type d -maxdepth 1 | wc -l)"
	@echo "Dependencies: $(shell go list -m all 2>/dev/null | wc -l || echo 'N/A')"
	@echo "LOC: $(shell find . -name '*.go' -not -path './vendor/*' -exec wc -l {} + | tail -1 | awk '{print $$1}' || echo 'N/A')"

ai-clean: ## Clean AI context cache
	@echo "Cleaning AI context cache..."
	rm -rf .ai-context
	@echo "✓ AI context cache cleaned"

##@ Performance

bench: ## Run all benchmarks
	@echo "🏃 Running all benchmarks..."
	@mkdir -p benchmarks
	go test -bench=. -benchmem -cpu=1 -count=3 \
		./internal/tugboat \
		./internal/tools \
		./internal/auth \
		./internal/services \
		./internal/storage \
		2>&1 | tee benchmarks/current.txt
	@echo "✅ Benchmarks completed - results saved to benchmarks/current.txt"

bench-compare: ## Compare with baseline benchmarks
	@echo "📊 Comparing benchmarks with baseline..."
	./scripts/benchmark-compare.sh -r
	@echo "✅ Benchmark comparison completed"

bench-profile: ## Generate CPU and memory profiles
	@echo "📈 Generating performance profiles..."
	@mkdir -p benchmarks/profiles
	go test -bench=BenchmarkTugboatClient_Sync -cpuprofile=benchmarks/profiles/cpu.prof \
		-memprofile=benchmarks/profiles/mem.prof \
		-benchmem ./internal/tugboat
	@echo "CPU profile: benchmarks/profiles/cpu.prof"
	@echo "Memory profile: benchmarks/profiles/mem.prof"
	@echo "View with: go tool pprof benchmarks/profiles/cpu.prof"

bench-save: ## Save current benchmarks as baseline
	@echo "💾 Saving current benchmarks as baseline..."
	./scripts/benchmark-compare.sh -s
	@echo "✅ Baseline saved"

bench-memory: ## Run memory-focused benchmarks
	@echo "💾 Running memory-focused benchmarks..."
	@mkdir -p benchmarks
	go test -bench=BenchmarkLargeFileProcessing -benchmem -memprofile=benchmarks/memory.prof \
		./internal/storage
	go test -bench=BenchmarkAuth_MemoryAllocation -benchmem \
		./internal/auth
	go test -bench=BenchmarkDataService_MemoryIntensive -benchmem \
		./internal/services
	@echo "✅ Memory benchmarks completed"

bench-report: ## Generate comprehensive benchmark report
	@echo "📋 Generating comprehensive benchmark report..."
	@./scripts/benchmark-compare.sh -c
	@echo "✅ Benchmark report generated - check benchmarks/comparison-report.txt"

##@ Information

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

info: version ## Show build information
	@echo "Binary Name: $(BINARY_NAME)"
	@echo "Build Directory: $(BUILD_DIR)"
	@echo "Distribution Directory: $(DIST_DIR)"
	@go version
