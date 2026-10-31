package domain

import "github.com/google/uuid"

type GenerationJobExample struct {
	ID       uuid.UUID `db:"id"`
	JobID    uuid.UUID `db:"job_id"`
	QAPairID uuid.UUID `db:"qa_pair_id"`
	Question string    `db:"question"`
	Answer   string    `db:"answer"`
}
