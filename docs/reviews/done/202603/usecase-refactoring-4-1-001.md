# コードレビュー: usecase-refactoring-4-1

## レビュー情報

| 項目                       | 内容                                                |
| -------------------------- | --------------------------------------------------- |
| レビュー日                 | 2026-03-21                                          |
| 対象ブランチ               | usecase-refactoring-4-1                             |
| ベースブランチ             | develop                                             |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md     |
| 変更ファイル数             | 11 ファイル                                         |
| 変更行数（実装）           | +334 / -226 行（大部分はコード移動。docs 更新含む） |
| 変更行数（テスト）         | +483 / -87 行                                       |

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

## ファイルごとのレビュー結果

### `go/internal/usecase/get_page_publish_data_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストのベストプラクティス

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#テストのベストプラクティス]**: `TestGetPagePublishDataUsecase_Execute` の65行目に `_ = spaceMemberID` がある。`TestGetPagePublishDataUsecase_Execute_NoFeaturedImage`（148-151行目）では `Build()` の戻り値をキャプチャせずに正しく処理している。同じパターンに統一すべき。

  ```go
  // 問題のあるコード（65行目）
  _ = spaceMemberID
  ```

  **修正案**:

  ```go
  // spaceMemberID のキャプチャを削除し、Build() の戻り値を使用しない形にする
  testutil.NewSpaceMemberBuilderDB(t, db).
      WithSpaceID(spaceID).
      WithUserID(userID).
      Build()
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り、`_ = spaceMemberID` を削除し、Build() の戻り値をキャプチャしない形に変更する
  - [ ] 現状のまま（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/internal/usecase/publish_page.go`: 共有ヘルパー関数の分離

**ステータス**: 要確認

**現状**:

`publish_page.go` に以下の共有ヘルパー関数が配置されている：

- `calculateAttachmentRefDiff` — `get_page_publish_data.go` と `publish_page.go`（`syncAttachmentReferences` 経由）から使用
- `applyAttachmentRefChanges` — `publish_page.go` から使用
- `syncAttachmentReferences` — `apply_suggestion.go` から使用
- `extractFeaturedImageAttachmentID` — `get_page_publish_data.go` と `auto_save_draft_page.go` から使用

これらは `PublishPageUsecase` 固有の関数ではなく、複数の UseCase から呼ばれるパッケージレベルの共有関数である。

**提案**:

Wikiリンク関連の共有関数を `linked_page.go` に切り出したのと同様に、添付ファイル関連の共有関数を `attachment_ref.go`（あるいは `attachment.go`）に切り出す。

```
internal/usecase/
├── linked_page.go        # Wikiリンク関連の共有関数（今回作成済み）
├── attachment_ref.go     # 添付ファイル参照関連の共有関数（提案）
├── publish_page.go       # PublishPageUsecaseのみ
└── ...
```

**メリット**:

- `publish_page.go` がUseCase本体に専念でき、見通しが良くなる
- `linked_page.go` と同じパターンで一貫性がある
- 添付ファイル関連の関数を探すとき `attachment_ref.go` を見ればよい

**トレードオフ**:

- ファイル数が増える
- 現状でも動作に問題はなく、今後の 4-2 タスクでさらに変更が入る可能性がある

**対応方針**:

<!-- 開発者が回答を記入してください -->

- [x] 提案通り、添付ファイル関連の共有関数を `attachment_ref.go` に切り出す
- [ ] 4-2 タスク（`auto_save_draft_page.go` / `manual_save_draft_page.go` のリファクタリング）と合わせて対応する
- [ ] 現状のまま（理由を回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Approve

**総評**:

作業計画書（タスク 4-1）の要件をすべて満たしている。主な変更内容：

1. **`GetPagePublishDataUsecase` の新規作成**: Markdownレンダリング・添付ファイル参照の差分計算・アイキャッチ画像抽出・添付ファイルフィルター・画像ラッピングをトランザクション外の読み取りUseCaseに移動。命名規則（`Get` プレフィックス）にも準拠。
2. **`syncAttachmentReferences` の分離**: `calculateAttachmentRefDiff`（読み取り）と `applyAttachmentRefChanges`（書き込み）に適切に分離。
3. **`PublishPageUsecase` から `attachmentRepo` を削除**: 添付ファイルの読み取りが `GetPagePublishDataUsecase` に移動したことで、書き込みUseCaseの依存が減少。
4. **`linked_page.go` への共有関数切り出し**: 複数UseCaseで共通利用されるWikiリンク関連関数の集約。
5. **Handler での処理フロー準拠**: `update.go` で読み取りUseCase → バリデーション → 事前計算 → 書き込みUseCaseの順序が明確に実装されている。

アーキテクチャガイドの「Handler での処理フロー（読み取り → 検証 → 書き込み）」パターンに適合しており、書き込みUseCaseがトランザクション内の永続化処理に専念する設計になっている。`resolveAndCreateLinkedPages` がトランザクション内に残る判断も作業計画書の方針と一致している。テストも充実しており、新規UseCase・既存UseCaseの更新の両方に対応している。

指摘事項はテストコードの軽微なスタイル問題（1件）と、設計改善の提案（1件）のみ。
