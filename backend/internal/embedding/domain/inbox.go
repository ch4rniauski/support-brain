package domain

import (
	"time"

	"github.com/google/uuid"
)

type InboxEvent struct {
	IdempotencyKey uuid.UUID `db:"idempotency_key"`
	EventType      string    `db:"event_type"`
	Payload        []byte    `db:"payload"`
	ProcessedAt    time.Time `db:"processed_at"`
}
