# コードレビュー: suggestion-12-4

## レビュー情報

| 項目                       | 内容                                    |
| -------------------------- | --------------------------------------- |
| レビュー日                 | 2026-03-26                              |
| 対象ブランチ               | suggestion-12-4                         |
| ベースブランチ             | suggestion-12-3                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md (12-4) |
| 変更ファイル数             | 22 ファイル                             |
| 変更行数（実装）           | +630 行（自動生成・ドキュメント除く）   |
| 変更行数（テスト）         | +541 行                                 |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/suggestions.sql`
- [x] `go/internal/handler/suggestion/edit.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/update.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/repository/suggestion.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/pages/suggestion/edit.templ`
- [ ] `go/internal/usecase/update_suggestion.go`
- [x] `go/internal/validator/suggestion.go`

### テストファイル

- [x] `go/internal/handler/suggestion/edit_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/handler/suggestion/update_test.go`
- [x] `go/internal/usecase/update_suggestion_test.go`
- [x] `go/internal/validator/suggestion_test.go`

### 自動生成ファイル

- [x] `go/internal/query/suggestions.sql.go`
- [x] `go/internal/templates/pages/suggestion/edit_templ.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/suggestion-12-4-001.md`
- [x] `go/docs/handler-guide.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/update_suggestion.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 書き込み UseCase のルール、WithTx パターン

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#Repository の WithTx パターン]**: `UpdateSuggestionUsecase` はデータベースへの永続化を行う書き込み UseCase だが、トランザクション管理（`db.BeginTx` / `WithTx` パターン）が実装されていない

  同様に単一の DB 操作のみの `CloseSuggestionUsecase` でもトランザクションが使用されている。既存パターンとの一貫性のため、`db` フィールドを追加し `BeginTx`/`WithTx` パターンを使用すべき。

  ```go
  // 現在のコード
  type UpdateSuggestionUsecase struct {
      suggestionRepo *repository.SuggestionRepository
      topicRepo      *repository.TopicRepository
      pageRepo       *repository.PageRepository
  }

  func (uc *UpdateSuggestionUsecase) Execute(ctx context.Context, input UpdateSuggestionInput) (*UpdateSuggestionOutput, error) {
      bodyHTML, err := uc.renderBodyHTML(ctx, input)
      if err != nil {
          return nil, fmt.Errorf("本文HTMLの生成に失敗しました: %w", err)
      }

      suggestion, err := uc.suggestionRepo.Update(ctx, repository.UpdateSuggestionInput{
          // ...
      })
      // ...
  }
  ```

  **修正案**:

  ```go
  type UpdateSuggestionUsecase struct {
      db             *sql.DB
      suggestionRepo *repository.SuggestionRepository
      topicRepo      *repository.TopicRepository
      pageRepo       *repository.PageRepository
  }

  func (uc *UpdateSuggestionUsecase) Execute(ctx context.Context, input UpdateSuggestionInput) (*UpdateSuggestionOutput, error) {
      // トランザクション前: データ取得・計算
      bodyHTML, err := uc.renderBodyHTML(ctx, input)
      if err != nil {
          return nil, fmt.Errorf("本文HTMLの生成に失敗しました: %w", err)
      }

      // トランザクション: 永続化のみ
      return uc.updateSuggestion(ctx, input, bodyHTML)
  }

  func (uc *UpdateSuggestionUsecase) updateSuggestion(ctx context.Context, input UpdateSuggestionInput, bodyHTML string) (*UpdateSuggestionOutput, error) {
      tx, err := uc.db.BeginTx(ctx, nil)
      if err != nil {
          return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
      }
      defer func() {
          _ = tx.Rollback()
      }()

      suggestionRepo := uc.suggestionRepo.WithTx(tx)

      suggestion, err := suggestionRepo.Update(ctx, repository.UpdateSuggestionInput{
          ID:       input.SuggestionID,
          SpaceID:  input.SpaceID,
          Title:    input.Title,
          Body:     input.Body,
          BodyHTML: bodyHTML,
      })
      if err != nil {
          return nil, fmt.Errorf("編集提案の更新に失敗しました: %w", err)
      }

      if err := tx.Commit(); err != nil {
          return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
      }

      return &UpdateSuggestionOutput{Suggestion: suggestion}, nil
  }
  ```

  `cmd/server/main.go` と `index_test.go` の `NewUpdateSuggestionUsecase` 呼び出しにも `db` を追加する必要がある。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りトランザクション管理を追加する
  - [ ] 単一操作のため現状のまま（理由を回答欄に記入）
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

タスク 12-4（編集提案本文の編集機能）の実装は作業計画書の仕様を正しく満たしている。ハンドラー、バリデーター、テンプレート、翻訳ファイルはすべて既存パターンに沿っており、セキュリティ面（CSRF トークン、space_id によるクエリスコープ、認可チェック）も問題ない。

テストは各レイヤー（Handler, UseCase, Validator）に対して正常系・異常系が網羅されており、テストガイドに準拠している。ハンドラーガイドにテスト関数配置ルールを追記したのも良い改善。

唯一の指摘は `UpdateSuggestionUsecase` のトランザクション管理の欠如で、同様の単一操作 UseCase（`CloseSuggestionUsecase`）との一貫性の観点から対応が望ましい。ただし、単一 UPDATE 操作は SQL レベルでアトミックなため、実害はない軽微な問題である。
