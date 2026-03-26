# コードレビュー: suggestion-12-4

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-26                       |
| 対象ブランチ               | suggestion-12-4                  |
| ベースブランチ             | suggestion-12-3                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 19 ファイル                      |
| 変更行数（実装）           | +597 / -3 行（自動生成除く）     |
| 変更行数（テスト）         | +522 / -0 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/suggestion/edit.go`
- [x] `go/internal/handler/suggestion/update.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/usecase/update_suggestion.go`
- [x] `go/internal/validator/suggestion.go`
- [x] `go/internal/repository/suggestion.go`
- [x] `go/db/queries/suggestions.sql`
- [x] `go/internal/query/suggestions.sql.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion/edit.templ`
- [x] `go/internal/templates/pages/suggestion/edit_templ.go`（自動生成）
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/handler/suggestion/edit_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/usecase/update_suggestion_test.go`
- [x] `go/internal/validator/suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion/edit_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md#テストファイル](/workspace/go/docs/handler-guide.md) - テストファイルの命名規則

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#テストファイル]**: `TestUpdate_*` テストが `edit_test.go` に配置されている

  ハンドラーガイドでは個別テストは `{action}_test.go` の命名を推奨しており、ガイドの例でも `password/` ディレクトリでは `edit.go` と `update.go` のテストが `handler_test.go` に統合されている。`TestUpdate_*` が `edit_test.go` に含まれているのは命名規則に合わない。

  **修正案**:

  案A: `TestUpdate_*` テスト関数を `edit_test.go` から `update_test.go` に移動する（`newSuggestionRequest` ヘルパーは共有のためファイル分割が必要）

  案B: `edit_test.go` を `handler_test.go` にリネームし、Edit と Update のテストをまとめる

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 案A: `update_test.go` に分離する
  - [ ] 案B: `handler_test.go` にリネームして統合する
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

タスク12-4（編集提案本文の編集）の実装が作業計画書に記載されたすべての要件を満たしています。具体的には:

- **Validator**: `SuggestionUpdateValidator` がタイトル必須・200文字制限、本文10000文字制限を正しく実装。`SuggestionCreateValidator` と定数（`suggestionTitleMaxLength`, `suggestionBodyMaxLength`）を適切に共有
- **UseCase**: `UpdateSuggestionUsecase` がMarkdown→HTML変換とWikiリンク解決を含む本文処理を正しく実装。トランザクション不要の単純更新であるため `db.BeginTx` なしで適切
- **Repository**: `Update` メソッドが `space_id` でスコープしたUPDATEクエリを使用しており、セキュリティガイドラインに準拠
- **Handler**: `edit.go`（GET）と `update.go`（PATCH）が標準のファイル命名に従い、認証→データ取得→認可→バリデーション→書き込みの処理フローを適切に実装
- **Template**: 構造体ベースの引数パターン（`EditData`）を採用し、templ-guideに準拠。CSRFトークン、Method Override（`_method=PATCH`）、フォームエラー表示が正しく実装
- **I18n**: ja.toml/en.toml両方に翻訳を追加し、descriptionも記述
- **テスト**: Handler（認証・404・正常表示・バリデーションエラー・正常更新）、UseCase（タイトル/本文更新・空本文・Markdown変換）、Validator（全パターン）を網羅

アーキテクチャの依存方向（Handler→UseCase→Repository）、命名規則、認可チェック（Policy使用）、セキュリティ（CSRF・space_idスコープ）すべて問題なし。既存の `create.go` / `SuggestionCreateValidator` と一貫したパターンで実装されています。
