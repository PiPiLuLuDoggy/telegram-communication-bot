# Makefile for Telegram Communication Bot

.PHONY: help build run test clean docker-build docker-run docker-stop docker-clean deps tidy fmt lint

# Default target
help:
	@echo "Available commands:"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  docker-stop  - Stop Docker Compose services"
	@echo "  docker-clean - Clean Docker resources"
	@echo "  deps         - Download dependencies"
	@echo "  tidy         - Tidy modules"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linters"

# Application name
APP_NAME := telegram-communication-bot
BINARY_NAME := telegram-bot

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	CGO_ENABLED=1 go build -o bin/$(BINARY_NAME) ./cmd/bot
	@echo "Build completed: bin/$(BINARY_NAME)"

# Run the application
run:
	@echo "Running $(APP_NAME)..."
	go run ./cmd/bot

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-stop:
	@echo "Stopping Docker Compose services..."
	docker-compose down

docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose down -v --rmi all
	docker system prune -f

# Production Docker commands
docker-prod-run:
	@echo "Starting production services..."
	docker-compose -f docker-compose.prod.yml up -d

docker-prod-stop:
	@echo "Stopping production services..."
	docker-compose -f docker-compose.prod.yml down

# Backup database (production)
backup:
	@echo "Running backup..."
	docker-compose -f docker-compose.prod.yml --profile backup up backup

# Go module commands
deps:
	@echo "Downloading dependencies..."
	go mod download

tidy:
	@echo "Tidying modules..."
	go mod tidy

# Code formatting and linting
fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Running linters..."
	golangci-lint run

# Install development dependencies
install-deps:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Development commands
dev-setup: install-deps deps
	@echo "Development environment setup completed"

# Generate example .env file
env-example:
	@echo "Creating .env from .env.example..."
	cp .env.example .env
	@echo "Please edit .env with your configuration"

# Check code quality
check: fmt lint test
	@echo "Code quality check completed"

# Release build (optimized)
release:
	@echo "Building release version..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/bot
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/bot
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/bot
	@echo "Release builds completed"

# Database migration (if needed in future)
migrate:
	@echo "Running database migrations..."
	# Add migration commands here if needed

# Log viewing
logs:
	@echo "Viewing application logs..."
	docker-compose logs -f telegram-bot

logs-prod:
	@echo "Viewing production logs..."
	docker-compose -f docker-compose.prod.yml logs -f telegram-bot