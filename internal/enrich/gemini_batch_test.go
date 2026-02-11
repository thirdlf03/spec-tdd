package enrich

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/thirdlf03/spec-tdd/internal/kire"
	"google.golang.org/genai"
)

func newTestBatchEnricher(fn batchGenerateFunc) *GeminiBatchEnricher {
	return &GeminiBatchEnricher{
		cfg: GeminiBatchEnricherConfig{
			APIKey:          "test-key",
			ClassifyModel:   "test-classify-model",
			ExampleModel:    "test-example-model",
			ClassifyTimeout: 30 * time.Second,
			ExampleTimeout:  60 * time.Second,
			MaxRetries:      2,
		},
		generate: fn,
	}
}

func TestNewGeminiBatchEnricher(t *testing.T) {
	t.Run("empty API key returns error", func(t *testing.T) {
		_, err := NewGeminiBatchEnricher(GeminiBatchEnricherConfig{APIKey: ""})
		if err == nil {
			t.Fatal("expected error for empty API key")
		}
	})

	t.Run("valid config returns enricher", func(t *testing.T) {
		e, err := NewGeminiBatchEnricher(GeminiBatchEnricherConfig{APIKey: "test-key"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e == nil {
			t.Fatal("expected non-nil enricher")
		}
	})

	t.Run("defaults are applied", func(t *testing.T) {
		e, err := NewGeminiBatchEnricher(GeminiBatchEnricherConfig{APIKey: "test-key"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ge, ok := e.(*GeminiBatchEnricher)
		if !ok {
			t.Fatal("expected *GeminiBatchEnricher type")
		}
		if ge.cfg.ClassifyModel != "gemini-2.5-flash-lite" {
			t.Errorf("ClassifyModel = %q, want %q", ge.cfg.ClassifyModel, "gemini-2.5-flash-lite")
		}
		if ge.cfg.ExampleModel != "gemini-2.5-flash" {
			t.Errorf("ExampleModel = %q, want %q", ge.cfg.ExampleModel, "gemini-2.5-flash")
		}
		if ge.cfg.ClassifyTimeout != 60*time.Second {
			t.Errorf("ClassifyTimeout = %v, want %v", ge.cfg.ClassifyTimeout, 60*time.Second)
		}
		if ge.cfg.ExampleTimeout != 180*time.Second {
			t.Errorf("ExampleTimeout = %v, want %v", ge.cfg.ExampleTimeout, 180*time.Second)
		}
	})
}

func TestGeminiBatchEnricher_BatchClassify(t *testing.T) {
	segments := []*kire.Segment{
		{Meta: kire.SegmentMeta{SegmentID: "seg-0001"}, Content: "# Overview"},
		{Meta: kire.SegmentMeta{SegmentID: "seg-0002"}, Content: "# Login"},
		{Meta: kire.SegmentMeta{SegmentID: "seg-0003"}, Content: "# NFR"},
	}

	t.Run("successful classification", func(t *testing.T) {
		e := newTestBatchEnricher(func(_ context.Context, model, _ string, _ *genai.Schema) (string, *genai.GenerateContentResponse, error) {
			if model != "test-classify-model" {
				t.Errorf("expected classify model, got %q", model)
			}
			results := []BatchClassifyResult{
				{SegmentID: "seg-0001", Category: CategoryOverview, Title: "概要"},
				{SegmentID: "seg-0002", Category: CategoryFunctionalRequirement, Title: "ログイン", ReqID: "REQ-001"},
				{SegmentID: "seg-0003", Category: CategoryNonFunctionalRequirement, Title: "非機能要件"},
			}
			b, _ := json.Marshal(results)
			return string(b), &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
			}, nil
		})

		results, err := e.BatchClassify(context.Background(), segments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("results count = %d, want 3", len(results))
		}
		if results[0].Category != CategoryOverview {
			t.Errorf("results[0].Category = %q, want overview", results[0].Category)
		}
		if results[1].Category != CategoryFunctionalRequirement {
			t.Errorf("results[1].Category = %q, want functional_requirement", results[1].Category)
		}
		if results[1].ReqID != "REQ-001" {
			t.Errorf("results[1].ReqID = %q, want REQ-001", results[1].ReqID)
		}
	})

	t.Run("empty segments returns nil", func(t *testing.T) {
		e := newTestBatchEnricher(nil)
		results, err := e.BatchClassify(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results != nil {
			t.Errorf("expected nil results, got %v", results)
		}
	})

	t.Run("truncated response returns ErrBatchTruncated", func(t *testing.T) {
		e := newTestBatchEnricher(func(_ context.Context, _, _ string, _ *genai.Schema) (string, *genai.GenerateContentResponse, error) {
			// Only return 2 results for 3 segments
			results := []BatchClassifyResult{
				{SegmentID: "seg-0001", Category: CategoryOverview, Title: "概要"},
				{SegmentID: "seg-0002", Category: CategoryFunctionalRequirement, Title: "ログイン"},
			}
			b, _ := json.Marshal(results)
			return string(b), &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
			}, nil
		})

		results, err := e.BatchClassify(context.Background(), segments)
		if !errors.Is(err, ErrBatchTruncated) {
			t.Errorf("expected ErrBatchTruncated, got %v", err)
		}
		if len(results) != 2 {
			t.Errorf("results count = %d, want 2", len(results))
		}
	})

	t.Run("MaxTokens finish reason returns ErrBatchTruncated", func(t *testing.T) {
		e := newTestBatchEnricher(func(_ context.Context, _, _ string, _ *genai.Schema) (string, *genai.GenerateContentResponse, error) {
			results := []BatchClassifyResult{
				{SegmentID: "seg-0001", Category: CategoryOverview, Title: "概要"},
				{SegmentID: "seg-0002", Category: CategoryFunctionalRequirement, Title: "ログイン"},
				{SegmentID: "seg-0003", Category: CategoryNonFunctionalRequirement, Title: "非機能要件"},
			}
			b, _ := json.Marshal(results)
			return string(b), &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonMaxTokens}},
			}, nil
		})

		_, err := e.BatchClassify(context.Background(), segments)
		if !errors.Is(err, ErrBatchTruncated) {
			t.Errorf("expected ErrBatchTruncated for MaxTokens, got %v", err)
		}
	})

	t.Run("retry on API error", func(t *testing.T) {
		callCount := 0
		e := newTestBatchEnricher(func(_ context.Context, _, _ string, _ *genai.Schema) (string, *genai.GenerateContentResponse, error) {
			callCount++
			if callCount <= 2 {
				return "", nil, context.DeadlineExceeded
			}
			results := []BatchClassifyResult{
				{SegmentID: "seg-0001", Category: CategoryOverview, Title: "概要"},
				{SegmentID: "seg-0002", Category: CategoryFunctionalRequirement, Title: "ログイン"},
				{SegmentID: "seg-0003", Category: CategoryNonFunctionalRequirement, Title: "非機能要件"},
			}
			b, _ := json.Marshal(results)
			return string(b), &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
			}, nil
		})

		results, err := e.BatchClassify(context.Background(), segments)
		if err != nil {
			t.Fatalf("unexpected error after retries: %v", err)
		}
		if callCount != 3 {
			t.Errorf("callCount = %d, want 3", callCount)
		}
		if len(results) != 3 {
			t.Errorf("results count = %d, want 3", len(results))
		}
	})

	t.Run("all retries fail returns error", func(t *testing.T) {
		e := newTestBatchEnricher(func(_ context.Context, _, _ string, _ *genai.Schema) (string, *genai.GenerateContentResponse, error) {
			return "", nil, context.DeadlineExceeded
		})

		_, err := e.BatchClassify(context.Background(), segments)
		if err == nil {
			t.Fatal("expected error when all retries fail")
		}
	})

	t.Run("unknown category normalized to other", func(t *testing.T) {
		e := newTestBatchEnricher(func(_ context.Context, _, _ string, _ *genai.Schema) (string, *genai.GenerateContentResponse, error) {
			results := []BatchClassifyResult{
				{SegmentID: "seg-0001", Category: "unknown_type", Title: "不明"},
				{SegmentID: "seg-0002", Category: CategoryFunctionalRequirement, Title: "ログイン"},
				{SegmentID: "seg-0003", Category: CategoryOther, Title: "その他"},
			}
			b, _ := json.Marshal(results)
			return string(b), &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
			}, nil
		})

		results, err := e.BatchClassify(context.Background(), segments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results[0].Category != CategoryOther {
			t.Errorf("results[0].Category = %q, want other", results[0].Category)
		}
	})
}

func TestGeminiBatchEnricher_BatchGenerateExamples(t *testing.T) {
	segments := []*kire.Segment{
		{Meta: kire.SegmentMeta{SegmentID: "seg-0001", HeadingPath: []string{"Doc", "Login"}}, Content: "# Login"},
		{Meta: kire.SegmentMeta{SegmentID: "seg-0002", HeadingPath: []string{"Doc", "Register"}}, Content: "# Register"},
	}

	t.Run("successful example generation", func(t *testing.T) {
		e := newTestBatchEnricher(func(_ context.Context, model, _ string, _ *genai.Schema) (string, *genai.GenerateContentResponse, error) {
			if model != "test-example-model" {
				t.Errorf("expected example model, got %q", model)
			}
			rawResults := []struct {
				SegmentID string `json:"segment_id"`
				Examples  []struct {
					Given string `json:"given"`
					When  string `json:"when"`
					Then  string `json:"then"`
				} `json:"examples"`
			}{
				{
					SegmentID: "seg-0001",
					Examples: []struct {
						Given string `json:"given"`
						When  string `json:"when"`
						Then  string `json:"then"`
					}{
						{Given: "ユーザーが存在する", When: "ログインする", Then: "成功"},
					},
				},
				{
					SegmentID: "seg-0002",
					Examples: []struct {
						Given string `json:"given"`
						When  string `json:"when"`
						Then  string `json:"then"`
					}{
						{Given: "メールアドレスが未登録", When: "登録する", Then: "アカウント作成"},
					},
				},
			}
			b, _ := json.Marshal(rawResults)
			return string(b), &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
			}, nil
		})

		results, err := e.BatchGenerateExamples(context.Background(), segments, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("results count = %d, want 2", len(results))
		}
		if len(results[0].Examples) != 1 {
			t.Errorf("results[0].Examples count = %d, want 1", len(results[0].Examples))
		}
		if results[0].Examples[0].Given != "ユーザーが存在する" {
			t.Errorf("Given = %q", results[0].Examples[0].Given)
		}
	})

	t.Run("empty GWT fields are filtered", func(t *testing.T) {
		e := newTestBatchEnricher(func(_ context.Context, _, _ string, _ *genai.Schema) (string, *genai.GenerateContentResponse, error) {
			rawResults := []struct {
				SegmentID string `json:"segment_id"`
				Examples  []struct {
					Given string `json:"given"`
					When  string `json:"when"`
					Then  string `json:"then"`
				} `json:"examples"`
			}{
				{
					SegmentID: "seg-0001",
					Examples: []struct {
						Given string `json:"given"`
						When  string `json:"when"`
						Then  string `json:"then"`
					}{
						{Given: "条件", When: "操作", Then: "結果"},
						{Given: "", When: "操作", Then: "結果"},
					},
				},
				{
					SegmentID: "seg-0002",
					Examples: []struct {
						Given string `json:"given"`
						When  string `json:"when"`
						Then  string `json:"then"`
					}{},
				},
			}
			b, _ := json.Marshal(rawResults)
			return string(b), &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
			}, nil
		})

		results, err := e.BatchGenerateExamples(context.Background(), segments, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results[0].Examples) != 1 {
			t.Errorf("results[0].Examples count = %d, want 1 (empty filtered)", len(results[0].Examples))
		}
	})

	t.Run("truncated response returns ErrBatchTruncated", func(t *testing.T) {
		e := newTestBatchEnricher(func(_ context.Context, _, _ string, _ *genai.Schema) (string, *genai.GenerateContentResponse, error) {
			// Only 1 result for 2 segments
			rawResults := []struct {
				SegmentID string `json:"segment_id"`
				Examples  []struct {
					Given string `json:"given"`
					When  string `json:"when"`
					Then  string `json:"then"`
				} `json:"examples"`
			}{
				{SegmentID: "seg-0001", Examples: nil},
			}
			b, _ := json.Marshal(rawResults)
			return string(b), &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
			}, nil
		})

		results, err := e.BatchGenerateExamples(context.Background(), segments, nil)
		if !errors.Is(err, ErrBatchTruncated) {
			t.Errorf("expected ErrBatchTruncated, got %v", err)
		}
		if len(results) != 1 {
			t.Errorf("results count = %d, want 1", len(results))
		}
	})

	t.Run("empty segments returns nil", func(t *testing.T) {
		e := newTestBatchEnricher(nil)
		results, err := e.BatchGenerateExamples(context.Background(), nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results != nil {
			t.Errorf("expected nil results, got %v", results)
		}
	})

	t.Run("context segments included in prompt", func(t *testing.T) {
		var capturedPrompt string
		e := newTestBatchEnricher(func(_ context.Context, _, prompt string, _ *genai.Schema) (string, *genai.GenerateContentResponse, error) {
			capturedPrompt = prompt
			rawResults := []struct {
				SegmentID string `json:"segment_id"`
				Examples  []struct {
					Given string `json:"given"`
					When  string `json:"when"`
					Then  string `json:"then"`
				} `json:"examples"`
			}{
				{SegmentID: "seg-0001", Examples: nil},
				{SegmentID: "seg-0002", Examples: nil},
			}
			b, _ := json.Marshal(rawResults)
			return string(b), &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
			}, nil
		})

		ctxSegs := []*kire.Segment{
			{Meta: kire.SegmentMeta{SegmentID: "ctx-0001"}, Content: "# 共通仕様\nContent-Type text/plain は 415 を返す"},
		}

		_, err := e.BatchGenerateExamples(context.Background(), segments, ctxSegs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(capturedPrompt, "共通仕様（各セグメントへの適用必須") {
			t.Error("prompt should contain context section header")
		}
		if !strings.Contains(capturedPrompt, "ctx-0001") {
			t.Error("prompt should contain context segment ID")
		}
		if !strings.Contains(capturedPrompt, "415") {
			t.Error("prompt should contain context segment content")
		}
	})

	t.Run("nil context produces no context section", func(t *testing.T) {
		var capturedPrompt string
		e := newTestBatchEnricher(func(_ context.Context, _, prompt string, _ *genai.Schema) (string, *genai.GenerateContentResponse, error) {
			capturedPrompt = prompt
			rawResults := []struct {
				SegmentID string `json:"segment_id"`
				Examples  []struct {
					Given string `json:"given"`
					When  string `json:"when"`
					Then  string `json:"then"`
				} `json:"examples"`
			}{
				{SegmentID: "seg-0001", Examples: nil},
				{SegmentID: "seg-0002", Examples: nil},
			}
			b, _ := json.Marshal(rawResults)
			return string(b), &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
			}, nil
		})

		_, err := e.BatchGenerateExamples(context.Background(), segments, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(capturedPrompt, "共通仕様（参照用") {
			t.Error("prompt should not contain context section when contextSegments is nil")
		}
	})
}

func TestFormatSegmentsWithContext(t *testing.T) {
	t.Run("formatSegmentsForClassify includes context", func(t *testing.T) {
		segments := []*kire.Segment{
			{Meta: kire.SegmentMeta{SegmentID: "seg-0001"}, Content: "# Login", Context: "Task管理API仕様書"},
			{Meta: kire.SegmentMeta{SegmentID: "seg-0002"}, Content: "# Register", Context: ""},
		}

		result := formatSegmentsForClassify(segments)
		if !strings.Contains(result, "segment_id: seg-0001 | context: Task管理API仕様書 ---") {
			t.Errorf("expected context in header for seg-0001, got: %s", result)
		}
		// seg-0002 has no context, should not include context field
		if strings.Contains(result, "seg-0002 | context:") {
			t.Errorf("seg-0002 should not have context field, got: %s", result)
		}
	})

	t.Run("formatSegmentsForExamples includes context", func(t *testing.T) {
		segments := []*kire.Segment{
			{Meta: kire.SegmentMeta{SegmentID: "seg-0001"}, Content: "# Login", Context: "Task管理API仕様書"},
			{Meta: kire.SegmentMeta{SegmentID: "seg-0002"}, Content: "# Register"},
		}
		titles := map[string]string{
			"seg-0001": "ログイン",
			"seg-0002": "登録",
		}

		result := formatSegmentsForExamples(segments, titles)
		if !strings.Contains(result, "segment_id: seg-0001 | title: ログイン | context: Task管理API仕様書 ---") {
			t.Errorf("expected context in header for seg-0001, got: %s", result)
		}
		if strings.Contains(result, "seg-0002 | title: 登録 | context:") {
			t.Errorf("seg-0002 should not have context field, got: %s", result)
		}
	})
}

func TestIsTruncatedResponse(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		if isTruncatedResponse(nil) {
			t.Error("nil response should not be truncated")
		}
	})

	t.Run("no candidates", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{}
		if isTruncatedResponse(resp) {
			t.Error("empty candidates should not be truncated")
		}
	})

	t.Run("MaxTokens is truncated", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonMaxTokens}},
		}
		if !isTruncatedResponse(resp) {
			t.Error("MaxTokens should be truncated")
		}
	})

	t.Run("Stop is not truncated", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
		}
		if isTruncatedResponse(resp) {
			t.Error("Stop should not be truncated")
		}
	})
}
