.PHONY: help dev check test build clean docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

dev: ## Run the development server
	@go run cmd/server/main.go

check: ## Run connection health checks
	@go run cmd/check/main.go

test: ## Run tests
	@go test -v ./...

build: ## Build the server binary
	@go build -o bin/server cmd/server/main.go
	@go build -o bin/check cmd/check/main.go
	@echo "Binaries built in ./bin/"

clean: ## Clean build artifacts
	@rm -rf bin/
	@echo "Cleaned build artifacts"

docker-up: ## Start Docker services (PostgreSQL + Redis)
	@docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Services started. Run 'make check' to verify connections."

docker-down: ## Stop Docker services
	@docker-compose down
	@echo "Services stopped"

docker-logs: ## View Docker service logs
	@docker-compose logs -f

tidy: ## Tidy go modules
	@go mod tidy
	@echo "Go modules tidied"

install: ## Install all dependencies
	@go mod download
	@echo "Dependencies installed"
