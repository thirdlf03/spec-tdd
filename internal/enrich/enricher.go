package enrich

import (
	"context"

	"github.com/thirdlf03/spec-tdd/internal/kire"
)

// Enricher はセグメントの LLM ベース enrichment を抽象化する。
type Enricher interface {
	// Enrich はセグメントを解析し、分類・メタデータ抽出・GWT 生成を行う。
	// enrichment 失敗時は err を返す（呼び出し元がフォールバックを判断）。
	Enrich(ctx context.Context, segment *kire.Segment) (*EnrichResult, error)
}

// MockEnricher はテスト用の Enricher 実装。
type MockEnricher struct {
	Result     *EnrichResult
	Err        error
	CalledWith []*kire.Segment
}

// Enrich は設定された結果を返す。
func (m *MockEnricher) Enrich(_ context.Context, segment *kire.Segment) (*EnrichResult, error) {
	m.CalledWith = append(m.CalledWith, segment)
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Result, nil
}
