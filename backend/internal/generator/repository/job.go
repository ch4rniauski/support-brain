package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ch4rniauski/support-brain/internal/generator/domain"
	"github.com/ch4rniauski/support-brain/pkg/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrJobNotFound = errors.New("generation job not found")

type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

func (r *JobRepository) FindActiveByClusterID(
	ctx context.Context,
	clusterID uuid.UUID,
) (*domain.GenerationJob, error) {
	const query = `
		SELECT id, cluster_id, status, model_name, prompt_version,
		       output_text, error_message, attempts, started_at, finished_at
		FROM generation_jobs
		WHERE cluster_id = $1
		  AND status IN ($2, $3)
		ORDER BY attempts DESC
		LIMIT 1
	`

	var job domain.GenerationJob
	err := r.pool.QueryRow(
		ctx,
		query,
		clusterID,
		domain.GenerationJobStatusPending,
		domain.GenerationJobStatusRunning,
	).Scan(
		&job.ID,
		&job.ClusterID,
		&job.Status,
		&job.ModelName,
		&job.PromptVersion,
		&job.OutputText,
		&job.ErrorMessage,
		&job.Attempts,
		&job.StartedAt,
		&job.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find active generation job: %w", err)
	}

	return &job, nil
}

func (r *JobRepository) HasSucceededByClusterID(ctx context.Context, clusterID uuid.UUID) (bool, error) {
	const query = `
		SELECT 1
		FROM generation_jobs
		WHERE cluster_id = $1 AND status = $2
		LIMIT 1
	`

	var exists int
	err := r.pool.QueryRow(ctx, query, clusterID, domain.GenerationJobStatusSucceeded).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check succeeded generation job: %w", err)
	}

	return true, nil
}

func (r *JobRepository) Create(ctx context.Context, job *domain.GenerationJob) error {
	const query = `
		INSERT INTO generation_jobs (
			id, cluster_id, status, model_name, prompt_version,
			output_text, error_message, attempts, started_at, finished_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		job.ID,
		job.ClusterID,
		job.Status,
		job.ModelName,
		job.PromptVersion,
		job.OutputText,
		job.ErrorMessage,
		job.Attempts,
		job.StartedAt,
		job.FinishedAt,
	)
	if err != nil {
		return fmt.Errorf("create generation job: %w", err)
	}

	return nil
}

func (r *JobRepository) SaveExamples(ctx context.Context, examples []domain.GenerationJobExample) error {
	if len(examples) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const query = `
		INSERT INTO generation_job_examples (id, job_id, qa_pair_id, question, answer)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, example := range examples {
		if _, err := tx.Exec(
			ctx,
			query,
			example.ID,
			example.JobID,
			example.QAPairID,
			example.Question,
			example.Answer,
		); err != nil {
			return fmt.Errorf("insert generation job example: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *JobRepository) MarkRunning(ctx context.Context, jobID uuid.UUID, startedAt time.Time) error {
	const query = `
		UPDATE generation_jobs
		SET status = $2,
		    attempts = attempts + 1,
		    started_at = $3,
		    error_message = NULL
		WHERE id = $1
	`

	tag, err := r.pool.Exec(
		ctx,
		query,
		jobID,
		domain.GenerationJobStatusRunning,
		startedAt,
	)
	if err != nil {
		return fmt.Errorf("mark generation job running: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}

	return nil
}

func (r *JobRepository) MarkPendingWithError(ctx context.Context, jobID uuid.UUID, errorMessage string) error {
	const query = `
		UPDATE generation_jobs
		SET status = $2,
		    error_message = $3,
		    finished_at = NULL
		WHERE id = $1
	`

	tag, err := r.pool.Exec(
		ctx,
		query,
		jobID,
		domain.GenerationJobStatusPending,
		errorMessage,
	)
	if err != nil {
		return fmt.Errorf("mark generation job pending with error: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}

	return nil
}

func (r *JobRepository) MarkFailed(
	ctx context.Context,
	jobID uuid.UUID,
	errorMessage string,
	finishedAt time.Time,
) error {
	const query = `
		UPDATE generation_jobs
		SET status = $2,
		    error_message = $3,
		    finished_at = $4
		WHERE id = $1
	`

	tag, err := r.pool.Exec(
		ctx,
		query,
		jobID,
		domain.GenerationJobStatusFailed,
		errorMessage,
		finishedAt,
	)
	if err != nil {
		return fmt.Errorf("mark generation job failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}

	return nil
}

func (r *JobRepository) MarkSucceededWithOutbox(
	ctx context.Context,
	jobID uuid.UUID,
	outputText string,
	finishedAt time.Time,
	event *outbox.Event,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const updateJob = `
		UPDATE generation_jobs
		SET status = $2,
		    output_text = $3,
		    error_message = NULL,
		    finished_at = $4
		WHERE id = $1
	`
	tag, err := tx.Exec(
		ctx,
		updateJob,
		jobID,
		domain.GenerationJobStatusSucceeded,
		outputText,
		finishedAt,
	)
	if err != nil {
		return fmt.Errorf("mark generation job succeeded: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}

	const insertOutbox = `
		INSERT INTO outbox_events (id, event_type, aggregate_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := tx.Exec(
		ctx,
		insertOutbox,
		event.ID,
		event.EventType,
		event.AggregateID,
		event.Payload,
		event.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
