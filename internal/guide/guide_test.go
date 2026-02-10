package guide

import (
	"strings"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func TestTopologicalSort_Linear(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-003", Title: "C", Depends: []string{"REQ-002"}},
		{ID: "REQ-001", Title: "A"},
		{ID: "REQ-002", Title: "B", Depends: []string{"REQ-001"}},
	}

	order, err := TopologicalSort(specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 items, got %d", len(order))
	}
	if order[0].ID != "REQ-001" || order[1].ID != "REQ-002" || order[2].ID != "REQ-003" {
		t.Fatalf("expected [REQ-001, REQ-002, REQ-003], got [%s, %s, %s]",
			order[0].ID, order[1].ID, order[2].ID)
	}
}

func TestTopologicalSort_Diamond(t *testing.T) {
	// REQ-001 -> REQ-002 \
	//                      -> REQ-004
	// REQ-001 -> REQ-003 /
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A"},
		{ID: "REQ-002", Title: "B", Depends: []string{"REQ-001"}},
		{ID: "REQ-003", Title: "C", Depends: []string{"REQ-001"}},
		{ID: "REQ-004", Title: "D", Depends: []string{"REQ-002", "REQ-003"}},
	}

	order, err := TopologicalSort(specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 4 {
		t.Fatalf("expected 4 items, got %d", len(order))
	}
	if order[0].ID != "REQ-001" {
		t.Fatalf("expected REQ-001 first, got %s", order[0].ID)
	}
	if order[3].ID != "REQ-004" {
		t.Fatalf("expected REQ-004 last, got %s", order[3].ID)
	}
}

func TestTopologicalSort_AllIndependent(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-003", Title: "C"},
		{ID: "REQ-001", Title: "A"},
		{ID: "REQ-002", Title: "B"},
	}

	order, err := TopologicalSort(specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be sorted by REQ-ID number
	if order[0].ID != "REQ-001" || order[1].ID != "REQ-002" || order[2].ID != "REQ-003" {
		t.Fatalf("expected numeric order, got [%s, %s, %s]",
			order[0].ID, order[1].ID, order[2].ID)
	}
}

func TestTopologicalSort_Cycle(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A", Depends: []string{"REQ-002"}},
		{ID: "REQ-002", Title: "B", Depends: []string{"REQ-001"}},
	}

	_, err := TopologicalSort(specs)
	if err == nil {
		t.Fatal("expected error for cycle, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got: %v", err)
	}
}

func TestBuildDependedByMap(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A"},
		{ID: "REQ-002", Title: "B", Depends: []string{"REQ-001"}},
		{ID: "REQ-003", Title: "C", Depends: []string{"REQ-001"}},
	}

	depBy := BuildDependedByMap(specs)

	if len(depBy["REQ-001"]) != 2 {
		t.Fatalf("expected REQ-001 to be depended by 2, got %d", len(depBy["REQ-001"]))
	}
	if len(depBy["REQ-002"]) != 0 {
		t.Fatalf("expected REQ-002 to have no dependents, got %d", len(depBy["REQ-002"]))
	}
}

func TestRenderGuide(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "User registration", Examples: []spec.Example{
			{ID: "E1", Given: "valid data", When: "POST /users", Then: "201"},
		}},
		{ID: "REQ-002", Title: "Login", Depends: []string{"REQ-001"}, Examples: []spec.Example{
			{ID: "E1", Given: "registered user", When: "POST /login", Then: "200"},
		}},
	}

	order, err := TopologicalSort(specs)
	if err != nil {
		t.Fatalf("TopologicalSort error: %v", err)
	}

	depBy := BuildDependedByMap(specs)

	data := GuideData{
		Specs:         specs,
		Order:         order,
		Prerequisites: make(map[string][]string),
		DependedBy:    depBy,
	}
	for _, s := range specs {
		data.Prerequisites[s.ID] = s.Depends
	}

	guide := RenderGuide(data, "tests", "req-{{id}}-{{slug}}.test.ts")

	// Check sections
	checks := []struct {
		label   string
		content string
	}{
		{"title", "# Implementation Guide"},
		{"total", "Total requirements: 2"},
		{"prerequisites", "## Prerequisites"},
		{"dependency graph", "## Dependency Graph"},
		{"implementation order", "## Implementation Order"},
		{"feature details", "## Feature Details"},
		{"REQ-001 in order", "1. **REQ-001**: User registration"},
		{"REQ-002 in order", "2. **REQ-002**: Login"},
		{"depends on", "**Depends on:** REQ-001"},
		{"required by", "**Required by:** REQ-002"},
		{"test file REQ-001", "tests/req-REQ-001-user-registration.test.ts"},
		{"test file REQ-002", "tests/req-REQ-002-login.test.ts"},
		{"example", "E1: Given valid data"},
	}

	for _, c := range checks {
		if !strings.Contains(guide, c.content) {
			t.Errorf("expected %s (%q) in guide, got:\n%s", c.label, c.content, guide)
		}
	}
}

func TestRenderGuide_NoDeps(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "Feature A"},
	}

	order, _ := TopologicalSort(specs)
	data := GuideData{
		Specs:         specs,
		Order:         order,
		Prerequisites: make(map[string][]string),
		DependedBy:    BuildDependedByMap(specs),
	}

	guide := RenderGuide(data, "tests", "req-{{id}}-{{slug}}.test.ts")

	if !strings.Contains(guide, "No dependencies detected") {
		t.Errorf("expected 'No dependencies detected', got:\n%s", guide)
	}
}
