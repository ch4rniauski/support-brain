-- +goose Up
CREATE TABLE conversations (
    id          UUID PRIMARY KEY,
    status      TEXT NOT NULL,
    operator_id TEXT,
    started_at  TIMESTAMPTZ NOT NULL,
    closed_at   TIMESTAMPTZ,
    CONSTRAINT conversations_status_check CHECK (status IN ('open', 'closed'))
);

CREATE TABLE messages (
    id              UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations (id),
    sender_type     TEXT NOT NULL,
    body            TEXT NOT NULL,
    sent_at         TIMESTAMPTZ NOT NULL,
    CONSTRAINT messages_sender_type_check CHECK (sender_type IN ('client', 'operator'))
);

CREATE INDEX messages_conversation_id_idx ON messages (conversation_id);

CREATE TABLE qa_pairs (
    id              UUID PRIMARY KEY,
    conversation_id UUID REFERENCES conversations (id),
    question        TEXT NOT NULL,
    answer          TEXT NOT NULL,
    answer_source   TEXT NOT NULL,
    template_id     UUID,
    operator_id     TEXT,
    CONSTRAINT qa_pairs_answer_source_check CHECK (
        answer_source IN ('manual', 'template_as_is', 'template_edited')
    )
);

CREATE INDEX qa_pairs_conversation_id_idx ON qa_pairs (conversation_id);

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
DROP TABLE IF EXISTS qa_pairs;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
