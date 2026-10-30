package dto

import "github.com/google/uuid"

type SendMessageRequest struct {
	ConversationID *uuid.UUID `json:"conversation_id"`
	SenderType     string     `json:"sender_type" validate:"required,oneof=client operator"`
	Body           string     `json:"body" validate:"required,min=1"`
}

type SendMessageResponse struct {
	ConversationID uuid.UUID `json:"conversation_id"`
}
