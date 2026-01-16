# BreatheRoute Makefile
# Run 'make help' to see available commands

.PHONY: help dev dev-down dev-logs dev-tools db-shell redis-shell \
        build test lint fmt clean \
        api worker migrate \
        docker-build docker-push

# Default target
.DEFAULT_GOAL := help

# Variables
GO := go
GOFLAGS := -race
DOCKER_COMPOSE := docker compose
PROJECT := breatheroute
REGISTRY := europe-west4-docker.pkg.dev
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Colors for output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RESET := \033[0m

## help: Show this help message
help:
	@echo "$(CYAN)BreatheRoute Development Commands$(RESET)"
	@echo ""
	@echo "$(GREEN)Development:$(RESET)"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'

#
# Development Environment
#

## dev: Start local development dependencies (Postgres, Redis)
dev:
	@echo "$(CYAN)Starting development environment...$(RESET)"
	$(DOCKER_COMPOSE) up -d postgres redis
	@echo "$(GREEN)Waiting for services to be healthy...$(RESET)"
	@sleep 3
	@$(DOCKER_COMPOSE) ps
	@echo ""
	@echo "$(GREEN)Development environment ready!$(RESET)"
	@echo "  PostgreSQL: localhost:5432 (user: breatheroute, password: localdev)"
	@echo "  Redis:      localhost:6379"

## dev-tools: Start development tools (pgAdmin, Redis Commander, MailHog)
dev-tools:
	@echo "$(CYAN)Starting development tools...$(RESET)"
	$(DOCKER_COMPOSE) --profile tools up -d
	@echo ""
	@echo "$(GREEN)Tools available at:$(RESET)"
	@echo "  pgAdmin:         http://localhost:8082 (admin@breatheroute.local / localdev)"
	@echo "  Redis Commander: http://localhost:8081"
	@echo "  MailHog:         http://localhost:8025"

## dev-down: Stop all development containers
dev-down:
	@echo "$(CYAN)Stopping development environment...$(RESET)"
	$(DOCKER_COMPOSE) --profile tools down

## dev-clean: Stop containers and remove volumes (fresh start)
dev-clean:
	@echo "$(YELLOW)Stopping containers and removing volumes...$(RESET)"
	$(DOCKER_COMPOSE) --profile tools down -v
	@echo "$(GREEN)Clean environment ready for fresh start$(RESET)"

## dev-logs: Tail logs from development containers
dev-logs:
	$(DOCKER_COMPOSE) logs -f

## db-shell: Open PostgreSQL shell
db-shell:
	$(DOCKER_COMPOSE) exec postgres psql -U breatheroute -d breatheroute

## redis-shell: Open Redis CLI
redis-shell:
	$(DOCKER_COMPOSE) exec redis redis-cli

#
# Build & Test
#

## build: Build all binaries
build:
	@echo "$(CYAN)Building binaries...$(RESET)"
	$(GO) build -ldflags="-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)" -o bin/api ./cmd/api
	$(GO) build -ldflags="-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)" -o bin/worker ./cmd/worker
	@echo "$(GREEN)Build complete: bin/api, bin/worker$(RESET)"

## api: Build and run the API service
api: build
	@echo "$(CYAN)Starting API service...$(RESET)"
	./bin/api

## worker: Build and run the Worker service
worker: build
	@echo "$(CYAN)Starting Worker service...$(RESET)"
	./bin/worker

## test: Run all tests
test:
	@echo "$(CYAN)Running tests...$(RESET)"
	$(GO) test $(GOFLAGS) -coverprofile=coverage.out ./...
	@echo "$(GREEN)Tests complete. Coverage report: coverage.out$(RESET)"

## test-coverage: Run tests and show coverage report
test-coverage: test
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(RESET)"

## lint: Run linter
lint:
	@echo "$(CYAN)Running linter...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "$(YELLOW)golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(RESET)"; \
	fi

## fmt: Format code
fmt:
	@echo "$(CYAN)Formatting code...$(RESET)"
	$(GO) fmt ./...
	@echo "$(GREEN)Formatting complete$(RESET)"

## clean: Remove build artifacts
clean:
	@echo "$(CYAN)Cleaning build artifacts...$(RESET)"
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean complete$(RESET)"

#
# Database Migrations
#

## migrate-up: Run database migrations
migrate-up:
	@echo "$(CYAN)Running migrations...$(RESET)"
	@if command -v migrate >/dev/null 2>&1; then \
		migrate -path ./migrations -database "postgres://breatheroute:localdev@localhost:5432/breatheroute?sslmode=disable" up; \
	else \
		echo "$(YELLOW)golang-migrate not installed. Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest$(RESET)"; \
	fi

## migrate-down: Rollback last migration
migrate-down:
	@echo "$(YELLOW)Rolling back last migration...$(RESET)"
	migrate -path ./migrations -database "postgres://breatheroute:localdev@localhost:5432/breatheroute?sslmode=disable" down 1

## migrate-create: Create a new migration (usage: make migrate-create name=create_users)
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "$(YELLOW)Usage: make migrate-create name=migration_name$(RESET)"; \
	else \
		migrate create -ext sql -dir ./migrations -seq $(name); \
		echo "$(GREEN)Migration created in ./migrations$(RESET)"; \
	fi

#
# Docker
#

## docker-build: Build Docker images
docker-build:
	@echo "$(CYAN)Building Docker images...$(RESET)"
	docker build -t $(PROJECT)-api:$(VERSION) -f Dockerfile.api .
	docker build -t $(PROJECT)-worker:$(VERSION) -f Dockerfile.worker .
	@echo "$(GREEN)Images built: $(PROJECT)-api:$(VERSION), $(PROJECT)-worker:$(VERSION)$(RESET)"

## docker-push: Push Docker images to registry (requires GCP auth)
docker-push: docker-build
	@echo "$(CYAN)Pushing images to registry...$(RESET)"
	@if [ -z "$(PROJECT_ID)" ]; then \
		echo "$(YELLOW)PROJECT_ID not set. Usage: make docker-push PROJECT_ID=breatheroute-staging$(RESET)"; \
	else \
		docker tag $(PROJECT)-api:$(VERSION) $(REGISTRY)/$(PROJECT_ID)/$(PROJECT)/api:$(VERSION); \
		docker tag $(PROJECT)-worker:$(VERSION) $(REGISTRY)/$(PROJECT_ID)/$(PROJECT)/worker:$(VERSION); \
		docker push $(REGISTRY)/$(PROJECT_ID)/$(PROJECT)/api:$(VERSION); \
		docker push $(REGISTRY)/$(PROJECT_ID)/$(PROJECT)/worker:$(VERSION); \
		echo "$(GREEN)Images pushed to $(REGISTRY)/$(PROJECT_ID)/$(PROJECT)$(RESET)"; \
	fi

#
# Utilities
#

## deps: Download Go dependencies
deps:
	@echo "$(CYAN)Downloading dependencies...$(RESET)"
	$(GO) mod download
	$(GO) mod tidy
	@echo "$(GREEN)Dependencies ready$(RESET)"

## generate: Run go generate
generate:
	@echo "$(CYAN)Running go generate...$(RESET)"
	$(GO) generate ./...

## check: Run all checks (fmt, lint, test)
check: fmt lint test
	@echo "$(GREEN)All checks passed!$(RESET)"
