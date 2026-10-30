package domain

import "github.com/google/uuid"

type ClusterMember struct {
	ID              uuid.UUID
	ClusterID       uuid.UUID
	QAPairID        uuid.UUID
	SimilarityScore *float64
}
