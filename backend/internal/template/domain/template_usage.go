package domain

import (
	"time"

	"github.com/google/uuid"
)

type TemplateUsedAs string

const (
	TemplateUsedAsAsIs   TemplateUsedAs = "as_is"
	TemplateUsedAsEdited TemplateUsedAs = "edited"
)

type TemplateUsage struct {
	ID             uuid.UUID      `db:"id"`
	TemplateID     uuid.UUID      `db:"template_id"`
	ConversationID *uuid.UUID     `db:"conversation_id"`
	QAPairID       *uuid.UUID     `db:"qa_pair_id"`
	UsedAs         TemplateUsedAs `db:"used_as"`
	SuggestedBody  string         `db:"suggested_body"`
	FinalBody      string         `db:"final_body"`
	Score          *float64       `db:"score"`
	OperatorID     *string        `db:"operator_id"`
	CreatedAt      time.Time      `db:"created_at"`
}
