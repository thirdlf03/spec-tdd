package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func TestExampleAddCommand(t *testing.T) {
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
	if err := spec.Save(path, &spec.Spec{ID: "REQ-001", Title: "Title"}); err != nil {
		t.Fatalf("save spec error: %v", err)
	}

	exampleReqID = "REQ-001"
	exampleGiven = "given"
	exampleWhen = "when"
	exampleThen = "then"

	if err := exampleAddCmd.RunE(exampleAddCmd, []string{}); err != nil {
		t.Fatalf("exampleAddCmd error: %v", err)
	}

	loaded, err := spec.Load(path)
	if err != nil {
		t.Fatalf("load spec error: %v", err)
	}
	if len(loaded.Examples) != 1 {
		t.Fatalf("expected 1 example, got %d", len(loaded.Examples))
	}
	if loaded.Examples[0].ID != "E1" {
		t.Fatalf("expected example ID E1, got %q", loaded.Examples[0].ID)
	}
}
