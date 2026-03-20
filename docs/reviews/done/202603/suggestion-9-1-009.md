# コードレビュー: suggestion-9-1

## レビュー情報

| 項目                       | 内容                              |
| -------------------------- | --------------------------------- |
| レビュー日                 | 2026-03-19                        |
| 対象ブランチ               | suggestion-9-1                    |
| ベースブランチ             | suggestion-8a-1                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md  |
| 変更ファイル数             | 29 ファイル（うち生成ファイル 2） |
| 変更行数（実装）           | +696 / -35 行                     |
| 変更行数（テスト）         | +640 / -0 行                      |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_change/index.go`
- [x] `go/internal/handler/suggestion_page_edit/handler.go`
- [x] `go/internal/handler/suggestion_page_edit/show.go`
- [x] `go/internal/handler/suggestion_page_edit/create.go`
- [x] `go/internal/usecase/start_suggestion_page_edit.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_page_edit/show.templ`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_page_edit/main_test.go`
- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・その他

- [x] `go/docs/security-guide.md`
- [x] `go/docs/testing-guide.md`
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion_page_edit/show_templ.go`（自動生成）
- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-9-1-001.md`（過去レビュー）
- [x] `docs/reviews/done/202603/suggestion-9-1-002.md`（過去レビュー）
- [x] `docs/reviews/done/202603/suggestion-9-1-003.md`（過去レビュー）
- [x] `docs/reviews/done/202603/suggestion-9-1-004.md`（過去レビュー）
- [x] `docs/reviews/done/202603/suggestion-9-1-005.md`（過去レビュー）
- [x] `docs/reviews/suggestion-9-1-006.md`（過去レビュー）
- [x] `docs/reviews/suggestion-9-1-007.md`（過去レビュー）
- [x] `docs/reviews/suggestion-9-1-008.md`（過去レビュー）

## ファイルごとのレビュー結果

問題のあるファイルのみ記載します。

### `go/internal/templates/pages/suggestion_change/index.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートガイド

**問題点・改善提案**:

- **[@go/docs/templ-guide.md#コンポーネントの再利用]**: `suggestionPageDiff` 関数が `IndexData` 構造体全体を受け取っている。`suggestionPageDiff` が実際に使うのは `data.CanEditSuggestionPages`、`data.CSRFToken`、`data.Space.Identifier`、`data.Suggestion.Number` のみ。テンプレート関数が必要以上に広い依存を持つと、コンポーネントの再利用性が下がり、テストも書きにくくなる。

  ```templ
  // 現在のコード
  templ suggestionPageDiff(data IndexData, pd viewmodel.SuggestionPageDiff) {
  ```

  **修正案**:

  必要なフィールドのみを渡す専用の構造体を定義するか、個別の引数として渡す。ただし、将来的に他のフィールドも必要になる可能性があるため、現状のまま `IndexData` を渡すのも許容できる判断ではある。

  **対応方針**:
  - [x] 専用の構造体を定義して必要なフィールドのみ渡す
  - [ ] 現状のまま（IndexData全体を渡す）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 9-1（編集提案ページ編集開始のUseCase・ハンドラー）の実装として、作業計画書の要件を正しく満たしている。

**良い点**:

- **アーキテクチャの遵守**: Handler → UseCase → Repository の依存方向が正しく守られており、Handler パッケージから repository の import はない
- **セキュリティ**: 認証チェック（未ログインは /sign_in にリダイレクト）、認可チェック（スペースメンバーでなければ 403）、ステータスチェック（オープンでなければ変更差分画面にリダイレクト）が適切に実装されている。CSRF トークンのフォームへの埋め込みも正しい。`url.PathEscape` によるURLエスケープも適切
- **space_id によるクエリスコープ**: UseCase 内の `FindByID`、`FindByIDs`、`FindByPageAndMember` いずれも `SpaceID` を条件に含めており、セキュリティガイドラインに準拠
- **テストカバレッジ**: UseCase テスト 4 ケース（下書きなし、リンク済み、コンフリクト、Force上書き）、ハンドラーテスト 4 ケース（未ログイン、非メンバー、正常系、コンフリクト）と、主要なパスが網羅されている
- **テストの DB アクセスパターン**: トランザクションが必要なテスト（UseCase の書き込みテスト）は `GetTestDB()` を使用し、読み取りのみ / 早期リターンのテストは `SetupTx` を使用している。テストガイドの使い分けに合致
- **国際化**: 日本語・英語の翻訳が揃っている。メッセージ ID の命名も `suggestion_page_edit_` プレフィックスで統一されている
- **ハンドラーガイドの遵守**: 標準ファイル名 `show.go`、`create.go`、`handler.go` が使用されており、メソッド名 `Show`、`Create` も規則通り
- **ドキュメント更新**: セキュリティガイドとテストガイドの改善が同時に行われており、CSRF ミドルウェアの責務の明確化は有用
