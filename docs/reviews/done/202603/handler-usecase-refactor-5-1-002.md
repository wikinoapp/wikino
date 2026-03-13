# コードレビュー: handler-usecase-refactor-5-1

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-12                                     |
| 対象ブランチ               | handler-usecase-refactor-5-1                   |
| ベースブランチ             | handler-usecase-refactor-4-1                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 13 ファイル                                    |
| 変更行数（実装）           | +390 / -372 行                                 |
| 変更行数（テスト）         | +21 / -8 行                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/get_page_detail.go`
- [x] `go/internal/usecase/get_edit_link_data.go`
- [x] `go/internal/handler/page/handler.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [ ] `go/internal/handler/draft_page/show.go`
- [x] `go/internal/viewmodel/edit_link_data.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/handler/draft_page/update_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`
- [x] `docs/reviews/done/202603/handler-usecase-refactor-5-1-001.md`

## ファイルごとのレビュー結果

### `go/internal/handler/draft_page/show.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **[コード重複]**: `draft_page/show.go` の 122〜141 行目で `GetEditLinkDataOutput` → `viewmodel.BuildEditLinkDataInput` への変換処理が、`page/edit.go` の `buildEditLinkResult` 関数と重複している。`page/edit.go` では `buildEditLinkResult` 関数として抽出されているが、`draft_page/show.go` ではインライン展開されたまま。

  `draft_page/show.go` はフェーズ 6 の対象であり、今回のスコープでは `getEditLinkDataUC` の導入のみが変更目的のため、完全な統一はフェーズ 6 で行うのが妥当。ただし、同じ変換ロジックが 2 箇所に存在する点は認識しておく必要がある。

  **修正案**:

  以下の 2 つの案が考えられる：
  - **案 A**: 現状維持。フェーズ 6 で `draft_page` ハンドラーの完全な UseCase 化を行う際に統一する。
  - **案 B**: `buildEditLinkResult` 関数を共通パッケージ（例: `viewmodel` パッケージ）に移動し、`draft_page/show.go` からも呼び出す。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 案 A: 現状維持（フェーズ 6 で解消）
  - [ ] 案 B: 共通関数に抽出して今回の PR で統一する
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

前回レビュー（001）で指摘された 2 点が適切に対応されている：

1. **`GetEditLinkDataUsecase` の独立化**: `FetchEditLinkData` メソッドが `GetPageDetailUsecase` のメソッドから独立した `GetEditLinkDataUsecase` として分離され、UseCase は 1 つの `Execute` メソッドのみを持つという一貫性が保たれている。プレフィックスも回答通り `Get` が使用されている。
2. **`viewmodel` パッケージの重複関数の削除**: `BuildExcludePageIDs` と `CollectTopicIDsFromPages` が `viewmodel/edit_link_data.go` から削除され、UseCase のプライベート関数 `buildExcludePageIDs` / `collectTopicIDsFromPages` に移動されている。`draft_page/show.go` も `getEditLinkDataUC` 経由に変更されているため、重複が解消されている。

**良い点**:

- `page/handler.go` から `repository` パッケージへの import が完全に排除されている
- `GetEditLinkDataUsecase` の Input/Output 型設計が適切で、`EditLinkBacklinks` 型により UseCase 層と Presentation 層のデータ構造が分離されている
- `page/edit.go` の `buildEditLinkResult` 関数が UseCase 出力 → ViewModel への変換を明確に担当しており、関心の分離が適切
- `GetPageDetailUsecase` は `GetTopicDetailUsecase` と一貫したパターンで実装されている

**軽微な指摘**:

- `draft_page/show.go` で `GetEditLinkDataOutput` → ViewModel 変換がインラインで書かれており、`page/edit.go` の `buildEditLinkResult` と重複しているが、フェーズ 6 のスコープで統一すれば問題ない

**作業計画書との整合性**:

- タスク 5-1 の要件（`GetPageDetailUsecase` の作成、`page/handler.go` から Repository 削除、UseCase 経由への変更）がすべて実装されている
- 前回レビューでの対応（`GetEditLinkDataUsecase` の独立化、`viewmodel` 重複削除、`draft_page/show.go` の UseCase 化）も完了している
