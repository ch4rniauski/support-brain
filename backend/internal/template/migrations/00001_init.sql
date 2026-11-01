-- +goose Up
CREATE TABLE templates (
    id                UUID PRIMARY KEY,
    cluster_id        UUID NOT NULL,
    status            TEXT NOT NULL,
    version           INT NOT NULL,
    body              TEXT NOT NULL,
    generation_job_id UUID,
    used_count        INT NOT NULL DEFAULT 0,
    approved_at       TIMESTAMPTZ,
    approved_by       TEXT,
    CONSTRAINT templates_status_check CHECK (
        status IN ('draft', 'approved', 'rejected', 'archived')
    ),
    CONSTRAINT templates_version_positive CHECK (version > 0),
    CONSTRAINT templates_used_count_non_negative CHECK (used_count >= 0)
);

CREATE INDEX templates_cluster_id_idx ON templates (cluster_id);
CREATE INDEX templates_status_idx ON templates (status);

CREATE TABLE template_versions (
    id          UUID PRIMARY KEY,
    template_id UUID NOT NULL REFERENCES templates (id),
    version     INT NOT NULL,
    body        TEXT NOT NULL,
    editor_id   TEXT,
    CONSTRAINT template_versions_version_positive CHECK (version > 0),
    CONSTRAINT template_versions_template_id_version_key UNIQUE (template_id, version)
);

CREATE TABLE template_usages (
    id              UUID PRIMARY KEY,
    template_id     UUID NOT NULL REFERENCES templates (id),
    conversation_id UUID,
    qa_pair_id      UUID,
    used_as         TEXT NOT NULL,
    suggested_body  TEXT NOT NULL,
    final_body      TEXT NOT NULL,
    score           DOUBLE PRECISION,
    operator_id     TEXT,
    created_at      TIMESTAMPTZ NOT NULL,
    CONSTRAINT template_usages_used_as_check CHECK (used_as IN ('as_is', 'edited'))
);

CREATE INDEX template_usages_template_id_idx ON template_usages (template_id);

-- +goose Down
DROP TABLE IF EXISTS template_usages;
DROP TABLE IF EXISTS template_versions;
DROP TABLE IF EXISTS templates;
