package domain

import "github.com/google/uuid"

type ClusterStatus string

const (
	ClusterStatusCollecting       ClusterStatus = "collecting"
	ClusterStatusReadyToGenerate  ClusterStatus = "ready_to_generate"
	ClusterStatusHasDraft         ClusterStatus = "has_draft"
	ClusterStatusHasApproved      ClusterStatus = "has_approved"
)

type Cluster struct {
	ID                   uuid.UUID
	Title                *string
	Status               ClusterStatus
	MemberCount          int
	SimilarityThreshold  float64
	GenerationThresholdN int
	ActiveTemplateID     *uuid.UUID
}
