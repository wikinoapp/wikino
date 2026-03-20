# コードレビュー: usecase-refactoring-1-1

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-20                                      |
| 対象ブランチ               | usecase-refactoring-1-1                         |
| ベースブランチ             | develop                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md |
| 変更ファイル数             | 4 ファイル                                      |
| 変更行数（実装）           | +4 / -18 行                                     |
| 変更行数（テスト）         | +65 / -0 行                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（Handler での処理フロー）
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/close_suggestion.go`
- [x] `go/internal/handler/suggestion_close/create.go`

### テストファイル

- [ ] `go/internal/usecase/close_suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/write-usecase-refactoring.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/close_suggestion_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド（テストカバレッジ、SetupTx vs GetTestDB の使い分け）
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase テストの方針

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#レイヤーごとのテストカバレッジ]**: UseCase テストが正常系のみで、異常系のテストがない

  現在のテストは「オープンステータスの編集提案をクローズできる」の1ケースのみ。リファクタリング後の UseCase は永続化処理に専念しているため異常系が限定的だが、最低限以下のケースがあるとより堅牢：
  - トランザクション内の `UpdateStatus` が失敗するケース（例: 存在しない ID を渡した場合）

  ただし、リファクタリングによりステータス検証が Handler に移動したため、UseCase レベルでの異常系は DB エラーのみであり、正常系1ケースでも許容範囲と言える。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 異常系テスト（存在しない ID など）を追加する
  - [ ] 現状のまま（正常系のみで十分と判断）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

作業計画書（タスク 1-1）の要件との整合性を確認：

| 要件                                                            | 状態 |
| --------------------------------------------------------------- | ---- |
| `CloseSuggestionInput` に `Suggestion *model.Suggestion` を追加 | ✅   |
| UseCase 内の `FindByID` を削除                                  | ✅   |
| UseCase 内のステータス検証を削除                                | ✅   |
| Handler（`suggestion_close/create.go`）の呼び出しを更新         | ✅   |
| 関連テストの更新                                                | ✅   |
| 外部から見た挙動（HTTP レスポンス、永続化結果）が変わらない     | ✅   |
| Handler でステータス検証を行っている                            | ✅   |

Handler 側では既に `getSuggestionDetailUsecase` で取得した `detailOutput.Suggestion` のステータスを確認（L74: クローズ済みチェック、L81: オープンステータスチェック）してから書き込み UseCase を呼び出しており、アーキテクチャガイドの「Handler での処理フロー（読み取り → 検証 → 書き込み）」に正しく従っている。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 1-1 の要件に忠実にリファクタリングされている。書き込み UseCase からデータ取得（`FindByID`）とステータス検証を削除し、永続化処理に専念させるという方針が正しく実装されている。Handler 側では既にステータスチェックが行われており、外部挙動に変更はない。

変更量も小さく（実装 +4/-18 行）、新規テストも追加されている。テストが正常系のみである点は確認事項として挙げたが、リファクタリングの性質上、UseCase の責務が永続化のみに縮小されたため許容範囲。
