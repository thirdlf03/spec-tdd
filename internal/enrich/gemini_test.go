package enrich

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/thirdlf03/spec-tdd/internal/kire"
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func TestNewGeminiEnricher(t *testing.T) {
	t.Run("empty API key returns error", func(t *testing.T) {
		_, err := NewGeminiEnricher(GeminiEnricherConfig{
			APIKey: "",
			Model:  "gemini-2.5-flash-lite",
		})
		if err == nil {
			t.Fatal("expected error for empty API key")
		}
	})

	t.Run("valid config returns enricher", func(t *testing.T) {
		e, err := NewGeminiEnricher(GeminiEnricherConfig{
			APIKey: "test-key",
			Model:  "gemini-2.5-flash-lite",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e == nil {
			t.Fatal("expected non-nil enricher")
		}
	})

	t.Run("defaults are applied", func(t *testing.T) {
		e, err := NewGeminiEnricher(GeminiEnricherConfig{
			APIKey: "test-key",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ge, ok := e.(*GeminiEnricher)
		if !ok {
			t.Fatal("expected *GeminiEnricher type")
		}
		if ge.cfg.Model != "gemini-2.5-flash-lite" {
			t.Errorf("Model = %q, want %q", ge.cfg.Model, "gemini-2.5-flash-lite")
		}
		if ge.cfg.Timeout != 30*time.Second {
			t.Errorf("Timeout = %v, want %v", ge.cfg.Timeout, 30*time.Second)
		}
		if ge.cfg.MaxRetries != 2 {
			t.Errorf("MaxRetries = %d, want %d", ge.cfg.MaxRetries, 2)
		}
	})
}

func TestGeminiEnricher_Enrich(t *testing.T) {
	seg := &kire.Segment{
		Meta: kire.SegmentMeta{
			SegmentID:   "seg-0001",
			HeadingPath: []string{"Doc", "Login"},
		},
		Content: "### REQ-001: ユーザーログイン\n\n正常系: ユーザーがログインする\n異常系: パスワードが間違っている",
	}

	t.Run("successful enrichment with functional requirement", func(t *testing.T) {
		e := newTestEnricher(func(ctx context.Context, prompt string) (string, error) {
			return `{"category":"functional_requirement","req_id":"REQ-001","title":"ユーザーログイン","examples":[{"given":"ユーザーが存在する","when":"正しいパスワードでログインする","then":"認証トークンが返却される"},{"given":"ユーザーが存在する","when":"間違ったパスワードでログインする","then":"エラーメッセージが表示される"}]}`, nil
		})

		result, err := e.Enrich(context.Background(), seg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Category != CategoryFunctionalRequirement {
			t.Errorf("Category = %q, want %q", result.Category, CategoryFunctionalRequirement)
		}
		if result.ReqID != "REQ-001" {
			t.Errorf("ReqID = %q, want %q", result.ReqID, "REQ-001")
		}
		if result.Title != "ユーザーログイン" {
			t.Errorf("Title = %q, want %q", result.Title, "ユーザーログイン")
		}
		if len(result.Examples) != 2 {
			t.Fatalf("Examples count = %d, want 2", len(result.Examples))
		}
	})

	t.Run("overview segment is classified correctly", func(t *testing.T) {
		e := newTestEnricher(func(ctx context.Context, prompt string) (string, error) {
			return `{"category":"overview","req_id":"","title":"プロジェクト概要","examples":[]}`, nil
		})

		result, err := e.Enrich(context.Background(), seg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Category != CategoryOverview {
			t.Errorf("Category = %q, want %q", result.Category, CategoryOverview)
		}
		if result.IsRequirement() {
			t.Error("overview should not be a requirement")
		}
	})

	t.Run("unknown category normalized to other", func(t *testing.T) {
		e := newTestEnricher(func(ctx context.Context, prompt string) (string, error) {
			return `{"category":"unknown_type","req_id":"","title":"不明","examples":[]}`, nil
		})

		result, err := e.Enrich(context.Background(), seg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Category != CategoryOther {
			t.Errorf("Category = %q, want %q", result.Category, CategoryOther)
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		e := newTestEnricher(func(ctx context.Context, prompt string) (string, error) {
			return `{invalid json`, nil
		})

		_, err := e.Enrich(context.Background(), seg, nil)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("empty GWT fields are filtered out", func(t *testing.T) {
		e := newTestEnricher(func(ctx context.Context, prompt string) (string, error) {
			return `{"category":"functional_requirement","req_id":"","title":"テスト","examples":[{"given":"条件","when":"操作","then":"結果"},{"given":"","when":"操作","then":"結果"},{"given":"条件","when":"","then":"結果"}]}`, nil
		})

		result, err := e.Enrich(context.Background(), seg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Examples) != 1 {
			t.Errorf("Examples count = %d, want 1 (empty fields filtered)", len(result.Examples))
		}
	})

	t.Run("retry on API error", func(t *testing.T) {
		callCount := 0
		e := newTestEnricher(func(ctx context.Context, prompt string) (string, error) {
			callCount++
			if callCount <= 2 {
				return "", context.DeadlineExceeded
			}
			return `{"category":"functional_requirement","req_id":"","title":"テスト","examples":[]}`, nil
		})

		result, err := e.Enrich(context.Background(), seg, nil)
		if err != nil {
			t.Fatalf("unexpected error after retries: %v", err)
		}
		if callCount != 3 {
			t.Errorf("callCount = %d, want 3 (1 initial + 2 retries)", callCount)
		}
		if result.Category != CategoryFunctionalRequirement {
			t.Errorf("Category = %q, want %q", result.Category, CategoryFunctionalRequirement)
		}
	})

	t.Run("all retries fail returns error", func(t *testing.T) {
		e := newTestEnricher(func(ctx context.Context, prompt string) (string, error) {
			return "", context.DeadlineExceeded
		})

		_, err := e.Enrich(context.Background(), seg, nil)
		if err == nil {
			t.Fatal("expected error when all retries fail")
		}
	})

	t.Run("examples with valid Example fields", func(t *testing.T) {
		e := newTestEnricher(func(ctx context.Context, prompt string) (string, error) {
			return `{"category":"functional_requirement","req_id":"REQ-001","title":"ログイン","examples":[{"given":"ユーザーが存在する","when":"ログインする","then":"成功する"}]}`, nil
		})

		result, err := e.Enrich(context.Background(), seg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Examples) != 1 {
			t.Fatalf("Examples count = %d, want 1", len(result.Examples))
		}
		ex := result.Examples[0]
		if ex.Given != "ユーザーが存在する" {
			t.Errorf("Given = %q", ex.Given)
		}
		if ex.When != "ログインする" {
			t.Errorf("When = %q", ex.When)
		}
		if ex.Then != "成功する" {
			t.Errorf("Then = %q", ex.Then)
		}
	})

	t.Run("context segments included in prompt", func(t *testing.T) {
		var capturedPrompt string
		e := newTestEnricher(func(_ context.Context, prompt string) (string, error) {
			capturedPrompt = prompt
			return `{"category":"functional_requirement","req_id":"","title":"テスト","examples":[]}`, nil
		})

		ctxSegs := []*kire.Segment{
			{Meta: kire.SegmentMeta{SegmentID: "ctx-0001"}, Content: "# 共通仕様\nContent-Type text/plain は 415 を返す"},
		}

		_, err := e.Enrich(context.Background(), seg, ctxSegs)
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

	t.Run("nil context produces no context section in prompt", func(t *testing.T) {
		var capturedPrompt string
		e := newTestEnricher(func(_ context.Context, prompt string) (string, error) {
			capturedPrompt = prompt
			return `{"category":"functional_requirement","req_id":"","title":"テスト","examples":[]}`, nil
		})

		_, err := e.Enrich(context.Background(), seg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(capturedPrompt, "共通仕様（参照用") {
			t.Error("prompt should not contain context section when contextSegments is nil")
		}
	})
}

// newTestEnricher creates a GeminiEnricher with a mock generateFunc.
func newTestEnricher(fn generateFunc) *GeminiEnricher {
	return &GeminiEnricher{
		cfg: GeminiEnricherConfig{
			APIKey:     "test-key",
			Model:      "gemini-2.5-flash-lite",
			Timeout:    30 * time.Second,
			MaxRetries: 2,
		},
		generate: fn,
	}
}

// verify MockEnricher satisfies Enricher interface
var _ Enricher = (*MockEnricher)(nil)

// verify concrete examples type
func TestEnrichResultExamplesAreSpecExamples(t *testing.T) {
	r := &EnrichResult{
		Examples: []spec.Example{
			{Given: "a", When: "b", Then: "c"},
		},
	}
	if len(r.Examples) != 1 {
		t.Fatalf("Examples count = %d, want 1", len(r.Examples))
	}
}
