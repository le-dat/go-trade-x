# Real-time Trading & Order Matching Platform
## Go Microservices — Engineering Roadmap

> **Stack:** Go · Gin · gRPC · Kafka · PostgreSQL · Redis · WebSocket · Docker  
> **Pattern:** Clean Architecture (handler → service → repository)  
> **Execution:** Phase-by-phase, confirm before advancing

---

## Global Engineering Standards

Applied to **every** service without exception:

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

## Phase 1 — System Design

**Goal:** Establish architecture contract before writing any code.

### Services & Responsibilities

| Service | Protocol | Role |
|---|---|---|
| API Gateway | REST (Gin) | Auth, routing, rate limiting, JWT validation |
| User Service | gRPC | Register, login, JWT issuance, balance management |
| Order Service | gRPC + Kafka | Validate balance, persist order, publish to Kafka |
| Matching Engine | Kafka consumer | In-memory order book, matching logic, emit trades |
| Market Service | Kafka consumer + WS | Consume trades, broadcast real-time to WebSocket clients |

### Communication Map

```
Client
  │
  ▼
API Gateway (Gin :8080)
  ├─── gRPC ──► User Service (:50051)    ──► PostgreSQL
  ├─── gRPC ──► Order Service (:50052)   ──► PostgreSQL
  │                │
  │            Kafka [orders topic]
  │                │
  │         Matching Engine (consumer)
  │                │
  │            Kafka [trades topic]
  │                │
  │          Market Service (consumer)
  │                │
  └── WebSocket ◄──┘  (ws://:8081)
```

### Order Lifecycle

```
1. Client POST /orders
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

### Kafka Topics

| Topic | Producer | Consumer(s) |
|---|---|---|
| `orders` | Order Service | Matching Engine |
| `trades` | Matching Engine | Market Service, Order Service, User Service |

**Deliverable:** Architecture diagram confirmed, data flow agreed.

---

## Phase 2 — Project Setup

**Input:** Phase 1 architecture  
**Output:** Runnable monorepo skeleton

### Repository Structure

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

### Setup Tasks

1. `go work init` + `go work use ./cmd/...`
2. `go.mod` per service or single shared module (decide: single module recommended)
3. `Makefile` targets: `build`, `test`, `proto`, `migrate`, `docker-up`, `lint`
4. Config via `pkg/config`: read from env, support `.env` file via `godotenv`
5. Logger init in `pkg/logger`: zap production config

**Deliverable:** `make build` succeeds for all 5 services.

---

## Phase 3 — API Gateway

**Input:** User Service + Order Service gRPC endpoints  
**Output:** REST API with auth, forwarding, middleware

### Endpoints

```
POST /api/v1/auth/register  → UserService.Register (gRPC)
POST /api/v1/auth/login     → UserService.Login (gRPC)
POST /api/v1/orders         → OrderService.PlaceOrder (gRPC) [JWT required]
GET  /api/v1/orders/:id     → OrderService.GetOrder (gRPC) [JWT required]
GET  /healthz
```

### Middleware Stack (applied in order)

1. `Recovery` — catch panics, return 500
2. `Logger` — zap structured request logging (method, path, status, latency)
3. `RateLimiter` — token bucket per IP (use `golang.org/x/time/rate`)
4. `Auth` — validate JWT, inject `userID` into context (protected routes only)

### Key Implementation Notes

- gRPC clients initialized once at startup, passed via dependency injection
- All gRPC calls use `ctx` with timeout derived from request context
- Return standardized JSON error envelope: `{"error": "message", "code": "ERROR_CODE"}`

**Deliverable:** Gateway starts, forwards to mock gRPC, middleware verified.

---

## Phase 4 — User Service

**Input:** PostgreSQL  
**Output:** gRPC service for auth + balance

### Proto Contract

```protobuf
service UserService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc GetBalance(GetBalanceRequest) returns (GetBalanceResponse);
  rpc DeductBalance(DeductBalanceRequest) returns (DeductBalanceResponse);
  rpc CreditBalance(CreditBalanceRequest) returns (CreditBalanceResponse);
}
```

### Database Schema

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE balances (
  user_id UUID REFERENCES users(id),
  asset TEXT NOT NULL,           -- "USD", "BTC", etc.
  available NUMERIC(20,8) NOT NULL DEFAULT 0,
  locked NUMERIC(20,8) NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id, asset)
);
```

### Implementation Layers

- **Repository:** Raw `database/sql` or `sqlc`-generated; `DeductBalance` uses `SELECT FOR UPDATE`
- **Service:** Hash passwords with `bcrypt`, issue JWT (HS256, 24h expiry)
- **Handler:** gRPC server, maps domain errors to gRPC status codes

**Deliverable:** `grpcurl` tests for register/login/balance pass.

---

## Phase 5 — Order Service

**Input:** User Service (gRPC), Kafka producer, PostgreSQL  
**Output:** gRPC service that validates, persists, and enqueues orders

### Proto Contract

```protobuf
service OrderService {
  rpc PlaceOrder(PlaceOrderRequest) returns (PlaceOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse);
}

message PlaceOrderRequest {
  string user_id = 1;
  string symbol = 2;    // "BTC/USD"
  string side = 3;      // "BUY" | "SELL"
  string type = 4;      // "LIMIT" | "MARKET"
  string price = 5;     // decimal string
  string quantity = 6;  // decimal string
  string idempotency_key = 7;
}
```

### Database Schema

```sql
CREATE TABLE orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  idempotency_key TEXT UNIQUE,
  user_id UUID NOT NULL,
  symbol TEXT NOT NULL,
  side TEXT NOT NULL,
  type TEXT NOT NULL,
  price NUMERIC(20,8),
  quantity NUMERIC(20,8) NOT NULL,
  filled_qty NUMERIC(20,8) DEFAULT 0,
  status TEXT DEFAULT 'PENDING',  -- PENDING, PARTIAL, FILLED, CANCELLED
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Placement Flow

```
1. Check idempotency_key → return existing if duplicate
2. Call UserService.DeductBalance (lock funds)
3. Persist order with status PENDING
4. Publish to Kafka [orders] topic
5. Return order ID to caller
```

**Deliverable:** Order placed → visible in DB → message in Kafka.

---

## Phase 6 — Kafka Integration

**Input:** Broker config  
**Output:** Reusable producer/consumer in `pkg/kafka`

### Producer

```go
// pkg/kafka/producer.go
type Producer interface {
    Publish(ctx context.Context, topic string, key string, value []byte) error
    Close() error
}
```

- Use `segmentio/kafka-go` writer with `RequiredAcks: kafka.RequireAll`
- Retry with exponential backoff (max 3 attempts)
- Key = `userID` for ordering guarantees per user

### Consumer

```go
// pkg/kafka/consumer.go
type Handler func(ctx context.Context, msg kafka.Message) error

type Consumer interface {
    Run(ctx context.Context, handler Handler) error
}
```

- Commit offset only after successful `Handler` return
- On handler error: log + dead-letter or retry (configurable)
- Graceful stop via context cancellation

### Topics Configuration

| Topic | Partitions | Retention |
|---|---|---|
| `orders` | 4 | 7 days |
| `trades` | 4 | 30 days |

**Deliverable:** Producer/consumer integration test passes (round-trip message).

---

## Phase 7 — Matching Engine ⚡ (Critical)

**Input:** Kafka `[orders]` topic  
**Output:** Matched trades published to Kafka `[trades]`

### Order Book Design

```
OrderBook per symbol (e.g., "BTC/USD")
├── Bids: max-heap (highest buy price first)
└── Asks: min-heap (lowest sell price first)
```

```go
// internal/matching/orderbook.go
type OrderBook struct {
    symbol string
    bids   *BidHeap    // max-heap
    asks   *AskHeap    // min-heap
    mu     sync.Mutex
}

func (ob *OrderBook) Match(incoming Order) []Trade
```

### Matching Algorithm

```
LIMIT BUY order arrives:
  while asks.Len() > 0 && incoming.Price >= asks.Peek().Price:
    fill = min(incoming.RemainingQty, asks.Peek().RemainingQty)
    emit Trade{buyOrderID, sellOrderID, price=asks.Peek().Price, qty=fill}
    update both orders
    if ask fully filled → pop from heap
    if incoming fully filled → stop

Remaining incoming qty → insert into bids heap
```

### Concurrency Model

```
Kafka Consumer goroutine
       │
       ▼  (per-symbol channel, buffered)
Symbol Router (map[symbol]chan Order)
       │
       ▼  (one goroutine per symbol — no lock contention across symbols)
OrderBook.Match()
       │
       ▼
Trade Publisher goroutine (batched, async)
```

- Each symbol runs on its own goroutine — no cross-symbol locking
- `sync.Mutex` only within a single `OrderBook` (low contention)
- Channel buffer = 1000 orders per symbol

### Trade Event Schema (Kafka message)

```json
{
  "trade_id": "uuid",
  "symbol": "BTC/USD",
  "buy_order_id": "uuid",
  "sell_order_id": "uuid",
  "price": "42000.50",
  "quantity": "0.01",
  "timestamp": "2026-01-01T00:00:00Z"
}
```

**Deliverable:** 10k orders/sec benchmark, matching latency < 1ms p99.

---

## Phase 8 — Market Service

**Input:** Kafka `[trades]` topic  
**Output:** WebSocket broadcast to subscribed clients

### WebSocket Hub Design

```go
type Hub struct {
    clients    map[string]map[*Client]bool  // symbol → clients
    register   chan *Client
    unregister chan *Client
    broadcast  chan TradeEvent
    mu         sync.RWMutex
}
```

### Client Subscription Protocol

```
// Client connects:
ws://host:8081/ws?symbol=BTC/USD

// Server pushes on each trade:
{"type":"trade","symbol":"BTC/USD","price":"42000.50","qty":"0.01","ts":"..."}

// Ping/pong keepalive: 30s interval
```

### Flow

```
Kafka consumer → TradeEvent → Hub.broadcast channel
Hub goroutine → range broadcast → for each client in symbol set → client.send channel
Client goroutine → write to WebSocket conn
```

- Client write timeout: 10s
- Max message size: 512 bytes
- On write error: unregister client silently

**Deliverable:** WebSocket client receives trade events < 100ms after match.

---

## Phase 9 — Dockerization

**Input:** All services complete  
**Output:** Single `docker-compose up` starts entire platform

### Per-Service Dockerfile Pattern

```dockerfile
# Multi-stage build
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/service ./cmd/SERVICE_NAME

FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/service /service
ENTRYPOINT ["/service"]
```

### docker-compose.yml Services

```yaml
services:
  postgres:     image: postgres:16-alpine
  kafka:        image: confluentinc/cp-kafka:7.6.0
  zookeeper:    image: confluentinc/cp-zookeeper:7.6.0
  redis:        image: redis:7-alpine       # rate limiting, session cache
  api-gateway:  build: ./docker/api-gateway
  user-service: build: ./docker/user-service
  order-service: build: ./docker/order-service
  matching-engine: build: ./docker/matching-engine
  market-service: build: ./docker/market-service
```

**Deliverable:** `make docker-up` → all services healthy → end-to-end order flow works.

---

## Phase 10 — Testing & Benchmarking

**Input:** Complete system  
**Output:** Test suite + benchmark results

### Unit Tests (per service)

| Component | What to test |
|---|---|
| `OrderBook.Match()` | BUY/SELL match, partial fill, no match, price priority |
| `UserService` | Register duplicate, login wrong password, balance deduction |
| `OrderService` | Idempotency key, insufficient balance rejection |
| `JWT auth` | Valid token, expired token, tampered token |

### Integration Tests

- Order placement → Kafka message produced (testcontainers-go)
- Matching engine consumes → trade emitted → WebSocket delivers

### Benchmark

```go
// internal/matching/engine_bench_test.go
func BenchmarkMatchingEngine(b *testing.B) {
    // Pre-load 1000 sell orders into book
    // b.N iterations of BUY order placement
    // Report: ns/op, allocs/op
}
```

**Target:** > 50,000 matches/sec on single core, < 2µs per match.

---

## Phase 11 — Final Deliverables

### 1. How to Run

```bash
# Prerequisites: Docker, Go 1.22+, make

git clone <repo>
make proto          # generate gRPC code
make docker-up      # start infra (postgres, kafka, redis)
make migrate        # run SQL migrations
make run-all        # start all services
```

### 2. Example API Calls

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"trader@example.com","password":"secret123"}'

# Login → get JWT
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -d '{"email":"trader@example.com","password":"secret123"}' | jq -r .token)

# Place order
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "BTC/USD",
    "side": "BUY",
    "type": "LIMIT",
    "price": "42000.00",
    "quantity": "0.01",
    "idempotency_key": "order-001"
  }'
```

### 3. WebSocket Usage

```javascript
const ws = new WebSocket("ws://localhost:8081/ws?symbol=BTC/USD");

ws.onmessage = (event) => {
  const trade = JSON.parse(event.data);
  console.log(`Trade: ${trade.qty} BTC @ $${trade.price}`);
};
```

---

## Execution Checklist

```
[ ] Phase 1  — Architecture confirmed
[ ] Phase 2  — Monorepo builds cleanly
[ ] Phase 3  — Gateway routes + middleware tested
[ ] Phase 4  — User Service: register/login/balance via grpcurl
[ ] Phase 5  — Order placed → in DB → in Kafka
[ ] Phase 6  — Kafka producer/consumer round-trip test
[ ] Phase 7  — Matching engine: 10k orders benchmark
[ ] Phase 8  — WebSocket delivers trades < 100ms
[ ] Phase 9  — docker-compose up → full flow works
[ ] Phase 10 — All unit tests green, benchmark documented
[ ] Phase 11 — README complete, curl examples verified
```

---

## Key Dependencies

```
github.com/gin-gonic/gin           v1.10+   REST framework
google.golang.org/grpc             v1.64+   gRPC
github.com/segmentio/kafka-go      v0.4+    Kafka client
github.com/jackc/pgx/v5                     PostgreSQL driver
github.com/golang-jwt/jwt/v5                JWT
go.uber.org/zap                             Structured logging
golang.org/x/time/rate                      Rate limiting
github.com/gorilla/websocket                WebSocket
github.com/stretchr/testify                 Testing
github.com/testcontainers/testcontainers-go Integration tests
```