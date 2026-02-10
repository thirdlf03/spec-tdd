package deps

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

// DepsResult represents a dependency detection result for one spec.
type DepsResult struct {
	ID      string   `json:"id"`
	Depends []string `json:"depends"`
	Reason  string   `json:"reason"`
}

// Detector detects dependencies between specs.
type Detector interface {
	Detect(ctx context.Context, specs []*spec.Spec) ([]DepsResult, error)
}

// MockDetector is a test-only Detector implementation.
type MockDetector struct {
	Results   []DepsResult
	Err       error
	CallCount int
}

// Detect returns the pre-configured results.
func (m *MockDetector) Detect(_ context.Context, _ []*spec.Spec) ([]DepsResult, error) {
	m.CallCount++
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Results, nil
}

var reqRefPattern = regexp.MustCompile(`REQ-\d+`)

// japaneseDepKeywords are Japanese dependency indicator words.
var japaneseDepKeywords = []string{"参照", "前提", "依存"}

// HeuristicDetector detects dependencies using text pattern matching.
type HeuristicDetector struct{}

// Detect scans spec descriptions and examples for references to other REQ-IDs,
// matching title keywords, and Japanese dependency keywords.
func (h *HeuristicDetector) Detect(_ context.Context, specs []*spec.Spec) ([]DepsResult, error) {
	// Build lookup maps
	idSet := make(map[string]bool, len(specs))
	titleWords := make(map[string][]string) // word -> []spec-ID
	for _, s := range specs {
		idSet[s.ID] = true
		for _, w := range extractKeywords(s.Title) {
			titleWords[w] = append(titleWords[w], s.ID)
		}
	}

	var results []DepsResult

	for _, s := range specs {
		text := collectText(s)
		depSet := make(map[string]bool)
		var reasons []string

		// 1. REQ-ID pattern references
		for _, ref := range reqRefPattern.FindAllString(text, -1) {
			if ref != s.ID && idSet[ref] && !depSet[ref] {
				depSet[ref] = true
				reasons = append(reasons, "REQ-ID reference: "+ref)
			}
		}

		// 2. Title keyword matching (4+ chars)
		for _, w := range extractKeywords(s.Title) {
			// Skip the word if it belongs only to this spec
			for _, ownerID := range titleWords[w] {
				if ownerID != s.ID {
					// Check if the word appears in other spec's text
					for _, other := range specs {
						if other.ID == s.ID || depSet[other.ID] {
							continue
						}
						otherText := collectText(other)
						if strings.Contains(strings.ToLower(otherText), w) {
							depSet[other.ID] = true
							reasons = append(reasons, "keyword match: "+w+" -> "+other.ID)
						}
					}
					break
				}
			}
		}

		// 3. Japanese dependency keywords
		for _, kw := range japaneseDepKeywords {
			if strings.Contains(text, kw) {
				// Find which specs are referenced near the keyword
				for _, other := range specs {
					if other.ID == s.ID || depSet[other.ID] {
						continue
					}
					// Check if this spec's text mentions the other spec's ID near a dependency keyword
					if strings.Contains(text, other.ID) {
						depSet[other.ID] = true
						reasons = append(reasons, "dependency keyword: "+kw+" with "+other.ID)
					}
				}
			}
		}

		if len(depSet) > 0 {
			deps := make([]string, 0, len(depSet))
			for d := range depSet {
				deps = append(deps, d)
			}
			// Sort for deterministic output
			sortReqIDs(deps)
			results = append(results, DepsResult{
				ID:      s.ID,
				Depends: deps,
				Reason:  strings.Join(reasons, "; "),
			})
		}
	}

	return results, nil
}

// collectText concatenates all text fields of a spec.
func collectText(s *spec.Spec) string {
	var b strings.Builder
	b.WriteString(s.Title)
	b.WriteString(" ")
	b.WriteString(s.Description)
	for _, ex := range s.Examples {
		b.WriteString(" ")
		b.WriteString(ex.Given)
		b.WriteString(" ")
		b.WriteString(ex.When)
		b.WriteString(" ")
		b.WriteString(ex.Then)
	}
	return b.String()
}

// extractKeywords returns lowercase words from text that are 4+ characters (ASCII)
// or 2+ characters (multibyte/Japanese).
func extractKeywords(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	var out []string
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?()[]{}\"'")
		charCount := utf8.RuneCountInString(w)
		byteLen := len(w)
		if byteLen > charCount {
			// Multibyte: 2+ runes
			if charCount >= 2 {
				out = append(out, w)
			}
		} else {
			// ASCII: 4+ chars
			if charCount >= 4 {
				out = append(out, w)
			}
		}
	}
	return out
}

// sortReqIDs sorts REQ-IDs numerically.
func sortReqIDs(ids []string) {
	reqIDPat := regexp.MustCompile(`^REQ-(\d+)$`)
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0; j-- {
			am := reqIDPat.FindStringSubmatch(ids[j-1])
			bm := reqIDPat.FindStringSubmatch(ids[j])
			if len(am) == 2 && len(bm) == 2 {
				var ai, bi int
				for _, c := range am[1] {
					ai = ai*10 + int(c-'0')
				}
				for _, c := range bm[1] {
					bi = bi*10 + int(c-'0')
				}
				if ai > bi {
					ids[j-1], ids[j] = ids[j], ids[j-1]
				}
			} else if ids[j-1] > ids[j] {
				ids[j-1], ids[j] = ids[j], ids[j-1]
			}
		}
	}
}
