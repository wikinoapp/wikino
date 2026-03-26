# コードレビュー: suggestion-12-3

## レビュー情報

| 項目                       | 内容                                  |
| -------------------------- | ------------------------------------- |
| レビュー日                 | 2026-03-26                            |
| 対象ブランチ               | suggestion-12-3                       |
| ベースブランチ             | suggestion-12-2                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md      |
| 変更ファイル数             | 10 ファイル（自動生成 2 を含む）      |
| 変更行数（実装）           | +105 / -13 行（自動生成ファイル除く） |
| 変更行数（テスト）         | +0 / -0 行                            |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/suggestion/show.go`
- [x] `go/internal/templates/components/post.templ`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/viewmodel/suggestion.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/templates/components/post_templ.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion/show_templ.go`（自動生成）
- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 12-3 の要件を過不足なく実装しており、品質が高い。

**良い点**:

- **既存パターンとの一貫性**: `MainTitle` の `Actions` 使用パターン（トピック画面の `showActions` と同じ構造）、Basecoat の Dropdown Menu パターン（`topic/show.templ`、`page/edit.templ` と同じ `id` / `aria-*` 構造）を正確に踏襲している
- **Post コンポーネントの汎用設計**: `PostAction` 構造体と `DropdownID` で再利用可能な設計になっており、将来のアクション追加にも対応しやすい。`Actions` が空の場合にドロップダウンを非表示にする制御も適切
- **権限チェックの配置**: 認可チェックを Handler で実行し、テンプレートにはフラグ（`CanUpdateSuggestion`、`CanUpdateSuggestionComment`）として渡す設計が、アーキテクチャガイドの「認可チェックは Handler で実行」の原則に従っている
- **commentActions の実装**: templ コンポーネントではなく通常の Go 関数として実装し、権限チェックの結果に応じて `nil` または `PostAction` スライスを返す設計がシンプルで分かりやすい
- **ViewModel の適切な拡張**: `SuggestionCommentForList` に `Number` フィールドを追加し、ドロップダウン ID の一意性とコメント編集 URL の生成に使用している
- **パス関数**: `SuggestionEditPath`、`SuggestionCommentEditPath` が作業計画書の URL 設計と完全に一致している
- **i18n**: 翻訳キーの命名規則（`suggestion_show_edit_button`、`suggestion_comment_edit_action`）がガイドラインに準拠し、description も適切に記述されている
