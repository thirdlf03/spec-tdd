package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func setupImportTestDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir error: %v", err)
	}

	// Create .tdd/specs directory
	specDir := filepath.Join(tmpDir, ".tdd", "specs")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}

	// Create kire output directory with JSONL and segments
	kireDir := filepath.Join(tmpDir, ".kire")
	if err := os.MkdirAll(kireDir, 0755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}

	jsonl := `{"segment_id":"seg-001","heading_path":["Doc","Login"],"file_path":"seg-001.md"}
{"segment_id":"seg-002","heading_path":["Doc","Auth","Logout"],"file_path":"seg-002.md"}`
	if err := os.WriteFile(filepath.Join(kireDir, "metadata.jsonl"), []byte(jsonl), 0644); err != nil {
		t.Fatalf("write jsonl error: %v", err)
	}

	seg1 := "# Login\n\nREQ-001\n\n- Given: ユーザーが存在する\n- When: ログインする\n- Then: トークン返却\n\nセッション期限は？\n"
	seg2 := "# Logout\n\nログアウト機能。\n"
	if err := os.WriteFile(filepath.Join(kireDir, "seg-001.md"), []byte(seg1), 0644); err != nil {
		t.Fatalf("write seg1 error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(kireDir, "seg-002.md"), []byte(seg2), 0644); err != nil {
		t.Fatalf("write seg2 error: %v", err)
	}

	return tmpDir
}

func TestImportKireCommand(t *testing.T) {
	t.Run("generates spec YAML files from kire output", func(t *testing.T) {
		tmpDir := setupImportTestDir(t)

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "2 created") {
			t.Errorf("expected '2 created' in output, got: %s", output)
		}

		// Verify files were created
		specDir := filepath.Join(tmpDir, ".tdd", "specs")
		s1, err := spec.Load(filepath.Join(specDir, "REQ-001.yml"))
		if err != nil {
			t.Fatalf("Load REQ-001 error: %v", err)
		}
		if s1.Title != "Login" {
			t.Errorf("REQ-001 Title = %q, want %q", s1.Title, "Login")
		}
		if len(s1.Examples) != 1 {
			t.Errorf("REQ-001 examples = %d, want 1", len(s1.Examples))
		}
		if len(s1.Questions) != 1 {
			t.Errorf("REQ-001 questions = %d, want 1", len(s1.Questions))
		}
		if s1.Source.SegmentID != "seg-001" {
			t.Errorf("REQ-001 Source.SegmentID = %q, want %q", s1.Source.SegmentID, "seg-001")
		}

		// REQ-002 was auto-assigned
		s2, err := spec.Load(filepath.Join(specDir, "REQ-002.yml"))
		if err != nil {
			t.Fatalf("Load REQ-002 error: %v", err)
		}
		if s2.Title != "Logout" {
			t.Errorf("REQ-002 Title = %q, want %q", s2.Title, "Logout")
		}
	})

	t.Run("dry-run does not write files", func(t *testing.T) {
		tmpDir := setupImportTestDir(t)

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		// Set dry-run flag
		if err := importKireCmd.Flags().Set("dry-run", "true"); err != nil {
			t.Fatalf("set dry-run flag: %v", err)
		}
		defer func() {
			_ = importKireCmd.Flags().Set("dry-run", "false")
		}()

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "dry-run") || !strings.Contains(output, "REQ-001") {
			t.Errorf("expected dry-run preview output, got: %s", output)
		}

		// Files should not exist
		specDir := filepath.Join(tmpDir, ".tdd", "specs")
		if _, err := os.Stat(filepath.Join(specDir, "REQ-001.yml")); err == nil {
			t.Fatal("expected no file in dry-run mode")
		}
	})

	t.Run("skip existing files without force", func(t *testing.T) {
		tmpDir := setupImportTestDir(t)

		specDir := filepath.Join(tmpDir, ".tdd", "specs")
		if err := spec.Save(filepath.Join(specDir, "REQ-001.yml"), &spec.Spec{
			ID: "REQ-001", Title: "Existing",
		}); err != nil {
			t.Fatalf("save existing error: %v", err)
		}

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "1 skipped") {
			t.Errorf("expected '1 skipped' in output, got: %s", output)
		}

		// Verify existing file was NOT overwritten
		s, err := spec.Load(filepath.Join(specDir, "REQ-001.yml"))
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}
		if s.Title != "Existing" {
			t.Errorf("existing file was overwritten: Title = %q", s.Title)
		}
	})

	t.Run("force overwrites existing files", func(t *testing.T) {
		tmpDir := setupImportTestDir(t)

		specDir := filepath.Join(tmpDir, ".tdd", "specs")
		if err := spec.Save(filepath.Join(specDir, "REQ-001.yml"), &spec.Spec{
			ID: "REQ-001", Title: "Existing",
		}); err != nil {
			t.Fatalf("save existing error: %v", err)
		}

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.Flags().Set("force", "true"); err != nil {
			t.Fatalf("set force flag: %v", err)
		}
		defer func() {
			_ = importKireCmd.Flags().Set("force", "false")
		}()

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "1 overwritten") {
			t.Errorf("expected '1 overwritten' in output, got: %s", output)
		}

		// Verify file was overwritten
		s, err := spec.Load(filepath.Join(specDir, "REQ-001.yml"))
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}
		if s.Title != "Login" {
			t.Errorf("expected overwritten title 'Login', got %q", s.Title)
		}
	})

	t.Run("idempotent: second run skips all", func(t *testing.T) {
		setupImportTestDir(t)

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		// First run
		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("first run error: %v", err)
		}

		// Second run
		buf.Reset()
		importKireCmd.SetOut(&buf)
		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("second run error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "0 created") {
			t.Errorf("expected '0 created' on second run, got: %s", output)
		}
		if !strings.Contains(output, "2 skipped") {
			t.Errorf("expected '2 skipped' on second run, got: %s", output)
		}
	})
}
