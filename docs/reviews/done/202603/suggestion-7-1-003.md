# コードレビュー: suggestion-7-1

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-19                                     |
| 対象ブランチ               | suggestion-7-1                                 |
| ベースブランチ             | suggestion-6-2                                 |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md               |
| 変更ファイル数             | 4 ファイル（実装 2, テスト 1, ドキュメント 1） |
| 変更行数（実装）           | +214 / -0 行                                   |
| 変更行数（テスト）         | +316 / -0 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（UseCase の WithTx パターン）
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/apply_suggestion.go`
- [x] `go/internal/testutil/suggestion_page_builder.go`

### テストファイル

- [ ] `go/internal/usecase/apply_suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/apply_suggestion_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略、テストヘルパー、テーブル駆動テスト
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase のテストパターン

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#テストのベストプラクティス] 「反映済み」ステータスの編集提案に対する異常系テストが不足している**

  「下書き」と「クローズ」ステータスの異常系テストは存在するが（225行目、260行目）、「反映済み」（`SuggestionStatusApplied`）ステータスの異常系テストがない。既に反映済みの編集提案を再度反映しようとした場合にエラーになることを検証すべき。

  **修正案**:

  ```go
  t.Run("異常系: 反映済みの編集提案は再度反映できない", func(t *testing.T) {
      spaceID := testutil.NewSpaceBuilderDB(t, db).
          WithIdentifier("apply-suggestion-6").
          Build()
      userID := testutil.NewUserBuilderDB(t, db).
          WithEmail("apply-suggestion-6@example.com").
          WithAtname("applysuggestion6").
          Build()
      spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
          WithSpaceID(spaceID).
          WithUserID(userID).
          Build()
      topicID := testutil.NewTopicBuilderDB(t, db).
          WithSpaceID(spaceID).
          WithName("General").
          Build()

      suggestionID := testutil.NewSuggestionBuilderDB(t, db).
          WithSpaceID(spaceID).
          WithTopicID(topicID).
          WithCreatedSpaceMemberID(spaceMemberID).
          WithStatus(model.SuggestionStatusApplied).
          Build()

      _, err := uc.Execute(context.Background(), ApplySuggestionInput{
          SuggestionID:  suggestionID,
          SpaceID:       spaceID,
          SpaceMemberID: spaceMemberID,
      })
      if err == nil {
          t.Error("expected error but got nil")
      }
  })
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りテストを追加する
  - [ ] 現状のまま（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[@go/docs/testing-guide.md#テストのベストプラクティス] 複数ページ正常系テストでページ内容の更新を検証していない**

  「正常系: 複数ページの編集提案を反映できる」テスト（131行目）では、反映後に `FindByIDs` で取得したページの件数のみを検証しており（220行目）、各ページの `Title`、`Body`、`BodyHTML` が正しく更新されたかを検証していない。複数ページ反映の本質は「すべてのページの内容が正しく反映される」ことであるため、内容の検証が必要。

  **修正案**:

  ```go
  // 両方のページが更新されていることを確認
  updatedPages, err := pageRepo.FindByIDs(context.Background(), []model.PageID{page1ID, page2ID}, spaceID)
  if err != nil {
      t.Fatalf("FindByIDs() error = %v", err)
  }
  if len(updatedPages) != 2 {
      t.Fatalf("updated pages count = %d, want 2", len(updatedPages))
  }

  // 各ページの内容が反映されていることを確認
  pageMap := make(map[model.PageID]*model.Page, len(updatedPages))
  for _, p := range updatedPages {
      pageMap[p.ID] = p
  }
  if pageMap[page1ID].Body != "提案本文1" {
      t.Errorf("Page1.Body = %q, want %q", pageMap[page1ID].Body, "提案本文1")
  }
  if pageMap[page2ID].Body != "提案本文2" {
      t.Errorf("Page2.Body = %q, want %q", pageMap[page2ID].Body, "提案本文2")
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り内容の検証を追加する
  - [ ] 現状のまま（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[@go/docs/testing-guide.md#テストのベストプラクティス] 単一ページ正常系テストでタイトル更新を検証していない**

  「正常系: 1つのページの編集提案を反映できる」テスト（26行目）では、ページ更新後に `Body` の検証（114行目）は行っているが、`Title` の検証が欠落している。SuggestionPage のタイトルがポインタ型（`*string`）であり nil の場合の挙動も重要なため、タイトル更新の検証を追加すべき。

  **修正案**:

  ```go
  // ページが更新されていることを確認（既存のBody検証の後に追加）
  if updatedPage.Body != "テスト本文" {
      t.Errorf("Page.Body = %q, want %q", updatedPage.Body, "テスト本文")
  }
  // タイトルの更新を検証
  wantTitle := "テスト提案ページ" // SuggestionPageBuilderDB のデフォルト値
  if updatedPage.Title == nil || *updatedPage.Title != wantTitle {
      t.Errorf("Page.Title = %v, want %q", updatedPage.Title, wantTitle)
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りタイトル検証を追加する
  - [ ] 現状のまま（理由を回答欄に記入）
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

前回レビュー（001, 002）で指摘された `FeaturedImageAttachmentID` の未設定問題が修正されており、`page.FeaturedImageAttachmentID` を暫定的に設定する対応がなされている（125行目）。作業計画書も 7-1a, 7-1b のタスクが追加され、`linked_page_ids` と `featured_image_attachment_id` の write-time 計算の設計方針が明確化されている。

実装コードについては、WithTx パターン、スペース ID によるクエリスコープ、エラーハンドリング、日本語コメント、命名規則（`ApplySuggestionUsecase`、`apply_suggestion.go`）のすべてがガイドラインに適合している。`LinkedPageIDs` を `page.LinkedPageIDs` から暫定的に取得している点は、7-1b で `sp.LinkedPageIDs` に置き換える計画が明示されているため問題ない。テストビルダーへの `WithTitle`/`WithBody`/`WithBodyHTML` メソッド追加も既存パターンに沿っている。

指摘はすべてテスト品質に関するもの（反映済みステータスの異常系テスト追加、複数ページテストでの内容検証、タイトル更新の検証）であり、いずれも軽微である。実装コードの修正は不要。
