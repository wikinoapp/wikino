# コードレビュー: go-topic-fix-1

## レビュー情報

| 項目                       | 内容                                          |
| -------------------------- | --------------------------------------------- |
| レビュー日                 | 2026-03-11                                    |
| 対象ブランチ               | go-topic-fix-1                                |
| ベースブランチ             | go-topic                                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/topic-show-go-migration.md |
| 変更ファイル数             | 24 ファイル                                   |
| 変更行数（実装）           | +446 / -545 行                                |
| 変更行数（テスト）         | +32 / -160 行                                 |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版の開発ガイド

## 変更ファイル一覧

### 実装ファイル（Go版）

- [x] `go/internal/handler/topic/show.go`
- [ ] `go/internal/templates/pages/topic/show.templ`
- [x] `go/internal/templates/components/card_link_page.templ`
- [x] `go/internal/templates/components/pagination.templ`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/icons_phosphor.go`
- [x] `go/internal/viewmodel/page.go`
- [x] `go/internal/viewmodel/link_list.go`
- [x] `go/internal/viewmodel/backlink_list.go`

### 自動生成ファイル

- [x] `go/internal/templates/components/card_link_page_templ.go`（自動生成）
- [x] `go/internal/templates/components/pagination_templ.go`（自動生成）
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/internal/templates/pages/topic/show_templ.go`（自動生成）

### テストファイル

- [x] `go/internal/viewmodel/page_test.go`

### 実装ファイル（Rails版 - 削除）

- [x] `rails/app/controllers/topics/show_controller.rb`（削除）
- [x] `rails/app/views/topics/show_view.rb`（削除）
- [x] `rails/app/views/topics/show_view.html.erb`（削除）
- [x] `rails/app/views/topics/show_view/header_component.rb`（削除）
- [x] `rails/app/views/topics/show_view/header_component.html.erb`（削除）
- [x] `rails/spec/requests/topics/show_spec.rb`（削除）
- [x] `rails/spec/system/global_hotkey_spec.rb`（修正）

### 設定・その他

- [x] `Dockerfile.dev`（修正）
- [x] `docs/plans/1_doing/topic-show-go-migration.md`（修正）
- [x] `docs/reviews/done/202603/go-topic-fix-1-001.md`（追加）

## ファイルごとのレビュー結果

### `go/internal/templates/pages/topic/show.templ`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **インデントの不統一**: ドロップダウンメニュー部分（127〜140行目）でタブとスペースが混在しており、インデントが崩れています。templ ファイルではタブによるインデントで統一すべきです。

  ```templ
  // 現在のコード（インデントが混在）
  <div id="topic-options-dropdown-popover" data-popover aria-hidden="true" class="min-w-56" data-align="end">
      <div role="menu" id="topic-options-dropdown-menu" aria-labelledby="topic-options-dropdown-trigger">
          <div role="menuitem">
            <a
                class="flex gap-2 items-center w-full"
  ```

  **修正案**: タブインデントで統一してください。

  ```templ
  <div id="topic-options-dropdown-popover" data-popover aria-hidden="true" class="min-w-56" data-align="end">
  	<div role="menu" id="topic-options-dropdown-menu" aria-labelledby="topic-options-dropdown-trigger">
  		<div role="menuitem">
  			<a
  				class="flex gap-2 items-center w-full"
  				href={ templ.SafeURL(string(templates.TopicSettingsPath(data.Space.Identifier.String(), data.Topic.Number))) }
  			>
  				@templates.Icon("gear-regular", "size-[18px]")
  				{ templates.T(ctx, "topic_show_settings") }
  			</a>
  		</div>
  	</div>
  </div>
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りタブインデントに統一する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

### 作業計画書との整合性

作業計画書と今回のGo側UIリファクタリングの整合性を確認しました。

| 要件                                             | 状態 | 備考                                                 |
| ------------------------------------------------ | ---- | ---------------------------------------------------- |
| ページカードにトピック情報を表示（カード内）     | ✅   | `Topic *Topic` ポインタで表示を制御（nilなら非表示） |
| 編集ボタンの権限制御                             | ✅   | `CanEdit` フィールドで制御                           |
| ドロップダウンメニューによるトピック設定アクセス | ✅   | `dots-three-regular` アイコンでドロップダウン実装    |
| ページネーションのUI改善                         | ✅   | `btn-ghost` + キャレットアイコン追加                 |
| max-width を 5xl → 3xl に変更                    | ✅   | パンくずリストにも `MaxWidthClass` を追加して統一    |
| Rails版の不要コード削除（3a-1）                  | ✅   | 前回レビュー（001）で確認済み                        |

### ViewModelの設計

`CardLinkPage` の `TopicName`/`TopicIcon` を `*Topic` ポインタに変更した設計は、アーキテクチャガイドの「モデルの重複を避ける」方針に沿っており適切です。`CanEdit` フィールドの追加も、既存の `Primary` フィールドと同様にコンストラクタ外で設定するパターンを踏襲しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

トピック詳細画面のUI改善とViewModelのリファクタリングは全体として適切に行われています。

良かった点:

- `CardLinkPage` の `TopicName`/`TopicIcon` を `*Topic` ポインタに変更し、ViewModelの構造をよりシンプルにした
- `CanEdit` フィールドの追加で編集ボタンの表示/非表示を権限ベースで制御できるようにした
- タッチデバイスとデスクトップで異なる編集ボタンUIを提供している
- テストが適切に更新されている（新しいテストケース追加含む）
- 既存の `link_list.go` と `backlink_list.go` にも一貫して `CanEdit` を適用している

修正が必要な点:

- `show.templ` のドロップダウンメニュー部分でタブとスペースが混在しており、インデントが崩れている（1件）
