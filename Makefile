.PHONY: build build-cli build-all clean test test-cli run fmt lint install-cli help

# Build variables
BINARY_NAME=cola-registry
CLI_BINARY_NAME=cola-regctl
BUILD_DIR=bin
DIST_DIR=dist
CMD_DIR=cmd/cola-registry
CLI_CMD_DIR=cmd/cola-regctl

# Version variables - can be overridden via environment or command line
VERSION ?= dev
BUILD_NUM ?= $(shell date +%Y%m%d-%H%M%S)
APP_NAME_SERVER = cola-registry
APP_LONG_NAME_SERVER = Command Launcher Registry Server
APP_NAME_CLI = cola-regctl
APP_LONG_NAME_CLI = Command Launcher Registry CLI

# Build flags
LDFLAGS_BASE = -s -w
LDFLAGS_SERVER = $(LDFLAGS_BASE) \
	-X 'main.version=$(VERSION)' \
	-X 'main.buildNum=$(BUILD_NUM)' \
	-X 'main.appName=$(APP_NAME_SERVER)' \
	-X 'main.appLongName=$(APP_LONG_NAME_SERVER)'
LDFLAGS_CLI = $(LDFLAGS_BASE) \
	-X 'main.version=$(VERSION)' \
	-X 'main.buildNum=$(BUILD_NUM)' \
	-X 'main.appName=$(APP_NAME_CLI)' \
	-X 'main.appLongName=$(APP_LONG_NAME_CLI)'

# Default target
all: build

## build: Build the server binary
build:
	@echo "Building $(BINARY_NAME) version=$(VERSION) build=$(BUILD_NUM)..."
	@CGO_ENABLED=0 go build -ldflags="$(LDFLAGS_SERVER)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## build-cli: Build the CLI client binary
build-cli:
	@echo "Building $(CLI_BINARY_NAME) version=$(VERSION) build=$(BUILD_NUM)..."
	@CGO_ENABLED=0 go build -ldflags="$(LDFLAGS_CLI)" -o $(BUILD_DIR)/$(CLI_BINARY_NAME) ./$(CLI_CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(CLI_BINARY_NAME)"

## build-all: Build both server and CLI binaries
build-all: build build-cli

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@echo "Clean complete"

## test: Run all tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.txt ./...
	@echo "Tests complete"

## test-cli: Run CLI client tests only
test-cli:
	@echo "Running CLI tests..."
	@go test -v -race ./internal/client/...
	@echo "CLI tests complete"

## run: Build and run the server
run: build
	@echo "Starting server..."
	@$(BUILD_DIR)/$(BINARY_NAME) server

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

## fmt-check: Check if code is formatted (for CI)
fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Code is not formatted. Run 'make fmt'"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "Format check passed"

## lint: Run linter (go vet)
lint:
	@echo "Running linter..."
	@go vet ./...
	@echo "Lint complete"

## install-cli: Install the CLI client to $GOPATH/bin
install-cli: build-cli
	@echo "Installing $(CLI_BINARY_NAME) to $$GOPATH/bin..."
	@cp $(BUILD_DIR)/$(CLI_BINARY_NAME) $$GOPATH/bin/
	@echo "Install complete"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
