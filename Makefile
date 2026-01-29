.PHONY: build run test clean swag init

# Configuration
APP_NAME := alpha-hygiene-backend
BINARY_NAME := $(APP_NAME)
MAIN_PACKAGE := ./cmd/app
SWAG := $(shell go env GOPATH)/bin/swag

# Colors for output
RED := $(shell tput -Txterm setaf 1)
GREEN := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET := $(shell tput -Txterm sgr0)

# Default target
all: build

# Build the application
build: swag
	@echo "$(GREEN)Building $(APP_NAME)...$(RESET)"
	@go build -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Run the application
run: swag
	@echo "$(GREEN)Running $(APP_NAME)...$(RESET)"
	@go run $(MAIN_PACKAGE)

# Run tests
test:
	@echo "$(GREEN)Running tests...$(RESET)"
	@go test -v ./...

# Clean up binary
clean:
	@echo "$(YELLOW)Cleaning up...$(RESET)"
	@rm -f $(BINARY_NAME)
	@rm -rf docs/

# Initialize swagger documentation
swag: $(SWAG)
	@echo "$(GREEN)Generating Swagger documentation...$(RESET)"
	@$(SWAG) init -g $(MAIN_PACKAGE)/main.go

# Install swag CLI if not present
$(SWAG):
	@echo "$(YELLOW)Installing swag CLI...$(RESET)"
	@go install github.com/swaggo/swag/v2/cmd/swag@latest

# Format code
fmt:
	@echo "$(GREEN)Formatting code...$(RESET)"
	@go fmt ./...

# Vet code
vet:
	@echo "$(GREEN)Vetting code...$(RESET)"
	@go vet ./...

# Lint code (requires golangci-lint installed)
lint:
	@echo "$(GREEN)Linting code...$(RESET)"
	@golangci-lint run ./...

# Create coverage report
coverage:
	@echo "$(GREEN)Generating coverage report...$(RESET)"
	@go test -v ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html

# Show coverage
show-coverage: coverage
	@echo "$(GREEN)Coverage report generated: coverage.html$(RESET)"

# Help
help:
	@echo "$(GREEN)Available commands:$(RESET)"
	@echo "  $(YELLOW)build$(RESET)      - Build the application"
	@echo "  $(YELLOW)run$(RESET)        - Run the application"
	@echo "  $(YELLOW)test$(RESET)       - Run tests"
	@echo "  $(YELLOW)clean$(RESET)      - Clean up binary"
	@echo "  $(YELLOW)swag$(RESET)       - Generate Swagger documentation"
	@echo "  $(YELLOW)fmt$(RESET)        - Format code"
	@echo "  $(YELLOW)vet$(RESET)        - Vet code"
	@echo "  $(YELLOW)lint$(RESET)       - Lint code"
	@echo "  $(YELLOW)coverage$(RESET)   - Create coverage report"
	@echo "  $(YELLOW)help$(RESET)       - Show this help"
