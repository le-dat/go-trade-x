# Specification Document — GoTradeX

Real-time Trading & Order Matching Platform

## Project Overview

GoTradeX is a high-performance trading platform implementing a Central Limit Order Book (CLOB) for real-time order matching. The system handles order placement, matching, and trade execution with sub-millisecond latency.

## Technology Stack

- **Language**: Go 1.22+
- **REST API**: Gin Framework
- **Service Communication**: gRPC
- **Message Queue**: Apache Kafka
- **Database**: PostgreSQL
- **Cache**: Redis
- **WebSocket**: gorilla/websocket
- **Container**: Docker + Docker Compose

---

## Phase Milestones

### Phase 1 — System Design ✅
- [x] Architecture contract established
- [x] Service responsibilities defined
- [x] Communication map confirmed
- [x] Kafka topics configured

### Phase 2 — Project Setup ✅
- [x] Monorepo structure created
- [x] Go workspace configured
- [x] Makefile with build, test, lint targets
- [x] Shared packages (auth, config, database, kafka, logger)

### Phase 3 — API Gateway ⚠️ (In Progress)
- [x] Gin framework with Swagger UI
- [x] JWT authentication middleware
- [x] Rate limiting middleware
- [x] Recovery middleware
- [x] Request logging middleware
- [ ] gRPC client integration
- [ ] Order endpoints connected

**Acceptance Criteria:**
- Gateway starts on port 8080
- `/healthz` returns 200
- Auth middleware validates JWT tokens
- Rate limiter enforces 100 req/min per IP

### Phase 4 — User Service
- [ ] gRPC server on port 50051
- [ ] User registration with bcrypt password hashing
- [ ] User login with JWT issuance (24h expiry)
- [ ] Balance management (GetBalance, DeductBalance, CreditBalance)
- [ ] PostgreSQL integration

**Acceptance Criteria:**
- `grpcurl localhost:50051 list` shows UserService
- Register returns user ID
- Login returns valid JWT token
- DeductBalance uses SELECT FOR UPDATE

### Phase 5 — Order Service
- [ ] gRPC server on port 50052
- [ ] PlaceOrder with idempotency key
- [ ] GetOrder to retrieve order status
- [ ] CancelOrder to cancel pending orders
- [ ] Kafka producer for orders topic

**Acceptance Criteria:**
- Duplicate idempotency_key returns existing order
- Insufficient balance rejects order
- Order persisted to PostgreSQL with PENDING status
- Kafka message published to orders topic

### Phase 6 — Kafka Integration
- [ ] Producer with retry (max 3 attempts)
- [ ] Consumer with offset commit after handler
- [ ] Graceful shutdown on context cancellation

**Acceptance Criteria:**
- Round-trip message test passes
- Producer retries on failure
- Consumer commits only after successful handler

### Phase 7 — Matching Engine ⚡ (Critical)
- [ ] In-memory order book per symbol
- [ ] Bid heap (max-heap for buy orders)
- [ ] Ask heap (min-heap for sell orders)
- [ ] Price-time priority matching
- [ ] Partial fill handling

**Acceptance Criteria:**
- 10,000 orders/sec throughput
- < 1ms p99 matching latency
- BUY order matches against lowest ASK when price >= ask

### Phase 8 — Market Service
- [ ] WebSocket hub per symbol
- [ ] Kafka consumer for trades topic
- [ ] Real-time broadcast to subscribed clients
- [ ] Ping/pong keepalive (30s interval)

**Acceptance Criteria:**
- Client receives trade < 100ms after match
- Client write timeout 10s
- Silent unregister on write error

### Phase 9 — Dockerization
- [ ] Multi-stage Dockerfiles per service
- [ ] docker-compose.yml with all services
- [ ] Health checks configured

**Acceptance Criteria:**
- `make docker-up` starts all services
- All services report healthy
- End-to-end order flow works

### Phase 10 — Testing & Benchmarking
- [ ] Unit tests for OrderBook matching
- [ ] Unit tests for UserService
- [ ] Unit tests for OrderService
- [ ] Integration tests with testcontainers
- [ ] Benchmark: > 50k matches/sec

**Acceptance Criteria:**
- All unit tests green
- Benchmark documented
- Integration tests pass

### Phase 11 — Final Deliverables
- [ ] README with run instructions
- [ ] Example curl commands verified
- [ ] WebSocket example verified

---

## Scope

### In Scope
- REST API for client interaction
- gRPC for internal service communication
- Kafka for async order/trade processing
- WebSocket for real-time market data
- PostgreSQL for persistence
- Docker for containerization

### Out of Scope
- User interface (frontend)
- Admin dashboard
- Multi-currency support (single asset initially)
- High-frequency trading optimizations beyond Phase 7 targets

---

## Acceptance Criteria Summary

| Phase | Metric | Target |
|-------|--------|--------|
| Phase 3 | Gateway starts | Port 8080, /healthz → 200 |
| Phase 4 | JWT expiry | 24 hours |
| Phase 5 | Idempotency | Duplicate key returns existing |
| Phase 7 | Matching throughput | > 10k orders/sec |
| Phase 7 | Matching latency | < 1ms p99 |
| Phase 8 | WebSocket latency | < 100ms |
| Phase 10 | Benchmark | > 50k matches/sec |

---

## Decisions Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-03-23 | Single Go module (monorepo) | Simplifies dependency management |
| 2026-03-23 | Kafka for async messaging | Decouples services, enables scalability |
| 2026-03-23 | Per-symbol goroutines in matching | Avoids cross-symbol lock contention |
| 2026-03-23 | JWT HS256 | Simpler than RSA, suitable for single-service auth |
