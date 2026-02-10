package enrich

import (
	"errors"

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

// IsExampleTarget は Example 生成対象かどうかを返す（FR + NFR）。
func (c SegmentCategory) IsExampleTarget() bool {
	return c == CategoryFunctionalRequirement || c == CategoryNonFunctionalRequirement
}

// IsExampleTarget は Example 生成対象かどうかを返す（FR + NFR）。
func (r *EnrichResult) IsExampleTarget() bool {
	return r.Category.IsExampleTarget()
}

// BatchClassifyResult はバッチ分類の1セグメント分の結果。
type BatchClassifyResult struct {
	SegmentID string          `json:"segment_id"`
	Category  SegmentCategory `json:"category"`
	Title     string          `json:"title"`
	ReqID     string          `json:"req_id"`
}

// BatchExampleResult はバッチ Example 生成の1セグメント分の結果。
type BatchExampleResult struct {
	SegmentID string         `json:"segment_id"`
	Examples  []spec.Example `json:"examples"`
}

// ErrBatchTruncated はバッチレスポンスが切断された場合のエラー。
var ErrBatchTruncated = errors.New("batch response was truncated")
