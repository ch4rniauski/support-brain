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
	ID         uuid.UUID
	Status     ConversationStatus
	OperatorID *string
	StartedAt  time.Time
	ClosedAt   *time.Time
}
