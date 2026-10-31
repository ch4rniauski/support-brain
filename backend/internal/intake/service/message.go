package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ch4rniauski/support-brain/internal/intake/domain"
	"github.com/ch4rniauski/support-brain/internal/intake/dto"
	"github.com/ch4rniauski/support-brain/internal/intake/repository"
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

type MessageService struct {
	conversations conversationStore
	messages      messageStore
	validate      *validator.Validate
}

func NewMessageService(
	conversations *repository.ConversationRepository,
	messages *repository.MessageRepository,
) *MessageService {
	return &MessageService{
		conversations: conversations,
		messages:      messages,
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

func (s *MessageService) ReplyMessage(ctx context.Context, req dto.ReplyMessageRequest) (*dto.SendMessageResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	msg := &domain.Message{
		ID:         uuid.New(),
		SenderType: domain.SenderTypeOperator,
		Body:       req.Body,
		SentAt:     time.Now().UTC(),
	}

	return s.appendToConversation(ctx, req.ConversationID, msg)
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
