# コードレビュー: usecase-refactoring-1-2

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-20                                      |
| 対象ブランチ               | usecase-refactoring-1-2                         |
| ベースブランチ             | develop                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md |
| 変更ファイル数             | 14 ファイル                                     |
| 変更行数（実装）           | +43 / -54 行                                    |
| 変更行数（テスト）         | +141 / -149 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャ、Handler での処理フロー
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略、レイヤーごとのテストカバレッジ
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

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
- [x] `docs/reviews/usecase-refactoring-1-2-001.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っています。

## 設計改善の提案

設計改善の提案はありません。

## 設計との整合性チェック

作業計画書タスク1-2（`apply_suggestion.go` のリファクタリング）の要件との整合性を確認した。

| 要件                                                                      | 状態 |
| ------------------------------------------------------------------------- | ---- |
| `ApplySuggestionInput` に `Suggestion`, `SuggestionPages`, `Pages` を追加 | 完了 |
| UseCase内の `FindByID`, `ListBySuggestionID`, `FindByIDs` を削除          | 完了 |
| UseCase内のステータス検証を削除                                           | 完了 |
| Pagesの取得を読み取りUseCaseで事前に行う                                  | 完了 |
| Handler（`suggestion_apply/create.go`）の呼び出しを更新                   | 完了 |
| 関連テストの更新                                                          | 完了 |
| 外部から見た挙動（HTTPレスポンス、永続化結果）は変わらない                | 完了 |

すべての要件が満たされており、設計との乖離はない。

## 総合評価

**評価**: Approve

**総評**:

前回レビュー（001）で指摘された「Closedステータスの異常系テスト不足」が `TestCreate_クローズ済みの編集提案はエラーリダイレクトされる` として追加され、問題が解消されている。

実装面の確認結果:

- **アーキテクチャ**: 「Handler での処理フロー（読み取り → 検証 → 書き込み）」パターンに正しく従っている。書き込みUseCaseがデータ取得・検証を行わず、トランザクション内の永続化処理に専念している
- **依存関係**: `ApplySuggestionUsecase` から `suggestionPageRepo` の依存が削除され、不要な依存が減少した
- **読み取りUseCase**: `GetSuggestionDetailUsecase` に `pageRepo` を追加し、`Pages` を出力に含めることで、Handlerが書き込みUseCaseに取得済みモデルを渡せるようになった。`suggestionPages` が空の場合はDBアクセスをスキップする適切なガード条件がある
- **セキュリティ**: `spaceID` によるクエリスコープが維持されている。`FindByIDs` の呼び出しで `space.ID` が引き続き渡されている
- **テスト**: 正常系（単一ページ、複数ページ、LinkedPageIDs/FeaturedImageAttachmentIDの反映）と異常系（Closedステータス、Applied状態のべき等性）が網羅されている
- **コンストラクタ更新**: `main.go` および関連する全テストファイルの `NewGetSuggestionDetailUsecase` / `NewApplySuggestionUsecase` 呼び出しが正しく更新されている
