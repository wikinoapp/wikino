# コードレビュー: go-topic-fix-1

## レビュー情報

| 項目                       | 内容                                                             |
| -------------------------- | ---------------------------------------------------------------- |
| レビュー日                 | 2026-03-11                                                       |
| 対象ブランチ               | go-topic-fix-1                                                   |
| ベースブランチ             | go-topic                                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/topic-show-go-migration.md                    |
| 変更ファイル数             | 25 ファイル                                                      |
| 変更行数（実装）           | +354 / -438 行（自動生成ファイル・ドキュメント・Rails 削除除く） |
| 変更行数（テスト）         | +32 / -160 行                                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版開発ガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/topic/show.go`
- [x] `go/internal/templates/components/card_link_page.templ`
- [x] `go/internal/templates/components/pagination.templ`
- [x] `go/internal/templates/icons_phosphor.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/topic/show.templ`
- [x] `go/internal/viewmodel/backlink_list.go`
- [x] `go/internal/viewmodel/link_list.go`
- [x] `go/internal/viewmodel/page.go`

### テストファイル

- [x] `go/internal/viewmodel/page_test.go`

### 自動生成ファイル

- [x] `go/internal/templates/components/card_link_page_templ.go`
- [x] `go/internal/templates/components/pagination_templ.go`
- [x] `go/internal/templates/pages/page/edit_templ.go`
- [x] `go/internal/templates/pages/topic/show_templ.go`

### Rails 版削除ファイル

- [x] `rails/app/controllers/topics/show_controller.rb`（削除）
- [x] `rails/app/views/topics/show_view.html.erb`（削除）
- [x] `rails/app/views/topics/show_view.rb`（削除）
- [x] `rails/app/views/topics/show_view/header_component.html.erb`（削除）
- [x] `rails/app/views/topics/show_view/header_component.rb`（削除）
- [x] `rails/spec/requests/topics/show_spec.rb`（削除）
- [x] `rails/spec/system/global_hotkey_spec.rb`（部分削除）

### 設定・その他

- [x] `Dockerfile.dev`
- [x] `docs/plans/1_doing/topic-show-go-migration.md`
- [x] `docs/reviews/done/202603/go-topic-fix-1-001.md`
- [x] `docs/reviews/done/202603/go-topic-fix-1-002.md`

## ファイルごとのレビュー結果

### `go/internal/templates/pages/topic/show.templ`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

**問題点・改善提案**:

- **インデントの不整合**: ドロップダウンメニューの `<div id="topic-options-dropdown-popover">` 要素（127 行目）のインデントが、兄弟要素の `<button>`（116 行目）より 1 タブ深くなっている

  ```templ
  // 現在のコード（show.templ 115-127行目）
  		<div id="topic-options-dropdown" class="dropdown-menu">
  			<button
  				...
  			</button>

  				<div id="topic-options-dropdown-popover" ...>
  ```

  **修正案**:

  ```templ
  		<div id="topic-options-dropdown" class="dropdown-menu">
  			<button
  				...
  			</button>

  			<div id="topic-options-dropdown-popover" ...>
  ```

  **対応方針**:
  - [x] インデントを揃える
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

全体的に良い変更です。

**良かった点**:

- `CardLinkPage` の `TopicName`/`TopicIcon` を `*Topic` ポインタに統合した設計改善は、モデルの重複を避けるアーキテクチャガイドの方針に合致しており、可読性も向上している
- `CanEdit` フィールドの追加により、ページカードの編集ボタン表示制御がハンドラーからテンプレートへ適切に分離されている
- ドロップダウンメニューへの変更はアクセシビリティ属性（`aria-haspopup`, `aria-controls`, `role`）が適切に設定されている
- ページネーションコンポーネントへのアイコン追加は UX の改善に寄与している
- Rails 版の削除が作業計画書のタスク 3a-1 の内容と一致しており、ルート定義は他で使用されているため適切に残されている
- テストが新しいフィールド構造に対応して更新されており、nil topicMap のケースも追加されている

**指摘事項**:

- `show.templ` のドロップダウンポップオーバーのインデント不整合（軽微、修正任意）
