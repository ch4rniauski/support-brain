package repository

import (
	"context"
	"fmt"

	"github.com/ch4rniauski/support-brain/internal/intake/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepository struct {
	pool *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

func (r *MessageRepository) Create(ctx context.Context, msg *domain.Message) error {
	const query = `
		INSERT INTO messages (id, conversation_id, sender_type, body, sent_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		msg.ID,
		msg.ConversationID,
		msg.SenderType,
		msg.Body,
		msg.SentAt,
	)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	return nil
}
