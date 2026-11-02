package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ch4rniauski/support-brain/internal/intake/domain"
	"github.com/ch4rniauski/support-brain/internal/intake/dto"
	"github.com/ch4rniauski/support-brain/internal/intake/repository"
	"github.com/ch4rniauski/support-brain/pkg/outbox"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	ErrInvalidInput         = errors.New("invalid input")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrConversationClosed   = errors.New("conversation is closed")
)

type conversationStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error)
	CreateWithMessage(ctx context.Context, conv *domain.Conversation, msg *domain.Message) error
}

type messageStore interface {
	Create(ctx context.Context, msg *domain.Message) error
}

type qaPairStore interface {
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.QAPair, error)
	CreateWithMessageAndOutbox(
		ctx context.Context,
		msg *domain.Message,
		pair *domain.QAPair,
		event *outbox.Event,
	) error
}

type MessageService struct {
	conversations conversationStore
	messages      messageStore
	qaPairs       qaPairStore
	validate      *validator.Validate
}

func NewMessageService(
	conversations *repository.ConversationRepository,
	messages *repository.MessageRepository,
	qaPairs *repository.QAPairRepository,
) *MessageService {
	return &MessageService{
		conversations: conversations,
		messages:      messages,
		qaPairs:       qaPairs,
		validate:      validator.New(),
	}
}

func (s *MessageService) SendMessage(ctx context.Context, req dto.SendMessageRequest) (*dto.SendMessageResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	now := time.Now().UTC()
	msg := &domain.Message{
		ID:         uuid.New(),
		SenderType: domain.SenderTypeClient,
		Body:       req.Body,
		SentAt:     now,
	}

	if req.ConversationID == nil {
		conv := &domain.Conversation{
			ID:        uuid.New(),
			Status:    domain.ConversationStatusOpen,
			StartedAt: now,
		}
		msg.ConversationID = conv.ID

		if err := s.conversations.CreateWithMessage(ctx, conv, msg); err != nil {
			return nil, fmt.Errorf("create conversation with message: %w", err)
		}

		return &dto.SendMessageResponse{ConversationID: conv.ID}, nil
	}

	return s.appendToConversation(ctx, *req.ConversationID, msg)
}

func (s *MessageService) ListQAPairs(ctx context.Context, ids []uuid.UUID) (*dto.ListQAPairsResponse, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: ids are required", ErrInvalidInput)
	}

	pairs, err := s.qaPairs.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	items := make([]dto.QAPairItem, 0, len(pairs))
	for _, pair := range pairs {
		items = append(items, dto.QAPairItem{
			ID:       pair.ID,
			Question: pair.Question,
			Answer:   pair.Answer,
		})
	}

	return &dto.ListQAPairsResponse{Items: items}, nil
}

func (s *MessageService) ReplyMessage(ctx context.Context, req dto.ReplyMessageRequest) (*dto.SendMessageResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	conv, err := s.conversations.GetByID(ctx, req.ConversationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	if conv.Status == domain.ConversationStatusClosed {
		return nil, ErrConversationClosed
	}

	now := time.Now().UTC()
	msg := &domain.Message{
		ID:             uuid.New(),
		ConversationID: conv.ID,
		SenderType:     domain.SenderTypeOperator,
		Body:           req.Body,
		SentAt:         now,
	}

	conversationID := conv.ID
	pair := &domain.QAPair{
		ID:             uuid.New(),
		ConversationID: &conversationID,
		Question:       req.Question,
		Answer:         req.Body,
		AnswerSource:   domain.AnswerSource(req.AnswerSource),
		TemplateID:     req.TemplateID,
	}

	payload, err := json.Marshal(dto.QAPairCreatedEvent{
		QAPairID: pair.ID,
		Question: pair.Question,
		Answer:   pair.Answer,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal qa pair created event: %w", err)
	}

	event := &outbox.Event{
		ID:          uuid.New(),
		EventType:   domain.EventTypeQAPairCreated,
		AggregateID: pair.ID,
		Payload:     payload,
		CreatedAt:   now,
	}

	if err := s.qaPairs.CreateWithMessageAndOutbox(ctx, msg, pair, event); err != nil {
		return nil, fmt.Errorf("create reply with qa pair: %w", err)
	}

	return &dto.SendMessageResponse{ConversationID: conv.ID}, nil
}

func (s *MessageService) appendToConversation(
	ctx context.Context,
	conversationID uuid.UUID,
	msg *domain.Message,
) (*dto.SendMessageResponse, error) {
	conv, err := s.conversations.GetByID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	if conv.Status == domain.ConversationStatusClosed {
		return nil, ErrConversationClosed
	}

	msg.ConversationID = conv.ID
	if err := s.messages.Create(ctx, msg); err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	return &dto.SendMessageResponse{ConversationID: conv.ID}, nil
}
