.PHONY: all build test clean proto generate

# Variables
PROTO_DIR := api/proto
PROTO_OUT_DIR := api/proto
GO_MODULE := github.com/narwhalmedia/narwhal

# Build all services
all: generate build

# Database commands
db-up:
	@echo "Starting PostgreSQL and Redis..."
	@cd deployments/docker && docker-compose -f docker-compose.dev.yml up -d

db-down:
	@echo "Stopping PostgreSQL and Redis..."
	@cd deployments/docker && docker-compose -f docker-compose.dev.yml down

db-reset: db-down
	@echo "Resetting database..."
	@cd deployments/docker && docker-compose -f docker-compose.dev.yml down -v
	@$(MAKE) db-up

db-test:
	@echo "Testing database connection and migrations..."
	@go run cmd/dbtest/main.go

db-psql:
	@echo "Connecting to PostgreSQL..."
	@docker exec -it narwhal-postgres psql -U narwhal -d narwhal_dev

# Migration commands
migrate:
	@echo "Running database migrations..."
	@go run cmd/migrate/main.go

migrate-status:
	@echo "Checking migration status..."
	@go run cmd/migrate/main.go -status

migrate-dry-run:
	@echo "Showing pending migrations..."
	@go run cmd/migrate/main.go -dry-run

# Buf commands
buf-lint:
	@echo "Linting proto files..."
	@buf lint

buf-breaking:
	@echo "Checking for breaking changes..."
	@buf breaking --against '.git#branch=main'

buf-format:
	@echo "Formatting proto files..."
	@buf format -w

buf-generate:
	@echo "Generating code from proto files..."
	@buf generate

# Generate protobuf files (using Buf)
proto: buf-generate

# Generate all code (proto, mocks, etc.)
generate: proto
	@echo "Generating mocks..."
	go generate ./...

# Build all services
build:
	@echo "Building services..."
	go build -o bin/library ./cmd/library
	go build -o bin/user ./cmd/user
	go build -o bin/dbtest ./cmd/dbtest
	go build -o bin/migrate ./cmd/migrate

# Build specific service
build-%:
	go build -o bin/$* ./cmd/$*

# Build specific service
build-library:
	go build -o bin/library ./cmd/library

build-user:
	go build -o bin/user ./cmd/user

build-dbtest:
	go build -o bin/dbtest ./cmd/dbtest

build-migrate:
	go build -o bin/migrate ./cmd/migrate

# Run services
run-library: build-library
	./bin/library

run-user: build-user
	./bin/user

run-dbtest: build-dbtest
	./bin/dbtest

run-migrate: build-migrate
	./bin/migrate

# Development with hot reload (requires air)
dev-library:
	air -c .air.library.toml

dev-user:
	air -c .air.user.toml

# Run tests
test:
	$(MAKE) test-workflows
	@echo "Running all tests..."
	go test -v -race ./...

# Run unit tests only (skip integration tests)
test-unit:
	@echo "Running unit tests..."
	go test -v -race -short ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -race -run Integration ./...

# Run tests for a specific package
test-pkg:
	@echo "Running tests for package $(PKG)..."
	go test -v -race ./$(PKG)/...

# Run benchmarks
test-bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Run tests in watch mode (requires gotestsum)
test-watch:
	@which gotestsum > /dev/null || go install gotest.tools/gotestsum@latest
	gotestsum --watch

# Clean test artifacts
test-clean:
	rm -f coverage.out coverage.html

# Run linters
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run specific service
run-%:
	go run ./cmd/$*/main.go

# Docker commands
docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Development with hot reload (requires air)
dev-%:
	cd cmd/$* && air

# Install development tools
install-tools:
	@echo "Installing Buf..."
	@curl -sSL "https://github.com/bufbuild/buf/releases/download/v1.28.1/buf-$(shell uname -s)-$(shell uname -m)" -o /usr/local/bin/buf && chmod +x /usr/local/bin/buf || go install github.com/bufbuild/buf/cmd/buf@latest
	@echo "Installing protoc plugins..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Installing other tools..."
	go install github.com/golang/mock/mockgen@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest
	go install gotest.tools/gotestsum@latest

# Database migrations
migrate-up:
	migrate -path ./migrations -database "postgresql://localhost/narwhal?sslmode=disable" up

migrate-down:
	migrate -path ./migrations -database "postgresql://localhost/narwhal?sslmode=disable" down

migrate-create:
	migrate create -ext sql -dir ./migrations -seq $(name)

# Help
help:
	@printf "\
Available targets:\n\
\n\
  all              - Generate code and build all services\n\
  build            - Build all services\n\
  build-<service>  - Build specific service\n\
  run-<service>    - Run specific service\n\
  dev-<service>    - Run service with hot reload\n\
\n\
  proto            - Generate protobuf files (using Buf)\n\
  generate         - Generate all code (proto, mocks, etc.)\n\
  buf-lint         - Lint proto files with Buf\n\
  buf-breaking     - Check for breaking proto changes\n\
  buf-format       - Format proto files\n\
\n\
  test             - Run all tests\n\
  test-unit        - Run unit tests only\n\
  test-coverage    - Run tests with coverage\n\
  test-integration - Run integration tests\n\
  test-watch       - Run tests in watch mode\n\
\n\
  db-up            - Start PostgreSQL and Redis\n\
  db-down          - Stop databases\n\
  db-reset         - Reset databases\n\
  db-test          - Test database connection\n\
  migrate          - Run database migrations\n\
\n\
  lint             - Run linters\n\
  fmt              - Format code\n\
\n\
  docker-build     - Build Docker images\n\
  docker-up        - Start services with Docker Compose\n\
  docker-down      - Stop services\n\
\n\
  clean            - Clean build artifacts\n\
  install-tools    - Install development tools\n\
  help             - Show this help message\n"

.PHONY: test-workflows

test-workflows:
	cd .github/workflows/test && go test -v ./...

# Add workflow tests to main test target