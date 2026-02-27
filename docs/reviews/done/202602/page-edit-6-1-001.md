# コードレビュー: page-edit-6-1

## レビュー情報

| 項目                       | 内容                                         |
| -------------------------- | -------------------------------------------- |
| レビュー日                 | 2026-02-20                                   |
| 対象ブランチ               | page-edit-6-1                                |
| ベースブランチ             | page-edit                                    |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md |
| 変更ファイル数             | 10 ファイル                                  |
| 変更行数（実装）           | +421 / -0 行                                 |
| 変更行数（テスト）         | +342 / -0 行                                 |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/pages.sql`
- [x] `go/internal/query/pages.sql.go`
- [x] `go/internal/repository/page.go`
- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/testutil/space_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/topic_builder.go`
- [ ] `go/internal/usecase/auto_save_draft_page.go`

### テストファイル

- [x] `go/internal/usecase/auto_save_draft_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/page-edit-go-migration.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/auto_save_draft_page.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ユースケース](/workspace/go/CLAUDE.md) - Usecase命名・構造規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

**問題点・改善提案**:

- **レースコンディション**: `findOrCreateLinkedPage` 関数（252-283行目）で、`NextPageNumber` で次のページ番号を取得してから `CreateLinkedPage` でページを作成するまでの間に、別のトランザクションが同じページ番号を使用する可能性がある

  ```go
  // 問題のあるコード（252-283行目）
  func findOrCreateLinkedPage(
      ctx context.Context,
      pageRepo *repository.PageRepository,
      spaceID model.SpaceID,
      topicID model.TopicID,
      title string,
  ) (*model.Page, error) {
      page, err := pageRepo.FindByTopicAndTitle(ctx, topicID, title, spaceID)
      if err != nil {
          return nil, err
      }
      if page != nil {
          return page, nil
      }

      nextNumber, err := pageRepo.NextPageNumber(ctx, spaceID)
      if err != nil {
          return nil, fmt.Errorf("次のページ番号の取得に失敗しました: %w", err)
      }

      page, err = pageRepo.CreateLinkedPage(ctx, repository.CreateLinkedPageInput{
          SpaceID: spaceID,
          TopicID: topicID,
          Number:  nextNumber,
          Title:   title,
      })
      if err != nil {
          return nil, fmt.Errorf("リンク先ページの作成に失敗しました: %w", err)
      }

      return page, nil
  }
  ```

  `pages` テーブルには `index_pages_on_space_id_and_number` というUNIQUE制約がある。2つの並行トランザクションが同じスペースで同時にリンク先ページを作成しようとした場合、同じ `nextNumber` を取得し、一方が UNIQUE 制約違反で失敗する。

  同じファイル内の `findOrCreateDraftPage`（137-177行目）では `isUniqueViolation` によるリトライが実装されているが、`findOrCreateLinkedPage` にはリトライロジックがない。

  Rails版では `acts_as_sequenced` gem がページ番号のレースコンディションを内部的にハンドリングしている。

  **修正案**:

  `findOrCreateDraftPage` と同様のリトライパターンを適用する。

  ```go
  func findOrCreateLinkedPage(
      ctx context.Context,
      pageRepo *repository.PageRepository,
      spaceID model.SpaceID,
      topicID model.TopicID,
      title string,
  ) (*model.Page, error) {
      for i := 0; i < findOrCreateRetryLimit; i++ {
          page, err := pageRepo.FindByTopicAndTitle(ctx, topicID, title, spaceID)
          if err != nil {
              return nil, err
          }
          if page != nil {
              return page, nil
          }

          nextNumber, err := pageRepo.NextPageNumber(ctx, spaceID)
          if err != nil {
              return nil, fmt.Errorf("次のページ番号の取得に失敗しました: %w", err)
          }

          page, err = pageRepo.CreateLinkedPage(ctx, repository.CreateLinkedPageInput{
              SpaceID: spaceID,
              TopicID: topicID,
              Number:  nextNumber,
              Title:   title,
          })
          if err != nil {
              if isUniqueViolation(err) {
                  slog.WarnContext(ctx, "リンク先ページのユニーク制約違反によりリトライ",
                      "attempt", i+1, "title", title)
                  continue
              }
              return nil, fmt.Errorf("リンク先ページの作成に失敗しました: %w", err)
          }

          return page, nil
      }

      return nil, fmt.Errorf("リンク先ページの作成が%d回のリトライ後も失敗しました", findOrCreateRetryLimit)
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りリトライロジックを追加する
  - [ ] 別のアプローチで対応する（下の回答欄に記入）
  - [ ] 現時点では対応しない（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

タスク6-1（下書き自動保存ユースケースの追加）の実装として、全体的に品質の高い実装です。

**良い点**:

- UseCase のパターン（命名、WithTx、単一 Execute メソッド）が既存コードと一貫している
- `findOrCreateDraftPage` のリトライロジックが適切に実装されている
- Wikiリンク処理パイプライン（スキャン → トピック検索 → ページ自動作成 → リンク変換 → 添付ファイルフィルター → 画像ラッピング）が作業計画書の仕様通り
- テストが正常系・異常系を網羅し、`GetTestDB()` を正しく使用している
- テストビルダー（`*BuilderDB`）がUsecaseテスト用に適切に設計されている
- セキュリティ面で `space_id` によるクエリスコープが適切に実装されている
- sqlc 生成コードの `GetNextPageNumber` クエリが正しくスコープされている

**修正が必要な点**:

- `findOrCreateLinkedPage` にレースコンディションがあり、リトライロジックの追加が必要（1件）
