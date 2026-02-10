package deps

import (
	"fmt"
	"strings"

	"github.com/thirdlf03/spec-tdd/internal/spec"
	"google.golang.org/genai"
)

// depsDetectPrompt is the prompt template for LLM-based dependency detection.
// %s is replaced with formatted spec summaries.
const depsDetectPrompt = `あなたはソフトウェア仕様書の依存関係分析エキスパートです。
以下の要件仕様一覧を分析し、各要件間の依存関係を検出してください。

## 依存関係の種類

以下の4種類の依存関係を検出してください:

1. **機能依存**: ある機能が別の機能の存在を前提としている（例: 「注文」は「カート」に依存）
2. **データ依存**: ある機能が別の機能で作成・管理されるデータを使用する（例: 「レポート生成」は「データ登録」に依存）
3. **仕様依存**: ある仕様が別の仕様で定義されたルールや制約を前提としている（例: 「権限チェック」は「ロール定義」に依存）
4. **参照依存**: ある仕様が別の仕様を明示的に参照している（例: 「REQ-003 を参照」）

## 判定ルール

- 依存関係は「AはBに依存する」= 「Bが先に実装されるべき」を意味する
- 自己参照は含めないこと
- 存在しない REQ-ID を参照しないこと
- 曖昧な関係は含めず、明確な依存のみを報告すること
- reason には依存の種類と具体的な理由を簡潔に記述すること

## 要件仕様一覧

%s

## 出力形式

各要件の依存関係をJSON配列で返してください。依存がない要件は配列から省略してください。`

// depsResponseSchema is the genai.Schema for dependency detection response.
var depsResponseSchema = &genai.Schema{
	Type: genai.TypeArray,
	Items: &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"id": {
				Type:        genai.TypeString,
				Description: "要件ID (例: REQ-001)",
			},
			"depends": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type:        genai.TypeString,
					Description: "依存先の要件ID",
				},
				Description: "依存先の要件IDリスト",
			},
			"reason": {
				Type:        genai.TypeString,
				Description: "依存関係の理由",
			},
		},
		Required: []string{"id", "depends", "reason"},
	},
}

// formatSpecsForDeps formats specs for the dependency detection prompt.
func formatSpecsForDeps(specs []*spec.Spec) string {
	var b strings.Builder
	for i, s := range specs {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(fmt.Sprintf("--- %s: %s ---\n", s.ID, s.Title))
		if s.Description != "" {
			b.WriteString(s.Description)
			b.WriteString("\n")
		}
		for _, ex := range s.Examples {
			b.WriteString(fmt.Sprintf("- Given: %s / When: %s / Then: %s\n", ex.Given, ex.When, ex.Then))
		}
	}
	return b.String()
}
