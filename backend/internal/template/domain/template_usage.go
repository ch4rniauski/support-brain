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
	ID             uuid.UUID
	TemplateID     uuid.UUID
	ConversationID *uuid.UUID
	QAPairID       *uuid.UUID
	UsedAs         TemplateUsedAs
	SuggestedBody  string
	FinalBody      string
	Score          *float64
	OperatorID     *string
	CreatedAt      time.Time
}
