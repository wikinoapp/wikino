# コードレビュー: sug-fix

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-04-05                       |
| 対象ブランチ               | sug-fix                          |
| ベースブランチ             | develop                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 71 ファイル                      |
| 変更行数（実装）           | +494 / -429 行（概算）           |
| 変更行数（テスト）         | +87 / -0 行                      |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/account/new.go`
- [x] `go/internal/handler/draft_page_index/index.go`
- [x] `go/internal/handler/email_confirmation/edit.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/handler/page_move/new.go`
- [x] `go/internal/handler/sign_in/new.go`
- [x] `go/internal/handler/suggestion/edit.go`
- [x] `go/internal/handler/suggestion/index.go`
- [x] `go/internal/handler/suggestion/new.go`
- [x] `go/internal/handler/suggestion/show.go`
- [x] `go/internal/handler/suggestion_change/index.go`
- [x] `go/internal/handler/suggestion_comment_edit/edit.go`
- [x] `go/internal/handler/suggestion_page/new.go`
- [x] `go/internal/handler/suggestion_page_edit/show.go`
- [x] `go/internal/handler/topic/show.go`
- [x] `go/internal/handler/welcome/show.go`
- [x] `go/internal/session/flash.go`
- [x] `go/internal/templates/components/content_card.templ`
- [x] `go/internal/templates/components/diff.templ`
- [x] `go/internal/templates/components/optional_label.templ`
- [x] `go/internal/templates/icons_phosphor.go`
- [x] `go/internal/templates/layouts/default.templ`
- [x] `go/internal/templates/layouts/simple.templ`
- [x] `go/internal/templates/pages/account/new.templ`
- [ ] `go/internal/templates/pages/draft_page/index.templ`
- [x] `go/internal/templates/pages/email_confirmation/edit.templ`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page_move/new.templ`
- [x] `go/internal/templates/pages/password/edit.templ`
- [x] `go/internal/templates/pages/password/reset.templ`
- [x] `go/internal/templates/pages/sign_in/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/recovery_new.templ`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/suggestion/edit.templ`
- [x] `go/internal/templates/pages/suggestion/index.templ`
- [x] `go/internal/templates/pages/suggestion/new.templ`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [ ] `go/internal/templates/pages/suggestion_comment/edit.templ`
- [ ] `go/internal/templates/pages/suggestion_page/new.templ`
- [x] `go/internal/viewmodel/diff.go`

### テストファイル

- [x] `go/internal/handler/draft_page_index/index_test.go`
- [x] `go/internal/viewmodel/diff_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/templates/components/content_card_templ.go` (自動生成)
- [x] `go/internal/templates/components/diff_templ.go` (自動生成)
- [x] `go/internal/templates/components/optional_label_templ.go` (自動生成)
- [x] `go/internal/templates/layouts/default_templ.go` (自動生成)
- [x] `go/internal/templates/layouts/simple_templ.go` (自動生成)
- [x] `go/internal/templates/pages/account/new_templ.go` (自動生成)
- [x] `go/internal/templates/pages/draft_page/index_templ.go` (自動生成)
- [x] `go/internal/templates/pages/email_confirmation/edit_templ.go` (自動生成)
- [x] `go/internal/templates/pages/page/edit_templ.go` (自動生成)
- [x] `go/internal/templates/pages/page_move/new_templ.go` (自動生成)
- [x] `go/internal/templates/pages/password/edit_templ.go` (自動生成)
- [x] `go/internal/templates/pages/password/reset_templ.go` (自動生成)
- [x] `go/internal/templates/pages/sign_in/new_templ.go` (自動生成)
- [x] `go/internal/templates/pages/sign_in_two_factor/new_templ.go` (自動生成)
- [x] `go/internal/templates/pages/sign_in_two_factor/recovery_new_templ.go` (自動生成)
- [x] `go/internal/templates/pages/sign_up/new_templ.go` (自動生成)
- [x] `go/internal/templates/pages/suggestion/edit_templ.go` (自動生成)
- [x] `go/internal/templates/pages/suggestion/index_templ.go` (自動生成)
- [x] `go/internal/templates/pages/suggestion/new_templ.go` (自動生成)
- [x] `go/internal/templates/pages/suggestion/show_templ.go` (自動生成)
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go` (自動生成)
- [x] `go/internal/templates/pages/suggestion_comment/edit_templ.go` (自動生成)
- [x] `go/internal/templates/pages/suggestion_page/new_templ.go` (自動生成)

## ファイルごとのレビュー結果

### `go/internal/templates/pages/suggestion_comment/edit.templ`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

**問題点・改善提案**:

- **インデントの不整合**: `@components.ContentCard() { ... }` 内の `<form>` 要素の子要素のインデントが `<form>` 開始タグのレベルと合っていない。`<form` は 7 タブで開始しているが、68-69 行目の `<input>` は 8 タブ（正しい）、71 行目の `<div class="grid gap-2">` は 9 タブ（8 タブであるべき）。また `</form>` (110 行目) が 8 タブだが、`<form` (63 行目) の 7 タブに合わせて 7 タブであるべき。

  ```templ
  // 現状 (62-71行目、110-111行目)
  					@components.ContentCard() {        // 6タブ
  						<form ...>                      // 7タブ
  							<input ... />               // 8タブ（正しい）

  								<div class="grid gap-2">  // 9タブ（ずれ）
  								...
  							</form>                       // 8タブ（ずれ）
  					}                                   // 6タブ（正しい）
  ```

  **修正案**:

  `<div class="grid gap-2">` 以下のフォーム内容を 8 タブに揃え、`</form>` を 7 タブに揃える。

  **対応方針**:
  - [x] インデントを揃える
  - [ ] 現状維持
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/suggestion_page/new.templ`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

**問題点・改善提案**:

- **インデントの不整合**: `@components.ContentCard()` 内の `<form>` の子要素が 2 タブ分ずれている。`<form` は 7 タブだが、`<input type="hidden">` (55 行目) が 10 タブ、`<div>` (57 行目) も 10 タブ。8 タブであるべき。また `</form>` (98 行目) が 9 タブだが、7 タブであるべき。

  ```templ
  // 現状 (49-55行目、98-99行目)
  					@components.ContentCard() {         // 6タブ
  						<form ...>                       // 7タブ
  									<input ... />        // 10タブ（2タブ分ずれ）
  									<div ...>            // 10タブ（2タブ分ずれ）
  									...
  								</form>                  // 9タブ（2タブ分ずれ）
  					}                                    // 6タブ（正しい）
  ```

  **修正案**:

  `<form>` 内の全要素を 8 タブに揃え、`</form>` を 7 タブに揃える。旧コード（`<div class="card"><div class="card-body">` の 2 層ラッピング）から `@components.ContentCard()` にリファクタリングした際に、内部のインデントが更新されていない。

  **対応方針**:
  - [x] インデントを揃える
  - [ ] 現状維持
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/draft_page/index.templ`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

**問題点・改善提案**:

- **インデントの不整合**: `@components.ContentCard() { ... }` 内の `<table>` 要素周辺のインデントが揃っていない。`@components.ContentCard()` は 9 タブ、`<div class="overflow-x-auto">` は 10 タブ（正しい）だが、`<table>` (74 行目) が 12 タブ（11 タブであるべき）。また `</div>` (106 行目) が 11 タブだが、開始 `<div>` が 10 タブなので 10 タブであるべき。

  ```templ
  // 現状 (72-74行目、106-107行目)
  									@components.ContentCard() {      // 9タブ
  										<div class="overflow-x-auto">  // 10タブ（正しい）
  												<table class="table">  // 12タブ（11タブであるべき）
  												...
  											</div>                     // 11タブ（10タブであるべき）
  									}                                  // 9タブ（正しい）
  ```

  **修正案**:

  `<table>` を 11 タブに、`</div>` を 10 タブに揃える。旧コードから `@components.ContentCard()` にリファクタリングした際のインデント調整漏れ。

  **対応方針**:
  - [x] インデントを揃える
  - [ ] 現状維持
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

前回レビュー（sug-fix-001）で指摘された 2 点（`diff_no_changes` の日英対称性、`suggestion_change_remove_page_confirm` の表現スタイル）は適切に対応されている。前回レビュー（sug-fix-002）で指摘された page/edit.templ のインデントも修正されている。

フラッシュメッセージのミドルウェア化は全ハンドラーに一貫して適用されており、`DefaultLayoutData`・`SimpleLayoutData` から `Flash` フィールドが正しく削除されている。`main.go` での `r.Use(flashMgr.Middleware)` の配置も CSRF ミドルウェアの後で適切。

差分計算の CRLF 正規化と末尾改行統一は、テストケースが末尾改行なし・CRLF/LF 混在の両方をカバーしており十分。

`ContentCard` コンポーネントの導入により、`<div class="card ..."><section class="px-4">` の繰り返しパターンが統一されている。`OptionalLabel` も再利用可能な粒度で適切。

編集提案一覧画面への「新規編集提案」ボタン追加（`suggestionNewButton` templ 関数）は、作業計画書のUI設計（トピック詳細画面から編集提案一覧へのナビゲーション）に沿っている。

翻訳キーの改善（タイトルセパレータ `|` 統一、トピック名追加、ボタンテキスト簡潔化、「任意」ラベル追加）は一貫性があり、日英の対称性も保たれている。

## 総合評価

**評価**: Comment

**総評**:

本 PR は前回レビュー（sug-fix-001, sug-fix-002）の指摘対応に加え、以下の改善を含む:

1. **フラッシュメッセージのミドルウェア化**: 全 15 ハンドラーから `GetFlash(w, r)` を削除し、ミドルウェア経由でコンテキストに格納する方式に統一。レイアウトデータ構造体の簡素化とボイラープレートの削減が実現されている
2. **差分計算の CRLF 正規化**: `normalizeNewlines` 関数の追加と末尾改行統一。テストが網羅的
3. **UI リファクタリング**: `ContentCard`・`OptionalLabel` コンポーネントの導入、タイトルフォーマットの統一、編集提案フォームの改善、下書き一覧レイアウトの改善
4. **前回レビュー指摘の対応**: i18n の日英対称性の修正、page/edit.templ のインデント修正

指摘事項はテンプレートのインデント不整合（3 ファイル）のみであり、機能的な問題やアーキテクチャ違反はない。いずれも `@components.ContentCard()` へのリファクタリング時に旧コードのインデントレベルが残っている軽微な問題。ハンドラーはすべて UseCase のみに依存しており、3 層アーキテクチャのルールに準拠している。作業計画書との乖離も見られない。
