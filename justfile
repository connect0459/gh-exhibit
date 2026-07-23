# Setup after clone
setup:
    pre-commit install

# Format all Go source files in place
fmt:
    gofmt -l -w .

# Check formatting without modifying files
fmt-check:
    #!/usr/bin/env bash
    set -euo pipefail
    unformatted=$(gofmt -l .)
    if [ -n "$unformatted" ]; then
        echo "The following files are not gofmt-formatted:"
        echo "$unformatted"
        exit 1
    fi

# Run go vet
vet:
    go vet ./...

# Run golangci-lint via pre-commit
lint:
    pre-commit run golangci-lint --all-files

# Build all packages
build:
    go build ./...

# Run tests with race detector and coverage
test:
    go test ./... -race -cover

# Verify code quality and build
verify: fmt-check vet lint build test
