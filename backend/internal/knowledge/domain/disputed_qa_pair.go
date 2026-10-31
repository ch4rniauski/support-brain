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
	ID                  uuid.UUID            `db:"id"`
	QAPairID            uuid.UUID            `db:"qa_pair_id"`
	Status              DisputedQAPairStatus `db:"status"`
	ResolutionClusterID *uuid.UUID           `db:"resolution_cluster_id"`
	ResolvedAt          *time.Time           `db:"resolved_at"`
}
