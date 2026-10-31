package domain

import (
	"time"

	"github.com/google/uuid"
)

type SenderType string

const (
	SenderTypeClient   SenderType = "client"
	SenderTypeOperator SenderType = "operator"
)

type Message struct {
	ID             uuid.UUID  `db:"id"`
	ConversationID uuid.UUID  `db:"conversation_id"`
	SenderType     SenderType `db:"sender_type"`
	Body           string     `db:"body"`
	SentAt         time.Time  `db:"sent_at"`
}
