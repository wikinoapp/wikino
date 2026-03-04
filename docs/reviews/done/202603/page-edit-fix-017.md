# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                                                                        |
| -------------------------- | ------------------------------------------------------------------------------------------- |
| レビュー日                 | 2026-03-04                                                                                  |
| 対象ブランチ               | page-edit-fix                                                                               |
| ベースブランチ             | page-edit                                                                                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md, docs/plans/1_doing/page-edit-rails-go-diff.md |
| 変更ファイル数             | 47 ファイル                                                                                 |
| 変更行数（実装）           | 約 +850 / -150 行（自動生成ファイル除く）                                                   |
| 変更行数（テスト）         | 約 +1475 / -20 行                                                                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

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
- [x] `rails/app/controllers/attachments/presigns/create_controller.rb`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/page_backlink_list/show_test.go`
- [x] `go/internal/handler/page_backlinks/main_test.go`
- [x] `go/internal/handler/page_backlinks/show_test.go`
- [x] `go/internal/handler/page_link_list/main_test.go`
- [x] `go/internal/handler/page_link_list/show_test.go`
- [x] `go/internal/middleware/reverse_proxy_test.go`
- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/templates/components/link_list_test.go`
- [x] `go/internal/templates/components/page_backlink_list_test.go`
- [x] `go/internal/usecase/auto_save_draft_page_test.go`
- [x] `go/internal/usecase/publish_page_test.go`
- [x] `rails/spec/requests/attachments/presigns/create_spec.rb`

### 設定・その他

- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/1_doing/page-edit-rails-go-diff.md`
- [x] `docs/plans/3_done/202603/e2e-ci-fix.md`
- [x] `docs/reviews/done/202603/page-edit-fix-003.md`
- [x] `docs/reviews/done/202603/page-edit-fix-016.md`
- [x] `go/internal/testutil/page_builder.go`

## ファイルごとのレビュー結果

### `go/internal/handler/draft_page/show.go` / `go/internal/handler/page/edit.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **コード重複**: `show.go`（draft_page）と`edit.go`（page）の変更が完全に同一のパターン（excludePageIDsの構築、`FindBacklinkedPagesPaginated`呼び出し、`NewBacklinkList`の引数変更）。両ファイルにまったく同じロジックが重複しており、将来の変更時に片方だけ更新してしまうリスクがある。

  **修正案**:

  現時点ではこの2ファイルの重複は許容範囲と考えられるが、今後同様の変更が入る場合はヘルパー関数への切り出しを検討すべき。

  **対応方針**:
  - [x] ヘルパー関数に切り出す
  - [ ] 現状のまま（許容範囲）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  viewmodel.BuildEditLinkData, viewmodel.BuildExcludePageIDs, viewmodel.CollectTopicIDsFromPages をヘルパー関数として切り出し、両ファイルから利用するようにリファクタリング済み。
  ```

### `go/db/queries/pages.sql` - `CreateLinkedPage`のpublished_atがNULLに変更

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@docs/plans/1_doing/page-edit-rails-go-diff.md](/workspace/docs/plans/1_doing/page-edit-rails-go-diff.md) - 差分分析ドキュメント

**問題点・改善提案**:

- **設計との整合性**: `CreateLinkedPage`クエリの`published_at`が`$5`（現在時刻）から`NULL`に変更されている。これはWikiリンクによる自動作成ページが「未公開」状態で作成されることを意味する。

  差分分析ドキュメント（page-edit-rails-go-diff.md）にはこの変更が差分として記載されていない。Rails版の`create_linked_page!`がpublished_atをどう設定しているか確認が必要。

  もしRails版がpublished_atを設定して公開状態で作成しているなら、Go版でNULLにすることは**新たな差分を生む**ことになり、差分修正の目的と矛盾する。逆にRails版もNULLで作成しているなら、これは既存の差分の修正であり正しい。

  **修正案**:

  Rails版の挙動を確認し、一致させる。

  **対応方針**:
  - [ ] Rails版がpublished_at=NULLで作成しているため、Go版もNULLで正しい
  - [ ] Rails版がpublished_atを設定して作成しているため、Go版も`$5`に戻す
  - [x] 意図的な仕様変更である（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  @docs/plans/1_doing/page-edit-go-migration.md に記載のとおり意図的な変更になります。仕様はRails版が正で、Rails版がこのようになっています。
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

差分分析ドキュメント（page-edit-rails-go-diff.md）に記載された2つの差分（page_editors未作成、廃棄済みページのフィルタリング）は適切に修正されている。テストも自動保存・公開の両方で検証しており、十分なカバレッジがある。

新規追加のSSEハンドラー（`page_link_list`, `page_backlinks`）、バックリンク除外ロジック、テンプレートの分割（Cards/Pagination）、Method Overrideの対応なども、アーキテクチャガイドラインに沿った実装がされている。

主な確認事項:

1. **`CreateLinkedPage`のpublished_at=NULL変更**: 差分分析ドキュメントに記載のない変更であり、Rails版との整合性を確認する必要がある
2. **コード重複**: `draft_page/show.go`と`page/edit.go`の同一ロジックは現時点で許容範囲だが、今後の保守性に注意

セキュリティ面では、Rails版presignコントローラーの`skip_forgery_protection`は認証とSameSite Cookieで保護されており、コメントで理由が説明されていて適切。テストも追加されている。
