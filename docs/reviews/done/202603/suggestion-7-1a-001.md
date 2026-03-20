# コードレビュー: suggestion-7-1a

## レビュー情報

| 項目                       | 内容                                     |
| -------------------------- | ---------------------------------------- |
| レビュー日                 | 2026-03-19                               |
| 対象ブランチ               | suggestion-7-1a                          |
| ベースブランチ             | suggestion-7-1                           |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md         |
| 変更ファイル数             | 9 ファイル                               |
| 変更行数（実装）           | +226 / -122 行（実質的な実装変更を含む） |
| 変更行数（テスト）         | +0 / -0 行                               |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - 開発環境ガイド（DBマイグレーション）
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260319072803_add_linked_page_ids_and_featured_image_to_suggestion_pages.sql`
- [x] `go/db/queries/suggestion_pages.sql`
- [x] `go/db/schema.sql`
- [x] `go/internal/model/suggestion_page.go`
- [x] `go/internal/query/models.go`
- [x] `go/internal/query/suggestion_pages.sql.go`
- [x] `go/internal/repository/suggestion_page.go`
- [x] `go/internal/testutil/suggestion_page_builder.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

### レビュー詳細（問題なし）

各ファイルについて確認した内容:

**マイグレーション**: `linked_page_ids VARCHAR[] NOT NULL DEFAULT '{}'` と `featured_image_attachment_id UUID` の型定義が `pages` テーブルの既存カラムと一致。`migrate:down` で逆順にカラムを削除しており正しい。

**SQLクエリ**: Create/Find/List/Update の全クエリに新カラムが追加されている。`space_id` によるスコープが維持されており、セキュリティガイドラインに準拠。

**モデル**: `LinkedPageIDs []PageID` と `FeaturedImageAttachmentID *AttachmentID` のフィールド定義が `model.Page` と同じパターン。ドメインID型を使用しておりアーキテクチャガイドに準拠。

**リポジトリ**: `model.PageIDsToStrings()` / `model.StringsToPageIDs()` によるスライス変換と、`*model.AttachmentID` のポインタ変換が `page.go` リポジトリの既存パターンと完全に一致。`Create` と `UpdateContent` の両方で同じ変換パターンが適用されている。

**sqlc生成コード**: `pq.Array` を使用したPostgreSQL配列の適切なハンドリング。自動生成ファイルであり手動編集なし。

**テストビルダー**: `SuggestionPageBuilder` と `SuggestionPageBuilderDB` の両方が更新されている。`linkedPageIDs` のデフォルト値が `[]string{}` に初期化されており、NOT NULL制約と整合。`WithLinkedPageIDs` と `WithFeaturedImageAttachmentID` メソッドが追加されている。

**作業計画書**: タスク 7-1a が完了チェック済み。旧 7-1b を分割して新 7-1b（draft_pagesへのカラム追加）と 7-1c（編集提案作成時の値保存）に再構成。設計の改善として、`featured_image_attachment_id` を `draft_pages` にも追加し、DraftPageからのコピーパターンを統一する方針は合理的。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 7-1a（suggestion_pagesテーブルへのlinked_page_idsとfeatured_image_attachment_idカラム追加）が作業計画書の仕様通りに実装されている。

- マイグレーション、モデル、リポジトリ、テストビルダーの全レイヤーが適切に更新されている
- `pages` テーブルの既存パターン（型定義、変換ロジック、ドメインID型）と完全に一致しており、コードベースの一貫性が保たれている
- セキュリティガイドラインに準拠し、全クエリで `space_id` スコープが維持されている
- 作業計画書のタスク分割（旧 7-1b → 新 7-1b + 7-1c）も合理的で、DraftPageからのコピーパターン統一という設計改善が含まれている
