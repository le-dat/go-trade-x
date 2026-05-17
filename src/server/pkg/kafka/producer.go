package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer wraps a kafka writer for publishing messages.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a Producer that writes to the given brokers.
func NewProducer(brokers []string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}
	return &Producer{writer: writer}
}

// Publish sends a message to the specified topic with the given key.
// The payload is serialized by the caller before calling Publish.
func (p *Producer) Publish(ctx context.Context, topic, key string, payload []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
	})
}

// Close closes the underlying kafka writer.
func (p *Producer) Close() error {
	return p.writer.Close()
}