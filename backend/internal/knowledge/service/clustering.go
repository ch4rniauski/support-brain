package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ch4rniauski/support-brain/internal/knowledge/domain"
	"github.com/ch4rniauski/support-brain/internal/knowledge/dto"
	"github.com/ch4rniauski/support-brain/internal/knowledge/repository"
	"github.com/ch4rniauski/support-brain/pkg/kafka"
	"github.com/ch4rniauski/support-brain/pkg/outbox"
	"github.com/google/uuid"
)

func permanent(err error) error {
	return fmt.Errorf("%w: %w", kafka.ErrPermanent, err)
}

type similarityClient interface {
	FindSimilar(
		ctx context.Context,
		qaPairID uuid.UUID,
		threshold float64,
		limit int,
	) (*dto.FindSimilarResponse, error)
}

type clusterStore interface {
	ExistsMemberByQAPairID(ctx context.Context, qaPairID uuid.UUID) (bool, error)
	ExistsPendingDisputedByQAPairID(ctx context.Context, qaPairID uuid.UUID) (bool, error)
	FindMembersByQAPairIDs(ctx context.Context, qaPairIDs []uuid.UUID) (map[uuid.UUID]domain.ClusterMember, error)
	GetClusterByID(ctx context.Context, id uuid.UUID) (*domain.Cluster, error)
	ListMemberQAPairIDs(ctx context.Context, clusterID uuid.UUID) ([]uuid.UUID, error)
	CreateClusterWithMember(ctx context.Context, cluster *domain.Cluster, member *domain.ClusterMember) error
	AddMemberAndIncrementCount(ctx context.Context, member *domain.ClusterMember) error
	CreateDisputedWithCandidates(
		ctx context.Context,
		disputed *domain.DisputedQAPair,
		candidates []domain.DisputedCandidate,
	) error
	MarkReadyToGenerateWithOutbox(
		ctx context.Context,
		clusterID uuid.UUID,
		event *outbox.Event,
	) (bool, error)
}

type ClusteringConfig struct {
	SimilarityThreshold  float64
	SearchLimit          int
	GenerationThresholdN int
}

type ClusteringService struct {
	store  clusterStore
	client similarityClient
	cfg    ClusteringConfig
}

func NewClusteringService(
	store *repository.ClusterRepository,
	client similarityClient,
	cfg ClusteringConfig,
) *ClusteringService {
	if cfg.SimilarityThreshold <= 0 {
		cfg.SimilarityThreshold = 0.85
	}
	if cfg.SearchLimit <= 0 {
		cfg.SearchLimit = 10
	}
	if cfg.GenerationThresholdN <= 0 {
		cfg.GenerationThresholdN = 5
	}

	return &ClusteringService{
		store:  store,
		client: client,
		cfg:    cfg,
	}
}

func (s *ClusteringService) Handle(ctx context.Context, eventType string, payload []byte) error {
	if eventType != "" && eventType != dto.EventTypeEmbeddingReady {
		log.Printf("knowledge: skip unsupported event type %q", eventType)
		return nil
	}

	var event dto.EmbeddingReadyEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return permanent(fmt.Errorf("unmarshal embedding ready event: %w", err))
	}
	if event.QAPairID == uuid.Nil {
		return permanent(fmt.Errorf("qa_pair_id is required"))
	}

	return s.handleEmbeddingReady(ctx, event.QAPairID)
}

func (s *ClusteringService) handleEmbeddingReady(ctx context.Context, qaPairID uuid.UUID) error {
	exists, err := s.store.ExistsMemberByQAPairID(ctx, qaPairID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	pending, err := s.store.ExistsPendingDisputedByQAPairID(ctx, qaPairID)
	if err != nil {
		return err
	}
	if pending {
		return nil
	}

	resp, err := s.client.FindSimilar(
		ctx,
		qaPairID,
		s.cfg.SimilarityThreshold,
		s.cfg.SearchLimit,
	)
	if err != nil {
		return fmt.Errorf("find similar: %w", err)
	}

	strongNeighbors := filterByThreshold(resp.Results, s.cfg.SimilarityThreshold)
	neighborIDs := make([]uuid.UUID, 0, len(strongNeighbors))
	for _, neighbor := range strongNeighbors {
		neighborIDs = append(neighborIDs, neighbor.QAPairID)
	}

	membersByQAPair, err := s.store.FindMembersByQAPairIDs(ctx, neighborIDs)
	if err != nil {
		return err
	}

	type clusteredNeighbor struct {
		QAPairID  uuid.UUID
		ClusterID uuid.UUID
		Score     float64
	}

	clusterVotes := make(map[uuid.UUID]float64)
	clusteredNeighbors := make([]clusteredNeighbor, 0)
	for _, neighbor := range strongNeighbors {
		member, ok := membersByQAPair[neighbor.QAPairID]
		if !ok {
			continue
		}
		clusteredNeighbors = append(clusteredNeighbors, clusteredNeighbor{
			QAPairID:  neighbor.QAPairID,
			ClusterID: member.ClusterID,
			Score:     neighbor.Score,
		})
		if best, exists := clusterVotes[member.ClusterID]; !exists || neighbor.Score > best {
			clusterVotes[member.ClusterID] = neighbor.Score
		}
	}

	switch len(clusterVotes) {
	case 0:
		clusterID := uuid.New()
		cluster := &domain.Cluster{
			ID:                   clusterID,
			Title:                nil,
			Status:               domain.ClusterStatusCollecting,
			MemberCount:          1,
			SimilarityThreshold:  s.cfg.SimilarityThreshold,
			GenerationThresholdN: s.cfg.GenerationThresholdN,
			ActiveTemplateID:     nil,
		}
		member := &domain.ClusterMember{
			ID:              uuid.New(),
			ClusterID:       clusterID,
			QAPairID:        qaPairID,
			SimilarityScore: nil,
		}
		if err := s.store.CreateClusterWithMember(ctx, cluster, member); err != nil {
			if errors.Is(err, repository.ErrAlreadyMember) {
				return nil
			}
			return err
		}
		return s.checkGenerationTrigger(ctx, clusterID)

	case 1:
		var targetClusterID uuid.UUID
		var bestScore float64
		for clusterID, score := range clusterVotes {
			targetClusterID = clusterID
			bestScore = score
		}
		scoreCopy := bestScore
		member := &domain.ClusterMember{
			ID:              uuid.New(),
			ClusterID:       targetClusterID,
			QAPairID:        qaPairID,
			SimilarityScore: &scoreCopy,
		}
		if err := s.store.AddMemberAndIncrementCount(ctx, member); err != nil {
			if errors.Is(err, repository.ErrAlreadyMember) {
				return nil
			}
			return err
		}
		return s.checkGenerationTrigger(ctx, targetClusterID)

	default:
		disputedID := uuid.New()
		disputed := &domain.DisputedQAPair{
			ID:                  disputedID,
			QAPairID:            qaPairID,
			Status:              domain.DisputedQAPairStatusPending,
			ResolutionClusterID: nil,
			ResolvedAt:          nil,
		}
		candidates := make([]domain.DisputedCandidate, 0, len(clusteredNeighbors))
		for _, neighbor := range clusteredNeighbors {
			candidates = append(candidates, domain.DisputedCandidate{
				ID:               uuid.New(),
				DisputedQAPairID: disputedID,
				ClusterID:        neighbor.ClusterID,
				NeighborQAPairID: neighbor.QAPairID,
				Score:            neighbor.Score,
			})
		}
		if err := s.store.CreateDisputedWithCandidates(ctx, disputed, candidates); err != nil {
			if errors.Is(err, repository.ErrAlreadyDisputed) {
				return nil
			}
			return err
		}
		return nil
	}
}

func (s *ClusteringService) checkGenerationTrigger(ctx context.Context, clusterID uuid.UUID) error {
	cluster, err := s.store.GetClusterByID(ctx, clusterID)
	if err != nil {
		return err
	}

	if cluster.MemberCount < cluster.GenerationThresholdN {
		return nil
	}

	switch cluster.Status {
	case domain.ClusterStatusHasDraft, domain.ClusterStatusHasApproved, domain.ClusterStatusReadyToGenerate:
		return nil
	case domain.ClusterStatusCollecting:
		// continue
	default:
		return nil
	}

	qaPairIDs, err := s.store.ListMemberQAPairIDs(ctx, clusterID)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(dto.ClusterReadyForGenerationEvent{
		ClusterID: clusterID,
		QAPairIDs: qaPairIDs,
	})
	if err != nil {
		return fmt.Errorf("marshal cluster ready for generation event: %w", err)
	}

	event := &outbox.Event{
		ID:          uuid.New(),
		EventType:   domain.EventTypeClusterReadyForGeneration,
		AggregateID: clusterID,
		Payload:     payload,
		CreatedAt:   time.Now().UTC(),
	}

	published, err := s.store.MarkReadyToGenerateWithOutbox(ctx, clusterID, event)
	if err != nil {
		return err
	}
	if !published {
		return nil
	}

	return nil
}

func filterByThreshold(results []dto.SimilarResult, threshold float64) []dto.SimilarResult {
	filtered := make([]dto.SimilarResult, 0, len(results))
	for _, result := range results {
		if result.Score >= threshold {
			filtered = append(filtered, result)
		}
	}
	return filtered
}
