BINARY_DIR=bin

.PHONY: build clean run-api run-user run-order run-matching run-market docker-up docker-down docker-build

build:
	mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_DIR)/api-gateway ./cmd/api-gateway
	go build -o $(BINARY_DIR)/user-service ./cmd/user-service
	go build -o $(BINARY_DIR)/order-service ./cmd/order-service
	go build -o $(BINARY_DIR)/matching-engine ./cmd/matching-engine
	go build -o $(BINARY_DIR)/market-service ./cmd/market-service

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

test:
	go test ./...
