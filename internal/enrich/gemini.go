package enrich

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/thirdlf03/spec-tdd/internal/apperrors"
	"github.com/thirdlf03/spec-tdd/internal/kire"
	"github.com/thirdlf03/spec-tdd/internal/spec"
	"google.golang.org/genai"
)

// generateFunc は Gemini API 呼び出しを抽象化する関数型。
// テスト時にモック差し替え可能。
type generateFunc func(ctx context.Context, prompt string) (string, error)

// GeminiEnricherConfig は GeminiEnricher の設定。
type GeminiEnricherConfig struct {
	APIKey     string
	Model      string
	Timeout    time.Duration
	MaxRetries int
}

// GeminiEnricher は Gemini API を使った Enricher の実装。
type GeminiEnricher struct {
	cfg      GeminiEnricherConfig
	generate generateFunc
}

// NewGeminiEnricher は Gemini API を使った Enricher を生成する。
// APIKey が空の場合はエラーを返す。
func NewGeminiEnricher(cfg GeminiEnricherConfig) (Enricher, error) {
	if cfg.APIKey == "" {
		return nil, apperrors.New("enrich.NewGeminiEnricher", apperrors.ErrInvalidInput,
			"GEMINI_API_KEY is required. Set the environment variable: export GEMINI_API_KEY=your-key")
	}

	if cfg.Model == "" {
		cfg.Model = "gemini-2.5-flash-lite"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 2
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, apperrors.Wrap("enrich.NewGeminiEnricher", err)
	}

	temp := float32(0)
	gen := func(ctx context.Context, prompt string) (string, error) {
		result, err := client.Models.GenerateContent(ctx, cfg.Model, genai.Text(prompt), &genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   enrichResponseSchema,
			Temperature:      &temp,
		})
		if err != nil {
			return "", err
		}
		if result == nil || len(result.Candidates) == 0 || result.Candidates[0].Content == nil || len(result.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("empty response from Gemini API")
		}
		return result.Candidates[0].Content.Parts[0].Text, nil
	}

	return &GeminiEnricher{
		cfg:      cfg,
		generate: gen,
	}, nil
}

// enrichResponse は Gemini API のレスポンスをパースするための構造体。
type enrichResponse struct {
	Category string `json:"category"`
	ReqID    string `json:"req_id"`
	Title    string `json:"title"`
	Examples []struct {
		Given string `json:"given"`
		When  string `json:"when"`
		Then  string `json:"then"`
	} `json:"examples"`
}

// Enrich はセグメントを解析し、分類・メタデータ抽出・GWT 生成を行う。
func (e *GeminiEnricher) Enrich(ctx context.Context, segment *kire.Segment, contextSegments []*kire.Segment) (*EnrichResult, error) {
	contextSection := formatContextSection(contextSegments)
	prompt := fmt.Sprintf(classifyAndEnrichPrompt, contextSection, segment.Content)

	var responseText string
	var lastErr error

	for attempt := 0; attempt <= e.cfg.MaxRetries; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, e.cfg.Timeout)
		text, err := e.generate(callCtx, prompt)
		cancel()

		if err == nil {
			responseText = text
			lastErr = nil
			break
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, apperrors.Wrapf("enrich.GeminiEnricher.Enrich", lastErr,
			"all retries failed for segment %s", segment.Meta.SegmentID)
	}

	var resp enrichResponse
	if err := json.Unmarshal([]byte(responseText), &resp); err != nil {
		return nil, apperrors.Wrapf("enrich.GeminiEnricher.Enrich", err,
			"failed to parse JSON response for segment %s", segment.Meta.SegmentID)
	}

	category := NormalizeCategory(resp.Category)

	var examples []spec.Example
	for _, ex := range resp.Examples {
		given := strings.TrimSpace(ex.Given)
		when := strings.TrimSpace(ex.When)
		then := strings.TrimSpace(ex.Then)
		if given == "" || when == "" || then == "" {
			continue
		}
		examples = append(examples, spec.Example{
			Given: given,
			When:  when,
			Then:  then,
		})
	}

	return &EnrichResult{
		Category: category,
		ReqID:    strings.TrimSpace(resp.ReqID),
		Title:    strings.TrimSpace(resp.Title),
		Examples: examples,
	}, nil
}
