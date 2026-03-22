# コードレビュー: usecase-refactoring-5-3

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-22                                      |
| 対象ブランチ               | usecase-refactoring-5-3                         |
| ベースブランチ             | develop                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md |
| 変更ファイル数             | 15 ファイル                                     |
| 変更行数（実装）           | +266 / -284 行                                  |
| 変更行数（テスト）         | +95 / -401 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page/update.go`
- [x] `go/internal/handler/draft_page_revision/handler.go`
- [x] `go/internal/handler/draft_page_revision/update.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/draft_page_content.go`（新規）
- [x] `go/internal/usecase/manual_save_draft_page.go`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`
- [x] `go/internal/usecase/auto_save_draft_page_test.go`
- [x] `go/internal/usecase/manual_save_draft_page_test.go`

### 削除ファイル

- [x] `go/internal/usecase/get_draft_page_save_data.go`（削除）
- [x] `go/internal/usecase/get_draft_page_save_data_test.go`（削除）

### 設定・その他

- [x] `docs/plans/1_doing/write-usecase-refactoring.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計との整合性チェック

作業計画書タスク5-3の要件に対する整合性を確認:

| 要件                                                                                                                                                                        | 状態                                                                                                            |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| Markdownレンダリング・Wikiリンクスキャン・トピック検索・アイキャッチ画像抽出・添付ファイルフィルター・画像ラッピングを共通ヘルパーまたは各UseCaseのトランザクション前に移動 | ✅ `calculateDraftPageSaveData` として `draft_page_content.go` に実装済み                                       |
| `AutoSaveDraftPageUsecase`, `ManualSaveDraftPageUsecase` に `topicRepo`, `attachmentRepo` を追加                                                                            | ✅ 両方の構造体とコンストラクタに追加済み                                                                       |
| `AutoSaveDraftPageInput`, `ManualSaveDraftPageInput` から `BodyHTML`, `FeaturedImageAttachmentID`, `WikilinkKeys`, `TopicMap` を削除し、必要なパラメータ（`Body` 等）を追加 | ✅ Input構造体はシンプルなパラメータのみに変更済み                                                              |
| `get_draft_page_save_data.go` を削除                                                                                                                                        | ✅ 削除済み、残存参照なし                                                                                       |
| Handler（`draft_page/update.go`, `draft_page_revision/update.go`）から `GetDraftPageSaveDataUsecase` の呼び出しを削除                                                       | ✅ 両方のHandlerから削除済み                                                                                    |
| Handler（`draft_page/handler.go`, `draft_page_revision/handler.go`）から依存を削除                                                                                          | ✅ 両方のHandler構造体から削除済み                                                                              |
| `cmd/server/main.go` のUseCase構築と依存注入を更新                                                                                                                          | ✅ `getDraftPageSaveDataUC` の構築を削除し、`topicRepo`, `attachmentRepo` を書き込みUseCaseに渡すように変更済み |
| 関連テストの更新                                                                                                                                                            | ✅ UseCase・Handlerのテストが新しいコンストラクタに合わせて更新済み                                             |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク5-3の要件がすべて正確に実装されている。`GetDraftPageSaveDataUsecase` の読み取りロジックを `AutoSaveDraftPageUsecase` / `ManualSaveDraftPageUsecase` のトランザクション前処理に統合し、共通ヘルパー `calculateDraftPageSaveData` を `draft_page_content.go` に切り出す設計は、作業計画書の「採用しなかった方針」で述べられた「書き込みUseCase内であっても、トランザクション開始前であればデータ取得を行ってよい」という方針に合致している。

良かった点:

- **責務の明確化**: `saveDraftPageContent`（トランザクション内の永続化処理）と `calculateDraftPageSaveData`（トランザクション外の事前計算）の分離が明確
- **ファイル命名**: `draft_page_content.go` は複数のUseCaseで共有される下書きページ関連のヘルパーをまとめるファイルとして適切
- **削除漏れなし**: `GetDraftPageSaveDataUsecase` への参照が完全に削除されている
- **テストの更新**: UseCase・Handlerのテストが新しいコンストラクタシグネチャに合わせて適切に更新されている
- **依存関係のルール遵守**: HandlerからRepositoryへの直接依存はなく、すべてUseCaseを経由している
- **コメントのガイドライン準拠**: 日本語のコメントで意図が明確に説明されている
