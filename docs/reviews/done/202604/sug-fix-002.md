# コードレビュー: sug-fix

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-04-05                       |
| 対象ブランチ               | sug-fix                          |
| ベースブランチ             | develop                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 71 ファイル                      |
| 変更行数（実装）           | +3906 / -2983 行                 |
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
- [x] `go/internal/templates/pages/draft_page/index.templ`
- [x] `go/internal/templates/pages/email_confirmation/edit.templ`
- [ ] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page_move/new.templ`
- [x] `go/internal/templates/pages/password/edit.templ`
- [x] `go/internal/templates/pages/password/reset.templ`
- [x] `go/internal/templates/pages/sign_in/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/recovery_new.templ`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [ ] `go/internal/templates/pages/suggestion/edit.templ`
- [x] `go/internal/templates/pages/suggestion/index.templ`
- [ ] `go/internal/templates/pages/suggestion/new.templ`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_comment/edit.templ`
- [x] `go/internal/templates/pages/suggestion_page/new.templ`
- [x] `go/internal/viewmodel/diff.go`

### テストファイル

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

### `go/internal/templates/pages/suggestion/new.templ`、`go/internal/templates/pages/suggestion/edit.templ`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

**問題点・改善提案**:

- **インデントの不整合**: `@components.ContentCard() { ... }` 内の `<form>` 要素の子要素で、`<input type="hidden">` と `<div class="grid gap-2">` のインデントレベルが揃っていない。`<input>` は 8 タブだが、`<div class="grid gap-2">` は 9 タブになっており、同じ DOM レベルの要素なのに 1 タブ分ずれている。

  suggestion/new.templ の例（83 行目 vs 85 行目）:

  ```templ
  // 現状（8タブ）
  								<input type="hidden" name="csrf_token" value={ data.CSRFToken }/>

  // 現状（9タブ — ずれている）
  									<div class="grid gap-2">
  ```

  同様に、`</form>` の閉じタグ（188 行目）は 8 タブだが、ContentCard の閉じ `}`（189 行目）は 6 タブ。`</form>` は `<form>`（7 タブ）の閉じなので 7 タブであるべき。

  suggestion/edit.templ でも同じパターン（68-71 行目、139-140 行目）。

  **修正案**:

  `<div class="grid gap-2">` のインデントを `<input>` と同じレベル（8 タブ）に揃え、`</form>` も開始タグと同じ 7 タブに揃える。

  **対応方針**:
  - [x] インデントを揃える
  - [ ] 現状維持（templ generate で自動修正されるため問題ない）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/page/edit.templ`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

**問題点・改善提案**:

- **インデントの不整合**: suggestion/new.templ、suggestion/edit.templ と同じ問題。`@components.ContentCard() { ... }` にリファクタリングした際に、`<form>` タグの属性内 `if` ブロック（87 行目）が 10 タブとなっており、`<form` の 7 タブから大きく離れている。また `</form>`（234 行目）が 9 タブだが、開始 `<form` は 7 タブ。

  ```templ
  // 85行目: 6タブ
  					@components.ContentCard() {
  // 86行目: 7タブ
  						<form
  // 87行目: 10タブ（大きくずれている）
  									if data.IsSuggestionMode() {
  // 234行目: 9タブ（</form>が<form>と揃っていない）
  								</form>
  // 235行目: 6タブ（ContentCardの閉じは正しい）
  					}
  ```

  **修正案**:

  `<form>` タグの属性を 8 タブに揃え、`</form>` を 7 タブに揃える。

  **対応方針**:
  - [x] インデントを揃える
  - [ ] 現状維持（templ generate で自動修正されるため問題ない）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

前回レビュー（sug-fix-001）で指摘された 2 点（`diff_no_changes` の日英対称性、`suggestion_change_remove_page_confirm` の表現スタイル）はいずれも適切に対応されていることを確認した。

フラッシュメッセージのミドルウェア化は全ハンドラーに一貫して適用されており、`DefaultLayoutData`・`SimpleLayoutData` から `Flash` フィールドが正しく削除されている。レイアウトテンプレートは `session.FlashFromContext(ctx)` を使用してコンテキストからフラッシュを取得しており、各ハンドラーの `GetFlash(w, r)` 呼び出しが確実に除去されている。

差分計算の CRLF 正規化（`normalizeNewlines`）は、テストも末尾改行なし・CRLF/LF 混在の両ケースをカバーしており、十分。

新規コンポーネント（`ContentCard`、`OptionalLabel`）は再利用可能な粒度で適切に設計されている。`ContentCard` の導入により、繰り返しパターン（`<div class="card py-4 rounded-none md:rounded-xl mx-0 md:mx-4"><section class="px-4">...</section></div>`）が統一された。

翻訳の改善（タイトルセパレータの `|` 統一、トピック名の追加、ボタンテキストの簡潔化）も一貫性のある変更であり、UI の情報量が向上している。

## 総合評価

**評価**: Comment

**総評**:

本 PR は前回レビュー（sug-fix-001）の指摘対応に加え、以下の改善を含む:

1. **フラッシュメッセージのミドルウェア化**: 全 15 ハンドラーから `GetFlash(w, r)` を削除し、ミドルウェア経由でコンテキストに格納する方式に統一。レイアウトデータ構造体の簡素化とボイラープレートの削減が実現されている
2. **差分計算の CRLF 正規化**: `normalizeNewlines` 関数の追加とテストの網羅
3. **UI リファクタリング**: `ContentCard`・`OptionalLabel` コンポーネントの導入、タイトルフォーマットの統一、編集提案フォームの改善

指摘事項はテンプレートのインデント不整合（3 ファイル）のみであり、機能的な問題やアーキテクチャ違反はない。ハンドラーはすべて UseCase のみに依存しており、3 層アーキテクチャのルールに準拠している。
