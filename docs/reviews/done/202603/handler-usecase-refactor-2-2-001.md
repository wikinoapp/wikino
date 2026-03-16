# コードレビュー: handler-usecase-refactor-2-2

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-12                                     |
| 対象ブランチ               | handler-usecase-refactor-2-2                   |
| ベースブランチ             | handler-usecase-refactor-2-1                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 29 ファイル                                    |
| 変更行数（実装）           | +530 / -81 行                                  |
| 変更行数（テスト）         | +1659 / -59 行                                 |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/account/create.go`
- [x] `go/internal/handler/account/handler.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/email_confirmation/handler.go`
- [x] `go/internal/handler/email_confirmation/update.go`
- [x] `go/internal/handler/password/handler.go`
- [x] `go/internal/handler/password/update.go`
- [x] `go/internal/handler/password_reset/create.go`
- [x] `go/internal/handler/password_reset/handler.go`
- [x] `go/internal/handler/password_reset/validator.go`（削除）
- [ ] `go/internal/validator/account.go`
- [x] `go/internal/validator/email_confirmation.go`
- [x] `go/internal/validator/password.go`
- [x] `go/internal/validator/password_reset.go`

### テストファイル

- [x] `go/internal/handler/account/create_test.go`
- [x] `go/internal/handler/account/new_test.go`
- [x] `go/internal/handler/email_confirmation/create_test.go`
- [x] `go/internal/handler/email_confirmation/edit_test.go`
- [x] `go/internal/handler/email_confirmation/update_test.go`
- [x] `go/internal/handler/password/edit_test.go`
- [x] `go/internal/handler/password/update_test.go`
- [x] `go/internal/handler/password_reset/create_test.go`
- [x] `go/internal/handler/password_reset/new_test.go`
- [x] `go/internal/validator/account_test.go`
- [x] `go/internal/validator/email_confirmation_test.go`
- [x] `go/internal/validator/password_test.go`
- [x] `go/internal/validator/password_reset_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/validator/account.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **[@go/docs/validation-guide.md#構造体の命名規則]**: `ErrEmailConfirmationNotFound` が `account.go` に定義されているが、`email_confirmation.go` からも参照されている。`account.go` にメール確認関連のエラー変数があると、コードを読む際に混乱する可能性がある。

  ```go
  // account.go に定義されている
  var (
      ErrEmailConfirmationNotFound = errors.New("メール確認情報が見つかりません")
      ErrEmailNotConfirmed         = errors.New("メール確認が完了していません")
      ErrAtnameAlreadyTaken        = errors.New("このアットネームは既に使用されています")
  )
  ```

  **修正案**:

  共有エラー変数（`ErrEmailConfirmationNotFound`）を `email_confirmation.go` に移動する。`ErrEmailNotConfirmed` は account のバリデーションでのみ使用されるため `account.go` に残す。

  ```go
  // email_confirmation.go に移動
  var (
      ErrEmailConfirmationNotFound          = errors.New("メール確認情報が見つかりません")
      ErrEmailConfirmationAlreadySucceeded  = errors.New("このメール確認は既に完了しています")
      ErrEmailConfirmationExpired           = errors.New("確認コードの有効期限が切れています")
      ErrEmailConfirmationCodeMismatch      = errors.New("確認コードが正しくありません")
      ErrEmailAlreadyRegistered             = errors.New("このメールアドレスは既に登録されています")
  )

  // account.go に残す
  var (
      ErrEmailNotConfirmed  = errors.New("メール確認が完了していません")
      ErrAtnameAlreadyTaken = errors.New("このアットネームは既に使用されています")
  )
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `ErrEmailConfirmationNotFound` を `email_confirmation.go` に移動する
  - [ ] 現状のまま（同一パッケージ内なので問題ない）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

タスク 2-2（account, email_confirmation, password, password_reset の Validator 移動）の実装が作業計画書の設計に従って正しく行われている。

**良い点**:

- Validator の `internal/validator/` パッケージへの移動が一貫したパターンで実施されている
- `email_confirmation/handler.go` から `repository` import が完全に除去されている
- `password_reset/handler.go` から `repository` import が完全に除去されている
- `main.go` で Validator を構築し Handler に渡すパターンが統一されている
- 命名規則（`{Resource}{Action}Validator`）が正しく適用されている
- テストが適切に更新されている

**注意点（今回のスコープ外）**:

- `account/handler.go` と `password/handler.go` は引き続き `repository` を import している。これは `new.go` と `edit.go` で Repository を直接呼び出しているためで、後続フェーズ（UseCase 化）で解消される想定。現時点では作業計画書通りの状態。

**問題点は軽微（1 件）**:

- `ErrEmailConfirmationNotFound` の配置場所について。同一パッケージ内なので動作に問題はないが、可読性の観点で確認が必要。
