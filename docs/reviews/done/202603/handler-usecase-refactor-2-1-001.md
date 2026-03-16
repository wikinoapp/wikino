# コードレビュー: handler-usecase-refactor-2-1

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-12                                           |
| 対象ブランチ               | handler-usecase-refactor-2-1                         |
| ベースブランチ             | handler-usecase-refactor-1-1                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md       |
| 変更ファイル数             | 21 ファイル                                          |
| 変更行数（実装）           | +371 / -24 行（ファイル移動含む、実質 +28 行の差分） |
| 変更行数（テスト）         | +791 / -41 行（ファイル移動含む）                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/sign_in/handler.go`
- [x] `go/internal/handler/sign_in_two_factor/create.go`
- [x] `go/internal/handler/sign_in_two_factor/handler.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/create.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/handler.go`
- [x] `go/internal/validator/sign_in.go`
- [x] `go/internal/validator/sign_in_two_factor.go`
- [x] `go/internal/validator/sign_in_two_factor_recovery.go`

### テストファイル

- [x] `go/internal/handler/sign_in/create_test.go`
- [x] `go/internal/handler/sign_in/new_test.go`
- [x] `go/internal/handler/sign_in_two_factor/create_test.go`
- [x] `go/internal/handler/sign_in_two_factor/new_test.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/create_test.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/new_test.go`
- [x] `go/internal/validator/main_test.go`
- [x] `go/internal/validator/sign_in_test.go`
- [x] `go/internal/validator/sign_in_two_factor_test.go`
- [x] `go/internal/validator/sign_in_two_factor_recovery_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 2-1 の仕様に沿って、正確にリファクタリングが行われています。

**良かった点**:

- **命名規則の一貫性**: `{Resource}{Action}Validator` パターン（`SignInCreateValidator`、`SignInTwoFactorCreateValidator`、`SignInTwoFactorRecoveryCreateValidator`）がバリデーションガイドに完全に準拠
- **`ErrTwoFactorNotEnabled` の重複排除**: 旧 `sign_in_two_factor/validator.go` と `sign_in_two_factor_recovery/validator.go` の両方で定義されていた `ErrTwoFactorNotEnabled` を、同一パッケージ統合により `sign_in_two_factor.go` の 1 箇所にまとめている
- **依存性注入パターンの適用**: `main.go` で Validator を構築し Handler コンストラクタに渡す設計が、バリデーションガイドの「構築パターンの変更」に準拠
- **不要な依存の除去**: `sign_in/handler.go` から `userRepo`、`userPasswordRepo`、`userSessionRepo` フィールドを適切に削除
- **TestMain パターン**: `internal/validator/main_test.go` にテストガイドのパターン通りの `TestMain` を追加
- **パッケージドキュメント**: `sign_in.go` に `// Package validator はバリデーションを提供します` を追加
- **タスクスコープの適切な制御**: Handler に残る `repository` への依存（`userTwoFactorAuthRepo` 等）は今回のタスクスコープ外であり、後続タスクで対応予定

**確認済み事項**:

- 旧 validator ファイル（`handler/sign_in/validator.go`、`handler/sign_in_two_factor/validator.go`、`handler/sign_in_two_factor_recovery/validator.go`）が適切に削除されている
- `create.go` の各ハンドラーが `validator.{Resource}{Action}ValidatorInput` を正しく参照している
- エラー変数（`ErrUserNotFound`、`ErrInvalidPassword` 等）の参照が `validator.Err*` に正しく更新されている
- テストファイルの変数名が `validator` パッケージとの衝突を避けて `v` にリネームされている
