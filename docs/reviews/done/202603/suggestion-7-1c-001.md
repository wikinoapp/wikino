# コードレビュー: suggestion-7-1c

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-19                       |
| 対象ブランチ               | suggestion-7-1c                  |
| ベースブランチ             | suggestion-7-1b                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 5 ファイル                       |
| 変更行数（実装）           | +44 / -30 行                     |
| 変更行数（テスト）         | +161 / -1 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/apply_suggestion.go`
- [x] `go/internal/usecase/create_suggestion.go`

### テストファイル

- [x] `go/internal/usecase/apply_suggestion_test.go`
- [x] `go/internal/usecase/create_suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/apply_suggestion_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **[テストの網羅性]**: テスト名「正常系: SuggestionPageのLinkedPageIDsと**FeaturedImageAttachmentID**がPageに反映される」と記載されているが、`FeaturedImageAttachmentID` の検証が含まれていない

  テストケースでは `WithLinkedPageIDs` のみ設定し、`WithFeaturedImageAttachmentID` を設定していない。また、アサーションでも `LinkedPageIDs` のみ検証し、`FeaturedImageAttachmentID` の検証が欠けている。

  対照的に、`create_suggestion_test.go` の同等テストでは両方のフィールドを設定・検証しており、一貫性がない。

  ```go
  // 現在のコード（apply_suggestion_test.go:189-198）
  testutil.NewSuggestionPageBuilderDB(t, db).
      WithSpaceID(spaceID).
      WithSuggestionID(suggestionID).
      WithPageID(pageID).
      WithPageRevisionID(pageRevisionID).
      WithTitle("提案タイトル").
      WithBody("提案本文").
      WithBodyHTML("<p>提案本文</p>").
      WithLinkedPageIDs([]model.PageID{linkedPageID}).
      Build()
  ```

  **修正案**:

  `FeaturedImageAttachmentID` のテストデータ設定とアサーションを追加する。ただし、`FeaturedImageAttachmentID` は実際のattachmentsレコードが必要な可能性があるため、テスト環境での制約を確認した上で対応する。

  ```go
  // SuggestionPageビルダーにFeaturedImageAttachmentIDを追加
  testutil.NewSuggestionPageBuilderDB(t, db).
      // ... 既存のフィールド ...
      WithLinkedPageIDs([]model.PageID{linkedPageID}).
      WithFeaturedImageAttachmentID(model.AttachmentID("some-attachment-id")).
      Build()

  // アサーションにFeaturedImageAttachmentIDの検証を追加
  if updatedPage.FeaturedImageAttachmentID == nil || *updatedPage.FeaturedImageAttachmentID != expectedAttachmentID {
      t.Errorf("Page.FeaturedImageAttachmentID = %v, want %v", updatedPage.FeaturedImageAttachmentID, expectedAttachmentID)
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] FeaturedImageAttachmentIDのテストデータ・アサーションを追加する
  - [ ] テスト名からFeaturedImageAttachmentIDの記述を削除する（LinkedPageIDsのみのテストとする）
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

タスク 7-1c の要件（linked_page_ids と featured_image_attachment_id の保存・反映・添付ファイル参照の同期）は適切に実装されている。

- `create_suggestion.go`: DraftPage から SuggestionPage へのフィールドコピーが正しく実装されている
- `apply_suggestion.go`: `page.LinkedPageIDs` → `sp.LinkedPageIDs` への変更、`syncAttachmentReferences` の追加が適切
- アーキテクチャガイドラインの WithTx パターン、UseCase の責務範囲、依存関係のルールに従っている
- TODOコメントの解消（旧: `// TODO: タスク7-1bでsp.LinkedPageIDsとsp.FeaturedImageAttachmentIDに変更する`）も適切

軽微な指摘として、`apply_suggestion_test.go` のテスト名と実際の検証内容の不一致がある（`FeaturedImageAttachmentID` の検証が欠けている）。修正は任意だが、テスト名と検証内容の一致は保守性の観点から推奨する。
