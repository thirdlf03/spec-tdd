# spec-tdd

Spec-Driven TDD (仕様駆動テスト駆動開発) を支援する CLI ツール。YAML DSL による要件管理から、例示マッピング、テストスケルトン生成、トレーサビリティレポートまでを一貫して管理する。

## Features

- **要件管理** — `REQ-###` 形式の YAML DSL で要件を構造化
- **例示マッピング** — Given/When/Then 形式のシナリオを要件に紐付け
- **テストスケルトン生成** — 仕様から vitest/jest テストファイルを自動生成
- **トレーサビリティ** — 要件とテストの紐付けを JSON/Markdown レポートで可視化
- **kire 連携** — Markdown 仕様書から REQ/Example を自動抽出 (`import kire`)

## Installation

```bash
go install github.com/thirdlf03/spec-tdd@latest
```

### Build from Source

```bash
git clone https://github.com/thirdlf03/spec-tdd.git
cd spec-tdd
make build
```

## Quick Start

```bash
# ワークスペースを初期化
spec-tdd init

# 要件を追加
spec-tdd req add --title "Login lockout after 5 failures"

# 例示を追加
spec-tdd example add --req REQ-001 \
  --given "User has failed login 4 times" \
  --when "User fails login again" \
  --then "Account is locked for 30 minutes"

# テストスケルトンを生成
spec-tdd scaffold

# トレーサビリティレポートを生成
spec-tdd trace

# 例示マッピングレポートを生成
spec-tdd map
```

## kire Import

[kire](https://github.com/thirdlf03/kire) で分割した Markdown 仕様書から REQ/Example を自動インポートできる。

```bash
# kire 出力をインポート (デフォルト: .kire/ ディレクトリ)
spec-tdd import kire

# ディレクトリとメタデータファイルを指定
spec-tdd import kire --dir ./output --jsonl ./output/metadata.jsonl

# プレビュー (ファイル書き込みなし)
spec-tdd import kire --dry-run

# 既存ファイルを上書き
spec-tdd import kire --force
```

**対応する入力**:
- kire JSONL メタデータ (`segment_id`, `heading_path`, `file_path`)
- kire Markdown セグメントファイル (context comment `<!-- kire: ... -->` 対応)

**自動抽出**:
- `REQ-###` パターンから要件 ID
- `Given/When/Then` パターンから Example
- `?` 終端行・`Questions:` セクションから質問

## Configuration

### App Configuration

優先順位 (高い順):

1. CLI フラグ
2. 環境変数 (`APP_` prefix)
3. 設定ファイル (`config.yaml`)
4. デフォルト値

### Spec Configuration (`.tdd/config.yml`)

```yaml
specDir: .tdd/specs
testDir: tests
runner: vitest          # vitest or jest
fileNamePattern: "req-{{id}}-{{slug}}.test.ts"
```

## Development

```bash
# Devbox (推奨)
devbox shell
devbox run build
devbox run test

# Make
make build    # Build binary
make test     # Run tests (race + coverage)
make lint     # Run golangci-lint
make fmt      # Format code
make vet      # Run go vet
make clean    # Clean build artifacts
```

## Project Structure

```
├── cmd/                   # CLI commands (Cobra)
│   ├── root.go            # Root command, Viper/Logger init
│   ├── init.go            # spec-tdd init
│   ├── req.go             # spec-tdd req add
│   ├── example.go         # spec-tdd example add
│   ├── scaffold.go        # spec-tdd scaffold
│   ├── trace.go           # spec-tdd trace
│   ├── map.go             # spec-tdd map
│   └── import.go          # spec-tdd import kire
├── internal/
│   ├── apperrors/         # AppError type + sentinel errors
│   ├── config/            # App config + spec config
│   ├── kire/              # kire JSONL/MD parser + Spec converter
│   ├── logger/            # Structured logging (slog)
│   ├── scaffold/          # Test template rendering
│   ├── spec/              # YAML DSL model (Spec, Example, SourceInfo)
│   └── trace/             # Test scanning + report generation
├── main.go
├── Makefile
└── go.mod
```

## License

MIT LICENSE [LICENSE](LICENSE)
