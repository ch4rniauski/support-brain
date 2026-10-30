package domain

import (
	"time"

	"github.com/google/uuid"
)

type DisputedQAPairStatus string

const (
	DisputedQAPairStatusPending  DisputedQAPairStatus = "pending"
	DisputedQAPairStatusResolved DisputedQAPairStatus = "resolved"
)

type DisputedQAPair struct {
	ID                   uuid.UUID
	QAPairID             uuid.UUID
	Status               DisputedQAPairStatus
	ResolutionClusterID  *uuid.UUID
	ResolvedAt           *time.Time
}
