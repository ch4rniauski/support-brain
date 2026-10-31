package domain

import (
	"time"

	"github.com/google/uuid"
)

type ConversationStatus string

const (
	ConversationStatusOpen   ConversationStatus = "open"
	ConversationStatusClosed ConversationStatus = "closed"
)

type Conversation struct {
	ID         uuid.UUID          `db:"id"`
	Status     ConversationStatus `db:"status"`
	OperatorID *string            `db:"operator_id"`
	StartedAt  time.Time          `db:"started_at"`
	ClosedAt   *time.Time         `db:"closed_at"`
}
