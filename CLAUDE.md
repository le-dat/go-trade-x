# CLAUDE.md — GoTradeX

> [ [Spec](docs/spec-doc.md) ] [ [Architecture](docs/architecture.md) ] [ [Plan](docs/project-plan.md) ] [ [Status](docs/project-status.md) ] [ [Changelog](docs/changelog.md) ]

> Claude reads this at the start of every session for core rules and tech stack.

---

## 1. Project Overview

**Product:** GoTradeX — Real-time Trading & Order Matching Platform
**Core mechanic:** Order matching engine with Kafka-based event streaming, WebSocket market data broadcast, and clean architecture microservices
**Links:** [Detailed Specification](docs/spec-doc.md) | [System Architecture](docs/architecture.md)

---

## 2. Repository Structure

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

## 3. Core Logic

Detailed in [Project Specification](docs/spec-doc.md).

The platform matches BUY/SELL orders in real-time using a price-time priority heap-based order book per symbol. Orders flow through Kafka topics — `orders` for placement, `trades` for execution — with a matching engine consuming and emitting trade events consumed by Market Service for WebSocket broadcast.

---

## 4. System Architecture

Detailed in [Architecture Design](docs/architecture.md).

### Services & Protocols

| Service | Protocol | Role |
|---------|----------|------|
| API Gateway | REST (Gin :8080) | Auth, routing, rate limiting, JWT validation |
| User Service | gRPC (:50051) | Register, login, JWT issuance, balance management |
| Order Service | gRPC (:50052) + Kafka | Validate balance, persist order, publish to Kafka |
| Matching Engine | Kafka consumer | In-memory order book, matching logic, emit trades |
| Market Service | Kafka consumer + WS (:8081) | Consume trades, broadcast real-time to WebSocket clients |

### Core Coding Patterns

- Context propagation on all calls (never `context.Background()` in handlers)
- Explicit error handling — no silent ignores
- Interfaces for all dependencies (testability)
- Structured logging via `zap`
- Graceful shutdown with signal handling (`SIGTERM`, `SIGINT`)
- `/healthz` endpoint on every service
- Timeout + deadline on all network calls
- No global mutable state
- Functions ≤ 30 lines, single responsibility

---

## 5. Environment Variables

```bash
# Services
API_GATEWAY_PORT=8080
USER_SERVICE_PORT=50051
ORDER_SERVICE_PORT=50052
MARKET_SERVICE_WS_PORT=8081

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=trading

# Kafka
KAFKA_BROKER=localhost:9092
KAFKA_ORDERS_TOPIC=orders
KAFKA_TRADES_TOPIC=trades

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Auth
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h

# Go
GO_ENV=development
CGO_ENABLED=0
```

---

## 6. Coding Patterns & Conventions

### Commit convention

```
feat: add new feature
fix: handle edge case
chore: update dependencies
test: add test coverage
docs: update documentation
```

### Go rules

- Context propagation on all calls; never pass `context.Background()` to handlers
- Explicit error handling — always check and propagate errors
- Use interfaces for all external dependencies (database, Kafka, gRPC clients)
- Structured logging via `zap.Logger.Sugar()` or `zap.L()` for production
- All public API functions must have docs (`// FunctionName does X`)
- Return typed errors; use `fmt.Errorf("context: %w", err)` for wrapping
- `sync.Mutex` for local critical sections only; never hold lock across await points
- gRPC calls use `ctx` with timeout derived from request context

### Error handling pattern

```go
// Domain errors wrapped with sentinel or typed errors
var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrOrderNotFound = errors.New("order not found")

// In handlers: log at call site, return appropriate gRPC/HTTP status
if err != nil {
    logger.With(zap.Error(err)).Error("PlaceOrder failed")
    return nil, status.Error(codes.InvalidArgument, err.Error())
}
```

### Kafka producer pattern

```go
// Publish with key for ordering guarantees per user
// Retry with exponential backoff (max 3 attempts)
// RequiredAcks: kafka.RequireAll
```

### Testing pattern

```go
func TestOrderBook_Match(t *testing.T) {
    ob := NewOrderBook("BTC/USD")
    // setup orders
    trades := ob.Match(incomingOrder)
    // assert trades, order states
}
```

---

## 7. Testing & Quality

**Essential Commands:**

- `go test ./...` — Run all tests
- `go vet ./... && golangci-lint run` — Static analysis
- `make proto` — Generate gRPC code from `.proto` files
- `make build` — Build all services
- `make docker-up` — Start infrastructure (Postgres, Kafka, Redis)
- `make migrate` — Run SQL migrations

---

## 8. Security & Safety

Full checklist in [Architecture (Security)](docs/architecture.md).

- JWT validation on every protected route via middleware
- `SELECT FOR UPDATE` for balance deduction to prevent race conditions
- Idempotency keys prevent duplicate order placement
- Rate limiting per IP via `golang.org/x/time/rate`
- No `synchronize: true` — always use migrations for schema changes
- Never commit `.env` files

---

## 9. Constraints & Rules (non-negotiable)

1. **Never** push directly to `main` — always feature branch + PR
2. **Never** commit `.env` or `.env.local` files
3. **Never** use `synchronize: true` in any ORM/data layer (production data loss risk)
4. **Never** delete migration files — always roll forward
5. **Always** run `make proto` after any `.proto` file change
6. **Always** create a migration after changing any database schema
7. **Always** use `/checkpoint` after completing a feature or ending a work session
8. **Always** update `docs/project-plan.md` via `/checkpoint` to track progress
9. Critical logic changes require a second pair of eyes (or explicit test coverage) before merge
10. Matching engine: each symbol runs on its own goroutine — never hold a mutex across await points

---

## 10. Available Slash Commands

| Command | When to use |
|---------|-------------|
| `/new-feature [name]` | Start any new feature (plans before coding) |
| `/commit` | Create a well-formatted git commit |
| `/pr` | Create a GitHub Pull Request |
| `/checkpoint` | Unified sync: update changelog + status + plan progress |
| `/generate-plan` | Create a detailed implementation plan |

---

## 11. Connected MCPs

| MCP | Purpose |
|-----|---------|
| `github` | Create issues, PRs, search code |
| `filesystem` | File read/write operations |

---

## 12. Architectural Decisions

| Decision | Reason |
|----------|--------|
| Kafka over direct gRPC for order matching | Decouples order service from matching engine; enables parallel consumers and replay |
| Heap-based order book per symbol | O(log n) insert/remove; natural price-time priority |
| Per-symbol goroutine in matching engine | No cross-symbol lock contention; linear scaling with symbols |
| WebSocket hub per market service | Broadcast trade events to subscribed clients with ping/pong keepalive |
| Redis for rate limiting | Shared state across API gateway instances; atomic token bucket |
| `segmentio/kafka-go` over Sarama | Lighter weight, Go-native, simpler writer API |

---

## 13. Known Issues

| Issue | Workaround |
|-------|------------|
| WebSocket reconnection on network partition | Clients should implement exponential backoff reconnect |
| Kafka consumer offset commit on handler error | Dead-letter queue or retry configurable per topic |
