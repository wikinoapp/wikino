# コードレビュー: usecase-5a-1

## レビュー情報

| 項目                       | 内容                                                                |
| -------------------------- | ------------------------------------------------------------------- |
| レビュー日                 | 2026-03-31                                                          |
| 対象ブランチ               | usecase-5a-1                                                        |
| ベースブランチ             | usecase-5-2                                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク 5a-1） |
| 変更ファイル数             | 14 ファイル                                                         |
| 変更行数（実装）           | +155 / -127 行                                                      |
| 変更行数（テスト）         | +227 / -57 行                                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/usecase-guide.md](/workspace/go/docs/usecase-guide.md) - ユースケースガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/email/confirmation_sender.go`
- [x] `go/internal/email/password_reset_sender.go`
- [x] `go/internal/usecase/send_email_confirmation.go`
- [x] `go/internal/usecase/send_password_reset.go`
- [x] `go/internal/worker/client.go`

### テストファイル

- [x] `go/internal/email/confirmation_sender_test.go`
- [x] `go/internal/email/password_reset_sender_test.go`
- [x] `go/internal/usecase/send_email_confirmation_test.go`
- [x] `go/internal/usecase/send_email_test.go`
- [x] `go/internal/usecase/send_password_reset_test.go`

### 設定・その他

- [x] `go/.golangci.yml`
- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/send_email_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

**問題点・改善提案**:

- **[設計との整合性]**: `mockSender` が `send_email_test.go` に移動されたが、この `mockSender` は `email.Sender` インターフェース（`Send` + `SendRaw`）を実装している。作業計画書のタスク 5a-2 で `SendEmailUsecase` / `send_email.go` / `send_email_test.go` 自体が削除される予定のため、このファイルへのモック移動は中間状態として許容できる。ただし、タスク 5a-2 の完了時に `send_email_test.go` が確実に削除されることを確認する必要がある。

  **対応方針**:
  - [ ] 5a-2 タスクで確実に削除する（現状維持で OK）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  確実に削除するためにできることがあれば対応をお願いします。(作業計画書の更新やコメントを残すなど)
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 5a-1 の要件を正確に実装できている。

**良い点**:

- UseCase → templates の例外依存が完全に解消されており、UseCase 側では interface のみに依存する Clean Architecture パターンが正しく適用されている
- `ConfirmationSender` / `PasswordResetSender` はそれぞれのメール種別に特化した Sender として email パッケージ内に適切に配置されている
- depguard ルールの `application-layer` に `templates` の deny が追加され、`email-layer` も新設されており、アーキテクチャの強制が機能している
- テストも UseCase 側は interface のモックを使用し、email パッケージ側は `NoopSender` を使用する形に適切に分離されている
- ドキュメント更新（`CLAUDE.md`, `architecture-guide.md`）で「UseCase の templates 依存は例外として許可」の記述が削除され、新しい依存ルール（ルール 9）が追加されている
- `worker/client.go` の DI 構成も `ConfirmationSender` / `PasswordResetSender` を経由する形に正しく更新されている

**軽微な確認事項**:

- `send_email_test.go` への `mockSender` 移動は、タスク 5a-2 でファイルごと削除される中間状態として問題ない。5a-2 で確実に対応すること
