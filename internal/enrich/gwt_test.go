package enrich

import (
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func TestMergeExamples(t *testing.T) {
	t.Run("existing GWT takes priority over enriched", func(t *testing.T) {
		existing := []spec.Example{
			{Given: "既存条件", When: "既存操作", Then: "既存結果"},
		}
		enriched := []spec.Example{
			{Given: "LLM条件", When: "LLM操作", Then: "LLM結果"},
		}

		result := MergeExamples(existing, enriched)
		if len(result) != 1 {
			t.Fatalf("expected 1 example, got %d", len(result))
		}
		if result[0].Given != "既存条件" {
			t.Errorf("Given = %q, want '既存条件'", result[0].Given)
		}
	})

	t.Run("uses enriched when no existing GWT", func(t *testing.T) {
		var existing []spec.Example
		enriched := []spec.Example{
			{Given: "LLM条件", When: "LLM操作", Then: "LLM結果"},
		}

		result := MergeExamples(existing, enriched)
		if len(result) != 1 {
			t.Fatalf("expected 1 example, got %d", len(result))
		}
		if result[0].Given != "LLM条件" {
			t.Errorf("Given = %q, want 'LLM条件'", result[0].Given)
		}
	})

	t.Run("assigns sequential Example IDs", func(t *testing.T) {
		examples := []spec.Example{
			{Given: "a", When: "b", Then: "c"},
			{Given: "d", When: "e", Then: "f"},
		}

		result := MergeExamples(nil, examples)
		if len(result) != 2 {
			t.Fatalf("expected 2 examples, got %d", len(result))
		}
		if result[0].ID != "E1" {
			t.Errorf("result[0].ID = %q, want E1", result[0].ID)
		}
		if result[1].ID != "E2" {
			t.Errorf("result[1].ID = %q, want E2", result[1].ID)
		}
	})

	t.Run("empty both returns nil", func(t *testing.T) {
		result := MergeExamples(nil, nil)
		if len(result) != 0 {
			t.Errorf("expected 0 examples, got %d", len(result))
		}
	})
}
