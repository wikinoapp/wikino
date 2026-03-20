# コードレビュー: suggestion-7-1c (7-1c タスク)

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-19                       |
| 対象ブランチ               | suggestion-7-1c                  |
| ベースブランチ             | suggestion-7-1b                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 10 ファイル                      |
| 変更行数（実装）           | +75 / -37 行                     |
| 変更行数（テスト）         | +175 / -3 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/apply_suggestion.go`
- [x] `go/internal/usecase/create_suggestion.go`

### テストファイル

- [x] `go/internal/usecase/apply_suggestion_test.go`
- [x] `go/internal/usecase/create_suggestion_test.go`

### 設定・その他

- [x] `go/db/migrations/20260319082723_add_featured_image_fk_to_suggestion_pages_and_draft_pages.sql`
- [x] `go/db/schema.sql`
- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-7-1c-001.md`
- [x] `docs/reviews/done/202603/suggestion-7-1c-002.md`
- [x] `docs/reviews/done/202603/suggestion-7-1c-003.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計との整合性チェック

作業計画書のタスク **7-1c** の要件:

1. **`create_suggestion.go` を更新: DraftPageからSuggestionPage作成時に `linked_page_ids` と `featured_image_attachment_id` の両方をDraftPageからコピー** - ✅ 実装済み（`create_suggestion.go:135-136`）
2. **`apply_suggestion.go` を更新: `page.LinkedPageIDs` の代わりに `sp.LinkedPageIDs` を使用し、`sp.FeaturedImageAttachmentID` を `UpdatePageInput` に渡す** - ✅ 実装済み（`apply_suggestion.go:133-134`、TODOコメントも削除済み）
3. **`apply_suggestion.go` を更新: 編集提案反映時に `syncAttachmentReferences` を呼び出して `page_attachment_references` テーブルを更新する（`attachmentRepo` と `pageAttachmentRefRepo` の依存を追加）** - ✅ 実装済み（`apply_suggestion.go:143-145`、依存追加も`apply_suggestion.go:22-23`）
4. **関連テストの更新** - ✅ 実装済み（両テストファイルにLinkedPageIDsとFeaturedImageAttachmentIDの検証テストを追加）

追加で実施されたマイグレーション（FK制約追加）:

- `suggestion_pages.featured_image_attachment_id` → `attachments(id)` のFK制約追加 - ✅ 適切
- `draft_pages.featured_image_attachment_id` → `attachments(id)` のFK制約追加 - ✅ 適切

作業計画書の要件はすべて実装されており、設計との乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク7-1cの要件がすべて正確に実装されている。主な変更点:

- `create_suggestion.go`: DraftPageからSuggestionPage作成時に`LinkedPageIDs`と`FeaturedImageAttachmentID`をコピーする処理を追加
- `apply_suggestion.go`: 編集提案反映時にSuggestionPageの値をPageに反映し、`syncAttachmentReferences`で添付ファイル参照の同期を実行。`attachmentRepo`と`pageAttachmentRefRepo`の依存を適切に追加
- マイグレーション: `suggestion_pages`と`draft_pages`の`featured_image_attachment_id`にFK制約を追加（データ整合性の強化）
- テスト: 両UseCaseにLinkedPageIDsとFeaturedImageAttachmentIDの検証テストを追加。テストデータの作成パターンも既存テストと一貫している

アーキテクチャガイドラインに従い、WithTxパターンの使用、エラーハンドリング、コメントの日本語記述など、すべてのコーディング規約に準拠している。
