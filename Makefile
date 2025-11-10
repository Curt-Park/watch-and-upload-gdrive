# Makefile for Watch and Upload to Google Drive (WUG)

# Variables
BINARY_NAME=wug
MODULE_NAME=github.com/Curt-Park/watch-and-upload-gdrive
BUILD_DIR=build
OUTPUT_DIR?=$(BUILD_DIR)
COVERAGE_DIR=coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out
COVERAGE_HTML=$(COVERAGE_DIR)/coverage.html

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-s -w"
BUILD_FLAGS=-trimpath

.PHONY: all build clean test test-verbose test-coverage test-coverage-html deps fmt vet lint help install run

# Default target
all: clean deps fmt vet test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 .
	@echo "Linux build complete: $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64"

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "macOS builds complete"

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Windows build complete: $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe"

# Build for specific platform (for CI/CD)
build-linux-amd64:
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 .

build-darwin-amd64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 .

build-darwin-arm64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 .

build-windows-amd64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe .

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	$(GOTEST) -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) ./...
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)
	@echo "Coverage report saved to $(COVERAGE_FILE)"

# Generate HTML coverage report
test-coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "HTML coverage report saved to $(COVERAGE_HTML)"

# Run tests in short mode (skip long-running tests)
test-short:
	@echo "Running short tests..."
	$(GOTEST) -short ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies updated"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Vet check complete"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install it from https://golangci-lint.run/"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

# Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(BUILD_FLAGS) $(LDFLAGS) .
	@echo "Installation complete"

# Run the application (example - modify as needed)
run:
	@echo "Running $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) .
	./$(BINARY_NAME)

# Show help
help:
	@echo "Available targets:"
	@echo "  all                 - Clean, deps, fmt, vet, test, and build"
	@echo "  build               - Build the application"
	@echo "  build-all           - Build for Linux, macOS, and Windows"
	@echo "  build-linux         - Build for Linux (amd64)"
	@echo "  build-darwin        - Build for macOS (amd64 and arm64)"
	@echo "  build-windows       - Build for Windows (amd64)"
	@echo "  test                - Run tests"
	@echo "  test-verbose        - Run tests with verbose output and race detector"
	@echo "  test-coverage       - Run tests with coverage report"
	@echo "  test-coverage-html  - Generate HTML coverage report"
	@echo "  test-short          - Run tests in short mode"
	@echo "  deps                - Download and tidy dependencies"
	@echo "  fmt                 - Format code"
	@echo "  vet                 - Run go vet"
	@echo "  lint                - Run golangci-lint (if installed)"
	@echo "  clean               - Remove build artifacts"
	@echo "  install             - Install binary to GOPATH/bin"
	@echo "  run                 - Build and run the application"
	@echo "  help                - Show this help message"

