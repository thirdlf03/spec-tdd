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

	result := TopologicalSort(specs)

	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if len(result.Order) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.Order))
	}
	if result.Order[0].ID != "REQ-001" || result.Order[1].ID != "REQ-002" || result.Order[2].ID != "REQ-003" {
		t.Fatalf("expected [REQ-001, REQ-002, REQ-003], got [%s, %s, %s]",
			result.Order[0].ID, result.Order[1].ID, result.Order[2].ID)
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

	result := TopologicalSort(specs)

	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if len(result.Order) != 4 {
		t.Fatalf("expected 4 items, got %d", len(result.Order))
	}
	if result.Order[0].ID != "REQ-001" {
		t.Fatalf("expected REQ-001 first, got %s", result.Order[0].ID)
	}
	if result.Order[3].ID != "REQ-004" {
		t.Fatalf("expected REQ-004 last, got %s", result.Order[3].ID)
	}
}

func TestTopologicalSort_AllIndependent(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-003", Title: "C"},
		{ID: "REQ-001", Title: "A"},
		{ID: "REQ-002", Title: "B"},
	}

	result := TopologicalSort(specs)

	// Should be sorted by REQ-ID number
	if result.Order[0].ID != "REQ-001" || result.Order[1].ID != "REQ-002" || result.Order[2].ID != "REQ-003" {
		t.Fatalf("expected numeric order, got [%s, %s, %s]",
			result.Order[0].ID, result.Order[1].ID, result.Order[2].ID)
	}
}

func TestTopologicalSort_Cycle(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A", Depends: []string{"REQ-002"}},
		{ID: "REQ-002", Title: "B", Depends: []string{"REQ-001"}},
	}

	result := TopologicalSort(specs)

	if len(result.Order) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Order))
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warnings for cycle, got none")
	}
	if !strings.Contains(result.Warnings[0], "cycle") {
		t.Fatalf("expected cycle warning, got: %s", result.Warnings[0])
	}
}

func TestTopologicalSort_CycleWithNonCycleNodes(t *testing.T) {
	// REQ-001 (root) -> REQ-002 -> REQ-003 (cycle with REQ-004)
	//                            -> REQ-004 -> REQ-003
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A"},
		{ID: "REQ-002", Title: "B", Depends: []string{"REQ-001"}},
		{ID: "REQ-003", Title: "C", Depends: []string{"REQ-002", "REQ-004"}},
		{ID: "REQ-004", Title: "D", Depends: []string{"REQ-003"}},
	}

	result := TopologicalSort(specs)

	if len(result.Order) != 4 {
		t.Fatalf("expected 4 items, got %d", len(result.Order))
	}
	// Non-cycle nodes should come first in correct order
	if result.Order[0].ID != "REQ-001" {
		t.Fatalf("expected REQ-001 first, got %s", result.Order[0].ID)
	}
	if result.Order[1].ID != "REQ-002" {
		t.Fatalf("expected REQ-002 second, got %s", result.Order[1].ID)
	}
	// Cyclic nodes should be appended after
	if result.Order[2].ID != "REQ-003" || result.Order[3].ID != "REQ-004" {
		t.Fatalf("expected cyclic nodes [REQ-003, REQ-004] at end, got [%s, %s]",
			result.Order[2].ID, result.Order[3].ID)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warnings for cycle, got none")
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

	result := TopologicalSort(specs)
	depBy := BuildDependedByMap(specs)

	data := GuideData{
		Specs:         specs,
		Order:         result.Order,
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

	// No warnings section when no cycles
	if strings.Contains(guide, "## Warnings") {
		t.Errorf("expected no Warnings section, got:\n%s", guide)
	}
}

func TestRenderGuide_NoDeps(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "Feature A"},
	}

	result := TopologicalSort(specs)
	data := GuideData{
		Specs:         specs,
		Order:         result.Order,
		Prerequisites: make(map[string][]string),
		DependedBy:    BuildDependedByMap(specs),
	}

	guide := RenderGuide(data, "tests", "req-{{id}}-{{slug}}.test.ts")

	if !strings.Contains(guide, "No dependencies detected") {
		t.Errorf("expected 'No dependencies detected', got:\n%s", guide)
	}
}

func TestRenderGuide_WithWarnings(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A", Depends: []string{"REQ-002"}},
		{ID: "REQ-002", Title: "B", Depends: []string{"REQ-001"}},
	}

	result := TopologicalSort(specs)
	depBy := BuildDependedByMap(specs)

	data := GuideData{
		Specs:         specs,
		Order:         result.Order,
		Prerequisites: make(map[string][]string),
		DependedBy:    depBy,
		Warnings:      result.Warnings,
	}
	for _, s := range specs {
		data.Prerequisites[s.ID] = s.Depends
	}

	guide := RenderGuide(data, "tests", "req-{{id}}-{{slug}}.test.ts")

	if !strings.Contains(guide, "## Warnings") {
		t.Errorf("expected Warnings section in guide, got:\n%s", guide)
	}
	if !strings.Contains(guide, "cycle") {
		t.Errorf("expected cycle info in warnings, got:\n%s", guide)
	}
}
