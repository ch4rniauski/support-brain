package outbox

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
)

type eventStore interface {
	ListUnpublished(ctx context.Context, limit int) ([]Event, error)
	MarkPublished(ctx context.Context, id uuid.UUID, publishedAt time.Time) error
}

type messagePublisher interface {
	Publish(ctx context.Context, topic string, key string, eventType string, payload []byte) error
}

type Publisher struct {
	store    eventStore
	producer messagePublisher
	topic    string
	interval time.Duration
	batch    int
}

func NewPublisher(
	store eventStore,
	producer messagePublisher,
	topic string,
	interval time.Duration,
	batch int,
) *Publisher {
	if interval <= 0 {
		interval = time.Second
	}
	if batch <= 0 {
		batch = 100
	}

	return &Publisher{
		store:    store,
		producer: producer,
		topic:    topic,
		interval: interval,
		batch:    batch,
	}
}

func (p *Publisher) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		if err := p.publishPending(ctx); err != nil {
			log.Printf("outbox publisher: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *Publisher) publishPending(ctx context.Context) error {
	events, err := p.store.ListUnpublished(ctx, p.batch)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := p.producer.Publish(
			ctx,
			p.topic,
			event.AggregateID.String(),
			event.EventType,
			event.Payload,
		); err != nil {
			return err
		}

		if err := p.store.MarkPublished(ctx, event.ID, time.Now().UTC()); err != nil {
			return err
		}
	}

	return nil
}
