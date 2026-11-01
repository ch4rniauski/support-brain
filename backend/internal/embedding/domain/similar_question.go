package domain

import "github.com/google/uuid"

type SimilarQuestion struct {
	QAPairID uuid.UUID
	Score    float64
}
