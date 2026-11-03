package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ch4rniauski/support-brain/internal/embedding/domain"
	"github.com/ch4rniauski/support-brain/internal/embedding/dto"
	"github.com/ch4rniauski/support-brain/internal/embedding/repository"
	"github.com/ch4rniauski/support-brain/pkg/kafka"
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

	exists              bool
	existsErr           error
	existsCalled        bool
	lastExistsQAPairID  uuid.UUID
	lastExistsModelName string
	saveWithOutboxErr   error
	saveCalled          bool
	savedEmbedding      *domain.QuestionEmbedding
	savedEvent          *outbox.Event
}

func (f *fakeEmbeddingStore) ExistsByQAPairAndModel(
	ctx context.Context,
	qaPairID uuid.UUID,
	modelName string,
) (bool, error) {
	f.existsCalled = true
	f.lastExistsQAPairID = qaPairID
	f.lastExistsModelName = modelName
	return f.exists, f.existsErr
}

func (f *fakeEmbeddingStore) SaveWithOutbox(
	ctx context.Context,
	embedding *domain.QuestionEmbedding,
	event *outbox.Event,
) error {
	f.saveCalled = true
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
	model      string
	vec        []float32
	err        error
	embedCalls int
	lastText   string
}

func (f *fakeEmbedder) Model() string {
	if f.model == "" {
		return "test-model"
	}
	return f.model
}

func (f *fakeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	f.embedCalls++
	f.lastText = text
	if f.err != nil {
		return nil, f.err
	}
	if f.vec == nil {
		return []float32{0.1, 0.2, 0.3}, nil
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

func mustMarshalQAPairCreated(t *testing.T, event dto.QAPairCreatedEvent) []byte {
	t.Helper()

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	return payload
}

func TestEmbeddingService_Handle(t *testing.T) {
	t.Parallel()

	qaPairID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	validEvent := dto.QAPairCreatedEvent{
		QAPairID: qaPairID,
		Question: "How do I reset my password?",
		Answer:   "Use the reset link.",
	}
	validPayload := mustMarshalQAPairCreated(t, validEvent)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		store := &fakeEmbeddingStore{}
		embedder := &fakeEmbedder{
			model: "nomic-embed-text",
			vec:   []float32{1, 2, 3},
		}
		svc := NewEmbeddingService(store, embedder)

		if err := svc.Handle(context.Background(), dto.EventTypeQAPairCreated, validPayload); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if !store.existsCalled {
			t.Fatal("expected ExistsByQAPairAndModel to be called")
		}
		if store.lastExistsQAPairID != qaPairID || store.lastExistsModelName != "nomic-embed-text" {
			t.Errorf("exists args = (%v, %q)", store.lastExistsQAPairID, store.lastExistsModelName)
		}
		if embedder.embedCalls != 1 || embedder.lastText != validEvent.Question {
			t.Errorf("Embed calls = %d text = %q", embedder.embedCalls, embedder.lastText)
		}
		if !store.saveCalled || store.savedEmbedding == nil || store.savedEvent == nil {
			t.Fatal("expected SaveWithOutbox to be called")
		}
		if store.savedEmbedding.QAPairID != qaPairID {
			t.Errorf("embedding QAPairID = %v, want %v", store.savedEmbedding.QAPairID, qaPairID)
		}
		if store.savedEmbedding.ModelName != "nomic-embed-text" {
			t.Errorf("embedding ModelName = %q, want nomic-embed-text", store.savedEmbedding.ModelName)
		}
		if len(store.savedEmbedding.Embedding) != 3 {
			t.Errorf("embedding vector len = %d, want 3", len(store.savedEmbedding.Embedding))
		}
		if store.savedEvent.EventType != domain.EventTypeEmbeddingReady {
			t.Errorf("outbox EventType = %q, want %q", store.savedEvent.EventType, domain.EventTypeEmbeddingReady)
		}
		if store.savedEvent.AggregateID != qaPairID {
			t.Errorf("outbox AggregateID = %v, want %v", store.savedEvent.AggregateID, qaPairID)
		}

		var ready dto.EmbeddingReadyEvent
		if err := json.Unmarshal(store.savedEvent.Payload, &ready); err != nil {
			t.Fatalf("unmarshal outbox payload: %v", err)
		}
		if ready.QAPairID != qaPairID {
			t.Errorf("outbox payload QAPairID = %v, want %v", ready.QAPairID, qaPairID)
		}
	})

	t.Run("skips unsupported event type", func(t *testing.T) {
		t.Parallel()

		store := &fakeEmbeddingStore{}
		embedder := &fakeEmbedder{}
		svc := NewEmbeddingService(store, embedder)

		if err := svc.Handle(context.Background(), "SomethingElse", validPayload); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if store.existsCalled || store.saveCalled || embedder.embedCalls != 0 {
			t.Fatal("expected no store/embedder calls for unsupported event")
		}
	})

	t.Run("empty event type is accepted", func(t *testing.T) {
		t.Parallel()

		store := &fakeEmbeddingStore{}
		svc := NewEmbeddingService(store, &fakeEmbedder{model: "m", vec: []float32{0.5}})

		if err := svc.Handle(context.Background(), "", validPayload); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if !store.saveCalled {
			t.Fatal("expected SaveWithOutbox for empty event type")
		}
	})

	t.Run("invalid json is permanent", func(t *testing.T) {
		t.Parallel()

		svc := NewEmbeddingService(&fakeEmbeddingStore{}, &fakeEmbedder{})

		err := svc.Handle(context.Background(), dto.EventTypeQAPairCreated, []byte("{"))
		if !errors.Is(err, kafka.ErrPermanent) {
			t.Fatalf("error = %v, want permanent", err)
		}
	})

	t.Run("missing qa pair id is permanent", func(t *testing.T) {
		t.Parallel()

		payload := mustMarshalQAPairCreated(t, dto.QAPairCreatedEvent{
			Question: "question",
		})
		svc := NewEmbeddingService(&fakeEmbeddingStore{}, &fakeEmbedder{})

		err := svc.Handle(context.Background(), dto.EventTypeQAPairCreated, payload)
		if !errors.Is(err, kafka.ErrPermanent) {
			t.Fatalf("error = %v, want permanent", err)
		}
	})

	t.Run("missing question is permanent", func(t *testing.T) {
		t.Parallel()

		payload := mustMarshalQAPairCreated(t, dto.QAPairCreatedEvent{
			QAPairID: qaPairID,
		})
		svc := NewEmbeddingService(&fakeEmbeddingStore{}, &fakeEmbedder{})

		err := svc.Handle(context.Background(), dto.EventTypeQAPairCreated, payload)
		if !errors.Is(err, kafka.ErrPermanent) {
			t.Fatalf("error = %v, want permanent", err)
		}
	})

	t.Run("already exists is no-op", func(t *testing.T) {
		t.Parallel()

		store := &fakeEmbeddingStore{exists: true}
		embedder := &fakeEmbedder{}
		svc := NewEmbeddingService(store, embedder)

		if err := svc.Handle(context.Background(), dto.EventTypeQAPairCreated, validPayload); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if embedder.embedCalls != 0 {
			t.Fatal("Embed must not be called when embedding already exists")
		}
		if store.saveCalled {
			t.Fatal("SaveWithOutbox must not be called when embedding already exists")
		}
	})

	t.Run("exists check fails", func(t *testing.T) {
		t.Parallel()

		storeErr := errors.New("db down")
		store := &fakeEmbeddingStore{existsErr: storeErr}
		svc := NewEmbeddingService(store, &fakeEmbedder{})

		err := svc.Handle(context.Background(), dto.EventTypeQAPairCreated, validPayload)
		if !errors.Is(err, storeErr) {
			t.Fatalf("error = %v, want %v", err, storeErr)
		}
	})

	t.Run("embed fails", func(t *testing.T) {
		t.Parallel()

		embedErr := errors.New("ollama down")
		store := &fakeEmbeddingStore{}
		svc := NewEmbeddingService(store, &fakeEmbedder{err: embedErr})

		err := svc.Handle(context.Background(), dto.EventTypeQAPairCreated, validPayload)
		if !errors.Is(err, embedErr) {
			t.Fatalf("error = %v, want %v", err, embedErr)
		}
		if store.saveCalled {
			t.Fatal("SaveWithOutbox must not be called when embed fails")
		}
	})

	t.Run("already processed is no-op", func(t *testing.T) {
		t.Parallel()

		store := &fakeEmbeddingStore{saveWithOutboxErr: repository.ErrAlreadyProcessed}
		svc := NewEmbeddingService(store, &fakeEmbedder{vec: []float32{1}})

		if err := svc.Handle(context.Background(), dto.EventTypeQAPairCreated, validPayload); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	})

	t.Run("save fails", func(t *testing.T) {
		t.Parallel()

		saveErr := errors.New("save failed")
		store := &fakeEmbeddingStore{saveWithOutboxErr: saveErr}
		svc := NewEmbeddingService(store, &fakeEmbedder{vec: []float32{1}})

		err := svc.Handle(context.Background(), dto.EventTypeQAPairCreated, validPayload)
		if !errors.Is(err, saveErr) {
			t.Fatalf("error = %v, want %v", err, saveErr)
		}
		if errors.Is(err, repository.ErrAlreadyProcessed) {
			t.Fatal("must not treat generic save error as already processed")
		}
	})
}
