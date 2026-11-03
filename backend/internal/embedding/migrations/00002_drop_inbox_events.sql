-- +goose Up
DROP TABLE IF EXISTS inbox_events;

-- For DBs that already applied the old 00001 (inbox only, no outbox).
CREATE TABLE IF NOT EXISTS outbox_events (
    id           UUID PRIMARY KEY,
    event_type   TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    payload      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS outbox_events_unpublished_idx
    ON outbox_events (created_at)
    WHERE published_at IS NULL;

-- +goose Down
CREATE TABLE IF NOT EXISTS inbox_events (
    idempotency_key UUID PRIMARY KEY,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    processed_at    TIMESTAMPTZ NOT NULL
);
