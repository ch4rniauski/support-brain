package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ch4rniauski/support-brain/internal/template/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadyImported = errors.New("template draft already imported")
	ErrNotFound        = errors.New("not found")
	ErrNotDraft        = errors.New("not draft")
)

type TemplateRepository struct {
	pool *pgxpool.Pool
}

func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

func (r *TemplateRepository) ExistsByGenerationJobID(
	ctx context.Context,
	generationJobID uuid.UUID,
) (bool, error) {
	const query = `
		SELECT 1
		FROM templates
		WHERE generation_job_id = $1
		LIMIT 1
	`

	var exists int
	err := r.pool.QueryRow(ctx, query, generationJobID).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check template by generation_job_id: %w", err)
	}

	return true, nil
}

func (r *TemplateRepository) CreateDraft(ctx context.Context, template *domain.Template) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const query = `
		INSERT INTO templates (
			id,
			cluster_id,
			status,
			version,
			body,
			generation_job_id,
			used_count,
			approved_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = tx.Exec(
		ctx,
		query,
		template.ID,
		template.ClusterID,
		template.Status,
		template.Version,
		template.Body,
		template.GenerationJobID,
		template.UsedCount,
		template.ApprovedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyImported
		}
		return fmt.Errorf("insert draft template: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *TemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	const query = `
		SELECT
			id,
			cluster_id,
			status,
			version,
			body,
			generation_job_id,
			used_count,
			approved_at
		FROM templates
		WHERE id = $1
	`

	var template domain.Template
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&template.ID,
		&template.ClusterID,
		&template.Status,
		&template.Version,
		&template.Body,
		&template.GenerationJobID,
		&template.UsedCount,
		&template.ApprovedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	return &template, nil
}

func (r *TemplateRepository) UpdateDraftBody(
	ctx context.Context,
	id uuid.UUID,
	body string,
) (*domain.Template, error) {
	const query = `
		UPDATE templates
		SET
			body = $2,
			version = version + 1
		WHERE id = $1 AND status = $3
		RETURNING
			id,
			cluster_id,
			status,
			version,
			body,
			generation_job_id,
			used_count,
			approved_at
	`

	var template domain.Template
	err := r.pool.QueryRow(ctx, query, id, body, domain.TemplateStatusDraft).Scan(
		&template.ID,
		&template.ClusterID,
		&template.Status,
		&template.Version,
		&template.Body,
		&template.GenerationJobID,
		&template.UsedCount,
		&template.ApprovedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update draft template body: %w", err)
	}

	return &template, nil
}

func (r *TemplateRepository) Approve(
	ctx context.Context,
	id uuid.UUID,
	approvedAt time.Time,
) (*domain.Template, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const selectQuery = `
		SELECT
			id,
			cluster_id,
			status,
			version,
			body,
			generation_job_id,
			used_count,
			approved_at
		FROM templates
		WHERE id = $1
		FOR UPDATE
	`

	var template domain.Template
	err = tx.QueryRow(ctx, selectQuery, id).Scan(
		&template.ID,
		&template.ClusterID,
		&template.Status,
		&template.Version,
		&template.Body,
		&template.GenerationJobID,
		&template.UsedCount,
		&template.ApprovedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get template for approve: %w", err)
	}

	if template.Status != domain.TemplateStatusDraft {
		return nil, ErrNotDraft
	}

	const archiveQuery = `
		UPDATE templates
		SET status = $1
		WHERE cluster_id = $2
			AND status = $3
			AND id <> $4
	`
	if _, err := tx.Exec(
		ctx,
		archiveQuery,
		domain.TemplateStatusArchived,
		template.ClusterID,
		domain.TemplateStatusApproved,
		template.ID,
	); err != nil {
		return nil, fmt.Errorf("archive approved templates: %w", err)
	}

	const approveQuery = `
		UPDATE templates
		SET
			status = $2,
			approved_at = $3
		WHERE id = $1
		RETURNING
			id,
			cluster_id,
			status,
			version,
			body,
			generation_job_id,
			used_count,
			approved_at
	`
	err = tx.QueryRow(
		ctx,
		approveQuery,
		template.ID,
		domain.TemplateStatusApproved,
		approvedAt,
	).Scan(
		&template.ID,
		&template.ClusterID,
		&template.Status,
		&template.Version,
		&template.Body,
		&template.GenerationJobID,
		&template.UsedCount,
		&template.ApprovedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("approve template: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &template, nil
}

func (r *TemplateRepository) ListByStatus(
	ctx context.Context,
	status domain.TemplateStatus,
) ([]domain.Template, error) {
	const query = `
		SELECT
			id,
			cluster_id,
			status,
			version,
			body,
			generation_job_id,
			used_count,
			approved_at
		FROM templates
		WHERE status = $1
		ORDER BY id
	`

	rows, err := r.pool.Query(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("list templates by status: %w", err)
	}
	defer rows.Close()

	templates := make([]domain.Template, 0)
	for rows.Next() {
		var template domain.Template
		if err := rows.Scan(
			&template.ID,
			&template.ClusterID,
			&template.Status,
			&template.Version,
			&template.Body,
			&template.GenerationJobID,
			&template.UsedCount,
			&template.ApprovedAt,
		); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, template)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate templates: %w", err)
	}

	return templates, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
