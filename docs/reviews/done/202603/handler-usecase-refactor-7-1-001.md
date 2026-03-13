# コードレビュー: handler-usecase-refactor-7-1

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-13                                     |
| 対象ブランチ               | handler-usecase-refactor-7-1                   |
| ベースブランチ             | handler-usecase-refactor-7-0                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 31 ファイル                                    |
| 変更行数（実装）           | +237 / -133 行                                 |
| 変更行数（テスト）         | +105 / -86 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - 開発環境ガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/.golangci.yml`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/account/handler.go`
- [x] `go/internal/handler/account/new.go`
- [x] `go/internal/handler/password/edit.go`
- [x] `go/internal/handler/password/handler.go`
- [x] `go/internal/handler/password_reset/create.go`
- [x] `go/internal/handler/password_reset/handler.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/sign_in/handler.go`
- [x] `go/internal/handler/sign_in_two_factor/handler.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/handler.go`
- [x] `go/internal/usecase/create_password_reset_token.go`
- [x] `go/internal/usecase/get_account_new_data.go`
- [x] `go/internal/usecase/get_password_reset_token_data.go`
- [x] `go/internal/validator/sign_in.go`

### テストファイル

- [x] `go/internal/handler/account/create_test.go`
- [x] `go/internal/handler/account/new_test.go`
- [x] `go/internal/handler/password/edit_test.go`
- [x] `go/internal/handler/password/update_test.go`
- [x] `go/internal/handler/password_reset/create_test.go`
- [x] `go/internal/handler/password_reset/new_test.go`
- [x] `go/internal/handler/sign_in/create_test.go`
- [x] `go/internal/handler/sign_in/new_test.go`
- [x] `go/internal/handler/sign_in_two_factor/create_test.go`
- [x] `go/internal/handler/sign_in_two_factor/new_test.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/create_test.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/new_test.go`
- [x] `go/internal/usecase/create_password_reset_token_test.go`
- [x] `go/internal/validator/sign_in_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありませんでした。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 7-1「`.golangci.yml` に Handler → Repository 禁止と UseCase → Policy 禁止の depguard ルールを追加」が正しく実装されています。

**良かった点**:

- `.golangci.yml` に追加された 2 つの depguard ルール（Handler → Repository 禁止、UseCase → Policy 禁止）が正確に定義されている
- `make lint` が 0 issues で通ることを確認済み。depguard ルールが正しく機能しており、既存コードに違反がない
- depguard ルール追加に伴い、Handler から Repository への直接依存を排除するために新しい読み取り UseCase（`GetAccountNewDataUsecase`、`GetPasswordResetTokenDataUsecase`）が適切に作成されている
- 新しい UseCase は命名規則（`Get` プレフィックス = 読み取り UseCase）に準拠している
- `SignInCreateValidator` に `userTwoFactorAuthRepo` を追加し、2FA 情報の取得を Handler から Validator に移動することで、Handler の Repository 依存を排除している
- すべてのテストが新しいコンストラクタシグネチャに対応して更新されている
- `create_password_reset_token.go` に `cfg` を追加して `cfg.AppURL()` で URL 生成することで、Handler からの URL 渡しを不要にし、UseCase の自立性を高めている
