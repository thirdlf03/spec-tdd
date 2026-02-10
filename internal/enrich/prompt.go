package enrich

import (
	"google.golang.org/genai"
)

// classifyAndEnrichPrompt はセグメント分類 + GWT 生成用のプロンプトテンプレート。
// %s にセグメント content が挿入される。
const classifyAndEnrichPrompt = `あなたはソフトウェア仕様書の分析エキスパートです。
以下のセグメントを分析し、JSON形式で結果を返してください。

## タスク

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

### 4. Given/When/Then Examples 生成

セグメントが functional_requirement または non_functional_requirement の場合、仕様から Given/When/Then 形式の Examples を生成してください。
overview/other の場合は空配列を返してください。
既に Given/When/Then 形式の記述がある場合はそれをそのまま使用してください。

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
					"given": {Type: genai.TypeString, Description: "テストの前提条件。何が存在しどういう状態かを具体的に記述"},
					"when":  {Type: genai.TypeString, Description: "実行する操作。HTTPメソッド+パスを明記"},
					"then":  {Type: genai.TypeString, Description: "期待される結果。HTTPステータスコードとレスポンス内容を含む"},
				},
				Required: []string{"given", "when", "then"},
			},
			Description: "Given/When/Then 形式の Examples",
		},
	},
	Required: []string{"category", "title"},
}
