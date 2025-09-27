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

# Generate templ files
.PHONY: generate-templ
generate-templ:
	@echo "🔄 Generating templ files..."
	@if command -v templ >/dev/null 2>&1; then \
		templ generate; \
	else \
		echo "⚠️  templ not installed. Install with: go install github.com/a-h/templ/cmd/templ@latest"; \
		exit 1; \
	fi
	@echo "✅ Templ files generated"

# Build web UI
.PHONY: webui
webui: generate-templ $(BIN_DIR)/$(APP_NAME)-webui

$(BIN_DIR)/$(APP_NAME)-webui: $(GO_FILES)
	@echo "Building Web UI..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $@ ./cmd/webui

# Build the application
.PHONY: build
build: $(BIN_DIR)/$(APP_NAME)

$(BIN_DIR)/$(APP_NAME): $(GO_FILES)
	@echo "🔨 Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)
	@echo "✅ Build complete: $(BIN_DIR)/$(APP_NAME)"

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


# Show help
.PHONY: help
help:
	@echo "🗂️  OxBin - Walrus Pastebin CLI"
	@echo ""
	@echo "Available targets:"
	@echo "  build           Build the CLI application"
	@echo "  generate-templ  Generate Go code from templ templates"
	@echo "  webui           Build the Web UI"
	@echo "  clean           Clean build artifacts"
	@echo "  deps            Install dependencies"
	@echo "  update-deps     Update dependencies"
	@echo "  test            Run tests"
	@echo "  fmt             Format code"
	@echo "  help            Show this help message"
	@echo ""
	@echo "Quick start:"
	@echo "  make build      # Build the application"
	@echo "  make run        # Build and run"
	@echo "  make help       # Show this help"


# Default help when no target specified
.DEFAULT_GOAL := help
