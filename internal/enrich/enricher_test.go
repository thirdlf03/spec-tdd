package enrich

import (
	"context"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/kire"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

// verify MockBatchEnricher satisfies BatchEnricher interface
var _ BatchEnricher = (*MockBatchEnricher)(nil)

func TestMockBatchEnricher(t *testing.T) {
	t.Run("returns configured classify results", func(t *testing.T) {
		mock := &MockBatchEnricher{
			ClassifyResults: []BatchClassifyResult{
				{SegmentID: "seg-0001", Category: CategoryFunctionalRequirement, Title: "テスト"},
			},
		}

		results, err := mock.BatchClassify(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("results count = %d, want 1", len(results))
		}
		if mock.ClassifyCallCount != 1 {
			t.Errorf("ClassifyCallCount = %d, want 1", mock.ClassifyCallCount)
		}
	})

	t.Run("returns configured classify error", func(t *testing.T) {
		mock := &MockBatchEnricher{
			ClassifyErr: context.DeadlineExceeded,
		}

		_, err := mock.BatchClassify(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns configured example results", func(t *testing.T) {
		mock := &MockBatchEnricher{
			ExampleResults: []BatchExampleResult{
				{SegmentID: "seg-0001", Examples: []spec.Example{
					{Given: "a", When: "b", Then: "c"},
				}},
			},
		}

		results, err := mock.BatchGenerateExamples(context.Background(), nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("results count = %d, want 1", len(results))
		}
		if mock.ExampleCallCount != 1 {
			t.Errorf("ExampleCallCount = %d, want 1", mock.ExampleCallCount)
		}
	})

	t.Run("returns configured example error", func(t *testing.T) {
		mock := &MockBatchEnricher{
			ExampleErr: ErrBatchTruncated,
		}

		_, err := mock.BatchGenerateExamples(context.Background(), nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

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

		result, err := mock.Enrich(context.Background(), seg, nil)
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

		_, err := mock.Enrich(context.Background(), seg, nil)
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

		_, _ = mock.Enrich(context.Background(), seg1, nil)
		_, _ = mock.Enrich(context.Background(), seg2, nil)

		if len(mock.CalledWith) != 2 {
			t.Fatalf("CalledWith count = %d, want 2", len(mock.CalledWith))
		}
		if mock.CalledWith[0].Meta.SegmentID != "seg-0001" {
			t.Errorf("CalledWith[0] = %q, want seg-0001", mock.CalledWith[0].Meta.SegmentID)
		}
	})
}
