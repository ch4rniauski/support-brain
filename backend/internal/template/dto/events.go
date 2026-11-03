package dto

import "github.com/google/uuid"

const EventTypeTemplateDraftCreated = "TemplateDraftCreated"

type TemplateDraftCreatedEvent struct {
	ClusterID       uuid.UUID `json:"cluster_id"`
	Text            string    `json:"text"`
	GenerationJobID uuid.UUID `json:"generation_job_id"`
}
