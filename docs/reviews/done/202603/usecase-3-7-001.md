# コードレビュー: usecase-3-7

## レビュー情報

| 項目                       | 内容                                                              |
| -------------------------- | ----------------------------------------------------------------- |
| レビュー日                 | 2026-03-30                                                        |
| 対象ブランチ               | usecase-3-7                                                       |
| ベースブランチ             | usecase-3-6                                                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク3-7） |
| 変更ファイル数             | 7 ファイル                                                        |
| 変更行数（実装）           | +193 / -76 行                                                     |
| 変更行数（テスト）         | +165 / -89 行                                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_apply/handler.go`
- [x] `go/internal/handler/suggestion_apply/create.go`
- [x] `go/internal/usecase/apply_suggestion.go`

### テストファイル

- [x] `go/internal/handler/suggestion_apply/create_test.go`
- [x] `go/internal/usecase/apply_suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計との整合性チェック

### 作業計画書タスク 3-7 の要件

> - [x] **3-7**: [Go] apply_suggestion UseCase の移行
>   - UseCase に TopicPolicy を統合（Validator あり）
>   - Handler（suggestion_apply/create.go）を更新

### 確認結果

| 要件                               | 状態 | 確認内容                                                                                                                                             |
| ---------------------------------- | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| UseCase に TopicPolicy を統合      | ✅   | `authorize()` メソッドで `policy.NewTopicPolicy` を使用し、`model.AppError` (Forbidden) を返す                                                       |
| Handler を更新                     | ✅   | Handler から `policy` の import と `getSuggestionDetailUsecase` 依存を削除。UseCase 呼び出し + エラーハンドリングのみに簡素化                        |
| UseCase がオーケストレーターになる | ✅   | `Execute` メソッドで fetchData → authorize → checkIdempotency → checkStatus → applySuggestion の流れを統括                                           |
| Handler は HTTP 入出力変換に徹する | ✅   | リクエストパース → UseCase 呼び出し → `handleCreateError` でエラー種別に応じた HTTP レスポンス生成                                                   |
| AppError パターンでエラーを返す    | ✅   | ResourceNotFound, Forbidden, Conflict の各ケースで `model.AppError` を使用                                                                           |
| べき等性の処理                     | ✅   | `checkIdempotency` で反映済みの場合は成功出力を返す                                                                                                  |
| テストが追加されている             | ✅   | UseCase テスト（正常系 3、認可エラー 1、ステータスエラー 1、べき等性 1）、Handler テスト（未ログイン、404、403×2、成功、ステータスエラー、べき等性） |

### 既存パターンとの一貫性

`create_suggestion.go`（先行して移行済み）のパターンと比較:

| 観点                         | create_suggestion.go                       | apply_suggestion.go（本PR）                   | 評価 |
| ---------------------------- | ------------------------------------------ | --------------------------------------------- | ---- |
| Execute の構造               | fetchData → authorize → validate → persist | fetchData → authorize → statusCheck → persist | ✅   |
| データ取得の返り値           | 個別の変数を返す                           | `applySuggestionData` 構造体で集約            | ✅   |
| 認可チェック                 | `authorize()` メソッドに分離               | `authorize()` メソッドに分離                  | ✅   |
| TopicMember の取得場所       | `authorize()` 内で条件付き取得             | `fetchData()` 内で条件付き取得                | ✅   |
| エラー型                     | `*model.AppError` を使用                   | `*model.AppError` を使用                      | ✅   |
| Handler のエラーハンドリング | `handleCreateError` メソッドに分離         | `handleCreateError` メソッドに分離            | ✅   |

TopicMember の取得場所が `fetchData` 内にある点は `create_suggestion.go`（`authorize` 内）と異なるが、データ取得の責務を `fetchData` に集約する設計として妥当であり、問題はない。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 3-7 の要件を満たしている。Handler から UseCase へのオーケストレーション責務の移行が適切に行われており、以下の点が良い:

- **Handler の簡素化**: `policy` import と `getSuggestionDetailUsecase` 依存を削除し、HTTP 入出力変換のみに集中
- **UseCase の構造化**: `fetchData` → `authorize` → `checkIdempotency` / `checkStatus` → `applySuggestion` の明確なステップ分離
- **エラーハンドリング**: `model.AppError` / `model.AsAppError` パターンによる型安全なエラー判別
- **テストの充実**: UseCase テストで正常系・認可エラー・ステータスエラー・べき等性を網羅。Handler テストでも各エラーケースのHTTPレスポンスを検証
- **既存パターンとの一貫性**: `create_suggestion.go` 等の先行移行済み UseCase と同じパターンに従っている
