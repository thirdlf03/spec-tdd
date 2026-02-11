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

// batchGenerateFunc はバッチ API 呼び出しを抽象化する関数型。
// model 名を受け取り、指定モデルで生成する。
type batchGenerateFunc func(ctx context.Context, model string, prompt string, schema *genai.Schema) (string, *genai.GenerateContentResponse, error)

// GeminiBatchEnricherConfig は GeminiBatchEnricher の設定。
type GeminiBatchEnricherConfig struct {
	APIKey          string
	ClassifyModel   string        // default: "gemini-2.5-flash-lite"
	ExampleModel    string        // default: "gemini-2.5-flash"
	ClassifyTimeout time.Duration // default: 60s
	ExampleTimeout  time.Duration // default: 120s
	MaxRetries      int           // default: 2
}

// GeminiBatchEnricher は Gemini API を使ったバッチ Enricher の実装。
type GeminiBatchEnricher struct {
	cfg      GeminiBatchEnricherConfig
	generate batchGenerateFunc
}

// NewGeminiBatchEnricher は Gemini API を使ったバッチ Enricher を生成する。
func NewGeminiBatchEnricher(cfg GeminiBatchEnricherConfig) (BatchEnricher, error) {
	if cfg.APIKey == "" {
		return nil, apperrors.New("enrich.NewGeminiBatchEnricher", apperrors.ErrInvalidInput,
			"GEMINI_API_KEY is required. Set the environment variable: export GEMINI_API_KEY=your-key")
	}

	if cfg.ClassifyModel == "" {
		cfg.ClassifyModel = "gemini-2.5-flash-lite"
	}
	if cfg.ExampleModel == "" {
		cfg.ExampleModel = "gemini-2.5-flash"
	}
	if cfg.ClassifyTimeout == 0 {
		cfg.ClassifyTimeout = 60 * time.Second
	}
	if cfg.ExampleTimeout == 0 {
		cfg.ExampleTimeout = 180 * time.Second
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
		return nil, apperrors.Wrap("enrich.NewGeminiBatchEnricher", err)
	}

	temp := float32(0)
	gen := func(ctx context.Context, model string, prompt string, schema *genai.Schema) (string, *genai.GenerateContentResponse, error) {
		result, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), &genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   schema,
			Temperature:      &temp,
		})
		if err != nil {
			return "", nil, err
		}
		if result == nil || len(result.Candidates) == 0 || result.Candidates[0].Content == nil || len(result.Candidates[0].Content.Parts) == 0 {
			return "", result, fmt.Errorf("empty response from Gemini API")
		}
		return result.Candidates[0].Content.Parts[0].Text, result, nil
	}

	return &GeminiBatchEnricher{
		cfg:      cfg,
		generate: gen,
	}, nil
}

// BatchClassify は全セグメントをバッチ分類する。
func (e *GeminiBatchEnricher) BatchClassify(ctx context.Context, segments []*kire.Segment) ([]BatchClassifyResult, error) {
	if len(segments) == 0 {
		return nil, nil
	}

	prompt := fmt.Sprintf(batchClassifyPrompt, formatSegmentsForClassify(segments))

	var responseText string
	var resp *genai.GenerateContentResponse
	var lastErr error

	for attempt := 0; attempt <= e.cfg.MaxRetries; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, e.cfg.ClassifyTimeout)
		text, r, err := e.generate(callCtx, e.cfg.ClassifyModel, prompt, batchClassifySchema)
		cancel()

		if err == nil {
			responseText = text
			resp = r
			lastErr = nil
			break
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, apperrors.Wrapf("enrich.GeminiBatchEnricher.BatchClassify", lastErr,
			"all retries failed for batch classify (%d segments)", len(segments))
	}

	var results []BatchClassifyResult
	if err := json.Unmarshal([]byte(responseText), &results); err != nil {
		return nil, apperrors.Wrapf("enrich.GeminiBatchEnricher.BatchClassify", err,
			"failed to parse JSON response")
	}

	// Normalize categories
	for i := range results {
		results[i].Category = NormalizeCategory(string(results[i].Category))
		results[i].ReqID = strings.TrimSpace(results[i].ReqID)
		results[i].Title = strings.TrimSpace(results[i].Title)
	}

	// 切断検出: レスポンスの segment_id 数 < 入力数
	if len(results) < len(segments) {
		return results, ErrBatchTruncated
	}
	if isTruncatedResponse(resp) {
		return results, ErrBatchTruncated
	}

	return results, nil
}

// BatchGenerateExamples は FR/NFR セグメントのバッチ Example 生成を行う。
// contextSegments は参照用の共通仕様セグメント（overview 等）。nil 可。
func (e *GeminiBatchEnricher) BatchGenerateExamples(ctx context.Context, segments []*kire.Segment, contextSegments []*kire.Segment) ([]BatchExampleResult, error) {
	if len(segments) == 0 {
		return nil, nil
	}

	// タイトルマップ（formatSegmentsForExamples で使用）
	titles := make(map[string]string, len(segments))
	for _, seg := range segments {
		if len(seg.Meta.HeadingPath) > 0 {
			titles[seg.Meta.SegmentID] = seg.Meta.HeadingPath[len(seg.Meta.HeadingPath)-1]
		}
	}

	contextSection := formatContextSection(contextSegments)
	prompt := fmt.Sprintf(batchExamplesPrompt, contextSection, formatSegmentsForExamples(segments, titles))

	var responseText string
	var resp *genai.GenerateContentResponse
	var lastErr error

	for attempt := 0; attempt <= e.cfg.MaxRetries; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, e.cfg.ExampleTimeout)
		text, r, err := e.generate(callCtx, e.cfg.ExampleModel, prompt, batchExamplesSchema)
		cancel()

		if err == nil {
			responseText = text
			resp = r
			lastErr = nil
			break
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, apperrors.Wrapf("enrich.GeminiBatchEnricher.BatchGenerateExamples", lastErr,
			"all retries failed for batch examples (%d segments)", len(segments))
	}

	var rawResults []struct {
		SegmentID string `json:"segment_id"`
		Examples  []struct {
			Given string `json:"given"`
			When  string `json:"when"`
			Then  string `json:"then"`
		} `json:"examples"`
	}
	if err := json.Unmarshal([]byte(responseText), &rawResults); err != nil {
		return nil, apperrors.Wrapf("enrich.GeminiBatchEnricher.BatchGenerateExamples", err,
			"failed to parse JSON response")
	}

	results := make([]BatchExampleResult, 0, len(rawResults))
	for _, raw := range rawResults {
		var examples []spec.Example
		for _, ex := range raw.Examples {
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
		results = append(results, BatchExampleResult{
			SegmentID: raw.SegmentID,
			Examples:  examples,
		})
	}

	// 切断検出
	if len(results) < len(segments) {
		return results, ErrBatchTruncated
	}
	if isTruncatedResponse(resp) {
		return results, ErrBatchTruncated
	}

	return results, nil
}

// isTruncatedResponse はレスポンスが MaxTokens で切断されたかを判定する。
func isTruncatedResponse(resp *genai.GenerateContentResponse) bool {
	if resp == nil || len(resp.Candidates) == 0 {
		return false
	}
	reason := resp.Candidates[0].FinishReason
	return reason == genai.FinishReasonMaxTokens
}
