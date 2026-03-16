# コードレビュー: handler-usecase-refactor-5-1

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-12                                     |
| 対象ブランチ               | handler-usecase-refactor-5-1                   |
| ベースブランチ             | handler-usecase-refactor-4-1                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 7 ファイル                                     |
| 変更行数（実装）           | +320 / -269 行                                 |
| 変更行数（テスト）         | +14 / -6 行                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/get_page_detail.go`
- [x] `go/internal/handler/page/handler.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/handler/page/edit_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/get_page_detail.go`

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase の設計パターン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#ユースケース]**: `FetchEditLinkData` メソッドが `GetPageDetailUsecase` のメソッドとして定義されているが、`Execute` メソッド以外のパブリックメソッドを UseCase に持たせるパターンは既存の UseCase に存在しない。`GetTopicDetailUsecase` 等は `Execute` メソッドのみで構成されている。

  `FetchEditLinkData` は「編集画面のリンクデータ取得」という独立した責務を持っており、`GetPageDetailUsecase` の `Execute` とは異なるデータセットを返す。このメソッドを `GetPageDetailUsecase` に配置する設計判断が妥当かどうか確認が必要。

  **修正案**:

  以下の 2 つの案が考えられる：
  - **案 A**: 現状維持（`GetPageDetailUsecase` のメソッドとして維持）。ページ詳細に関連するデータ取得を 1 つの UseCase に集約する方針として合理的。
  - **案 B**: `FetchEditLinkDataUsecase` として独立した UseCase に分離する。UseCase は 1 つの `Execute` メソッドのみを持つという一貫性を保てる。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 案 A: 現状維持（`GetPageDetailUsecase` のメソッドとして維持）
  - [ ] 案 B: `FetchEditLinkDataUsecase` として独立した UseCase に分離
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  案Bが良いですが、`Fetch` ではなく `Get` プレフィックスを付けて定義したいです
  ```

- **[コード重複]**: `buildExcludePageIDs` と `collectTopicIDsFromPages` が UseCase にプライベート関数として定義されているが、同一ロジックの関数が `viewmodel/edit_link_data.go` にも `BuildExcludePageIDs` / `CollectTopicIDsFromPages` としてエクスポートされた状態で残っている。現在 `viewmodel` 側の関数は `draft_page/show.go` から呼び出されている。

  今回のタスク（5-1）のスコープでは `draft_page` ハンドラーは対象外のため、`viewmodel` 側を削除すると他のハンドラーが壊れる。ただし、将来的に `draft_page` も UseCase 化される際（フェーズ 6）にこの重複を解消する必要がある。

  **修正案**:

  現時点では対応不要。フェーズ 6 で `draft_page` ハンドラーを UseCase 化する際に、`viewmodel` 側の重複関数を削除する。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] フェーズ 6 で解消する（現時点では対応不要）
  - [x] 今回の PR で `viewmodel` 側の関数を削除し、`draft_page/show.go` も UseCase 経由に変更する
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

作業計画書タスク 5-1 の要件通りに `GetPageDetailUsecase` が作成され、`page/handler.go` から Repository フィールドが削除されて UseCase 経由に統一されている。アーキテクチャの依存方向（Handler → UseCase → Repository）が正しく守られており、Handler パッケージ内の非テストファイルから `repository` パッケージへの import が完全に排除されている。

**良い点**:

- Handler が大幅にシンプルになり、HTTP 処理に専念する形に改善された
- `buildEditLinkResult` 関数の分離により、UseCase の出力から ViewModel への変換が明確になった
- テストコードが UseCase の初期化パターンに正しく更新されている
- 既存の `GetTopicDetailUsecase` と一貫したパターンで実装されている

**確認事項**:

1. `FetchEditLinkData` メソッドの配置場所（UseCase に複数のパブリックメソッドを持たせるパターンの是非）
2. `viewmodel` パッケージとの関数重複（フェーズ 6 で解消予定であれば許容）

**作業計画書との整合性**:

- 作業計画書に `page/show.go を UseCase 経由に修正` と記載があるが、`page/show.go` は存在しない。`page` ハンドラーは `edit.go` と `update.go` のみで構成されており、両方とも UseCase 経由に修正されている。作業計画書の記載が実際のファイル構成と異なっていた可能性がある。
