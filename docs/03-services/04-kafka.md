# Step 04 — Kafka Integration: Producer & Consumer

## Goal

Implement reusable Kafka producer and consumer in `pkg/kafka`.

> **Prerequisite**: [Step 03 — User Service](./03-user-service.md)

---

## Step 4.1 — Producer

`pkg/kafka/producer.go`:

```go
type Producer interface {
    Publish(ctx context.Context, topic string, key string, value []byte) error
    Close() error
}

type kafkaProducer struct {
    writer *kafka.Writer
}

func NewProducer(broker string) Producer {
    return &kafkaProducer{
        writer: &kafka.Writer{
            Addr:         kafka.TCP(broker),
            RequiredAcks: kafka.RequireAll,
            MaxAttempts:  3,
            BatchSize:    1,
        },
    }
}

func (p *kafkaProducer) Publish(ctx context.Context, topic string, key string, value []byte) error {
    return p.writer.WriteMessages(ctx, kafka.Message{
        Key:   []byte(key),
        Value: value,
        Topic: topic,
    })
}
```

Key features:
- `RequiredAcks: RequireAll` — wait for all replicas
- `MaxAttempts: 3` — retry with exponential backoff
- `Key = userID` — per-user ordering guarantees
- `BatchSize: 1` — immediate send for low latency

---

## Step 4.2 — Consumer

`pkg/kafka/consumer.go`:

```go
type Handler func(ctx context.Context, msg kafka.Message) error

type Consumer interface {
    Run(ctx context.Context, topic string, handler Handler) error
}

type kafkaConsumer struct {
    reader *kafka.Reader
}

func NewConsumer(broker, groupID, topic string) Consumer {
    return &kafkaConsumer{
        reader: kafka.NewReader(kafka.ReaderConfig{
            Brokers:  []string{broker},
            GroupID: groupID,
            Topic:   topic,
        }),
    }
}

func (c *kafkaConsumer) Run(ctx context.Context, topic string, handler Handler) error {
    for {
        select {
        case <-ctx.Done():
            return c.reader.Close()
        default:
            msg, err := c.reader.FetchMessage(ctx)
            if err != nil {
                continue
            }
            if err := handler(ctx, msg); err != nil {
                continue
            }
            if err := c.reader.CommitMessages(ctx, msg); err != nil {
                continue
            }
        }
    }
}
```

Key features:
- Commit offset **only after successful `Handler` return**
- Graceful shutdown on context cancellation

---

## Step 4.3 — Round-trip Test

```bash
make docker-up

go test ./pkg/kafka/... -v -run TestRoundTrip
```

---

## Verification Checklist

- [ ] Producer retries on failure (max 3 attempts)
- [ ] Consumer commits offset only after successful handler
- [ ] Graceful shutdown works
- [ ] Round-trip test passes

> ➡️ Next: [Step 05 — Order Service](./05-order-service.md)