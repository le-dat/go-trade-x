# Step 05 — Order Service: Place/Cancel Order via gRPC + Kafka

## Goal

Implement the gRPC Order Service that validates balance, persists orders, and publishes to Kafka.

> **Prerequisite**: [Step 04 — Kafka Integration](./04-kafka.md)

---

## Proto Contract

```protobuf
service OrderService {
  rpc PlaceOrder(PlaceOrderRequest) returns (PlaceOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse);
}

message PlaceOrderRequest {
  string user_id = 1;
  string symbol = 2;      // "BTC/USD"
  string side = 3;        // "BUY" | "SELL"
  string type = 4;       // "LIMIT" | "MARKET"
  string price = 5;      // decimal string
  string quantity = 6;   // decimal string
  string idempotency_key = 7;
}
```

---

## Database Schema

`migrations/002_create_orders.up.sql`:

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

CREATE TABLE IF NOT EXISTS outbox (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  topic TEXT NOT NULL,
  key TEXT NOT NULL,
  payload JSONB NOT NULL,
  status TEXT DEFAULT 'PENDING', -- PENDING, PROCESSED, FAILED
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

Order status values: `PENDING`, `PARTIAL`, `FILLED`, `CANCELLED`

---

## Step 5.1 — Create Proto

`proto/order.proto`:
```protobuf
service OrderService {
  rpc PlaceOrder(PlaceOrderRequest) returns (PlaceOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse);
}
```

```bash
make proto
```

---

## Step 5.2 — Service Logic

`internal/order/service.go`:

```go
func (s *orderService) PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error) {
    // 1. Check idempotency — return existing if duplicate
    existing, err := s.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
    if err == nil {
        return &PlaceOrderResponse{OrderId: existing.ID, Status: existing.Status}, nil
    }

    // 2. Deduct balance via User Service (gRPC)
    _, err = s.userClient.DeductBalance(ctx, req.UserId, "USD", req.Quantity)
    if err != nil {
        return nil, fmt.Errorf("deduct balance: %w", err)
    }

    // 3. Atomic DB Transaction: Order + Outbox
    err = s.db.WithTransaction(ctx, func(tx Transaction) error {
        order := &Order{...}
        if err := s.repo.CreateWithTx(ctx, tx, order); err != nil {
            return err
        }

        outboxMsg := &OutboxMessage{
            Topic:   "orders",
            Key:     order.UserID,
            Payload: order.ToJSON(),
        }
        return s.outboxRepo.CreateWithTx(ctx, tx, outboxMsg)
    })

    if err != nil {
        // Rollback balance if DB transaction fails
        s.userClient.CreditBalance(ctx, req.UserId, "USD", req.Quantity)
        return nil, fmt.Errorf("transaction failed: %w", err)
    }

    return &PlaceOrderResponse{OrderId: order.ID, Status: order.Status}, nil
}
```

---

## Step 5.3 — Outbox Relay (The "Worker")

`internal/order/relay.go`:

```go
func (r *Relay) Start(ctx context.Context) {
    ticker := time.NewTicker(100 * time.Millisecond)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            messages := r.repo.GetPending(ctx)
            for _, msg := range messages {
                if err := r.kafka.Publish(ctx, msg.Topic, msg.Key, msg.Payload); err == nil {
                    r.repo.MarkProcessed(ctx, msg.ID)
                }
            }
        }
    }
}
```

---

## Step 5.4 — Start & Test

```bash
make docker-up
make migrate

make run-order

# Test via API Gateway
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -d '{"email":"test@test.com","password":"secret"}' | jq -r '.token')

curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"symbol":"BTC/USD","side":"BUY","type":"LIMIT","price":"42000","quantity":"0.01"}'
```

---

## Verification Checklist

- [ ] Order in PostgreSQL with status PENDING
- [ ] Message in Kafka `[orders]` topic
- [ ] Duplicate `idempotency_key` returns existing order
- [ ] Insufficient balance returns error

> ➡️ Next: [Step 06 — Matching Engine](./06-matching-engine.md)