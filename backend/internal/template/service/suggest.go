package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ch4rniauski/support-brain/internal/template/dto"
	"github.com/google/uuid"
)

type embeddingClient interface {
	FindSimilarByText(
		ctx context.Context,
		text string,
		threshold float64,
		limit int,
	) (*dto.FindSimilarByTextResponse, error)
}

type knowledgeClient interface {
	FindTemplatesByQAPairs(
		ctx context.Context,
		qaPairIDs []uuid.UUID,
	) (*dto.FindTemplatesByQAPairsResponse, error)
}

type SuggestConfig struct {
	SimilarityThreshold float64
	SearchLimit         int
}

func (s *TemplateService) Suggest(
	ctx context.Context,
	req dto.SuggestTemplatesRequest,
) (*dto.SuggestTemplatesResponse, error) {
	req.Question = strings.TrimSpace(req.Question)
	if err := s.validate.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	empty := &dto.SuggestTemplatesResponse{Templates: []dto.SuggestedTemplate{}}

	similar, err := s.embedding.FindSimilarByText(
		ctx,
		req.Question,
		s.suggest.SimilarityThreshold,
		s.suggest.SearchLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("find similar by text: %w", err)
	}
	if similar == nil || len(similar.Results) == 0 {
		return empty, nil
	}

	qaPairIDs := make([]uuid.UUID, 0, len(similar.Results))
	scoreByQAPair := make(map[uuid.UUID]float64, len(similar.Results))
	for _, result := range similar.Results {
		qaPairIDs = append(qaPairIDs, result.QAPairID)
		scoreByQAPair[result.QAPairID] = result.Score
	}

	mapped, err := s.knowledge.FindTemplatesByQAPairs(ctx, qaPairIDs)
	if err != nil {
		return nil, fmt.Errorf("find templates by qa pairs: %w", err)
	}
	if mapped == nil || len(mapped.Items) == 0 {
		return empty, nil
	}

	templateIDs, scoreByTemplate := collectTemplateScores(mapped.Items, scoreByQAPair)
	if len(templateIDs) == 0 {
		return empty, nil
	}

	templates, err := s.store.FindApprovedByIDs(ctx, templateIDs)
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return empty, nil
	}

	suggested := make([]dto.SuggestedTemplate, 0, len(templates))
	for _, template := range templates {
		suggested = append(suggested, dto.SuggestedTemplate{
			ID:        template.ID,
			Body:      template.Body,
			ClusterID: template.ClusterID,
			Score:     scoreByTemplate[template.ID],
		})
	}

	sort.SliceStable(suggested, func(i, j int) bool {
		return suggested[i].Score > suggested[j].Score
	})

	return &dto.SuggestTemplatesResponse{Templates: suggested}, nil
}

func collectTemplateScores(
	items []dto.TemplateByQAPairItem,
	scoreByQAPair map[uuid.UUID]float64,
) ([]uuid.UUID, map[uuid.UUID]float64) {
	scoreByTemplate := make(map[uuid.UUID]float64)
	templateIDs := make([]uuid.UUID, 0)
	seenTemplate := make(map[uuid.UUID]struct{})

	for _, item := range items {
		if item.TemplateID == nil {
			continue
		}

		// A user question can match multiple Q/A pairs in the same cluster (same cluster => same template),
		// e.g. 0.85 and 0.90 — keep the best score (0.90) for that template
		templateID := *item.TemplateID
		score := scoreByQAPair[item.QAPairID]
		if current, ok := scoreByTemplate[templateID]; !ok || score > current {
			scoreByTemplate[templateID] = score
		}
		if _, seen := seenTemplate[templateID]; seen {
			continue
		}
		seenTemplate[templateID] = struct{}{}
		templateIDs = append(templateIDs, templateID)
	}

	return templateIDs, scoreByTemplate
}
