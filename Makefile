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

.PHONY: help build clean test test-coverage lint fmt vet deps update-deps \
        build-all version version-info version-bump-patch version-bump-minor \
        version-bump-major version-bump-prerelease version-check \
        release-patch release-minor release-major release-prerelease \
        install uninstall run dev check-quality create-pr pre-commit

# Default target
help: ## Show this help message
	@echo "🔥 Forgor CLI Makefile (v$(VERSION))"
	@echo "Git: $(GIT_COMMIT)"
	@echo "Built: $(BUILD_DATE)"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the binary
	@echo "🔨 Building $(BINARY_NAME) v$(VERSION)..."
	@echo "📦 Target: $(BINARY_NAME)"
	@echo "🏷️  Version: $(VERSION)"
	@echo "📝 Git Commit: $(GIT_COMMIT)"
	@echo "📅 Build Date: $(BUILD_DATE)"
	@echo ""
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) -v
	@echo ""
	@echo "✅ Build complete: $(BINARY_NAME) v$(VERSION)"

build-all: ## Build for all platforms
	@echo "🔨 Building $(BINARY_NAME) v$(VERSION) for all platforms..."
	@echo "📝 Git Commit: $(GIT_COMMIT)"
	@echo "📅 Build Date: $(BUILD_DATE)"
	@echo ""
	@mkdir -p $(BUILD_DIR)
	@echo "🐧 Building for Linux amd64..."
	@GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	@echo "🐧 Building for Linux arm64..."
	@GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64
	@echo "🍎 Building for macOS amd64..."
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64
	@echo "🍎 Building for macOS arm64..."
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64
	@echo "🪟 Skipping Building for Windows amd64 (not supported yet)..."
	# @GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe
	@echo ""
	@echo "✅ Build complete! $(BINARY_NAME) v$(VERSION) binaries in $(BUILD_DIR)/"

install: build ## Install the binary to GOPATH/bin
	@echo "📦 Installing $(BINARY_NAME) v$(VERSION) to $(GOPATH)/bin"
	@cp $(BINARY_NAME) $(GOPATH)/bin/
	@echo "✅ $(BINARY_NAME) v$(VERSION) installed successfully!"

uninstall: ## Remove the binary from GOPATH/bin
	@echo "🗑️  Removing $(BINARY_NAME) from $(GOPATH)/bin"
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "✅ $(BINARY_NAME) uninstalled successfully!"

clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts for $(BINARY_NAME) v$(VERSION)..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	@echo "✅ Clean complete!"

# Testing targets
test: ## Run tests
	@echo "🧪 Running tests for $(BINARY_NAME) v$(VERSION)..."
	@echo "📝 Git Commit: $(GIT_COMMIT)"
	@echo ""
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	@echo "🧪 Running tests with coverage for $(BINARY_NAME) v$(VERSION)..."
	@echo "📝 Git Commit: $(GIT_COMMIT)"
	@echo ""
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo ""
	@echo "📊 Coverage report generated: coverage.html"

test-ci: ## Run tests in CI mode
	@echo "🤖 Running CI tests for $(BINARY_NAME) v$(VERSION)..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Code quality targets
fmt: ## Format code
	@echo "🎨 Formatting code for $(BINARY_NAME) v$(VERSION)..."
	@gofmt -s -w .
	@echo "✅ Code formatted successfully"

vet: ## Run go vet
	@echo "🔍 Running go vet for $(BINARY_NAME) v$(VERSION)..."
	@$(GOCMD) vet ./...
	@echo "✅ go vet passed"

lint: vet ## Run linting (includes vet)
	@echo "🔍 Running additional linting for $(BINARY_NAME) v$(VERSION)..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint not found, skipping advanced linting"; \
	fi

check-quality: fmt vet test ## Run all quality checks
	@echo "✅ All quality checks passed for $(BINARY_NAME) v$(VERSION)!"

# Dependency management
deps: ## Download dependencies
	@echo "📦 Downloading dependencies for $(BINARY_NAME) v$(VERSION)..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✅ Dependencies updated!"

update-deps: ## Update dependencies
	@echo "🔄 Updating dependencies for $(BINARY_NAME) v$(VERSION)..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy
	@echo "✅ Dependencies updated!"

# Development targets
run: build ## Build and run the application
	@echo "🚀 Running $(BINARY_NAME) v$(VERSION)..."
	@echo ""
	./$(BINARY_NAME)

dev: ## Run in development mode (with version info)
	@echo "🚀 Running $(BINARY_NAME) in development mode..."
	@echo "🏷️  Version: $(VERSION)"
	@echo "📝 Git Commit: $(GIT_COMMIT)"
	@echo "📅 Build Date: $(BUILD_DATE)"
	@echo "=========================="
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) && ./$(BINARY_NAME)

# Version management targets
version: version-info ## Show version information

version-info: ## Display current version info
	@echo "🔥 Forgor CLI Version Information"
	@echo "================================="
	@echo "📦 Binary Name: $(BINARY_NAME)"
	@echo "🏷️  Current Version: $(VERSION)"
	@echo "📝 Git Commit: $(GIT_COMMIT)"
	@echo "📅 Build Date: $(BUILD_DATE)"
	@echo "💾 Version File: VERSION"

version-check: ## Validate VERSION file format
	@echo "🔍 Validating VERSION file format..."
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
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

version-bump-minor: ## Bump minor version
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh bump minor; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

version-bump-major: ## Bump major version
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh bump major; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

version-bump-prerelease: ## Bump prerelease version
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh bump prerelease; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

# Release targets
release-patch: version-bump-patch ## Create a patch release
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh release; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

release-minor: version-bump-minor ## Create a minor release
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh release; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

release-major: version-bump-major ## Create a major release
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh release; \
	else \
		echo "❌ scripts/version.sh not found"; \
		exit 1; \
	fi

release-prerelease: version-bump-prerelease ## Create a prerelease
	@if [ -f "scripts/version.sh" ]; then \
		scripts/version.sh release; \
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