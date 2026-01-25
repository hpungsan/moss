# ================================================================
# Makefile for Moss

# Directories
BIN_DIR := bin
COVERAGE_DIR := coverage
BINARY := $(BIN_DIR)/moss
PKG := ./cmd/moss

# Version (override with: make build-release VERSION=1.0.0)
VERSION ?= dev
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

# Default target
.PHONY: all
all: build

# -------------------------------------------------------------------
# Build
# -------------------------------------------------------------------
.PHONY: build build-release install
build:
	@echo "Building Moss -> $(BINARY)"
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) $(PKG)
	@echo "✔ Build complete: $(BINARY)"

build-release:
	@echo "Building Moss $(VERSION) -> $(BINARY)"
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BINARY) $(PKG)
	@echo "✔ Release build complete: $(BINARY) (version: $(VERSION))"

install:
	@echo "Installing Moss $(VERSION) to GOPATH/bin..."
	go install $(LDFLAGS) $(PKG)
	@echo "✔ Installed: $$(which moss || echo 'moss (check GOPATH/bin is in PATH)')"

# -------------------------------------------------------------------
# Test
# -------------------------------------------------------------------
.PHONY: test test-verbose test-fresh test-race
test:
	go test ./...

test-verbose:
	go test -v ./...

test-fresh:
	@echo "Clearing test cache and running verbose tests..."
	go clean -testcache
	go test -v -count=1 ./...

test-race:
	@echo "Running tests with race detection (fresh, shuffled)..."
	CGO_ENABLED=1 go test -race -count=1 -shuffle=on ./...

# -------------------------------------------------------------------
# Coverage
# -------------------------------------------------------------------
.PHONY: cover cover-html
cover:
	@mkdir -p $(COVERAGE_DIR)
	@echo "Running tests with coverage..."
	go test -covermode=atomic -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@echo "Coverage summary:"
	go tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -n 10

cover-html: cover
	@echo "Generating HTML coverage report..."
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "✔ Open $(COVERAGE_DIR)/coverage.html in your browser"

# -------------------------------------------------------------------
# Format
# -------------------------------------------------------------------
.PHONY: fmt fmt-check
fmt:
	@echo "Formatting Go code..."
	gofmt -w .
	@echo "✔ Formatting complete"

fmt-check:
	@echo "Checking Go code formatting..."
	@if [ $$(gofmt -l . | wc -l) -ne 0 ]; then \
		echo "ERROR: Code is not formatted. Run 'make fmt' to fix."; \
		gofmt -d .; \
		exit 1; \
	fi
	@echo "✔ All code is properly formatted"

# -------------------------------------------------------------------
# Lint & Security
# -------------------------------------------------------------------
.PHONY: vet lint vulncheck check
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✔ Vet checks passed"

lint:
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout=5m
	@echo "✔ Lint checks passed"

vulncheck: build
	@echo "Running govulncheck..."
	@govulncheck -mode=binary $(BINARY) || govulncheck ./...
	@echo "✔ Vulnerability check passed"

check: fmt-check vet lint vulncheck
	@echo "✔ All pre-commit checks passed"

# -------------------------------------------------------------------
# Clean
# -------------------------------------------------------------------
.PHONY: clean
clean:
	@echo "Cleaning build and coverage artifacts..."
	rm -rf $(BIN_DIR) $(COVERAGE_DIR)
	@echo "✔ Clean complete"

# -------------------------------------------------------------------
# Help
# -------------------------------------------------------------------
.PHONY: help
help:
	@echo "Available targets:"
	@echo ""
	@echo "  # Build"
	@echo "  make build         - Build moss binary (dev)"
	@echo "  make build-release - Build with version (VERSION=1.0.0)"
	@echo "  make install       - Install to GOPATH/bin"
	@echo ""
	@echo "  # Test"
	@echo "  make test        - Run all tests"
	@echo "  make test-verbose - Run tests with verbose output"
	@echo "  make test-fresh  - Run tests (no cache)"
	@echo "  make test-race   - Run tests with race detection"
	@echo ""
	@echo "  # Coverage"
	@echo "  make cover       - Run tests with coverage (summary)"
	@echo "  make cover-html  - Generate HTML coverage report"
	@echo ""
	@echo "  # Format"
	@echo "  make fmt         - Format all Go code"
	@echo "  make fmt-check   - Check formatting (CI-friendly)"
	@echo ""
	@echo "  # Lint & Security"
	@echo "  make vet         - Run go vet"
	@echo "  make lint        - Run golangci-lint"
	@echo "  make vulncheck   - Run govulncheck"
	@echo "  make check       - Run all checks (fmt, vet, lint, vulncheck)"
	@echo ""
	@echo "  # Utility"
	@echo "  make clean       - Remove build and coverage artifacts"
