package domain

import "github.com/google/uuid"

type DisputedCandidate struct {
	ID                uuid.UUID
	DisputedQAPairID  uuid.UUID
	ClusterID         uuid.UUID
	NeighborQAPairID  uuid.UUID
	Score             float64
}
