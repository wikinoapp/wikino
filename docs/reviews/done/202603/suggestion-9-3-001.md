# コードレビュー: suggestion-9-3

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-20                       |
| 対象ブランチ               | suggestion-9-3                   |
| ベースブランチ             | develop                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 18 ファイル                      |
| 変更行数（実装）           | +378 / -65 行                    |
| 変更行数（テスト）         | +583 / -7 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_page/handler.go`
- [x] `go/internal/handler/suggestion_page/update.go`
- [x] `go/internal/usecase/update_suggestion_page.go`
- [x] `go/internal/validator/suggestion_page.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_page/main_test.go`
- [x] `go/internal/handler/suggestion_page/update_test.go`
- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/validator/suggestion_page_test.go`

### 設定・その他

- [x] `go/docs/architecture-guide.md`
- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/plans/1_doing/write-usecase-refactoring.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_page/update.go`: バリデーションエラーの HTTP ステータスコード

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Handler での処理フロー
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - エラーメッセージ

**問題点・改善提案**:

- **[@go/docs/security-guide.md#エラーメッセージ]**: `ErrDraftPageNotFound` と `ErrDraftPageNotLinked` の場合に 500 Internal Server Error を返しているが、これらはシステムエラーではなくバリデーションエラー（前提条件の不一致）である

  ```go
  // 現在のコード (L96-100)
  if validationResult.Err != nil {
      slog.ErrorContext(ctx, "編集提案ページ更新のバリデーションに失敗", "error", validationResult.Err)
      http.Error(w, "Internal Server Error", http.StatusInternalServerError)
      return
  }
  ```

  `ErrDraftPageNotFound`（ユーザーの下書きが存在しない）と `ErrDraftPageNotLinked`（下書きが別の編集提案にリンクされている）は、ユーザーが直接URLを入力した場合や、別タブで下書きの状態が変わった場合に発生しうる。500 は「サーバー側の問題」を示唆するため、404 や 403 のほうが適切ではないか。

  **修正案**:

  ```go
  if validationResult.Err != nil {
      if errors.Is(validationResult.Err, validator.ErrDraftPageNotFound) ||
          errors.Is(validationResult.Err, validator.ErrDraftPageNotLinked) {
          slog.WarnContext(ctx, "編集提案ページ更新の前提条件を満たしていない", "error", validationResult.Err)
          handler.NotFound(w, r)
          return
      }
      slog.ErrorContext(ctx, "編集提案ページ更新のバリデーションに失敗", "error", validationResult.Err)
      http.Error(w, "Internal Server Error", http.StatusInternalServerError)
      return
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り、ErrDraftPageNotFound / ErrDraftPageNotLinked は 404 に変更する
  - [ ] 現状のまま（500 のまま）とする（理由を回答欄に記入）
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

タスク 9-3「編集提案を更新」アクションの実装は、作業計画書の設計通りに正確に実装されている。

**良かった点**:

- Handler → Validator → Write UseCase の3ステップフローがアーキテクチャガイドのパターンに正確に従っている
- Validator が DraftPage を取得・検証し、Result に含めて返すパターンが確立されたパターンを踏襲している
- Write UseCase がトランザクション内で永続化のみに専念しており、データ取得・検証を含まない
- テストカバレッジが充実しており、認証なし・非メンバー・正常系・クローズ済みの4パターンをカバー
- 正常系テストで SuggestionPage の更新・SuggestionPageRevision の作成・DraftPage の `suggestion_page_id` がクリアされないことの3点を検証している
- `edit.templ` の `_method` hidden field の条件分岐を簡素化し、常に PATCH を使うようにした変更は、URL 設計との整合性を改善している
- `page/edit.go` の `output.DraftPage != nil && output.DraftPage.SuggestionPageID != nil` の追加チェックにより、nil pointer dereference を防止している
- アーキテクチャガイドに「Handler での処理フロー」セクションを追加し、確立されたパターンをドキュメント化した

**指摘点**: 1件（軽微。バリデーションエラー時のHTTPステータスコードの選択について確認が必要）
