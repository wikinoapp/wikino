# コードレビュー: usecase-5a-1

## レビュー情報

| 項目                       | 内容                                                                |
| -------------------------- | ------------------------------------------------------------------- |
| レビュー日                 | 2026-03-31                                                          |
| 対象ブランチ               | usecase-5a-1                                                        |
| ベースブランチ             | usecase-5-2                                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク 5a-1） |
| 変更ファイル数             | 15 ファイル                                                         |
| 変更行数（実装）           | +174 / -127 行                                                      |
| 変更行数（テスト）         | +208 / -57 行                                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/email/confirmation_sender.go`
- [x] `go/internal/email/password_reset_sender.go`
- [x] `go/internal/usecase/send_email_confirmation.go`
- [x] `go/internal/usecase/send_password_reset.go`
- [x] `go/internal/worker/client.go`
- [x] `go/.golangci.yml`

### テストファイル

- [x] `go/internal/email/confirmation_sender_test.go`
- [x] `go/internal/email/password_reset_sender_test.go`
- [x] `go/internal/usecase/send_email_confirmation_test.go`
- [x] `go/internal/usecase/send_password_reset_test.go`
- [ ] `go/internal/usecase/send_email_test.go`

### ドキュメント・設定

- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`
- [x] `docs/reviews/usecase-5a-1-001.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/send_email_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

**問題点・改善提案**:

- **[作業計画書 タスク 5a-2 との関係]**: `send_email_test.go` に `mockSender` が移動されているが、この `mockSender` は `email.Sender` インターフェース（`Send` + `SendRaw` の両方）を実装しており、`send_email_confirmation_test.go` や `send_password_reset_test.go` では既にこのモックを使用せず、interface ごとに専用のモック（`mockEmailConfirmationSender`, `mockPasswordResetSender`）を定義している。

  `mockSender` 自体は `SendEmailUsecase` のテストで引き続き使われているが、タスク 5a-2 では `SendEmailUsecase` ごと削除予定のため、**このファイル全体がタスク 5a-2 で削除される一時的なコード**である。

  作業計画書のタスク 5a-2 にも「`internal/usecase/send_email.go`, `internal/usecase/send_email_test.go` を削除（5a-1 で `mockSender` がこのファイルに移動されたため、ファイルごと削除すること）」と記載されているため、**意図的な一時配置**として問題ないことを確認したい。

  **対応方針**:
  - [x] 意図通りの一時配置であり、タスク 5a-2 で削除する（対応不要）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 5a-1 の要件をすべて満たしている。

- `ConfirmationSender` / `PasswordResetSender` を email パッケージに新設し、テンプレートレンダリング責務を UseCase から移動
- UseCase は `EmailConfirmationSender` / `PasswordResetSender` interface に依存し、`internal/templates` を直接 import しない
- `application-layer` に `templates` deny を追加、`email-layer` の depguard ルールを新設
- `main.go`（worker/client.go）の DI 構成を更新
- テストも各レイヤーで適切に更新

実装は作業計画書の設計に忠実で、アーキテクチャガイドラインにも準拠している。`send_email_test.go` への `mockSender` 移動は 5a-2 での削除が前提の一時的な措置であり、作業計画書にも明記されている。
