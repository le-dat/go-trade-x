# Architecture — GoTradeX

Real-time Trading & Order Matching Platform

## System Overview

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

## Services

| Service | Protocol | Port | Role |
|---------|----------|------|------|
| API Gateway | REST (Gin) | 8080 | Auth, routing, rate limiting, JWT validation |
| User Service | gRPC | 50051 | Register, login, JWT issuance, balance management |
| Order Service | gRPC + Kafka | 50052 | Validate balance, persist order, publish to Kafka |
| Matching Engine | Kafka consumer | - | In-memory order book, matching logic, emit trades |
| Market Service | Kafka consumer + WS | 8081 | Consume trades, broadcast real-time to WebSocket clients |

## Kafka Topics

| Topic | Partitions | Retention | Producer | Consumer |
|-------|-----------|-----------|----------|----------|
| `orders` | 4 | 7 days | Order Service | Matching Engine |
| `trades` | 4 | 30 days | Matching Engine | Market Service, Order Service, User Service |

## Data Flow

### Order Placement Flow

```
1. Client POST /api/v1/orders (with JWT)
2. API Gateway validates JWT → extracts userID
3. API Gateway calls OrderService.PlaceOrder (gRPC)
4. Order Service calls UserService.DeductBalance (gRPC)
5. Order Service persists order (status: PENDING)
6. Order Service publishes to Kafka [orders]
7. Matching Engine consumes → attempts match
8a. Match found → publish to Kafka [trades]
8b. No match → order stays in order book
9. Market Service consumes [trades] → broadcasts via WebSocket
10. Order/User Services consume [trades] → update balances & order status
```

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

## Technology Stack

- **Framework**: Gin (REST), gRPC (service-to-service)
- **Language**: Go 1.22+
- **Database**: PostgreSQL
- **Message Queue**: Apache Kafka
- **Cache**: Redis
- **WebSocket**: gorilla/websocket
- **Auth**: JWT (HS256)
- **Logging**: Uber Zap
- **Container**: Docker + Docker Compose

## Engineering Standards

See `CLAUDE.md` for:
- Global engineering standards (context propagation, error handling, interfaces, logging, graceful shutdown)
- Phase-by-phase execution checklist
- Key dependencies and versions
