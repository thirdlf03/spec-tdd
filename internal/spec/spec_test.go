package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadSpec(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "REQ-001.yml")

	orig := &Spec{
		ID:    "REQ-001",
		Title: "Test title",
		Examples: []Example{
			{ID: "E1", Given: "a", When: "b", Then: "c"},
		},
	}

	if err := Save(path, orig); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.ID != orig.ID || loaded.Title != orig.Title {
		t.Fatalf("loaded spec mismatch")
	}
}

func TestNextReqID(t *testing.T) {
	mp := t.TempDir()
	if err := os.MkdirAll(mp, 0755); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}

	if err := Save(filepath.Join(mp, "REQ-001.yml"), &Spec{ID: "REQ-001", Title: "A"}); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if err := Save(filepath.Join(mp, "REQ-010.yml"), &Spec{ID: "REQ-010", Title: "B"}); err != nil {
		t.Fatalf("save error: %v", err)
	}

	id, err := NextReqID(mp)
	if err != nil {
		t.Fatalf("NextReqID error: %v", err)
	}
	if id != "REQ-011" {
		t.Fatalf("NextReqID = %q, want %q", id, "REQ-011")
	}
}

func TestNextExampleID(t *testing.T) {
	s := &Spec{ID: "REQ-001", Title: "A", Examples: []Example{{ID: "E1", Given: "a", When: "b", Then: "c"}}}
	if next := NextExampleID(s); next != "E2" {
		t.Fatalf("NextExampleID = %q, want %q", next, "E2")
	}
}

func TestSourceInfo_BackwardCompatibility(t *testing.T) {
	t.Run("load spec without source field", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "REQ-001.yml")

		// Source フィールドなしの Spec を保存
		orig := &Spec{
			ID:    "REQ-001",
			Title: "No source",
		}
		if err := Save(path, orig); err != nil {
			t.Fatalf("Save error: %v", err)
		}

		loaded, err := Load(path)
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}

		// Source はゼロ値であるべき
		if loaded.Source.SegmentID != "" {
			t.Fatalf("expected empty SegmentID, got %q", loaded.Source.SegmentID)
		}
		if len(loaded.Source.HeadingPath) != 0 {
			t.Fatalf("expected empty HeadingPath, got %v", loaded.Source.HeadingPath)
		}
	})

	t.Run("save and load spec with source field", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "REQ-002.yml")

		orig := &Spec{
			ID:    "REQ-002",
			Title: "With source",
			Source: SourceInfo{
				SegmentID:   "seg-001",
				HeadingPath: []string{"設計書", "認証", "ログイン"},
			},
		}
		if err := Save(path, orig); err != nil {
			t.Fatalf("Save error: %v", err)
		}

		loaded, err := Load(path)
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}

		if loaded.Source.SegmentID != "seg-001" {
			t.Fatalf("SegmentID = %q, want %q", loaded.Source.SegmentID, "seg-001")
		}
		if len(loaded.Source.HeadingPath) != 3 || loaded.Source.HeadingPath[2] != "ログイン" {
			t.Fatalf("HeadingPath = %v, want [設計書 認証 ログイン]", loaded.Source.HeadingPath)
		}
	})

	t.Run("source omitted in YAML when zero value", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "REQ-003.yml")

		orig := &Spec{
			ID:    "REQ-003",
			Title: "No source field",
		}
		if err := Save(path, orig); err != nil {
			t.Fatalf("Save error: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile error: %v", err)
		}

		content := string(data)
		if contains(content, "source:") {
			t.Fatalf("YAML should not contain 'source:' when SourceInfo is zero value, got:\n%s", content)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
