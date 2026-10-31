package domain

import "github.com/google/uuid"

type DisputedCandidate struct {
	ID               uuid.UUID `db:"id"`
	DisputedQAPairID uuid.UUID `db:"disputed_qa_pair_id"`
	ClusterID        uuid.UUID `db:"cluster_id"`
	NeighborQAPairID uuid.UUID `db:"neighbor_qa_pair_id"`
	Score            float64   `db:"score"`
}
