package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ch4rniauski/support-brain/internal/embedding/domain"
	"github.com/ch4rniauski/support-brain/internal/embedding/dto"
	"github.com/ch4rniauski/support-brain/internal/embedding/repository"
	"github.com/ch4rniauski/support-brain/pkg/outbox"
	"github.com/google/uuid"
)

type fakeEmbeddingStore struct {
	findSimilar       []domain.SimilarQuestion
	findSimilarErr    error
	lastFindQAPairID  uuid.UUID
	lastFindModelName string
	lastFindThreshold float64
	lastFindLimit     int

	exists            bool
	existsErr         error
	saveWithOutboxErr error
	savedEmbedding    *domain.QuestionEmbedding
	savedEvent        *outbox.Event
}

func (f *fakeEmbeddingStore) ExistsByQAPairAndModel(
	ctx context.Context,
	qaPairID uuid.UUID,
	modelName string,
) (bool, error) {
	return f.exists, f.existsErr
}

func (f *fakeEmbeddingStore) SaveWithOutbox(
	ctx context.Context,
	embedding *domain.QuestionEmbedding,
	event *outbox.Event,
) error {
	f.savedEmbedding = embedding
	f.savedEvent = event
	return f.saveWithOutboxErr
}

func (f *fakeEmbeddingStore) FindSimilar(
	ctx context.Context,
	qaPairID uuid.UUID,
	modelName string,
	threshold float64,
	limit int,
) ([]domain.SimilarQuestion, error) {
	f.lastFindQAPairID = qaPairID
	f.lastFindModelName = modelName
	f.lastFindThreshold = threshold
	f.lastFindLimit = limit
	if f.findSimilarErr != nil {
		return nil, f.findSimilarErr
	}
	return f.findSimilar, nil
}

type fakeEmbedder struct {
	model string
	vec   []float32
	err   error
}

func (f *fakeEmbedder) Model() string {
	if f.model == "" {
		return "test-model"
	}
	return f.model
}

func (f *fakeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.vec, nil
}

func TestEmbeddingService_FindSimilar(t *testing.T) {
	t.Parallel()

	qaPairID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	matchID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	validReq := dto.FindSimilarRequest{
		QAPairID:  qaPairID,
		ModelName: "nomic-embed-text",
		Threshold: 0.8,
		Limit:     5,
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		store := &fakeEmbeddingStore{
			findSimilar: []domain.SimilarQuestion{
				{QAPairID: matchID, Score: 0.91},
			},
		}
		svc := NewEmbeddingService(store, &fakeEmbedder{})

		resp, err := svc.FindSimilar(context.Background(), validReq)
		if err != nil {
			t.Fatalf("FindSimilar() error = %v", err)
		}
		if resp.QAPairID != qaPairID {
			t.Errorf("QAPairID = %v, want %v", resp.QAPairID, qaPairID)
		}
		if len(resp.Results) != 1 {
			t.Fatalf("Results len = %d, want 1", len(resp.Results))
		}
		if resp.Results[0].QAPairID != matchID {
			t.Errorf("Results[0].QAPairID = %v, want %v", resp.Results[0].QAPairID, matchID)
		}
		if resp.Results[0].Score != 0.91 {
			t.Errorf("Results[0].Score = %v, want 0.91", resp.Results[0].Score)
		}
		if store.lastFindQAPairID != validReq.QAPairID ||
			store.lastFindModelName != validReq.ModelName ||
			store.lastFindThreshold != validReq.Threshold ||
			store.lastFindLimit != validReq.Limit {
			t.Errorf("store called with unexpected args: %#v", store)
		}
	})

	t.Run("success with empty results", func(t *testing.T) {
		t.Parallel()

		store := &fakeEmbeddingStore{findSimilar: nil}
		svc := NewEmbeddingService(store, &fakeEmbedder{})

		resp, err := svc.FindSimilar(context.Background(), validReq)
		if err != nil {
			t.Fatalf("FindSimilar() error = %v", err)
		}
		if resp.QAPairID != qaPairID {
			t.Errorf("QAPairID = %v, want %v", resp.QAPairID, qaPairID)
		}
		if resp.Results == nil {
			t.Fatal("Results = nil, want empty slice")
		}
		if len(resp.Results) != 0 {
			t.Fatalf("Results len = %d, want 0", len(resp.Results))
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			req  dto.FindSimilarRequest
		}{
			{
				name: "missing qa pair id",
				req: dto.FindSimilarRequest{
					ModelName: "nomic-embed-text",
					Threshold: 0.8,
					Limit:     5,
				},
			},
			{
				name: "missing model name",
				req: dto.FindSimilarRequest{
					QAPairID:  qaPairID,
					Threshold: 0.8,
					Limit:     5,
				},
			},
			{
				name: "threshold too high",
				req: dto.FindSimilarRequest{
					QAPairID:  qaPairID,
					ModelName: "nomic-embed-text",
					Threshold: 1.1,
					Limit:     5,
				},
			},
			{
				name: "limit too low",
				req: dto.FindSimilarRequest{
					QAPairID:  qaPairID,
					ModelName: "nomic-embed-text",
					Threshold: 0.8,
					Limit:     0,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				store := &fakeEmbeddingStore{}
				svc := NewEmbeddingService(store, &fakeEmbedder{})

				resp, err := svc.FindSimilar(context.Background(), tt.req)
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("error = %v, want %v", err, ErrInvalidInput)
				}
				if resp != nil {
					t.Errorf("response = %#v, want nil", resp)
				}
				if store.lastFindModelName != "" {
					t.Error("store must not be called on invalid input")
				}
			})
		}
	})

	t.Run("embedding not found", func(t *testing.T) {
		t.Parallel()

		store := &fakeEmbeddingStore{findSimilarErr: repository.ErrEmbeddingNotFound}
		svc := NewEmbeddingService(store, &fakeEmbedder{})

		resp, err := svc.FindSimilar(context.Background(), validReq)
		if !errors.Is(err, ErrEmbeddingNotFound) {
			t.Fatalf("error = %v, want %v", err, ErrEmbeddingNotFound)
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		storeErr := errors.New("db down")
		store := &fakeEmbeddingStore{findSimilarErr: storeErr}
		svc := NewEmbeddingService(store, &fakeEmbedder{})

		resp, err := svc.FindSimilar(context.Background(), validReq)
		if !errors.Is(err, storeErr) {
			t.Fatalf("error = %v, want %v", err, storeErr)
		}
		if errors.Is(err, ErrEmbeddingNotFound) {
			t.Fatal("must not map generic store error to ErrEmbeddingNotFound")
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
	})
}
