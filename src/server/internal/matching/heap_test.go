package matching

import (
	"container/heap"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestBidHeapPricePriority(t *testing.T) {
	h := &BidHeap{}
	heap.Init(h)

	now := time.Now()
	heap.Push(h, &Order{ID: "1", Price: decimal.NewFromFloat(100.0), CreatedAt: now})
	heap.Push(h, &Order{ID: "2", Price: decimal.NewFromFloat(101.0), CreatedAt: now})
	heap.Push(h, &Order{ID: "3", Price: decimal.NewFromFloat(99.0), CreatedAt: now})

	// Highest price should be on top
	if (*h)[0].ID != "2" {
		t.Errorf("expected highest price (101) first, got %s", (*h)[0].ID)
	}
}

func TestAskHeapPricePriority(t *testing.T) {
	h := &AskHeap{}
	heap.Init(h)

	now := time.Now()
	heap.Push(h, &Order{ID: "1", Price: decimal.NewFromFloat(100.0), CreatedAt: now})
	heap.Push(h, &Order{ID: "2", Price: decimal.NewFromFloat(99.0), CreatedAt: now})
	heap.Push(h, &Order{ID: "3", Price: decimal.NewFromFloat(101.0), CreatedAt: now})

	// Lowest price should be on top
	if (*h)[0].ID != "2" {
		t.Errorf("expected lowest price (99) first, got %s", (*h)[0].ID)
	}
}

func TestBidHeapFIFOSamePrice(t *testing.T) {
	h := &BidHeap{}
	heap.Init(h)

	now := time.Now()
	heap.Push(h, &Order{ID: "1", Price: decimal.NewFromFloat(100.0), CreatedAt: now})
	heap.Push(h, &Order{ID: "2", Price: decimal.NewFromFloat(100.0), CreatedAt: now.Add(time.Millisecond)})

	// Earlier order should be higher priority at same price
	if (*h)[0].ID != "1" {
		t.Errorf("expected earlier order first, got %s", (*h)[0].ID)
	}
}

func TestAskHeapFIFOSamePrice(t *testing.T) {
	h := &AskHeap{}
	heap.Init(h)

	now := time.Now()
	heap.Push(h, &Order{ID: "1", Price: decimal.NewFromFloat(100.0), CreatedAt: now})
	heap.Push(h, &Order{ID: "2", Price: decimal.NewFromFloat(100.0), CreatedAt: now.Add(time.Millisecond)})

	// Earlier order should be higher priority at same price
	if (*h)[0].ID != "1" {
		t.Errorf("expected earlier order first, got %s", (*h)[0].ID)
	}
}

func TestBidHeapPop(t *testing.T) {
	h := &BidHeap{}
	heap.Init(h)

	now := time.Now()
	heap.Push(h, &Order{ID: "1", Price: decimal.NewFromFloat(100.0), CreatedAt: now})
	heap.Push(h, &Order{ID: "2", Price: decimal.NewFromFloat(101.0), CreatedAt: now})
	heap.Push(h, &Order{ID: "3", Price: decimal.NewFromFloat(99.0), CreatedAt: now})

	// Pop highest
	popped := heap.Pop(h).(*Order)
	if popped.ID != "2" {
		t.Errorf("expected to pop id=2, got %s", popped.ID)
	}

	// Next highest should be 100
	if (*h)[0].ID != "1" {
		t.Errorf("expected id=1 next, got %s", (*h)[0].ID)
	}
}

func TestOrderRemainingQty(t *testing.T) {
	o := &Order{
		ID:        "1",
		Quantity:  decimal.NewFromFloat(100.0),
		FilledQty: decimal.NewFromFloat(30.0),
	}

	remaining := o.RemainingQty()
	if !remaining.Equal(decimal.NewFromFloat(70.0)) {
		t.Errorf("expected remaining=70, got %s", remaining)
	}
}

func TestOrderFullyFilled(t *testing.T) {
	o := &Order{
		ID:        "1",
		Quantity:  decimal.NewFromFloat(100.0),
		FilledQty: decimal.NewFromFloat(100.0),
	}

	if !o.RemainingQty().IsZero() {
		t.Errorf("expected remaining=0, got %s", o.RemainingQty())
	}
}

func BenchmarkHeapPush(b *testing.B) {
	h := &BidHeap{}
	heap.Init(h)

	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		heap.Push(h, &Order{
			ID:        string(rune(i)),
			Price:     decimal.NewFromFloat(float64(i)),
			CreatedAt: now,
		})
	}
}

func BenchmarkHeapPop(b *testing.B) {
	h := &BidHeap{}
	heap.Init(h)

	now := time.Now()
	for i := 0; i < 1000; i++ {
		heap.Push(h, &Order{
			ID:        string(rune(i)),
			Price:     decimal.NewFromFloat(float64(i)),
			CreatedAt: now,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		heap.Pop(h)
		heap.Push(h, &Order{
			ID:        string(rune(i)),
			Price:     decimal.NewFromFloat(float64(i)),
			CreatedAt: now,
		})
	}
}