package enrich

import (
	"strings"
	"testing"
)

func TestClassifyAndEnrichPrompt(t *testing.T) {
	if classifyAndEnrichPrompt == "" {
		t.Fatal("classifyAndEnrichPrompt should not be empty")
	}
	if !strings.Contains(classifyAndEnrichPrompt, "%s") {
		t.Error("classifyAndEnrichPrompt should contain a format placeholder for segment content")
	}
	if !strings.Contains(classifyAndEnrichPrompt, "functional_requirement") {
		t.Error("classifyAndEnrichPrompt should mention functional_requirement category")
	}
	if !strings.Contains(classifyAndEnrichPrompt, "Given") {
		t.Error("classifyAndEnrichPrompt should mention Given/When/Then")
	}
}

func TestEnrichResponseSchema(t *testing.T) {
	if enrichResponseSchema == nil {
		t.Fatal("enrichResponseSchema should not be nil")
	}

	props, ok := enrichResponseSchema.Properties["category"]
	if !ok || props == nil {
		t.Error("enrichResponseSchema should have 'category' property")
	}

	props, ok = enrichResponseSchema.Properties["title"]
	if !ok || props == nil {
		t.Error("enrichResponseSchema should have 'title' property")
	}

	props, ok = enrichResponseSchema.Properties["examples"]
	if !ok || props == nil {
		t.Error("enrichResponseSchema should have 'examples' property")
	}

	props, ok = enrichResponseSchema.Properties["req_id"]
	if !ok || props == nil {
		t.Error("enrichResponseSchema should have 'req_id' property")
	}
}
