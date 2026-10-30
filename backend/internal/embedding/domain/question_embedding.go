package domain

import "github.com/google/uuid"

type QuestionEmbedding struct {
	ID        uuid.UUID
	QAPairID  uuid.UUID
	ModelName string
	Embedding []float32
}
