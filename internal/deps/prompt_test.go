package deps

import (
	"strings"
	"testing"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

func TestFormatSpecsForDeps(t *testing.T) {
	specs := []*spec.Spec{
		{
			ID:          "REQ-001",
			Title:       "ユーザー登録",
			Description: "新規ユーザーを登録する",
			Examples: []spec.Example{
				{Given: "valid email", When: "POST /users", Then: "201 created"},
			},
		},
		{
			ID:    "REQ-002",
			Title: "ログイン",
		},
	}

	result := formatSpecsForDeps(specs)

	if !strings.Contains(result, "REQ-001: ユーザー登録") {
		t.Errorf("expected REQ-001 header, got:\n%s", result)
	}
	if !strings.Contains(result, "新規ユーザーを登録する") {
		t.Errorf("expected description, got:\n%s", result)
	}
	if !strings.Contains(result, "Given: valid email") {
		t.Errorf("expected example, got:\n%s", result)
	}
	if !strings.Contains(result, "REQ-002: ログイン") {
		t.Errorf("expected REQ-002 header, got:\n%s", result)
	}
}

func TestFormatSpecsForDeps_Empty(t *testing.T) {
	result := formatSpecsForDeps(nil)
	if result != "" {
		t.Errorf("expected empty string, got: %q", result)
	}
}
