# コードレビュー: go-page-edit-2-1

## レビュー情報

| 項目                       | 内容                                       |
| -------------------------- | ------------------------------------------ |
| レビュー日                 | 2026-03-07                                 |
| 対象ブランチ               | go-page-edit-2-1                           |
| ベースブランチ             | go-page-edit                               |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-rollout.md |
| 変更ファイル数             | 36 ファイル                                |
| 変更行数（実装）           | +4 / -763 行                               |
| 変更行数（テスト）         | +0 / -2324 行                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版の開発ガイド
- [@rails/docs/architecture-guide.md](/workspace/rails/docs/architecture-guide.md) - アーキテクチャガイド
- [@rails/docs/security-guide.md](/workspace/rails/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `rails/app/components/pages/header_actions_component.html.erb`
- [x] `rails/app/controllers/draft_pages/update_controller.rb`（削除）
- [x] `rails/app/controllers/page_locations/index_controller.rb`（削除）
- [x] `rails/app/controllers/pages/edit_controller.rb`（削除）
- [x] `rails/app/controllers/pages/new_controller.rb`
- [x] `rails/app/controllers/pages/update_controller.rb`（削除）
- [x] `rails/app/forms/pages/edit_form.rb`（削除）
- [x] `rails/app/services/draft_pages/update_service.rb`（削除）
- [x] `rails/app/services/pages/update_service.rb`（削除）
- [x] `rails/app/views/draft_pages/sidebar_view.html.erb`
- [x] `rails/app/views/draft_pages/update_view.html.erb`（削除）
- [x] `rails/app/views/draft_pages/update_view.rb`（削除）
- [x] `rails/app/views/pages/edit_view.html.erb`（削除）
- [x] `rails/app/views/pages/edit_view.rb`（削除）

### テストファイル

- [x] `rails/spec/forms/pages/edit_form_spec.rb`（削除）
- [x] `rails/spec/requests/draft_pages/update_spec.rb`（削除）
- [x] `rails/spec/requests/page_locations/index_spec.rb`（削除）
- [x] `rails/spec/requests/pages/edit_spec.rb`（削除）
- [x] `rails/spec/requests/pages/update_spec.rb`（削除）
- [x] `rails/spec/services/pages/update_service_spec.rb`（削除）
- [x] `rails/spec/system/components/markdown_editor_component/file_upload_spec.rb`（削除）
- [x] `rails/spec/system/components/markdown_editor_component/list_continuation_spec.rb`（削除）
- [x] `rails/spec/system/components/markdown_editor_component/paste_spec.rb`（削除）
- [x] `rails/spec/system/components/markdown_editor_component/shared_helpers.rb`（削除）
- [x] `rails/spec/system/components/markdown_editor_component/tab_indent_spec.rb`（削除）
- [x] `rails/spec/system/components/markdown_editor_component/wiki_link_autocomplete_spec.rb`（削除）
- [x] `rails/spec/system/global_hotkey_spec.rb`（削除）

### 設定・その他

- [x] `rails/config/routes.rb`
- [x] `rails/config/locales/forms.en.yml`
- [x] `rails/config/locales/forms.ja.yml`
- [x] `rails/config/locales/meta.en.yml`
- [x] `rails/config/locales/meta.ja.yml`
- [x] `rails/sorbet/rbi/dsl/generated_path_helpers_module.rbi`（自動生成）
- [x] `rails/sorbet/rbi/dsl/generated_url_helpers_module.rbi`（自動生成）
- [x] `rails/sorbet/rbi/dsl/pages/edit_form.rbi`（自動生成・削除）
- [x] `docs/plans/1_doing/page-edit-go-rollout.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルが正常にレビューを通過しました。

## 設計との整合性チェック

### 作業計画書タスク 2-1 との対応

作業計画書のタスク 2-1 に記載された削除対象と実装を照合した結果：

| 計画書の削除対象                                     | 実装状況  |
| ---------------------------------------------------- | --------- |
| `app/controllers/pages/edit_controller.rb`           | ✅ 削除済 |
| `app/controllers/pages/update_controller.rb`         | ✅ 削除済 |
| `app/controllers/draft_pages/update_controller.rb`   | ✅ 削除済 |
| `app/controllers/page_locations/index_controller.rb` | ✅ 削除済 |
| `app/services/pages/update_service.rb`               | ✅ 削除済 |
| `app/services/draft_pages/update_service.rb`         | ✅ 削除済 |
| `app/forms/pages/edit_form.rb`                       | ✅ 削除済 |
| `app/views/pages/edit_view.html.erb`                 | ✅ 削除済 |
| `app/views/draft_pages/update_view.html.erb`         | ✅ 削除済 |
| 関連するルーティング                                 | ✅ 削除済 |
| 関連するテスト                                       | ✅ 削除済 |
| 関連する翻訳                                         | ✅ 削除済 |

### 削除しないもの（計画書の指定）の確認

| 保持対象                              | 確認結果                    |
| ------------------------------------- | --------------------------- |
| `PageRecord`, `DraftPageRecord`       | ✅ 削除されていない         |
| ページ表示関連（`ShowController` 等） | ✅ 削除されていない         |
| フィーチャーフラグの仕組み            | ✅ 削除されていない（別PR） |

### 追加で削除されたもの（計画書に明示されていないが適切な削除）

- `app/views/pages/edit_view.rb`、`app/views/draft_pages/update_view.rb` - ビュークラス（テンプレートと対になるもの）
- `spec/system/components/markdown_editor_component/` 配下一式 - ページエディタのシステムテスト
- `spec/system/global_hotkey_spec.rb` - ページ編集に関連するグローバルホットキーテスト
- Sorbet RBI ファイル（自動生成）

### `edit_page_path` ヘルパーの置き換え

ルート削除に伴い `edit_page_path` ヘルパーが使えなくなったため、以下の3箇所で文字列補間によるハードコードURLに置き換えられている：

1. `rails/app/controllers/pages/new_controller.rb` - リダイレクト先
2. `rails/app/components/pages/header_actions_component.html.erb` - 編集リンク
3. `rails/app/views/draft_pages/sidebar_view.html.erb` - 下書き一覧の編集リンク

Go版のページ編集URLは `/s/:space_identifier/pages/:page_number/edit` で、Railsルーティング外のため文字列補間は妥当。残存参照の検索でも `edit_page_path`、`page_location_list_path`、`draft_page_path` への参照は存在しないことを確認済み。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 2-1 の要件通り、Rails版ページ編集・公開関連のコードが正確に削除されている。

- 削除対象のコントローラー・サービス・フォーム・ビュー・ルーティング・テスト・翻訳がすべて適切に削除されている
- 保持すべきレコードやページ表示関連のコードは影響を受けていない
- ルートヘルパーの削除に伴う3箇所のURL置き換えも適切に対応されている
- 削除されたクラスへの残存参照がないことを確認済み
- 36ファイルの変更で +4 / -3087 行の大幅なコード削除だが、削除PRとして適切なスコープ
