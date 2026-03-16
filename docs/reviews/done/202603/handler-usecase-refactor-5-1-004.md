# コードレビュー: handler-usecase-refactor-5-1

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-12                                     |
| 対象ブランチ               | handler-usecase-refactor-5-1                   |
| ベースブランチ             | handler-usecase-refactor-4-1                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 17 ファイル                                    |
| 変更行数（実装）           | +390 / -372 行                                 |
| 変更行数（テスト）         | +357 / -8 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

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
- [x] `go/internal/usecase/get_edit_link_data_test.go`
- [x] `go/internal/usecase/get_page_detail_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`
- [x] `docs/reviews/done/202603/handler-usecase-refactor-5-1-001.md`
- [x] `docs/reviews/done/202603/handler-usecase-refactor-5-1-002.md`
- [x] `docs/reviews/done/202603/handler-usecase-refactor-5-1-003.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 5-1「GetPageDetailUsecase の作成と page ハンドラーの修正」が正確に実装されている。

**良い点**:

- `page/handler.go` から 6 つの Repository フィールドを削除し、2 つの UseCase フィールドに置き換えることで、Handler → Repository の直接依存を完全に排除した。`repository` パッケージの import も消えている
- `GetPageDetailUsecase` は not-found ケースで `nil, nil` を返し、データ整合性エラー（ページが存在するのにトピックが見つからない場合）のみ `error` を返す設計が適切
- `GetEditLinkDataUsecase` は `page/edit.go` と `draft_page/show.go` の重複ロジックを集約している
- テストカバレッジが十分（正常系・異常系を網羅）
- UseCase の命名規則（`Get{Entity}Usecase`、ファイル名 `get_{entity}.go`）がガイドラインに準拠
- Handler 構造体のフィールド数が 7 つで、ガイドラインの上限 8 以内

**観察事項（修正不要）**:

- `draft_page/show.go` と `page/edit.go` で `EditLinkBacklinks → PageSliceWithCount` への変換コードに軽微な重複があるが、Phase 6 で `draft_page` ハンドラーのリファクタリング時に解消される想定であり、現時点での対応は不要
