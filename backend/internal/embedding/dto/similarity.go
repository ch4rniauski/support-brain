package dto

import "github.com/google/uuid"

type FindSimilarRequest struct {
	QAPairID  uuid.UUID `json:"qa_pair_id" validate:"required"`
	ModelName string    `json:"model_name" validate:"required"`
	Threshold float64   `json:"threshold" validate:"required,gte=0,lte=1"`
	Limit     int       `json:"limit" validate:"required,gte=1,lte=100"`
}

type FindSimilarByTextRequest struct {
	Text      string  `json:"text" validate:"required"`
	Threshold float64 `json:"threshold" validate:"required,gte=0,lte=1"`
	Limit     int     `json:"limit" validate:"omitempty,gte=1,lte=100"`
}

type SimilarResult struct {
	QAPairID uuid.UUID `json:"qa_pair_id"`
	Score    float64   `json:"score"`
}

type FindSimilarResponse struct {
	QAPairID uuid.UUID       `json:"qa_pair_id"`
	Results  []SimilarResult `json:"results"`
}

type FindSimilarByTextResponse struct {
	Results []SimilarResult `json:"results"`
}
