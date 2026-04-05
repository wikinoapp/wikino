# コードレビュー: sug-fix

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-04-05                       |
| 対象ブランチ               | sug-fix                          |
| ベースブランチ             | develop                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 70 ファイル                      |
| 変更行数（実装）           | +494 / -429 行                   |
| 変更行数（テスト）         | +87 / -0 行                      |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

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

- [x] `go/internal/viewmodel/diff_test.go`

### 設定・その他

- [ ] `go/internal/i18n/locales/en.toml`
- [ ] `go/internal/i18n/locales/ja.toml`
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

### `go/internal/i18n/locales/ja.toml` / `go/internal/i18n/locales/en.toml`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

**問題点・改善提案**:

- **[@go/docs/i18n-guide.md#翻訳の命名規則]**: `diff_no_changes` の日英翻訳でスコープが非対称

  ja.toml では「変更はありません」→「本文に変更はありません」と「本文」を追加しているが、en.toml は "No changes" のまま。日本語では「本文に」というスコープが追加されているのに英語は変更されていない。

  ```toml
  # ja.toml
  other = "本文に変更はありません"

  # en.toml
  other = "No changes"
  ```

  **修正案**:

  ```toml
  # en.toml
  other = "No changes in body"
  ```

  **対応方針**:
  - [x] en.toml も「body」のスコープを追加して対称にする
  - [ ] ja.toml の「本文に」を削除して en.toml と揃える
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[@go/docs/i18n-guide.md#翻訳の命名規則]**: `suggestion_change_remove_page_confirm` の補足説明の表現スタイルが日英で異なる

  ja.toml では `\n(編集提案から削除しても下書きは残り続けます)` と括弧付きの補足を追加、en.toml では `\n(Your draft will not be deleted)` と追加。括弧の使い方は対称だが、日本語の「残り続けます」と英語の「will not be deleted」はニュアンスが微妙に異なる（残り続ける vs 削除されない）。些細な問題なので参考情報として記載。

  **修正案**: 特になし（現状でも問題ない）

  **対応方針**:
  - [ ] 現状維持
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  日本語に合わせて英語の修正をお願いします。
  ```

## 設計改善の提案

設計改善の提案はありません。

フラッシュメッセージのミドルウェア化（`FlashManager.Middleware`）は、各ハンドラーから `GetFlash(w, r)` の呼び出しを除去し、レイアウトテンプレートで `session.FlashFromContext(ctx)` を一箇所で呼ぶ設計に統一しており、コードの重複削減とアーキテクチャの一貫性向上に寄与している。

差分計算の改行コード正規化（`normalizeNewlines`）も、ブラウザからの CRLF 送信と DB 内の LF データの不一致による誤差分の問題を適切に解決しており、テストも十分にカバーしている。

新規コンポーネント（`ContentCard`、`OptionalLabel`）も再利用可能な粒度で適切に設計されている。

## 総合評価

**評価**: Comment

**総評**:

本PRは主に以下の3つの改善をまとめたものである:

1. **フラッシュメッセージのミドルウェア化**: 各ハンドラーで個別に `GetFlash(w, r)` を呼んでいたパターンをミドルウェアに集約し、レイアウトテンプレートが `context` から直接フラッシュを取得する設計に変更。レイアウトデータ構造体（`DefaultLayoutData`, `SimpleLayoutData`）から `Flash` フィールドを削除し、ハンドラーのボイラープレートを大幅に削減している。変更は一貫しており、全ハンドラーに正しく適用されている。

2. **差分計算の改行コード正規化**: `ComputeDiffBlocks` で CRLF → LF 正規化と末尾改行の統一を行うことで、ブラウザのフォーム送信によるノイズ差分を防止。テストケースも適切に追加されている。

3. **UI/翻訳の微調整**: ページタイトルのセパレータ統一（`-` → `|`）、タイトルへのトピック名追加、「任意」ラベルコンポーネントの導入、各種翻訳文言の改善。

指摘事項は翻訳の日英対称性に関する軽微な点のみであり、コードの品質は高い。
