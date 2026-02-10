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

func setBatchEnrichFlags(t *testing.T, enabled bool, exampleModel string) {
	t.Helper()
	if err := importKireCmd.Flags().Set("enrich", boolStr(enabled)); err != nil {
		t.Fatalf("set enrich flag: %v", err)
	}
	if err := importKireCmd.Flags().Set("enrich-example-model", exampleModel); err != nil {
		t.Fatalf("set enrich-example-model flag: %v", err)
	}
	t.Cleanup(func() {
		_ = importKireCmd.Flags().Set("enrich", "false")
		_ = importKireCmd.Flags().Set("enrich-example-model", "")
	})
}

func TestImportKireBatchEnrich(t *testing.T) {
	t.Run("2-pass batch mode classifies and generates examples for FR and NFR", func(t *testing.T) {
		tmpDir := setupEnrichTestDir(t)

		testBatchEnricher = &enrich.MockBatchEnricher{
			ClassifyResults: []enrich.BatchClassifyResult{
				{SegmentID: "seg-0000", Category: enrich.CategoryOverview, Title: "概要"},
				{SegmentID: "seg-0001", Category: enrich.CategoryFunctionalRequirement, Title: "ユーザーログイン", ReqID: "REQ-001"},
				{SegmentID: "seg-0002", Category: enrich.CategoryNonFunctionalRequirement, Title: "非機能要件"},
			},
			ExampleResults: []enrich.BatchExampleResult{
				{SegmentID: "seg-0001", Examples: []spec.Example{
					{Given: "ユーザーが存在する", When: "ログインする", Then: "トークン返却"},
				}},
				{SegmentID: "seg-0002", Examples: []spec.Example{
					{Given: "システムが通常運用状態である", When: "100件の同時リクエストを送信する", Then: "レスポンスタイムが2秒以内"},
				}},
			},
		}
		t.Cleanup(func() {
			testBatchEnricher = nil
		})

		setBatchEnrichFlags(t, true, "test-example-model")

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		output := buf.String()

		// Check classification output
		if !strings.Contains(output, "classifying 3 segments") {
			t.Errorf("expected classification message, got: %s", output)
		}

		// Check example generation output: FR + NFR = 2 segments
		if !strings.Contains(output, "generating examples for 2 segments") {
			t.Errorf("expected example generation for 2 segments, got: %s", output)
		}

		// FR + NFR = 2 created
		if !strings.Contains(output, "2 created") {
			t.Errorf("expected '2 created' in output, got: %s", output)
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
		if s.Examples[0].Given != "ユーザーが存在する" {
			t.Errorf("Examples[0].Given = %q", s.Examples[0].Given)
		}

		// Verify NFR spec (auto-assigned REQ-002)
		s2, err := spec.Load(filepath.Join(specDir, "REQ-002.yml"))
		if err != nil {
			t.Fatalf("Load REQ-002 (NFR) error: %v", err)
		}
		if s2.Title != "非機能要件" {
			t.Errorf("NFR Title = %q, want '非機能要件'", s2.Title)
		}
		if len(s2.Examples) != 1 {
			t.Errorf("NFR Examples count = %d, want 1", len(s2.Examples))
		}
	})

	t.Run("batch mode skips overview and other segments", func(t *testing.T) {
		setupEnrichTestDir(t)

		testBatchEnricher = &enrich.MockBatchEnricher{
			ClassifyResults: []enrich.BatchClassifyResult{
				{SegmentID: "seg-0000", Category: enrich.CategoryOverview, Title: "概要"},
				{SegmentID: "seg-0001", Category: enrich.CategoryOverview, Title: "ログイン概要"},
				{SegmentID: "seg-0002", Category: enrich.CategoryOther, Title: "その他"},
			},
			ExampleResults: nil,
		}
		t.Cleanup(func() {
			testBatchEnricher = nil
		})

		setBatchEnrichFlags(t, true, "test-example-model")

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		output := buf.String()

		// No FR or NFR → 0 created
		if !strings.Contains(output, "0 created") {
			t.Errorf("expected '0 created', got: %s", output)
		}
	})

	t.Run("batch mode auto-assigns REQ-IDs", func(t *testing.T) {
		tmpDir := setupEnrichTestDir(t)

		testBatchEnricher = &enrich.MockBatchEnricher{
			ClassifyResults: []enrich.BatchClassifyResult{
				{SegmentID: "seg-0000", Category: enrich.CategoryFunctionalRequirement, Title: "機能A"},
				{SegmentID: "seg-0001", Category: enrich.CategoryFunctionalRequirement, Title: "機能B"},
				{SegmentID: "seg-0002", Category: enrich.CategoryOverview, Title: "概要"},
			},
			ExampleResults: []enrich.BatchExampleResult{
				{SegmentID: "seg-0000", Examples: nil},
				{SegmentID: "seg-0001", Examples: nil},
			},
		}
		t.Cleanup(func() {
			testBatchEnricher = nil
		})

		setBatchEnrichFlags(t, true, "test-example-model")

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "2 created") {
			t.Errorf("expected '2 created', got: %s", output)
		}

		// Check auto-assigned IDs
		specDir := filepath.Join(tmpDir, ".tdd", "specs")
		if _, err := spec.Load(filepath.Join(specDir, "REQ-001.yml")); err != nil {
			t.Errorf("REQ-001 should exist: %v", err)
		}
		if _, err := spec.Load(filepath.Join(specDir, "REQ-002.yml")); err != nil {
			t.Errorf("REQ-002 should exist: %v", err)
		}
	})

	t.Run("batch classify call count", func(t *testing.T) {
		setupEnrichTestDir(t)

		mock := &enrich.MockBatchEnricher{
			ClassifyResults: []enrich.BatchClassifyResult{
				{SegmentID: "seg-0000", Category: enrich.CategoryOverview, Title: "概要"},
				{SegmentID: "seg-0001", Category: enrich.CategoryOverview, Title: "ログイン"},
				{SegmentID: "seg-0002", Category: enrich.CategoryOverview, Title: "非機能要件"},
			},
		}
		testBatchEnricher = mock
		t.Cleanup(func() {
			testBatchEnricher = nil
		})

		setBatchEnrichFlags(t, true, "test-example-model")

		var buf bytes.Buffer
		importKireCmd.SetOut(&buf)

		if err := importKireCmd.RunE(importKireCmd, []string{}); err != nil {
			t.Fatalf("importKireCmd error: %v", err)
		}

		// Should be exactly 1 classify call
		if mock.ClassifyCallCount != 1 {
			t.Errorf("ClassifyCallCount = %d, want 1", mock.ClassifyCallCount)
		}
		// No FR segments → no example call
		if mock.ExampleCallCount != 0 {
			t.Errorf("ExampleCallCount = %d, want 0", mock.ExampleCallCount)
		}
	})

	t.Run("GEMINI_API_KEY required in batch mode", func(t *testing.T) {
		setupEnrichTestDir(t)

		oldBatch := testBatchEnricher
		testBatchEnricher = nil
		t.Cleanup(func() {
			testBatchEnricher = oldBatch
		})

		oldKey := os.Getenv("GEMINI_API_KEY")
		os.Unsetenv("GEMINI_API_KEY")
		t.Cleanup(func() {
			if oldKey != "" {
				os.Setenv("GEMINI_API_KEY", oldKey)
			}
		})

		setBatchEnrichFlags(t, true, "test-model")

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

	t.Run("batch mode existing GWT takes priority", func(t *testing.T) {
		tmpDir := setupEnrichTestDir(t)

		// Override seg1 to have existing GWT
		kireDir := filepath.Join(tmpDir, ".kire")
		seg1WithGWT := "### REQ-001: ユーザーログイン\n\n- Given: 既存の条件\n- When: 既存の操作\n- Then: 既存の結果\n"
		if err := os.WriteFile(filepath.Join(kireDir, "01-login.md"), []byte(seg1WithGWT), 0644); err != nil {
			t.Fatalf("write seg1 error: %v", err)
		}

		testBatchEnricher = &enrich.MockBatchEnricher{
			ClassifyResults: []enrich.BatchClassifyResult{
				{SegmentID: "seg-0000", Category: enrich.CategoryOverview, Title: "概要"},
				{SegmentID: "seg-0001", Category: enrich.CategoryFunctionalRequirement, Title: "ユーザーログイン", ReqID: "REQ-001"},
				{SegmentID: "seg-0002", Category: enrich.CategoryNonFunctionalRequirement, Title: "非機能要件"},
			},
			ExampleResults: []enrich.BatchExampleResult{
				{SegmentID: "seg-0001", Examples: []spec.Example{
					{Given: "LLM生成条件", When: "LLM生成操作", Then: "LLM生成結果"},
				}},
			},
		}
		t.Cleanup(func() {
			testBatchEnricher = nil
		})

		setBatchEnrichFlags(t, true, "test-example-model")

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
			t.Errorf("Given = %q, want '既存の条件'", s.Examples[0].Given)
		}
	})

	t.Run("1-pass mode processes FR and NFR", func(t *testing.T) {
		tmpDir := setupEnrichTestDir(t)

		results := []*enrich.EnrichResult{
			{Category: enrich.CategoryOverview, Title: "概要"},
			{Category: enrich.CategoryFunctionalRequirement, ReqID: "REQ-001", Title: "ユーザーログイン",
				Examples: []spec.Example{
					{Given: "ユーザーが存在する", When: "ログインする", Then: "トークン返却"},
				},
			},
			{Category: enrich.CategoryNonFunctionalRequirement, Title: "非機能要件",
				Examples: []spec.Example{
					{Given: "システムが稼働中", When: "負荷テストを実行する", Then: "応答時間が基準以内"},
				},
			},
		}
		testEnricher = &sequentialMockEnricher{results: results}
		t.Cleanup(func() {
			testEnricher = nil
		})

		// enrich=true but NO enrich-example-model → 1-pass mode
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

		specDir := filepath.Join(tmpDir, ".tdd", "specs")
		s, err := spec.Load(filepath.Join(specDir, "REQ-001.yml"))
		if err != nil {
			t.Fatalf("Load REQ-001 error: %v", err)
		}
		if s.Title != "ユーザーログイン" {
			t.Errorf("Title = %q", s.Title)
		}

		// NFR spec (auto-assigned REQ-002)
		s2, err := spec.Load(filepath.Join(specDir, "REQ-002.yml"))
		if err != nil {
			t.Fatalf("Load REQ-002 (NFR) error: %v", err)
		}
		if s2.Title != "非機能要件" {
			t.Errorf("NFR Title = %q", s2.Title)
		}
	})
}

func TestBatchGenerateWithFallback(t *testing.T) {
	t.Run("success on first try", func(t *testing.T) {
		mock := &enrich.MockBatchEnricher{
			ExampleResults: []enrich.BatchExampleResult{
				{SegmentID: "seg-0001"},
				{SegmentID: "seg-0002"},
			},
		}

		segments := make([]*kire.Segment, 2)
		results, err := batchGenerateWithFallback(nil, mock, segments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("results count = %d, want 2", len(results))
		}
		if mock.ExampleCallCount != 1 {
			t.Errorf("ExampleCallCount = %d, want 1", mock.ExampleCallCount)
		}
	})

	t.Run("truncated triggers recursive split", func(t *testing.T) {
		callCount := 0
		mock := &splitRetryMockBatchEnricher{
			classifyResults: nil,
			onExample: func(segs int) ([]enrich.BatchExampleResult, error) {
				callCount++
				if callCount == 1 {
					return []enrich.BatchExampleResult{{SegmentID: "seg-0001"}}, enrich.ErrBatchTruncated
				}
				// Return results for sub-batches
				results := make([]enrich.BatchExampleResult, segs)
				for i := range results {
					results[i] = enrich.BatchExampleResult{SegmentID: "split"}
				}
				return results, nil
			},
		}

		segments := make([]*kire.Segment, 4)
		for i := range segments {
			segments[i] = &kire.Segment{Meta: kire.SegmentMeta{SegmentID: "seg"}}
		}

		results, err := batchGenerateWithFallback(nil, mock, segments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should have 4 results (2 from each half)
		if len(results) != 4 {
			t.Errorf("results count = %d, want 4", len(results))
		}
		// 1 initial + 2 retries = 3 calls
		if callCount != 3 {
			t.Errorf("callCount = %d, want 3", callCount)
		}
	})

	t.Run("large batch auto-splits by maxBatchSize", func(t *testing.T) {
		callCount := 0
		mock := &splitRetryMockBatchEnricher{
			classifyResults: nil,
			onExample: func(segs int) ([]enrich.BatchExampleResult, error) {
				callCount++
				if segs > maxBatchSize {
					t.Errorf("batch size %d exceeds maxBatchSize %d", segs, maxBatchSize)
				}
				results := make([]enrich.BatchExampleResult, segs)
				for i := range results {
					results[i] = enrich.BatchExampleResult{SegmentID: "seg"}
				}
				return results, nil
			},
		}

		// 30 segments → should be split into batches <= maxBatchSize
		segments := make([]*kire.Segment, 30)
		for i := range segments {
			segments[i] = &kire.Segment{Meta: kire.SegmentMeta{SegmentID: "seg"}}
		}

		results, err := batchGenerateWithFallback(nil, mock, segments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 30 {
			t.Errorf("results count = %d, want 30", len(results))
		}
		// 30 segments / maxBatchSize(10) → needs multiple calls
		if callCount < 3 {
			t.Errorf("callCount = %d, want at least 3", callCount)
		}
	})

	t.Run("non-truncation error returns immediately", func(t *testing.T) {
		mock := &enrich.MockBatchEnricher{
			ExampleErr: os.ErrPermission,
		}

		segments := make([]*kire.Segment, 2)
		_, err := batchGenerateWithFallback(nil, mock, segments)
		if err == nil {
			t.Fatal("expected error")
		}
		if mock.ExampleCallCount != 1 {
			t.Errorf("ExampleCallCount = %d, want 1 (no retry)", mock.ExampleCallCount)
		}
	})
}

// splitRetryMockBatchEnricher はテスト用のモック。
type splitRetryMockBatchEnricher struct {
	classifyResults []enrich.BatchClassifyResult
	onExample       func(segCount int) ([]enrich.BatchExampleResult, error)
}

func (m *splitRetryMockBatchEnricher) BatchClassify(_ context.Context, _ []*kire.Segment) ([]enrich.BatchClassifyResult, error) {
	return m.classifyResults, nil
}

func (m *splitRetryMockBatchEnricher) BatchGenerateExamples(_ context.Context, segs []*kire.Segment) ([]enrich.BatchExampleResult, error) {
	return m.onExample(len(segs))
}

func TestMergeEntriesByReqID(t *testing.T) {
	t.Run("merges entries with same REQ-ID", func(t *testing.T) {
		entries := []importEntry{
			{
				seg: &kire.Segment{Meta: kire.SegmentMeta{SegmentID: "seg-001"}},
				spec: &spec.Spec{
					ID:        "REQ-001",
					Title:     "ログイン",
					Examples:  []spec.Example{{ID: "E1", Given: "G1", When: "W1", Then: "T1"}},
					Questions: []string{"Q1"},
				},
			},
			{
				seg: &kire.Segment{Meta: kire.SegmentMeta{SegmentID: "seg-002"}},
				spec: &spec.Spec{
					ID:        "REQ-002",
					Title:     "ログアウト",
					Examples:  []spec.Example{{ID: "E1", Given: "G2", When: "W2", Then: "T2"}},
					Questions: []string{"Q2"},
				},
			},
			{
				seg: &kire.Segment{Meta: kire.SegmentMeta{SegmentID: "seg-003"}},
				spec: &spec.Spec{
					ID:        "REQ-001",
					Title:     "ログイン (続き)",
					Examples:  []spec.Example{{ID: "E1", Given: "G3", When: "W3", Then: "T3"}},
					Questions: []string{"Q1", "Q3"},
				},
			},
		}

		result := mergeEntriesByReqID(entries)

		if len(result) != 2 {
			t.Fatalf("result count = %d, want 2", len(result))
		}

		// REQ-001 should be first (出現順保持)
		r1 := result[0]
		if r1.spec.ID != "REQ-001" {
			t.Errorf("result[0].ID = %q, want REQ-001", r1.spec.ID)
		}
		// タイトルは最初のエントリ
		if r1.spec.Title != "ログイン" {
			t.Errorf("result[0].Title = %q, want 'ログイン'", r1.spec.Title)
		}
		// Examples 結合・再採番
		if len(r1.spec.Examples) != 2 {
			t.Fatalf("result[0].Examples count = %d, want 2", len(r1.spec.Examples))
		}
		if r1.spec.Examples[0].ID != "E1" || r1.spec.Examples[0].Given != "G1" {
			t.Errorf("result[0].Examples[0] = %+v", r1.spec.Examples[0])
		}
		if r1.spec.Examples[1].ID != "E2" || r1.spec.Examples[1].Given != "G3" {
			t.Errorf("result[0].Examples[1] = %+v", r1.spec.Examples[1])
		}
		// Questions 重複除去
		if len(r1.spec.Questions) != 2 {
			t.Fatalf("result[0].Questions count = %d, want 2", len(r1.spec.Questions))
		}
		if r1.spec.Questions[0] != "Q1" || r1.spec.Questions[1] != "Q3" {
			t.Errorf("result[0].Questions = %v", r1.spec.Questions)
		}

		// REQ-002 should be second
		r2 := result[1]
		if r2.spec.ID != "REQ-002" {
			t.Errorf("result[1].ID = %q, want REQ-002", r2.spec.ID)
		}
	})

	t.Run("no duplicates returns unchanged", func(t *testing.T) {
		entries := []importEntry{
			{
				seg:  &kire.Segment{Meta: kire.SegmentMeta{SegmentID: "seg-001"}},
				spec: &spec.Spec{ID: "REQ-001", Title: "A", Examples: []spec.Example{{Given: "G", When: "W", Then: "T"}}},
			},
			{
				seg:  &kire.Segment{Meta: kire.SegmentMeta{SegmentID: "seg-002"}},
				spec: &spec.Spec{ID: "REQ-002", Title: "B"},
			},
		}

		result := mergeEntriesByReqID(entries)

		if len(result) != 2 {
			t.Fatalf("result count = %d, want 2", len(result))
		}
		if result[0].spec.ID != "REQ-001" || result[1].spec.ID != "REQ-002" {
			t.Errorf("IDs = [%s, %s]", result[0].spec.ID, result[1].spec.ID)
		}
		// Example IDs should still be renumbered
		if len(result[0].spec.Examples) == 1 && result[0].spec.Examples[0].ID != "E1" {
			t.Errorf("Examples[0].ID = %q, want E1", result[0].spec.Examples[0].ID)
		}
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		result := mergeEntriesByReqID(nil)
		if len(result) != 0 {
			t.Errorf("result count = %d, want 0", len(result))
		}
	})

	t.Run("preserves source from first entry", func(t *testing.T) {
		entries := []importEntry{
			{
				seg: &kire.Segment{Meta: kire.SegmentMeta{SegmentID: "seg-001"}},
				spec: &spec.Spec{
					ID:    "REQ-001",
					Title: "A",
					Source: spec.SourceInfo{SegmentID: "seg-001", FilePath: "first.md"},
				},
			},
			{
				seg: &kire.Segment{Meta: kire.SegmentMeta{SegmentID: "seg-005"}},
				spec: &spec.Spec{
					ID:    "REQ-001",
					Title: "A continued",
					Source: spec.SourceInfo{SegmentID: "seg-005", FilePath: "second.md"},
				},
			},
		}

		result := mergeEntriesByReqID(entries)
		if len(result) != 1 {
			t.Fatalf("result count = %d, want 1", len(result))
		}
		if result[0].spec.Source.FilePath != "first.md" {
			t.Errorf("Source.FilePath = %q, want 'first.md'", result[0].spec.Source.FilePath)
		}
	})
}
