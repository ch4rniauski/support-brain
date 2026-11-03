package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/ch4rniauski/support-brain/internal/template/domain"
	"github.com/ch4rniauski/support-brain/internal/template/dto"
	"github.com/ch4rniauski/support-brain/internal/template/repository"
	"github.com/ch4rniauski/support-brain/pkg/kafka"
	"github.com/google/uuid"
)

func permanent(err error) error {
	return fmt.Errorf("%w: %w", kafka.ErrPermanent, err)
}

type templateStore interface {
	ExistsByGenerationJobID(ctx context.Context, generationJobID uuid.UUID) (bool, error)
	CreateDraft(ctx context.Context, template *domain.Template) error
}

type TemplateService struct {
	store templateStore
}

func NewTemplateService(store templateStore) *TemplateService {
	return &TemplateService{store: store}
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
