# コードレビュー: suggestion-5-1

## レビュー情報

| 項目                       | 内容                                          |
| -------------------------- | --------------------------------------------- |
| レビュー日                 | 2026-03-17                                    |
| 対象ブランチ               | suggestion-5-1                                |
| ベースブランチ             | suggestion-4a-2                               |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md              |
| 変更ファイル数             | 16 ファイル                                   |
| 変更行数（実装）           | +338 / -0 行（自動生成の show_templ.go 除く） |
| 変更行数（テスト）         | +363 / -0 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/suggestion_comment/handler.go`
- [ ] `go/internal/handler/suggestion_comment/create.go`
- [x] `go/internal/usecase/create_suggestion_comment.go`
- [x] `go/internal/validator/suggestion_comment.go`
- [x] `go/internal/handler/suggestion/show.go`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/pages/suggestion/show_templ.go`
- [x] `go/internal/templates/path.go`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [x] `go/internal/handler/suggestion_comment/main_test.go`
- [x] `go/internal/validator/suggestion_comment_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-5-1-001.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_comment/create.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

**問題点・改善提案**:

- **コードの重複**: 103行目で `SuggestionShowPath` を再度呼び出しているが、55行目で `suggestionPath` 変数に既に格納済み

  ```go
  // 問題のあるコード（103行目）
  http.Redirect(w, r, string(templates.SuggestionShowPath(string(spaceIdentifier), int32(suggestionNumber))), http.StatusSeeOther)
  ```

  **修正案**:

  ```go
  // 55行目で既に計算済みの変数を再利用
  http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
  ```

  **対応方針**:
  - [x] 修正案の通り `suggestionPath` を再利用する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

作業計画書（タスク 5-1）に記載された要件と実装を照合した結果:

| 要件                                                    | 実装状況  |
| ------------------------------------------------------- | --------- |
| `internal/validator/suggestion_comment.go` に Validator | ✅ 実装済 |
| `internal/usecase/create_suggestion_comment.go` に UC   | ✅ 実装済 |
| `internal/handler/suggestion_comment/handler.go`        | ✅ 実装済 |
| `internal/handler/suggestion_comment/create.go`         | ✅ 実装済 |
| テンプレートにコメントフォーム・コメント一覧を追加      | ✅ 実装済 |
| `cmd/server/main.go` にルーティング登録                 | ✅ 実装済 |
| 翻訳ファイル（ja.toml, en.toml）にメッセージ追加        | ✅ 実装済 |
| スペースメンバーのみコメント可能                        | ✅ 実装済 |

設計との乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

タスク 5-1（コメント機能）の実装として、Validator・UseCase・Handler・テンプレート・i18n・テストが一通り揃っており、作業計画書との整合性も問題ない。

**良い点**:

- 3層アーキテクチャの依存関係ルールを遵守している（Handler → UseCase → Repository、Validator は Application層に配置）
- セキュリティ面: CSRF トークン、スペースメンバーの認可チェック、`space_id` をリポジトリに渡すクエリスコープが適切に実装されている
- テストが主要シナリオ（未ログイン、バリデーションエラー、存在しない編集提案、正常系、非メンバー）を網羅している
- i18n メッセージが ja/en 両方に追加されており、description も記述されている
- 既存ハンドラーパターン（リダイレクトステータスコード、認証チェック方法）との一貫性が保たれている

**指摘**:

- `create.go` 103行目の `SuggestionShowPath` の重複呼び出しは軽微な問題（修正は任意）
