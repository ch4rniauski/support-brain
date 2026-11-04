package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ch4rniauski/support-brain/internal/knowledge/dto"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var ErrInvalidInput = errors.New("invalid input")

type clusterTemplateStore interface {
	FindApprovedTemplateIDsByQAPairIDs(ctx context.Context, qaPairIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
}

type ClusterService struct {
	store    clusterTemplateStore
	validate *validator.Validate
}

func NewClusterService(store clusterTemplateStore) *ClusterService {
	return &ClusterService{
		store:    store,
		validate: validator.New(),
	}
}

func (s *ClusterService) FindTemplatesByQAPairIDs(
	ctx context.Context,
	req dto.FindTemplatesByQAPairsRequest,
) (*dto.FindTemplatesByQAPairsResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	qaPairIDs := dedupeUUIDs(req.QAPairIDs)
	templatesByQAPair, err := s.store.FindApprovedTemplateIDsByQAPairIDs(ctx, qaPairIDs)
	if err != nil {
		return nil, err
	}

	items := make([]dto.TemplateByQAPairItem, 0, len(qaPairIDs))
	for _, qaPairID := range qaPairIDs {
		item := dto.TemplateByQAPairItem{QAPairID: qaPairID}
		if templateID, ok := templatesByQAPair[qaPairID]; ok {
			item.TemplateID = &templateID
		}
		items = append(items, item)
	}

	return &dto.FindTemplatesByQAPairsResponse{Items: items}, nil
}

func dedupeUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
