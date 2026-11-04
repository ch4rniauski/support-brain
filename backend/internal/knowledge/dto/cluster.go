package dto

import "github.com/google/uuid"

type FindTemplatesByQAPairsRequest struct {
	QAPairIDs []uuid.UUID `json:"qa_pair_ids" validate:"required,min=1,dive,required"`
}

type TemplateByQAPairItem struct {
	QAPairID   uuid.UUID  `json:"qa_pair_id"`
	TemplateID *uuid.UUID `json:"template_id"`
}

type FindTemplatesByQAPairsResponse struct {
	Items []TemplateByQAPairItem `json:"items"`
}
