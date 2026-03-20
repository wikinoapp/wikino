# コードレビュー: usecase-refactoring-3-1

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-20                                      |
| 対象ブランチ               | usecase-refactoring-3-1                         |
| ベースブランチ             | develop                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md |
| 変更ファイル数             | 11 ファイル                                     |
| 変更行数（実装）           | +249 / -136 行                                  |
| 変更行数（テスト）         | +308 / -86 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャ、UseCase
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/create.go`
- [ ] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/usecase/create_suggestion.go`
- [x] `go/internal/usecase/get_latest_page_revisions.go`
- [x] `go/internal/usecase/get_suggestion_body_html.go`

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`
- [ ] `go/internal/usecase/create_suggestion_test.go`
- [ ] `go/internal/usecase/get_latest_page_revisions_test.go`
- [ ] `go/internal/usecase/get_suggestion_body_html_test.go`

### 設定・その他

- [ ] `docs/plans/1_doing/write-usecase-refactoring.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion/handler.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md#依存性注入のガイドライン](/workspace/go/docs/handler-guide.md) - Handler構造体のフィールド数

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#肥大化の警告]**: Handler 構造体のフィールドが10個あり、ガイドラインの「8個を超えたらリソース分割を検討」を超過している

  ```go
  type Handler struct {
      cfg                           *config.Config          // 1
      flashMgr                      *session.FlashManager   // 2
      getSuggestionListUsecase      *usecase.GetSuggestionListUsecase      // 3
      getSuggestionDetailUsecase    *usecase.GetSuggestionDetailUsecase    // 4
      getSuggestionNewUsecase       *usecase.GetSuggestionNewUsecase       // 5
      getSuggestionBodyHTMLUsecase  *usecase.GetSuggestionBodyHTMLUsecase  // 6
      getLatestPageRevisionsUsecase *usecase.GetLatestPageRevisionsUsecase // 7
      createSuggestionUsecase       *usecase.CreateSuggestionUsecase       // 8
      sidebarHelper                 *sidebar.Helper         // 9
      createValidator               *validator.SuggestionCreateValidator   // 10
  }
  ```

  ガイドラインでは「検討」であり必須ではないが、今後さらに機能追加でフィールドが増える可能性がある。`getSuggestionBodyHTMLUsecase` と `getLatestPageRevisionsUsecase` は `Create` アクション専用のため、編集提案作成を別リソースハンドラー（例: 将来的に分割が必要になった場合）に分けることも選択肢。

  **修正案**: 現時点では記録のみで、次に依存が増えた時に分割を検討する。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 現時点で分割する（suggestion と suggestion_create に分離）
  - [ ] 現状のまま（次にフィールドが追加される際に分割を検討する）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  一旦現状のままで大丈夫です。そもそもですが、Handler構造体のフィールドが増えるとどういう問題が起きるのでしょうか？
  ```

### `go/internal/usecase/create_suggestion_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - 並行テスト

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#テストのベストプラクティス]**: トップレベルテスト関数 `TestCreateSuggestionUsecase_Execute` の先頭に `t.Parallel()` がない。テストガイドでは「すべてのトップレベルテスト関数の先頭で必ず `t.Parallel()` を呼ぶ」と規定されている

  ```go
  // 現在のコード (L40)
  func TestCreateSuggestionUsecase_Execute(t *testing.T) {
      db := testutil.GetTestDB()
  ```

  **修正案**:

  ```go
  func TestCreateSuggestionUsecase_Execute(t *testing.T) {
      t.Parallel()

      db := testutil.GetTestDB()
  ```

  また、サブテスト（L51「1つの下書きページから...」、L227「複数の下書きページから...」、L300「DraftPageのsuggestion_page_idが設定される」）にも `t.Parallel()` がない。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] トップレベルと全サブテストに `t.Parallel()` を追加する
  - [ ] トップレベルのみに追加する（サブテスト間のデータ依存がある場合）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/get_latest_page_revisions_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - 並行テスト

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#テストのベストプラクティス]**: トップレベルテスト関数 `TestGetLatestPageRevisionsUsecase_Execute` の先頭に `t.Parallel()` がない

  ```go
  // 現在のコード (L13)
  func TestGetLatestPageRevisionsUsecase_Execute(t *testing.T) {
      db := testutil.GetTestDB()
  ```

  **修正案**:

  ```go
  func TestGetLatestPageRevisionsUsecase_Execute(t *testing.T) {
      t.Parallel()

      db := testutil.GetTestDB()
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `t.Parallel()` を追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/get_suggestion_body_html_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - 並行テスト

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#テストのベストプラクティス]**: トップレベルテスト関数 `TestGetSuggestionBodyHTMLUsecase_Execute` の先頭に `t.Parallel()` がない

  ```go
  // 現在のコード (L13)
  func TestGetSuggestionBodyHTMLUsecase_Execute(t *testing.T) {
      db := testutil.GetTestDB()
  ```

  **修正案**:

  ```go
  func TestGetSuggestionBodyHTMLUsecase_Execute(t *testing.T) {
      t.Parallel()

      db := testutil.GetTestDB()
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `t.Parallel()` を追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `docs/plans/1_doing/write-usecase-refactoring.md`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@CLAUDE.md](/workspace/CLAUDE.md) - 作業計画書の運用フロー

**問題点・改善提案**:

- **設計との整合性**: タスクリストの記述（L285）「ページリビジョン取得（`FindLatestByPageID`）をValidatorに移動し、結果を Input に含める」は実際の実装と異なる。実装では読み取りUseCase（`GetLatestPageRevisionsUsecase`）に移動している。設計セクション（L146）は更新済みだが、タスクリスト内の説明が古いまま残っている

  ```markdown
  // 現在のタスクリスト (L285)

  - ページリビジョン取得（`FindLatestByPageID`）をValidatorに移動し、結果を Input に含める
  ```

  **修正案**:

  ```markdown
  - ページリビジョン取得（`FindLatestByPageID`）を読み取りUseCase（`GetLatestPageRevisionsUsecase`）で事前に行い、結果を Input に含める
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] タスクリストの記述を実装に合わせて更新する
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

`create_suggestion.go` のリファクタリングは作業計画書の方針に沿って適切に実施されている。主な変更点として：

- **書き込みUseCaseの責務明確化**: `CreateSuggestionUsecase` からデータ取得（Wikiリンク解決、ページリビジョン取得）が削除され、トランザクション内の永続化処理に専念するようになった
- **読み取りUseCaseの新設**: `GetSuggestionBodyHTMLUsecase`（本文HTML生成）と `GetLatestPageRevisionsUsecase`（ページリビジョン取得）が適切に分離された
- **Handler の処理フロー**: 「読み取り → 検証 → 書き込み」パターンに従い、Handler が各 UseCase を順序正しく呼び出している
- **テストの追加**: 新設 UseCase にテストが追加されており、正常系・異常系がカバーされている

指摘事項は軽微であり、`t.Parallel()` の追加とドキュメントの更新のみ。アーキテクチャ・セキュリティ・命名規則は問題なし。
