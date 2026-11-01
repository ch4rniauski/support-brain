-- +goose Up
CREATE TABLE clusters (
    id                      UUID PRIMARY KEY,
    title                   TEXT,
    status                  TEXT NOT NULL,
    member_count            INT NOT NULL DEFAULT 0,
    similarity_threshold    DOUBLE PRECISION NOT NULL,
    generation_threshold_n  INT NOT NULL,
    active_template_id      UUID,
    CONSTRAINT clusters_status_check CHECK (
        status IN ('collecting', 'ready_to_generate', 'has_draft', 'has_approved')
    ),
    CONSTRAINT clusters_member_count_non_negative CHECK (member_count >= 0),
    CONSTRAINT clusters_generation_threshold_positive CHECK (generation_threshold_n > 0)
);

CREATE TABLE cluster_members (
    id               UUID PRIMARY KEY,
    cluster_id       UUID NOT NULL REFERENCES clusters (id),
    qa_pair_id       UUID NOT NULL,
    similarity_score DOUBLE PRECISION,
    CONSTRAINT cluster_members_cluster_id_qa_pair_id_key UNIQUE (cluster_id, qa_pair_id)
);

CREATE INDEX cluster_members_cluster_id_idx ON cluster_members (cluster_id);
CREATE INDEX cluster_members_qa_pair_id_idx ON cluster_members (qa_pair_id);

CREATE TABLE disputed_qa_pairs (
    id                    UUID PRIMARY KEY,
    qa_pair_id            UUID NOT NULL,
    status                TEXT NOT NULL,
    resolution_cluster_id UUID REFERENCES clusters (id),
    resolved_at           TIMESTAMPTZ,
    CONSTRAINT disputed_qa_pairs_status_check CHECK (status IN ('pending', 'resolved'))
);

CREATE INDEX disputed_qa_pairs_status_idx ON disputed_qa_pairs (status);
CREATE INDEX disputed_qa_pairs_qa_pair_id_idx ON disputed_qa_pairs (qa_pair_id);

CREATE TABLE disputed_candidates (
    id                 UUID PRIMARY KEY,
    disputed_qa_pair_id UUID NOT NULL REFERENCES disputed_qa_pairs (id),
    cluster_id         UUID NOT NULL REFERENCES clusters (id),
    neighbor_qa_pair_id UUID NOT NULL,
    score              DOUBLE PRECISION NOT NULL
);

CREATE INDEX disputed_candidates_disputed_qa_pair_id_idx
    ON disputed_candidates (disputed_qa_pair_id);

-- +goose Down
DROP TABLE IF EXISTS disputed_candidates;
DROP TABLE IF EXISTS disputed_qa_pairs;
DROP TABLE IF EXISTS cluster_members;
DROP TABLE IF EXISTS clusters;
