# Makefile for Exam System Backend

# Variables
BINARY_NAME=exam-system
MAIN_PATH=./main.go
BUILD_DIR=./build
DOCKER_IMAGE=exam-system-backend

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Build for Linux (useful for deployment)
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(MAIN_PATH)

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run $(MAIN_PATH)

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@echo "Starting development server with hot reload..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Running without hot reload..."; \
		$(GOCMD) run $(MAIN_PATH); \
	fi

# Test the application
test:
	@echo "Running tests..."
	$(GOTEST) -v ./tests/...

# Test with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./tests/...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests in watch mode (requires gotestsum: go install gotest.tools/gotestsum@latest)
test-watch:
	@if command -v gotestsum > /dev/null; then \
		gotestsum --watch ./tests/...; \
	else \
		echo "gotestsum not found. Install with: go install gotest.tools/gotestsum@latest"; \
		$(GOTEST) -v ./tests/...; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	@if command -v golangci-lint > /dev/null; then \
		echo "Running linter..."; \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install from: https://golangci-lint.run/usage/install/"; \
	fi

# Security check (requires gosec: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
security:
	@if command -v gosec > /dev/null; then \
		echo "Running security check..."; \
		gosec ./...; \
	else \
		echo "gosec not found. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Database migrations (requires migrate CLI)
migrate-up:
	@if command -v migrate > /dev/null; then \
		echo "Running database migrations..."; \
		migrate -path ./migrations -database "postgres://postgres:password@localhost:5432/exam_system?sslmode=disable" up; \
	else \
		echo "migrate CLI not found. Install from: https://github.com/golang-migrate/migrate"; \
	fi

migrate-down:
	@if command -v migrate > /dev/null; then \
		echo "Rolling back database migrations..."; \
		migrate -path ./migrations -database "postgres://postgres:password@localhost:5432/exam_system?sslmode=disable" down; \
	else \
		echo "migrate CLI not found. Install from: https://github.com/golang-migrate/migrate"; \
	fi

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE)

docker-compose-up:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-compose-down:
	@echo "Stopping services with Docker Compose..."
	docker-compose down

# Setup development environment
setup-dev:
	@echo "Setting up development environment..."
	@echo "Installing development tools..."
	$(GOGET) github.com/cosmtrek/air@latest
	$(GOGET) gotest.tools/gotestsum@latest
	@echo "Copying example environment file..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Please update .env with your configuration"; \
	fi
	@echo "Development environment setup complete!"

# Generate API documentation (requires swag: go install github.com/swaggo/swag/cmd/swag@latest)
docs:
	@if command -v swag > /dev/null; then \
		echo "Generating API documentation..."; \
		swag init -g main.go; \
	else \
		echo "swag not found. Install with: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi

# Check for vulnerabilities (requires govulncheck: go install golang.org/x/vuln/cmd/govulncheck@latest)
vuln-check:
	@if command -v govulncheck > /dev/null; then \
		echo "Checking for vulnerabilities..."; \
		govulncheck ./...; \
	else \
		echo "govulncheck not found. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

# Full check (format, lint, test, security, vulnerabilities)
check: fmt lint test security vuln-check
	@echo "All checks completed!"

# Help
help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  build-linux    - Build for Linux"
	@echo "  run            - Run the application"
	@echo "  dev            - Run with hot reload"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  test-watch     - Run tests in watch mode"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  deps-update    - Update dependencies"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  security       - Run security check"
	@echo "  migrate-up     - Run database migrations"
	@echo "  migrate-down   - Rollback database migrations"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  docker-compose-up   - Start services with Docker Compose"
	@echo "  docker-compose-down - Stop services with Docker Compose"
	@echo "  setup-dev      - Setup development environment"
	@echo "  docs           - Generate API documentation"
	@echo "  vuln-check     - Check for vulnerabilities"
	@echo "  check          - Run all checks"
	@echo "  help           - Show this help"

.PHONY: build build-linux run dev test test-coverage test-watch clean deps deps-update fmt lint security migrate-up migrate-down docker-build docker-run docker-compose-up docker-compose-down setup-dev docs vuln-check check help

