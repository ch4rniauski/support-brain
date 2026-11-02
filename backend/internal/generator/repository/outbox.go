package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ch4rniauski/support-brain/internal/generator/domain"
	"github.com/ch4rniauski/support-brain/pkg/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) ListUnpublished(ctx context.Context, limit int) ([]outbox.Event, error) {
	const query = `
		SELECT id, event_type, aggregate_id, payload, created_at, published_at
		FROM outbox_events
		WHERE published_at IS NULL
		  AND event_type = $1
		ORDER BY created_at
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, domain.EventTypeTemplateDraftCreated, limit)
	if err != nil {
		return nil, fmt.Errorf("list unpublished outbox events: %w", err)
	}
	defer rows.Close()

	events := make([]outbox.Event, 0, limit)
	for rows.Next() {
		var event outbox.Event
		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.AggregateID,
			&event.Payload,
			&event.CreatedAt,
			&event.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox events: %w", err)
	}

	return events, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, id uuid.UUID, publishedAt time.Time) error {
	const query = `
		UPDATE outbox_events
		SET published_at = $2
		WHERE id = $1 AND published_at IS NULL
	`

	if _, err := r.pool.Exec(ctx, query, id, publishedAt); err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}

	return nil
}
