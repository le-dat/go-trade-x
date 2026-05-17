package matching

import (
	"container/heap"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestOrderBookMatchBuyPartialFill(t *testing.T) {
	ob := NewOrderBook("BTC/USD")

	// Existing ask at 100
	heap.Push(&ob.asks, &Order{
		ID:       "ask1",
		Price:    decimal.NewFromFloat(100.0),
		Quantity: decimal.NewFromFloat(5.0),
	})

	// Incoming buy order for 3 at price 100
	incoming := &Order{
		ID:        "buy1",
		Symbol:    "BTC/USD",
		Side:      "BUY",
		Price:     decimal.NewFromFloat(100.0),
		Quantity:  decimal.NewFromFloat(3.0),
		CreatedAt: time.Now(),
	}

	trades := ob.Match(incoming)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity.InexactFloat64() != 3.0 {
		t.Errorf("expected fill qty 3, got %s", trades[0].Quantity)
	}
	if trades[0].BuyOrder != "buy1" || trades[0].SellOrder != "ask1" {
		t.Errorf("wrong trade parties: buy=%s sell=%s", trades[0].BuyOrder, trades[0].SellOrder)
	}

	// Ask should be partially filled
	if ob.asks[0].FilledQty.InexactFloat64() != 3.0 {
		t.Errorf("ask filled qty = %s, expected 3", ob.asks[0].FilledQty)
	}
}

func TestOrderBookMatchBuyFullFill(t *testing.T) {
	ob := NewOrderBook("BTC/USD")

	// Existing ask at 100 for 5
	heap.Push(&ob.asks, &Order{
		ID:       "ask1",
		Price:    decimal.NewFromFloat(100.0),
		Quantity: decimal.NewFromFloat(5.0),
	})

	// Incoming buy for 10 at price 100 (will fully consume ask of 5, remaining 5 goes to book)
	incoming := &Order{
		ID:        "buy1",
		Symbol:    "BTC/USD",
		Side:      "BUY",
		Price:     decimal.NewFromFloat(100.0),
		Quantity:  decimal.NewFromFloat(10.0),
		CreatedAt: time.Now(),
	}

	trades := ob.Match(incoming)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity.InexactFloat64() != 5.0 {
		t.Errorf("expected fill qty 5, got %s", trades[0].Quantity)
	}

	// Ask should be fully consumed
	if len(ob.asks) != 0 {
		t.Errorf("ask heap should be empty, has %d items", len(ob.asks))
	}
	// Insert remaining qty into bids
	if remaining := incoming.RemainingQty(); remaining.GreaterThan(decimal.Zero) {
		ob.Insert(&Order{
			ID:        incoming.ID,
			Symbol:    incoming.Symbol,
			Side:      incoming.Side,
			Price:     incoming.Price,
			Quantity:  remaining,
			FilledQty: decimal.Zero,
			CreatedAt: incoming.CreatedAt,
		})
	}
	// Buy order should be inserted into bids with remaining qty 5
	if len(ob.bids) != 1 {
		t.Errorf("bids should have 1 order, has %d", len(ob.bids))
	}
}

func TestOrderBookMatchBuyMultipleAsks(t *testing.T) {
	ob := NewOrderBook("BTC/USD")

	// Add two asks
	heap.Push(&ob.asks, &Order{
		ID:       "ask1",
		Price:    decimal.NewFromFloat(100.0),
		Quantity: decimal.NewFromFloat(3.0),
	})
	heap.Push(&ob.asks, &Order{
		ID:       "ask2",
		Price:    decimal.NewFromFloat(101.0),
		Quantity: decimal.NewFromFloat(5.0),
	})
	heap.Init(&ob.asks)

	// Buy for 6 at price 101
	incoming := &Order{
		ID:        "buy1",
		Symbol:    "BTC/USD",
		Side:      "BUY",
		Price:     decimal.NewFromFloat(101.0),
		Quantity:  decimal.NewFromFloat(6.0),
		CreatedAt: time.Now(),
	}

	trades := ob.Match(incoming)

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}
	// First trade against ask1 (price 100)
	if trades[0].SellOrder != "ask1" || trades[0].Quantity.InexactFloat64() != 3.0 {
		t.Errorf("first trade: sell=%s qty=%s", trades[0].SellOrder, trades[0].Quantity)
	}
	// Second trade against ask2 (price 101)
	if trades[1].SellOrder != "ask2" || trades[1].Quantity.InexactFloat64() != 3.0 {
		t.Errorf("second trade: sell=%s qty=%s", trades[1].SellOrder, trades[1].Quantity)
	}
}

func TestOrderBookMatchBuyInsufficientPrice(t *testing.T) {
	ob := NewOrderBook("BTC/USD")

	// Existing ask at 100
	heap.Push(&ob.asks, &Order{
		ID:       "ask1",
		Price:    decimal.NewFromFloat(100.0),
		Quantity: decimal.NewFromFloat(5.0),
	})
	heap.Init(&ob.asks)

	// Incoming buy at price 99 (below ask)
	incoming := &Order{
		ID:        "buy1",
		Symbol:    "BTC/USD",
		Side:      "BUY",
		Price:     decimal.NewFromFloat(99.0),
		Quantity:  decimal.NewFromFloat(5.0),
		CreatedAt: time.Now(),
	}

	trades := ob.Match(incoming)

	// No trades should occur
	if len(trades) != 0 {
		t.Errorf("expected 0 trades, got %d", len(trades))
	}
	// Order should be inserted into bids (caller's responsibility)
	ob.Insert(incoming)
	if len(ob.bids) != 1 {
		t.Errorf("bids should have 1 order, has %d", len(ob.bids))
	}
}

func TestOrderBookMatchSellAgainstBids(t *testing.T) {
	ob := NewOrderBook("BTC/USD")

	// Existing bid at 100
	heap.Push(&ob.bids, &Order{
		ID:       "bid1",
		Price:    decimal.NewFromFloat(100.0),
		Quantity: decimal.NewFromFloat(5.0),
	})
	heap.Init(&ob.bids)

	// Incoming sell at price 100
	incoming := &Order{
		ID:        "sell1",
		Symbol:    "BTC/USD",
		Side:      "SELL",
		Price:     decimal.NewFromFloat(100.0),
		Quantity:  decimal.NewFromFloat(3.0),
		CreatedAt: time.Now(),
	}

	trades := ob.Match(incoming)

	if len(trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades))
	}
	if trades[0].BuyOrder != "bid1" || trades[0].SellOrder != "sell1" {
		t.Errorf("wrong trade parties: buy=%s sell=%s", trades[0].BuyOrder, trades[0].SellOrder)
	}
}

func BenchmarkMatchingEngine(b *testing.B) {
	ob := NewOrderBook("BTC/USD")

	// Pre-populate with 1000 asks
	for i := 0; i < 1000; i++ {
		heap.Push(&ob.asks, &Order{
			ID:       string(rune('a'+i%26)) + string(rune('0'+i%10)),
			Price:    decimal.NewFromFloat(100.0 + float64(i)*0.01),
			Quantity: decimal.NewFromFloat(10.0),
		})
	}
	heap.Init(&ob.asks)

	incoming := &Order{
		ID:        "bench",
		Symbol:    "BTC/USD",
		Side:      "BUY",
		Price:     decimal.NewFromFloat(110.0),
		Quantity:  decimal.NewFromFloat(1.0),
		CreatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use a fresh order each time to avoid modifications
		o := *incoming
		ob.Match(&o)
	}
}