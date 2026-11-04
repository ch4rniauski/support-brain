package dto

import "github.com/google/uuid"

type ListQAPairsRequest struct {
	IDs []uuid.UUID `json:"ids"`
}

type IntakeQAPairItem struct {
	ID       uuid.UUID `json:"id"`
	Question string    `json:"question"`
	Answer   string    `json:"answer"`
}

type ListQAPairsResponse struct {
	Items []IntakeQAPairItem `json:"items"`
}
