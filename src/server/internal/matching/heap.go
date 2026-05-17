package matching

import (
	"time"

	"github.com/shopspring/decimal"
)

// Order represents a limit order in the matching engine.
type Order struct {
	ID        string
	Symbol    string
	Side      string // "BUY" or "SELL"
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	FilledQty decimal.Decimal
	CreatedAt time.Time
}

// RemainingQty returns the unfilled quantity.
func (o *Order) RemainingQty() decimal.Decimal {
	return o.Quantity.Sub(o.FilledQty)
}

// Trade represents a trade between two orders.
type Trade struct {
	BuyOrder  string
	SellOrder string
	Symbol    string
	Price     decimal.Decimal
	Quantity  decimal.Decimal
}

// BidHeap is a max-heap of orders by price (highest first).
type BidHeap []*Order

func (h BidHeap) Len() int            { return len(h) }
func (h BidHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h BidHeap) Less(i, j int) bool {
	if h[i].Price.Equal(h[j].Price) {
		return h[i].CreatedAt.Before(h[j].CreatedAt)
	}
	return h[i].Price.GreaterThan(h[j].Price)
}
func (h *BidHeap) Push(x any) { *h = append(*h, x.(*Order)) }
func (h *BidHeap) Pop() any {
	last := len(*h) - 1
	o := (*h)[last]
	*h = (*h)[:last]
	return o
}

// AskHeap is a min-heap of orders by price (lowest first).
type AskHeap []*Order

func (h AskHeap) Len() int            { return len(h) }
func (h AskHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h AskHeap) Less(i, j int) bool {
	if h[i].Price.Equal(h[j].Price) {
		return h[i].CreatedAt.Before(h[j].CreatedAt)
	}
	return h[i].Price.LessThan(h[j].Price)
}
func (h *AskHeap) Push(x any) { *h = append(*h, x.(*Order)) }
func (h *AskHeap) Pop() any {
	last := len(*h) - 1
	o := (*h)[last]
	*h = (*h)[:last]
	return o
}