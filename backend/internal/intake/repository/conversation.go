package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ch4rniauski/support-brain/internal/intake/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type ConversationRepository struct {
	pool *pgxpool.Pool
}

func NewConversationRepository(pool *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{pool: pool}
}

func (r *ConversationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error) {
	const query = `
		SELECT id, status, operator_id, started_at, closed_at
		FROM conversations
		WHERE id = $1
	`

	var conv domain.Conversation
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&conv.ID,
		&conv.Status,
		&conv.OperatorID,
		&conv.StartedAt,
		&conv.ClosedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	return &conv, nil
}

func (r *ConversationRepository) CreateWithMessage(
	ctx context.Context,
	conv *domain.Conversation,
	msg *domain.Message,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertConversation = `
		INSERT INTO conversations (id, status, operator_id, started_at, closed_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := tx.Exec(
		ctx,
		insertConversation,
		conv.ID,
		conv.Status,
		conv.OperatorID,
		conv.StartedAt,
		conv.ClosedAt,
	); err != nil {
		return fmt.Errorf("create conversation: %w", err)
	}

	const insertMessage = `
		INSERT INTO messages (id, conversation_id, sender_type, body, sent_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := tx.Exec(
		ctx,
		insertMessage,
		msg.ID,
		msg.ConversationID,
		msg.SenderType,
		msg.Body,
		msg.SentAt,
	); err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
