package domain

import "github.com/google/uuid"

type QuestionEmbedding struct {
	ID        uuid.UUID `db:"id"`
	QAPairID  uuid.UUID `db:"qa_pair_id"`
	ModelName string    `db:"model_name"`
	Embedding []float32 `db:"embedding"`
}
