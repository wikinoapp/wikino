# コードレビュー: usecase-3-6

## レビュー情報

| 項目                       | 内容                                                        |
| -------------------------- | ----------------------------------------------------------- |
| レビュー日                 | 2026-03-30                                                  |
| 対象ブランチ               | usecase-3-6                                                 |
| ベースブランチ             | usecase-3-5                                                 |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md        |
| 変更ファイル数             | 10 ファイル（レビュードキュメント・作業計画書チェック除く） |
| 変更行数（実装）           | +186 / -109 行                                              |
| 変更行数（テスト）         | +261 / -157 行                                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/update_suggestion_page.go`
- [x] `go/internal/handler/suggestion_page/update.go`
- [x] `go/internal/handler/suggestion_page/handler.go`
- [x] `go/internal/validator/suggestion_page.go`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/usecase/update_suggestion_page_test.go`
- [x] `go/internal/handler/suggestion_page/update_test.go`
- [x] `go/internal/validator/suggestion_page_test.go`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計との整合性チェック

作業計画書のタスク 3-6 は以下を要求しています：

- **UseCase に SuggestionPageValidator + TopicPolicy を統合** → ✅ `update_suggestion_page.go` の `Execute` メソッドで fetchData → authorize → validate → persist の 4 ステップフローを実装済み
- **Handler（suggestion_page/update.go）を更新** → ✅ Handler は HTTP 入出力変換のみに徹しており、UseCase を 1 回呼び出すだけのシンプルな構造に変更済み
- **Validator の Result 型を廃止し `(data, error)` の 2 値返しに変更** → ✅ `SuggestionPageUpdateValidatorResult` を削除し `(*model.DraftPage, error)` を返すように変更済み
- **エラー型の使い分け** → ✅ Validator エラーは `AppError(Conflict)` に変換、認可エラーは `AppError(Forbidden)`、リソース不在は `AppError(ResourceNotFound)` として返却
- **Handler の AppError ハンドリング** → ✅ `model.AsAppError(err)` で判別し、コードに応じた HTTP レスポンスを返却。既存の `create_suggestion.go` ハンドラーと同一パターン

乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 3-6 の要件をすべて満たしており、既存のリファクタリング済み UseCase（`create_suggestion.go`, `close_suggestion.go` 等）と完全に一貫したパターンで実装されています。

**良かった点**:

- Handler が大幅にシンプルになり、HTTP の入出力変換に徹する設計方針が達成されている
- UseCase 内の fetchData → authorize → validate → persist の 4 ステップが明確に分離されており可読性が高い
- Validator の `SuggestionPageUpdateValidatorResult` 型を廃止し Go 標準の `(data, error)` 2 値返しに統一したことで、コードがシンプルになっている
- Handler テストで `TestUpdate_下書きステータスの編集提案ページが更新される` を `TestUpdate_反映済みの編集提案は更新できない` に置き換え、認可ロジックが UseCase に移動したことに合わせたテスト戦略の変更が適切
- UseCase テストに異常系（存在しないスペース、非メンバー、クローズ済み提案）が網羅的に追加されている
- i18n メッセージ `error_suggestion_page_update_conflict` が日英両方で追加されている
