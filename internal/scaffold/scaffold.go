package scaffold

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns a title into a file-safe slug.
func Slugify(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "spec"
	}
	return s
}

// RenderTest renders a test file for a spec.
func RenderTest(s *spec.Spec, runner string) string {
	var sb strings.Builder

	if runner == "vitest" {
		sb.WriteString("import { describe, it } from \"vitest\"\n\n")
	}

	desc := fmt.Sprintf("%s: %s", s.ID, s.Title)
	sb.WriteString(fmt.Sprintf("describe(%q, () => {\n", desc))

	examples := s.Examples
	if len(examples) == 0 {
		examples = []spec.Example{{ID: "E1", Given: "TODO", When: "TODO", Then: "TODO: add examples"}}
	}

	for i, ex := range examples {
		exID := strings.TrimSpace(ex.ID)
		if exID == "" {
			exID = fmt.Sprintf("E%d", i+1)
		}
		name := fmt.Sprintf("%s %s: %s", s.ID, exID, ex.Then)
		sb.WriteString(fmt.Sprintf("  it(%q, () => {\n", name))
		sb.WriteString(fmt.Sprintf("    // Given: %s\n", ex.Given))
		sb.WriteString(fmt.Sprintf("    // When: %s\n", ex.When))
		sb.WriteString(fmt.Sprintf("    // Then: %s\n", ex.Then))
		sb.WriteString("    throw new Error(\"TODO: implement\")\n")
		sb.WriteString("  })\n\n")
	}

	sb.WriteString("})\n")
	return sb.String()
}

// ApplyPattern applies a filename pattern.
func ApplyPattern(pattern, id, slug string) string {
	out := strings.ReplaceAll(pattern, "{{id}}", id)
	out = strings.ReplaceAll(out, "{{slug}}", slug)
	return out
}
