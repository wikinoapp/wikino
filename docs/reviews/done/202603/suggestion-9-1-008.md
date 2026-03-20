# コードレビュー: suggestion-9-1

## レビュー情報

| 項目                       | 内容                                |
| -------------------------- | ----------------------------------- |
| レビュー日                 | 2026-03-19                          |
| 対象ブランチ               | suggestion-9-1                      |
| ベースブランチ             | suggestion-8a-1                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md    |
| 変更ファイル数             | 19 ファイル（レビュー・計画書除く） |
| 変更行数（実装）           | +689 / -35 行                       |
| 変更行数（テスト）         | +640 / -0 行                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_change/index.go`
- [x] `go/internal/handler/suggestion_page_edit/handler.go`
- [ ] `go/internal/handler/suggestion_page_edit/create.go`
- [ ] `go/internal/handler/suggestion_page_edit/show.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go`
- [x] `go/internal/templates/pages/suggestion_page_edit/show.templ`
- [x] `go/internal/templates/pages/suggestion_page_edit/show_templ.go`
- [x] `go/internal/usecase/start_suggestion_page_edit.go`
- [x] `go/internal/viewmodel/suggestion.go`

### テストファイル

- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/handler/suggestion_page_edit/main_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/docs/security-guide.md`
- [x] `go/docs/testing-guide.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_page_edit/create.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **[設計との整合性]**: UseCase内で「下書きが存在し、別の編集提案ページにリンクされている場合」（`draft.SuggestionPageID != nil && *draft.SuggestionPageID != input.SuggestionPageID`）もConflictとして扱われるが、確認画面（show.templ）のメッセージは「このページには既存の下書きがあります」としか表示しない。ユーザーが別の編集提案ページの編集中である場合に、そのことを明示するメッセージがあるとUXが向上する。ただし作業計画書にはこの区別に関する記載がないため、現時点では許容範囲。

  **対応方針**:
  - [x] 確認画面のメッセージを「通常の下書き」と「別の編集提案の下書き」で区別する（将来タスク）
  - [ ] 現状のまま（初期リリースとしては十分）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/suggestion_page_edit/show.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

**問題点・改善提案**:

- **[セキュリティ/UX]**: Show ハンドラー（確認画面）がオープンステータスかどうかをチェックしていない。クローズ済みや反映済みの編集提案に対しても確認画面が表示される。確認画面の「上書きして編集を続ける」ボタンをクリックすると Create ハンドラーでステータスチェックされるため、不正な操作は防がれる。しかし、UX の観点からはオープンでない場合に確認画面を表示せずリダイレクトした方が自然。

  **修正案**:

  ```go
  // スペースメンバーチェックの後に追加
  // オープンステータスでなければ変更差分画面にリダイレクト
  if detailOutput.Suggestion.Status != model.SuggestionStatusOpen {
      changesPath := string(templates.SuggestionChangesPath(string(spaceIdentifier), int32(suggestionNumber)))
      http.Redirect(w, r, changesPath, http.StatusSeeOther)
      return
  }
  ```

  **対応方針**:
  - [ ] 修正案の通りオープンステータスチェックを追加する
  - [ ] 現状のまま（Create側で防いでいるので問題ない）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  リダイレクトではなく404が良いかと思ったのですが、どう思いますか？
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

タスク 9-1（編集提案ページ編集開始のUseCase・ハンドラー）の実装として、作業計画書の仕様に忠実に実装されている。主な評価ポイント:

**良い点**:

- 3層アーキテクチャのルールに厳密に従っている。Handler → UseCase → Repository の依存方向が正しく、Handler から Repository への直接依存はない
- セキュリティ面で適切な対策がされている: CSRF トークンの受け渡し、スペースIDによるクエリスコープ、認証・認可チェック、`url.PathEscape` によるURLパスのエスケープ
- UseCase の WithTx パターンが正しく使用されている
- テストが正常系・異常系（未ログイン、非メンバー、コンフリクト、Force上書き）を網羅しており充実している
- I18n が日英両方で適切に対応されている
- 確認画面（通常編集の下書き → 編集提案編集への切り替え）のフローが作業計画書の設計通り実装されている

**指摘事項**:

- 2件のComment（いずれも軽微なUX改善の提案で、修正は任意）
- 実装行数は689行で300行の目安を超えているが、i18n（50行）、テンプレート（91+17行）、main.goのルーティング登録（13行）を含むため、コアロジックとしては妥当な範囲
