package domain

import "github.com/google/uuid"

type ClusterMember struct {
	ID              uuid.UUID `db:"id"`
	ClusterID       uuid.UUID `db:"cluster_id"`
	QAPairID        uuid.UUID `db:"qa_pair_id"`
	SimilarityScore *float64  `db:"similarity_score"`
}
