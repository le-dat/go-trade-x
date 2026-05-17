package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

// MessageHandler is a function that processes a kafka message.
type MessageHandler func(ctx context.Context, msg kafka.Message) error

// Consumer wraps a kafka reader for consuming messages.
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer creates a Consumer that reads from the given topic.
func NewConsumer(brokers []string, groupID, topic string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		MaxWait:        500 * time.Millisecond,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})
	return &Consumer{reader: reader}
}

// Run starts consuming messages and calls handler for each.
// It blocks until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			continue
		}
		if err := handler(ctx, msg); err != nil {
			// Log but continue processing
			continue
		}
	}
}

// Close closes the underlying kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}