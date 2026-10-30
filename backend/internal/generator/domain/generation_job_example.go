package domain

import "github.com/google/uuid"

type GenerationJobExample struct {
	ID       uuid.UUID
	JobID    uuid.UUID
	QAPairID uuid.UUID
	Question string
	Answer   string
}
