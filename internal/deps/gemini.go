package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/thirdlf03/spec-tdd/internal/apperrors"
	"github.com/thirdlf03/spec-tdd/internal/spec"
	"google.golang.org/genai"
)

// depsGenerateFunc is an abstracted Gemini API call function for testing.
type depsGenerateFunc func(ctx context.Context, prompt string) (string, error)

// GeminiDetectorConfig holds configuration for GeminiDetector.
type GeminiDetectorConfig struct {
	APIKey     string
	Model      string
	Timeout    time.Duration
	MaxRetries int
}

// GeminiDetector uses Gemini API for dependency detection.
type GeminiDetector struct {
	cfg      GeminiDetectorConfig
	generate depsGenerateFunc
}

// NewGeminiDetector creates a new GeminiDetector.
func NewGeminiDetector(cfg GeminiDetectorConfig) (Detector, error) {
	if cfg.APIKey == "" {
		return nil, apperrors.New("deps.NewGeminiDetector", apperrors.ErrInvalidInput,
			"GEMINI_API_KEY is required")
	}
	if cfg.Model == "" {
		cfg.Model = "gemini-2.5-flash"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
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
		return nil, apperrors.Wrap("deps.NewGeminiDetector", err)
	}

	gen := func(ctx context.Context, prompt string) (string, error) {
		result, err := client.Models.GenerateContent(ctx, cfg.Model, genai.Text(prompt), &genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   depsResponseSchema,
		})
		if err != nil {
			return "", err
		}
		if result == nil || len(result.Candidates) == 0 || result.Candidates[0].Content == nil || len(result.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("empty response from Gemini API")
		}
		return result.Candidates[0].Content.Parts[0].Text, nil
	}

	return &GeminiDetector{
		cfg:      cfg,
		generate: gen,
	}, nil
}

// newGeminiDetectorWithFunc creates a GeminiDetector with a custom generate function (for testing).
func newGeminiDetectorWithFunc(cfg GeminiDetectorConfig, gen depsGenerateFunc) *GeminiDetector {
	if cfg.Model == "" {
		cfg.Model = "gemini-2.5-flash"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 2
	}
	return &GeminiDetector{cfg: cfg, generate: gen}
}

// Detect calls the Gemini API to detect dependencies between specs.
func (d *GeminiDetector) Detect(ctx context.Context, specs []*spec.Spec) ([]DepsResult, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	prompt := fmt.Sprintf(depsDetectPrompt, formatSpecsForDeps(specs))

	var responseText string
	var lastErr error

	for attempt := 0; attempt <= d.cfg.MaxRetries; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, d.cfg.Timeout)
		text, err := d.generate(callCtx, prompt)
		cancel()

		if err == nil {
			responseText = text
			lastErr = nil
			break
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, apperrors.Wrapf("deps.GeminiDetector.Detect", lastErr,
			"all retries failed")
	}

	var rawResults []DepsResult
	if err := json.Unmarshal([]byte(responseText), &rawResults); err != nil {
		return nil, apperrors.Wrapf("deps.GeminiDetector.Detect", err,
			"failed to parse JSON response")
	}

	// Build valid ID set
	idSet := make(map[string]bool, len(specs))
	for _, s := range specs {
		idSet[s.ID] = true
	}

	reqIDPat := regexp.MustCompile(`^REQ-\d+$`)

	// Filter and validate results
	var results []DepsResult
	for _, r := range rawResults {
		if !idSet[r.ID] {
			continue
		}
		var validDeps []string
		for _, dep := range r.Depends {
			dep = strings.TrimSpace(dep)
			if dep == r.ID {
				continue // self-reference
			}
			if !idSet[dep] {
				continue // non-existent
			}
			if !reqIDPat.MatchString(dep) {
				continue // invalid format
			}
			validDeps = append(validDeps, dep)
		}
		if len(validDeps) > 0 {
			results = append(results, DepsResult{
				ID:      r.ID,
				Depends: validDeps,
				Reason:  r.Reason,
			})
		}
	}

	return results, nil
}
