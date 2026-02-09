package enrich

import (
	"fmt"

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
