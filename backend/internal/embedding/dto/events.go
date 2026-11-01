package dto

import "github.com/google/uuid"

const (
	EventTypeQAPairCreated  = "QaPairCreated"
	EventTypeEmbeddingReady = "EmbeddingReady"
)

type QAPairCreatedEvent struct {
	QAPairID uuid.UUID `json:"qa_pair_id"`
	Question string    `json:"question"`
	Answer   string    `json:"answer"`
}

type EmbeddingReadyEvent struct {
	QAPairID uuid.UUID `json:"qa_pair_id"`
}
