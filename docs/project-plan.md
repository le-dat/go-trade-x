# Project Plan — GoTradeX

> Generated: 2026-03-31
> Target: Milestone 1 — Core Platform (Phases 3–7)
> Last Updated: 2026-04-03

---

## 🛠 Automation Recommendations

All previously recommended automations have been implemented:

- [x] **Command: `go-build.md`** ✅ — One-click build/lint/test for all 5 services
- [x] **Command: `proto-dev.md`** ✅ — Proto scaffolding and code generation
- [x] **Command: `dev-setup.md`** ✅ — Docker infra startup + health checks
- [x] **Command: `kafka-test.md`** ✅ — Kafka round-trip test
- [x] **Agent: `matching-engine-agent.md`** ✅ — Focused matching engine implementation
- [x] **Agent: `go-review-agent.md`** ✅ — Go code review (concurrency, Kafka, gRPC patterns)

**Completed:**
- [x] **Agent: `user-service-agent.md`** ✅ — Reason: User service is Phase 4 with DB schema + bcrypt + JWT + gRPC; a focused agent ensures all parts wired correctly end-to-end.

---

## Gap Analysis

| Phase | Complexity | Risk | Gap |
|-------|-----------|------|-----|
| Phase 3 (API Gateway) | Medium | Low | gRPC client is mock; needs real gRPC |
| Phase 4 (User Service) | Medium | Medium | Entire gRPC service missing; no DB schema |
| Phase 5 (Order Service) | Medium | Medium | No Kafka producer; no DB schema |
| Phase 6 (Kafka pkg) | High | High | `pkg/kafka` doesn't exist yet |
| Phase 7 (Matching Engine) | Very High | Critical | Heap algorithm + concurrency; most complex part |
| Phase 8 (Market Service) | Medium | Medium | WebSocket hub + Kafka consumer missing |
| Phase 9 (Docker) | Low | Low | docker-compose exists but missing redis; Dockerfile per-service |

**Recommendation:** Implement Phase 6 (Kafka pkg) **before** Phase 4 and 5, since both services depend on it. Implement Phase 7 last since it depends on everything else.

---

## Phase 3 — API Gateway Completion ✅ (~1 day)

> **Status: Complete** — Phase 3 API Gateway finished (commit 11a16bc).

### Step 1: Install gRPC toolchain ✅

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/cmd/protoc-gen-grpc-gateway@latest
go install github.com/envoyproxy/protoc-gen-validate@latest
```

- Done when: `which protoc-gen-go && which protoc-gen-go-grpc` succeed

### Step 2: Replace mock gRPC clients with real clients

- Edit: `cmd/api-gateway/clients/grpc.go`
- Replace mock factory with real `grpc.DialContext` to `:50051` (User) and `:50052` (Order)
- Done when: `go build ./cmd/api-gateway` succeeds with real gRPC stubs

### Step 3: Verify gateway starts + middleware

```bash
make run-api
curl http://localhost:8080/healthz
```

- Done when: `/healthz` returns `{"status": "ok"}`

---

## Phase 4 — User Service ✅ (Steps 4-6 complete, Step 7 pending)

### Step 4: Create `proto/user.proto` ✅

```protobuf
syntax = "proto3";
package user;
option go_package = "github.com/verno/gotradex/pkg/proto/user";

```protobuf
syntax = "proto3";
package user;
option go_package = "github.com/verno/gotradex/pkg/proto/user";

service UserService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc GetBalance(GetBalanceRequest) returns (GetBalanceResponse);
  rpc DeductBalance(DeductBalanceRequest) returns (DeductBalanceResponse);
  rpc CreditBalance(CreditBalanceRequest) returns (CreditBalanceResponse);
}
```

- Done when: `protoc $(PROTO_DIR)/user.proto` generates `pkg/proto/user/*.pb.go` ✅

### Step 5: Create SQL migrations for users/balances ✅

```sql
-- migrations/001_create_users.up.sql
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS balances (
  user_id UUID REFERENCES users(id),
  asset TEXT NOT NULL,
  available NUMERIC(20,8) NOT NULL DEFAULT 0,
  locked NUMERIC(20,8) NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id, asset)
);

-- migrations/001_create_users.down.sql
DROP TABLE IF EXISTS balances;
DROP TABLE IF EXISTS users;
```

- Done when: `psql $DATABASE_URL -f migrations/001_create_users.up.sql` succeeds ✅

### Step 6: Implement `cmd/user-service/main.go` ✅

- Repository: `internal/user/repository.go` — raw `pgx` or `sqlc`-generated
- Service: `internal/user/service.go` — bcrypt + JWT (HS256, 24h)
- Handler: gRPC server on `:50051`
- Done when: `grpcurl localhost:50051 list` shows `user.UserService`

### Step 7: Verify with grpcurl (pending)

```bash
grpcurl -plaintext -d '{"email":"test@test.com","password":"secret"}' localhost:50051 user.UserService/Register
```

- Done when: Register returns user ID; Login returns JWT

---

## Phase 5 — Kafka Package (Prerequisite) (~2 days)

> **Run this before Phase 4/5 service wiring.** Both Order Service and Matching Engine depend on it.

### Step 8: Implement `pkg/kafka/producer.go`

```go
type Producer interface {
    Publish(ctx context.Context, topic string, key string, value []byte) error
    Close() error
}
```

- Use `segmentio/kafka-go` writer with `RequiredAcks: RequireAll`
- Retry with exponential backoff (max 3 attempts)
- Key = `userID` for per-user ordering
- Done when: Unit test publishes a message and consumer receives it

### Step 9: Implement `pkg/kafka/consumer.go`

```go
type Handler func(ctx context.Context, msg kafka.Message) error
type Consumer interface {
    Run(ctx context.Context, topic string, handler Handler) error
}
```

- Commit offset only after successful `Handler` return
- Graceful stop via context cancellation
- Done when: Consumer processes a message and commits offset

### Step 10: Write round-trip integration test

```bash
# Start kafka via docker-compose
make docker-up
# Run round-trip test
go test ./pkg/kafka/... -v
```

- Done when: Test passes — message published to `orders`, consumer receives it

---

## Phase 6 — Order Service (~3 days)

### Step 11: Create `proto/order.proto`

```protobuf
service OrderService {
  rpc PlaceOrder(PlaceOrderRequest) returns (PlaceOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse);
}
```

### Step 12: Create `migrations/002_create_orders.up.sql`

```sql
CREATE TABLE IF NOT EXISTS orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  idempotency_key TEXT UNIQUE,
  user_id UUID NOT NULL,
  symbol TEXT NOT NULL,
  side TEXT NOT NULL,
  type TEXT NOT NULL,
  price NUMERIC(20,8),
  quantity NUMERIC(20,8) NOT NULL,
  filled_qty NUMERIC(20,8) DEFAULT 0,
  status TEXT DEFAULT 'PENDING',
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Step 13: Implement `cmd/order-service/main.go`

- Repository: `internal/order/repository.go`
- Service: `internal/order/service.go` — idempotency check, calls UserService.DeductBalance
- Handler: gRPC server on `:50052`
- Kafka producer: use `pkg/kafka.Producer`
- Done when: `go build ./cmd/order-service` succeeds

### Step 14: Verify full order placement

```bash
# Start all infra
make docker-up
# Place order via gateway
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login ... | jq -r .token)
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"symbol":"BTC/USD","side":"BUY","type":"LIMIT","price":"42000","quantity":"0.01"}'
```

- Done when: Order in DB, message in Kafka `[orders]` topic

---

## Phase 7 — Matching Engine ⚡ (~5 days)

> Highest risk. Recommended: create `matching-engine-agent.md` to focus on this.

### Step 15: Implement `internal/matching/heap.go`

```go
type BidHeap []*Order   // max-heap by price
type AskHeap []*Order   // min-heap by price
```

- `heap.Init`, `heap.Push`, `heap.Pop` implementations
- Price-time priority: if prices equal, earlier `created_at` wins
- Done when: `go test ./internal/matching/...` passes

### Step 16: Implement `internal/matching/orderbook.go`

```go
type OrderBook struct {
    symbol string
    bids   *BidHeap
    asks   *AskHeap
    mu     sync.Mutex
}
func (ob *OrderBook) Match(incoming Order) []Trade
```

- BUY order: while `incoming.Price >= asks.Peek().Price`, match against ask min-heap
- SELL order: while `incoming.Price <= bids.Peek().Price`, match against bid max-heap
- Partial fill handling
- Done when: `go test ./internal/matching/...` — all match cases pass

### Step 17: Implement `internal/matching/engine.go`

- Symbol router: `map[symbol]chan Order`
- Per-symbol goroutine (no cross-symbol locking)
- Channel buffer: 1000 orders per symbol
- Consume from Kafka `[orders]`, publish to `[trades]`
- Done when: `go build ./cmd/matching-engine` succeeds

### Step 18: Benchmark

```bash
go test -bench=BenchmarkMatchingEngine -benchmem -count=5 ./internal/matching/...
```

- Done when: `> 10,000 orders/sec`, `< 1ms p99` matching latency

---

## Phase 8 — Market Service (~2 days)

### Step 19: Implement `internal/market/hub.go`

```go
type Hub struct {
    clients    map[string]map[*Client]bool  // symbol → clients
    register   chan *Client
    unregister chan *Client
    broadcast  chan TradeEvent
    mu         sync.RWMutex
}
```

### Step 20: Implement `internal/market/client.go`

- Gorilla WebSocket upgrade on `ws://:8081/ws?symbol=BTC/USD`
- Ping/pong keepalive (30s interval)
- Client write timeout: 10s, max message: 512 bytes
- Silent unregister on write error

### Step 21: Wire Kafka consumer → hub broadcast

- Consume from `[trades]` topic
- On each trade: `hub.broadcast <- tradeEvent`
- Done when: WebSocket client receives trade < 100ms after Kafka message

---

## Phase 9 — Dockerization Completion (~1 day)

### Step 22: Create `docker/Dockerfile` per-service

```dockerfile
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

### Step 23: Add Redis to docker-compose.yml

```yaml
redis:
  image: redis:7-alpine
  ports:
    - "6379:6379"
```

### Step 24: Verify full docker-compose up

```bash
make docker-up
docker compose ps  # all services healthy
```

- Done when: All 5 services + postgres + kafka + redis running

---

## Phase 10 — Testing & Benchmarking (~2 days)

### Step 25: Write unit tests

```bash
# OrderBook matching tests
go test ./internal/matching/... -v

# UserService tests
go test ./internal/user/... -v

# OrderService tests
go test ./internal/order/... -v

# JWT tests
go test ./pkg/auth/... -v
```

### Step 26: Integration tests (testcontainers-go)

- Order placement → Kafka → Matching → Trade → WebSocket
- Done when: Full flow test passes in CI

---

## Phase 11 — Final Deliverables (~1 day)

### Step 27: Verify end-to-end flow

```bash
make docker-up
# Register + Login
TOKEN=$(curl ... | jq -r .token)
# Place order
curl ... -H "Authorization: Bearer $TOKEN" ...
# WebSocket receives trade
```

### Step 28: Update README with verified examples

---

## Dependencies Summary

```
Step 1  → (no deps)
Step 2  → Step 1
Step 3  → Step 2
Step 4  → Step 1
Step 5  → (no deps)
Step 6  → Step 4, Step 5
Step 7  → Step 6
Step 8  → (no deps)
Step 9  → Step 8
Step 10 → Step 9
Step 11 → Step 1
Step 12 → Step 5
Step 13 → Step 9, Step 11, Step 12
Step 14 → Step 13
Step 15 → (no deps)
Step 16 → Step 15
Step 17 → Step 10, Step 15, Step 16
Step 18 → Step 17
Step 19 → (no deps)
Step 20 → Step 19
Step 21 → Step 10, Step 19
Step 22 → (no deps)
Step 23 → Step 22
Step 24 → Step 23
Step 25 → Step 7, Step 14, Step 18, Step 21
Step 26 → Step 25
Step 27 → Step 14, Step 21, Step 24
Step 28 → Step 27
```

---

## Critical Path

```
Step 1 → Step 4 → Step 5 → Step 6 → Step 7                          (Phase 4)
Step 1 → Step 8 → Step 9 → Step 10                                  (Phase 6)
Step 10 + Step 6 + Step 11 → Step 12 → Step 13 → Step 14            (Phase 5)
Step 15 → Step 16 → Step 17 → Step 18                                (Phase 7)
Step 19 → Step 20 → Step 21                                          (Phase 8)
```

**Shortest path to end-to-end:** Steps 1→4→5→6→7 (Phase 4) → Steps 8→9→10 (Phase 6) → Steps 11→12→13→14 (Phase 5) → Steps 15→16→17→18 (Phase 7) → Steps 19→20→21 (Phase 8) → Steps 22→23→24 (Phase 9)
