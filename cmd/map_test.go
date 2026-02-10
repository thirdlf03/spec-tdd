package cmd

import (
	"strings"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func TestRenderMapMarkdown_WithSource(t *testing.T) {
	t.Run("source info is displayed when present", func(t *testing.T) {
		specs := []*spec.Spec{
			{
				ID:    "REQ-001",
				Title: "Login",
				Source: spec.SourceInfo{
					SegmentID:   "seg-001",
					HeadingPath: []string{"設計書", "認証", "ログイン"},
				},
				Examples: []spec.Example{
					{ID: "E1", Given: "a", When: "b", Then: "c"},
				},
			},
		}

		output := renderMapMarkdown(specs)

		if !strings.Contains(output, "Source: segment_id=seg-001") {
			t.Errorf("expected source info in output, got:\n%s", output)
		}
		if !strings.Contains(output, "設計書 > 認証 > ログイン") {
			t.Errorf("expected heading_path in output, got:\n%s", output)
		}
	})

	t.Run("source info includes file_path when present", func(t *testing.T) {
		specs := []*spec.Spec{
			{
				ID:    "REQ-001",
				Title: "Login",
				Source: spec.SourceInfo{
					SegmentID:   "seg-001",
					HeadingPath: []string{"設計書", "認証"},
					FilePath:    "seg-0001.md",
				},
			},
		}

		output := renderMapMarkdown(specs)

		if !strings.Contains(output, "file_path=seg-0001.md") {
			t.Errorf("expected file_path in output, got:\n%s", output)
		}
	})

	t.Run("source info is omitted when zero value", func(t *testing.T) {
		specs := []*spec.Spec{
			{
				ID:    "REQ-001",
				Title: "Login",
				Examples: []spec.Example{
					{ID: "E1", Given: "a", When: "b", Then: "c"},
				},
			},
		}

		output := renderMapMarkdown(specs)

		if strings.Contains(output, "Source:") {
			t.Errorf("expected no source info for zero value, got:\n%s", output)
		}
	})
}
