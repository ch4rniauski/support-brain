package dto

import "github.com/google/uuid"

const (
	EventTypeEmbeddingReady            = "EmbeddingReady"
	EventTypeClusterReadyForGeneration = "ClusterReadyForGeneration"
)

type EmbeddingReadyEvent struct {
	QAPairID uuid.UUID `json:"qa_pair_id"`
}

type ClusterReadyForGenerationEvent struct {
	ClusterID uuid.UUID   `json:"cluster_id"`
	QAPairIDs []uuid.UUID `json:"qa_pair_ids"`
}
