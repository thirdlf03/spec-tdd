package deps

import (
	"context"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func TestHeuristicDetector_ReqIDReference(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "User registration"},
		{ID: "REQ-002", Title: "Login", Description: "Requires REQ-001 user to exist"},
	}

	d := &HeuristicDetector{}
	results, err := d.Detect(context.Background(), specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	found := false
	for _, r := range results {
		if r.ID == "REQ-002" {
			found = true
			if len(r.Depends) != 1 || r.Depends[0] != "REQ-001" {
				t.Fatalf("expected REQ-002 to depend on REQ-001, got %v", r.Depends)
			}
		}
	}
	if !found {
		t.Fatal("expected result for REQ-002")
	}
}

func TestHeuristicDetector_NoSelfReference(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "Feature A", Description: "See REQ-001 for details"},
	}

	d := &HeuristicDetector{}
	results, err := d.Detect(context.Background(), specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, r := range results {
		if r.ID == "REQ-001" {
			for _, dep := range r.Depends {
				if dep == "REQ-001" {
					t.Fatal("should not have self-reference")
				}
			}
		}
	}
}

func TestHeuristicDetector_JapaneseKeywords(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "ユーザー登録"},
		{ID: "REQ-002", Title: "ログイン", Description: "REQ-001 を前提とする"},
	}

	d := &HeuristicDetector{}
	results, err := d.Detect(context.Background(), specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, r := range results {
		if r.ID == "REQ-002" {
			found = true
			hasDep := false
			for _, dep := range r.Depends {
				if dep == "REQ-001" {
					hasDep = true
				}
			}
			if !hasDep {
				t.Fatalf("expected REQ-002 to depend on REQ-001, got %v", r.Depends)
			}
		}
	}
	if !found {
		t.Fatal("expected result for REQ-002")
	}
}

func TestHeuristicDetector_NoDeps(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "Login"},
		{ID: "REQ-002", Title: "Cart"},
	}

	d := &HeuristicDetector{}
	results, err := d.Detect(context.Background(), specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected no results, got %d", len(results))
	}
}

func TestMockDetector(t *testing.T) {
	mock := &MockDetector{
		Results: []DepsResult{
			{ID: "REQ-001", Depends: []string{"REQ-002"}, Reason: "test"},
		},
	}

	results, err := mock.Detect(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "REQ-001" {
		t.Fatalf("unexpected results: %v", results)
	}
	if mock.CallCount != 1 {
		t.Fatalf("expected 1 call, got %d", mock.CallCount)
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		input string
		want  int // minimum expected keywords
	}{
		{"User registration flow", 2},  // "user" is 4 chars, "registration" is 12, "flow" is 4
		{"ログイン認証", 1},                   // Japanese word 6 runes
		{"a b c", 0},                    // all too short
	}

	for _, tt := range tests {
		kws := extractKeywords(tt.input)
		if len(kws) < tt.want {
			t.Errorf("extractKeywords(%q) = %v, want at least %d keywords", tt.input, kws, tt.want)
		}
	}
}

func TestSortReqIDs(t *testing.T) {
	ids := []string{"REQ-003", "REQ-001", "REQ-002"}
	sortReqIDs(ids)
	if ids[0] != "REQ-001" || ids[1] != "REQ-002" || ids[2] != "REQ-003" {
		t.Fatalf("expected sorted order, got %v", ids)
	}
}
