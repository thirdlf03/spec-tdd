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

func TestValidate_Depends(t *testing.T) {
	tests := []struct {
		name    string
		spec    Spec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid depends",
			spec: Spec{ID: "REQ-001", Title: "A", Depends: []string{"REQ-002", "REQ-003"}},
		},
		{
			name: "empty depends is valid",
			spec: Spec{ID: "REQ-001", Title: "A"},
		},
		{
			name:    "invalid format",
			spec:    Spec{ID: "REQ-001", Title: "A", Depends: []string{"INVALID"}},
			wantErr: true,
			errMsg:  "must match REQ-###",
		},
		{
			name:    "self-reference",
			spec:    Spec{ID: "REQ-001", Title: "A", Depends: []string{"REQ-001"}},
			wantErr: true,
			errMsg:  "self-reference",
		},
		{
			name:    "duplicate",
			spec:    Spec{ID: "REQ-001", Title: "A", Depends: []string{"REQ-002", "REQ-002"}},
			wantErr: true,
			errMsg:  "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !searchString(err.Error(), tt.errMsg) {
					t.Fatalf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateDependsGraph(t *testing.T) {
	t.Run("valid DAG", func(t *testing.T) {
		specs := []*Spec{
			{ID: "REQ-001", Title: "A"},
			{ID: "REQ-002", Title: "B", Depends: []string{"REQ-001"}},
			{ID: "REQ-003", Title: "C", Depends: []string{"REQ-001", "REQ-002"}},
		}
		if err := ValidateDependsGraph(specs); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing reference", func(t *testing.T) {
		specs := []*Spec{
			{ID: "REQ-001", Title: "A", Depends: []string{"REQ-999"}},
		}
		err := ValidateDependsGraph(specs)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !searchString(err.Error(), "does not exist") {
			t.Fatalf("expected 'does not exist' error, got %q", err.Error())
		}
	})

	t.Run("cycle detected", func(t *testing.T) {
		specs := []*Spec{
			{ID: "REQ-001", Title: "A", Depends: []string{"REQ-002"}},
			{ID: "REQ-002", Title: "B", Depends: []string{"REQ-001"}},
		}
		err := ValidateDependsGraph(specs)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !searchString(err.Error(), "cycle") {
			t.Fatalf("expected 'cycle' error, got %q", err.Error())
		}
	})

	t.Run("no depends at all", func(t *testing.T) {
		specs := []*Spec{
			{ID: "REQ-001", Title: "A"},
			{ID: "REQ-002", Title: "B"},
		}
		if err := ValidateDependsGraph(specs); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestDepends_BackwardCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "REQ-001.yml")

	// Save spec without depends
	orig := &Spec{ID: "REQ-001", Title: "No depends"}
	if err := Save(path, orig); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	// YAML should not contain "depends:" when empty
	if searchString(string(data), "depends:") {
		t.Fatalf("YAML should not contain 'depends:' when empty, got:\n%s", string(data))
	}

	// Load and verify
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(loaded.Depends) != 0 {
		t.Fatalf("expected empty Depends, got %v", loaded.Depends)
	}
}

func TestSourceInfo_FilePath(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "REQ-001.yml")

	orig := &Spec{
		ID:    "REQ-001",
		Title: "With filepath",
		Source: SourceInfo{
			SegmentID:   "seg-001",
			HeadingPath: []string{"設計書", "認証"},
			FilePath:    "seg-0001.md",
		},
	}
	if err := Save(path, orig); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.Source.FilePath != "seg-0001.md" {
		t.Fatalf("FilePath = %q, want %q", loaded.Source.FilePath, "seg-0001.md")
	}
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
