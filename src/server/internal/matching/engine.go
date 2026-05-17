package matching

import (
	"context"
	"encoding/json"
	"sync"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/verno/gotradex/pkg/kafka"
)

// Engine is the matching engine that consumes orders and produces trades.
type Engine struct {
	mu       sync.RWMutex
	books    map[string]*OrderBook
	router   chan *Order
	consumer *kafka.Consumer
	pub      *kafka.Producer
	log      *zap.Logger
}

// NewEngine creates a new matching engine.
func NewEngine(brokers []string, log *zap.Logger) *Engine {
	return &Engine{
		books:    make(map[string]*OrderBook),
		router:   make(chan *Order, 10000),
		consumer: kafka.NewConsumer(brokers, "matching-engine", "orders"),
		pub:      kafka.NewProducer(brokers),
		log:      log,
	}
}

// Start begins consuming orders from Kafka and matching them.
func (e *Engine) Start(ctx context.Context) error {
	return e.consumer.Run(ctx, e.handleOrder)
}

// handleOrder processes a single order message.
func (e *Engine) handleOrder(_ context.Context, msg kafkago.Message) error {
	var incoming Order
	if err := json.Unmarshal(msg.Value, &incoming); err != nil {
		e.log.With(zap.Error(err)).Warn("failed to unmarshal order message")
		return nil // Don't retry malformed messages
	}

	e.mu.Lock()
	book, exists := e.books[incoming.Symbol]
	if !exists {
		book = NewOrderBook(incoming.Symbol)
		e.books[incoming.Symbol] = book
		go e.runSymbol(incoming.Symbol, book)
	}
	e.mu.Unlock()

	trades := book.Match(&incoming)
	// Insert remaining quantity into heap
	if remaining := incoming.RemainingQty(); remaining.GreaterThan(decimal.Zero) {
		book.Insert(&incoming)
	}

	for _, trade := range trades {
		tradeJSON, _ := json.Marshal(trade)
		e.pub.Publish(context.Background(), "trades", trade.Symbol, tradeJSON)
	}

	return nil
}

// runSymbol processes trades for a specific symbol (per-symbol goroutine).
func (e *Engine) runSymbol(symbol string, book *OrderBook) {
	e.log.With(zap.String("symbol", symbol)).Info("matching engine: started symbol goroutine")
	// Symbol goroutine currently just logs; trades are published directly in handleOrder
	// This goroutine exists to allow future per-symbol trade channel handling
}

// Recover rebuilds the order book state from the order topic without publishing trades.
func (e *Engine) Recover(ctx context.Context) error {
	e.log.Info("matching engine: state recovery not yet implemented")
	// In production, this would replay all historical orders to rebuild the orderbook
	// without emitting trades, using a non-group Kafka reader.
	return nil
}

// Stop gracefully shuts down the engine.
func (e *Engine) Stop() error {
	if err := e.consumer.Close(); err != nil {
		return err
	}
	return e.pub.Close()
}