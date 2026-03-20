# コードレビュー: usecase-refactoring-1-1

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-20                                      |
| 対象ブランチ               | usecase-refactoring-1-1                         |
| ベースブランチ             | develop                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md |
| 変更ファイル数             | 3 ファイル（実装・テストのみ）                  |
| 変更行数（実装）           | +7 / -18 行                                     |
| 変更行数（テスト）         | +89 / -0 行                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（Handler での処理フロー）
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン（スペースIDによるクエリスコープ）

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/close_suggestion.go`
- [x] `go/internal/handler/suggestion_close/create.go`

### テストファイル

- [x] `go/internal/usecase/close_suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/write-usecase-refactoring.md`
- [x] `docs/reviews/usecase-refactoring-1-1-001.md`

## ファイルごとのレビュー結果

問題のあるファイルはありませんでした。全ファイルのチェックボックスにチェック済みです。

### レビュー確認の詳細

**`go/internal/usecase/close_suggestion.go`**:

- アーキテクチャガイド「Handler での処理フロー」に準拠: `FindByID` とステータス検証が削除され、永続化処理に専念している
- `WithTx` パターンが正しく使用されている
- `UpdateStatus` で `input.Suggestion.SpaceID` を渡しており、セキュリティガイドのスペースIDによるクエリスコープに準拠
- エラーメッセージが日本語で記述されている（コーディング規約準拠）
- `updatedSuggestion == nil` のガードにより、万一 DB 上で見つからない場合のエラーハンドリングも適切

**`go/internal/handler/suggestion_close/create.go`**:

- 変更は最小限（`SuggestionID`/`SpaceID` の個別指定を `Suggestion` モデル渡しに変更）
- ステータス検証は Handler 側（L74: クローズ済みチェック、L81: オープンステータスチェック）に既に存在しており、「読み取り → 検証 → 書き込み」のフローが正しく維持されている

**`go/internal/usecase/close_suggestion_test.go`**:

- `GetTestDB()` を使用: UseCase テスト（自前でトランザクション管理）に適切なヘルパーを使用
- `t.Parallel()` が各テストケースに含まれている
- テストデータに一意な識別子（`close-sugg-1`, `close-sugg-2`）を使用しており、並行実行安全
- 正常系（オープン→クローズ）と異常系（存在しないID）の両方をカバー
- 前回レビュー（001）で指摘された異常系テスト不足が対応済み

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

作業計画書の要件はすべて満たされている。前回レビュー（001）の指摘事項（異常系テスト追加）も対応済み。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 1-1 の要件に忠実にリファクタリングされている。書き込み UseCase からデータ取得（`FindByID`）とステータス検証を削除し、永続化処理に専念させるという方針が正しく実装されている。Handler 側では既にステータスチェックが行われており、外部挙動に変更はない。前回レビュー（001）で指摘された異常系テスト不足も対応済みで、正常系・異常系の両方がカバーされている。

アーキテクチャガイド、セキュリティガイド（スペースIDスコープ）、コーディング規約、テストガイドのすべてに準拠しており、問題点は見当たらない。
