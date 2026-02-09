package enrich

import (
	"google.golang.org/genai"
)

// classifyAndEnrichPrompt はセグメント分類 + GWT 生成用のプロンプトテンプレート。
// %s にセグメント content が挿入される。
const classifyAndEnrichPrompt = `あなたはソフトウェア仕様書の分析エキスパートです。
以下のセグメントを分析し、JSON形式で結果を返してください。

## タスク

1. **セグメント種別分類**: 以下のいずれかに分類してください。
   - "functional_requirement": 機能要件（具体的な機能の振る舞いを定義している）
   - "non_functional_requirement": 非機能要件（パフォーマンス、セキュリティ等）
   - "overview": 概要・導入（プロジェクト概要、背景説明等）
   - "other": 上記以外

2. **REQ-ID 抽出**: セグメント内に「### REQ-XXX: タイトル」パターンがあれば、REQ-ID（例: "REQ-001"）を抽出してください。なければ空文字列を返してください。

3. **タイトル抽出**: セグメントの主題を表す簡潔なタイトルを抽出または生成してください。

4. **Given/When/Then Examples 生成**: セグメントが機能要件の場合、正常系・異常系の記述から Given/When/Then 形式の Examples を生成してください。
   - 各 Example は given（前提条件）、when（操作）、then（期待結果）の3フィールドを持つ
   - 機能要件でない場合は空配列を返してください
   - 既に Given/When/Then 形式の記述がある場合はそれをそのまま使用してください

## セグメント内容

%s

## 出力形式

JSONオブジェクトで返してください。`

// enrichResponseSchema は Gemini API の構造化出力用スキーマ。
var enrichResponseSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"category": {
			Type:        genai.TypeString,
			Enum:        []string{"functional_requirement", "non_functional_requirement", "overview", "other"},
			Description: "セグメントの種別分類",
		},
		"req_id": {
			Type:        genai.TypeString,
			Description: "セグメント内の REQ-XXX パターン。存在しない場合は空文字列",
		},
		"title": {
			Type:        genai.TypeString,
			Description: "要件の正確なタイトル",
		},
		"examples": {
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"given": {Type: genai.TypeString, Description: "前提条件"},
					"when":  {Type: genai.TypeString, Description: "操作・アクション"},
					"then":  {Type: genai.TypeString, Description: "期待される結果"},
				},
				Required: []string{"given", "when", "then"},
			},
			Description: "Given/When/Then 形式の Examples",
		},
	},
	Required: []string{"category", "title"},
}
