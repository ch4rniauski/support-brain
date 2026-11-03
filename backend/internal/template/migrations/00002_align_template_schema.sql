-- +goose Up
DROP TABLE IF EXISTS template_versions;

ALTER TABLE templates DROP COLUMN IF EXISTS approved_by;

CREATE UNIQUE INDEX IF NOT EXISTS templates_generation_job_id_key
    ON templates (generation_job_id)
    WHERE generation_job_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS templates_generation_job_id_key;

ALTER TABLE templates ADD COLUMN IF NOT EXISTS approved_by TEXT;

CREATE TABLE IF NOT EXISTS template_versions (
    id          UUID PRIMARY KEY,
    template_id UUID NOT NULL REFERENCES templates (id),
    version     INT NOT NULL,
    body        TEXT NOT NULL,
    editor_id   TEXT,
    CONSTRAINT template_versions_version_positive CHECK (version > 0),
    CONSTRAINT template_versions_template_id_version_key UNIQUE (template_id, version)
);
