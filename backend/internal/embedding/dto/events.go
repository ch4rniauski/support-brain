package dto

import "github.com/google/uuid"

const EventTypeQAPairCreated = "QaPairCreated"

type QAPairCreatedEvent struct {
	QAPairID uuid.UUID `json:"qa_pair_id"`
	Question string    `json:"question"`
	Answer   string    `json:"answer"`
}
