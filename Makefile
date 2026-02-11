# Makefile for gitignore CLI tool

# Project settings
BINARY_NAME := gitignore
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Go settings
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Directories
SRC_DIR := ./src/cmd/gitignore
BUILD_DIR := ./dist
DIST_DIR := ./dist

# Platform targets
PLATFORMS := \
	darwin-amd64 \
	darwin-arm64 \
	linux-amd64 \
	windows-amd64

# Default target
.DEFAULT_GOAL := build

# Phony targets
.PHONY: all build build-all test test-short test-integration test-coverage \
        clean fmt vet lint deps install uninstall help \
        $(PLATFORMS)

# Build for current platform
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(SRC_DIR)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms
build-all: $(PLATFORMS)
	@echo "All platforms built successfully!"

# Cross-compilation targets
darwin-amd64:
	@echo "Building for macOS Intel (darwin/amd64)..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(SRC_DIR)

darwin-arm64:
	@echo "Building for macOS Silicon (darwin/arm64)..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(SRC_DIR)

linux-amd64:
	@echo "Building for Linux 64-bit (linux/amd64)..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(SRC_DIR)

windows-amd64:
	@echo "Building for Windows 64-bit (windows/amd64)..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(SRC_DIR)

# Run all tests
test:
	@echo "Running all tests..."
	$(GOTEST) -v ./...

# Run tests without integration tests (faster)
test-short:
	@echo "Running short tests..."
	$(GOTEST) -v -short ./...

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -run "Integration" ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) -v -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report: $(BUILD_DIR)/coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Install to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Uninstall from GOPATH/bin
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -f coverage.out

# Create release archives
release: build-all
	@echo "Creating release archives..."
	@mkdir -p $(DIST_DIR)/release
	@# macOS Intel
	cd $(DIST_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@# macOS Silicon
	cd $(DIST_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@# Linux
	cd $(DIST_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@# Windows
	cd $(DIST_DIR) && zip -q release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@echo "Release archives created in $(DIST_DIR)/release/"

# Show help
help:
	@echo "gitignore CLI - Makefile targets"
	@echo ""
	@echo "Build targets:"
	@echo "  build          Build for current platform"
	@echo "  build-all      Build for all platforms"
	@echo "  darwin-amd64   Build for macOS Intel"
	@echo "  darwin-arm64   Build for macOS Silicon"
	@echo "  linux-amd64    Build for Linux 64-bit"
	@echo "  windows-amd64  Build for Windows 64-bit"
	@echo ""
	@echo "Test targets:"
	@echo "  test           Run all tests"
	@echo "  test-short     Run short tests (skip integration)"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  test-integration  Run integration tests only"
	@echo ""
	@echo "Other targets:"
	@echo "  deps           Download and tidy dependencies"
	@echo "  fmt            Format code"
	@echo "  vet            Run go vet"
	@echo "  lint           Run golangci-lint"
	@echo "  install        Install to GOPATH/bin"
	@echo "  uninstall      Remove from GOPATH/bin"
	@echo "  clean          Remove build artifacts"
	@echo "  release        Create release archives"
	@echo "  help           Show this help message"
