# Step 12 — Architecture Decisions: Key Design Choices

## Goal

Document key architectural decisions and rationale.

---

## Decision 1: Monorepo with Go Workspace

**Decision**: Single Go module (monorepo) with `go.work`

**Rationale**: Simplifies dependency management across 5 services.

---

## Decision 2: Kafka over Direct gRPC for Order Matching

**Decision**: Kafka for async order/trade messaging

**Rationale**:
- Decouples Order Service from Matching Engine
- Enables parallel consumers
- Enables replay for debugging/reprocessing
- Handles backpressure naturally

---

## Decision 3: Per-Symbol Goroutines in Matching Engine

**Decision**: Each symbol runs on its own goroutine in the matching engine

**Rationale**: Avoids cross-symbol lock contention; linear scaling with symbols.

---

## Decision 4: Heap-Based Order Book

**Decision**: Heap data structure for bid/ask order books

**Rationale**: O(log n) insert/remove; natural price-time priority (FIFO at same price).

---

## Decision 5: JWT HS256

**Decision**: JWT with HS256 signing

**Rationale**: Simpler than RSA; suitable for single-service auth where gateway and user service share the secret.

---

## Decision 6: segmentio/kafka-go

**Decision**: Use `segmentio/kafka-go` over `Sarama`

**Rationale**: Lighter weight, Go-native, simpler writer API with better defaults.

---

## Decision 10: OpenTelemetry (OTel) for Observability

**Decision**: Integrate OTel for distributed tracing across services.

**Rationale**: Essential for debugging distributed flows (Gateway -> Order -> Kafka -> ME -> Market).

---

## Decision 7: SELECT FOR UPDATE for Balance

**Decision**: `SELECT ... FOR UPDATE` inside transaction for balance deduction

**Rationale**: Prevents race conditions on concurrent deductions (row-level locking).

---

## Decision 8: Transactional Outbox Pattern

**Decision**: Use an `outbox` table in PostgreSQL for Order -> Kafka messaging.

**Rationale**:
- Ensures **Atomic Consistency**: Order is only created if the intent to publish to Kafka is also recorded.
- Prevents "lost orders" if the service crashes after DB commit but before Kafka publish.
- Decouples DB transaction from Kafka availability.

---

## Decision 9: Deterministic State Replay for Matching Engine

**Decision**: Replay `orders` topic from a known snapshot or from the beginning on startup.

**Rationale**:
- Matching Engine is in-memory for performance (<1ms latency).
- Replay ensures the orderbook is correctly rebuilt after a restart or crash.

---

## Rejected Alternatives

| Alternative | Reason for Rejection |
|------------|----------------------|
| Redis Streams instead of Kafka | Kafka has better replay capabilities for matching |
| Table-level locking for balance | Row-level locking has better concurrency |
| RS256 for JWT | HS256 sufficient for single-service, simpler to implement |
| Direct gRPC from API Gateway to services | Kafka needed for order matching decoupling |

---

## Open Questions

| Question | Status |
|----------|--------|
| Multi-currency support | Deferred — Phase 11+ |
| Admin dashboard | Out of scope |
| HFT optimizations | Deferred — Beyond Phase 7 targets |