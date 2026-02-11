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
  - データ正規化ルール（Unicode正規化、全角/半角変換、空白処理、文字除去等）
  - データ変換・フォーマット変換の仕様
  ※ セグメントに「背景」「データ定義」が含まれていても、APIエンドポイントや具体的な操作仕様があれば functional_requirement とする
  ※ 具体的な入力→出力の変換ルールが定義されていれば functional_requirement とする
- "non_functional_requirement": 非機能要件。パフォーマンス、セキュリティ、可用性など、機能の「振る舞い」ではなく「品質特性」のみを定義しているセグメント
- "overview": 概要・導入。プロジェクトの背景・目的・スコープ・用語集など、具体的な機能仕様を一切含まないセグメント
- "other": 上記のいずれにも該当しないセグメント（付録、変更履歴、メモなど）

**重要な判定ルール**:
- APIエンドポイント（HTTPメソッド + パス）が1つでも定義されていれば → functional_requirement
- 「背景」+「データ定義」+「APIエンドポイント」が混在する場合 → functional_requirement
- 「共通仕様」でもエンドポイント定義を含む場合 → functional_requirement
- データ正規化・変換ルール（Unicode、全角半角、空白トリム等）がある → functional_requirement
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
// 第1 %s: 共通仕様コンテキスト（空文字列の場合もある）
// 第2 %s: セグメント連結テキスト
const batchExamplesPrompt = `あなたはソフトウェア仕様書の分析エキスパートです。
以下の複数セグメント（機能要件および非機能要件）について、Given/When/Then 形式の Examples を**網羅的に**生成してください。
仕様に記載されたルールに対してテストケースが1つも生成されない「漏れ」は絶対に避けてください。
%s
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

## 網羅性チェックリスト（必須）

各セグメントで以下のカテゴリを**すべて確認**し、該当する仕様があれば必ず Example を生成してください。

### A. エンドポイント別（POST / GET単体 / GETリスト / PATCH / DELETE がある場合）

**POST（作成）**:
- 最小フィールド（必須のみ）での正常作成
- 全フィールド指定での正常作成
- ID指定 vs ID自動生成
- ID重複（既存 + 論理削除済み）→ 409
- 一意性制約違反（title/name/key等）→ 409
- 各フィールドのバリデーションエラー（必須欠落、フォーマット不正、範囲外）→ 422
- unknown field → 400
- 不正JSON / 非オブジェクトbody → 400
- Content-Type不正 → 415

**GET（単体取得）**:
- 存在するリソース → 200 + ETag
- 存在しないID → 404
- 論理削除済み（includeDeleted なし）→ 404
- 論理削除済み（includeDeleted=true）→ 200

**GET（リスト取得）** ← 見落としやすいので特に注意:
- フィルタなしのデフォルト取得
- 各クエリパラメータ（q, status, tags, includeDeleted, sort, order, limit, offset 等）の正常動作
- 仕様に列挙された sort の**各値**（例: createdAt, updatedAt, priority 等）をそれぞれテスト
- boolean/enum フィルタは**全有効値**をテスト（例: includeDeleted の true/false/only、status の各値）
- 不正なクエリパラメータ値 → 422
- ページネーション（limit, offset）の境界値検証（limit の最小値・最大値）
- ソート順序の検証（asc/desc, null値の扱い）

**PATCH（更新）**:
- 単一フィールド更新の正常系
- null送信によるフィールドクリア（description, dueDate, tags 等）
- 空オブジェクト {} → 400
- 仕様で不変と指定された**各フィールドを個別に**テスト → 400（例: id, createdAt, version 等それぞれ別の Example）
- 存在しないリソース → 404
- 論理削除済みリソース → 409
- 一意性制約違反（title/name変更時）→ 409
- If-Match 欠落 → 428（仕様にある場合）
- If-Match 不一致 → 412（仕様にある場合）
- **PATCHでもPOSTと同じバリデーションが適用される**: 更新可能な各フィールドに対して、POSTで定義されたバリデーション（文字数上限超過、フォーマット不正、範囲外等）をPATCH時にもテスト
- PATCH時の正規化: 更新するフィールドに正規化ルールがあれば、PATCH時にも正規化が適用されることをテスト

**DELETE（削除）**:
- 正常削除（論理削除 or 物理削除）
- 既に削除済み → 404
- 存在しないID → 404
- If-Match 欠落 → 428（仕様にある場合）
- If-Match 不一致 → 412（仕様にある場合）
- 依存関係がある場合の削除制約（子リソース、被依存等）

### B. 共通仕様の適用（共通仕様セクションがある場合）

共通仕様に定義されたルールは**各リソースのエンドポイントに対して**テストを生成すること:
- ETag/If-Match: 仕様で要求されている全リソースの PATCH/DELETE で 428 と 412 を生成
- 正規化ルール: 各フィールドの正規化（trim, collapse, lowercase, dedup, sort 等）
- エラー形式: error.code の値が仕様通りか

### C. 境界値テスト

仕様に数値制限・文字数制限がある場合は**必ず**境界値の Example を生成:
- 文字数上限（例: title 80文字、description 2000文字）→ 上限ちょうど（成功）と上限+1（失敗）
- 配列要素数上限（例: tags 5個）→ 上限ちょうど（成功）と上限+1（失敗）
- 数値範囲（例: priority 1..5）→ 下限-1、上限+1
- 文字数下限（例: name 3文字以上）→ 下限ちょうど（成功）と下限-1（失敗）

### D. データ正規化テスト

仕様に正規化ルールがある場合は**個別に** Example を生成:
- Unicode正規化（全角→半角、ゼロ幅文字除去、NFC等）
- 空白正規化（trim、連続空白collapse）
- null変換（空文字列→null）
- 配列正規化（dedup、sort、空要素除去）

### E. 状態遷移・ビジネスルール

- 状態遷移制約（例: done→openの可否、子タスク未完了時の親done制約）
- 依存関係制約（循環依存、未完了依存先）
- カスケード動作（削除時の子リソースへの影響）

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

// formatContextSection は overview セグメントを共通仕様コンテキストとしてフォーマットする。
// contextSegments が空の場合は空文字列を返す。
func formatContextSection(contextSegments []*kire.Segment) string {
	if len(contextSegments) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n## 共通仕様（各セグメントへの適用必須）\n\n")
	b.WriteString("以下は全リソースに横断的に適用される共通仕様です。\n")
	b.WriteString("**重要**: 各セグメントの Example 生成時に、以下のルールを**必ず適用**してください:\n")
	b.WriteString("- エラーフォーマット（error.code, error.message）が仕様と一致するか\n")
	b.WriteString("- ETag/If-Match が必要なエンドポイントでは 428/412 のテストも生成\n")
	b.WriteString("- 正規化ルール（title trim、tags dedup+sort、description 空→null 等）のテストも生成\n")
	b.WriteString("- 不正JSON(400)、unknown field(400)、Content-Type不正(415)のテストも各リソースで生成\n")
	b.WriteString("\nただし、以下の共通仕様セグメント自体に対する Example は生成しないでください。\n\n")
	for i, seg := range contextSegments {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(fmt.Sprintf("--- context: %s ---\n", seg.Meta.SegmentID))
		b.WriteString(seg.Content)
	}
	b.WriteString("\n")
	return b.String()
}

// formatSegmentsForClassify はセグメントをバッチ分類用のテキストに連結する。
func formatSegmentsForClassify(segments []*kire.Segment) string {
	var b strings.Builder
	for i, seg := range segments {
		if i > 0 {
			b.WriteString("\n\n")
		}
		header := fmt.Sprintf("--- segment_id: %s", seg.Meta.SegmentID)
		if seg.Context != "" {
			header += fmt.Sprintf(" | context: %s", seg.Context)
		}
		header += " ---\n"
		b.WriteString(header)
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
		header := fmt.Sprintf("--- segment_id: %s | title: %s", seg.Meta.SegmentID, title)
		if seg.Context != "" {
			header += fmt.Sprintf(" | context: %s", seg.Context)
		}
		header += " ---\n"
		b.WriteString(header)
		b.WriteString(seg.Content)
	}
	return b.String()
}
