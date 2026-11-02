package dto

import "github.com/google/uuid"

const (
	EventTypeClusterReadyForGeneration = "ClusterReadyForGeneration"
	EventTypeTemplateDraftCreated      = "TemplateDraftCreated"
)

type ClusterReadyForGenerationEvent struct {
	ClusterID uuid.UUID   `json:"cluster_id"`
	QAPairIDs []uuid.UUID `json:"qa_pair_ids"`
}

type TemplateDraftCreatedEvent struct {
	ClusterID       uuid.UUID `json:"cluster_id"`
	Text            string    `json:"text"`
	ModelName       string    `json:"model_name"`
	PromptVersion   string    `json:"prompt_version"`
	GenerationJobID uuid.UUID `json:"generation_job_id"`
}
