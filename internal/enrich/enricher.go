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

// BatchEnricher はバッチ enrichment を抽象化する。
type BatchEnricher interface {
	// BatchClassify は全セグメントをバッチ分類する。
	BatchClassify(ctx context.Context, segments []*kire.Segment) ([]BatchClassifyResult, error)
	// BatchGenerateExamples は FR セグメントのバッチ Example 生成を行う。
	BatchGenerateExamples(ctx context.Context, segments []*kire.Segment) ([]BatchExampleResult, error)
}

// MockBatchEnricher はテスト用の BatchEnricher 実装。
type MockBatchEnricher struct {
	ClassifyResults  []BatchClassifyResult
	ClassifyErr      error
	ExampleResults   []BatchExampleResult
	ExampleErr       error
	ClassifyCallCount int
	ExampleCallCount  int
}

// BatchClassify は設定された分類結果を返す。
func (m *MockBatchEnricher) BatchClassify(_ context.Context, _ []*kire.Segment) ([]BatchClassifyResult, error) {
	m.ClassifyCallCount++
	if m.ClassifyErr != nil {
		return nil, m.ClassifyErr
	}
	return m.ClassifyResults, nil
}

// BatchGenerateExamples は設定された Example 結果を返す。
func (m *MockBatchEnricher) BatchGenerateExamples(_ context.Context, _ []*kire.Segment) ([]BatchExampleResult, error) {
	m.ExampleCallCount++
	if m.ExampleErr != nil {
		return nil, m.ExampleErr
	}
	return m.ExampleResults, nil
}
