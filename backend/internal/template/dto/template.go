package dto

import (
	"time"

	"github.com/google/uuid"
)

type TemplateItem struct {
	ID              uuid.UUID  `json:"id"`
	ClusterID       uuid.UUID  `json:"cluster_id"`
	Status          string     `json:"status"`
	Version         int        `json:"version"`
	Body            string     `json:"body"`
	GenerationJobID *uuid.UUID `json:"generation_job_id,omitempty"`
	UsedCount       int        `json:"used_count"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
}

type ListTemplatesResponse struct {
	Items []TemplateItem `json:"items"`
}

type UpdateTemplateRequest struct {
	Body string `json:"body"`
}
