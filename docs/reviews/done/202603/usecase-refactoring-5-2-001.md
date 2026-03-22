# コードレビュー: usecase-refactoring-5-2

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-22                                      |
| 対象ブランチ               | usecase-refactoring-5-2                         |
| ベースブランチ             | develop                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md |
| 変更ファイル数             | 9 ファイル                                      |
| 変更行数（実装）           | +74 / -129 行                                   |
| 変更行数（テスト）         | +96 / -415 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（特に「Handler での処理フロー」セクション）
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/page/handler.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/usecase/get_page_publish_data.go`（削除）
- [x] `go/internal/usecase/publish_page.go`

### テストファイル

- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/usecase/get_page_publish_data_test.go`（削除）
- [x] `go/internal/usecase/publish_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/write-usecase-refactoring.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計との整合性チェック

作業計画書のタスク **5-2** に記載された要件を確認:

- [x] `PublishPageUsecase` に `attachmentRepo` を追加 → `publish_page.go:24` で追加済み
- [x] `GetPagePublishDataUsecase` のMarkdownレンダリング・添付ファイル参照差分計算・アイキャッチ画像抽出・添付ファイルフィルター・画像ラッピングを `PublishPageUsecase` のトランザクション前に移動 → `calculatePublishData` メソッドとして `publish_page.go:226-252` に実装済み
- [x] `PublishPageInput` から `BodyHTML`, `FeaturedImageAttachmentID`, `AttachmentRefsToAdd`, `AttachmentRefsToRemove` を削除し、必要なパラメータ（`Body` 等）を追加 → `PublishPageInput` から4フィールドが削除され、`Body` が追加済み
- [x] `get_page_publish_data.go` を削除 → 削除済み
- [x] Handler（`page/update.go`）から `GetPagePublishDataUsecase` の呼び出しを削除 → 削除済み
- [x] Handler（`page/handler.go`）から依存を削除 → `getPagePublishDataUC` フィールドと `NewHandler` の引数から削除済み
- [x] `cmd/server/main.go` のUseCase構築と依存注入を更新 → `getPagePublishDataUC` の構築を削除し、`publishPageUC` に `attachmentRepo` を追加済み
- [x] 関連テストの更新 → `get_page_publish_data_test.go` を削除し、`publish_page_test.go` のテストを更新済み。`edit_test.go` の `setupHandler` から不要な `nil` 引数を削除済み

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク5-2の要件通りに `GetPagePublishDataUsecase` を `PublishPageUsecase` に統合するリファクタリングが適切に実施されている。

**良かった点**:

- `calculatePublishData` をプライベートメソッドとして切り出し、トランザクション前の事前計算であることが明確になっている
- `publishData` 構造体でパッケージ外に公開不要なデータをまとめ、カプセル化が適切
- 既存の `calculateAttachmentRefDiff`、`extractFeaturedImageAttachmentID` などの関数をそのまま再利用し、変更を最小限に抑えている
- テストが `GetPagePublishDataUsecase` 削除に伴い適切に統合・更新されており、カバレッジが維持されている
- HandlerからUseCaseへの呼び出しが1回に減り、コードがシンプルになった
- アーキテクチャガイドの「書き込みUseCase内であっても、トランザクション開始前であればデータ取得を行ってよい」という方針（フェーズ5で採用）に合致している
