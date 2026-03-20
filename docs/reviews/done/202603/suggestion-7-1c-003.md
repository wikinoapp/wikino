# コードレビュー: suggestion-7-1c

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-19                             |
| 対象ブランチ               | suggestion-7-1c                        |
| ベースブランチ             | suggestion-7-1b                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 7 ファイル（実装 4, テスト 2, 設定 1） |
| 変更行数（実装）           | +69 / -31 行                           |
| 変更行数（テスト）         | +172 / -1 行                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/apply_suggestion.go`
- [x] `go/internal/usecase/create_suggestion.go`

### テストファイル

- [ ] `go/internal/usecase/create_suggestion_test.go`
- [x] `go/internal/usecase/apply_suggestion_test.go`

### 設定・その他

- [x] `go/db/migrations/20260319082723_add_featured_image_fk_to_suggestion_pages_and_draft_pages.sql`
- [x] `go/db/schema.sql`
- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/create_suggestion_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド（並行テスト）

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#並行テスト]**: 新しいテストケース「正常系: DraftPageのLinkedPageIDsとFeaturedImageAttachmentIDがSuggestionPageにコピーされる」に `t.Parallel()` が欠けている

  既存のテストケースでは同じファイル内で `t.Parallel()` を使用していないケースもあるが、 `apply_suggestion_test.go` の対応するテストケースでは `t.Parallel()` が付いている。同一PRの対応テスト間で一貫性がない。

  ```go
  // 現在のコード（create_suggestion_test.go:140）
  t.Run("正常系: DraftPageのLinkedPageIDsとFeaturedImageAttachmentIDがSuggestionPageにコピーされる", func(t *testing.T) {
      spaceID := testutil.NewSpaceBuilderDB(t, db).
  ```

  **修正案**:

  ```go
  t.Run("正常系: DraftPageのLinkedPageIDsとFeaturedImageAttachmentIDがSuggestionPageにコピーされる", func(t *testing.T) {
      t.Parallel()
      spaceID := testutil.NewSpaceBuilderDB(t, db).
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] `t.Parallel()` を追加する
  - [ ] 追加しない（既存テストケースに合わせて省略する方針とする）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

タスク 7-1c の実装内容（編集提案作成時に `linked_page_ids` と `featured_image_attachment_id` を保存し、反映時に使用する）は作業計画書の要件通りに正しく実装されている。

実装面では:

- `create_suggestion.go`: DraftPage から SuggestionPage 作成時に `LinkedPageIDs` と `FeaturedImageAttachmentID` を正しくコピーしている
- `apply_suggestion.go`: `page.LinkedPageIDs` / `page.FeaturedImageAttachmentID` の代わりに `sp.LinkedPageIDs` / `sp.FeaturedImageAttachmentID` を使用するよう修正され、さらに `syncAttachmentReferences` の呼び出しが追加されている
- マイグレーション: FK 制約の追加のみで、シンプルかつ正確
- アーキテクチャ: UseCase が Repository に依存し、3 層アーキテクチャの依存関係ルールに従っている。WithTx パターンも正しく使用されている
- セキュリティ: `space_id` を条件に含めたクエリスコープが維持されている

テストでは `LinkedPageIDs` と `FeaturedImageAttachmentID` が正しくコピー・反映されることを検証する新しいテストケースが追加されており、テストカバレッジは十分。唯一の指摘は `create_suggestion_test.go` の新テストケースで `t.Parallel()` が欠けている点のみ。
