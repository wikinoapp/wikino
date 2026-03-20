# コードレビュー: suggestion-7-1b

## レビュー情報

| 項目                       | 内容                                                        |
| -------------------------- | ----------------------------------------------------------- |
| レビュー日                 | 2026-03-19                                                  |
| 対象ブランチ               | suggestion-7-1b                                             |
| ベースブランチ             | suggestion-7-1a                                             |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md                            |
| 変更ファイル数             | 12 ファイル（実装 6, テスト 1, 自動生成 3, ドキュメント 2） |
| 変更行数（実装）           | 約 +180 / -100 行                                           |
| 変更行数（テスト）         | +193 行                                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - 開発環境ガイド（マイグレーション）
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260319074138_add_featured_image_attachment_id_to_draft_pages.sql`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/internal/model/draft_page.go`
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`

### テストファイル

- [x] `go/internal/repository/draft_page_test.go`

### 自動生成ファイル

- [x] `go/db/schema.sql`
- [x] `go/internal/query/draft_pages.sql.go`
- [x] `go/internal/query/models.go`

### ドキュメント

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-7-1b-001.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計との整合性チェック

作業計画書タスク **7-1b** の要件:

| 要件                                                                                                                            | 状態                                                                                     |
| ------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `go/db/migrations/` に `draft_pages` テーブルへの `featured_image_attachment_id` (UUID nullable) カラム追加マイグレーション作成 | ✅ 実装済み                                                                              |
| `internal/model/draft_page.go` に `FeaturedImageAttachmentID *AttachmentID` フィールドを追加                                    | ✅ 実装済み                                                                              |
| `internal/query/queries/draft_pages.sql` のクエリを更新（新カラムを含むSELECT/INSERT/UPDATE）                                   | ✅ 実装済み                                                                              |
| `internal/repository/draft_page.go` の `toModel` を更新                                                                         | ✅ 実装済み                                                                              |
| DraftPage保存時に `featured_image_attachment_id` を計算・保存するよう更新（`linked_page_ids` と同じタイミング）                 | ✅ 実装済み（`saveDraftPageContent` 内で `extractFeaturedImageAttachmentID` を呼び出し） |
| `make sqlc-generate` でコード生成                                                                                               | ✅ 実施済み                                                                              |
| テストビルダー（`internal/testutil/draft_page_builder.go`）を更新                                                               | ✅ 実装済み（tx版・db版の両方）                                                          |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 7-1b の要件をすべて満たしており、ガイドラインに準拠した実装になっている。

良かった点:

- `linked_page_ids` の既存パターンに忠実に従っており、コードベース全体の一貫性が保たれている
- `saveDraftPageContent` が `auto_save` と `manual_save` の両方で共有されているため、1箇所の変更で両方のパスが対応できている
- テストが Create/Update の両方で、値あり/nil の4パターンをカバーしており、十分な網羅性がある
- テストビルダーのtx版・db版の両方に `WithFeaturedImageAttachmentID` メソッドが追加されている
- `findOrCreateDraftPage` での初期作成時は body が空のため `FeaturedImageAttachmentID` を明示的に渡す必要がなく、Go のゼロ値（nil）で正しく動作する
