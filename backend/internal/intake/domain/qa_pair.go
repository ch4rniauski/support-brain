package domain

import "github.com/google/uuid"

type AnswerSource string

const (
	AnswerSourceManual         AnswerSource = "manual"
	AnswerSourceTemplateAsIs   AnswerSource = "template_as_is"
	AnswerSourceTemplateEdited AnswerSource = "template_edited"
)

type QAPair struct {
	ID             uuid.UUID
	ConversationID *uuid.UUID
	Question       string
	Answer         string
	AnswerSource   AnswerSource
	TemplateID     *uuid.UUID
	OperatorID     *string
}
