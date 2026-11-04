package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ch4rniauski/support-brain/internal/knowledge/dto"
	"github.com/google/uuid"
)

type fakeClusterTemplateStore struct {
	templates map[uuid.UUID]uuid.UUID
	err       error
	lastIDs   []uuid.UUID
}

func (f *fakeClusterTemplateStore) FindApprovedTemplateIDsByQAPairIDs(
	ctx context.Context,
	qaPairIDs []uuid.UUID,
) (map[uuid.UUID]uuid.UUID, error) {
	f.lastIDs = append([]uuid.UUID(nil), qaPairIDs...)
	if f.err != nil {
		return nil, f.err
	}
	if f.templates == nil {
		return map[uuid.UUID]uuid.UUID{}, nil
	}

	result := make(map[uuid.UUID]uuid.UUID, len(qaPairIDs))
	for _, id := range qaPairIDs {
		if templateID, ok := f.templates[id]; ok {
			result[id] = templateID
		}
	}
	return result, nil
}

func TestClusterService_FindTemplatesByQAPairIDs(t *testing.T) {
	t.Parallel()

	qa1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	qa2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	qa3 := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	templateID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	t.Run("returns approved active template", func(t *testing.T) {
		t.Parallel()

		store := &fakeClusterTemplateStore{
			templates: map[uuid.UUID]uuid.UUID{
				qa1: templateID,
			},
		}
		svc := NewClusterService(store)

		resp, err := svc.FindTemplatesByQAPairIDs(context.Background(), dto.FindTemplatesByQAPairsRequest{
			QAPairIDs: []uuid.UUID{qa1},
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(resp.Items) != 1 {
			t.Fatalf("items len = %d, want 1", len(resp.Items))
		}
		if resp.Items[0].QAPairID != qa1 {
			t.Errorf("qa_pair_id = %v, want %v", resp.Items[0].QAPairID, qa1)
		}
		if resp.Items[0].TemplateID == nil || *resp.Items[0].TemplateID != templateID {
			t.Errorf("template_id = %v, want %v", resp.Items[0].TemplateID, templateID)
		}
	})

	t.Run("null when not found", func(t *testing.T) {
		t.Parallel()

		svc := NewClusterService(&fakeClusterTemplateStore{})

		resp, err := svc.FindTemplatesByQAPairIDs(context.Background(), dto.FindTemplatesByQAPairsRequest{
			QAPairIDs: []uuid.UUID{qa1},
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if resp.Items[0].TemplateID != nil {
			t.Errorf("template_id = %v, want nil", resp.Items[0].TemplateID)
		}
	})

	t.Run("dedupes and preserves order", func(t *testing.T) {
		t.Parallel()

		store := &fakeClusterTemplateStore{
			templates: map[uuid.UUID]uuid.UUID{
				qa1: templateID,
			},
		}
		svc := NewClusterService(store)

		resp, err := svc.FindTemplatesByQAPairIDs(context.Background(), dto.FindTemplatesByQAPairsRequest{
			QAPairIDs: []uuid.UUID{qa2, qa1, qa2, qa3, qa1},
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(resp.Items) != 3 {
			t.Fatalf("items len = %d, want 3", len(resp.Items))
		}
		wantOrder := []uuid.UUID{qa2, qa1, qa3}
		for i, want := range wantOrder {
			if resp.Items[i].QAPairID != want {
				t.Errorf("items[%d].qa_pair_id = %v, want %v", i, resp.Items[i].QAPairID, want)
			}
		}
		if len(store.lastIDs) != 3 || store.lastIDs[0] != qa2 || store.lastIDs[1] != qa1 || store.lastIDs[2] != qa3 {
			t.Errorf("store called with %v, want %v", store.lastIDs, wantOrder)
		}
		if resp.Items[0].TemplateID != nil {
			t.Error("qa2 should have null template_id")
		}
		if resp.Items[1].TemplateID == nil || *resp.Items[1].TemplateID != templateID {
			t.Error("qa1 should have approved template_id")
		}
		if resp.Items[2].TemplateID != nil {
			t.Error("qa3 should have null template_id")
		}
	})

	t.Run("invalid empty list", func(t *testing.T) {
		t.Parallel()

		store := &fakeClusterTemplateStore{}
		svc := NewClusterService(store)

		resp, err := svc.FindTemplatesByQAPairIDs(context.Background(), dto.FindTemplatesByQAPairsRequest{
			QAPairIDs: nil,
		})
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidInput)
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
		if store.lastIDs != nil {
			t.Error("store must not be called on invalid input")
		}
	})

	t.Run("store fails", func(t *testing.T) {
		t.Parallel()

		storeErr := errors.New("db down")
		svc := NewClusterService(&fakeClusterTemplateStore{err: storeErr})

		resp, err := svc.FindTemplatesByQAPairIDs(context.Background(), dto.FindTemplatesByQAPairsRequest{
			QAPairIDs: []uuid.UUID{qa1},
		})
		if !errors.Is(err, storeErr) {
			t.Fatalf("error = %v, want %v", err, storeErr)
		}
		if resp != nil {
			t.Errorf("response = %#v, want nil", resp)
		}
	})
}
