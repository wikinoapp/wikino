# コードレビュー: usecase-3-10

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3-10                                         |
| ベースブランチ             | usecase-3-9                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 61 ファイル                                          |
| 変更行数（実装）           | +221 / -203 行（自動生成 `*_templ.go` を除く）       |
| 変更行数（テスト）         | +405 / -438 行                                       |

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

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/email_confirmation/update.go`
- [x] `go/internal/handler/errors.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/handler/page_move/create.go`
- [x] `go/internal/handler/page_move/new.go`
- [x] `go/internal/handler/password/edit.go`
- [x] `go/internal/handler/password_reset/create.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/sign_in/handler.go`
- [x] `go/internal/handler/sign_in_two_factor/create.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/create.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/handler/suggestion/edit.go`
- [x] `go/internal/handler/suggestion/new.go`
- [x] `go/internal/handler/suggestion/update.go`
- [x] `go/internal/handler/suggestion_comment_edit/edit.go`
- [x] `go/internal/handler/suggestion_comment_edit/update.go`
- [x] `go/internal/usecase/create_user_session.go`
- [ ] `go/internal/usecase/sign_in.go`
- [x] `go/internal/validator/email_confirmation.go`
- [x] `go/internal/validator/password.go`
- [x] `go/internal/validator/password_reset.go`
- [x] `go/internal/validator/sign_in.go`
- [x] `go/internal/validator/sign_in_two_factor.go`
- [x] `go/internal/validator/sign_in_two_factor_recovery.go`

### テンプレートファイル

- [x] `go/internal/templates/components/form_errors.templ`
- [x] `go/internal/templates/pages/email_confirmation/edit.templ`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page_move/new.templ`
- [x] `go/internal/templates/pages/password/edit.templ`
- [x] `go/internal/templates/pages/password/reset.templ`
- [x] `go/internal/templates/pages/sign_in/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/recovery_new.templ`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/suggestion/edit.templ`
- [x] `go/internal/templates/pages/suggestion/new.templ`
- [x] `go/internal/templates/pages/suggestion_comment/edit.templ`

### テストファイル

- [ ] `go/internal/handler/sign_in/create_test.go`
- [x] `go/internal/handler/sign_in/new_test.go`
- [x] `go/internal/usecase/create_user_session_test.go`
- [x] `go/internal/usecase/sign_in_test.go`
- [x] `go/internal/validator/sign_in_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`
- [x] `docs/reviews/usecase-3-10-001.md`
- [x] `docs/reviews/usecase-3-10-002.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/sign_in.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#命名規則](/workspace/go/docs/architecture-guide.md) - UseCase 命名規則
- [@go/docs/architecture-guide.md#Handler での処理フロー](/workspace/go/docs/architecture-guide.md) - UseCase の責務

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#命名規則]**: UseCase の命名規則は `{Action}{Entity}Usecase`（ファイル名: `{action}_{entity}.go`）だが、`SignInUsecase`（`sign_in.go`）は `{Action}{Entity}` の形式ではない

  現在のアーキテクチャガイドの命名規則:

  ```
  - ファイル名: {action}_{entity}.go（例: create_session.go）
  - 構造体名: {Action}{Entity}Usecase（例: CreateSessionUsecase）
  ```

  `SignInUsecase` は「SignIn」がアクションそのものであり、エンティティが不明確。リファクタリングの方向性として操作ベースの命名に移行する意図は理解できるが、既存の命名規則と不整合がある。

  **修正案**:

  以下のいずれかの対応が考えられる:
  - 案 A: アーキテクチャガイドの UseCase 命名規則を更新し、操作ベースの命名を許可する
  - 案 B: 命名を既存パターンに合わせて `CreateSignInUsecase` / `create_sign_in.go` とする

  **対応方針**:
  - [ ] 案 A: アーキテクチャガイドの命名規則を更新する（リファクタリング完了後にまとめて更新予定であれば OK）
  - [x] 案 B: 命名を `CreateSignInUsecase` / `create_sign_in.go` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/sign_in/create_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

**問題点・改善提案**:

- **[@go/docs/testing-guide.md]**: `setupHandler` ヘルパー関数が `new_test.go` とは別に定義されている `mockTurnstileVerifier` を使用しているが、`new_test.go` にも同じ目的の `mockTurnstileVerifierForNew` が別途定義されている。2 つのモック定義が同一パッケージ内に存在しているが、`setupHandler` は `create_test.go` にのみ使われている

  `create_test.go` で `setupHandler` を定義してテストコードの重複を削減したのは良い改善。ただし `new_test.go` 側はまだ旧来のパターン（手動でハンドラーを組み立てる）のままで、`setupHandler` を共有していない。一貫性の観点では `new_test.go` も `setupHandler` を活用できるが、本 PR のスコープ外であれば現状で問題ない。

  **修正案**:

  現状のまま。`new_test.go` のリファクタリングは別 PR でも OK。

  **対応方針**:
  - [ ] 現状のまま（別 PR で対応）
  - [x] `new_test.go` も `setupHandler` を共有するよう修正する
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

作業計画書に記載された「UseCase をオーケストレーターにする」方針に沿って、サインイン機能のリファクタリングが正しく実装されている。

**良い点**:

- **Handler が薄くなった**: `sign_in/create.go` から Validator 呼び出しとセッション作成を `SignInUsecase` に統合し、Handler は HTTP 入出力変換に専念している
- **`ValidationErrorToFormErrors` の削除**: `model.ValidationError` をテンプレートに直接渡すことで、中間変換が不要になった
- **`session.FormErrors` → `model.ValidationError` への一括置換**: 全テンプレートとハンドラーで一貫して `model.ValidationError` を使用している
- **Validator の返り値改善**: `sign_in.go` の Validator が `(*Output, error)` パターンに変更され、作業計画書で定めたエラー型の使い分け（ValidationError を error として返す）が実現されている
- **テストの改善**: ハンドラーテストに `setupHandler` ヘルパーを導入し、重複コードを大幅に削減。UseCase テストも新規に追加されている
- **テストデータのユニーク化**: テスト間でメールアドレス・atname が衝突しないよう修正されている
