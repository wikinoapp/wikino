# コードレビュー: usecase-refactoring-4-1

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-21                                      |
| 対象ブランチ               | usecase-refactoring-4-1                         |
| ベースブランチ             | develop                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md |
| 変更ファイル数             | 13 ファイル                                     |
| 変更行数（実装）           | +399 / -297 行（大部分はコード移動）            |
| 変更行数（テスト）         | +568 / -87 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ、UseCase、Handler での処理フロー
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約、コメント、ログ出力
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/page/handler.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/usecase/attachment_ref.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/get_page_publish_data.go`
- [x] `go/internal/usecase/linked_page.go`
- [x] `go/internal/usecase/publish_page.go`

### テストファイル

- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/usecase/get_page_publish_data_test.go`
- [x] `go/internal/usecase/publish_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/write-usecase-refactoring.md`
- [x] `docs/reviews/usecase-refactoring-4-1-001.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。前回レビュー（001）の指摘事項がすべて適切に対応されています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

前回レビュー（001）の指摘事項がすべて対応済みであり、作業計画書（タスク 4-1）の要件を完全に満たしている。

**前回レビューからの対応確認**:

1. **`_ = spaceMemberID` の削除**: `get_page_publish_data_test.go` の `TestGetPagePublishDataUsecase_Execute` で、`Build()` の戻り値をキャプチャしない形に修正済み
2. **`attachment_ref.go` の作成**: 添付ファイル関連の共有関数（`calculateAttachmentRefDiff`, `applyAttachmentRefChanges`, `syncAttachmentReferences`, `extractFeaturedImageAttachmentID`）を `publish_page.go` から独立ファイルに切り出し済み。`linked_page.go` と同じパターンで一貫性がある

**アーキテクチャ準拠の確認**:

- **読み取り → 検証 → 事前計算 → 書き込みの順序**: `update.go` で `getPageDetailUC`（読み取り）→ `updateValidator`（検証）→ `getPagePublishDataUC`（事前計算）→ `publishPageUC`（書き込み）の順序が明確
- **書き込み UseCase の責務限定**: `PublishPageUsecase` から `attachmentRepo` が削除され、トランザクション内の永続化処理に専念
- **共有関数の適切な分離**: `linked_page.go`（Wikiリンク関連）と `attachment_ref.go`（添付ファイル関連）に共有関数が整理され、各 UseCase ファイルが自身の責務に集中
- **テストの充実**: 新規 UseCase のテスト（3 関数）、既存テストの更新（`t.Parallel()` 追加含む）が適切に行われている
