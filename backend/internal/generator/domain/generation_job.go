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
	ID            uuid.UUID           `db:"id"`
	ClusterID     uuid.UUID           `db:"cluster_id"`
	Status        GenerationJobStatus `db:"status"`
	ModelName     string              `db:"model_name"`
	PromptVersion string              `db:"prompt_version"`
	OutputText    *string             `db:"output_text"`
	ErrorMessage  *string             `db:"error_message"`
	Attempts      int                 `db:"attempts"`
	StartedAt     *time.Time          `db:"started_at"`
	FinishedAt    *time.Time          `db:"finished_at"`
}
