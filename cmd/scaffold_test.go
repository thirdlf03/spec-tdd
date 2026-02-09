package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func TestScaffoldCommand(t *testing.T) {
	tmpDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd error: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir error: %v", err)
	}

	specDir := filepath.Join(tmpDir, ".tdd", "specs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	path := filepath.Join(specDir, "REQ-001.yml")
	if err := spec.Save(path, &spec.Spec{
		ID:       "REQ-001",
		Title:    "Sample",
		Examples: []spec.Example{{ID: "E1", Given: "a", When: "b", Then: "c"}},
	}); err != nil {
		t.Fatalf("save spec error: %v", err)
	}

	scaffoldRunner = "vitest"
	scaffoldForce = false

	if err := scaffoldCmd.RunE(scaffoldCmd, []string{}); err != nil {
		t.Fatalf("scaffoldCmd error: %v", err)
	}

	generated := filepath.Join(tmpDir, "tests", "req-REQ-001-sample.test.ts")
	if _, err := os.Stat(generated); err != nil {
		t.Fatalf("expected generated test file, got %v", err)
	}

	if err := scaffoldCmd.RunE(scaffoldCmd, []string{}); err == nil {
		t.Fatal("expected error when scaffold overwrites without --force")
	}
}
