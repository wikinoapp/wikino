# コードレビュー: usecase-4-3

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-4-3                                          |
| ベースブランチ             | usecase-4-2                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 8 ファイル                                           |
| 変更行数（実装）           | +49 / -254 行                                        |
| 変更行数（テスト）         | +7 / -4 行                                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/worker/client.go`
- [x] `go/internal/worker/send_email.go`
- [x] `go/internal/worker/send_email_confirmation.go`
- [x] `go/internal/worker/send_password_reset.go`
- [x] `go/internal/worker/cleanup_rate_limits.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/worker/send_email_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計との整合性チェック

### タスク 4-3 の要件との対応

作業計画書のタスク 4-3 の記述:

> **4-3**: [Go] Worker を薄い Adapter に変更
>
> - 4つの Worker からビジネスロジックを削除し、UseCase を呼ぶだけに変更
> - Worker を Presentation 層として位置づける（ドキュメントはフェーズ 5 で更新）
> - `main.go` の DI 構成を更新

**確認結果**: すべての要件が実装されています。

| 要件                                                      | 状態     |
| --------------------------------------------------------- | -------- |
| SendEmailWorker を UseCase 呼び出しのみに変更             | 実装済み |
| SendEmailConfirmationWorker を UseCase 呼び出しのみに変更 | 実装済み |
| SendPasswordResetWorker を UseCase 呼び出しのみに変更     | 実装済み |
| CleanupRateLimitsWorker を UseCase 呼び出しのみに変更     | 実装済み |
| `main.go` の DI 構成を更新                                | 実装済み |

### 詳細な整合性確認

**Worker の薄い Adapter 化**:

4つの Worker（`SendEmailWorker`, `SendEmailConfirmationWorker`, `SendPasswordResetWorker`, `CleanupRateLimitsWorker`）すべてが、ビジネスロジック（バリデーション、ログ出力、テンプレートレンダリング等）を削除し、`job.Args` から `usecase.Input` へのマッピング + `uc.Execute()` の呼び出しのみに変更されている。作業計画書の方針通り。

**DI 構成の更新**:

- `worker.NewClient` が `*ratelimit.Limiter` を新たに受け取るように変更
- `client.go` 内で UseCase を生成し、Worker に注入する構成に変更
- `main.go` で `rateLimiter` の初期化を `worker.NewClient` 呼び出しの前に移動（依存関係の順序を正しく反映）

**`hasWorkers` フラグの削除**:

`CleanupRateLimitsWorker` が `emailSender` の有無に関係なく常に登録されるため、ワーカーが 0 個になるケースが無くなった。`hasWorkers` フラグとそれに基づく Start/Stop のスキップロジックが正しく削除されている。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 4-3（Worker を薄い Adapter に変更）が作業計画書の要件通りに実装されている。4つの Worker すべてからビジネスロジックが削除され、UseCase への委譲のみを行う薄い Adapter パターンに統一された。変更は大幅な行数削減（-254行の実装コード）となっており、ロジックが UseCase に正しく移動されたことが確認できる。`main.go` の DI 構成も適切に更新され、`hasWorkers` フラグの不要化に伴う削除も正しい。テストも UseCase 経由の呼び出しに更新されている。ガイドライン違反は見当たらない。
