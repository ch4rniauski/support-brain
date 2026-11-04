package dto

import "github.com/google/uuid"

type SuggestTemplatesRequest struct {
	Question string `json:"question" validate:"required"`
}

type SuggestedTemplate struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	ClusterID uuid.UUID `json:"cluster_id"`
	Score     float64   `json:"score"`
}

type SuggestTemplatesResponse struct {
	Templates []SuggestedTemplate `json:"templates"`
}

type FindSimilarByTextRequest struct {
	Text      string  `json:"text"`
	Threshold float64 `json:"threshold"`
	Limit     int     `json:"limit"`
}

type SimilarQuestionResult struct {
	QAPairID uuid.UUID `json:"qa_pair_id"`
	Score    float64   `json:"score"`
}

type FindSimilarByTextResponse struct {
	Results []SimilarQuestionResult `json:"results"`
}

type FindTemplatesByQAPairsRequest struct {
	QAPairIDs []uuid.UUID `json:"qa_pair_ids"`
}

type TemplateByQAPairItem struct {
	QAPairID   uuid.UUID  `json:"qa_pair_id"`
	TemplateID *uuid.UUID `json:"template_id"`
}

type FindTemplatesByQAPairsResponse struct {
	Items []TemplateByQAPairItem `json:"items"`
}
