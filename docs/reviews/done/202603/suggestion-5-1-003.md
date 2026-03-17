# コードレビュー: suggestion-5-1

## レビュー情報

| 項目                       | 内容                                |
| -------------------------- | ----------------------------------- |
| レビュー日                 | 2026-03-17                          |
| 対象ブランチ               | suggestion-5-1                      |
| ベースブランチ             | suggestion-4a-2                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md    |
| 変更ファイル数             | 13 ファイル（レビュー・計画書除く） |
| 変更行数（実装）           | +342 行                             |
| 変更行数（テスト）         | +363 行                             |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/show.go`
- [ ] `go/internal/handler/suggestion_comment/create.go`
- [x] `go/internal/handler/suggestion_comment/handler.go`
- [x] `go/internal/templates/pages/suggestion/show.templ`
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

なし

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_comment/create.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#ファイル命名規則] / 既存パターンとの不一致**: `SuggestionShowPath` の引数 `suggestionNumber` の型は `int32` だが、create.go の55行目で `int32(suggestionNumber)` と手動キャストしている。`suggestionNumber` は `int64` として `strconv.ParseInt` から取得されているが、既存の suggestion/show.go ハンドラーでは `model.SuggestionNumber` 型を使用して UseCase に渡している。create.go でも UseCase への入力で `model.SuggestionNumber(suggestionNumber)` とキャストしているため、パース後すぐに `model.SuggestionNumber` 型にキャストしてそれを一貫して使うと良い。ただし、この問題は既存の show.go と同じパターンなので軽微。

  **修正案**:

  ```go
  // 現在（34行目付近）
  suggestionNumber, err := strconv.ParseInt(suggestionNumberStr, 10, 32)
  // ...
  // 55行目
  suggestionPath := string(templates.SuggestionShowPath(string(spaceIdentifier), int32(suggestionNumber)))
  // 69行目
  SuggestionNumber: model.SuggestionNumber(suggestionNumber),
  ```

  ```go
  // 修正案: パース後すぐにドメイン型にキャスト
  suggestionNumberInt, err := strconv.ParseInt(suggestionNumberStr, 10, 32)
  if err != nil {
      handler.NotFound(w, r)
      return
  }
  suggestionNumber := model.SuggestionNumber(suggestionNumberInt)
  // ...
  suggestionPath := string(templates.SuggestionShowPath(string(spaceIdentifier), int32(suggestionNumber)))
  // ...
  SuggestionNumber: suggestionNumber,
  ```

  **対応方針**:
  - [x] 修正案の通り変更する
  - [ ] 現状のまま（show.go と同じパターンのため）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

### 作業計画書との照合

作業計画書のフェーズ 5-1 タスクの内容:

> - `internal/validator/suggestion_comment.go` に `SuggestionCommentCreateValidator` を作成（本文必須、長さ制限）
> - `internal/usecase/create_suggestion_comment.go` に `CreateSuggestionCommentUsecase` を作成
> - `internal/handler/suggestion_comment/handler.go` に Handler 構造体を定義
> - `internal/handler/suggestion_comment/create.go` に `Create` メソッドを実装（POST /s/{space}/suggestions/{suggestion_number}/comments）
> - `internal/templates/pages/suggestion/` にコメントフォーム・コメント一覧の部分テンプレートを追加
> - `cmd/server/main.go` にルーティング登録
> - 翻訳ファイルにメッセージ追加

**確認結果**: すべての項目が実装されている。

| 要件                                 | 実装状況 |
| ------------------------------------ | -------- |
| Validator 作成（本文必須・長さ制限） | ✅       |
| UseCase 作成                         | ✅       |
| Handler 構造体定義                   | ✅       |
| Create メソッド実装                  | ✅       |
| コメントフォームテンプレート追加     | ✅       |
| ルーティング登録                     | ✅       |
| 翻訳ファイルメッセージ追加           | ✅       |
| スペースメンバーのみコメント可能     | ✅       |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

フェーズ 5-1 の作業計画書に記載された全項目が正しく実装されている。

良い点:

- 3層アーキテクチャに準拠した実装（Handler → UseCase → Repository の依存方向が正しい）
- バリデーターが `internal/validator/` に正しく配置されている
- セキュリティ面：CSRF トークンがフォームに含まれている、スペースメンバーの認可チェックがハンドラーで実行されている
- テストが充実している（未ログイン、バリデーションエラー、存在しない編集提案、正常系、非メンバーの5ケース）
- 翻訳が ja/en 両方で追加されている
- 既存のコードパターン（`GetSuggestionDetailUsecase` の nil チェックパターンなど）に準拠している

指摘は1件のみで、いずれも軽微。マージ可能。
