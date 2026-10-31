package dto

import "github.com/google/uuid"

type SendMessageRequest struct {
	ConversationID *uuid.UUID `json:"conversation_id"`
	Body           string     `json:"body" validate:"required,min=1"`
}

type ReplyMessageRequest struct {
	ConversationID uuid.UUID `json:"conversation_id" validate:"required"`
	Question       string    `json:"question" validate:"required,min=1"`
	Body           string    `json:"body" validate:"required,min=1"`
}

type SendMessageResponse struct {
	ConversationID uuid.UUID `json:"conversation_id"`
}
