# コードレビュー: usecase-3-2

## レビュー情報

| 項目                       | 内容                                                              |
| -------------------------- | ----------------------------------------------------------------- |
| レビュー日                 | 2026-03-27                                                        |
| 対象ブランチ               | usecase-3-2                                                       |
| ベースブランチ             | usecase-3-1                                                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク3-2） |
| 変更ファイル数             | 10 ファイル                                                       |
| 変更行数（実装）           | +183 / -91 行                                                     |
| 変更行数（テスト）         | +147 / -22 行                                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/update_suggestion.go`
- [x] `go/internal/handler/suggestion/update.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/edit.go`
- [x] `go/internal/validator/suggestion.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/usecase/update_suggestion_test.go`
- [x] `go/internal/validator/suggestion_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3-2（update_suggestion UseCase の移行）が作業計画書の設計通りに実装されています。

**良かった点**:

- **create_suggestion との一貫性**: `Execute` メソッド内の処理フロー（1. データ取得 → 2. 認可チェック → 3. バリデーション → 4. ビジネスロジック → 5. 永続化）が `CreateSuggestionUsecase` と完全に一貫している
- **authorize メソッドのパターン統一**: `spaceMember == nil` チェック → 条件付き `topicMember` 取得 → Policy チェックの流れが create と update で統一されている
- **Handler の簡素化**: `update.go` から Policy / Validator の直接呼び出しが除去され、Handler は HTTP の入出力変換に徹している
- **エラーハンドリング**: `handleUpdateError` メソッドで `ValidationError` / `AppError` / 素の `error` を適切に判別し、作業計画書の「Handler での使用パターン」に沿っている
- **Validator の戻り値変更**: `SuggestionUpdateValidator.Validate` が `*SuggestionUpdateValidatorResult` ではなく `error` を返すように変更され、UseCase → Validator → error の依存方向が設計通り
- **テストの充実**: 正常系に加え、AppError（NotFound / Forbidden）と ValidationError の異常系テストが追加されている
- **edit.go の認可チェック維持**: 読み取り UseCase はフォーム表示専用のため、Handler での Policy チェックを維持する判断が正しい。コメントで理由が説明されている
- **作業計画書の更新**: タスク 3-2 の完了マーク、および新たに発見した課題（edit.go の読み取り UseCase 最適化）をタスク 3a-3 として追加している
