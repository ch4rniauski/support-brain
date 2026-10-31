package domain

import (
	"time"

	"github.com/google/uuid"
)

const EventTypeQAPairCreated = "QaPairCreated"

type OutboxEvent struct {
	ID          uuid.UUID  `db:"id"`
	EventType   string     `db:"event_type"`
	AggregateID uuid.UUID  `db:"aggregate_id"`
	Payload     []byte     `db:"payload"`
	CreatedAt   time.Time  `db:"created_at"`
	PublishedAt *time.Time `db:"published_at"`
}
