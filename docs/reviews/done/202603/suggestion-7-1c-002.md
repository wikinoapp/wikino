# コードレビュー: suggestion-7-1c

## レビュー情報

| 項目                       | 内容                                          |
| -------------------------- | --------------------------------------------- |
| レビュー日                 | 2026-03-19                                    |
| 対象ブランチ               | suggestion-7-1c                               |
| ベースブランチ             | suggestion-7-1b                               |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md（タスク7-1c) |
| 変更ファイル数             | 6 ファイル                                    |
| 変更行数（実装）           | +44 / -30 行                                  |
| 変更行数（テスト）         | +169 / -1 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/create_suggestion.go`
- [x] `go/internal/usecase/apply_suggestion.go`

### テストファイル

- [x] `go/internal/usecase/create_suggestion_test.go`
- [ ] `go/internal/usecase/apply_suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-7-1c-001.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/apply_suggestion_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **テストデータの一貫性**: `create_suggestion_test.go` の新テストでは `FeaturedImageAttachmentID` にハードコードされた偽の UUID（`"01961234-5678-7abc-8def-0123456789ab"`）を使用しているのに対し、`apply_suggestion_test.go` の新テストでは `testutil.NewAttachmentBuilderDB` で実際の添付ファイルレコードを DB に作成している。

  `apply_suggestion_test.go` 側は `syncAttachmentReferences` が呼ばれるため実際のレコードが必要で正しいが、`create_suggestion_test.go` 側も将来的に FK 制約が追加された場合にテストが壊れるリスクがある。

  ただし、現時点で `suggestion_pages.featured_image_attachment_id` に FK 制約がないのであれば、このまま問題ない。

  **修正案**:

  現時点では FK 制約がないため、修正は不要。将来 FK 制約を追加する場合に `create_suggestion_test.go` のテストを更新する。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 現時点では対応不要（FK 制約追加時に対応）
  - [ ] `create_suggestion_test.go` でも `testutil.NewAttachmentBuilderDB` を使うよう修正する
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  `suggestion_pages.featured_image_attachment_id` と `draft_pages.featured_image_attachment_id` にも
  `pages.featured_image_attachment_id` 同様にFK制約を追加してください
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 7-1c の要件（編集提案作成時に `linked_page_ids` と `featured_image_attachment_id` を DraftPage から SuggestionPage にコピーし、反映時にそれらを Page に適用する）がすべて正しく実装されている。

良かった点:

- `apply_suggestion.go` での変更が最小限かつ的確。TODOコメント（`// TODO: タスク7-1bでsp.LinkedPageIDsとsp.FeaturedImageAttachmentIDに変更する`）が適切に解消されている
- `syncAttachmentReferences` の呼び出しが `publish_page.go` と同じパターンで追加されており、既存コードとの一貫性が保たれている
- テストが正常系を適切にカバーしており、`LinkedPageIDs` と `FeaturedImageAttachmentID` の両方が正しくコピー・反映されることを検証している
- アーキテクチャガイドラインに準拠（UseCase が Repository に依存、WithTx パターンの使用）
- コーディング規約に準拠（日本語コメント、slog 使用、エラーメッセージの `fmt.Errorf` ラップ）
