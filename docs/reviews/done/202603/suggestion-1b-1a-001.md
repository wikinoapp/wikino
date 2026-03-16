# コードレビュー: suggestion-1b-1a

## レビュー情報

| 項目                       | 内容                                  |
| -------------------------- | ------------------------------------- |
| レビュー日                 | 2026-03-16                            |
| 対象ブランチ               | suggestion-1b-1a                      |
| ベースブランチ             | suggestion-1b-1                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md      |
| 変更ファイル数             | 10 ファイル                           |
| 変更行数（実装）           | +52 / -7 行（自動生成・スキーマ除く） |
| 変更行数（テスト）         | +0 / -0 行                            |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（ドメインID型、ModelとRepositoryの1:1関係）
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約（コメント）
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - 開発環境ガイド（マイグレーション、カラム定義）
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260316081143_add_number_to_suggestions.sql`
- [x] `go/db/queries/suggestions.sql`
- [x] `go/internal/model/id.go`
- [x] `go/internal/model/suggestion.go`
- [x] `go/internal/repository/suggestion.go`
- [x] `go/internal/testutil/suggestion_builder.go`

### テストファイル

（なし — 作業計画書でテスト 0 と記載）

### 設定・その他（自動生成含む）

- [x] `go/db/schema.sql`（自動生成）
- [x] `go/internal/query/models.go`（sqlc自動生成）
- [x] `go/internal/query/suggestions.sql.go`（sqlc自動生成）
- [x] `docs/plans/1_doing/suggestion.md`（タスクチェックボックス更新）

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

各ファイルの確認内容:

- **マイグレーション**: `INTEGER NOT NULL DEFAULT 0` + ユニークインデックス `[topic_id, number]` は作業計画書通り。down マイグレーションも適切
- **sqlcクエリ**: `GetNextSuggestionNumber` は既存の `GetNextPageNumber`（`go/db/queries/pages.sql`）と同じ `COALESCE(MAX(number), 0) + 1` パターンに準拠
- **ドメインID型**: `SuggestionNumber int32` と `String()` メソッドは既存の `PageNumber` パターンに完全準拠（[@go/docs/architecture-guide.md#ドメインID型](/workspace/go/docs/architecture-guide.md)）
- **モデル**: `Number SuggestionNumber` フィールドの配置位置が適切（ID系フィールドの直後）
- **リポジトリ**: `CreateSuggestionInput`、`Create`、`toModel`、`GetNextNumber` の更新が既存パターンに準拠
- **テストビルダー**: `nextNumber` の自動計算追加。ユニーク制約違反を防ぐ適切な実装
- **コメント**: 日本語で記述されており、コーディング規約に準拠（[@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md)）

## 設計との整合性チェック

作業計画書タスク **1b-1a** の要件をすべて確認:

| 要件                                                                | 実装状況 |
| ------------------------------------------------------------------- | -------- |
| `go/db/migrations/` に `number` (INTEGER NOT NULL) カラム追加       | ✅       |
| ユニークインデックス: `[topic_id, number]` を追加                   | ✅       |
| `internal/model/suggestion.go` に `Number` フィールドを追加         | ✅       |
| suggestionsのsqlcクエリを更新（numberを含むselect、Createでnumber） | ✅       |
| `internal/repository/suggestion.go` の `toModel` を更新             | ✅       |

すべての要件が実装されており、乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

suggestionsテーブルへの `number` カラム追加が、作業計画書の仕様通りに実装されています。既存の `PageNumber` / `GetNextPageNumber` パターンと完全に一貫した実装であり、ドメインID型の使用、コメントの日本語記述、マイグレーションのup/down両方の記述など、すべてのガイドラインに準拠しています。自動生成ファイル（sqlc、schema.sql）も正しく更新されています。
