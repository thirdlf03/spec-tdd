package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/enrich"
	"github.com/thirdlf03/spec-tdd/internal/kire"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func setupEnrichTestDir(t *testing.T) string {
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

	// 3 segments: overview, functional req, non-functional req
	jsonl := `{"content":"# 概要\n\nプロジェクトの概要。\n","metadata":{"source":"doc.md","segment_index":0,"filename":"00-overview.md","heading_path":["Doc","概要"],"token_count":50,"block_count":3}}
{"content":"### REQ-001: ユーザーログイン\n\n正常系: ユーザーが存在する\n異常系: パスワードが間違い\n","metadata":{"source":"doc.md","segment_index":1,"filename":"01-login.md","heading_path":["Doc","認証","ログイン"],"token_count":80,"block_count":5}}
{"content":"# 非機能要件\n\nレスポンスタイム1秒以内\n","metadata":{"source":"doc.md","segment_index":2,"filename":"02-nfr.md","heading_path":["Doc","非機能要件"],"token_count":40,"block_count":2}}`
	if err := os.WriteFile(filepath.Join(kireDir, "metadata.jsonl"), []byte(jsonl), 0644); err != nil {
		t.Fatalf("write jsonl error: %v", err)
	}

	seg0 := "# 概要\n\nプロジェクトの概要。\n"
	seg1 := "### REQ-001: ユーザーログイン\n\n正常系: ユーザーが存在する\n異常系: パスワードが間違い\n"
	seg2 := "# 非機能要件\n\nレスポンスタイム1秒以内\n"
	if err := os.WriteFile(filepath.Join(kireDir, "00-overview.md"), []byte(seg0), 0644); err != nil {
		t.Fatalf("write seg0 error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(kireDir, "01-login.md"), []byte(seg1), 0644); err != nil {
		t.Fatalf("write seg1 error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(kireDir, "02-nfr.md"), []byte(seg2), 0644); err != nil {
		t.Fatalf("write seg2 error: %v", err)
	}

	return tmpDir
}

func setEnrichFlags(t *testing.T, enabled bool) {
	t.Helper()
	if err := importKireCmd.Flags().Set("enrich", boolStr(enabled)); err != nil {
		t.Fatalf("set enrich flag: %v", err)
	}
	t.Cleanup(func() {
		_ = importKireCmd.Flags().Set("enrich", "false")
	})
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestImportKireEnrich(t *testing.T) {
	t.Run("enrichment processes FR and NFR, skips overview", func(t *testing.T) {
		tmpDir := setupEnrichTestDir(t)

		// Setup mock enricher
		callIndex := 0
		results := []*enrich.EnrichResult{
			{Category: enrich.CategoryOverview, Title: "概要"},
			{Category: enrich.CategoryFunctionalRequirement, ReqID: "REQ-001", Title: "ユーザーログイン",
				Examples: []spec.Example{
					{Given: "ユーザーが存在する", When: "正しいパスワードでログイン", Then: "認証トークンが返却される"},
				},
			},
			{Category: enrich.CategoryNonFunctionalRequirement, Title: "非機能要件",
				Examples: []spec.Example{
					{Given: "システムが稼働中", When: "負荷テストを実行する", Then: "応答時間が基準以内"},
				},
			},
		}
		testEnricher = &enrich.MockEnricher{
			Result: results[0],
		}
		// Override Enrich to return different results per call
		mockE := &sequentialMockEnricher{results: results}
		testEnricher = mockE
		_ = callIndex // unused after mock setup
		t.Cleanup(func() {
			testEnricher = nil
		})

		setEnrichFlags(t, true)

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		output := buf.String()

		// FR + NFR = 2 created
		if !strings.Contains(output, "2 created") {
			t.Errorf("expected '2 created' in output, got: %s", output)
		}

		// Enrichment summary: 2 enriched (FR + NFR), 1 skipped (overview)
		if !strings.Contains(output, "2 enriched") {
			t.Errorf("expected '2 enriched' in output, got: %s", output)
		}
		if !strings.Contains(output, "1 skipped") {
			t.Errorf("expected '1 skipped' in output, got: %s", output)
		}

		// Verify FR spec
		specDir := filepath.Join(tmpDir, ".tdd", "specs")
		s, err := spec.Load(filepath.Join(specDir, "REQ-001.yml"))
		if err != nil {
			t.Fatalf("Load REQ-001 error: %v", err)
		}
		if s.Title != "ユーザーログイン" {
			t.Errorf("Title = %q, want 'ユーザーログイン'", s.Title)
		}
		if len(s.Examples) != 1 {
			t.Errorf("Examples count = %d, want 1", len(s.Examples))
		}

		// Verify NFR spec (auto-assigned REQ-002)
		s2, err := spec.Load(filepath.Join(specDir, "REQ-002.yml"))
		if err != nil {
			t.Fatalf("Load REQ-002 (NFR) error: %v", err)
		}
		if s2.Title != "非機能要件" {
			t.Errorf("NFR Title = %q", s2.Title)
		}
		if len(s2.Examples) != 1 {
			t.Errorf("NFR Examples count = %d, want 1", len(s2.Examples))
		}
	})

	t.Run("enrichment fallback on error", func(t *testing.T) {
		setupEnrichTestDir(t)

		// Mock that returns error for all segments
		testEnricher = &enrich.MockEnricher{
			Err: os.ErrPermission,
		}
		t.Cleanup(func() {
			testEnricher = nil
		})

		setEnrichFlags(t, true)

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		output := buf.String()

		// All 3 segments should fall back to regex extraction
		if !strings.Contains(output, "3 created") {
			t.Errorf("expected '3 created' (fallback) in output, got: %s", output)
		}
		if !strings.Contains(output, "3 errors") {
			t.Errorf("expected '3 errors' in output, got: %s", output)
		}
	})

	t.Run("dry-run with enrich shows enrichment preview", func(t *testing.T) {
		setupEnrichTestDir(t)

		results := []*enrich.EnrichResult{
			{Category: enrich.CategoryOverview, Title: "概要"},
			{Category: enrich.CategoryFunctionalRequirement, ReqID: "REQ-001", Title: "ユーザーログイン"},
			{Category: enrich.CategoryNonFunctionalRequirement, Title: "非機能要件"},
		}
		testEnricher = &sequentialMockEnricher{results: results}
		t.Cleanup(func() {
			testEnricher = nil
		})

		setEnrichFlags(t, true)
		if err := importKireCmd.Flags().Set("dry-run", "true"); err != nil {
			t.Fatalf("set dry-run flag: %v", err)
		}
		defer func() {
			_ = importKireCmd.Flags().Set("dry-run", "false")
		}()

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "[dry-run]") {
			t.Errorf("expected '[dry-run]' in output, got: %s", output)
		}
		if !strings.Contains(output, "REQ-001") {
			t.Errorf("expected 'REQ-001' in output, got: %s", output)
		}
	})

	t.Run("progress output during enrichment", func(t *testing.T) {
		setupEnrichTestDir(t)

		results := []*enrich.EnrichResult{
			{Category: enrich.CategoryOverview, Title: "概要"},
			{Category: enrich.CategoryFunctionalRequirement, ReqID: "REQ-001", Title: "ログイン"},
			{Category: enrich.CategoryOther, Title: "その他"},
		}
		testEnricher = &sequentialMockEnricher{results: results}
		t.Cleanup(func() {
			testEnricher = nil
		})

		setEnrichFlags(t, true)

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "enriching: seg-0000") {
			t.Errorf("expected progress for seg-0000, got: %s", output)
		}
		if !strings.Contains(output, "skipped (overview)") {
			t.Errorf("expected 'skipped (overview)', got: %s", output)
		}
		if !strings.Contains(output, "done") {
			t.Errorf("expected 'done' for functional requirement, got: %s", output)
		}
	})

	t.Run("GEMINI_API_KEY required error when not using test enricher", func(t *testing.T) {
		setupEnrichTestDir(t)

		// Don't set testEnricher - force real path
		oldEnricher := testEnricher
		testEnricher = nil
		t.Cleanup(func() {
			testEnricher = oldEnricher
		})

		// Unset env var
		oldKey := os.Getenv("GEMINI_API_KEY")
		os.Unsetenv("GEMINI_API_KEY")
		t.Cleanup(func() {
			if oldKey != "" {
				os.Setenv("GEMINI_API_KEY", oldKey)
			}
		})

		setEnrichFlags(t, true)

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		err := importKireCmd.RunE(importKireCmd, []string{})
		if err == nil {
			t.Fatal("expected error for missing GEMINI_API_KEY")
		}
		if !strings.Contains(err.Error(), "GEMINI_API_KEY") {
			t.Errorf("error should mention GEMINI_API_KEY, got: %v", err)
		}
	})

	t.Run("existing GWT examples take priority over enriched", func(t *testing.T) {
		tmpDir := setupEnrichTestDir(t)

		// Segment 1 has existing GWT in content, enricher also provides GWT
		results := []*enrich.EnrichResult{
			{Category: enrich.CategoryOverview, Title: "概要"},
			{Category: enrich.CategoryFunctionalRequirement, ReqID: "REQ-001", Title: "ユーザーログイン",
				Examples: []spec.Example{
					{Given: "LLM生成条件", When: "LLM生成操作", Then: "LLM生成結果"},
				},
			},
			{Category: enrich.CategoryNonFunctionalRequirement, Title: "非機能要件"},
		}
		testEnricher = &sequentialMockEnricher{results: results}
		t.Cleanup(func() {
			testEnricher = nil
		})

		// Override seg1 to have existing GWT
		kireDir := filepath.Join(tmpDir, ".kire")
		seg1WithGWT := "### REQ-001: ユーザーログイン\n\n- Given: 既存の条件\n- When: 既存の操作\n- Then: 既存の結果\n"
		if err := os.WriteFile(filepath.Join(kireDir, "01-login.md"), []byte(seg1WithGWT), 0644); err != nil {
			t.Fatalf("write seg1 error: %v", err)
		}

		setEnrichFlags(t, true)

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		specDir := filepath.Join(tmpDir, ".tdd", "specs")
		s, err := spec.Load(filepath.Join(specDir, "REQ-001.yml"))
		if err != nil {
			t.Fatalf("Load REQ-001 error: %v", err)
		}
		if len(s.Examples) != 1 {
			t.Fatalf("Examples count = %d, want 1", len(s.Examples))
		}
		// Existing GWT should take priority
		if s.Examples[0].Given != "既存の条件" {
			t.Errorf("Given = %q, want '既存の条件' (existing should take priority)", s.Examples[0].Given)
		}
	})

	t.Run("HeadingPath fallback when enrichment returns empty title", func(t *testing.T) {
		tmpDir := setupEnrichTestDir(t)

		// Enricher returns empty title for functional requirement
		results := []*enrich.EnrichResult{
			{Category: enrich.CategoryOverview, Title: "概要"},
			{Category: enrich.CategoryFunctionalRequirement, ReqID: "REQ-001", Title: ""},
			{Category: enrich.CategoryNonFunctionalRequirement, Title: "非機能要件"},
		}
		testEnricher = &sequentialMockEnricher{results: results}
		t.Cleanup(func() {
			testEnricher = nil
		})

		setEnrichFlags(t, true)

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		// Title should fall back to HeadingPath last element "ログイン"
		specDir := filepath.Join(tmpDir, ".tdd", "specs")
		s, err := spec.Load(filepath.Join(specDir, "REQ-001.yml"))
		if err != nil {
			t.Fatalf("Load REQ-001 error: %v", err)
		}
		if s.Title != "ログイン" {
			t.Errorf("Title = %q, want 'ログイン' (HeadingPath fallback)", s.Title)
		}
	})

	t.Run("duplicate REQ-ID detection", func(t *testing.T) {
		setupEnrichTestDir(t)

		// Both segments classified as functional with same REQ-ID
		results := []*enrich.EnrichResult{
			{Category: enrich.CategoryFunctionalRequirement, ReqID: "REQ-001", Title: "機能A"},
			{Category: enrich.CategoryFunctionalRequirement, ReqID: "REQ-001", Title: "機能B"},
			{Category: enrich.CategoryNonFunctionalRequirement, Title: "非機能要件"},
		}
		testEnricher = &sequentialMockEnricher{results: results}
		t.Cleanup(func() {
			testEnricher = nil
		})

		setEnrichFlags(t, true)

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		err := importKireCmd.RunE(importKireCmd, []string{})
		if err == nil {
			t.Fatal("expected error for duplicate REQ-IDs")
		}
		if !strings.Contains(err.Error(), "REQ-001") {
			t.Errorf("error should mention REQ-001, got: %v", err)
		}
	})
}

// sequentialMockEnricher returns results in order for each call.
type sequentialMockEnricher struct {
	results []*enrich.EnrichResult
	index   int
}

func (m *sequentialMockEnricher) Enrich(_ context.Context, _ *kire.Segment) (*enrich.EnrichResult, error) {
	if m.index >= len(m.results) {
		return &enrich.EnrichResult{Category: enrich.CategoryOther, Title: "unknown"}, nil
	}
	r := m.results[m.index]
	m.index++
	return r, nil
}
