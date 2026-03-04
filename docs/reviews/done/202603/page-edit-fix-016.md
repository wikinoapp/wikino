# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                                                                                                                               |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| レビュー日                 | 2026-03-04                                                                                                                                         |
| 対象ブランチ               | page-edit-fix                                                                                                                                      |
| ベースブランチ             | page-edit                                                                                                                                          |
| 作業計画書（指定があれば） | [page-edit-go-migration.md](../plans/1_doing/page-edit-go-migration.md), [page-edit-rails-go-diff.md](../plans/1_doing/page-edit-rails-go-diff.md) |
| 変更ファイル数             | 44 ファイル                                                                                                                                        |
| 変更行数（実装）           | +757 / -92 行                                                                                                                                      |
| 変更行数（テスト）         | +1055 / -17 行                                                                                                                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版の開発ガイド
- [@rails/docs/security-guide.md](/workspace/rails/docs/security-guide.md) - Railsセキュリティガイドライン
- [@rails/docs/testing-guide.md](/workspace/rails/docs/testing-guide.md) - Railsテストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/pages.sql`
- [x] `go/internal/handler/draft_page/show.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page_backlink_list/show.go`
- [x] `go/internal/handler/page_backlinks/handler.go`
- [x] `go/internal/handler/page_backlinks/show.go`
- [x] `go/internal/handler/page_link_list/handler.go`
- [x] `go/internal/handler/page_link_list/show.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/query/pages.sql.go`（自動生成）
- [x] `go/internal/repository/page.go`
- [x] `go/internal/repository/page_editor.go`
- [x] `go/internal/templates/components/backlink_list.templ`
- [x] `go/internal/templates/components/backlink_list_templ.go`（自動生成）
- [x] `go/internal/templates/components/link_list.templ`
- [x] `go/internal/templates/components/link_list_templ.go`（自動生成）
- [x] `go/internal/templates/components/page_backlink_list.templ`
- [x] `go/internal/templates/components/page_backlink_list_templ.go`（自動生成）
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/publish_page.go`
- [x] `go/internal/viewmodel/link_list.go`
- [x] `go/web/style.css`
- [x] `go/internal/testutil/page_builder.go`
- [x] `rails/app/controllers/attachments/presigns/create_controller.rb`
- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/1_doing/page-edit-rails-go-diff.md`
- [x] `docs/plans/3_done/202603/e2e-ci-fix.md`
- [x] `docs/reviews/done/202603/page-edit-fix-003.md`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/page_backlink_list/show_test.go`
- [x] `go/internal/handler/page_backlinks/main_test.go`
- [x] `go/internal/handler/page_backlinks/show_test.go`
- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/templates/components/link_list_test.go`
- [x] `go/internal/templates/components/page_backlink_list_test.go`
- [x] `go/internal/usecase/auto_save_draft_page_test.go`
- [x] `go/internal/usecase/publish_page_test.go`
- [x] `go/internal/middleware/reverse_proxy_test.go`
- [x] `rails/spec/requests/attachments/presigns/create_spec.rb`

## ファイルごとのレビュー結果

### `go/internal/handler/page_link_list/handler.go` + `go/internal/handler/page_link_list/show.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@CLAUDE.md#pull-requestのガイドライン](/workspace/CLAUDE.md) - PRガイドライン（実装とテストのセット化）

**問題点・改善提案**:

- **[@CLAUDE.md#実装とテストのセット化]**: `page_link_list` ハンドラーに対応するテストファイルが存在しない

  `page_link_list/show.go` は新規作成されたハンドラーファイル（210行）だが、テストファイル（`show_test.go` または `handler_test.go`）が含まれていない。PRガイドラインでは「実装コードとそのテストコードは同じPRに含める」ことが必須とされている。

  同等の `page_backlinks/show.go` には `show_test.go`（384行）が存在しており、一貫性の観点からもテストが必要。

  **修正案**:

  `page_link_list/show_test.go` を作成し、以下のケースをテストする:
  - 未認証ユーザーのアクセス
  - 存在しないスペース
  - 非スペースメンバー
  - 無効なページ番号
  - リンクなしのケース
  - リンクありの正常系
  - ページネーション

  **対応方針**:
  - [x] テストファイルを作成する（推奨）
  - [ ] 別PRでテストを追加する（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

### 作業計画書 `page-edit-rails-go-diff.md` との整合性

**差分1（page_editorsが作成されない）**: ✅ 実装済み

- `auto_save_draft_page.go`: `resolveAndCreateLinkedPages` に `pageEditorRepo` パラメータが追加され、ページ新規作成時に `pageEditorRepo.FindOrCreate` が呼び出されている
- `publish_page.go`: 同様に `resolveAndCreateLinkedPages` で `pageEditorRepo` が使用されている
- `auto_save_draft_page_test.go`: 自動作成ページに `page_editors` が作成されることを検証するテストが追加されている
- `publish_page_test.go`: 同様のテストが追加されている

**差分2（discarded_atフィルタ削除）**: ✅ 実装済み

- `go/db/queries/pages.sql`: `FindPageByTopicAndTitle` クエリから `AND discarded_at IS NULL` が削除されている
- `auto_save_draft_page_test.go`: 廃棄済みページと同名のWikiリンクで自動保存が成功するテストが追加
- `publish_page_test.go`: 同様のテストが追加

**差分3（ページ番号採番戦略）**: ✅ 現状維持（計画通り）

### 作業計画書 `page-edit-go-migration.md` との整合性

本PRは差分修正に焦点を当てたPRであり、作業計画書の差分修正タスク（フェーズ1）は全て完了している。リンク一覧・バックリンク一覧のSSEハンドラー追加とRails presignコントローラーの修正は、ページ編集画面の関連機能として適切に実装されている。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

作業計画書の差分修正（差分1: page_editors作成、差分2: discarded_atフィルタ削除）は仕様通りに正しく実装されている。セキュリティ面（認証・認可・space_idスコーピング・CSRF対策）、アーキテクチャ面（3層分離・Repository経由のデータアクセス）、コーディング規約（slog使用・日本語コメント・i18n対応）はすべて問題ない。テストカバレッジも良好。

唯一の指摘は、新規ハンドラー `page_link_list` にテストファイルが含まれていない点。同等の `page_backlinks` にはテストが存在しており、一貫性とPRガイドラインの観点からテストの追加を推奨する。
