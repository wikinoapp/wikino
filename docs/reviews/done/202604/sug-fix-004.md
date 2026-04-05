# コードレビュー: sug-fix

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-04-05                       |
| 対象ブランチ               | sug-fix                          |
| ベースブランチ             | develop                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 74 ファイル                      |
| 変更行数（実装）           | +709 / -628 行（概算）           |
| 変更行数（テスト）         | +88 / -1 行                      |

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
- [x] `go/internal/templates/pages/draft_page/index.templ`
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
- [x] `go/internal/templates/pages/suggestion_comment/edit.templ`
- [x] `go/internal/templates/pages/suggestion_page/new.templ`
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

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

過去 3 回のレビュー（sug-fix-001, sug-fix-002, sug-fix-003）で指摘されたすべての問題が適切に対応されていることを確認した:

- **sug-fix-001**: `diff_no_changes` の日英対称性 → en.toml に "in body" スコープが追加され対称になっている
- **sug-fix-001**: `suggestion_change_remove_page_confirm` の日英ニュアンス差 → 英語が "(Your draft will remain)" に修正され日本語と対称
- **sug-fix-002**: suggestion/new.templ、suggestion/edit.templ、page/edit.templ のインデント不整合 → すべて修正済み
- **sug-fix-003**: suggestion_comment/edit.templ、suggestion_page/new.templ、draft_page/index.templ のインデント不整合 → すべて修正済み

フラッシュメッセージのミドルウェア化は全 15 ハンドラーに一貫して適用されており、`DefaultLayoutData`・`SimpleLayoutData` から `Flash` フィールドが正しく削除されている。`main.go` での `r.Use(flashMgr.Middleware)` の配置は CSRF ミドルウェアの後で適切。`FlashFromContext` は空構造体型のコンテキストキーを使用しており、Go のベストプラクティスに沿っている。

差分計算の `normalizeNewlines` は CRLF → LF 正規化と末尾改行の統一を適切に行っている。空文字列への不要な改行追加もない。テストケースは末尾改行なし・CRLF/LF 混在の両方をカバーしている。

`ContentCard`・`OptionalLabel` コンポーネントの導入により、テンプレートの繰り返しパターンが統一された。翻訳キーの改善（タイトルセパレータ `|` 統一、トピック名追加、ボタンテキスト簡潔化、「任意」ラベル追加）は一貫しており、日英の対称性も保たれている。

## 総合評価

**評価**: Approve

**総評**:

本 PR は過去 3 回のレビュー（sug-fix-001〜003）の指摘をすべて適切に対応した上で、以下の改善を含む:

1. **フラッシュメッセージのミドルウェア化**: 全 15 ハンドラーから `GetFlash(w, r)` を削除し、ミドルウェア経由でコンテキストに格納する方式に統一。レイアウトデータ構造体の簡素化とボイラープレートの削減が実現されている
2. **差分計算の CRLF 正規化**: `normalizeNewlines` 関数の追加と末尾改行統一。テストが網羅的
3. **UI リファクタリング**: `ContentCard`・`OptionalLabel` コンポーネントの導入、タイトルフォーマットの統一、編集提案フォームの改善、下書き一覧レイアウトの改善
4. **前回レビュー指摘の全対応**: i18n の日英対称性修正、テンプレートインデント修正（6 ファイル）

機能的な問題、アーキテクチャ違反、セキュリティ上の懸念はなし。ハンドラーはすべて UseCase のみに依存しており 3 層アーキテクチャのルールに準拠。作業計画書との乖離も見られない。マージ可能と判断する。
