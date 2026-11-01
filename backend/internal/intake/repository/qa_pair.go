package repository

import (
	"context"
	"fmt"

	"github.com/ch4rniauski/support-brain/internal/intake/domain"
	"github.com/ch4rniauski/support-brain/pkg/outbox"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QAPairRepository struct {
	pool *pgxpool.Pool
}

func NewQAPairRepository(pool *pgxpool.Pool) *QAPairRepository {
	return &QAPairRepository{pool: pool}
}

func (r *QAPairRepository) CreateWithMessageAndOutbox(
	ctx context.Context,
	msg *domain.Message,
	pair *domain.QAPair,
	event *outbox.Event,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

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

	const insertQAPair = `
		INSERT INTO qa_pairs (
			id, question, answer, answer_source, conversation_id, template_id, operator_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	if _, err := tx.Exec(
		ctx,
		insertQAPair,
		pair.ID,
		pair.Question,
		pair.Answer,
		pair.AnswerSource,
		pair.ConversationID,
		pair.TemplateID,
		pair.OperatorID,
	); err != nil {
		return fmt.Errorf("create qa_pair: %w", err)
	}

	const insertOutbox = `
		INSERT INTO outbox_events (id, event_type, aggregate_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := tx.Exec(
		ctx,
		insertOutbox,
		event.ID,
		event.EventType,
		event.AggregateID,
		event.Payload,
		event.CreatedAt,
	); err != nil {
		return fmt.Errorf("create outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
