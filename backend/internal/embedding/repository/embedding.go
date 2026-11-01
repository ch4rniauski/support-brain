package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ch4rniauski/support-brain/internal/embedding/domain"
	"github.com/ch4rniauski/support-brain/pkg/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadyProcessed  = errors.New("event already processed")
	ErrEmbeddingNotFound = errors.New("embedding not found")
)

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

func (r *EmbeddingRepository) FindSimilar(
	ctx context.Context,
	qaPairID uuid.UUID,
	modelName string,
	threshold float64,
	limit int,
) ([]domain.SimilarQuestion, error) {
	const existsQuery = `
		SELECT 1
		FROM question_embeddings
		WHERE qa_pair_id = $1 AND model_name = $2
	`
	var exists int
	err := r.pool.QueryRow(ctx, existsQuery, qaPairID, modelName).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEmbeddingNotFound
		}
		return nil, fmt.Errorf("check source embedding: %w", err)
	}

	const query = `
		WITH query AS (
			SELECT embedding
			FROM question_embeddings
			WHERE qa_pair_id = $1 AND model_name = $2
		)
		SELECT
			qe.qa_pair_id,
			1 - (qe.embedding <=> query.embedding) AS score
		FROM question_embeddings qe
		CROSS JOIN query
		WHERE qe.model_name = $2
		  AND qe.qa_pair_id <> $1
		  AND 1 - (qe.embedding <=> query.embedding) >= $3
		ORDER BY qe.embedding <=> query.embedding
		LIMIT $4
	`

	rows, err := r.pool.Query(ctx, query, qaPairID, modelName, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("find similar embeddings: %w", err)
	}
	defer rows.Close()

	results := make([]domain.SimilarQuestion, 0, limit)
	for rows.Next() {
		var item domain.SimilarQuestion
		if err := rows.Scan(&item.QAPairID, &item.Score); err != nil {
			return nil, fmt.Errorf("scan similar embedding: %w", err)
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate similar embeddings: %w", err)
	}

	return results, nil
}

func (r *EmbeddingRepository) SaveWithInboxAndOutbox(
	ctx context.Context,
	inbox *domain.InboxEvent,
	embedding *domain.QuestionEmbedding,
	event *outbox.Event,
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

	const insertOutbox = `
		INSERT INTO outbox_events (id, event_type, aggregate_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.Exec(
		ctx,
		insertOutbox,
		event.ID,
		event.EventType,
		event.AggregateID,
		event.Payload,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
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
