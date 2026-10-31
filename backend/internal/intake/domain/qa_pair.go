package domain

import "github.com/google/uuid"

type AnswerSource string

const (
	AnswerSourceManual         AnswerSource = "manual"
	AnswerSourceTemplateAsIs   AnswerSource = "template_as_is"
	AnswerSourceTemplateEdited AnswerSource = "template_edited"
)

type QAPair struct {
	ID             uuid.UUID    `db:"id"`
	ConversationID *uuid.UUID   `db:"conversation_id"`
	Question       string       `db:"question"`
	Answer         string       `db:"answer"`
	AnswerSource   AnswerSource `db:"answer_source"`
	TemplateID     *uuid.UUID   `db:"template_id"`
	OperatorID     *string      `db:"operator_id"`
}
