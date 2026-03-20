# コードレビュー: suggestion-9-1

## レビュー情報

| 項目                       | 内容                                          |
| -------------------------- | --------------------------------------------- |
| レビュー日                 | 2026-03-19                                    |
| 対象ブランチ               | suggestion-9-1                                |
| ベースブランチ             | suggestion-8a-1                               |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md（タスク9-1） |
| 変更ファイル数             | 20 ファイル（自動生成2ファイルを含む）        |
| 変更行数（実装）           | 約 +700 行                                    |
| 変更行数（テスト）         | 約 +640 行                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_change/index.go`
- [ ] `go/internal/handler/suggestion_page_edit/create.go`
- [x] `go/internal/handler/suggestion_page_edit/handler.go`
- [x] `go/internal/handler/suggestion_page_edit/show.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_page_edit/show.templ`
- [ ] `go/internal/templates/path.go`
- [x] `go/internal/usecase/start_suggestion_page_edit.go`
- [x] `go/internal/viewmodel/suggestion.go`

### テストファイル

- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/handler/suggestion_page_edit/main_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・その他

- [x] `go/docs/security-guide.md`
- [x] `go/docs/testing-guide.md`
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion_page_edit/show_templ.go`（自動生成）

## ファイルごとのレビュー結果

### `go/internal/templates/path.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティ

**問題点・改善提案**:

- **URLパスセグメントに `url.QueryEscape` を使用している**: `SuggestionPageEditShowPath` でURLパスの一部に `url.QueryEscape` を使用しているが、パスセグメントには `url.PathEscape` が正しい。`QueryEscape` はスペースを `+` にエンコードするのに対し、`PathEscape` は `%20` にエンコードする。ULIDは英数字のみなので実害はないが、意味的に正しい関数を使用すべき。

  ```go
  // 問題のあるコード
  return Path(fmt.Sprintf("/s/%s/suggestions/%d/page_edits/%s", spaceIdentifier, suggestionNumber, url.QueryEscape(suggestionPageID)))
  ```

  **修正案**:

  ```go
  return Path(fmt.Sprintf("/s/%s/suggestions/%d/page_edits/%s", spaceIdentifier, suggestionNumber, url.PathEscape(suggestionPageID)))
  ```

  **対応方針**:
  - [x] `url.PathEscape` に変更する
  - [ ] 現状のまま（ULIDには影響なし）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/suggestion_page_edit/create.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **switch文にdefaultケースがない**: `output.Status` のswitch文に `default` ケースがないため、将来 `StartSuggestionPageEditStatus` に新しい値が追加された場合、レスポンスが書き込まれずにハンドラーが終了する可能性がある。

  ```go
  // 問題のあるコード（L104-113）
  switch output.Status {
  case usecase.StartSuggestionPageEditRedirect:
      editPath := string(templates.PageEditPath(string(spaceIdentifier), int32(output.PageNumber)))
      http.Redirect(w, r, editPath, http.StatusSeeOther)
  case usecase.StartSuggestionPageEditConflict:
      confirmPath := string(templates.SuggestionPageEditShowPath(string(spaceIdentifier), int32(suggestionNumber), string(suggestionPageID)))
      http.Redirect(w, r, confirmPath, http.StatusSeeOther)
  }
  ```

  **修正案**:

  ```go
  switch output.Status {
  case usecase.StartSuggestionPageEditRedirect:
      editPath := string(templates.PageEditPath(string(spaceIdentifier), int32(output.PageNumber)))
      http.Redirect(w, r, editPath, http.StatusSeeOther)
  case usecase.StartSuggestionPageEditConflict:
      confirmPath := string(templates.SuggestionPageEditShowPath(string(spaceIdentifier), int32(suggestionNumber), string(suggestionPageID)))
      http.Redirect(w, r, confirmPath, http.StatusSeeOther)
  default:
      slog.ErrorContext(ctx, "予期しないステータス", "status", output.Status)
      http.Error(w, "Internal Server Error", http.StatusInternalServerError)
  }
  ```

  **対応方針**:
  - [x] defaultケースを追加する
  - [ ] 現状のまま（現時点では2つの値しかない）
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

タスク9-1（編集提案ページ編集開始のUseCase・ハンドラー）の要件を正しく満たしている。作業計画書に記載された全ての仕様（変更差分画面の「編集する」ボタン、確認画面、通常編集→編集提案編集の切り替えフロー、既にリンク済みの場合のリダイレクト）が実装されている。

良い点:

- アーキテクチャガイドに準拠した3層構造（Handler → UseCase → Repository）が正しく実装されている
- テストパターンが適切（トランザクションを使うケースでは `GetTestDB()` を使用、早期リターンするケースでは `SetupTx` を使用）
- セキュリティ: 認証・認可チェック、space_idによるクエリスコープが適切
- I18n: 全てのユーザー向けメッセージが国際化対応済み（ja/en）
- テンプレートのデータ構造体パターンが templ-guide に準拠

指摘事項は軽微（`url.PathEscape` の使用とdefaultケースの追加）であり、修正は任意。
