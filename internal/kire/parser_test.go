package kire

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseJSONL(t *testing.T) {
	t.Run("parse valid JSONL and sort by segment_id", func(t *testing.T) {
		tmpDir := t.TempDir()
		jsonlPath := filepath.Join(tmpDir, "metadata.jsonl")

		content := `{"segment_id":"seg-003","heading_path":["Doc","Auth","Login"],"file_path":"seg-003.md"}
{"segment_id":"seg-001","heading_path":["Doc","Intro"],"file_path":"seg-001.md"}
{"segment_id":"seg-002","heading_path":["Doc","Auth"],"file_path":"seg-002.md"}`

		if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		metas, err := ParseJSONL(jsonlPath)
		if err != nil {
			t.Fatalf("ParseJSONL error: %v", err)
		}

		if len(metas) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(metas))
		}

		// Must be sorted by segment_id ascending
		if metas[0].SegmentID != "seg-001" {
			t.Errorf("metas[0].SegmentID = %q, want %q", metas[0].SegmentID, "seg-001")
		}
		if metas[1].SegmentID != "seg-002" {
			t.Errorf("metas[1].SegmentID = %q, want %q", metas[1].SegmentID, "seg-002")
		}
		if metas[2].SegmentID != "seg-003" {
			t.Errorf("metas[2].SegmentID = %q, want %q", metas[2].SegmentID, "seg-003")
		}

		// Verify fields are parsed correctly
		if metas[2].FilePath != "seg-003.md" {
			t.Errorf("metas[2].FilePath = %q, want %q", metas[2].FilePath, "seg-003.md")
		}
		if len(metas[2].HeadingPath) != 3 || metas[2].HeadingPath[2] != "Login" {
			t.Errorf("metas[2].HeadingPath = %v, want [Doc Auth Login]", metas[2].HeadingPath)
		}
	})

	t.Run("file not found returns error with path", func(t *testing.T) {
		_, err := ParseJSONL("/nonexistent/metadata.jsonl")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
		if !strings.Contains(err.Error(), "/nonexistent/metadata.jsonl") {
			t.Errorf("error should contain file path, got: %v", err)
		}
	})

	t.Run("invalid JSON line returns error with line number", func(t *testing.T) {
		tmpDir := t.TempDir()
		jsonlPath := filepath.Join(tmpDir, "metadata.jsonl")

		content := `{"segment_id":"seg-001","heading_path":["Doc"],"file_path":"seg-001.md"}
{invalid json}
{"segment_id":"seg-003","heading_path":["Doc"],"file_path":"seg-003.md"}`

		if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		_, err := ParseJSONL(jsonlPath)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "2") {
			t.Errorf("error should contain line number 2, got: %v", err)
		}
	})

	t.Run("empty file returns empty slice", func(t *testing.T) {
		tmpDir := t.TempDir()
		jsonlPath := filepath.Join(tmpDir, "metadata.jsonl")

		if err := os.WriteFile(jsonlPath, []byte(""), 0644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		metas, err := ParseJSONL(jsonlPath)
		if err != nil {
			t.Fatalf("ParseJSONL error: %v", err)
		}
		if len(metas) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(metas))
		}
	})

	t.Run("file with blank lines skips them", func(t *testing.T) {
		tmpDir := t.TempDir()
		jsonlPath := filepath.Join(tmpDir, "metadata.jsonl")

		content := `{"segment_id":"seg-001","heading_path":["Doc"],"file_path":"seg-001.md"}

{"segment_id":"seg-002","heading_path":["Doc"],"file_path":"seg-002.md"}
`

		if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		metas, err := ParseJSONL(jsonlPath)
		if err != nil {
			t.Fatalf("ParseJSONL error: %v", err)
		}
		if len(metas) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(metas))
		}
	})
}

func TestReadSegment(t *testing.T) {
	t.Run("read valid segment file", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdContent := "# Login\n\nUsers can log in with credentials.\n"
		if err := os.WriteFile(filepath.Join(tmpDir, "seg-001.md"), []byte(mdContent), 0644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		meta := SegmentMeta{
			SegmentID:   "seg-001",
			HeadingPath: []string{"Doc", "Login"},
			FilePath:    "seg-001.md",
		}

		seg, err := ReadSegment(tmpDir, meta)
		if err != nil {
			t.Fatalf("ReadSegment error: %v", err)
		}
		if seg == nil {
			t.Fatal("expected non-nil segment")
		}
		if seg.Content != mdContent {
			t.Errorf("Content = %q, want %q", seg.Content, mdContent)
		}
		if seg.Meta.SegmentID != "seg-001" {
			t.Errorf("Meta.SegmentID = %q, want %q", seg.Meta.SegmentID, "seg-001")
		}
	})

	t.Run("file not found returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		meta := SegmentMeta{
			SegmentID:   "seg-999",
			HeadingPath: []string{"Doc"},
			FilePath:    "nonexistent.md",
		}

		seg, err := ReadSegment(tmpDir, meta)
		if err != nil {
			t.Fatalf("ReadSegment should not return error for missing file, got: %v", err)
		}
		if seg != nil {
			t.Fatal("expected nil segment for missing file")
		}
	})

	t.Run("extract kire context comment", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdContent := "<!-- kire: 設計書 > 認証 > ログイン -->\n\n# Login\n\nContent here.\n"
		if err := os.WriteFile(filepath.Join(tmpDir, "seg-001.md"), []byte(mdContent), 0644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		meta := SegmentMeta{
			SegmentID:   "seg-001",
			HeadingPath: []string{"Doc", "Login"},
			FilePath:    "seg-001.md",
		}

		seg, err := ReadSegment(tmpDir, meta)
		if err != nil {
			t.Fatalf("ReadSegment error: %v", err)
		}
		if seg.Context != "設計書 > 認証 > ログイン" {
			t.Errorf("Context = %q, want %q", seg.Context, "設計書 > 認証 > ログイン")
		}
	})

	t.Run("no context comment returns empty context", func(t *testing.T) {
		tmpDir := t.TempDir()
		mdContent := "# Login\n\nNo context comment here.\n"
		if err := os.WriteFile(filepath.Join(tmpDir, "seg-001.md"), []byte(mdContent), 0644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		meta := SegmentMeta{
			SegmentID:   "seg-001",
			HeadingPath: []string{"Doc", "Login"},
			FilePath:    "seg-001.md",
		}

		seg, err := ReadSegment(tmpDir, meta)
		if err != nil {
			t.Fatalf("ReadSegment error: %v", err)
		}
		if seg.Context != "" {
			t.Errorf("Context = %q, want empty string", seg.Context)
		}
	})
}
