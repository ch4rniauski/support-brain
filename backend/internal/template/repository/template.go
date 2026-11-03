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

var ErrAlreadyImported = errors.New("template draft already imported")

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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
