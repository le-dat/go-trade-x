package matching

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/verno/gotradex/pkg/kafka"
)

// Engine is the matching engine that consumes orders and produces trades.
type Engine struct {
	mu       sync.Mutex
	books    map[string]*OrderBook
	consumer *kafka.Consumer
	pub      *kafka.Producer
	log      *zap.Logger
}

// NewEngine creates a new matching engine.
func NewEngine(brokers []string, groupID string, log *zap.Logger) *Engine {
	return &Engine{
		books:    make(map[string]*OrderBook),
		consumer: kafka.NewConsumer(brokers, groupID, "orders", log),
		pub:      kafka.NewProducer(brokers),
		log:      log,
	}
}

// Start begins consuming orders from Kafka and matching them.
func (e *Engine) Start(ctx context.Context) error {
	return e.consumer.Run(ctx, e.handleOrder)
}

// validateOrder checks order fields and returns an error if invalid.
func (e *Engine) validateOrder(order *Order) error {
	if order.Price.LessThanOrEqual(decimal.Zero) {
		return errors.New("order price must be positive")
	}
	if order.Quantity.LessThanOrEqual(decimal.Zero) {
		return errors.New("order quantity must be positive")
	}
	if order.FilledQty.GreaterThan(order.Quantity) {
		return errors.New("filled quantity cannot exceed order quantity")
	}
	if order.Side != "BUY" && order.Side != "SELL" {
		return errors.New("order side must be BUY or SELL")
	}
	return nil
}

// handleOrder processes a single order message.
func (e *Engine) handleOrder(_ context.Context, msg kafkago.Message) error {
	var incoming Order
	if err := json.Unmarshal(msg.Value, &incoming); err != nil {
		e.log.With(zap.Error(err)).Warn("failed to unmarshal order message")
		return nil // Don't retry malformed messages
	}

	if err := e.validateOrder(&incoming); err != nil {
		e.log.With(zap.Error(err)).Warn("invalid order", zap.String("order_id", incoming.ID))
		return nil // Drop invalid orders
	}

	// Get or create orderbook for symbol and process order under lock
	e.mu.Lock()
	book, exists := e.books[incoming.Symbol]
	if !exists {
		book = NewOrderBook(incoming.Symbol)
		e.books[incoming.Symbol] = book
	}

	trades := book.Match(&incoming)
	// Insert remaining quantity into heap
	if remaining := incoming.RemainingQty(); remaining.GreaterThan(decimal.Zero) {
		book.Insert(&incoming)
	}
	e.mu.Unlock()

	for _, trade := range trades {
		tradeJSON, err := json.Marshal(trade)
		if err != nil {
			e.log.With(zap.Error(err)).Error("failed to marshal trade")
			continue
		}
		if err := e.pub.Publish(context.Background(), "trades", trade.Symbol, tradeJSON); err != nil {
			e.log.With(zap.Error(err)).Error("failed to publish trade",
				zap.String("trade_id", trade.BuyOrder+"-"+trade.SellOrder))
		}
	}

	return nil
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