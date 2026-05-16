# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
make build              # Build all 5 services to ./bin
make dev-up             # Start all services via docker (local dev)
make prod-up            # Start services for production (no port exposure)
make run-all            # Run all services locally
make run-api            # Run only API Gateway
make run-user           # Run only User Service
make run-order          # Run only Order Service
make run-matching       # Run only Matching Engine
make run-market          # Run only Market Service
make test               # Run all tests
make lint               # Run golangci-lint
make proto              # Generate gRPC code from proto/*.proto
make migrate            # Run SQL migrations
make docker-up          # Start PostgreSQL, Kafka via docker compose
make docker-stop        # Stop all docker compose services
make docker-down        # Stop infrastructure
make docker-logs        # Follow all docker logs
make docker-logs-<svc>  # Follow specific service logs
```

## Architecture

GoTradeX is a real-time cryptocurrency trading platform with 5 microservices:

- **API Gateway** (Gin/REST) — Entry point, handles auth, routing, rate limiting
- **User Service** (gRPC) — JWT auth, user registration, balance management (PostgreSQL)
- **Order Service** (gRPC + Kafka) — Validates balances, persists orders, publishes to Kafka `orders` topic
- **Matching Engine** (Kafka Consumer) — In-memory order book per symbol (max-heap bids, min-heap asks), emits trades to Kafka `trades` topic
- **Market Service** (Kafka Consumer + WebSocket) — Broadcasts real-time trades to WebSocket clients

### Communication Flow
```
Client ─(REST)─► API Gateway ─(gRPC)─► Order Service ─(produce)─► Kafka [orders]
                                                                        │
                                                                   (consume)
                                                                        ▼
                                                              Matching Engine
                                                                        │
                                                                   (produce)
                                                                        ▼
Client ◄(ws://)◄ Market Service ◄(consume)────────────────────────── Kafka [trades]
```

### Key Design Decisions

1. **Kafka over direct gRPC for matching** — Decouples Order Service from Matching Engine, enables replay
2. **Per-symbol goroutines in matching engine** — Avoids cross-symbol lock contention
3. **Heap-based order book** — O(log n) insert/remove, natural price-time priority
4. **SELECT FOR UPDATE for balance** — Row-level locking prevents race conditions on concurrent deductions
5. **segmentio/kafka-go** — Lightweight, Go-native, simpler API than Sarama

## Service Ports

| Service | Port | Protocol |
|---------|------|----------|
| API Gateway | 8080 | REST/HTTP |
| User Service | 50051 | gRPC |
| Market Service | 8081 | WebSocket |

## Database

PostgreSQL on port 5436 (local) / 5432 (docker). Connection: `postgres://postgres:postgres@localhost:5436/gotradex?sslmode=disable`

## Proto Files

gRPC definitions are in `proto/` with a replace directive pointing back to the local module. Run `make proto` after modifying `.proto` files.