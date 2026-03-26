# コードレビュー: suggestion-12-2

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-26                             |
| 対象ブランチ               | suggestion-12-2                        |
| ベースブランチ             | suggestion-12-1                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 12 ファイル                            |
| 変更行数（実装）           | +102 / -9 行（自動生成・スキーマ除く） |
| 変更行数（テスト）         | +68 / -0 行                            |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（UseCase、Repository）
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260326051852_add_number_to_suggestion_comments.sql`
- [x] `go/db/queries/suggestion_comments.sql`
- [x] `go/internal/model/id.go`
- [x] `go/internal/model/suggestion_comment.go`
- [x] `go/internal/repository/suggestion_comment.go`
- [ ] `go/internal/usecase/create_suggestion_comment.go`
- [x] `go/internal/testutil/suggestion_comment_builder.go`

### テストファイル

- [x] `go/internal/repository/suggestion_comment_test.go`

### 設定・その他（自動生成含む）

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `go/db/schema.sql`
- [x] `go/internal/query/models.go`
- [x] `go/internal/query/suggestion_comments.sql.go`

## ファイルごとのレビュー結果

### `go/internal/usecase/create_suggestion_comment.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#ユースケース](/workspace/go/docs/architecture-guide.md) - UseCase の書き込みパターン、WithTx パターン

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#handler-での処理フロー読み取り--検証--書き込み]**: `GetNextNumber` + `Create` がトランザクション外で実行されている

  `create_suggestion.go`（既存の編集提案作成UseCase）では、`GetNextNumber` → `Create` がトランザクション内で実行されている（`BeginTx` → `WithTx` → `GetNextNumber` → `Create` → `Commit`）。一方、`create_suggestion_comment.go` では `GetNextNumber` と `Create` が個別のDB操作として実行されており、トランザクションで保護されていない。

  ユニークインデックス `[suggestion_id, number]` がデータ整合性を保護するため、重複番号がINSERTされることはないが、同時にコメントが作成された場合にユニーク制約違反でサーバーエラーになる可能性がある。

  ```go
  // 現在のコード（トランザクションなし）
  func (uc *CreateSuggestionCommentUsecase) Execute(ctx context.Context, input CreateSuggestionCommentInput) (*CreateSuggestionCommentOutput, error) {
      nextNumber, err := uc.suggestionCommentRepo.GetNextNumber(ctx, input.SuggestionID)
      // ...
      comment, err := uc.suggestionCommentRepo.Create(ctx, repository.CreateSuggestionCommentInput{
          // ...
          Number: nextNumber,
      })
  ```

  **修正案**:

  `create_suggestion.go` と同じパターンで、`db *sql.DB` を依存に追加し、`BeginTx` → `WithTx` → `GetNextNumber` → `Create` → `Commit` のトランザクションパターンを適用する。

  ```go
  type CreateSuggestionCommentUsecase struct {
      db                    *sql.DB
      suggestionCommentRepo *repository.SuggestionCommentRepository
  }

  func (uc *CreateSuggestionCommentUsecase) Execute(ctx context.Context, input CreateSuggestionCommentInput) (*CreateSuggestionCommentOutput, error) {
      bodyHTML := markup.RenderMarkdown(input.Body)

      return uc.createComment(ctx, input, bodyHTML)
  }

  func (uc *CreateSuggestionCommentUsecase) createComment(ctx context.Context, input CreateSuggestionCommentInput, bodyHTML string) (*CreateSuggestionCommentOutput, error) {
      tx, err := uc.db.BeginTx(ctx, nil)
      if err != nil {
          return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
      }
      defer func() {
          _ = tx.Rollback()
      }()

      suggestionCommentRepo := uc.suggestionCommentRepo.WithTx(tx)

      nextNumber, err := suggestionCommentRepo.GetNextNumber(ctx, input.SuggestionID)
      // ...
      comment, err := suggestionCommentRepo.Create(ctx, repository.CreateSuggestionCommentInput{...})
      // ...

      if err := tx.Commit(); err != nil {
          return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
      }

      return &CreateSuggestionCommentOutput{Comment: comment}, nil
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りトランザクションを追加する
  - [ ] 現状のまま（ユニークインデックスによる保護で十分と判断）
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

タスク12-2（`suggestion_comments`への`number`カラム追加）の実装として、作業計画書の仕様通りに正しく実装されている。

- マイグレーション: 既存データの`ROW_NUMBER()`によるバックフィル、`DEFAULT`の一時設定と削除、ユニークインデックスの作成、ロールバック対応すべて適切
- モデル: `SuggestionCommentNumber`ドメインID型が`SuggestionNumber`と同じパターンで追加されている
- リポジトリ: `GetNextNumber`メソッドと`Create`の`Number`パラメータ追加が既存パターンに沿っている
- テスト: `GetNextNumber`のテスト（0件・1件以上の2ケース）と`Create`テストの`Number`検証が追加されている
- テストビルダー: `nextNumber`の自動計算が追加され、テストデータ作成が簡潔に保たれている

唯一の指摘は、`create_suggestion_comment.go`のトランザクション不使用が`create_suggestion.go`のパターンと異なる点。ユニークインデックスでデータ整合性は保護されているため必須ではないが、既存パターンとの一貫性の観点で検討に値する。
