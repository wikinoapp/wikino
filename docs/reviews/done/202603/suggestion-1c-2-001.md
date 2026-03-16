# コードレビュー: suggestion-1c-2

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-16                       |
| 対象ブランチ               | suggestion-1c-2                  |
| ベースブランチ             | suggestion-1c-1                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 7 ファイル                       |
| 変更行数（実装）           | +278 / -0 行                     |
| 変更行数（テスト）         | +489 / -0 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/suggestion_page_revisions.sql`
- [x] `go/internal/query/models.go`（自動生成）
- [x] `go/internal/query/suggestion_page_revisions.sql.go`（自動生成）
- [x] `go/internal/repository/suggestion_page_revision.go`
- [x] `go/internal/testutil/suggestion_page_revision_builder.go`

### テストファイル

- [x] `go/internal/repository/suggestion_page_revision_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`（タスクチェックボックスの更新のみ）

## ファイルごとのレビュー結果

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク1c-2（suggestion_page_revisionsテーブルのsqlcクエリとリポジトリ）が作業計画書通りに実装されている。

**良い点**:

- 既存の `SuggestionPageRepository` と完全に一貫したパターンで実装されている（構造体定義、WithTx、Create、toModel/toModels、Input構造体）
- すべてのSQLクエリに `space_id` 条件が含まれており、セキュリティガイドラインの「スペースIDによるクエリスコープ」に準拠している
- ドメインID型を正しく使用している（`model.SuggestionPageRevisionID`, `model.SuggestionPageID`, `model.SpaceMemberID`）
- テストが充実しており、正常系（タイトルあり/なし）、空結果、異なるスペースIDでのスコープ検証をカバーしている
- テストビルダーが既存のビルダーパターン（`SuggestionPageBuilder`）と一貫しており、必須フィールドのバリデーションも実装されている
- `FindLatest` で `sql.ErrNoRows` の場合に `nil, nil` を返すパターンが既存の `FindByID` と一致している
- nullable な `Title` フィールドの `sql.NullString` ↔ `*string` 変換が正確
