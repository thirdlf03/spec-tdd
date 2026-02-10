package enrich

import (
	"fmt"
	"strings"

	"github.com/thirdlf03/spec-tdd/internal/kire"
	"google.golang.org/genai"
)

// batchClassifyPrompt は全セグメントのバッチ分類用プロンプト。
// %s にセグメント連結テキストが挿入される。
const batchClassifyPrompt = `あなたはソフトウェア仕様書の分析エキスパートです。
以下の複数セグメントをそれぞれ分析し、JSON配列で結果を返してください。

## タスク

各セグメントについて以下を判定してください。

### 1. セグメント種別分類

以下のいずれかに分類してください。

- "functional_requirement": 機能要件。以下のいずれかを含むセグメントはこれに該当する:
  - APIエンドポイント定義（POST, GET, PATCH, DELETE + URLパス）
  - CRUD操作の仕様（作成・取得・更新・削除）
  - データ操作のバリデーションルール
  - 入出力の具体的な振る舞い定義
  ※ セグメントに「背景」「データ定義」が含まれていても、APIエンドポイントや具体的な操作仕様があれば functional_requirement とする
- "non_functional_requirement": 非機能要件。パフォーマンス、セキュリティ、可用性など、機能の「振る舞い」ではなく「品質特性」のみを定義しているセグメント
- "overview": 概要・導入。プロジェクトの背景・目的・スコープ・用語集など、具体的な機能仕様を一切含まないセグメント
- "other": 上記以外

**重要な判定ルール**:
- APIエンドポイント（HTTPメソッド + パス）が1つでも定義されていれば → functional_requirement
- 「背景」+「データ定義」+「APIエンドポイント」が混在する場合 → functional_requirement
- 「共通仕様」でもエンドポイント定義を含む場合 → functional_requirement
- overviewは「機能仕様を一切含まない」セグメントにのみ使う

### 2. REQ-ID 抽出

セグメント内に「### REQ-XXX: タイトル」パターンがあれば、REQ-ID（例: "REQ-001"）を抽出してください。なければ空文字列を返してください。

### 3. タイトル抽出

セグメントの主題を表す簡潔なタイトルを抽出または生成してください。

## セグメント一覧

%s

## 出力形式

各セグメントの結果をJSON配列で返してください。segment_id を必ず含めてください。`

// batchExamplesPrompt は FR+NFR セグメントのバッチ Example 生成用プロンプト。
// %s にセグメント連結テキストが挿入される。
const batchExamplesPrompt = `あなたはソフトウェア仕様書の分析エキスパートです。
以下の複数セグメント（機能要件および非機能要件）について、Given/When/Then 形式の Examples を生成してください。

## GWT 生成ルール

**Givenの書き方（重要）**:
- テストの前提条件として「何が存在するか」「どういう状態か」を具体的に書く
- 悪い例: "クライアントは有効な認証情報を持っている" ← テストで何を setUp すべきか不明
- 良い例: "IDが'label-001'のラベルが存在する" ← setUp が明確
- 良い例: "タスクAが存在し、blockedByが['taskB']である" ← 状態が具体的
- 良い例: "IDが'deleted-001'のラベルが論理削除されている" ← エッジケースの状態

**Whenの書き方**:
- 機能要件: HTTPメソッド + パスを明記する
- 非機能要件: 測定条件・負荷条件・セキュリティ操作を具体的に書く
- 良い例(FR): "PATCH /v1/labels/label-001 にcolorを'#FF0000'に更新するリクエストを送信する"
- 良い例(NFR): "100件の同時リクエストを送信する"
- 良い例(NFR): "SQLインジェクション文字列を含むリクエストを送信する"

**機能要件でカバーすべきケース**:
- 各エンドポイントの正常系（CRUD成功）
- バリデーションエラー（必須項目欠落、フォーマット不正、範囲外の値）
- 存在しないリソースへの操作（404）
- 重複・競合（409）
- 論理削除済みリソースへの操作
- 仕様に記載されたデータ正規化ルール（空白除去、重複除去、ソート等）
- 状態遷移の制約（ロック、依存関係、親子関係）

**非機能要件でカバーすべきケース**:
- パフォーマンス: レスポンスタイム、スループット、同時接続数
- セキュリティ: 認証・認可、インジェクション対策、暗号化
- 可用性: 障害復旧、タイムアウト、リトライ
- データ整合性: バックアップ、整合性チェック

既に Given/When/Then 形式の記述がある場合はそれをそのまま使用してください。

## セグメント一覧

%s

## 出力形式

各セグメントの結果をJSON配列で返してください。segment_id を必ず含めてください。`

// batchClassifySchema はバッチ分類のレスポンススキーマ。
var batchClassifySchema = &genai.Schema{
	Type: genai.TypeArray,
	Items: &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"segment_id": {
				Type:        genai.TypeString,
				Description: "セグメントID",
			},
			"category": {
				Type:        genai.TypeString,
				Enum:        []string{"functional_requirement", "non_functional_requirement", "overview", "other"},
				Description: "セグメントの種別分類",
			},
			"title": {
				Type:        genai.TypeString,
				Description: "要件の正確なタイトル",
			},
			"req_id": {
				Type:        genai.TypeString,
				Description: "セグメント内の REQ-XXX パターン。存在しない場合は空文字列",
			},
		},
		Required: []string{"segment_id", "category", "title"},
	},
}

// batchExamplesSchema はバッチ Example 生成のレスポンススキーマ。
var batchExamplesSchema = &genai.Schema{
	Type: genai.TypeArray,
	Items: &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"segment_id": {
				Type:        genai.TypeString,
				Description: "セグメントID",
			},
			"examples": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"given": {Type: genai.TypeString, Description: "テストの前提条件。何が存在しどういう状態かを具体的に記述"},
						"when":  {Type: genai.TypeString, Description: "実行する操作。HTTPメソッド+パスを明記"},
						"then":  {Type: genai.TypeString, Description: "期待される結果。HTTPステータスコードとレスポンス内容を含む"},
					},
					Required: []string{"given", "when", "then"},
				},
				Description: "Given/When/Then 形式の Examples",
			},
		},
		Required: []string{"segment_id", "examples"},
	},
}

// formatSegmentsForClassify はセグメントをバッチ分類用のテキストに連結する。
func formatSegmentsForClassify(segments []*kire.Segment) string {
	var b strings.Builder
	for i, seg := range segments {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(fmt.Sprintf("--- segment_id: %s ---\n", seg.Meta.SegmentID))
		b.WriteString(seg.Content)
	}
	return b.String()
}

// formatSegmentsForExamples は FR+NFR セグメントをバッチ Example 生成用のテキストに連結する。
// タイトル情報を含む。
func formatSegmentsForExamples(segments []*kire.Segment, titles map[string]string) string {
	var b strings.Builder
	for i, seg := range segments {
		if i > 0 {
			b.WriteString("\n\n")
		}
		title := titles[seg.Meta.SegmentID]
		b.WriteString(fmt.Sprintf("--- segment_id: %s | title: %s ---\n", seg.Meta.SegmentID, title))
		b.WriteString(seg.Content)
	}
	return b.String()
}
