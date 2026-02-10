package enrich

import (
	"github.com/thirdlf03/spec-tdd/internal/spec"
)

// SegmentCategory はセグメントの種別。
type SegmentCategory string

const (
	CategoryFunctionalRequirement    SegmentCategory = "functional_requirement"
	CategoryNonFunctionalRequirement SegmentCategory = "non_functional_requirement"
	CategoryOverview                 SegmentCategory = "overview"
	CategoryOther                    SegmentCategory = "other"
)

var validCategories = map[SegmentCategory]bool{
	CategoryFunctionalRequirement:    true,
	CategoryNonFunctionalRequirement: true,
	CategoryOverview:                 true,
	CategoryOther:                    true,
}

// IsValid は有効なカテゴリかどうかを返す。
func (c SegmentCategory) IsValid() bool {
	return validCategories[c]
}

// NormalizeCategory は文字列を SegmentCategory に変換する。
// 未知の文字列は CategoryOther として扱う。
func NormalizeCategory(raw string) SegmentCategory {
	c := SegmentCategory(raw)
	if c.IsValid() {
		return c
	}
	return CategoryOther
}

// EnrichResult は 1 セグメントの enrichment 結果。
type EnrichResult struct {
	Category SegmentCategory // セグメント種別
	ReqID    string          // 抽出された REQ-ID（空の場合は自動採番）
	Title    string          // 抽出または生成されたタイトル
	Examples []spec.Example  // 生成された GWT Examples
}

// IsRequirement は機能要件かどうかを返す。
func (r *EnrichResult) IsRequirement() bool {
	return r.Category == CategoryFunctionalRequirement
}
