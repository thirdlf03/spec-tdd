package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/deps"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func setupDepsTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create .tdd/config.yml
	tddDir := filepath.Join(tmpDir, ".tdd")
	specDir := filepath.Join(tddDir, "specs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatal(err)
	}
	configContent := "specDir: .tdd/specs\ntestDir: tests\nrunner: vitest\nfileNamePattern: \"req-{{id}}-{{slug}}.test.ts\"\n"
	if err := os.WriteFile(filepath.Join(tddDir, "config.yml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create specs
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "User registration"},
		{ID: "REQ-002", Title: "Login"},
	}
	for _, s := range specs {
		path := filepath.Join(specDir, s.ID+".yml")
		if err := spec.Save(path, s); err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir
}

func TestDepsCmd_HeuristicMode(t *testing.T) {
	tmpDir := setupDepsTestDir(t)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to tmpDir: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("chdir back: %v", err)
		}
	}()

	oldDetector := testDepsDetector
	testDepsDetector = &deps.MockDetector{
		Results: []deps.DepsResult{
			{ID: "REQ-002", Depends: []string{"REQ-001"}, Reason: "test"},
		},
	}
	defer func() { testDepsDetector = oldDetector }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"deps"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "updated: REQ-002") {
		t.Errorf("expected update message, got:\n%s", output)
	}
	if !strings.Contains(output, "1 specs updated") {
		t.Errorf("expected summary, got:\n%s", output)
	}

	// Verify file was updated
	loaded, err := spec.Load(filepath.Join(tmpDir, ".tdd", "specs", "REQ-002.yml"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(loaded.Depends) != 1 || loaded.Depends[0] != "REQ-001" {
		t.Fatalf("expected Depends=[REQ-001], got %v", loaded.Depends)
	}
}

func TestDepsCmd_DryRun(t *testing.T) {
	tmpDir := setupDepsTestDir(t)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to tmpDir: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("chdir back: %v", err)
		}
	}()

	oldDetector := testDepsDetector
	testDepsDetector = &deps.MockDetector{
		Results: []deps.DepsResult{
			{ID: "REQ-002", Depends: []string{"REQ-001"}, Reason: "test"},
		},
	}
	defer func() { testDepsDetector = oldDetector }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"deps", "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected dry-run output, got:\n%s", output)
	}

	// File should NOT be updated
	loaded, err := spec.Load(filepath.Join(tmpDir, ".tdd", "specs", "REQ-002.yml"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(loaded.Depends) != 0 {
		t.Fatalf("expected no Depends in dry-run, got %v", loaded.Depends)
	}
}

func TestDepsCmd_CycleWarning(t *testing.T) {
	tmpDir := setupDepsTestDir(t)

	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to tmpDir: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("chdir back: %v", err)
		}
	}()

	oldDetector := testDepsDetector
	testDepsDetector = &deps.MockDetector{
		Results: []deps.DepsResult{
			{ID: "REQ-001", Depends: []string{"REQ-002"}, Reason: "cycle test"},
			{ID: "REQ-002", Depends: []string{"REQ-001"}, Reason: "cycle test"},
		},
	}
	defer func() { testDepsDetector = oldDetector }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"deps"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "warning:") && !strings.Contains(output, "cycle") {
		t.Errorf("expected cycle warning, got:\n%s", output)
	}
}
