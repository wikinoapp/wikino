# コードレビュー: usecase-4-1

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-4-1                                          |
| ベースブランチ             | usecase-3a-3                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 15 ファイル                                          |
| 変更行数（実装）           | +359 / -92 行                                        |
| 変更行数（テスト）         | +564 / -209 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/create_email_confirmation.go`（新規: 旧 SendEmailConfirmationUsecase のリネーム）
- [x] `go/internal/usecase/send_email_confirmation.go`（書き換え: メール送信 UseCase に変更）
- [x] `go/internal/usecase/send_email.go`（新規: 汎用メール送信 UseCase）
- [x] `go/internal/usecase/send_password_reset.go`（新規: パスワードリセットメール送信 UseCase）
- [x] `go/internal/handler/email_confirmation/handler.go`（リネーム: sendEmailConfirmationUC → createEmailConfirmationUC）
- [x] `go/internal/handler/email_confirmation/create.go`（リネーム: UseCase の参照名を更新）
- [x] `go/cmd/server/main.go`（DI 構成のリネーム）

### テストファイル

- [x] `go/internal/usecase/create_email_confirmation_test.go`（新規: 旧テストの移動）
- [x] `go/internal/usecase/send_email_confirmation_test.go`（書き換え: 新 UseCase のテスト）
- [x] `go/internal/usecase/send_email_test.go`（新規）
- [x] `go/internal/usecase/send_password_reset_test.go`（新規）
- [x] `go/internal/handler/email_confirmation/create_test.go`（リネーム）
- [x] `go/internal/handler/email_confirmation/edit_test.go`（リネーム）
- [x] `go/internal/handler/email_confirmation/update_test.go`（リネーム）

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`（タスク 4-1 チェックボックス更新）

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っています。

## 設計との整合性チェック

### 作業計画書タスク 4-1 との整合性

作業計画書のタスク 4-1 の要件:

| 要件                                                            | 状態    | 備考                                                                       |
| --------------------------------------------------------------- | ------- | -------------------------------------------------------------------------- |
| `send_email_confirmation` UseCase を新規作成                    | ✅ 完了 | テンプレートレンダリング + メール送信を実装                                |
| `send_password_reset` UseCase を新規作成                        | ✅ 完了 | テンプレートレンダリング + メール送信を実装                                |
| `send_email` UseCase を新規作成                                 | ✅ 完了 | 汎用メール送信を実装                                                       |
| テンプレートレンダリング + メール送信ロジックを Worker から移動 | ⚠️ 部分 | UseCase にロジックを複製。Worker からの削除はタスク 4-3 で行う想定（妥当） |

**補足**: Worker からのロジック削除はタスク 4-3（Worker を薄い Adapter に変更）で行われる計画であり、現時点での一時的なロジック重複は段階的移行の過渡期として妥当。

### アーキテクチャとの整合性

| チェック項目                                   | 結果 | 備考                                                                           |
| ---------------------------------------------- | ---- | ------------------------------------------------------------------------------ |
| UseCase → templates 依存（メールレンダリング） | ✅   | 作業計画書の検討事項 3 で例外として許可されている                              |
| UseCase → email パッケージ依存                 | ✅   | Infrastructure 層への依存は許可                                                |
| UseCase → i18n 依存                            | ✅   | 翻訳取得のための依存は許可                                                     |
| 命名規則 `{action}_{entity}.go`                | ✅   | `send_email_confirmation.go`, `send_password_reset.go`, `send_email.go`        |
| UseCase 構造体の命名 `{Action}{Entity}Usecase` | ✅   | `SendEmailConfirmationUsecase`, `SendPasswordResetUsecase`, `SendEmailUsecase` |
| Execute メソッドのシグネチャ                   | ✅   | `Execute(ctx context.Context, input) error` パターンに準拠                     |
| 旧 UseCase のリネーム（Send → Create）         | ✅   | 責務に合った命名に変更。Handler・main.go も整合                                |

### テストカバレッジ

| UseCase                          | 正常系テスト              | 異常系テスト            | 備考                             |
| -------------------------------- | ------------------------- | ----------------------- | -------------------------------- |
| `SendEmailConfirmationUsecase`   | ✅ ja + en                | ✅ 空メール, 送信エラー | 4 テスト関数                     |
| `SendPasswordResetUsecase`       | ✅ ja + en                | ✅ 空メール, 送信エラー | 4 テスト関数                     |
| `SendEmailUsecase`               | ✅                        | ✅ 空宛先, 送信エラー   | 3 テスト関数                     |
| `CreateEmailConfirmationUsecase` | ✅ ja + en + イベント種別 | -                       | 既存テストの移動（3 テスト関数） |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 4-1 の要件を正確に実装している。

**良かった点**:

- 旧 `SendEmailConfirmationUsecase`（実質は Create 処理）を `CreateEmailConfirmationUsecase` に適切にリネームし、新しい `SendEmailConfirmationUsecase` を責務に沿った形で新設した判断が良い
- 3 つの新規 UseCase がすべて同じパターン（入力バリデーション → テンプレートレンダリング → メール送信）で統一されており、一貫性が高い
- テストで `mockSender` を `send_email_confirmation_test.go` に定義し、同パッケージの他テストファイルから共有するパターンがシンプルで良い
- Handler・main.go・テストファイルの UseCase 参照リネームが漏れなく行われている
