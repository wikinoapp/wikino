# コードレビュー: handler-usecase-refactor-5-1

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-12                                     |
| 対象ブランチ               | handler-usecase-refactor-5-1                   |
| ベースブランチ             | handler-usecase-refactor-4-1                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 11 ファイル（Go コードのみ）                   |
| 変更行数（実装）           | +382 / -373 行                                 |
| 変更行数（テスト）         | +29 / -7 行                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ、UseCase
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page/show.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page/handler.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/usecase/get_edit_link_data.go`
- [x] `go/internal/usecase/get_page_detail.go`
- [x] `go/internal/viewmodel/edit_link_data.go`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/page/edit_test.go`

## ファイルごとのレビュー結果

### UseCase テストファイルの欠落

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略
- [@CLAUDE.md#Pull Requestのガイドライン](/workspace/CLAUDE.md) - 実装とテストのセット化

**問題点・改善提案**:

- **[@CLAUDE.md#実装とテストのセット化]**: 新規 UseCase（`get_page_detail.go`、`get_edit_link_data.go`）に対応するテストファイルが存在しない

  タスク 4-1 で作成された `get_topic_detail.go` には `get_topic_detail_test.go` が作成されている。同じパターンに従い、新規 UseCase にもテストを追加すべき。

  **修正案**:
  - `go/internal/usecase/get_page_detail_test.go` を作成（正常系: スペース・メンバー・ページが存在する場合、異常系: スペースが存在しない場合に nil を返す、ページが存在しない場合に nil を返すなど）
  - `go/internal/usecase/get_edit_link_data_test.go` を作成（正常系: リンクデータの取得、リンクが存在しない場合の空スライス返却など）
  - テストパターンは `get_topic_detail_test.go` を参考にする

  **対応方針**:
  - [x] 両方のテストファイルを追加する
  - [ ] `get_page_detail_test.go` のみ追加する（`get_edit_link_data_test.go` はハンドラーテストで間接的にカバーされるため）
  - [ ] テスト追加はフェーズ 6 以降で別タスクとして対応する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/internal/handler/draft_page/show.go` と `go/internal/handler/page/edit.go`: `EditLinkBacklinks` → `PageSliceWithCount` 変換の重複

**ステータス**: 要確認

**現状**:

`draft_page/show.go`（L122-129）と `page/edit.go` の `buildEditLinkResult` 関数（L148-155）で、`usecase.EditLinkBacklinks` → `viewmodel.PageSliceWithCount` の変換パターンが重複している。

```go
// draft_page/show.go L122-129
backlinksPerPage := make(map[model.PageID]*viewmodel.PageSliceWithCount, len(linkData.BacklinksPerPage))
for pageID, backlinks := range linkData.BacklinksPerPage {
    backlinksPerPage[pageID] = &viewmodel.PageSliceWithCount{
        Pages:      backlinks.Pages,
        TotalCount: backlinks.TotalCount,
    }
}

// page/edit.go L149-155 (buildEditLinkResult内)
backlinksPerPage := make(map[model.PageID]*viewmodel.PageSliceWithCount, len(linkData.BacklinksPerPage))
for pageID, backlinks := range linkData.BacklinksPerPage {
    backlinksPerPage[pageID] = &viewmodel.PageSliceWithCount{
        Pages:      backlinks.Pages,
        TotalCount: backlinks.TotalCount,
    }
}
```

**提案**:

`draft_page/show.go` でも `buildEditLinkResult` 相当のヘルパーを使用するか、あるいはフェーズ 6 で `draft_page` ハンドラーを UseCase 化する際に `page/edit.go` の `buildEditLinkResult` を共通パッケージに移動する。

ただし、`draft_page/show.go` はフェーズ 6 で大幅にリファクタリングされる予定であるため、現時点での対応は不要かもしれない。

**メリット**:

- 重複コードの削減
- 変換ロジック変更時の一貫性保証

**トレードオフ**:

- フェーズ 6 で再度変更が入るため、現時点で共通化しても無駄になる可能性がある
- `buildEditLinkResult` は `page` パッケージのプライベート関数であり、共通化するにはパッケージ外に出す必要がある

**対応方針**:

- [x] 現時点では対応せず、フェーズ 6 で対応する
- [ ] 共通ヘルパーを作成する
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Request Changes

**総評**:

page ハンドラーから Repository への直接依存を完全に排除し、UseCase 経由に統一するリファクタリングは正しく実施されている。

**良い点**:

- `page/handler.go` から `repository` パッケージの import が完全に削除されている
- `GetPageDetailUsecase` は既存の `GetTopicDetailUsecase` と一貫した設計パターンに従っている
- `viewmodel.BuildExcludePageIDs` と `viewmodel.CollectTopicIDsFromPages` を UseCase パッケージの非公開関数に移動したのは適切（データ取得ロジックは Application 層の責務）
- `GetEditLinkDataUsecase` の抽出により、page handler と draft_page handler の両方でリンクデータ取得ロジックが再利用可能になった
- エラーハンドリングが適切（スペース・メンバー・ページ未発見時は nil 返却、トピック未発見時はエラー返却）

**修正が必要な点**:

- 新規 UseCase のテストファイルが欠落している（タスク 4-1 の `get_topic_detail_test.go` パターンに従うべき）
