# コードレビュー: usecase-refactoring-4-2

## レビュー情報

| 項目                       | 内容                                              |
| -------------------------- | ------------------------------------------------- |
| レビュー日                 | 2026-03-22                                        |
| 対象ブランチ               | usecase-refactoring-4-2                           |
| ベースブランチ             | develop                                           |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md   |
| 変更ファイル数             | 15 ファイル（ドキュメント 3 + 実装 7 + テスト 5） |
| 変更行数（実装）           | +224 / -119 行                                    |
| 変更行数（テスト）         | +433 / -102 行                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（特に「Handler での処理フロー」セクション）
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

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
- [x] `go/internal/usecase/publish_page.go`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`
- [x] `go/internal/usecase/auto_save_draft_page_test.go`
- [x] `go/internal/usecase/get_draft_page_save_data_test.go`
- [x] `go/internal/usecase/manual_save_draft_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/write-usecase-refactoring.md`
- [x] `docs/reviews/usecase-refactoring-4-2-001.md`
- [x] `docs/reviews/usecase-refactoring-4-2-002.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っています。

## 設計との整合性チェック

### 作業計画書（タスク 4-2）との整合性

作業計画書のタスク 4-2 で定義された要件と実装を照合する。

| 要件                                                                               | 実装状況                                                                        |
| ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| `markup.ScanWikilinks`、`uniqueTopicNames` をトランザクション前に移動              | ✅ `GetDraftPageSaveDataUsecase` 内で `scanAndLookupWikilinks` として実行       |
| `topicRepo.FindBySpaceAndNames` をトランザクション前に移動                         | ✅ `scanAndLookupWikilinks` 内で実行                                            |
| `findOrCreateLinkedPage`、`pageEditorRepo.FindOrCreate` はトランザクション内に残す | ✅ `saveDraftPageContent` 内（トランザクション内）に残存                        |
| 事前計算結果を `resolveAndCreateLinkedPages` に引数として渡す                      | ✅ `WikilinkKeys` と `TopicMap` を引数として渡している                          |
| Markdownレンダリングをトランザクション前に移動                                     | ✅ `GetDraftPageSaveDataUsecase` 内で `markup.RenderMarkdown` を実行            |
| 添付ファイルフィルターをトランザクション前に移動                                   | ✅ `GetDraftPageSaveDataUsecase` 内で `markup.FilterAttachments` を実行         |
| 画像ラッピングをトランザクション前に移動                                           | ✅ `GetDraftPageSaveDataUsecase` 内で `markup.WrapStandaloneImageLinks` を実行  |
| アイキャッチ画像抽出をトランザクション前に移動                                     | ✅ `GetDraftPageSaveDataUsecase` 内で `extractFeaturedImageAttachmentID` を実行 |
| `auto_save_draft_page.go` と `manual_save_draft_page.go` の Input を更新           | ✅ `BodyHTML`, `FeaturedImageAttachmentID`, `WikilinkKeys`, `TopicMap` を追加   |
| 自動保存のパフォーマンスに影響がないことを確認                                     | ✅ DB呼び出し回数は変わらず、処理がトランザクション外に移動しただけ             |
| 関連テストの更新                                                                   | ✅ 全テストが `GetDraftPageSaveDataUsecase` を事前に呼ぶパターンに更新          |

設計との乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 4-2 の要件をすべて満たした、きれいなリファクタリングです。

**良かった点**:

- **アーキテクチャガイドラインへの準拠**: 「Handler での処理フロー（読み取り → 検証 → 書き込み）」パターンに正確に従っている。読み取り UseCase（`GetDraftPageSaveDataUsecase`）がトランザクション外で事前計算を行い、書き込み UseCase は永続化に専念している
- **`scanAndLookupWikilinks` の分離**: `resolveAndCreateLinkedPages` から読み取り部分（スキャン + トピック検索）を分離し、書き込み部分（ページ自動作成）はトランザクション内に残した設計が適切
- **テストカバレッジ**: 新規 UseCase（`GetDraftPageSaveDataUsecase`）に 3 テスト、自動保存に 7 テスト（Wikiリンク・既存ページ・page_editor 作成・廃棄済みページを含む）、手動保存に 2 テストと十分なカバレッジ
- **Input の型変換パターン**: `AutoSaveDraftPageInput` → `saveDraftPageContentInput` への型変換で、公開 API と内部実装を分離しつつコードの重複を防いでいる
- **コメント**: 日本語で意図が明確に記述されており、ガイドラインに準拠
- **`t.Parallel()` の使用**: すべてのトップレベルテスト関数で `t.Parallel()` が呼ばれており、テストガイドに準拠

**注意点**:

- タスク 4-3 で `GetDraftPageSaveDataUsecase` と既存の `GetSaveDraftPageDataUsecase` の命名混同問題への対応が予定されている。現時点では問題として扱わない
