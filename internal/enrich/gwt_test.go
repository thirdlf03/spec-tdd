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

func TestDeduplicateExamples(t *testing.T) {
	t.Run("removes exact duplicates", func(t *testing.T) {
		examples := []spec.Example{
			{ID: "E1", Given: "条件A", When: "操作A", Then: "結果A"},
			{ID: "E2", Given: "条件B", When: "操作B", Then: "結果B"},
			{ID: "E3", Given: "条件A", When: "操作A", Then: "結果A"},
		}

		result := DeduplicateExamples(examples)
		if len(result) != 2 {
			t.Fatalf("expected 2 examples, got %d", len(result))
		}
		if result[0].Given != "条件A" {
			t.Errorf("result[0].Given = %q, want '条件A'", result[0].Given)
		}
		if result[1].Given != "条件B" {
			t.Errorf("result[1].Given = %q, want '条件B'", result[1].Given)
		}
		// IDs re-numbered
		if result[0].ID != "E1" || result[1].ID != "E2" {
			t.Errorf("IDs = [%s, %s], want [E1, E2]", result[0].ID, result[1].ID)
		}
	})

	t.Run("removes duplicates with whitespace differences", func(t *testing.T) {
		examples := []spec.Example{
			{ID: "E1", Given: "条件A", When: "操作A", Then: "結果A"},
			{ID: "E2", Given: "  条件A  ", When: " 操作A ", Then: " 結果A "},
		}

		result := DeduplicateExamples(examples)
		if len(result) != 1 {
			t.Fatalf("expected 1 example, got %d", len(result))
		}
		// First occurrence is kept (original, not trimmed)
		if result[0].Given != "条件A" {
			t.Errorf("result[0].Given = %q, want '条件A'", result[0].Given)
		}
	})

	t.Run("removes duplicates with case differences", func(t *testing.T) {
		examples := []spec.Example{
			{ID: "E1", Given: "User exists", When: "POST /login", Then: "200 OK"},
			{ID: "E2", Given: "user exists", When: "post /login", Then: "200 ok"},
		}

		result := DeduplicateExamples(examples)
		if len(result) != 1 {
			t.Fatalf("expected 1 example, got %d", len(result))
		}
	})

	t.Run("no duplicates returns unchanged", func(t *testing.T) {
		examples := []spec.Example{
			{ID: "E1", Given: "条件A", When: "操作A", Then: "結果A"},
			{ID: "E2", Given: "条件B", When: "操作B", Then: "結果B"},
		}

		result := DeduplicateExamples(examples)
		if len(result) != 2 {
			t.Fatalf("expected 2 examples, got %d", len(result))
		}
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		result := DeduplicateExamples(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("single example returns as-is", func(t *testing.T) {
		examples := []spec.Example{
			{ID: "E1", Given: "条件A", When: "操作A", Then: "結果A"},
		}

		result := DeduplicateExamples(examples)
		if len(result) != 1 {
			t.Fatalf("expected 1 example, got %d", len(result))
		}
		if result[0].ID != "E1" {
			t.Errorf("ID = %q, want E1", result[0].ID)
		}
	})
}
