.PHONY: all build test bench lint clean help

# Variables
BINARY_NAME=poltergeist
GO=go
GOFLAGS=-v

# Default target
all: lint test build

# Build the project
build:
	@echo "📦 Building..."
	$(GO) build $(GOFLAGS) ./...

# Run tests
test:
	@echo "🧪 Running tests..."
	$(GO) test -v -race ./...

# Run tests with coverage
cover:
	@echo "📊 Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "📄 Coverage report: coverage.html"

# Run benchmarks
bench:
	@echo "⚡ Running benchmarks..."
	$(GO) test -run=^$$ -bench=. -benchmem ./...

# Run linter
lint:
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, running go vet..."; \
		$(GO) vet ./...; \
	fi

# Format code
fmt:
	@echo "✨ Formatting code..."
	$(GO) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	$(GO) clean
	rm -f coverage.out coverage.html

# Run example
example:
	@echo "🚀 Running example..."
	$(GO) run ./examples/main.go

# Update dependencies
deps:
	@echo "📥 Updating dependencies..."
	$(GO) mod tidy
	$(GO) mod download

# Show help
help:
	@echo "Poltergeist Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all      - Run lint, test, and build (default)"
	@echo "  build    - Build the project"
	@echo "  test     - Run tests"
	@echo "  cover    - Run tests with coverage report"
	@echo "  bench    - Run benchmarks"
	@echo "  lint     - Run linter"
	@echo "  fmt      - Format code"
	@echo "  clean    - Clean build artifacts"
	@echo "  example  - Run example server"
	@echo "  deps     - Update dependencies"
	@echo "  help     - Show this help"

