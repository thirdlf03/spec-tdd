package enrich

import (
	"testing"
)

func TestSegmentCategory_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		category SegmentCategory
		want     bool
	}{
		{"functional_requirement is valid", CategoryFunctionalRequirement, true},
		{"non_functional_requirement is valid", CategoryNonFunctionalRequirement, true},
		{"overview is valid", CategoryOverview, true},
		{"other is valid", CategoryOther, true},
		{"unknown string is invalid", SegmentCategory("unknown"), false},
		{"empty string is invalid", SegmentCategory(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.category.IsValid()
			if got != tt.want {
				t.Errorf("SegmentCategory(%q).IsValid() = %v, want %v", tt.category, got, tt.want)
			}
		})
	}
}

func TestNormalizeCategory(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want SegmentCategory
	}{
		{"functional_requirement", "functional_requirement", CategoryFunctionalRequirement},
		{"non_functional_requirement", "non_functional_requirement", CategoryNonFunctionalRequirement},
		{"overview", "overview", CategoryOverview},
		{"other", "other", CategoryOther},
		{"unknown becomes other", "unknown_type", CategoryOther},
		{"empty becomes other", "", CategoryOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeCategory(tt.raw)
			if got != tt.want {
				t.Errorf("NormalizeCategory(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestEnrichResult_IsRequirement(t *testing.T) {
	tests := []struct {
		name     string
		category SegmentCategory
		want     bool
	}{
		{"functional_requirement is requirement", CategoryFunctionalRequirement, true},
		{"non_functional_requirement is not requirement", CategoryNonFunctionalRequirement, false},
		{"overview is not requirement", CategoryOverview, false},
		{"other is not requirement", CategoryOther, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EnrichResult{Category: tt.category}
			got := r.IsRequirement()
			if got != tt.want {
				t.Errorf("EnrichResult{Category: %q}.IsRequirement() = %v, want %v", tt.category, got, tt.want)
			}
		})
	}
}
