# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                                                                        |
| -------------------------- | ------------------------------------------------------------------------------------------- |
| レビュー日                 | 2026-03-04                                                                                  |
| 対象ブランチ               | page-edit-fix                                                                               |
| ベースブランチ             | page-edit                                                                                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md, docs/plans/1_doing/page-edit-rails-go-diff.md |
| 変更ファイル数             | 49 ファイル（うち実装 25 + テスト 13 + ドキュメント/レビュー 6 + 自動生成 5）               |
| 変更行数（実装）           | +904 / -192 行                                                                              |
| 変更行数（テスト）         | +1458 / -17 行                                                                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版開発ガイド
- [@rails/docs/security-guide.md](/workspace/rails/docs/security-guide.md) - Rails版セキュリティガイドライン

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
- [x] `go/internal/templates/components/link_list.templ`
- [x] `go/internal/templates/components/page_backlink_list.templ`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/publish_page.go`
- [x] `go/internal/viewmodel/edit_link_data.go`
- [x] `go/internal/viewmodel/link_list.go`
- [x] `go/web/style.css`
- [x] `rails/app/controllers/attachments/presigns/create_controller.rb`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/page_backlink_list/show_test.go`
- [x] `go/internal/handler/page_backlinks/main_test.go`
- [x] `go/internal/handler/page_backlinks/show_test.go`
- [x] `go/internal/handler/page_link_list/main_test.go`
- [x] `go/internal/handler/page_link_list/show_test.go`
- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/templates/components/link_list_test.go`
- [x] `go/internal/templates/components/page_backlink_list_test.go`
- [x] `go/internal/usecase/auto_save_draft_page_test.go`
- [x] `go/internal/usecase/publish_page_test.go`
- [x] `rails/spec/requests/attachments/presigns/create_spec.rb`

### 自動生成ファイル

- [x] `go/internal/templates/components/backlink_list_templ.go`
- [x] `go/internal/templates/components/link_list_templ.go`
- [x] `go/internal/templates/components/page_backlink_list_templ.go`
- [x] `go/internal/templates/pages/page/edit_templ.go`

### ドキュメント

- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/1_doing/page-edit-rails-go-diff.md`
- [x] `docs/plans/3_done/202603/e2e-ci-fix.md`
- [x] `docs/reviews/done/202603/page-edit-fix-003.md`
- [x] `docs/reviews/done/202603/page-edit-fix-016.md`
- [x] `docs/reviews/done/202603/page-edit-fix-017.md`

## ファイルごとのレビュー結果

問題なし。全ファイルがガイドラインに準拠しています。

## 設計との整合性チェック

### page-edit-rails-go-diff.md（差分修正計画書）

| タスク                                                                | ステータス | 確認結果                                                                                                                                                                                                                          |
| --------------------------------------------------------------------- | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **差分1**: Wikiリンクによるページ自動作成時に`page_editors`を作成する | 完了       | `resolveAndCreateLinkedPages`に`pageEditorRepo`と`spaceMemberID`が追加され、ページ新規作成時に`pageEditorRepo.FindOrCreate`が呼ばれる。`AutoSaveDraftPageUsecase`と`PublishPageUsecase`の両方で対応済み。テストも追加されている。 |
| **差分2**: `FindByTopicAndTitle`から`discarded_at`フィルタを削除する  | 完了       | `pages.sql`の`FindPageByTopicAndTitle`から`AND discarded_at IS NULL`が削除済み。コメントも「廃棄済みを含む」に更新済み。`pages.sql.go`も再生成済み。テストも追加されている。                                                      |

### page-edit-go-migration.md（親計画書）との整合性

差分修正以外の追加変更も確認:

| 変更内容                                              | 整合性                                                                                                                                 |
| ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `CreateLinkedPage`の`published_at`を`NULL`に変更      | Rails版の`first_or_create!`は`published_at`を設定しないためNULL。Go版がRails版と同じ挙動に修正された。正しい。                         |
| バックリンクのexcludePageIDs対応                      | 編集中のページ自身とリンク先ページをバックリンクから除外。UIの重複表示を防ぐ改善。                                                     |
| `page_backlinks`/`page_link_list`ハンドラーの新規追加 | SSEによるページネーションの追加ページ取得用。ページ編集画面のリンク一覧/バックリンク一覧のページネーション対応。                       |
| Rails側の`skip_forgery_protection`追加                | Go版のページ編集画面からファイルアップロード時、Rails側のCSRF検証をスキップする必要がある。認証とSameSite Cookieで保護。テストも追加。 |
| `PublishPageUsecase`のDraftPage削除を条件付きに       | `DraftPageID`が空の場合（下書きなしで直接公開する場合）にDraftPage削除をスキップ。合理的な修正。                                       |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書（page-edit-rails-go-diff.md）に記載された差分1（page_editors作成漏れ）と差分2（discarded_at フィルタ問題）の修正が正しく実装されている。加えて、`CreateLinkedPage`の`published_at`がRails版に合わせて`NULL`に修正され、バックリンクのexcludePageIDs対応やSSEページネーションの追加ページ取得ハンドラーなど、UI品質向上の改善も含まれている。

良い点:

- 3層アーキテクチャの依存関係ルールを厳守している（Handler → Repository、UseCase → Repository）
- すべてのSQLクエリで`space_id`が条件に含まれており、セキュリティガイドラインに準拠
- 認証・認可チェックが全ハンドラーで一貫して実装されている
- テストカバレッジが充実しており、正常系・異常系・セキュリティ系のケースを網羅
- Rails版の`skip_forgery_protection`追加に対して、セキュリティトレードオフを考慮したコメントとテストが追加されている
- `findOrCreateLinkedPage`の戻り値にboolを追加してページ新規作成の有無を判定する設計は、既存コードへの影響を最小限に抑えつつ差分1を解決している
