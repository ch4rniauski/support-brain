package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ch4rniauski/support-brain/internal/embedding/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAlreadyProcessed = errors.New("event already processed")

type EmbeddingRepository struct {
	pool *pgxpool.Pool
}

func NewEmbeddingRepository(pool *pgxpool.Pool) *EmbeddingRepository {
	return &EmbeddingRepository{pool: pool}
}

func (r *EmbeddingRepository) ExistsInInbox(ctx context.Context, idempotencyKey uuid.UUID) (bool, error) {
	const query = `SELECT 1 FROM inbox_events WHERE idempotency_key = $1`

	var exists int
	err := r.pool.QueryRow(ctx, query, idempotencyKey).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check inbox event: %w", err)
	}

	return true, nil
}

func (r *EmbeddingRepository) SaveWithInbox(
	ctx context.Context,
	inbox *domain.InboxEvent,
	embedding *domain.QuestionEmbedding,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertInbox = `
		INSERT INTO inbox_events (idempotency_key, event_type, payload, processed_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.Exec(
		ctx,
		insertInbox,
		inbox.IdempotencyKey,
		inbox.EventType,
		inbox.Payload,
		inbox.ProcessedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyProcessed
		}
		return fmt.Errorf("insert inbox event: %w", err)
	}

	const insertEmbedding = `
		INSERT INTO question_embeddings (id, qa_pair_id, model_name, embedding)
		VALUES ($1, $2, $3, $4::vector)
	`
	_, err = tx.Exec(
		ctx,
		insertEmbedding,
		embedding.ID,
		embedding.QAPairID,
		embedding.ModelName,
		vectorLiteral(embedding.Embedding),
	)
	if err != nil {
		return fmt.Errorf("insert question embedding: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func vectorLiteral(values []float32) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprintf("%g", value)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
