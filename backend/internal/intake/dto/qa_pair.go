package dto

import "github.com/google/uuid"

type QAPairItem struct {
	ID       uuid.UUID `json:"id"`
	Question string    `json:"question"`
	Answer   string    `json:"answer"`
}

type ListQAPairsResponse struct {
	Items []QAPairItem `json:"items"`
}
