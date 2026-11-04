package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ch4rniauski/support-brain/internal/template/domain"
	"github.com/ch4rniauski/support-brain/internal/template/dto"
	"github.com/ch4rniauski/support-brain/internal/template/repository"
	"github.com/ch4rniauski/support-brain/pkg/kafka"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func permanent(err error) error {
	return fmt.Errorf("%w: %w", kafka.ErrPermanent, err)
}

var (
	ErrInvalidStatus    = errors.New("invalid template status")
	ErrInvalidInput     = errors.New("invalid input")
	ErrTemplateNotFound = errors.New("template not found")
	ErrNotDraft         = errors.New("template is not in draft status")
)

type templateStore interface {
	ExistsByGenerationJobID(ctx context.Context, generationJobID uuid.UUID) (bool, error)
	CreateDraft(ctx context.Context, template *domain.Template) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Template, error)
	UpdateDraftBody(ctx context.Context, id uuid.UUID, body string) (*domain.Template, error)
	Approve(ctx context.Context, id uuid.UUID, approvedAt time.Time) (*domain.Template, error)
	Reject(ctx context.Context, id uuid.UUID) (*domain.Template, error)
	ListByStatus(ctx context.Context, status domain.TemplateStatus) ([]domain.Template, error)
	FindApprovedByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Template, error)
}

type TemplateService struct {
	store     templateStore
	embedding embeddingClient
	knowledge knowledgeClient
	suggest   SuggestConfig
	validate  *validator.Validate
}

func NewTemplateService(
	store templateStore,
	embedding embeddingClient,
	knowledge knowledgeClient,
	suggest SuggestConfig,
) *TemplateService {
	if suggest.SimilarityThreshold <= 0 {
		suggest.SimilarityThreshold = 0.85
	}
	if suggest.SearchLimit <= 0 {
		suggest.SearchLimit = 10
	}

	return &TemplateService{
		store:     store,
		embedding: embedding,
		knowledge: knowledge,
		suggest:   suggest,
		validate:  validator.New(),
	}
}

func (s *TemplateService) GetByID(ctx context.Context, id uuid.UUID) (*dto.TemplateItem, error) {
	template, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}

	item := toTemplateItem(*template)
	return &item, nil
}

func (s *TemplateService) UpdateDraft(
	ctx context.Context,
	id uuid.UUID,
	req dto.UpdateTemplateRequest,
) (*dto.TemplateItem, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return nil, ErrInvalidInput
	}

	template, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}

	if template.Status != domain.TemplateStatusDraft {
		return nil, ErrNotDraft
	}

	updated, err := s.store.UpdateDraftBody(ctx, id, body)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotDraft
		}
		return nil, err
	}

	item := toTemplateItem(*updated)
	return &item, nil
}

func (s *TemplateService) Approve(ctx context.Context, id uuid.UUID) (*dto.TemplateItem, error) {
	approved, err := s.store.Approve(ctx, id, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return nil, ErrTemplateNotFound
		case errors.Is(err, repository.ErrNotDraft):
			return nil, ErrNotDraft
		default:
			return nil, err
		}
	}

	item := toTemplateItem(*approved)
	return &item, nil
}

func (s *TemplateService) Reject(ctx context.Context, id uuid.UUID) (*dto.TemplateItem, error) {
	template, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}

	if template.Status != domain.TemplateStatusDraft {
		return nil, ErrNotDraft
	}

	rejected, err := s.store.Reject(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotDraft
		}
		return nil, err
	}

	item := toTemplateItem(*rejected)
	return &item, nil
}

func (s *TemplateService) ListByStatus(
	ctx context.Context,
	status domain.TemplateStatus,
) (*dto.ListTemplatesResponse, error) {
	if !isValidStatus(status) {
		return nil, ErrInvalidStatus
	}

	templates, err := s.store.ListByStatus(ctx, status)
	if err != nil {
		return nil, err
	}

	items := make([]dto.TemplateItem, 0, len(templates))
	for _, template := range templates {
		items = append(items, toTemplateItem(template))
	}

	return &dto.ListTemplatesResponse{Items: items}, nil
}

func toTemplateItem(template domain.Template) dto.TemplateItem {
	return dto.TemplateItem{
		ID:              template.ID,
		ClusterID:       template.ClusterID,
		Status:          string(template.Status),
		Version:         template.Version,
		Body:            template.Body,
		GenerationJobID: template.GenerationJobID,
		UsedCount:       template.UsedCount,
		ApprovedAt:      template.ApprovedAt,
	}
}

func isValidStatus(status domain.TemplateStatus) bool {
	switch status {
	case domain.TemplateStatusDraft,
		domain.TemplateStatusApproved,
		domain.TemplateStatusRejected,
		domain.TemplateStatusArchived:
		return true
	default:
		return false
	}
}

func (s *TemplateService) Handle(ctx context.Context, eventType string, payload []byte) error {
	if eventType != "" && eventType != dto.EventTypeTemplateDraftCreated {
		log.Printf("template: skip unsupported event type %q", eventType)
		return nil
	}

	var event dto.TemplateDraftCreatedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return permanent(fmt.Errorf("unmarshal template draft created event: %w", err))
	}

	if event.ClusterID == uuid.Nil {
		return permanent(fmt.Errorf("cluster_id is required"))
	}
	if event.GenerationJobID == uuid.Nil {
		return permanent(fmt.Errorf("generation_job_id is required"))
	}

	text := strings.TrimSpace(event.Text)
	if text == "" {
		return permanent(fmt.Errorf("text is required"))
	}

	exists, err := s.store.ExistsByGenerationJobID(ctx, event.GenerationJobID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	generationJobID := event.GenerationJobID
	template := &domain.Template{
		ID:              uuid.New(),
		ClusterID:       event.ClusterID,
		Status:          domain.TemplateStatusDraft,
		Version:         1,
		Body:            text,
		GenerationJobID: &generationJobID,
		UsedCount:       0,
		ApprovedAt:      nil,
	}

	if err := s.store.CreateDraft(ctx, template); err != nil {
		if errors.Is(err, repository.ErrAlreadyImported) {
			return nil
		}
		return err
	}

	return nil
}
