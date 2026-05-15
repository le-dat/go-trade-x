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

    // 2. Deduct balance (lock funds)
    _, err = s.userClient.DeductBalance(ctx, req.UserId, "USD", req.Quantity)
    if err != nil {
        return nil, fmt.Errorf("deduct balance: %w", err)
    }

    // 3. Persist order
    order := &Order{...}
    if err := s.repo.Create(ctx, order); err != nil {
        s.userClient.CreditBalance(ctx, req.UserId, "USD", req.Quantity)
        return nil, fmt.Errorf("create order: %w", err)
    }

    // 4. Publish to Kafka [orders]
    if err := s.kafka.Publish(ctx, "orders", req.UserId, order.ToJSON()); err != nil {
        return nil, fmt.Errorf("publish order: %w", err)
    }

    return &PlaceOrderResponse{OrderId: order.ID, Status: order.Status}, nil
}
```

---

## Step 5.3 — Start & Test

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