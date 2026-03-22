# コードレビュー: usecase-refactoring-4-2

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-22                                      |
| 対象ブランチ               | usecase-refactoring-4-2                         |
| ベースブランチ             | develop                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md |
| 変更ファイル数             | 17 ファイル                                     |
| 変更行数（実装）           | +214 / -109 行                                  |
| 変更行数（テスト）         | +433 / -102 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page/update.go`
- [x] `go/internal/handler/draft_page_revision/handler.go`
- [x] `go/internal/handler/draft_page_revision/update.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/get_draft_page_save_data.go`
- [x] `go/internal/usecase/linked_page.go`
- [x] `go/internal/usecase/manual_save_draft_page.go`
- [ ] `go/internal/usecase/publish_page.go`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`
- [x] `go/internal/usecase/auto_save_draft_page_test.go`
- [x] `go/internal/usecase/get_draft_page_save_data_test.go`
- [x] `go/internal/usecase/manual_save_draft_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/write-usecase-refactoring.md`
- [x] `docs/reviews/usecase-refactoring-4-2-001.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/publish_page.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#Handler での処理フロー](/workspace/go/docs/architecture-guide.md) - 読み取り → 検証 → 書き込みの分離

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#Handler での処理フロー]**: `scanAndLookupWikilinks` がトランザクション内で呼ばれている

  `scanAndLookupWikilinks` は純粋な読み取り操作（`ScanWikilinks` + `FindBySpaceAndNames`）であり、`auto_save_draft_page.go` / `manual_save_draft_page.go` では `GetDraftPageSaveDataUsecase` を通じてトランザクション外で実行されるようリファクタリング済み。しかし `publish_page.go` では依然としてトランザクション内で、かつ `topicRepo.WithTx(tx)` を使って呼ばれている。

  ```go
  // publish_page.go:88-101（トランザクション内）
  topicRepo := uc.topicRepo.WithTx(tx)
  // ...
  keys, topicMapForLinks, err := scanAndLookupWikilinks(ctx, input.Body, input.CurrentTopicName, input.SpaceID, topicRepo)
  ```

  **修正案**:

  `scanAndLookupWikilinks` をトランザクション開始前に移動し、トランザクションなしの `uc.topicRepo` を使用する。

  ```go
  // トランザクション前に読み取り
  keys, topicMapForLinks, err := scanAndLookupWikilinks(ctx, input.Body, input.CurrentTopicName, input.SpaceID, uc.topicRepo)
  if err != nil {
      return nil, fmt.Errorf("wikiリンクのスキャンに失敗しました: %w", err)
  }

  tx, err := uc.db.BeginTx(ctx, nil)
  // ...
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りトランザクション前に移動する
  - [ ] 現状のまま（`publish_page.go` のリファクタリングはタスク 4-1 のスコープであり、このPRでは関数シグネチャの変更への追従のみに留める）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書のタスク 4-2 の要件通りに実装されている。主な変更点は以下の通り：

1. **新規読み取りUseCase `GetDraftPageSaveDataUsecase` の作成**: Markdownレンダリング、Wikiリンクスキャン、アイキャッチ画像抽出、添付ファイルフィルター、画像ラッピングをトランザクション外で事前実行する読み取りUseCaseとして適切に分離されている
2. **`linked_page.go` への関数分離**: `scanAndLookupWikilinks` の切り出しにより、読み取り（スキャン + トピック検索）と書き込み（ページの自動作成）の責務が明確になった
3. **書き込みUseCaseの簡素化**: `AutoSaveDraftPageUsecase` / `ManualSaveDraftPageUsecase` から `topicRepo` と `attachmentRepo` の依存が除去され、トランザクション内はリンク先ページの自動作成とDraftPageの更新に専念している
4. **Handler の処理フロー**: `draft_page/update.go` と `draft_page_revision/update.go` で「読み取りUseCase → 認可チェック → 事前計算 → 書き込みUseCase」の順序が適切に実装されている
5. **テストカバレッジ**: 新規UseCase、Wikiリンク、既存ページリンク、添付ファイル、廃棄済みページ、ページエディタ作成など幅広いケースがカバーされている

`publish_page.go` での `scanAndLookupWikilinks` のトランザクション配置について1件確認事項があるが、機能的な問題はなく、全体としてアーキテクチャガイドラインに準拠した質の高いリファクタリングである。
