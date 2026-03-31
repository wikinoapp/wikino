# コードレビュー: usecase-3-10

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3-10                                         |
| ベースブランチ             | usecase-3-9                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 35 ファイル（自動生成・レビュードキュメント除く）    |
| 変更行数（実装）           | +222 / -204 行                                       |
| 変更行数（テスト）         | +404 / -537 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
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
- [x] `go/internal/usecase/create_sign_in.go`
- [x] `go/internal/usecase/create_user_session.go`
- [x] `go/internal/validator/sign_in.go`
- [x] `go/internal/validator/email_confirmation.go`
- [x] `go/internal/validator/password.go`
- [x] `go/internal/validator/password_reset.go`
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

- [x] `go/internal/handler/sign_in/create_test.go`
- [x] `go/internal/handler/sign_in/new_test.go`
- [x] `go/internal/usecase/create_sign_in_test.go`
- [x] `go/internal/usecase/create_user_session_test.go`
- [x] `go/internal/validator/sign_in_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルのみ記載。

### `go/internal/usecase/create_sign_in.go`: UseCase が `session` パッケージに依存している

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - レイヤー間の依存関係

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#レイヤー間の依存関係]**: `create_sign_in.go` が `internal/session` パッケージの `GenerateSecureToken()` を import している。`session` パッケージは Presentation 層のヘルパーとして分類されており、Application 層（UseCase）から Presentation 層への依存は依存の方向に反する。

  ```go
  // 問題のあるコード（create_sign_in.go:10）
  import (
      "github.com/wikinoapp/wikino/go/internal/session"
  )

  // 使用箇所（create_sign_in.go:67）
  token, err := session.GenerateSecureToken()
  ```

  ただし、`create_user_session.go` でも同じく `session.GenerateSecureToken()` を使用しており、既存コードとの一貫性は保たれている。また、`GenerateSecureToken()` はトークン生成ユーティリティであり、Presentation 層のロジックではない。この問題は `create_sign_in.go` 固有の問題ではなく、`session` パッケージの分類の問題である。

  **修正案**:

  以下のいずれかで対応：
  - 案 A: `GenerateSecureToken()` を `internal/auth/` パッケージに移動する（トークン生成は認証関連のユーティリティ）
  - 案 B: 現状維持（既存の `create_user_session.go` との一貫性を優先し、将来のリファクタリングで対応）

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 案 A: `GenerateSecureToken()` を `internal/auth/` に移動する
  - [ ] 案 B: 現状維持（既存コードとの一貫性を優先）
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

タスク 3-10 の目的である「create_user_session UseCase に SignInCreateValidator を統合し、Handler を薄い Adapter にする」が作業計画書の設計通りに正しく実装されている。

**良かった点**:

- **作業計画書との整合性が高い**: 新しい `CreateSignInUsecase` がバリデーション → 2FA 判定 → セッション作成のフローを一貫して行い、Handler は HTTP 入出力変換に徹している
- **`ValidationErrorToFormErrors` ヘルパーの削除**: `errors.go` から `ValidationErrorToFormErrors` 変換関数を削除し、Handler が `model.ValidationError` を直接テンプレートに渡すパターンに統一。全ハンドラーで `session.FormErrors` → `model.ValidationError` への置換が完了している
- **テストの品質向上**: テストコードで `setupHandler` ヘルパーを導入してボイラープレートを大幅に削減。`create_user_session_test.go` のテストも各ケースが独立した `t.Run` + `t.Parallel()` に整理され、テストガイドに準拠
- **影響範囲の適切な管理**: テンプレート・バリデーター・ハンドラーの `session.FormErrors` 参照を `model.ValidationError` に一括置換し、移行の進捗を着実に進めている

**指摘事項**:

- 1 件の軽微な指摘（`session` パッケージへの依存）があるが、既存コードとの一貫性が保たれており、必須対応ではない
