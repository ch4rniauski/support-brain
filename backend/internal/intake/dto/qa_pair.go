package dto

import "github.com/google/uuid"

type ListQAPairsRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,dive,required"`
}

type QAPairItem struct {
	ID       uuid.UUID `json:"id"`
	Question string    `json:"question"`
	Answer   string    `json:"answer"`
}

type ListQAPairsResponse struct {
	Items []QAPairItem `json:"items"`
}
