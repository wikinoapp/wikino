# コードレビュー: suggestion-9-1

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-19                       |
| 対象ブランチ               | suggestion-9-1                   |
| ベースブランチ             | suggestion-8a-1                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 21 ファイル                      |
| 変更行数（実装）           | +784 / -35 行                    |
| 変更行数（テスト）         | +640 / -0 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_change/index.go`
- [x] `go/internal/handler/suggestion_page_edit/handler.go`
- [ ] `go/internal/handler/suggestion_page_edit/new.go`
- [x] `go/internal/handler/suggestion_page_edit/create.go`
- [x] `go/internal/usecase/start_suggestion_page_edit.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_page_edit/new.templ`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_page_edit/main_test.go`
- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-9-1-001.md`
- [x] `go/docs/testing-guide.md`
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion_page_edit/new_templ.go`（自動生成）

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_page_edit/new.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - 認証・認可
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラー

**問題点・改善提案**:

- **[@go/docs/security-guide.md#認可]**: `suggestion_page_id` クエリパラメータが現在の編集提案に属するかどうかのバリデーションが不足している。`detailOutput.SuggestionPages` から該当IDを探してタイトルを取得しているが、見つからない場合でも確認画面が表示される。悪意のあるユーザーが別の編集提案の `suggestion_page_id` を指定した場合、確認画面は表示されるが、実際のPOSTアクション（`create.go`）ではUseCaseで検証されるため実害はない。しかし、確認画面のGETでも早期に404を返す方が一貫性がある

  ```go
  // 問題のあるコード（new.go:73-82）
  var suggestionPageTitle string
  for _, sp := range detailOutput.SuggestionPages {
      if sp.ID == suggestionPageID {
          if sp.Title != nil {
              suggestionPageTitle = *sp.Title
          }
          break
      }
  }
  // ↑ 見つからない場合でも処理が続行される
  ```

  **修正案**:

  ```go
  var suggestionPageTitle string
  found := false
  for _, sp := range detailOutput.SuggestionPages {
      if sp.ID == suggestionPageID {
          if sp.Title != nil {
              suggestionPageTitle = *sp.Title
          }
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

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り、見つからない場合は404を返す
  - [ ] 現状のまま（POSTで検証するため実害なし）
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

タスク9-1（編集提案ページ編集開始のUseCase・ハンドラー）が作業計画書に沿って正確に実装されている。

**良い点**:

- 作業計画書の仕様通り、DraftPageの切り替えフロー（通常編集→編集提案編集、既存リンク済み→リダイレクト、コンフリクト→確認画面、Force→上書き）が正しく実装されている
- セキュリティ面では、認証チェック・スペースメンバー認可・オープンステータス検証・CSRFトークン・`space_id` によるクエリスコープがすべて適切に実装されている
- ハンドラーガイドの標準ファイル名（`handler.go`, `new.go`, `create.go`）とメソッド名（`New`, `Create`）に準拠
- UseCaseのWithTxパターンによるトランザクション管理が正しく実装されている
- テストの網羅性が高い（UseCase: 4テスト、Handler: 4テスト）
- i18nが日英両方で適切に対応されている
- 3層アーキテクチャの依存関係ルール（Handler → UseCase → Repository）が守られている

**指摘**: `new.go` の `suggestion_page_id` バリデーションに関する軽微な指摘が1件。POSTアクションで検証されるため実害はなく、対応は任意。
