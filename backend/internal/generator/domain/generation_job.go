package domain

import (
	"time"

	"github.com/google/uuid"
)

type GenerationJobStatus string

const (
	GenerationJobStatusPending   GenerationJobStatus = "pending"
	GenerationJobStatusRunning   GenerationJobStatus = "running"
	GenerationJobStatusSucceeded GenerationJobStatus = "succeeded"
	GenerationJobStatusFailed    GenerationJobStatus = "failed"
)

type GenerationJob struct {
	ID            uuid.UUID
	ClusterID     uuid.UUID
	Status        GenerationJobStatus
	ModelName     string
	PromptVersion string
	OutputText    *string
	ErrorMessage  *string
	Attempts      int
	StartedAt     *time.Time
	FinishedAt    *time.Time
}
