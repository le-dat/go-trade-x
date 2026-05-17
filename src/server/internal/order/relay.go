package order

import (
	"context"
	"time"

	"github.com/verno/gotradex/pkg/kafka"
	"go.uber.org/zap"
)

// Relay polls the outbox table and publishes pending messages to Kafka.
type Relay struct {
	repo     Repository
	prod     *kafka.Producer
	log      *zap.Logger
	interval time.Duration
}

// NewRelay creates a new outbox relay.
func NewRelay(repo Repository, prod *kafka.Producer, log *zap.Logger) *Relay {
	return &Relay{
		repo:     repo,
		prod:     prod,
		log:      log,
		interval: 100 * time.Millisecond,
	}
}

// Start runs the relay loop until ctx is cancelled.
func (r *Relay) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.log.Info("Outbox relay started")
	for {
		select {
		case <-ctx.Done():
			r.log.Info("Outbox relay stopped")
			return
		case <-ticker.C:
			r.processPending(ctx)
		}
	}
}

func (r *Relay) processPending(ctx context.Context) {
	messages, err := r.repo.GetPendingOutbox(ctx, 100)
	if err != nil {
		r.log.With(zap.Error(err)).Error("failed to fetch pending outbox messages")
		return
	}

	for _, msg := range messages {
		if err := r.prod.Publish(ctx, msg.Topic, msg.Key, msg.Payload); err != nil {
			r.log.With(zap.Error(err)).Error("failed to publish outbox message",
				zap.String("msg_id", msg.ID.String()),
				zap.String("topic", msg.Topic),
			)
			// Mark as failed to prevent infinite retry loop; manual intervention needed
			if markErr := r.repo.MarkOutboxFailed(ctx, msg.ID); markErr != nil {
				r.log.With(zap.Error(markErr)).Error("failed to mark outbox as failed",
					zap.String("msg_id", msg.ID.String()),
				)
			}
			continue
		}
		if err := r.repo.MarkOutboxProcessed(ctx, msg.ID); err != nil {
			r.log.With(zap.Error(err)).Error("failed to mark outbox processed",
				zap.String("msg_id", msg.ID.String()),
			)
			// Don't continue — message stays PENDING and will be retried.
			// This is intentional: we don't want to publish duplicate events.
		}
	}
}