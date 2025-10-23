#!/bin/bash
# Set up Go environment
export GO111MODULE=on
export GOFLAGS="-mod=readonly"

# Create useful aliases
alias gotest='go test -v -race ./...'
alias gobench='go test -bench=. -benchmem'
alias gocover='go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out'

# Function to quickly analyze a package
analyze_pkg() {
    echo "Analyzing package: $1"
    go doc -all "$1" | head -50
    echo "---"
    go list -f '{{.Imports}}' "$1"
}
