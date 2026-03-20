# コードレビュー: usecase-refactoring-1-2

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-20                                      |
| 対象ブランチ               | usecase-refactoring-1-2                         |
| ベースブランチ             | develop                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md |
| 変更ファイル数             | 13 ファイル                                     |
| 変更行数（実装）           | +43 / -54 行                                    |
| 変更行数（テスト）         | +85 / -149 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャ、Handler での処理フロー
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略、レイヤーごとのテストカバレッジ

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/apply_suggestion.go`
- [x] `go/internal/usecase/get_suggestion_detail.go`
- [x] `go/internal/handler/suggestion_apply/create.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/usecase/apply_suggestion_test.go`
- [x] `go/internal/usecase/get_suggestion_detail_test.go`
- [x] `go/internal/handler/suggestion_apply/create_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/handler/suggestion_close/create_test.go`
- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [x] `go/internal/handler/suggestion_page/update_test.go`
- [x] `go/internal/handler/suggestion_page_edit/create_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/write-usecase-refactoring.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/apply_suggestion_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - レイヤーごとのテストカバレッジ
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Handler での処理フロー

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#レイヤーごとのテストカバレッジ]**: 「異常系: オープンステータス以外の編集提案は反映できない」テストが削除されたが、対応するハンドラーテストが不足している

  ステータス検証がUseCaseからHandler（`suggestion_apply/create.go` L80-85）に移動したことに伴い、UseCaseテストから異常系テスト（Closedステータスの編集提案を渡した場合）が削除された。しかし、`create_test.go` には「反映済み（Applied）の場合はべき等に成功する」テスト（`TestCreate_反映済みの編集提案はべき等に成功する`）はあるが、「Closedステータスの場合はエラーリダイレクトされる」テストがない。

  テスティングガイドには「Handler テストはエンドポイントの統合テストとして重要」「正常系だけでなく異常系も網羅」と記載されている。Handler に移動したステータス検証の異常系パスがテストされていない状態になっている。

  **修正案**:

  `go/internal/handler/suggestion_apply/create_test.go` に、Closedステータスの編集提案に対する反映リクエストがエラーリダイレクト（303 + flashエラー）を返すことを検証するテストを追加する。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] ハンドラーテストに「Closedステータスの編集提案はエラーリダイレクトされる」テストを追加する
  - [ ] 既存のべき等性テストで十分と判断し、追加しない（理由を回答欄に記入）
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

作業計画書タスク1-2（`apply_suggestion.go` のリファクタリング）の要件をすべて満たしている。

- `ApplySuggestionInput` に `Suggestion`, `SuggestionPages`, `Pages` が追加され、`SuggestionID`/`SpaceID` が削除された
- UseCase内の `FindByID`, `ListBySuggestionID`, `FindByIDs` とステータス検証が正しく削除された
- Pagesの取得は読み取りUseCase（`GetSuggestionDetailUsecase`）に追加された。`suggestionPages` が空の場合はDBアクセスをスキップする適切なガード条件がある
- Handler（`suggestion_apply/create.go`）が読み取りUseCaseの出力を書き込みUseCaseに渡す「Handler での処理フロー」パターンに正しく従っている
- `main.go` のコンストラクタ呼び出しが適切に更新されている
- テストコードは全体として行数が減少（-64行）しており、テスト内でのデータ取得コードは増加したがUseCase内の冗長なロジックが削減された結果、全体としてシンプルになっている

指摘事項は1件（軽微）: Closedステータスの異常系テストがHandler側に移植されていない点のみ。修正は任意。
