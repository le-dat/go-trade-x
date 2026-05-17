package matching

import (
	"container/heap"
	"sync"

	"github.com/shopspring/decimal"
)

// OrderBook maintains bid and ask heaps for a single symbol.
type OrderBook struct {
	symbol string
	bids   BidHeap
	asks   AskHeap
	mu     sync.Mutex
}

// NewOrderBook creates a new order book for the given symbol.
func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{symbol: symbol}
}

// Match attempts to fill an incoming order against the opposite side.
// Returns trades that were generated. Remaining quantity stays in the order
// for the caller's heap insertion.
func (ob *OrderBook) Match(incoming *Order) []Trade {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var trades []Trade
	remaining := incoming.RemainingQty()

	if incoming.Side == "BUY" {
		trades = ob.matchBuy(remaining, incoming)
	} else {
		trades = ob.matchSell(remaining, incoming)
	}

	return trades
}

func (ob *OrderBook) matchBuy(remaining decimal.Decimal, incoming *Order) []Trade {
	var trades []Trade
	for remaining.GreaterThan(decimal.Zero) && len(ob.asks) > 0 {
		bestAsk := ob.asks[0]
		if incoming.Price.LessThan(bestAsk.Price) {
			break
		}
		fillQty := minDecimal(remaining, bestAsk.RemainingQty())
		trades = append(trades, Trade{
			BuyOrder:  incoming.ID,
			SellOrder: bestAsk.ID,
			Symbol:    incoming.Symbol,
			Price:     bestAsk.Price,
			Quantity:  fillQty,
		})
		bestAsk.FilledQty = bestAsk.FilledQty.Add(fillQty)
		remaining = remaining.Sub(fillQty)
		if bestAsk.RemainingQty().IsZero() {
			heap.Pop(&ob.asks)
		}
	}
	return trades
}

func (ob *OrderBook) matchSell(remaining decimal.Decimal, incoming *Order) []Trade {
	var trades []Trade
	for remaining.GreaterThan(decimal.Zero) && len(ob.bids) > 0 {
		bestBid := ob.bids[0]
		if incoming.Price.GreaterThan(bestBid.Price) {
			break
		}
		fillQty := minDecimal(remaining, bestBid.RemainingQty())
		trades = append(trades, Trade{
			BuyOrder:  bestBid.ID,
			SellOrder: incoming.ID,
			Symbol:    incoming.Symbol,
			Price:     bestBid.Price,
			Quantity:  fillQty,
		})
		bestBid.FilledQty = bestBid.FilledQty.Add(fillQty)
		remaining = remaining.Sub(fillQty)
		if bestBid.RemainingQty().IsZero() {
			heap.Pop(&ob.bids)
		}
	}
	return trades
}

// Insert inserts a fully-remaining order into the appropriate heap.
// Call after Match when remaining > 0.
func (ob *OrderBook) Insert(order *Order) {
	if order.Side == "BUY" {
		heap.Push(&ob.bids, order)
	} else {
		heap.Push(&ob.asks, order)
	}
}

func minDecimal(a, b decimal.Decimal) decimal.Decimal {
	if a.LessThan(b) {
		return a
	}
	return b
}