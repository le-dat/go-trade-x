# Step 06 — Matching Engine: Heap-Based Order Book + Price-Time Priority

## Goal

Implement the heap-based order matching engine with per-symbol goroutines.

> **Prerequisite**: [Step 05 — Order Service](./05-order-service.md)

---

## Order Book Design

```
OrderBook per symbol (e.g., "BTC/USD")
├── Bids: max-heap (highest buy price first)
└── Asks: min-heap (lowest sell price first)
```

## Matching Algorithm

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

---

## Step 6.1 — Heap Implementation

`internal/matching/heap.go`:

```go
type Order struct {
    ID        string
    Symbol    string
    Side      string  // "BUY" or "SELL"
    Price     decimal.Decimal
    Quantity  decimal.Decimal
    FilledQty decimal.Decimal
    CreatedAt time.Time
}

// BidHeap — max-heap by price
func (h BidHeap) Less(i, j int) bool {
    if h[i].Price.Equal(h[j].Price) {
        return h[i].CreatedAt.Before(h[j].CreatedAt)
    }
    return h[i].Price.GreaterThan(h[j].Price)
}

// AskHeap — min-heap by price
func (h AskHeap) Less(i, j int) bool {
    if h[i].Price.Equal(h[j].Price) {
        return h[i].CreatedAt.Before(h[j].CreatedAt)
    }
    return h[i].Price.LessThan(h[j].Price)
}
```

---

## Step 6.2 — Order Book

`internal/matching/orderbook.go`:

```go
type OrderBook struct {
    symbol string
    bids   *BidHeap
    asks   *AskHeap
    mu     sync.Mutex
}

func (ob *OrderBook) Match(incoming *Order) []Trade {
    ob.mu.Lock()
    defer ob.mu.Unlock()

    var trades []Trade
    remaining := incoming.RemainingQty()

    if incoming.Side == "BUY" {
        for remaining.GreaterThan(decimal.Zero) && ob.asks.Len() > 0 {
            bestAsk := ob.asks[0]
            if incoming.Price.LessThan(bestAsk.Price) {
                break
            }
            fillQty := min(remaining, bestAsk.RemainingQty())
            trades = append(trades, Trade{
                BuyOrder:  incoming.ID,
                SellOrder: bestAsk.ID,
                Price:     bestAsk.Price,
                Quantity:  fillQty,
            })
            bestAsk.FilledQty = bestAsk.FilledQty.Add(fillQty)
            remaining = remaining.Sub(fillQty)
            if bestAsk.RemainingQty().IsZero() {
                heap.Pop(ob.asks)
            }
        }
        if remaining.GreaterThan(decimal.Zero) {
            heap.Push(ob.bids, incoming)
        }
    }
    // ... similar for SELL against bids
    return trades
}
```

---

## Step 6.3 — Engine

`internal/matching/engine.go`:

```go
type Engine struct {
    books     map[string]*OrderBook
    router    chan *Order
    publisher *kafka.Producer
    consumer  *kafka.Consumer
}

func (e *Engine) Start(ctx context.Context) error {
    return e.consumer.Run(ctx, "orders", e.handleOrder)
}

func (e *Engine) handleOrder(ctx context.Context, msg kafka.Message) error {
    var order Order
    json.Unmarshal(msg.Value, &order)

    e.mu.Lock()
    book, exists := e.books[order.Symbol]
    if !exists {
        book = NewOrderBook(order.Symbol)
        e.books[order.Symbol] = book
        go e.runSymbol(order.Symbol, book)
    }
    e.mu.Unlock()

    trades := book.Match(&order)
    for _, trade := range trades {
        e.publisher.Publish(ctx, "trades", trade.Symbol, trade.ToJSON())
    }
    return nil
}

func (e *Engine) runSymbol(symbol string, book *OrderBook) {
    for trade := range book.TradeChannel {
        e.publisher.Publish(context.Background(), "trades", trade.Symbol, trade.ToJSON())
    }
}
```

Key design: **per-symbol goroutine** — no cross-symbol lock contention.

---

## Step 6.4 — State Recovery & Replay

To achieve <1ms latency, the Matching Engine keeps orderbooks in RAM. On startup, it must rebuild this state.

`internal/matching/engine.go`:

```go
func (e *Engine) Recover(ctx context.Context) error {
    // 1. Create a new Kafka reader without a GroupID 
    //    to read from the beginning of the topic
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers: []string{e.broker},
        Topic:   "orders",
        StartOffset: kafka.FirstOffset,
    })

    // 2. Replay all messages into Match() but SILENCE the publisher
    //    so we don't broadcast duplicate trades
    for {
        msg, err := reader.ReadMessage(ctx)
        if err != nil {
            break // End of topic or error
        }
        var order Order
        json.Unmarshal(msg.Value, &order)
        
        // Rebuild orderbook without emitting trades
        e.rebuildOrderbook(&order)
    }
    return nil
}
```

---

## Step 6.5 — Benchmark

```bash
go test -bench=BenchmarkMatchingEngine -benchmem -count=5 ./internal/matching/...
```

Targets:
- Throughput: > 10,000 orders/sec
- Latency p99: < 1ms

---

## Verification Checklist

- [ ] `go test ./internal/matching/...` passes
- [ ] BUY matches against lowest ASK when price >= ask
- [ ] Price-time priority (FIFO at same price)
- [ ] Partial fill handling works

> ➡️ Next: [Step 07 — Market Service](./07-market-service.md)