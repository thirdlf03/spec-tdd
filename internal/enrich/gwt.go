package enrich

import (
	"fmt"
	"strings"

	"github.com/thirdlf03/spec-tdd/internal/spec"
)

// MergeExamples は既存の GWT Examples と enrichment で生成された Examples をマージする。
// 既存の GWT が 1 つ以上ある場合はそちらを優先し、なければ enriched を使用する。
// 最終的な Examples に E1, E2, ... の連番 ID を付与する。
func MergeExamples(existing, enriched []spec.Example) []spec.Example {
	var source []spec.Example
	if len(existing) > 0 {
		source = existing
	} else {
		source = enriched
	}

	if len(source) == 0 {
		return nil
	}

	result := make([]spec.Example, len(source))
	for i, ex := range source {
		result[i] = spec.Example{
			ID:    fmt.Sprintf("E%d", i+1),
			Given: ex.Given,
			When:  ex.When,
			Then:  ex.Then,
		}
	}
	return result
}

// normalizeGWT は Given/When/Then を正規化キーに変換する。
// trim + lowercase で空白差異・大文字小文字の違いを吸収する。
func normalizeGWT(given, when, then string) string {
	return strings.ToLower(strings.TrimSpace(given)) + "|" +
		strings.ToLower(strings.TrimSpace(when)) + "|" +
		strings.ToLower(strings.TrimSpace(then))
}

// DeduplicateExamples は Given+When+Then の正規化キーで重複を検出し、
// 最初に出現したものを残して後続の重複を除去する。
// ID は E1, E2, ... に再採番される。
func DeduplicateExamples(examples []spec.Example) []spec.Example {
	if len(examples) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(examples))
	var unique []spec.Example

	for _, ex := range examples {
		key := normalizeGWT(ex.Given, ex.When, ex.Then)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, ex)
	}

	// 再採番
	for i := range unique {
		unique[i].ID = fmt.Sprintf("E%d", i+1)
	}
	return unique
}
