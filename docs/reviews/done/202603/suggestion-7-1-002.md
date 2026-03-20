# コードレビュー: suggestion-7-1

## レビュー情報

| 項目                       | 内容                                     |
| -------------------------- | ---------------------------------------- |
| レビュー日                 | 2026-03-19                               |
| 対象ブランチ               | suggestion-7-1                           |
| ベースブランチ             | suggestion-6-2                           |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md         |
| 変更ファイル数             | 5 ファイル（実装 2, テスト 1, その他 2） |
| 変更行数（実装）           | +213 行                                  |
| 変更行数（テスト）         | +316 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [ ] `go/internal/usecase/apply_suggestion.go`
- [x] `go/internal/testutil/suggestion_page_builder.go`

### テストファイル

- [x] `go/internal/usecase/apply_suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-7-1-001.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/apply_suggestion.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase の WithTx パターン、トランザクション管理
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - スペース ID によるクエリスコープ
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コメント、ログ出力

**問題点・改善提案**:

- **[データ損失] `FeaturedImageAttachmentID` が未設定のためページの既存値が NULL にリセットされる**

  `pageRepo.Update()` 呼び出し時に `FeaturedImageAttachmentID` フィールドを設定していない（125 行目付近）。`UpdatePageInput.FeaturedImageAttachmentID` は `*model.AttachmentID` 型のため、未設定では nil となり、SQL の `UPDATE pages SET featured_image_attachment_id = $9` で既存の値が NULL に上書きされる。

  タスク 7-1b で `sp.FeaturedImageAttachmentID` に置き換える予定であることは理解しているが、7-1 と 7-1b の間に編集提案が反映された場合、ページのアイキャッチ画像情報が消失する。

  ```go
  // 現在のコード（118-128行目）
  _, err = pageRepo.Update(ctx, repository.UpdatePageInput{
      ID:            sp.PageID,
      SpaceID:       input.SpaceID,
      TopicID:       page.TopicID,
      Title:         sp.Title,
      Body:          sp.Body,
      BodyHTML:      sp.BodyHTML,
      LinkedPageIDs: page.LinkedPageIDs,
      ModifiedAt:    now,
      PublishedAt:   &now,
  })
  ```

  **修正案**:

  `page.FeaturedImageAttachmentID` を設定して既存値を保持する（7-1b で `sp.FeaturedImageAttachmentID` に置き換えるまでの暫定対応）。

  ```go
  _, err = pageRepo.Update(ctx, repository.UpdatePageInput{
      ID:                        sp.PageID,
      SpaceID:                   input.SpaceID,
      TopicID:                   page.TopicID,
      Title:                     sp.Title,
      Body:                      sp.Body,
      BodyHTML:                  sp.BodyHTML,
      LinkedPageIDs:             page.LinkedPageIDs,
      FeaturedImageAttachmentID: page.FeaturedImageAttachmentID,
      ModifiedAt:                now,
      PublishedAt:               &now,
  })
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `page.FeaturedImageAttachmentID` を設定する
  - [ ] フィーチャーフラグで制御されており 7-1b が近いため現状のままにする
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

編集提案反映の UseCase として、アーキテクチャガイドに沿った良い実装になっている。WithTx パターンの使用、スペース ID によるクエリスコープ、エラーメッセージの日本語化、ステータスバリデーションなどが適切に実装されている。テストも正常系・異常系を網羅しており、テストビルダーも既存パターンに従っている。

1 点、`FeaturedImageAttachmentID` の未設定によるデータ損失の可能性がある。フィーチャーフラグで制御されているため実害が出る可能性は低いが、`page.FeaturedImageAttachmentID` を暫定的に設定しておくことで安全に対応できる。
