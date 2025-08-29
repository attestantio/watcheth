# Binary name
BINARY_NAME=watcheth
BUILD_DIR=build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build variables
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# LDFLAGS for version info
LDFLAGS=-ldflags "-X github.com/watcheth/watcheth/internal/version.Version=$(VERSION) -X github.com/watcheth/watcheth/internal/version.BuildTime=$(BUILD_TIME) -X github.com/watcheth/watcheth/internal/version.CommitHash=$(COMMIT_HASH)"

# Default target
.DEFAULT_GOAL := build

# Build for current platform
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) -v

# Build for Linux AMD64
.PHONY: build-linux
build-linux:
	@echo "Building $(BINARY_NAME) for Linux AMD64..."
	@mkdir -p $(BUILD_DIR)/linux-amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/linux-amd64/$(BINARY_NAME) -v

# Build for all major platforms
.PHONY: build-all
build-all:
	@echo "Building $(BINARY_NAME) for all platforms..."
	@mkdir -p $(BUILD_DIR)/linux-amd64
	@mkdir -p $(BUILD_DIR)/linux-arm64
	@mkdir -p $(BUILD_DIR)/darwin-amd64
	@mkdir -p $(BUILD_DIR)/darwin-arm64
	
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/linux-amd64/$(BINARY_NAME) -v
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/linux-arm64/$(BINARY_NAME) -v
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/darwin-amd64/$(BINARY_NAME) -v
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/darwin-arm64/$(BINARY_NAME) -v

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-linux-amd64

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Install dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOGET) -v ./...
	$(GOMOD) tidy

# Install binary to GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

# Run the application
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

# Display help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build for current platform"
	@echo "  build-linux - Build for Linux AMD64"
	@echo "  build-all   - Build for all major platforms"
	@echo "  clean       - Remove build artifacts"
	@echo "  test        - Run tests"
	@echo "  deps        - Download dependencies"
	@echo "  install     - Install binary to GOPATH/bin"
	@echo "  run         - Build and run the application"
	@echo "  help        - Display this help message"