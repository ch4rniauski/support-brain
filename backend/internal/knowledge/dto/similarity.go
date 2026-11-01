package dto

import "github.com/google/uuid"

type FindSimilarRequest struct {
	QAPairID  uuid.UUID `json:"qa_pair_id"`
	ModelName string    `json:"model_name"`
	Threshold float64   `json:"threshold"`
	Limit     int       `json:"limit"`
}

type SimilarResult struct {
	QAPairID uuid.UUID `json:"qa_pair_id"`
	Score    float64   `json:"score"`
}

type FindSimilarResponse struct {
	QAPairID uuid.UUID       `json:"qa_pair_id"`
	Results  []SimilarResult `json:"results"`
}
