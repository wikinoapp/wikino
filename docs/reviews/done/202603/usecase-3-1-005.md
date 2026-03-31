# コードレビュー: usecase-3-1

## レビュー情報

| 項目                       | 内容                                                     |
| -------------------------- | -------------------------------------------------------- |
| レビュー日                 | 2026-03-27                                               |
| 対象ブランチ               | usecase-3-1                                              |
| ベースブランチ             | usecase-2-1                                              |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md     |
| 変更ファイル数             | 22 ファイル（うちレビュードキュメント 4 + 計画書更新 1） |
| 変更行数（実装）           | 約 +340 / -110 行                                        |
| 変更行数（テスト）         | 約 +590 / -150 行                                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion_comment/create.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/usecase/create_suggestion.go`
- [x] `go/internal/validator/suggestion.go`

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [x] `go/internal/policy/topic_test.go`
- [x] `go/internal/usecase/create_suggestion_test.go`
- [x] `go/internal/validator/suggestion_test.go`

### 設定・その他

- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`
- [x] `docs/reviews/usecase-3-1-001.md`
- [x] `docs/reviews/usecase-3-1-002.md`
- [x] `docs/reviews/usecase-3-1-003.md`
- [x] `docs/reviews/usecase-3-1-004.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3-1（create_suggestion UseCase の移行）は、作業計画書の設計に忠実に実装されている。主な良い点：

1. **UseCase がオーケストレーターとして正しく機能**: `Execute` メソッドが fetchData → authorize → validate → business logic → persist の順序を守り、各ステップがプライベートメソッドに委譲されている（ルール3「Execute 内にロジックを直接書かない」を遵守）

2. **Handler の薄い Adapter 化が完了**: `suggestion/create.go` は HTTP パース → UseCase 呼び出し → エラーハンドリング/リダイレクトに徹しており、ドメインロジックが Handler から完全に排除されている

3. **エラーハンドリングが計画通り**: `errors.As` パターンで `ValidationError` / `AppError` / 素の `error` を型で判別し、適切な HTTP レスポンスを返している。Forbidden は 404 として返すセキュリティ考慮も適切

4. **Validator の `(data, error)` 返し移行が完了**: `SuggestionCreateValidator` の Result 型が廃止され、Go 標準の2値返しに変更されている。`session.FormErrors` → `model.ValidationError` への移行が進んでいる

5. **Policy の CanCreateSuggestion が全ロールで適切に実装**: Owner（同スペース内）、Admin/Member（所属トピック内）、Guest（公開トピックのみ）の権限分離が正しく、テストカバレッジも十分

6. **`validationErrorToFormErrors` ブリッジ関数**: テンプレートが `session.FormErrors` を期待する過渡期の対応として適切。コメントで一時的であることが明示されている

7. **UseCase テストに異常系が追加**: 存在しないスペース/トピック、非メンバー、バリデーションエラーの各ケースがテストされており、エラー型の検証も `model.AsAppError` / `model.AsValidationError` で行われている

8. **パイロット（タスク 2-1）で確立したパターンとの一貫性**: `suggestion_comment/create.go` の Forbidden → 404 変更も含め、エラーハンドリングパターンが統一されている
