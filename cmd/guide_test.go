package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func setupGuideTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	tddDir := filepath.Join(tmpDir, ".tdd")
	specDir := filepath.Join(tddDir, "specs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatal(err)
	}
	configContent := "specDir: .tdd/specs\ntestDir: tests\nrunner: vitest\nfileNamePattern: \"req-{{id}}-{{slug}}.test.ts\"\n"
	if err := os.WriteFile(filepath.Join(tddDir, "config.yml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "User registration"},
		{ID: "REQ-002", Title: "Login", Depends: []string{"REQ-001"}},
	}
	for _, s := range specs {
		path := filepath.Join(specDir, s.ID+".yml")
		if err := spec.Save(path, s); err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir
}

func TestGuideCmd_GeneratesFile(t *testing.T) {
	tmpDir := setupGuideTestDir(t)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"guide"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "wrote .tdd/GUIDE.md") {
		t.Errorf("expected 'wrote .tdd/GUIDE.md', got:\n%s", output)
	}

	// Verify file content
	data, err := os.ReadFile(filepath.Join(tmpDir, ".tdd", "GUIDE.md"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# Implementation Guide") {
		t.Errorf("expected guide header in file, got:\n%s", content)
	}
	if !strings.Contains(content, "REQ-001") || !strings.Contains(content, "REQ-002") {
		t.Errorf("expected REQ-IDs in guide, got:\n%s", content)
	}
}

func TestGuideCmd_CycleError(t *testing.T) {
	tmpDir := t.TempDir()

	tddDir := filepath.Join(tmpDir, ".tdd")
	specDir := filepath.Join(tddDir, "specs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatal(err)
	}
	configContent := "specDir: .tdd/specs\ntestDir: tests\nrunner: vitest\nfileNamePattern: \"req-{{id}}-{{slug}}.test.ts\"\n"
	if err := os.WriteFile(filepath.Join(tddDir, "config.yml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A", Depends: []string{"REQ-002"}},
		{ID: "REQ-002", Title: "B", Depends: []string{"REQ-001"}},
	}
	for _, s := range specs {
		path := filepath.Join(specDir, s.ID+".yml")
		if err := spec.Save(path, s); err != nil {
			t.Fatal(err)
		}
	}

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"guide"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for cycle, got nil")
	}
}

func TestGuideCmd_CustomOutput(t *testing.T) {
	tmpDir := setupGuideTestDir(t)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	outputPath := filepath.Join(tmpDir, "custom-guide.md")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"guide", "--output", outputPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output file at %s, got error: %v", outputPath, err)
	}
}
