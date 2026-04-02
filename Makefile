BINARY_DIR=bin
PROTO_DIR=proto
MIGRATIONS_DIR=migrations

.PHONY: build clean run-api run-user run-order run-matching run-market run-all docker-up docker-down docker-build proto migrate lint help

build:
	mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_DIR)/api-gateway ./cmd/api-gateway
	go build -o $(BINARY_DIR)/user-service ./cmd/user-service
	go build -o $(BINARY_DIR)/order-service ./cmd/order-service
	go build -o $(BINARY_DIR)/matching-engine ./cmd/matching-engine
	go build -o $(BINARY_DIR)/market-service ./cmd/market-service

proto:
	# Install protoc-gen-go and protoc-gen-go-grpc first:
	# go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	# go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto

migrate:
	@echo "Running migrations from $(MIGRATIONS_DIR)..."

lint:
	golangci-lint run

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

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

help:
	@echo "Available targets:"
	@echo "  build         - Build all services"
	@echo "  proto         - Generate gRPC code"
	@echo "  migrate       - Run database migrations"
	@echo "  lint          - Run golangci-lint"
	@echo "  docker-up     - Start infrastructure with docker-compose"
	@echo "  docker-down   - Stop infrastructure"
	@echo "  test          - Run all tests"
	@echo "  run-all       - Run all services locally"
	@echo "  run-<service> - Run specific service locally"
