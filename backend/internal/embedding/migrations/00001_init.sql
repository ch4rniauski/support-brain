-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE inbox_events (
    idempotency_key UUID PRIMARY KEY,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    processed_at    TIMESTAMPTZ NOT NULL
);

-- nomic-embed-text produces 768-dimensional vectors
CREATE TABLE question_embeddings (
    id         UUID PRIMARY KEY,
    qa_pair_id UUID NOT NULL,
    model_name TEXT NOT NULL,
    embedding  vector(768) NOT NULL,
    CONSTRAINT question_embeddings_qa_pair_id_model_name_key UNIQUE (qa_pair_id, model_name)
);

CREATE INDEX question_embeddings_qa_pair_id_idx ON question_embeddings (qa_pair_id);

-- +goose Down
DROP TABLE IF EXISTS question_embeddings;
DROP TABLE IF EXISTS inbox_events;
