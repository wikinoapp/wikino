# コードレビュー: usecase-refactoring-4-2

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-21                                      |
| 対象ブランチ               | usecase-refactoring-4-2                         |
| ベースブランチ             | develop                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md |
| 変更ファイル数             | 16 ファイル                                     |
| 変更行数（実装）           | +214 / -109 行                                  |
| 変更行数（テスト）         | +413 / -102 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page/update.go`
- [x] `go/internal/handler/draft_page_revision/handler.go`
- [x] `go/internal/handler/draft_page_revision/update.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [ ] `go/internal/usecase/get_draft_page_save_data.go`
- [x] `go/internal/usecase/linked_page.go`
- [x] `go/internal/usecase/manual_save_draft_page.go`
- [x] `go/internal/usecase/publish_page.go`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`
- [ ] `go/internal/usecase/auto_save_draft_page_test.go`
- [x] `go/internal/usecase/get_draft_page_save_data_test.go`
- [ ] `go/internal/usecase/manual_save_draft_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/write-usecase-refactoring.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/get_draft_page_save_data.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase 命名規則

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#命名規則]**: 既存の `GetSaveDraftPageDataUsecase`（`get_save_draft_page_data.go`）と新規の `GetDraftPageSaveDataUsecase`（`get_draft_page_save_data.go`）のファイル名・構造体名が酷似しており、混同しやすい

  既存の UseCase は「下書き保存に必要なページ詳細データ（Space, Page, Topic, SpaceMember 等）を取得する」もの。新規の UseCase は「下書き保存のための事前計算（Markdown レンダリング、Wikilink スキャン、アイキャッチ画像抽出等）を行う」もの。責務は異なるが名前がほぼ同じ。

  **修正案**:

  新規 UseCase の名前をより具体的なものに変更する。例：
  - `PrepareDraftPageSaveUsecase` / `prepare_draft_page_save.go`（事前計算を強調）
  - `GetDraftPagePrecomputedDataUsecase` / `get_draft_page_precomputed_data.go`（事前計算データの取得を強調）

  ただし、読み取り UseCase のプレフィックスは `Get` に統一するガイドラインがあるため、`Get` プレフィックスを維持する場合は後者が適切。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 名前を変更する（具体案を回答欄に記入）
  - [ ] 現状のまま（理由を回答欄に記入）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  `GetDraftPageSaveDataUsecase` という感じで複数の取得系メソッドの呼び出しをまとめる必要も無い気がしてきました。
  ハンドラー内で永続化に必要なデータの取得を1つずつ行うというのも処理の流れが追いやすくて良い気もします。
  どう思いますか？懸念点などあれば教えてください
  ```

### `go/internal/usecase/auto_save_draft_page_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - 並行テスト

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#並行テスト]**: すべてのトップレベルテスト関数の先頭で `t.Parallel()` を呼ぶべきだが、以下のテスト関数で呼ばれていない

  該当テスト関数:
  - `TestAutoSaveDraftPageUsecase_Execute_NewDraftPage`
  - `TestAutoSaveDraftPageUsecase_Execute_ExistingDraftPage`
  - `TestAutoSaveDraftPageUsecase_Execute_EmptyBody`
  - `TestAutoSaveDraftPageUsecase_Execute_WithWikilinks`
  - `TestAutoSaveDraftPageUsecase_Execute_WikilinkExistingPage`
  - `TestAutoSaveDraftPageUsecase_Execute_WikilinkCreatesPageEditor`
  - `TestAutoSaveDraftPageUsecase_Execute_WikilinkDiscardedPage`
  - `TestUniqueTopicNames`

  テストデータにユニークな識別子（例: `auto-save-new`, `auto-save-existing` 等）を使用しているため、並行実行は安全。

  **修正案**:

  各テスト関数の先頭に `t.Parallel()` を追加する。

  ```go
  func TestAutoSaveDraftPageUsecase_Execute_NewDraftPage(t *testing.T) {
      t.Parallel()
      // ...
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 全テスト関数に `t.Parallel()` を追加する
  - [ ] 既存テストの問題のため別 PR で対応する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/manual_save_draft_page_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - 並行テスト

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#並行テスト]**: `TestManualSaveDraftPageUsecase_Execute` と `TestManualSaveDraftPageUsecase_Execute_WithoutDraftPage` で `t.Parallel()` が呼ばれていない。テストデータにユニークな識別子を使用しているため並行実行は安全。

  **修正案**:

  ```go
  func TestManualSaveDraftPageUsecase_Execute(t *testing.T) {
      t.Parallel()
      // ...
  }

  func TestManualSaveDraftPageUsecase_Execute_WithoutDraftPage(t *testing.T) {
      t.Parallel()
      // ...
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] `t.Parallel()` を追加する
  - [ ] 既存テストの問題のため別 PR で対応する
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

作業計画書タスク 4-2 の方針に正確に沿ったリファクタリングが実施されている。主な変更内容：

- **新規読み取り UseCase `GetDraftPageSaveDataUsecase`**: Markdown レンダリング、Wikilink スキャン・トピック検索、アイキャッチ画像抽出、添付ファイルフィルターをトランザクション外に正しく分離している
- **`linked_page.go` の分割**: `resolveAndCreateLinkedPages` を `scanAndLookupWikilinks`（純粋な読み取り）と `resolveAndCreateLinkedPages`（書き込み）に分離し、トランザクション保持時間の短縮に貢献
- **書き込み UseCase の依存削減**: `AutoSaveDraftPageUsecase` と `ManualSaveDraftPageUsecase` から `topicRepo` と `attachmentRepo` を除去し、責務が明確化

アーキテクチャガイドの「Handler での処理フロー（読み取り → 検証 → 書き込み）」パターンに合致しており、設計品質は高い。テストも十分なカバレッジがある。

指摘事項は `t.Parallel()` の欠如（ガイドライン違反だが軽微）と UseCase 名の類似性（可読性の懸念）の 2 点のみ。いずれも修正必須ではなく、マージ可能な状態。
