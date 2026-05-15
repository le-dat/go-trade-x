# Step 01 — Infrastructure: GoTradeX System Overview

## Goal

Establish the complete GoTradeX architecture before writing any code.

---

## System Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                                Client                                    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         API Gateway (:8080)                              │
│                    Gin REST + JWT Auth + Rate Limiting                   │
└─────────────────────────────────────────────────────────────────────────┘
         │                                        │
         │ gRPC                                  │ WebSocket
         ▼                                        ▼
┌──────────────────────┐              ┌──────────────────────────────────┐
│   User Service       │              │        Market Service             │
│   (:50051)          │              │        (:8081 WS)                 │
│   gRPC              │              │   Kafka Consumer + WS Broadcast    │
└──────────────────────┘              └──────────────────────────────────┘
         │                                        ▲
         │                                        │
         ▼                                        │
┌──────────────────────┐                            │
│   Order Service      │         Kafka [trades]      │
│   (:50052)          │◄────────────────────────────┘
│   gRPC + Kafka      │
└──────────────────────┘
         │
         │ Kafka [orders]
         ▼
┌──────────────────────┐
│  Matching Engine     │
│  Kafka Consumer      │
│  In-Memory Orderbook │
└──────────────────────┘
```

---

## Services

| Service | Protocol | Port | Role |
|---------|----------|------|------|
| API Gateway | REST (Gin) | 8080 | Auth, routing, rate limiting, JWT validation |
| User Service | gRPC | 50051 | Register, login, JWT issuance, balance management |
| Order Service | gRPC + Kafka | 50052 | Validate balance, persist order, publish to Kafka |
| Matching Engine | Kafka consumer | - | In-memory order book, matching logic, emit trades |
| Market Service | Kafka consumer + WS | 8081 | Consume trades, broadcast real-time to WebSocket clients |

---

## Kafka Topics

| Topic | Partitions | Retention | Producer | Consumer |
|-------|-----------|-----------|----------|----------|
| `orders` | 4 | 7 days | Order Service | Matching Engine |
| `trades` | 4 | 30 days | Matching Engine | Market Service, Order Service, User Service |

---

## Repository Structure

```
/trading-platform
├── cmd/
│   ├── api-gateway/        main.go
│   ├── user-service/       main.go
│   ├── order-service/      main.go
│   ├── matching-engine/    main.go
│   └── market-service/     main.go
├── internal/
│   ├── user/               handler, service, repository
│   ├── order/              handler, service, repository
│   ├── matching/           engine, orderbook, heap
│   └── market/             hub, client
├── pkg/
│   ├── auth/               JWT utilities
│   ├── config/             env loader
│   ├── database/           postgres connection
│   ├── kafka/              producer, consumer wrappers
│   └── logger/             zap setup
├── proto/                  .proto files + generated code
├── migrations/             SQL migration files
├── docker/                 per-service Dockerfiles
├── docker-compose.yml
├── Makefile
└── go.work                 Go workspace (monorepo)
```

---

## Technology Stack

- **Language**: Go 1.22+
- **REST API**: Gin Framework
- **Service Communication**: gRPC
- **Message Queue**: Apache Kafka (segmentio/kafka-go)
- **Database**: PostgreSQL (pgx/v5)
- **Cache**: Redis
- **WebSocket**: gorilla/websocket
- **Auth**: JWT (HS256)
- **Logging**: Uber Zap
- **Container**: Docker + Docker Compose

---

## Order Lifecycle

```
1. Client POST /api/v1/orders
2. Gateway validates JWT → calls Order Service (gRPC)
3. Order Service calls User Service → check balance
4. Order Service persists order (status: PENDING)
5. Order Service publishes to Kafka [orders]
6. Matching Engine consumes → attempts match
7a. Match found → publish to Kafka [trades]
7b. No match → order stays in order book
8. Market Service consumes [trades] → broadcasts via WebSocket
9. Order/User Services consume [trades] → update balances & order status
```

---

## Engineering Standards

See [05-architecture/12-architecture.md](../05-architecture/12-architecture.md) for:
- Context propagation rules
- Error handling patterns
- Graceful shutdown
- Commit conventions

> ➡️ Next: [Step 02 — API Gateway](../02-api-gateway/02-api-gateway.md)