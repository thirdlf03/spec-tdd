package kire

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractReqID(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "extract REQ-001 from content",
			content: "# Login\n\nREQ-001\n\nSome description.",
			want:    "REQ-001",
		},
		{
			name:    "extract REQ-123 from inline text",
			content: "This requirement REQ-123 describes login.",
			want:    "REQ-123",
		},
		{
			name:    "no REQ pattern returns empty",
			content: "# Login\n\nSome description without req id.",
			want:    "",
		},
		{
			name:    "multiple REQ patterns returns first",
			content: "REQ-002\nREQ-005\n",
			want:    "REQ-002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractReqID(tt.content)
			if got != tt.want {
				t.Errorf("ExtractReqID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractExamples(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantFirst [3]string // given, when, then
	}{
		{
			name: "single GWT set",
			content: `# Examples

- Given: ユーザーが存在する
- When: 正しいパスワードでログインする
- Then: 認証トークンが返却される
`,
			wantCount: 1,
			wantFirst: [3]string{"ユーザーが存在する", "正しいパスワードでログインする", "認証トークンが返却される"},
		},
		{
			name: "multiple GWT sets",
			content: `- Given: condition A
- When: action A
- Then: result A

- Given: condition B
- When: action B
- Then: result B
`,
			wantCount: 2,
			wantFirst: [3]string{"condition A", "action A", "result A"},
		},
		{
			name: "bullet with asterisk",
			content: `* Given: cond
* When: act
* Then: res
`,
			wantCount: 1,
			wantFirst: [3]string{"cond", "act", "res"},
		},
		{
			name:      "no GWT pattern returns empty",
			content:   "# Login\n\nJust a description.\n",
			wantCount: 0,
		},
		{
			name: "case insensitive",
			content: `- given: a
- when: b
- then: c
`,
			wantCount: 1,
			wantFirst: [3]string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			examples := ExtractExamples(tt.content)
			if len(examples) != tt.wantCount {
				t.Fatalf("ExtractExamples() returned %d examples, want %d", len(examples), tt.wantCount)
			}
			if tt.wantCount > 0 {
				if examples[0].Given != tt.wantFirst[0] {
					t.Errorf("Given = %q, want %q", examples[0].Given, tt.wantFirst[0])
				}
				if examples[0].When != tt.wantFirst[1] {
					t.Errorf("When = %q, want %q", examples[0].When, tt.wantFirst[1])
				}
				if examples[0].Then != tt.wantFirst[2] {
					t.Errorf("Then = %q, want %q", examples[0].Then, tt.wantFirst[2])
				}
			}
		})
	}
}

func TestExtractQuestions(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantFirst string
	}{
		{
			name:      "lines ending with question mark",
			content:   "# Login\n\nセッションの有効期限は？\nパスワードポリシーは？\n",
			wantCount: 2,
			wantFirst: "セッションの有効期限は？",
		},
		{
			name: "Questions section",
			content: `# Login

## Questions

- セッション管理はどうする
- タイムアウトの設定値は
`,
			wantCount: 2,
			wantFirst: "セッション管理はどうする",
		},
		{
			name:      "no questions returns empty",
			content:   "# Login\n\nJust a description.\n",
			wantCount: 0,
		},
		{
			name:      "heading with question mark is excluded",
			content:   "# What is this?\n\nContent.\n",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			questions := ExtractQuestions(tt.content)
			if len(questions) != tt.wantCount {
				t.Fatalf("ExtractQuestions() returned %d, want %d", len(questions), tt.wantCount)
			}
			if tt.wantCount > 0 && questions[0] != tt.wantFirst {
				t.Errorf("questions[0] = %q, want %q", questions[0], tt.wantFirst)
			}
		})
	}
}

func TestConvertToSpec(t *testing.T) {
	t.Run("heading_path last element becomes title", func(t *testing.T) {
		tmpDir := t.TempDir()
		seg := &Segment{
			Meta: SegmentMeta{
				SegmentID:   "seg-001",
				HeadingPath: []string{"設計書", "認証", "ログイン"},
				FilePath:    "seg-001.md",
			},
			Content: "# ログイン\n\nログイン機能の説明。\n",
		}

		s, err := ConvertToSpec(seg, tmpDir)
		if err != nil {
			t.Fatalf("ConvertToSpec error: %v", err)
		}
		if s.Title != "ログイン" {
			t.Errorf("Title = %q, want %q", s.Title, "ログイン")
		}
	})

	t.Run("REQ ID from content is prioritized", func(t *testing.T) {
		tmpDir := t.TempDir()
		seg := &Segment{
			Meta: SegmentMeta{
				SegmentID:   "seg-001",
				HeadingPath: []string{"Doc", "Login"},
				FilePath:    "seg-001.md",
			},
			Content: "# Login\n\nREQ-042\n\nDescription.\n",
		}

		s, err := ConvertToSpec(seg, tmpDir)
		if err != nil {
			t.Fatalf("ConvertToSpec error: %v", err)
		}
		if s.ID != "REQ-042" {
			t.Errorf("ID = %q, want %q", s.ID, "REQ-042")
		}
	})

	t.Run("auto-assign ID when no REQ in content", func(t *testing.T) {
		tmpDir := t.TempDir()
		seg := &Segment{
			Meta: SegmentMeta{
				SegmentID:   "seg-001",
				HeadingPath: []string{"Doc", "Login"},
				FilePath:    "seg-001.md",
			},
			Content: "# Login\n\nNo REQ ID here.\n",
		}

		s, err := ConvertToSpec(seg, tmpDir)
		if err != nil {
			t.Fatalf("ConvertToSpec error: %v", err)
		}
		if s.ID != "REQ-001" {
			t.Errorf("ID = %q, want %q", s.ID, "REQ-001")
		}
	})

	t.Run("source field is populated", func(t *testing.T) {
		tmpDir := t.TempDir()
		seg := &Segment{
			Meta: SegmentMeta{
				SegmentID:   "seg-005",
				HeadingPath: []string{"Doc", "Auth"},
				FilePath:    "seg-005.md",
			},
			Content: "# Auth\n\nDescription.\n",
		}

		s, err := ConvertToSpec(seg, tmpDir)
		if err != nil {
			t.Fatalf("ConvertToSpec error: %v", err)
		}
		if s.Source.SegmentID != "seg-005" {
			t.Errorf("Source.SegmentID = %q, want %q", s.Source.SegmentID, "seg-005")
		}
		if len(s.Source.HeadingPath) != 2 || s.Source.HeadingPath[1] != "Auth" {
			t.Errorf("Source.HeadingPath = %v, want [Doc Auth]", s.Source.HeadingPath)
		}
	})

	t.Run("examples and questions are extracted", func(t *testing.T) {
		tmpDir := t.TempDir()
		seg := &Segment{
			Meta: SegmentMeta{
				SegmentID:   "seg-001",
				HeadingPath: []string{"Doc", "Login"},
				FilePath:    "seg-001.md",
			},
			Content: `# Login

- Given: ユーザーが存在する
- When: パスワードでログイン
- Then: トークン返却

セッションの有効期限は？
`,
		}

		s, err := ConvertToSpec(seg, tmpDir)
		if err != nil {
			t.Fatalf("ConvertToSpec error: %v", err)
		}
		if len(s.Examples) != 1 {
			t.Fatalf("expected 1 example, got %d", len(s.Examples))
		}
		if len(s.Questions) != 1 {
			t.Fatalf("expected 1 question, got %d", len(s.Questions))
		}
	})

	t.Run("idempotent: same input produces same output", func(t *testing.T) {
		tmpDir := t.TempDir()
		seg := &Segment{
			Meta: SegmentMeta{
				SegmentID:   "seg-001",
				HeadingPath: []string{"Doc", "Login"},
				FilePath:    "seg-001.md",
			},
			Content: "# Login\n\nREQ-010\n\nDescription.\n",
		}

		s1, err := ConvertToSpec(seg, tmpDir)
		if err != nil {
			t.Fatalf("ConvertToSpec error: %v", err)
		}
		s2, err := ConvertToSpec(seg, tmpDir)
		if err != nil {
			t.Fatalf("ConvertToSpec error: %v", err)
		}
		if s1.ID != s2.ID || s1.Title != s2.Title {
			t.Errorf("not idempotent: s1=%+v, s2=%+v", s1, s2)
		}
	})

	t.Run("auto-assign considers existing specs", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create an existing spec file
		specDir := filepath.Join(tmpDir, "specs")
		if err := os.MkdirAll(specDir, 0755); err != nil {
			t.Fatalf("MkdirAll error: %v", err)
		}
		// Write a minimal valid YAML
		yamlContent := "id: REQ-005\ntitle: Existing\n"
		if err := os.WriteFile(filepath.Join(specDir, "REQ-005.yml"), []byte(yamlContent), 0644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		seg := &Segment{
			Meta: SegmentMeta{
				SegmentID:   "seg-001",
				HeadingPath: []string{"Doc", "Login"},
				FilePath:    "seg-001.md",
			},
			Content: "# Login\n\nNo REQ ID.\n",
		}

		s, err := ConvertToSpec(seg, specDir)
		if err != nil {
			t.Fatalf("ConvertToSpec error: %v", err)
		}
		if s.ID != "REQ-006" {
			t.Errorf("ID = %q, want %q (next after existing REQ-005)", s.ID, "REQ-006")
		}
	})
}
