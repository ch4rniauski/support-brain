package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ch4rniauski/support-brain/internal/embedding/domain"
	"github.com/ch4rniauski/support-brain/internal/embedding/dto"
	"github.com/ch4rniauski/support-brain/internal/embedding/repository"
	"github.com/ch4rniauski/support-brain/pkg/kafka"
	"github.com/ch4rniauski/support-brain/pkg/outbox"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrEmbeddingNotFound = errors.New("embedding not found")
)

func permanent(err error) error {
	return fmt.Errorf("%w: %w", kafka.ErrPermanent, err)
}

type embedder interface {
	Model() string
	Embed(ctx context.Context, text string) ([]float32, error)
}

type embeddingStore interface {
	ExistsInInbox(ctx context.Context, idempotencyKey uuid.UUID) (bool, error)
	SaveWithInboxAndOutbox(
		ctx context.Context,
		inbox *domain.InboxEvent,
		embedding *domain.QuestionEmbedding,
		event *outbox.Event,
	) error
	FindSimilar(
		ctx context.Context,
		qaPairID uuid.UUID,
		modelName string,
		threshold float64,
		limit int,
	) ([]domain.SimilarQuestion, error)
}

type EmbeddingService struct {
	store    embeddingStore
	embedder embedder
	validate *validator.Validate
}

func NewEmbeddingService(store embeddingStore, embedder embedder) *EmbeddingService {
	return &EmbeddingService{
		store:    store,
		embedder: embedder,
		validate: validator.New(),
	}
}

func (s *EmbeddingService) FindSimilar(
	ctx context.Context,
	req dto.FindSimilarRequest,
) (*dto.FindSimilarResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	matches, err := s.store.FindSimilar(ctx, req.QAPairID, req.ModelName, req.Threshold, req.Limit)
	if err != nil {
		if errors.Is(err, repository.ErrEmbeddingNotFound) {
			return nil, ErrEmbeddingNotFound
		}
		return nil, err
	}

	results := make([]dto.SimilarResult, 0, len(matches))
	for _, match := range matches {
		results = append(results, dto.SimilarResult{
			QAPairID: match.QAPairID,
			Score:    match.Score,
		})
	}

	return &dto.FindSimilarResponse{
		QAPairID: req.QAPairID,
		Results:  results,
	}, nil
}

func (s *EmbeddingService) Handle(ctx context.Context, eventType string, payload []byte) error {
	if eventType != "" && eventType != dto.EventTypeQAPairCreated {
		log.Printf("embedding: skip unsupported event type %q", eventType)
		return nil
	}

	var event dto.QAPairCreatedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return permanent(fmt.Errorf("unmarshal qa pair created event: %w", err))
	}

	if event.QAPairID == uuid.Nil {
		return permanent(fmt.Errorf("qa_pair_id is required"))
	}
	if event.Question == "" {
		return permanent(fmt.Errorf("question is required"))
	}

	exists, err := s.store.ExistsInInbox(ctx, event.QAPairID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	vector, err := s.embedder.Embed(ctx, event.Question)
	if err != nil {
		return fmt.Errorf("embed question: %w", err)
	}

	outboxPayload, err := json.Marshal(dto.EmbeddingReadyEvent{
		QAPairID: event.QAPairID,
	})
	if err != nil {
		return fmt.Errorf("marshal embedding ready event: %w", err)
	}

	now := time.Now().UTC()
	inbox := &domain.InboxEvent{
		IdempotencyKey: event.QAPairID,
		EventType:      dto.EventTypeQAPairCreated,
		Payload:        payload,
		ProcessedAt:    now,
	}
	embedding := &domain.QuestionEmbedding{
		ID:        uuid.New(),
		QAPairID:  event.QAPairID,
		ModelName: s.embedder.Model(),
		Embedding: vector,
	}
	outboxEvent := &outbox.Event{
		ID:          uuid.New(),
		EventType:   domain.EventTypeEmbeddingReady,
		AggregateID: event.QAPairID,
		Payload:     outboxPayload,
		CreatedAt:   now,
	}

	if err := s.store.SaveWithInboxAndOutbox(ctx, inbox, embedding, outboxEvent); err != nil {
		if errors.Is(err, repository.ErrAlreadyProcessed) {
			return nil
		}
		return err
	}

	return nil
}
