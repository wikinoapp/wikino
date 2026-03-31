# コードレビュー: usecase-1-2

## レビュー情報

| 項目                       | 内容                                                                |
| -------------------------- | ------------------------------------------------------------------- |
| レビュー日                 | 2026-03-27                                                          |
| 対象ブランチ               | usecase-1-2                                                         |
| ベースブランチ             | usecase-1-1                                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク 1-2）  |
| 変更ファイル数             | 18 ファイル                                                         |
| 変更行数（実装）           | +123 新規（dispatcher.go）、他ファイル約 +80 / -80 行（リファクタ） |
| 変更行数（テスト）         | +192 新規（dispatcher_test.go）、他テスト約 +120 / -110 行          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/dispatcher/dispatcher.go`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/usecase/create_password_reset_token.go`
- [x] `go/internal/usecase/send_email_confirmation.go`
- [x] `go/internal/worker/cleanup_rate_limits.go`
- [x] `go/internal/worker/client.go`
- [x] `go/internal/worker/send_email.go`
- [x] `go/internal/worker/send_email_confirmation.go`
- [x] `go/internal/worker/send_password_reset.go`

### テストファイル

- [x] `go/internal/dispatcher/dispatcher_test.go`
- [x] `go/internal/handler/email_confirmation/create_test.go`
- [x] `go/internal/handler/email_confirmation/edit_test.go`
- [x] `go/internal/handler/email_confirmation/update_test.go`
- [x] `go/internal/usecase/create_password_reset_token_test.go`
- [x] `go/internal/usecase/send_email_confirmation_test.go`
- [x] `go/internal/worker/cleanup_rate_limits_test.go`
- [x] `go/internal/worker/send_email_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計との整合性チェック

### タスク 1-2 の要件との対応

| 要件                                                                          | 状態 |
| ----------------------------------------------------------------------------- | ---- |
| `internal/dispatcher/dispatcher.go` を新規作成                                | ✅   |
| 既存の Worker Args 型を `internal/worker/` から `internal/dispatcher/` に移動 | ✅   |
| Enqueue メソッドを定義                                                        | ✅   |
| 既存の UseCase の `worker` 経由投入を `dispatcher` 経由に変更                 | ✅   |
| テストの追加                                                                  | ✅   |

### 詳細な確認結果

**Dispatcher パッケージの構成**:

- `JobInserter` インターフェースによる抽象化で、テスト時にモックに差し替え可能。作業計画書の設計（`NewDispatcher(riverClient)` で具象型を受け取る）を改善し、インターフェースを導入している点は良い判断
- 4 つの Args 型（`SendEmailArgs`, `SendEmailConfirmationArgs`, `SendPasswordResetArgs`, `CleanupRateLimitsArgs`）が正しく移動されている
- 各 Args 型に `Kind()` と `InsertOpts()` メソッドが定義されている
- 4 つの Enqueue メソッド（`EnqueueSendEmail`, `EnqueueEmailConfirmation`, `EnqueuePasswordReset`, `EnqueueCleanupRateLimits`）が定義されている

**依存の方向**:

- Dispatcher（Domain/Infrastructure 層）→ River（外部ライブラリ）のみ。上位層への依存なし ✅
- Worker（Application 層）→ Dispatcher（Domain/Infrastructure 層）✅
- UseCase（Application 層）→ Dispatcher（Domain/Infrastructure 層）✅
- 循環依存なし ✅

**UseCase の変更**:

- `send_email_confirmation.go`: `*worker.Client` → `*dispatcher.Dispatcher` に変更 ✅
- `create_password_reset_token.go`: `*worker.Client` → `*dispatcher.Dispatcher` に変更 ✅

**Worker の変更**:

- 全 4 Worker が `dispatcher` パッケージの Args 型を参照するように変更 ✅
- Worker のビジネスロジック自体は変更なし（タスク 4-3 で対応予定）✅
- `worker.Client` からエンキュー関連のメソッドが削除されている ✅

**main.go の DI 構成**:

- `dispatcher.NewDispatcher(riverClient.Client())` で Dispatcher を初期化 ✅
- 2 つの UseCase（`sendEmailConfirmationUC`, `createPasswordResetTokenUC`）に `jobDispatcher` を渡している ✅

**テスト**:

- `dispatcher_test.go`: 全 Enqueue メソッドと全 Args 型の `Kind()` メソッドをテスト ✅
- 全テスト関数で `t.Parallel()` が呼ばれている ✅
- 既存テストが Dispatcher を使用するように正しく更新されている ✅
- ハンドラーテストでもモック `JobInserter` を使用してテスト ✅

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 1-2（Dispatcher パッケージの新設）が作業計画書の設計通りに正しく実装されている。

良かった点:

- **インターフェースの導入**: 作業計画書の設計では具象型（`*river.Client`）を直接受け取る形だったが、`JobInserter` インターフェースを導入することでテスタビリティが向上している
- **一貫した変更**: Args 型の移動、UseCase の変更、Worker の変更、テストの更新がすべて漏れなく行われている
- **依存の方向**: Dispatcher が Domain/Infrastructure 層として正しく配置され、上位層への依存がない
- **テストカバレッジ**: 全 Enqueue メソッドと Args 型のテストが網羅されている
