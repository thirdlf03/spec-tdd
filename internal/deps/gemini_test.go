package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func TestGeminiDetector_Success(t *testing.T) {
	response := []DepsResult{
		{ID: "REQ-002", Depends: []string{"REQ-001"}, Reason: "data dependency"},
	}
	respJSON, _ := json.Marshal(response)

	gen := func(_ context.Context, _ string) (string, error) {
		return string(respJSON), nil
	}

	d := newGeminiDetectorWithFunc(GeminiDetectorConfig{}, gen)
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A"},
		{ID: "REQ-002", Title: "B"},
	}

	results, err := d.Detect(context.Background(), specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 || results[0].ID != "REQ-002" || results[0].Depends[0] != "REQ-001" {
		t.Fatalf("unexpected results: %v", results)
	}
}

func TestGeminiDetector_FilterSelfReference(t *testing.T) {
	response := []DepsResult{
		{ID: "REQ-001", Depends: []string{"REQ-001", "REQ-002"}, Reason: "test"},
	}
	respJSON, _ := json.Marshal(response)

	gen := func(_ context.Context, _ string) (string, error) {
		return string(respJSON), nil
	}

	d := newGeminiDetectorWithFunc(GeminiDetectorConfig{}, gen)
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A"},
		{ID: "REQ-002", Title: "B"},
	}

	results, err := d.Detect(context.Background(), specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Depends) != 1 || results[0].Depends[0] != "REQ-002" {
		t.Fatalf("self-reference should be filtered, got %v", results[0].Depends)
	}
}

func TestGeminiDetector_FilterNonExistent(t *testing.T) {
	response := []DepsResult{
		{ID: "REQ-001", Depends: []string{"REQ-999"}, Reason: "test"},
	}
	respJSON, _ := json.Marshal(response)

	gen := func(_ context.Context, _ string) (string, error) {
		return string(respJSON), nil
	}

	d := newGeminiDetectorWithFunc(GeminiDetectorConfig{}, gen)
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A"},
	}

	results, err := d.Detect(context.Background(), specs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results (dep filtered), got %d", len(results))
	}
}

func TestGeminiDetector_Retry(t *testing.T) {
	callCount := 0
	response := []DepsResult{
		{ID: "REQ-001", Depends: []string{"REQ-002"}, Reason: "test"},
	}
	respJSON, _ := json.Marshal(response)

	gen := func(_ context.Context, _ string) (string, error) {
		callCount++
		if callCount < 3 {
			return "", fmt.Errorf("transient error")
		}
		return string(respJSON), nil
	}

	d := newGeminiDetectorWithFunc(GeminiDetectorConfig{MaxRetries: 2}, gen)
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A"},
		{ID: "REQ-002", Title: "B"},
	}

	results, err := d.Detect(context.Background(), specs)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if callCount != 3 {
		t.Fatalf("expected 3 calls, got %d", callCount)
	}
}

func TestGeminiDetector_AllRetriesFail(t *testing.T) {
	gen := func(_ context.Context, _ string) (string, error) {
		return "", fmt.Errorf("persistent error")
	}

	d := newGeminiDetectorWithFunc(GeminiDetectorConfig{MaxRetries: 1}, gen)
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A"},
	}

	_, err := d.Detect(context.Background(), specs)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGeminiDetector_InvalidJSON(t *testing.T) {
	gen := func(_ context.Context, _ string) (string, error) {
		return "not json", nil
	}

	d := newGeminiDetectorWithFunc(GeminiDetectorConfig{}, gen)
	specs := []*spec.Spec{
		{ID: "REQ-001", Title: "A"},
	}

	_, err := d.Detect(context.Background(), specs)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestGeminiDetector_EmptySpecs(t *testing.T) {
	gen := func(_ context.Context, _ string) (string, error) {
		return "[]", nil
	}

	d := newGeminiDetectorWithFunc(GeminiDetectorConfig{}, gen)

	results, err := d.Detect(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results for empty specs, got %v", results)
	}
}
