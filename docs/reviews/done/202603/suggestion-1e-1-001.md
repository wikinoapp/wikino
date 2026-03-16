# コードレビュー: suggestion-1e-1

## レビュー情報

| 項目                       | 内容                                    |
| -------------------------- | --------------------------------------- |
| レビュー日                 | 2026-03-16                              |
| 対象ブランチ               | suggestion-1e-1                         |
| ベースブランチ             | suggestion-1d-2                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md        |
| 変更ファイル数             | 10 ファイル                             |
| 変更行数（実装）           | 約 +80 / -50 行（自動生成ファイル除く） |
| 変更行数（テスト）         | +170 / -0 行                            |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - 開発環境ガイド（マイグレーション）

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260316092918_add_suggestion_page_id_to_draft_pages.sql`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/db/schema.sql`（自動生成）
- [x] `go/internal/model/draft_page.go`
- [x] `go/internal/query/draft_pages.sql.go`（自動生成）
- [x] `go/internal/query/models.go`（自動生成）
- [x] `go/internal/repository/draft_page.go`

### テストファイル

- [x] `go/internal/repository/draft_page_test.go`

### 設定・その他

- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/testutil/draft_page_builder.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストヘルパー

**問題点・改善提案**:

- **一貫性**: `DraftPageBuilderDB` に `suggestionPageID` フィールドが追加されているが、`WithSuggestionPageID` メソッドが定義されていない。`DraftPageBuilder`（tx版）には同メソッドが存在するため、不整合がある

  ```go
  // DraftPageBuilder（tx版）: WithSuggestionPageID あり ✅
  // DraftPageBuilderDB（db版）: フィールドあり、メソッドなし ❌
  ```

  **修正案**:

  `DraftPageBuilderDB` にも `WithSuggestionPageID` メソッドを追加する:

  ```go
  // WithSuggestionPageID は編集提案ページIDを設定します
  func (b *DraftPageBuilderDB) WithSuggestionPageID(suggestionPageID model.SuggestionPageID) *DraftPageBuilderDB {
  	s := string(suggestionPageID)
  	b.suggestionPageID = &s
  	return b
  }
  ```

  **対応方針**:
  - [x] メソッドを追加する
  - [ ] 現時点では不要なので対応しない（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

タスク 1e-1（draft_pagesテーブルへのsuggestion_page_idカラム追加）の要件がすべて適切に実装されている。

- **マイグレーション**: nullable FK + 部分インデックス（WHERE IS NOT NULL）は適切。migrate:downも正しい
- **モデル**: `SuggestionPageID *SuggestionPageID` でポインタ型を使用し、nullable を正しく表現
- **リポジトリ**: Create/UpdateSuggestionPageID/toModelすべてでnullable変換を正しく処理。`UpdateDraftPage` は内容更新用であり `suggestion_page_id` を変更しないのは適切な設計判断
- **セキュリティ**: `UpdateDraftPageSuggestionPageID` クエリに `AND space_id = $4` が含まれており、スペースIDによるクエリスコープが守られている
- **テスト**: Create（suggestion_page_id付き）、UpdateSuggestionPageID（設定・クリア）の両方をカバー。テストデータのセットアップも適切
- **アーキテクチャ**: すべての変更がDomain/Infrastructure層に閉じており、依存関係のルールに違反していない

唯一の指摘は `DraftPageBuilderDB` の `WithSuggestionPageID` メソッドの欠如だが、現時点で使用箇所がないため軽微。
