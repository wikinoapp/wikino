# コードレビュー: suggestion-9-1

## レビュー情報

| 項目                       | 内容                                     |
| -------------------------- | ---------------------------------------- |
| レビュー日                 | 2026-03-19                               |
| 対象ブランチ               | suggestion-9-1                           |
| ベースブランチ             | suggestion-8a-1                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md         |
| 変更ファイル数             | 24 ファイル（自動生成・レビュー除く 17） |
| 変更行数（実装）           | 約 +585 行                               |
| 変更行数（テスト）         | 約 +640 行                               |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_change/index.go`
- [ ] `go/internal/handler/suggestion_page_edit/create.go`
- [x] `go/internal/handler/suggestion_page_edit/handler.go`
- [x] `go/internal/handler/suggestion_page_edit/new.go`
- [x] `go/internal/usecase/start_suggestion_page_edit.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [x] `go/internal/templates/page_name.go`
- [ ] `go/internal/templates/path.go`
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion_page_edit/new.templ`
- [x] `go/internal/templates/pages/suggestion_page_edit/new_templ.go`（自動生成）

### テストファイル

- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/handler/suggestion_page_edit/main_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/docs/security-guide.md`
- [x] `go/docs/testing-guide.md`
- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-9-1-001.md`
- [x] `docs/reviews/done/202603/suggestion-9-1-002.md`
- [x] `docs/reviews/done/202603/suggestion-9-1-003.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_page_edit/create.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - 認可チェック
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **[@go/docs/security-guide.md#認可]**: `suggestionPageID` が現在の編集提案に属していることの検証が欠落している

  `new.go` では `detailOutput.SuggestionPages` を走査して `suggestionPageID` の存在を確認しているが、`create.go` ではこの検証がない。攻撃者が別の編集提案の `suggestionPageID` を送信した場合、UseCase はスペース ID のみで検証するため、誤った編集提案ページにリンクされた下書きが作成される可能性がある。

  ```go
  // create.go の現在のコード（78行目付近）
  // suggestionPageIDが現在のsuggestionに属することの検証がない
  output, err := h.startSuggestionPageEditUsecase.Execute(ctx, usecase.StartSuggestionPageEditInput{
      SpaceID:          detailOutput.Space.ID,
      SpaceMemberID:    detailOutput.SpaceMember.ID,
      SuggestionPageID: suggestionPageID,
      Force:            force,
  })
  ```

  **修正案**:

  `new.go` と同様に、UseCase 呼び出し前に `suggestionPageID` が現在の編集提案に属していることを検証する。

  ```go
  // オープンステータスチェックの後に追加
  found := false
  for _, sp := range detailOutput.SuggestionPages {
      if sp.ID == suggestionPageID {
          found = true
          break
      }
  }
  if !found {
      handler.NotFound(w, r)
      return
  }
  ```

  **対応方針**:
  - [x] 修正案の通り、`detailOutput.SuggestionPages` を走査して検証を追加する
  - [ ] UseCase 側で `suggestionPageID` と `suggestionID` の一致を検証する方式に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/path.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - XSS 対策・入力バリデーション

**問題点・改善提案**:

- **[@go/docs/security-guide.md#ユーザー入力の扱い]**: `SuggestionPageEditNewPath` でクエリパラメータを URL エンコードしていない

  ```go
  // 現在のコード
  func SuggestionPageEditNewPath(spaceIdentifier string, suggestionNumber int32, suggestionPageID string) Path {
      return Path(fmt.Sprintf("/s/%s/suggestions/%d/page_edits/new?suggestion_page_id=%s", spaceIdentifier, suggestionNumber, suggestionPageID))
  }
  ```

  `suggestionPageID` は ULID（英数字のみ）であるため実際には安全だが、他のパスヘルパー関数とは異なりクエリパラメータを含む唯一の関数であり、`url.QueryEscape()` を適用する方がロバストである。

  **修正案**:

  ```go
  import "net/url"

  func SuggestionPageEditNewPath(spaceIdentifier string, suggestionNumber int32, suggestionPageID string) Path {
      return Path(fmt.Sprintf("/s/%s/suggestions/%d/page_edits/new?suggestion_page_id=%s", spaceIdentifier, suggestionNumber, url.QueryEscape(suggestionPageID)))
  }
  ```

  **対応方針**:
  - [x] 修正案の通り `url.QueryEscape()` を適用する
  - [ ] ULID は英数字のみであるため現状のままとする
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

タスク 9-1（編集提案ページ編集開始の UseCase・ハンドラー）の実装。作業計画書に記述された以下のフローが正しく実装されている:

- 変更差分画面の各ページに「編集する」ボタンを配置
- 既存の通常編集 DraftPage がある場合の確認画面（`new.go`）
- DraftPage の `suggestion_page_id` 設定と SuggestionPage の内容による初期化（`start_suggestion_page_edit.go`）
- 編集提案作成者の場合は `suggestion_page_id` 設定済みのためリダイレクト

アーキテクチャ（Handler → UseCase → Repository）、セキュリティ（CSRF トークン、認証・スペースメンバーチェック、space_id によるクエリスコープ）、テスト（正常系・異常系の網羅）、国際化（全メッセージ i18n 対応）はガイドラインに準拠している。

`create.go` における `suggestionPageID` の所属検証欠落は修正が必要。`path.go` の URL エンコード不足は軽微。
