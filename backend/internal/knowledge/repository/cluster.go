package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ch4rniauski/support-brain/internal/knowledge/domain"
	"github.com/ch4rniauski/support-brain/pkg/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadyMember   = errors.New("qa pair already clustered")
	ErrAlreadyDisputed = errors.New("qa pair already disputed")
	ErrClusterNotFound = errors.New("cluster not found")
)

type ClusterRepository struct {
	pool *pgxpool.Pool
}

func NewClusterRepository(pool *pgxpool.Pool) *ClusterRepository {
	return &ClusterRepository{pool: pool}
}

func (r *ClusterRepository) ExistsMemberByQAPairID(ctx context.Context, qaPairID uuid.UUID) (bool, error) {
	const query = `SELECT 1 FROM cluster_members WHERE qa_pair_id = $1`

	var exists int
	err := r.pool.QueryRow(ctx, query, qaPairID).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check cluster member: %w", err)
	}

	return true, nil
}

func (r *ClusterRepository) ExistsPendingDisputedByQAPairID(ctx context.Context, qaPairID uuid.UUID) (bool, error) {
	const query = `
		SELECT 1
		FROM disputed_qa_pairs
		WHERE qa_pair_id = $1 AND status = $2
	`

	var exists int
	err := r.pool.QueryRow(ctx, query, qaPairID, domain.DisputedQAPairStatusPending).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check pending disputed qa pair: %w", err)
	}

	return true, nil
}

func (r *ClusterRepository) FindMembersByQAPairIDs(
	ctx context.Context,
	qaPairIDs []uuid.UUID,
) (map[uuid.UUID]domain.ClusterMember, error) {
	result := make(map[uuid.UUID]domain.ClusterMember, len(qaPairIDs))
	if len(qaPairIDs) == 0 {
		return result, nil
	}

	const query = `
		SELECT id, cluster_id, qa_pair_id, similarity_score
		FROM cluster_members
		WHERE qa_pair_id = ANY($1)
	`

	rows, err := r.pool.Query(ctx, query, qaPairIDs)
	if err != nil {
		return nil, fmt.Errorf("find cluster members by qa pair ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var member domain.ClusterMember
		if err := rows.Scan(
			&member.ID,
			&member.ClusterID,
			&member.QAPairID,
			&member.SimilarityScore,
		); err != nil {
			return nil, fmt.Errorf("scan cluster member: %w", err)
		}
		result[member.QAPairID] = member
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cluster members: %w", err)
	}

	return result, nil
}

func (r *ClusterRepository) GetClusterByID(ctx context.Context, id uuid.UUID) (*domain.Cluster, error) {
	const query = `
		SELECT id, title, status, member_count, similarity_threshold, generation_threshold_n, active_template_id
		FROM clusters
		WHERE id = $1
	`

	var cluster domain.Cluster
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&cluster.ID,
		&cluster.Title,
		&cluster.Status,
		&cluster.MemberCount,
		&cluster.SimilarityThreshold,
		&cluster.GenerationThresholdN,
		&cluster.ActiveTemplateID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClusterNotFound
		}
		return nil, fmt.Errorf("get cluster: %w", err)
	}

	return &cluster, nil
}

func (r *ClusterRepository) ListMemberQAPairIDs(ctx context.Context, clusterID uuid.UUID) ([]uuid.UUID, error) {
	const query = `
		SELECT qa_pair_id
		FROM cluster_members
		WHERE cluster_id = $1
		ORDER BY id
	`

	rows, err := r.pool.Query(ctx, query, clusterID)
	if err != nil {
		return nil, fmt.Errorf("list cluster member qa pair ids: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan cluster member qa pair id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cluster member qa pair ids: %w", err)
	}

	return ids, nil
}

func (r *ClusterRepository) CreateClusterWithMember(
	ctx context.Context,
	cluster *domain.Cluster,
	member *domain.ClusterMember,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertCluster = `
		INSERT INTO clusters (
			id, title, status, member_count, similarity_threshold, generation_threshold_n, active_template_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = tx.Exec(
		ctx,
		insertCluster,
		cluster.ID,
		cluster.Title,
		cluster.Status,
		cluster.MemberCount,
		cluster.SimilarityThreshold,
		cluster.GenerationThresholdN,
		cluster.ActiveTemplateID,
	)
	if err != nil {
		return fmt.Errorf("insert cluster: %w", err)
	}

	if err := insertClusterMember(ctx, tx, member); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *ClusterRepository) AddMemberAndIncrementCount(
	ctx context.Context,
	member *domain.ClusterMember,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := insertClusterMember(ctx, tx, member); err != nil {
		return err
	}

	const updateCount = `
		UPDATE clusters
		SET member_count = member_count + 1
		WHERE id = $1
	`
	tag, err := tx.Exec(ctx, updateCount, member.ClusterID)
	if err != nil {
		return fmt.Errorf("increment cluster member count: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrClusterNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *ClusterRepository) CreateDisputedWithCandidates(
	ctx context.Context,
	disputed *domain.DisputedQAPair,
	candidates []domain.DisputedCandidate,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertDisputed = `
		INSERT INTO disputed_qa_pairs (
			id, qa_pair_id, status, resolution_cluster_id, resolved_at
		) VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.Exec(
		ctx,
		insertDisputed,
		disputed.ID,
		disputed.QAPairID,
		disputed.Status,
		disputed.ResolutionClusterID,
		disputed.ResolvedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyDisputed
		}
		return fmt.Errorf("insert disputed qa pair: %w", err)
	}

	const insertCandidate = `
		INSERT INTO disputed_candidates (
			id, disputed_qa_pair_id, cluster_id, neighbor_qa_pair_id, score
		) VALUES ($1, $2, $3, $4, $5)
	`
	for _, candidate := range candidates {
		_, err = tx.Exec(
			ctx,
			insertCandidate,
			candidate.ID,
			candidate.DisputedQAPairID,
			candidate.ClusterID,
			candidate.NeighborQAPairID,
			candidate.Score,
		)
		if err != nil {
			return fmt.Errorf("insert disputed candidate: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *ClusterRepository) MarkReadyToGenerateWithOutbox(
	ctx context.Context,
	clusterID uuid.UUID,
	event *outbox.Event,
) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const updateStatus = `
		UPDATE clusters
		SET status = $2
		WHERE id = $1
		  AND status = $3
	`
	tag, err := tx.Exec(
		ctx,
		updateStatus,
		clusterID,
		domain.ClusterStatusReadyToGenerate,
		domain.ClusterStatusCollecting,
	)
	if err != nil {
		return false, fmt.Errorf("mark cluster ready to generate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}

	const insertOutbox = `
		INSERT INTO outbox_events (id, event_type, aggregate_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.Exec(
		ctx,
		insertOutbox,
		event.ID,
		event.EventType,
		event.AggregateID,
		event.Payload,
		event.CreatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("insert outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit tx: %w", err)
	}

	return true, nil
}

func insertClusterMember(ctx context.Context, tx pgx.Tx, member *domain.ClusterMember) error {
	const query = `
		INSERT INTO cluster_members (id, cluster_id, qa_pair_id, similarity_score)
		VALUES ($1, $2, $3, $4)
	`
	_, err := tx.Exec(
		ctx,
		query,
		member.ID,
		member.ClusterID,
		member.QAPairID,
		member.SimilarityScore,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyMember
		}
		return fmt.Errorf("insert cluster member: %w", err)
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
