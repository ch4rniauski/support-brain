package domain

import "github.com/google/uuid"

type ClusterStatus string

const (
	ClusterStatusCollecting      ClusterStatus = "collecting"
	ClusterStatusReadyToGenerate ClusterStatus = "ready_to_generate"
	ClusterStatusHasDraft        ClusterStatus = "has_draft"
	ClusterStatusHasApproved     ClusterStatus = "has_approved"
)

type Cluster struct {
	ID                   uuid.UUID     `db:"id"`
	Title                *string       `db:"title"`
	Status               ClusterStatus `db:"status"`
	MemberCount          int           `db:"member_count"`
	SimilarityThreshold  float64       `db:"similarity_threshold"`
	GenerationThresholdN int           `db:"generation_threshold_n"`
	ActiveTemplateID     *uuid.UUID    `db:"active_template_id"`
}
