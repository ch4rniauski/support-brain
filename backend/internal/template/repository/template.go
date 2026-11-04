package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ch4rniauski/support-brain/internal/template/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadyImported = errors.New("template draft already imported")
	ErrNotFound        = errors.New("not found")
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
