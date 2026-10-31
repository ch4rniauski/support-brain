package domain

import (
	"time"

	"github.com/google/uuid"
)

type TemplateStatus string

const (
	TemplateStatusDraft    TemplateStatus = "draft"
	TemplateStatusApproved TemplateStatus = "approved"
	TemplateStatusRejected TemplateStatus = "rejected"
	TemplateStatusArchived TemplateStatus = "archived"
)

type Template struct {
	ID              uuid.UUID      `db:"id"`
	ClusterID       uuid.UUID      `db:"cluster_id"`
	Status          TemplateStatus `db:"status"`
	Version         int            `db:"version"`
	Body            string         `db:"body"`
	GenerationJobID *uuid.UUID     `db:"generation_job_id"`
	UsedCount       int            `db:"used_count"`
	ApprovedAt      *time.Time     `db:"approved_at"`
	ApprovedBy      *string        `db:"approved_by"`
}
