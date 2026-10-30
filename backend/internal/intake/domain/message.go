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
	ID             uuid.UUID
	ConversationID uuid.UUID
	SenderType     SenderType
	Body           string
	SentAt         time.Time
}
