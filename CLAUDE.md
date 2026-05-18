# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
make build           # Build all 5 services to ./bin
make dev-up          # Start infra (postgres + kafka) + all services via docker
make prod-up         # Production deploy (no port exposure)
make docker-up       # Infrastructure only (postgres + kafka)
make migrate         # Apply SQL migrations
make proto           # Generate gRPC code from proto/*.proto
make swagger         # Generate Swagger docs
make lint            # Run golangci-lint
make test            # Run all tests with race detection
```

Single test: `go test -v ./server/internal/user/... -run TestName`

## Architecture

GoTradeX is a real-time cryptocurrency trading platform. The codebase lives under `server/`.

### Services

| Service | Protocol | Port |
|---------|----------|------|
| API Gateway | REST/Gin | 8080 |
| User Service | gRPC | 50051 |
| Order Service | gRPC + Kafka | 50052 |
| Matching Engine | Kafka consumer | — |
| Market Service | WebSocket + Kafka | 8081 |

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

1. **Kafka for order matching** — Decouples Order Service from Matching Engine; enables replay for state recovery
2. **Per-symbol goroutines in matching engine** — Avoids cross-symbol lock contention; linear scaling with symbols
3. **Heap-based order book** — O(log n) insert/remove; natural price-time priority (FIFO at same price level)
4. **Transactional outbox** — Order + outbox message written in same DB transaction; outbox relay publishes to Kafka. Prevents "lost orders" if service crashes after DB commit but before Kafka publish
5. **SELECT FOR UPDATE for balance** — Row-level locking prevents race conditions on concurrent deductions
6. **segmentio/kafka-go** — Lightweight Go-native library over Sarama

### Go Workspace

`server/go.work` enables the monorepo. All `cmd/`, `internal/`, `pkg/`, `proto/` are under `server/`. Import path is `github.com/verno/gotradex`.

### Required Environment Variables

- `JWT_SECRET` — **Required**. Application will log fatal if not set.

Optional:
- `DATABASE_URL` — defaults to `postgres://postgres:postgres@localhost:5436/gotradex?sslmode=disable`
- `KAFKA_BROKERS` — defaults to `localhost:9092`

## Implemented Services

- **API Gateway** — Gin REST with JWT auth, rate limiter (100 rps per IP), Swagger UI at `/swagger/*any`
- **User Service** — gRPC with JWT issuance, user registration, balance management (PostgreSQL)

## Missing Services

The following are documented but not yet implemented:
- Order Service (`internal/order/`, outbox relay)
- Matching Engine (`internal/matching/`, heap, orderbook, state recovery)
- Market Service (`internal/market/`, WebSocket hub)
- Kafka pkg (`pkg/kafka/`, producer + consumer)

## Proto Files

gRPC definitions are in `server/proto/` (user.proto, order.proto). Generated code checked in (`*.pb.go`). Run `make proto` after modifying `.proto` files.