package domain

import (
	"time"

	"github.com/google/uuid"
)

type TemplateStatus string

const (
	TemplateStatusDraft     TemplateStatus = "draft"
	TemplateStatusApproved  TemplateStatus = "approved"
	TemplateStatusRejected  TemplateStatus = "rejected"
	TemplateStatusArchived  TemplateStatus = "archived"
)

type Template struct {
	ID              uuid.UUID
	ClusterID       uuid.UUID
	Status          TemplateStatus
	Version         int
	Body            string
	GenerationJobID *uuid.UUID
	UsedCount       int
	ApprovedAt      *time.Time
	ApprovedBy      *string
}
