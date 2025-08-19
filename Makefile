# Makefile for genifest - Kubernetes manifest generation tool
# Copyright 2025 Qubling LLC

# ===== Configuration =====
BINARY_NAME := genifest
PACKAGE := github.com/zostay/genifest
VERSION_FILE := internal/cmd/version.txt
VERSION := $(shell cat $(VERSION_FILE) 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -ldflags "-X $(PACKAGE)/internal/cmd.Version=$(VERSION) -X $(PACKAGE)/internal/cmd.Commit=$(COMMIT) -X $(PACKAGE)/internal/cmd.BuildTime=$(BUILD_TIME)"
BUILD_FLAGS := $(LDFLAGS)

# Go configuration
GO := go
GOFMT := gofmt
GOLANGCI_LINT := golangci-lint

# Directories
BUILD_DIR := build
DIST_DIR := dist
EXAMPLES_DIR := examples

# Platform targets for cross-compilation
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
RESET := \033[0m

# ===== Default Target =====
.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)Genifest Development Makefile$(RESET)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(RESET)"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)Examples:$(RESET)"
	@echo "  make build          # Build the binary"
	@echo "  make test           # Run all tests"
	@echo "  make lint           # Run linters"
	@echo "  make install        # Install to GOPATH/bin"
	@echo "  make release        # Build release binaries"

# ===== Development Commands =====
.PHONY: build
build: ## Build the binary
	@echo "$(BLUE)Building $(BINARY_NAME)...$(RESET)"
	$(GO) build $(BUILD_FLAGS) -o $(BINARY_NAME) .

.PHONY: build-debug
build-debug: ## Build the binary with debug symbols
	@echo "$(BLUE)Building $(BINARY_NAME) with debug symbols...$(RESET)"
	$(GO) build -gcflags="all=-N -l" $(BUILD_FLAGS) -o $(BINARY_NAME) .

.PHONY: install
install: ## Install the binary to GOPATH/bin
	@echo "$(BLUE)Installing $(BINARY_NAME)...$(RESET)"
	$(GO) install $(BUILD_FLAGS) .

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(RESET)"
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	$(GO) clean -cache -testcache -modcache

# ===== Testing =====
.PHONY: test
test: ## Run all tests
	@echo "$(BLUE)Running tests...$(RESET)"
	$(GO) test -v ./...

.PHONY: test-short
test-short: ## Run tests with -short flag
	@echo "$(BLUE)Running short tests...$(RESET)"
	$(GO) test -short -v ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "$(BLUE)Running tests with race detection...$(RESET)"
	$(GO) test -race -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(RESET)"
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(RESET)"

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "$(BLUE)Running benchmarks...$(RESET)"
	$(GO) test -bench=. -benchmem ./...

# ===== Code Quality =====
.PHONY: lint
lint: ## Run linters
	@echo "$(BLUE)Running linters...$(RESET)"
	$(GOLANGCI_LINT) run --timeout=5m

.PHONY: lint-fix
lint-fix: ## Run linters and auto-fix issues
	@echo "$(BLUE)Running linters with auto-fix...$(RESET)"
	$(GOLANGCI_LINT) run --fix --timeout=5m

.PHONY: fmt
fmt: ## Format Go code
	@echo "$(BLUE)Formatting Go code...$(RESET)"
	$(GOFMT) -w -s .

.PHONY: fmt-check
fmt-check: ## Check if code is formatted
	@echo "$(BLUE)Checking code formatting...$(RESET)"
	@if [ -n "$$($(GOFMT) -l .)" ]; then \
		echo "$(RED)Code is not formatted. Run 'make fmt' to fix.$(RESET)"; \
		$(GOFMT) -l .; \
		exit 1; \
	else \
		echo "$(GREEN)Code is properly formatted.$(RESET)"; \
	fi

.PHONY: vet
vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(RESET)"
	$(GO) vet ./...

.PHONY: check
check: fmt-check vet lint test ## Run all checks (fmt, vet, lint, test)

# ===== Dependencies =====
.PHONY: deps
deps: ## Download dependencies
	@echo "$(BLUE)Downloading dependencies...$(RESET)"
	$(GO) mod download

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "$(BLUE)Updating dependencies...$(RESET)"
	$(GO) get -u ./...
	$(GO) mod tidy

.PHONY: deps-verify
deps-verify: ## Verify dependencies
	@echo "$(BLUE)Verifying dependencies...$(RESET)"
	$(GO) mod verify

.PHONY: deps-tidy
deps-tidy: ## Tidy dependencies
	@echo "$(BLUE)Tidying dependencies...$(RESET)"
	$(GO) mod tidy

# ===== Development Tools Setup =====
.PHONY: tools
tools: ## Install development tools
	@echo "$(BLUE)Installing development tools...$(RESET)"
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin)
	@echo "$(GREEN)Development tools installed$(RESET)"

# ===== Release =====
.PHONY: release
release: clean ## Build release binaries for all platforms
	@echo "$(BLUE)Building release binaries...$(RESET)"
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		output="$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-$$os-$$arch$$ext"; \
		echo "Building $$output..."; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GO) build $(BUILD_FLAGS) -o $$output .; \
		if [ $$? -ne 0 ]; then \
			echo "$(RED)Failed to build for $$platform$(RESET)"; \
			exit 1; \
		fi; \
	done
	@echo "$(GREEN)Release binaries built in $(DIST_DIR)/$(RESET)"

.PHONY: release-checksums
release-checksums: release ## Generate checksums for release binaries
	@echo "$(BLUE)Generating checksums...$(RESET)"
	@cd $(DIST_DIR) && sha256sum $(BINARY_NAME)-* > checksums.txt
	@echo "$(GREEN)Checksums generated: $(DIST_DIR)/checksums.txt$(RESET)"

# ===== Examples and Testing =====
.PHONY: run-example
run-example: build ## Run example with guestbook
	@echo "$(BLUE)Running guestbook example...$(RESET)"
	./$(BINARY_NAME) run $(EXAMPLES_DIR)/guestbook

.PHONY: validate-example
validate-example: build ## Validate guestbook example
	@echo "$(BLUE)Validating guestbook example...$(RESET)"
	./$(BINARY_NAME) validate $(EXAMPLES_DIR)/guestbook

.PHONY: config-example
config-example: build ## Show merged config for guestbook example
	@echo "$(BLUE)Showing guestbook example configuration...$(RESET)"
	./$(BINARY_NAME) config $(EXAMPLES_DIR)/guestbook

.PHONY: tags-example
tags-example: build ## Show tags for guestbook example
	@echo "$(BLUE)Showing guestbook example tags...$(RESET)"
	./$(BINARY_NAME) tags $(EXAMPLES_DIR)/guestbook

# ===== Documentation =====
.PHONY: docs-install
docs-install: ## Install documentation dependencies
	@echo "$(BLUE)Installing documentation dependencies...$(RESET)"
	@which pip > /dev/null || (echo "$(RED)Python pip not found. Please install Python and pip.$(RESET)" && exit 1)
	pip install mkdocs-material mkdocs-git-revision-date-localized-plugin mkdocs-git-committers-plugin-2 mkdocs-minify-plugin

.PHONY: docs-serve
docs-serve: ## Serve documentation locally
	@echo "$(BLUE)Starting documentation server...$(RESET)"
	@which mkdocs > /dev/null || (echo "$(RED)MkDocs not found. Run 'make docs-install' first.$(RESET)" && exit 1)
	mkdocs serve

.PHONY: docs-build
docs-build: ## Build documentation
	@echo "$(BLUE)Building documentation...$(RESET)"
	@which mkdocs > /dev/null || (echo "$(RED)MkDocs not found. Run 'make docs-install' first.$(RESET)" && exit 1)
	mkdocs build

.PHONY: docs-deploy
docs-deploy: ## Deploy documentation to GitHub Pages
	@echo "$(BLUE)Deploying documentation...$(RESET)"
	@which mkdocs > /dev/null || (echo "$(RED)MkDocs not found. Run 'make docs-install' first.$(RESET)" && exit 1)
	mkdocs gh-deploy --force

.PHONY: docs-clean
docs-clean: ## Clean documentation build artifacts
	@echo "$(BLUE)Cleaning documentation build artifacts...$(RESET)"
	rm -rf site/

# ===== Docker (Optional) =====
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(RESET)"
	docker build -t $(BINARY_NAME):$(VERSION) .

.PHONY: docker-run
docker-run: docker-build ## Run Docker container
	@echo "$(BLUE)Running Docker container...$(RESET)"
	docker run --rm -it $(BINARY_NAME):$(VERSION)

# ===== Utility =====
.PHONY: version
version: ## Show version information
	@echo "$(BLUE)Version Information:$(RESET)"
	@echo "  Version: $(VERSION)"
	@echo "  Commit:  $(COMMIT)"
	@echo "  Built:   $(BUILD_TIME)"

.PHONY: debug-vars
debug-vars: ## Show makefile variables
	@echo "$(BLUE)Makefile Variables:$(RESET)"
	@echo "  BINARY_NAME: $(BINARY_NAME)"
	@echo "  PACKAGE:     $(PACKAGE)"
	@echo "  VERSION:     $(VERSION)"
	@echo "  COMMIT:      $(COMMIT)"
	@echo "  BUILD_TIME:  $(BUILD_TIME)"
	@echo "  BUILD_FLAGS: $(BUILD_FLAGS)"

# ===== Git Hooks =====
.PHONY: git-hooks
git-hooks: ## Install git hooks
	@echo "$(BLUE)Installing git hooks...$(RESET)"
	@echo '#!/bin/sh\nmake check' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "$(GREEN)Git hooks installed$(RESET)"

# ===== CI/CD =====
.PHONY: ci
ci: deps-verify fmt-check vet lint test ## Run CI pipeline
	@echo "$(GREEN)CI pipeline completed successfully$(RESET)"

# Make sure intermediate files are cleaned up
.INTERMEDIATE: coverage.out