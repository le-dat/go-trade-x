BINARY_DIR := bin
PROTO_DIR := proto
MIGRATIONS_DIR := migrations
DATABASE_URL ?= postgres://postgres:postgres@localhost:5436/gotradex?sslmode=disable

# Services
SERVICES := api-gateway user-service order-service matching-engine market-service
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -s -w"

.PHONY: all help build clean test lint proto swagger migrate migrate-down \
	dev-up dev-down dev-logs prod-up prod-down prod-logs docker-build

all: build

help: ## Show this help message
	@echo "GoTradeX Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ==============================================================================
# Development Environment
# ==============================================================================

dev-up: infra-up migrate ## [DEV] Start all services (Local) + Auto Migrate
	docker compose -f docker-compose.yml -f docker-compose.override.yml up -d --remove-orphans
	@echo "Development environment is running."
	@echo "  API Gateway: http://localhost:8080"

dev-down: ## [DEV] Stop all services
	docker compose -f docker-compose.yml -f docker-compose.override.yml down

dev-logs: ## [DEV] Follow all logs
	docker compose -f docker-compose.yml -f docker-compose.override.yml logs -f --tail=100

# ==============================================================================
# Production Environment
# ==============================================================================

prod-up: infra-up migrate ## [PROD] Start all services (Production) + Auto Migrate
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --remove-orphans
	@echo "Production environment is running."

prod-down: ## [PROD] Stop all services
	docker compose -f docker-compose.yml -f docker-compose.prod.yml down

prod-pull: ## [PROD] Pull all images
	docker compose -f docker-compose.yml -f docker-compose.prod.yml pull

prod-restart: ## [PROD] Stop and start all services
	docker compose -f docker-compose.yml -f docker-compose.prod.yml down
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --remove-orphans

prod-logs: ## [PROD] Follow all logs
	docker compose -f docker-compose.yml -f docker-compose.prod.yml logs -f --tail=100


# ==============================================================================
# Internal Infrastructure Helpers
# ==============================================================================

infra-up:
	@docker compose up -d postgres kafka
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker compose exec -T postgres pg_isready -U postgres >/dev/null 2>&1; do \
		echo "Waiting for DB..."; \
		sleep 2; \
	done
	@echo "Infrastructure is ready."

# ==============================================================================
# Build & Tools
# ==============================================================================

build: $(SERVICES) ## Build all 5 services in parallel to ./bin

run-order: ## Run order-service locally
	go run ./cmd/order-service

$(SERVICES):
	@echo "Building $@ (version: $(VERSION))..."
	@mkdir -p $(BINARY_DIR)
	@go build $(LDFLAGS) -o $(BINARY_DIR)/$@ ./cmd/$@

docker-build: ## Build all service docker images
	docker compose build

proto: ## Generate gRPC code from proto/*.proto
	protoc --go_out=. --go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--go_opt=paths=source_relative $(PROTO_DIR)/*.proto

swagger: ## Generate Swagger documentation
	swag init -g cmd/api-gateway/main.go -o cmd/api-gateway/docs

migrate: ## Apply all SQL up migrations
	@echo "Running migrations..."
	@find $(MIGRATIONS_DIR) -name "*.up.sql" | sort | xargs -I {} sh -c 'echo "  Applying {}..."; psql "$(DATABASE_URL)" -f {}'

migrate-down: ## Roll back all SQL migrations
	@echo "Rolling back migrations..."
	@find $(MIGRATIONS_DIR) -name "*.down.sql" | sort -r | xargs -I {} sh -c 'echo "  Rolling back {}..."; psql "$(DATABASE_URL)" -f {}'

lint: ## Run golangci-lint
	golangci-lint run

test: ## Run all tests with race detection
	go test -v -race ./...

clean: ## Remove build binaries
	rm -rf $(BINARY_DIR)