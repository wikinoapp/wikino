# コードレビュー: suggestion-5-1

## レビュー情報

| 項目                       | 内容                                     |
| -------------------------- | ---------------------------------------- |
| レビュー日                 | 2026-03-17                               |
| 対象ブランチ               | suggestion-5-1                           |
| ベースブランチ             | suggestion-4a-2                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md         |
| 変更ファイル数             | 15 ファイル                              |
| 変更行数（実装）           | +343 行（自動生成の show_templ.go 除く） |
| 変更行数（テスト）         | +363 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/show.go`
- [ ] `go/internal/handler/suggestion_comment/create.go`
- [x] `go/internal/handler/suggestion_comment/handler.go`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/pages/suggestion/show_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/create_suggestion_comment.go`
- [x] `go/internal/validator/suggestion_comment.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [x] `go/internal/handler/suggestion_comment/main_test.go`
- [x] `go/internal/validator/suggestion_comment_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_comment/create.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **パス生成の一貫性**: 同一ファイル内で `fmt.Sprintf` と `templates.SuggestionShowPath` の 2 つの方式でパスを生成している

  56行目（バリデーションエラー時のリダイレクト先）:

  ```go
  suggestionPath := fmt.Sprintf("/s/%s/suggestions/%d", string(spaceIdentifier), suggestionNumber)
  ```

  104行目（成功時のリダイレクト先）:

  ```go
  http.Redirect(w, r, string(templates.SuggestionShowPath(string(spaceIdentifier), int32(suggestionNumber))), http.StatusSeeOther)
  ```

  同じパスを 2 箇所で異なる方法で生成しており、パスの定義が変わった場合に片方だけ更新される可能性がある。

  **修正案**:

  56行目を `templates.SuggestionShowPath` に統一する:

  ```go
  suggestionPath := string(templates.SuggestionShowPath(string(spaceIdentifier), int32(suggestionNumber)))
  ```

  **対応方針**:
  - [x] 修正案の通り `templates.SuggestionShowPath` に統一する
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

フェーズ 5-1（コメント機能）の実装として、作業計画書の要件をすべて満たしている。

**良い点**:

- 3 層アーキテクチャの依存関係ルールに準拠（Handler → UseCase → Repository）
- バリデーターが `internal/validator/` パッケージに配置されており、Handler パッケージから repository の import を排除している
- CSRF トークン、認証チェック、スペースメンバーチェックのセキュリティ対策が適切
- i18n 対応が ja/en 両方で完全に行われている
- テストカバレッジが充実（未ログイン、バリデーションエラー、404、403、正常系）
- ハンドラーのファイル名が標準の 8 種類に従っている（`handler.go`, `create.go`）
- UseCase で Markdown を HTML に変換するロジックが適切に分離されている

**指摘事項**: 1 件（軽微、パス生成の一貫性）
