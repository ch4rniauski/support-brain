-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

-- nomic-embed-text produces 768-dimensional vectors
CREATE TABLE question_embeddings (
    id         UUID PRIMARY KEY,
    qa_pair_id UUID NOT NULL,
    model_name TEXT NOT NULL,
    embedding  vector(768) NOT NULL,
    CONSTRAINT question_embeddings_qa_pair_id_model_name_key UNIQUE (qa_pair_id, model_name)
);

CREATE INDEX question_embeddings_qa_pair_id_idx ON question_embeddings (qa_pair_id);

CREATE TABLE outbox_events (
    id           UUID PRIMARY KEY,
    event_type   TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    payload      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ
);

CREATE INDEX outbox_events_unpublished_idx
    ON outbox_events (created_at)
    WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS question_embeddings;
