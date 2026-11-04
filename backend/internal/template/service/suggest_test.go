package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ch4rniauski/support-brain/internal/template/domain"
	"github.com/ch4rniauski/support-brain/internal/template/dto"
	"github.com/google/uuid"
)

type suggestStoreStub struct {
	templates []domain.Template
	err       error
	lastIDs   []uuid.UUID
}

func (s *suggestStoreStub) ExistsByGenerationJobID(ctx context.Context, generationJobID uuid.UUID) (bool, error) {
	panic("unexpected ExistsByGenerationJobID")
}

func (s *suggestStoreStub) CreateDraft(ctx context.Context, template *domain.Template) error {
	panic("unexpected CreateDraft")
}

func (s *suggestStoreStub) GetByID(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	panic("unexpected GetByID")
}

func (s *suggestStoreStub) UpdateDraftBody(ctx context.Context, id uuid.UUID, body string) (*domain.Template, error) {
	panic("unexpected UpdateDraftBody")
}

func (s *suggestStoreStub) Approve(ctx context.Context, id uuid.UUID, approvedAt time.Time) (*domain.Template, error) {
	panic("unexpected Approve")
}

func (s *suggestStoreStub) Reject(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	panic("unexpected Reject")
}

func (s *suggestStoreStub) ListByStatus(ctx context.Context, status domain.TemplateStatus) ([]domain.Template, error) {
	panic("unexpected ListByStatus")
}

func (s *suggestStoreStub) FindApprovedByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Template, error) {
	s.lastIDs = append([]uuid.UUID(nil), ids...)
	if s.err != nil {
		return nil, s.err
	}
	return s.templates, nil
}

type fakeEmbeddingClient struct {
	resp *dto.FindSimilarByTextResponse
	err  error
}

func (f *fakeEmbeddingClient) FindSimilarByText(
	ctx context.Context,
	text string,
	threshold float64,
	limit int,
) (*dto.FindSimilarByTextResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

type fakeKnowledgeClient struct {
	resp *dto.FindTemplatesByQAPairsResponse
	err  error
}

func (f *fakeKnowledgeClient) FindTemplatesByQAPairs(
	ctx context.Context,
	qaPairIDs []uuid.UUID,
) (*dto.FindTemplatesByQAPairsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func newSuggestService(
	store templateStore,
	embedding embeddingClient,
	knowledge knowledgeClient,
) *TemplateService {
	return NewTemplateService(store, embedding, knowledge, SuggestConfig{
		SimilarityThreshold: 0.85,
		SearchLimit:         10,
	})
}

func TestTemplateService_Suggest(t *testing.T) {
	t.Parallel()

	qa1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	qa2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	tpl1 := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	tpl2 := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	cluster1 := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	cluster2 := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")

	t.Run("returns empty when embedding has no results", func(t *testing.T) {
		t.Parallel()

		svc := newSuggestService(
			&suggestStoreStub{},
			&fakeEmbeddingClient{resp: &dto.FindSimilarByTextResponse{}},
			&fakeKnowledgeClient{},
		)

		resp, err := svc.Suggest(context.Background(), dto.SuggestTemplatesRequest{
			Question: "How do I reset my password?",
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if resp.Templates == nil || len(resp.Templates) != 0 {
			t.Fatalf("templates = %#v, want empty slice", resp.Templates)
		}
	})

	t.Run("returns empty when knowledge has no template ids", func(t *testing.T) {
		t.Parallel()

		svc := newSuggestService(
			&suggestStoreStub{},
			&fakeEmbeddingClient{resp: &dto.FindSimilarByTextResponse{
				Results: []dto.SimilarQuestionResult{{QAPairID: qa1, Score: 0.9}},
			}},
			&fakeKnowledgeClient{resp: &dto.FindTemplatesByQAPairsResponse{
				Items: []dto.TemplateByQAPairItem{{QAPairID: qa1, TemplateID: nil}},
			}},
		)

		resp, err := svc.Suggest(context.Background(), dto.SuggestTemplatesRequest{
			Question: "question",
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(resp.Templates) != 0 {
			t.Fatalf("templates len = %d, want 0", len(resp.Templates))
		}
	})

	t.Run("returns approved templates sorted by score", func(t *testing.T) {
		t.Parallel()

		store := &suggestStoreStub{
			templates: []domain.Template{
				{ID: tpl1, ClusterID: cluster1, Body: "body-1", Status: domain.TemplateStatusApproved},
				{ID: tpl2, ClusterID: cluster2, Body: "body-2", Status: domain.TemplateStatusApproved},
			},
		}
		svc := newSuggestService(
			store,
			&fakeEmbeddingClient{resp: &dto.FindSimilarByTextResponse{
				Results: []dto.SimilarQuestionResult{
					{QAPairID: qa1, Score: 0.8},
					{QAPairID: qa2, Score: 0.95},
				},
			}},
			&fakeKnowledgeClient{resp: &dto.FindTemplatesByQAPairsResponse{
				Items: []dto.TemplateByQAPairItem{
					{QAPairID: qa1, TemplateID: &tpl1},
					{QAPairID: qa2, TemplateID: &tpl2},
				},
			}},
		)

		resp, err := svc.Suggest(context.Background(), dto.SuggestTemplatesRequest{
			Question: "question",
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(resp.Templates) != 2 {
			t.Fatalf("templates len = %d, want 2", len(resp.Templates))
		}
		if resp.Templates[0].ID != tpl2 || resp.Templates[0].Score != 0.95 {
			t.Errorf("first = %#v, want tpl2 with 0.95", resp.Templates[0])
		}
		if resp.Templates[1].ID != tpl1 || resp.Templates[1].Score != 0.8 {
			t.Errorf("second = %#v, want tpl1 with 0.8", resp.Templates[1])
		}
	})

	t.Run("dedupes template ids and keeps best score", func(t *testing.T) {
		t.Parallel()

		store := &suggestStoreStub{
			templates: []domain.Template{
				{ID: tpl1, ClusterID: cluster1, Body: "body-1", Status: domain.TemplateStatusApproved},
			},
		}
		svc := newSuggestService(
			store,
			&fakeEmbeddingClient{resp: &dto.FindSimilarByTextResponse{
				Results: []dto.SimilarQuestionResult{
					{QAPairID: qa1, Score: 0.7},
					{QAPairID: qa2, Score: 0.92},
				},
			}},
			&fakeKnowledgeClient{resp: &dto.FindTemplatesByQAPairsResponse{
				Items: []dto.TemplateByQAPairItem{
					{QAPairID: qa1, TemplateID: &tpl1},
					{QAPairID: qa2, TemplateID: &tpl1},
				},
			}},
		)

		resp, err := svc.Suggest(context.Background(), dto.SuggestTemplatesRequest{
			Question: "question",
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(store.lastIDs) != 1 || store.lastIDs[0] != tpl1 {
			t.Fatalf("FindApprovedByIDs ids = %v, want [%v]", store.lastIDs, tpl1)
		}
		if len(resp.Templates) != 1 {
			t.Fatalf("templates len = %d, want 1", len(resp.Templates))
		}
		if resp.Templates[0].Score != 0.92 {
			t.Errorf("score = %v, want 0.92", resp.Templates[0].Score)
		}
	})

	t.Run("invalid empty question", func(t *testing.T) {
		t.Parallel()

		svc := newSuggestService(&suggestStoreStub{}, &fakeEmbeddingClient{}, &fakeKnowledgeClient{})

		resp, err := svc.Suggest(context.Background(), dto.SuggestTemplatesRequest{
			Question: "   ",
		})
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidInput)
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
	})

	t.Run("embedding fails", func(t *testing.T) {
		t.Parallel()

		embedErr := errors.New("embedding down")
		svc := newSuggestService(
			&suggestStoreStub{},
			&fakeEmbeddingClient{err: embedErr},
			&fakeKnowledgeClient{},
		)

		_, err := svc.Suggest(context.Background(), dto.SuggestTemplatesRequest{Question: "q"})
		if !errors.Is(err, embedErr) {
			t.Fatalf("error = %v, want %v", err, embedErr)
		}
	})
}
