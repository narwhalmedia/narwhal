# Narwhal Media Server Makefile
# This Makefile provides targets for building, testing, and managing the Narwhal Media Server project.

# -----------------------------------------------------------------------------
# Variables
# -----------------------------------------------------------------------------

# Tool versions
BUF_VERSION := 1.28.1
PROTOC_VERSION := 25.1
GATEWAY_VERSION := v2.14.7
GOLANGCI_LINT_VERSION := 1.55.2

# Build variables
BINARY_DIR := bin
PROTO_GEN_DIR := api/proto/gen
SERVICES := media download transcode stream gateway

# -----------------------------------------------------------------------------
# Targets
# -----------------------------------------------------------------------------

.PHONY: all build test clean proto dev dev-down help tools lint deps

# Default target
all: deps proto build

# -----------------------------------------------------------------------------
# Development Tools
# -----------------------------------------------------------------------------

tools:
	@echo "Installing development tools..."
	go install github.com/bufbuild/buf/cmd/buf@v$(BUF_VERSION)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v$(PROTOC_VERSION)
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v$(PROTOC_VERSION)
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@$(GATEWAY_VERSION)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)

# -----------------------------------------------------------------------------
# Dependencies
# -----------------------------------------------------------------------------

deps:
	@echo "Managing dependencies..."
	go mod tidy

# -----------------------------------------------------------------------------
# Protobuf Generation
# -----------------------------------------------------------------------------

proto:
	@echo "Generating protobuf code..."
	cd api/proto
	buf generate

# -----------------------------------------------------------------------------
# Build
# -----------------------------------------------------------------------------

build: $(SERVICES)

$(SERVICES):
	@echo "Building $@ service..."
	go build -o $(BINARY_DIR)/$@ ./cmd/$@

# -----------------------------------------------------------------------------
# Testing and Linting
# -----------------------------------------------------------------------------

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Running linter..."
	golangci-lint run

# -----------------------------------------------------------------------------
# Cleanup
# -----------------------------------------------------------------------------

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BINARY_DIR)/
	rm -rf $(PROTO_GEN_DIR)/

# -----------------------------------------------------------------------------
# Development Environment
# -----------------------------------------------------------------------------

dev:
	@echo "Starting development environment..."
	docker-compose up -d

dev-down:
	@echo "Stopping development environment..."
	docker-compose down

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------

help:
	@echo "Narwhal Media Server - Available targets:"
	@echo ""
	@echo "Development:"
	@echo "  tools     - Install development tools (buf, protoc, etc.)"
	@echo "  deps      - Update and tidy Go dependencies"
	@echo "  proto     - Generate protobuf code"
	@echo "  build     - Build all services"
	@echo "  test      - Run tests"
	@echo "  lint      - Run linter"
	@echo ""
	@echo "Environment:"
	@echo "  dev       - Start development environment"
	@echo "  dev-down  - Stop development environment"
	@echo ""
	@echo "Maintenance:"
	@echo "  clean     - Clean build artifacts"
	@echo ""
	@echo "Individual services can be built with: make <service>"
	@echo "Available services: $(SERVICES)"
