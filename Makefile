# CQAR Makefile
# Build automation for Crypto Quant Asset Registry

.PHONY: help
help: ## Display this help message
	@echo "CQAR - Crypto Quant Asset Registry"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Build variables
BINARY_NAME=cqar
BUILD_DIR=./cmd/server
OUTPUT_DIR=./bin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# Database migration settings
MIGRATE_DIR=./migrations
DB_URL?=postgres://cqar:cqar_dev_password@localhost:5432/cqar?sslmode=disable

.PHONY: build
build: ## Build the CQAR binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	@go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) $(BUILD_DIR)
	@echo "✓ Binary built: $(OUTPUT_DIR)/$(BINARY_NAME)"

.PHONY: build-linux
build-linux: ## Build the CQAR binary for Linux
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(OUTPUT_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 $(BUILD_DIR)
	@echo "✓ Binary built: $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64"

.PHONY: run
run: ## Run the CQAR service locally
	@echo "Running $(BINARY_NAME)..."
	@go run $(BUILD_DIR)/main.go

.PHONY: test
test: ## Run unit tests
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Unit tests completed"

.PHONY: test-coverage
test-coverage: test ## Run tests and generate coverage report
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -tags=integration ./test/integration/...
	@echo "✓ Integration tests completed"

.PHONY: clean
clean: ## Remove build artifacts and generated files
	@echo "Cleaning build artifacts..."
	@rm -rf $(OUTPUT_DIR)
	@rm -f coverage.out coverage.html
	@echo "✓ Cleaned"

.PHONY: deps
deps: ## Download and tidy dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies updated"

.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

.PHONY: lint
lint: ## Run linters
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	@golangci-lint run ./...
	@echo "✓ Linting completed"

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ go vet completed"

.PHONY: migrate-create
migrate-create: ## Create a new migration file (usage: make migrate-create NAME=create_assets_table)
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required. Usage: make migrate-create NAME=create_assets_table"; exit 1; fi
	@echo "Creating migration: $(NAME)..."
	@mkdir -p $(MIGRATE_DIR)
	@timestamp=$$(date -u '+%Y%m%d%H%M%S'); \
	touch $(MIGRATE_DIR)/$${timestamp}_$(NAME).up.sql; \
	touch $(MIGRATE_DIR)/$${timestamp}_$(NAME).down.sql; \
	echo "✓ Created migration files:"; \
	echo "  $(MIGRATE_DIR)/$${timestamp}_$(NAME).up.sql"; \
	echo "  $(MIGRATE_DIR)/$${timestamp}_$(NAME).down.sql"

.PHONY: migrate-up
migrate-up: ## Run database migrations up
	@echo "Running migrations..."
	@which migrate > /dev/null || (echo "golang-migrate not installed. Install from https://github.com/golang-migrate/migrate" && exit 1)
	@migrate -path $(MIGRATE_DIR) -database "$(DB_URL)" up
	@echo "✓ Migrations completed"

.PHONY: migrate-down
migrate-down: ## Rollback last database migration
	@echo "Rolling back migration..."
	@which migrate > /dev/null || (echo "golang-migrate not installed. Install from https://github.com/golang-migrate/migrate" && exit 1)
	@migrate -path $(MIGRATE_DIR) -database "$(DB_URL)" down 1
	@echo "✓ Migration rolled back"

.PHONY: migrate-force
migrate-force: ## Force migration version (usage: make migrate-force VERSION=1)
	@if [ -z "$(VERSION)" ]; then echo "Error: VERSION is required. Usage: make migrate-force VERSION=1"; exit 1; fi
	@echo "Forcing migration to version $(VERSION)..."
	@migrate -path $(MIGRATE_DIR) -database "$(DB_URL)" force $(VERSION)
	@echo "✓ Migration forced to version $(VERSION)"

.PHONY: migrate-status
migrate-status: ## Check migration status
	@echo "Checking migration status..."
	@which migrate > /dev/null || (echo "golang-migrate not installed. Install from https://github.com/golang-migrate/migrate" && exit 1)
	@migrate -path $(MIGRATE_DIR) -database "$(DB_URL)" version

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t cqar:$(VERSION) -t cqar:latest .
	@echo "✓ Docker image built: cqar:$(VERSION)"

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run --rm -p 8080:8080 -p 9090:9090 cqar:latest

.PHONY: proto-generate
proto-generate: ## Generate protobuf code (requires CQC)
	@echo "Note: Protobuf types are provided by CQC dependency"
	@echo "No local generation needed"

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✓ Tools installed"

.PHONY: all
all: deps fmt vet lint test build ## Run all checks and build

.PHONY: ci
ci: deps fmt vet test build ## CI pipeline target

.DEFAULT_GOAL := help
