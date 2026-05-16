BINARY_DIR=bin
PROTO_DIR=proto
MIGRATIONS_DIR=migrations
DATABASE_URL?=postgres://postgres:postgres@localhost:5436/gotradex?sslmode=disable

.PHONY: build clean run-api run-user run-order run-matching run-market run-all \
	dev-up prod-up docker-up docker-down docker-build docker-start docker-stop docker-logs \
	docker-migrate docker-logs-api docker-logs-user docker-logs-order docker-logs-matching docker-logs-market \
	proto migrate migrate-up migrate-down lint help swagger \
	verify-user verify-order-service

build:
	mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_DIR)/api-gateway ./cmd/api-gateway
	go build -o $(BINARY_DIR)/user-service ./cmd/user-service
	go build -o $(BINARY_DIR)/order-service ./cmd/order-service
	go build -o $(BINARY_DIR)/matching-engine ./cmd/matching-engine
	go build -o $(BINARY_DIR)/market-service ./cmd/market-service

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto

migrate-up:
	@echo "Running migrations from $(MIGRATIONS_DIR)..."
	@echo "Using DATABASE_URL=$(DATABASE_URL)"
	@for f in $(MIGRATIONS_DIR)/*.up.sql; do \
		echo "Applying $$f..."; \
		psql "$(DATABASE_URL)" -f "$$f" || exit 1; \
	done
	@echo "Migrations complete."

migrate-down:
	@echo "Rolling back migrations..."
	@for f in $(MIGRATIONS_DIR)/*.down.sql; do \
		echo "Rolling back $$f..."; \
		psql "$(DATABASE_URL)" -f "$$f" || true; \
	done
	@echo "Rollback complete."

migrate: migrate-up

lint:
	golangci-lint run

swagger:
	$(shell go env GOPATH)/bin/swag init -g cmd/api-gateway/main.go -o cmd/api-gateway/docs

docker-build:
	docker compose build

docker-up:
	docker compose up -d

dev-up:
	docker compose -f docker-compose.yml -f docker-compose.override.yml up -d

prod-up:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d

docker-start:
	docker compose up -d

docker-stop:
	docker compose down

docker-logs:
	docker compose logs -f

docker-logs-api:
	docker compose logs -f api-gateway

docker-logs-user:
	docker compose logs -f user-service

docker-logs-order:
	docker compose logs -f order-service

docker-logs-matching:
	docker compose logs -f matching-engine

docker-logs-market:
	docker compose logs -f market-service

docker-migrate: migrate-up

docker-down:
	docker compose down

postgres-up:
	docker compose up -d postgres

clean:
	rm -rf $(BINARY_DIR)

run-api:
	go run ./cmd/api-gateway

run-user:
	go run ./cmd/user-service

run-order:
	go run ./cmd/order-service

run-matching:
	go run ./cmd/matching-engine

run-market:
	go run ./cmd/market-service

run-all:
	go run ./cmd/user-service & \
	go run ./cmd/order-service & \
	go run ./cmd/matching-engine & \
	go run ./cmd/market-service & \
	go run ./cmd/api-gateway

test:
	go test ./...

# Verification targets
verify-user: migrate-up
	@echo "Starting user service for verification..."
	@timeout 15s env DATABASE_URL="$(DATABASE_URL)" PATH="$(PATH):$(GOPATH)/bin" go run ./cmd/user-service &
	@sleep 4
	@echo ""
	@echo "=== Testing Register ==="
	@PATH="$(PATH):$(GOPATH)/bin" grpcurl -plaintext -d '{"email":"test2@test.com","password":"secret123"}' localhost:50051 user.UserService/Register || true
	@echo ""
	@echo "=== Testing Login ==="
	@PATH="$(PATH):$(GOPATH)/bin" grpcurl -plaintext -d '{"email":"test@test.com","password":"secret123"}' localhost:50051 user.UserService/Login || true
	@echo ""
	@echo "=== Testing GetBalance ==="
	@PATH="$(PATH):$(GOPATH)/bin" grpcurl -plaintext -d '{"user_id":"0bfcd9c3-0a81-4852-afaf-63fba9a32148"}' localhost:50051 user.UserService/GetBalance || true
	@echo ""
	@ps aux | grep "go run ./cmd/user-service" | grep -v grep | awk '{print $$2}' | xargs -r kill 2>/dev/null || true
	@echo "Verification complete."

verify-order-service: migrate-up
	@echo "Order service verification (requires user-service running)"
	@echo "Run 'make run-user' in another terminal first"

help:
	@echo "Available targets:"
	@echo "  build              - Build all services"
	@echo "  proto              - Generate gRPC code"
	@echo "  swagger            - Generate Swagger documentation"
	@echo "  migrate            - Run all up migrations"
	@echo "  migrate-down       - Roll back all migrations"
	@echo "  lint               - Run golangci-lint"
	@echo "  dev-up             - Start all services via docker (local dev)"
	@echo "  prod-up            - Start services for production (no port exposure)"
	@echo "  docker-up          - Start infrastructure (postgres + kafka)"
	@echo "  docker-start       - Start all services via docker compose"
	@echo "  docker-stop        - Stop all docker compose services"
	@echo "  docker-down        - Stop infrastructure"
	@echo "  docker-build       - Build all service images"
	@echo "  docker-logs        - Follow all docker logs"
	@echo "  docker-logs-<svc>  - Follow logs for a specific service"
	@echo "  docker-migrate     - Run migrations on docker postgres"
	@echo "  test               - Run all tests"
	@echo "  run-all            - Run all services locally"
	@echo "  run-<service>      - Run specific service locally"
	@echo "  verify-user        - Run migrations and verify user-service with grpcurl"
