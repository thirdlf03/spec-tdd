package enrich

import (
	"context"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/kire"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func TestMockEnricher(t *testing.T) {
	t.Run("returns configured result", func(t *testing.T) {
		mock := &MockEnricher{
			Result: &EnrichResult{
				Category: CategoryFunctionalRequirement,
				ReqID:    "REQ-001",
				Title:    "ログイン機能",
				Examples: []spec.Example{
					{Given: "ユーザーが存在する", When: "ログインする", Then: "トークン返却"},
				},
			},
		}

		seg := &kire.Segment{
			Meta:    kire.SegmentMeta{SegmentID: "seg-0001"},
			Content: "# Login",
		}

		result, err := mock.Enrich(context.Background(), seg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Category != CategoryFunctionalRequirement {
			t.Errorf("Category = %q, want %q", result.Category, CategoryFunctionalRequirement)
		}
		if result.ReqID != "REQ-001" {
			t.Errorf("ReqID = %q, want %q", result.ReqID, "REQ-001")
		}
		if result.Title != "ログイン機能" {
			t.Errorf("Title = %q, want %q", result.Title, "ログイン機能")
		}
		if len(result.Examples) != 1 {
			t.Fatalf("Examples count = %d, want 1", len(result.Examples))
		}
	})

	t.Run("returns configured error", func(t *testing.T) {
		mock := &MockEnricher{
			Err: context.DeadlineExceeded,
		}

		seg := &kire.Segment{
			Meta:    kire.SegmentMeta{SegmentID: "seg-0001"},
			Content: "# Login",
		}

		_, err := mock.Enrich(context.Background(), seg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("records called segments", func(t *testing.T) {
		mock := &MockEnricher{
			Result: &EnrichResult{Category: CategoryOther, Title: "test"},
		}

		seg1 := &kire.Segment{Meta: kire.SegmentMeta{SegmentID: "seg-0001"}, Content: "a"}
		seg2 := &kire.Segment{Meta: kire.SegmentMeta{SegmentID: "seg-0002"}, Content: "b"}

		_, _ = mock.Enrich(context.Background(), seg1)
		_, _ = mock.Enrich(context.Background(), seg2)

		if len(mock.CalledWith) != 2 {
			t.Fatalf("CalledWith count = %d, want 2", len(mock.CalledWith))
		}
		if mock.CalledWith[0].Meta.SegmentID != "seg-0001" {
			t.Errorf("CalledWith[0] = %q, want seg-0001", mock.CalledWith[0].Meta.SegmentID)
		}
	})
}
