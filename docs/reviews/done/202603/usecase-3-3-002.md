# コードレビュー: usecase-3-3

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-27                                           |
| 対象ブランチ               | usecase-3-3                                          |
| ベースブランチ             | usecase-3-2                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 13 ファイル                                          |
| 変更行数（実装）           | 約 +160 / -115 行                                    |
| 変更行数（テスト）         | 約 +200 / -50 行                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/errors.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/handler/suggestion/update.go`
- [x] `go/internal/handler/suggestion_comment_edit/handler.go`
- [x] `go/internal/handler/suggestion_comment_edit/update.go`
- [x] `go/internal/usecase/update_suggestion_comment.go`
- [x] `go/internal/validator/suggestion_comment.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/handler/suggestion_comment_edit/edit_test.go`
- [x] `go/internal/usecase/update_suggestion_comment_test.go`
- [x] `go/internal/validator/suggestion_comment_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルのみ記載。問題がないファイルは上記チェックボックスにチェック済み。

（全ファイル問題なし）

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 3-3「update_suggestion_comment UseCase の移行」が正しく実装されている。

**良かった点**:

- **既存パターンとの高い一貫性**: `create_suggestion.go` の `fetchData → authorize → validate → business logic → persist` パターンと完全に一致しており、コードベース全体の統一感が保たれている
- **`ValidationErrorToFormErrors` の共通化**: `suggestion/create.go` にあったプライベート関数を `handler/errors.go` に公開関数として移動し、3箇所（suggestion/create, suggestion/update, suggestion_comment_edit/update）で共有できるようにした点が良い
- **Handler の薄さ**: `update.go` が「リクエストパース → UseCase呼び出し → レスポンス」のみに整理され、作業計画書の方針通り
- **エラーハンドリングの一貫性**: `handleUpdateError` のパターンが `suggestion/create.go` の `handleCreateError` と同一構造
- **テストカバレッジの充実**: 正常系2件に加え、異常系3件（存在しないスペース、非メンバー、バリデーションエラー）が追加され、UseCase 内に移動した認可・バリデーションのテストが独立して行われている
- **Validator の `error` 返し**: `SuggestionCommentUpdateValidatorResult` を廃止し `error` を返すパターンへの変更が作業計画書の設計通り
- **アーキテクチャルール準拠**: UseCase → policy/validator/repository の依存、Handler → UseCase のみの依存が正しく守られている
- **セキュリティ**: space_id スコープがクエリレベルで維持されている（`FindBySpaceAndNumber`, `FindByNumber` に space.ID を渡している）
