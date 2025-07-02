.PHONY: build test clean install dev fmt lint run help

# Build info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS := -X 'forgor/cmd.Version=$(VERSION)' \
           -X 'forgor/cmd.GitCommit=$(COMMIT)' \
           -X 'forgor/cmd.BuildDate=$(DATE)'

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
BINARY_NAME=forgor
BINARY_UNIX=$(BINARY_NAME)_unix

# Default target
all: build

## Build the binary
build:
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) -v

## Build for production (optimized)
build-prod:
	CGO_ENABLED=0 $(GOBUILD) -ldflags "$(LDFLAGS) -s -w" -o $(BINARY_NAME) -v

## Run tests
test:
	$(GOTEST) -v ./...

## Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

## Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f coverage.out

## Install the binary to $GOPATH/bin
install:
	$(GOCMD) install -ldflags "$(LDFLAGS)"

## Development mode - run without building
dev:
	$(GOCMD) run -ldflags "$(LDFLAGS)" main.go

## Format code
fmt:
	$(GOFMT) -s -w .

## Run linter (requires golangci-lint)
lint:
	golangci-lint run

## Tidy dependencies
tidy:
	$(GOMOD) tidy

## Download dependencies
deps:
	$(GOMOD) download

## Create alias for ff command
alias:
	@echo "Run this command to create an alias:"
	@echo "echo 'alias ff=\"$(PWD)/$(BINARY_NAME)\"' >> ~/.bashrc"
	@echo "echo 'alias ff=\"$(PWD)/$(BINARY_NAME)\"' >> ~/.zshrc"

## Run with example query
run-example:
	./$(BINARY_NAME) "find all txt files with hello in them"

## Create sample config
config-sample:
	mkdir -p ~/.config/forgor
	cp configs/config.yaml ~/.config/forgor/config.yaml.example
	@echo "Sample config created at ~/.config/forgor/config.yaml.example"
	@echo "Copy it to ~/.config/forgor/config.yaml and edit as needed"

## Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)_linux_amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)_darwin_amd64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)_darwin_arm64
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)_windows_amd64.exe

## Show help
help:
	@echo "Available commands:"
	@grep -E '^## ' Makefile | sed 's/## /  /' 