package trace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func TestCountTestsByReq(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sample.test.ts")
	content := `describe("REQ-001: Title", () => {
  it("REQ-001 E1: first", () => {})
  it("REQ-002 E1: second", () => {})
})`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	counts, err := CountTestsByReq(tmpDir)
	if err != nil {
		t.Fatalf("CountTestsByReq error: %v", err)
	}

	if counts["REQ-001"] != 1 {
		t.Fatalf("REQ-001 count = %d, want 1", counts["REQ-001"])
	}
	if counts["REQ-002"] != 1 {
		t.Fatalf("REQ-002 count = %d, want 1", counts["REQ-002"])
	}
}

func TestBuildReportStatus(t *testing.T) {
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A", Examples: []spec.Example{{ID: "E1", Given: "a", When: "b", Then: "c"}}},
		{ID: "REQ-002", Title: "B", Examples: []spec.Example{}},
		{ID: "REQ-003", Title: "C", Examples: []spec.Example{{ID: "E1", Given: "a", When: "b", Then: "c"}, {ID: "E2", Given: "a", When: "b", Then: "c"}}},
	}
	counts := map[string]int{
		"REQ-001": 1,
		"REQ-003": 1,
	}

	report := BuildReport(specs, counts)
	if len(report.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(report.Items))
	}

	statuses := map[string]string{}
	for _, item := range report.Items {
		statuses[item.ID] = item.Status
	}

	if statuses["REQ-001"] != "OK" {
		t.Fatalf("REQ-001 status = %q, want OK", statuses["REQ-001"])
	}
	if statuses["REQ-002"] != "MISSING" {
		t.Fatalf("REQ-002 status = %q, want MISSING", statuses["REQ-002"])
	}
	if statuses["REQ-003"] != "PARTIAL" {
		t.Fatalf("REQ-003 status = %q, want PARTIAL", statuses["REQ-003"])
	}
}
