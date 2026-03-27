# mcp2cli Makefile

# Version info
VERSION ?= dev
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "none")
GIT_REPOSITORY := github.com/weibaohui/mcp2cli

# Define all supported platforms and architectures for cross-platform build
ALL_PLATFORMS := \
    linux/amd64 \
    linux/arm64 \
    windows/amd64 \
    windows/arm64 \
    darwin/amd64 \
    darwin/arm64

# Bin directory
BIN_DIR := bin

.PHONY: all build build-all build-all-cross clean install test lint fmt build-linux build-darwin build-windows version help

all: build

# Build for current platform
build:
	@echo "Building mcp2cli for current platform..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.GitTag=$(GIT_TAG) -X main.GitRepo=$(GIT_REPOSITORY)" -o $(BIN_DIR)/mcp2cli .
	@echo "Build complete: $(BIN_DIR)/mcp2cli"

# Cross-platform build for all platforms
build-all-cross:
	@echo "Building mcp2cli for all platforms..."
	@mkdir -p $(BIN_DIR)
	@for platform in $(ALL_PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/}; \
		echo "Building platform: $$GOOS/$$GOARCH ..."; \
		OUTPUT_FILE="$(BIN_DIR)/mcp2cli-$$GOOS-$$GOARCH"; \
		if [ "$$GOOS" = "windows" ]; then \
			OUTPUT_FILE="$(BIN_DIR)/mcp2cli-$$GOOS-$$GOARCH.exe"; \
		fi; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.GitTag=$(GIT_TAG) -X main.GitRepo=$(GIT_REPOSITORY)" -o "$$OUTPUT_FILE" .; \
		echo "  --> $$OUTPUT_FILE"; \
	done
	@echo ""
	@echo "All platforms built successfully!"
	@ls -lh $(BIN_DIR)/

# Build all platforms (alias)
build-all: build-all-cross

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@echo "Clean complete"

# Install to system (requires sudo on Unix)
install:
	@echo "Installing mcp2cli to /usr/local/bin..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.GitTag=$(GIT_TAG) -X main.GitRepo=$(GIT_REPOSITORY)" -o $(BIN_DIR)/mcp2cli .
	@install -Dm755 $(BIN_DIR)/mcp2cli /usr/local/bin/mcp2cli
	@echo "Installed to /usr/local/bin/mcp2cli"

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint code
lint:
	gofmt -w .
	go vet ./...
	@echo "Lint complete"

# Format code
fmt:
	gofmt -w .
	@echo "Format complete"

# Build for Linux only
build-linux:
	@echo "Building mcp2cli for Linux..."
	@mkdir -p $(BIN_DIR)
	@for platform in linux/amd64 linux/arm64; do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/}; \
		echo "Building platform: $$GOOS/$$GOARCH ..."; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.GitTag=$(GIT_TAG) -X main.GitRepo=$(GIT_REPOSITORY)" -o "$(BIN_DIR)/mcp2cli-$$GOOS-$$GOARCH" .; \
		echo "  --> $(BIN_DIR)/mcp2cli-$$GOOS-$$GOARCH"; \
	done
	@echo "Linux build complete!"

# Build for macOS only
build-darwin:
	@echo "Building mcp2cli for macOS..."
	@mkdir -p $(BIN_DIR)
	@for platform in darwin/amd64 darwin/arm64; do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/}; \
		echo "Building platform: $$GOOS/$$GOARCH ..."; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.GitTag=$(GIT_TAG) -X main.GitRepo=$(GIT_REPOSITORY)" -o "$(BIN_DIR)/mcp2cli-$$GOOS-$$GOARCH" .; \
		echo "  --> $(BIN_DIR)/mcp2cli-$$GOOS-$$GOARCH"; \
	done
	@echo "macOS build complete!"

# Build for Windows only
build-windows:
	@echo "Building mcp2cli for Windows..."
	@mkdir -p $(BIN_DIR)
	@for platform in windows/amd64 windows/arm64; do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/}; \
		echo "Building platform: $$GOOS/$$GOARCH ..."; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.GitTag=$(GIT_TAG) -X main.GitRepo=$(GIT_REPOSITORY)" -o "$(BIN_DIR)/mcp2cli-$$GOOS-$$GOARCH.exe" .; \
		echo "  --> $(BIN_DIR)/mcp2cli-$$GOOS-$$GOARCH.exe"; \
	done
	@echo "Windows build complete!"

# Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Git Tag: $(GIT_TAG)"
	@echo "Repository: $(GIT_REPOSITORY)"

# Help
help:
	@echo "mcp2cli Makefile targets:"
	@echo ""
	@echo "  build         - Build for current platform"
	@echo "  build-all     - Build for all platforms (linux, windows, darwin)"
	@echo "  build-linux   - Build for Linux only"
	@echo "  build-darwin  - Build for macOS only"
	@echo "  build-windows - Build for Windows only"
	@echo "  install       - Install to /usr/local/bin"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  test-cover    - Run tests with coverage report"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  version       - Show version info"
	@echo "  help          - Show this help"
