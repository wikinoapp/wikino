# コードレビュー: handler-usecase-refactor-6-2

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-13                                     |
| 対象ブランチ               | handler-usecase-refactor-6-2                   |
| ベースブランチ             | handler-usecase-refactor-6-1                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 9 ファイル                                     |
| 変更行数（実装）           | +205 / -187 行                                 |
| 変更行数（テスト）         | +54 / -28 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ、UseCase
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page/update.go`
- [x] `go/internal/handler/draft_page_revision/handler.go`
- [ ] `go/internal/handler/draft_page_revision/update.go`
- [x] `go/internal/usecase/get_save_draft_page_data.go`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/handler/draft_page_revision/update.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase の責務
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - 認可チェック

**問題点・改善提案**:

- **認可チェック（policy）の配置場所の不一致**: `draft_page/update.go` では `GetSaveDraftPageDataUsecase` 内部に認可チェック（`topicPolicy.CanUpdatePage`）が含まれているが、`draft_page_revision/update.go` ではハンドラー内で認可チェックを行っている。

  `draft_page/update.go`:

  ```go
  // UseCase内部でpolicyチェック済み
  output, err := h.getSaveDraftPageDataUC.Execute(ctx, ...)
  if output == nil {
      http.Error(w, "Not Found", http.StatusNotFound)
      return
  }
  ```

  `draft_page_revision/update.go`:

  ```go
  // UseCase(GetPageDetailUsecase)はpolicyチェックしない → ハンドラーでチェック
  output, err := h.getPageDetailUC.Execute(ctx, ...)
  topicPolicy := policy.NewTopicPolicy(output.SpaceMember, output.TopicMember)
  if !topicPolicy.CanUpdatePage(output.Page) {
      http.Error(w, "Not Found", http.StatusNotFound)
      return
  }
  ```

  `GetPageDetailUsecase` は汎用的な UseCase（ページ詳細表示など認可が不要な場面でも使う）なので、認可チェックを内部に含めないのは妥当。一方、`GetSaveDraftPageDataUsecase` は保存専用なので認可チェックを内包するのも妥当。

  ただし、この不一致を意図的なものとして認識しているか確認したい。特に、`draft_page_revision/update.go` でも `GetSaveDraftPageDataUsecase` を使うことで統一できないか検討の余地がある。`draft_page_revision/update.go` ではフォームから `topicNumber` を受け取らない（ページの既存 TopicID を使う）ため、`GetSaveDraftPageDataUsecase` の入力パラメータ（`TopicNumber` が必須）とは合わない。その観点では現在の設計は合理的。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 現状の設計で問題なし（意図的な不一致）
  - [ ] `draft_page_revision/update.go` でも認可チェックを UseCase に移動する（別の UseCase を作成）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  認可チェックはClean Architectureなどの一般的な設計ではどの層で行いますか？
  個人的にはPresentation層で行うと良いかなと思ったので、UseCaseがPolicyに依存している箇所を修正し、
  depguardでUseCaseからPolicyへの依存を禁止するのが良いかなと思いました。
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 6-2（draft_page ハンドラーの create, update, delete の読み取り部分を UseCase 化）の実装として適切にリファクタリングされている。

**良い点**:

- `draft_page/handler.go` から `repository` パッケージへの依存が完全に除去されており、アーキテクチャガイドの「Handler から Repository への直接依存は禁止」ルールに準拠
- `draft_page_revision/handler.go` からも同様に `repository` 依存が除去されている
- 新規 UseCase `GetSaveDraftPageDataUsecase` の設計が適切。認可チェック（TopicPolicy）を内包し、ハンドラーをシンプルに保っている
- 命名規則（`{action}_{entity}.go`、`{Action}{Entity}Usecase`）に準拠
- テストが網羅的（未認証、不正パラメータ、スペース未存在、非メンバー、ページ未存在、認可拒否、トピック未存在を全てカバー）
- `main.go` でのリポジトリ変数の再利用が適切（以前は `repository.NewXxxRepository(queries)` を重複作成していた箇所をテストで修正）

**指摘事項**:

- 認可チェックの配置場所について 1 件確認事項あり（問題ではなく設計確認）
