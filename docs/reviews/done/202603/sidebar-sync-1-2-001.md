# コードレビュー: sidebar-sync-1-2

## レビュー情報

| 項目                       | 内容                                       |
| -------------------------- | ------------------------------------------ |
| レビュー日                 | 2026-03-07                                 |
| 対象ブランチ               | sidebar-sync-1-2                           |
| ベースブランチ             | sidebar-sync                               |
| 作業計画書（指定があれば） | docs/plans/1_doing/sidebar-sync.md         |
| 変更ファイル数             | 23 ファイル                                |
| 変更行数（実装）           | +229 / -383 行（テスト含む、差し引き削減） |
| 変更行数（テスト）         | +5 / -32 行                                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版コーディング規約
- [@rails/docs/architecture-guide.md](/workspace/rails/docs/architecture-guide.md) - アーキテクチャガイド
- [@rails/docs/security-guide.md](/workspace/rails/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `rails/app/assets/stylesheets/application.css`
- [x] `rails/app/components/base_ui/breadcrumb_component.html.erb`
- [x] `rails/app/components/breadcrumbs/settings_component.html.erb`
- [x] `rails/app/components/breadcrumbs/space_component.html.erb`
- [ ] `rails/app/components/headers/global_component.html.erb`
- [x] `rails/app/components/layouts/column1_component.html.erb`
- [x] `rails/app/components/navbars/bottom_component.html.erb`
- [x] `rails/app/components/sidebar/content_component.html.erb`（削除）
- [x] `rails/app/components/sidebar/content_component.rb`（削除）
- [x] `rails/app/components/sidebar/item_link_component.html.erb`（削除）
- [x] `rails/app/components/sidebar/item_link_component.rb`（削除）
- [x] `rails/app/components/sidebar/joined_topics_component.html.erb`
- [x] `rails/app/components/sidebar/joined_topics_component.rb`
- [x] `rails/app/components/sidebar_component.html.erb`
- [x] `rails/app/components/sidebar_component.rb`
- [x] `rails/app/controllers/joined_topics/index_controller.rb`
- [x] `rails/app/javascript/application.ts`
- [x] `rails/app/views/joined_topics/index_view.html.erb`
- [x] `rails/app/views/joined_topics/index_view.rb`

### テストファイル

- [x] `rails/spec/system/joined_topics/index_spec.rb`

### 設定・その他

- [x] `rails/sorbet/rbi/dsl/generated_path_helpers_module.rbi`（自動生成）
- [x] `rails/sorbet/rbi/dsl/generated_url_helpers_module.rbi`（自動生成）
- [x] `docs/plans/1_doing/sidebar-sync.md`

## ファイルごとのレビュー結果

### `rails/app/components/headers/global_component.html.erb`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - コーディング規約
- [@rails/docs/architecture-guide.md](/workspace/rails/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **未使用パラメータ `show_sidebar`**: テンプレートから `show_sidebar` の参照が削除されたが、`global_component.rb` にはパラメータ・attr_reader・初期化ロジックが残っている。`spaces/show_view.html.erb` から `show_sidebar: layout.show_sidebar?` が渡されているが、テンプレートでは使用されておらずデッドコードになっている。

  サイドバーの表示制御は `layouts/column1_component` の `show_sidebar?` で行われているため、ヘッダーのトグルボタンを常に表示する設計自体は妥当。ただしパラメータが未使用のまま残っているのは整理が必要。

  **修正案**:

  `global_component.rb` から `show_sidebar` パラメータを削除し、呼び出し元（`spaces/show_view.html.erb` 等）からも引数を削除する。

  **対応方針**:
  - [x] `show_sidebar` パラメータを `global_component.rb` と呼び出し元から削除する
  - [ ] タスク 1-3（Stimulus削除）と合わせて対応する
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

タスク 1-2（basecoat-cssサイドバーコンポーネントへの移行）の要件を適切に実装している。主要な変更点：

- `SidebarComponent` を basecoat-css の `<aside class="sidebar">` 構造に正しく書き換えている
- `Sidebar::ContentComponent` と `Sidebar::ItemLinkComponent` を削除し、`SidebarComponent` に統合している
- `Sidebar::JoinedTopicsComponent` の `variant` 引数を正しく削除している
- ナビゲーションのアイコンサイズ（22px → 16px）、スタイリング（`text-gray-700` → `text-foreground`、`hover:bg-brand-300/70` → `hover:bg-brand-300/40` 等）をGo版に合わせている
- basecoat-css JS（sidebar.min.js）の動的読み込みを追加している
- CSS変数（`--sidebar`, `--sidebar-accent` 等）を追加している
- パンくずリストにホームアイコンを追加し、`<nav>` で囲む改善を行っている
- ヘッダーのレイアウトを3カラム構成（トグルボタン / パンくず / 空き）に変更し、パンくずを中央配置にしている
- テストを変更に合わせて適切に更新し、不要になった variant テストを削除している

指摘は1件（`show_sidebar` パラメータのデッドコード）のみで、軽微な整理事項。タスク 1-3 と合わせて対応しても問題ない。
