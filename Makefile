# Forgor CLI Makefile

# Build variables
BINARY_NAME=forgor
BUILD_DIR=dist
VERSION=$(shell cat VERSION 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-X 'forgor/cmd.Version=$(VERSION)' -X 'forgor/cmd.GitCommit=$(GIT_COMMIT)' -X 'forgor/cmd.BuildDate=$(BUILD_DATE)'

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod

# Git variables
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

.PHONY: help build clean test test-coverage lint fmt vet deps update-deps \
        build-all version version-info version-bump-patch version-bump-minor \
        version-bump-major version-bump-prerelease version-check \
        release-patch release-minor release-major release-prerelease \
        install uninstall run dev check-quality create-pr pre-commit auto-push-version

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the binary
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) -v

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Linux amd64..."
	@GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	@echo "Building for Linux arm64..."
	@GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64
	@echo "Building for macOS amd64..."
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64
	@echo "Building for macOS arm64..."
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64
	@echo "Building for Windows amd64..."
	@GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe
	@echo "Build complete! Binaries in $(BUILD_DIR)/"

install: build ## Install the binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin"
	@cp $(BINARY_NAME) $(GOPATH)/bin/

uninstall: ## Remove the binary from GOPATH/bin
	@echo "Removing $(BINARY_NAME) from $(GOPATH)/bin"
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)

# Testing targets
test: ## Run tests
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-ci: ## Run tests in CI mode
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Code quality targets
fmt: ## Format code
	@echo "Formatting code..."
	@gofmt -s -w .
	@echo "Code formatted successfully"

vet: ## Run go vet
	@echo "Running go vet..."
	@$(GOCMD) vet ./...
	@echo "go vet passed"

lint: vet ## Run linting (includes vet)
	@echo "Running additional linting..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, skipping advanced linting"; \
	fi

check-quality: fmt vet test ## Run all quality checks
	@echo "✅ All quality checks passed!"

# Dependency management
deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

update-deps: ## Update dependencies
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

# Development targets
run: build ## Build and run the application
	./$(BINARY_NAME)

dev: ## Run in development mode (with version info)
	@echo "Running in development mode..."
	@echo "Version: $(VERSION)"
	@echo "Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "=========================="
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) && ./$(BINARY_NAME)

# Auto-push version changes if no other changes
auto-push-version: ## Auto commit and push version changes if no other changes
	@echo "Checking for version-only changes..."
	@if [ -n "$$(git status --porcelain | grep -v "VERSION\|CHANGELOG.md")" ]; then \
		echo "⚠️  Other changes detected besides version files. Not auto-pushing."; \
		echo "   Please commit and push manually after reviewing changes."; \
	else \
		echo "✅ Only version-related changes detected. Auto-pushing..."; \
		git push origin $(GIT_BRANCH); \
		git push origin --tags; \
		echo "✅ Changes pushed successfully!"; \
	fi

# Version management targets
version: version-info ## Show version information

version-info: ## Display current version info
	@echo "Current version: $(VERSION)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Build date: $(BUILD_DATE)"

version-check: ## Validate VERSION file format
	@if [ ! -f "VERSION" ]; then \
		echo "❌ VERSION file not found"; \
		exit 1; \
	fi
	@if ! grep -E '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$$' VERSION >/dev/null; then \
		echo "❌ Invalid version format in VERSION file"; \
		echo "Expected: MAJOR.MINOR.PATCH[-PRERELEASE]"; \
		exit 1; \
	fi
	@echo "✅ VERSION file format is valid: $(VERSION)"

version-bump-patch: ## Bump patch version
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh bump patch; \
		$(MAKE) auto-push-version; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

version-bump-minor: ## Bump minor version
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh bump minor; \
		$(MAKE) auto-push-version; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

version-bump-major: ## Bump major version
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh bump major; \
		$(MAKE) auto-push-version; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

version-bump-prerelease: ## Bump prerelease version
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh bump prerelease; \
		$(MAKE) auto-push-version; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

# Release targets
release-patch: version-bump-patch ## Create a patch release
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh release; \
		$(MAKE) auto-push-version; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

release-minor: version-bump-minor ## Create a minor release
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh release; \
		$(MAKE) auto-push-version; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

release-major: version-bump-major ## Create a major release
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh release; \
		$(MAKE) auto-push-version; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

release-prerelease: version-bump-prerelease ## Create a prerelease
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh release; \
		$(MAKE) auto-push-version; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

# PR and development workflow targets
pre-commit: check-quality version-check ## Run pre-commit checks
	@echo "✅ Ready for commit!"

create-pr: ## Create a pull request with quality checks
	@if [ ! -f "scripts/create-pr.sh" ]; then \
		echo "❌ scripts/create-pr.sh not found"; \
		exit 1; \
	fi
	@scripts/create-pr.sh $(filter-out $@,$(MAKECMDGOALS))

# Allow passing arguments to create-pr
%:
	@: 