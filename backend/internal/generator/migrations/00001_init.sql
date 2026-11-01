-- +goose Up
CREATE TABLE generation_jobs (
    id             UUID PRIMARY KEY,
    cluster_id     UUID NOT NULL,
    status         TEXT NOT NULL,
    model_name     TEXT NOT NULL,
    prompt_version TEXT NOT NULL,
    output_text    TEXT,
    error_message  TEXT,
    attempts       INT NOT NULL DEFAULT 0,
    started_at     TIMESTAMPTZ,
    finished_at    TIMESTAMPTZ,
    CONSTRAINT generation_jobs_status_check CHECK (
        status IN ('pending', 'running', 'succeeded', 'failed')
    ),
    CONSTRAINT generation_jobs_attempts_non_negative CHECK (attempts >= 0)
);

CREATE INDEX generation_jobs_cluster_id_idx ON generation_jobs (cluster_id);
CREATE INDEX generation_jobs_status_idx ON generation_jobs (status);

CREATE TABLE generation_job_examples (
    id         UUID PRIMARY KEY,
    job_id     UUID NOT NULL REFERENCES generation_jobs (id),
    qa_pair_id UUID NOT NULL,
    question   TEXT NOT NULL,
    answer     TEXT NOT NULL
);

CREATE INDEX generation_job_examples_job_id_idx ON generation_job_examples (job_id);

-- +goose Down
DROP TABLE IF EXISTS generation_job_examples;
DROP TABLE IF EXISTS generation_jobs;
