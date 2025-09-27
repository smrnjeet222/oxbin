# OxBin Makefile
# Walrus Pastebin CLI Application

# Variables
APP_NAME := oxbin
BIN_DIR := bin
BUILD_DIR := .
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build: $(BIN_DIR)/$(APP_NAME)

$(BIN_DIR)/$(APP_NAME): $(GO_FILES)
	@echo "🔨 Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) .
	@echo "✅ Build complete: $(BIN_DIR)/$(APP_NAME)"

# Build with race detection (for development)
.PHONY: build-race
build-race:
	@echo "🔨 Building $(APP_NAME) with race detection..."
	@mkdir -p $(BIN_DIR)
	go build -race $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-race .
	@echo "✅ Race build complete: $(BIN_DIR)/$(APP_NAME)-race"

# Run the application
.PHONY: run
run: build
	@echo "🚀 Running $(APP_NAME)..."
	./$(BIN_DIR)/$(APP_NAME)

# Run with race detection
.PHONY: run-race
run-race: build-race
	@echo "🚀 Running $(APP_NAME) with race detection..."
	./$(BIN_DIR)/$(APP_NAME)-race

# Clean build artifacts
.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	go clean
	@echo "✅ Clean complete"

# Install dependencies
.PHONY: deps
deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed"

# Update dependencies
.PHONY: update-deps
update-deps:
	@echo "🔄 Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "✅ Dependencies updated"

# Run tests
.PHONY: test
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

# Run linter
.PHONY: lint
lint:
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		go vet ./...; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "💡 Install goimports for better formatting: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi
	@echo "✅ Code formatted"

# Check for security issues
.PHONY: security
security:
	@echo "🔒 Checking for security issues..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "⚠️  gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Development setup
.PHONY: dev-setup
dev-setup: deps
	@echo "🛠️  Setting up development environment..."
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "✅ Development environment ready"

# Create release build
.PHONY: release
release: clean test lint
	@echo "🚀 Creating release build..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -a -installsuffix cgo -o $(BIN_DIR)/$(APP_NAME) .
	@echo "✅ Release build complete: $(BIN_DIR)/$(APP_NAME)"

# Cross-compile for different platforms
.PHONY: build-all
build-all: clean
	@echo "🌍 Building for multiple platforms..."
	@mkdir -p $(BIN_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 .
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-linux-arm64 .
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-darwin-amd64 .
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-darwin-arm64 .
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-windows-amd64.exe .
	
	@echo "✅ Cross-compilation complete"
	@ls -la $(BIN_DIR)/


# Show help
.PHONY: help
help:
	@echo "🗂️  OxBin - Walrus Pastebin CLI"
	@echo ""
	@echo "Available targets:"
	@echo "  build           Build the application"
	@echo "  build-race      Build with race detection"
	@echo "  run             Build and run the application"
	@echo "  run-race        Build and run with race detection"
	@echo "  clean           Clean build artifacts"
	@echo "  deps            Install dependencies"
	@echo "  update-deps     Update dependencies"
	@echo "  test            Run tests"
	@echo "  test-coverage   Run tests with coverage report"
	@echo "  lint            Run linter"
	@echo "  fmt             Format code"
	@echo "  security        Check for security issues"
	@echo "  dev-setup       Setup development environment"
	@echo "  release         Create optimized release build"
	@echo "  build-all       Cross-compile for multiple platforms"
	@echo "  help            Show this help message"
	@echo ""
	@echo "Quick start:"
	@echo "  make build      # Build the application"
	@echo "  make run        # Build and run"
	@echo "  make help       # Show this help"


# Default help when no target specified
.DEFAULT_GOAL := help
