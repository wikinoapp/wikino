# コードレビュー: usecase-4-2

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-4-2                                          |
| ベースブランチ             | usecase-4-1                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 3 ファイル                                           |
| 変更行数（実装）           | +50 / -0 行                                          |
| 変更行数（テスト）         | +62 / -0 行                                          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/cleanup_rate_limits.go`

### テストファイル

- [x] `go/internal/usecase/cleanup_rate_limits_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計との整合性チェック

### 作業計画書タスク 4-2 の要件との照合

| 要件                                               | 状態                                         |
| -------------------------------------------------- | -------------------------------------------- |
| `cleanup_rate_limits` の UseCase を新規作成        | ✅                                           |
| ロジックを Worker から UseCase に移動              | ✅                                           |
| 想定ファイル数: 約 2 ファイル（実装 1 + テスト 1） | ✅                                           |
| 想定行数: 約 100 行（実装 50 行 + テスト 50 行）   | ✅ 実績: 112 行（実装 50 行 + テスト 62 行） |

### 確認した設計パターンとの整合性

- **UseCase 命名規則**: `CleanupRateLimitsUsecase` は `{Action}{Entity}Usecase` パターンに準拠 ✅
- **ファイル命名規則**: `cleanup_rate_limits.go` は `{action}_{entity}.go` パターンに準拠 ✅
- **コンストラクタ命名**: `NewCleanupRateLimitsUsecase` は `New{Action}{Entity}Usecase` パターンに準拠 ✅
- **Execute メソッド**: `Execute(ctx context.Context, input CleanupRateLimitsInput) error` — 戻り値が Output 構造体ではなく `error` のみだが、副作用のみで出力データがないため妥当 ✅
- **Input 構造体**: `CleanupRateLimitsInput` を定義 ✅
- **ログ出力**: `log/slog` を使用、`slog.InfoContext` / `slog.ErrorContext` でコンテキスト付きログ ✅
- **エラーハンドリング**: `fmt.Errorf` でラップ、Worker 版（bare return）より改善されている ✅
- **Usecase の `c` は小文字**: ✅
- **テスト**: `t.Parallel()` 使用、`testutil.SetupTx(t)` で実データベース使用 ✅
- **Worker は未変更**: タスク 4-3 のスコープであり、本タスクでは UseCase 新設のみ。正しいスコープ ✅

### 4-1 のメール送信 UseCase との一貫性

| 観点             | send_email_confirmation             | cleanup_rate_limits                 | 一貫性 |
| ---------------- | ----------------------------------- | ----------------------------------- | ------ |
| 構造体命名       | `SendEmailConfirmationUsecase`      | `CleanupRateLimitsUsecase`          | ✅     |
| Input 構造体     | `SendEmailConfirmationInput`        | `CleanupRateLimitsInput`            | ✅     |
| Execute の返り値 | `error`                             | `error`                             | ✅     |
| ログ出力         | `slog.InfoContext` / `ErrorContext` | `slog.InfoContext` / `ErrorContext` | ✅     |
| エラーラッピング | `fmt.Errorf("...: %w", err)`        | `fmt.Errorf("...: %w", err)`        | ✅     |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 4-2（cleanup_rate_limits Worker 用の UseCase を新設）の要件を正確に満たしている。

- 既存の Worker からロジックを UseCase に移動する設計方針に忠実に従っている
- 命名規則、ログ出力、エラーハンドリングなどすべてのガイドラインに準拠
- 4-1 で作成されたメール送信 UseCase と一貫したパターンで実装されている
- テストは正常系とデフォルト値の 2 ケースをカバーしており、シンプルな UseCase の検証として十分
- Worker の変更は次のタスク（4-3）に正しく委ねられている
