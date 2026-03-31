# コードレビュー: usecase-3-10

## レビュー情報

| 項目                       | 内容                                                                        |
| -------------------------- | --------------------------------------------------------------------------- |
| レビュー日                 | 2026-03-30                                                                  |
| 対象ブランチ               | usecase-3-10                                                                |
| ベースブランチ             | usecase-3-9                                                                 |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md                        |
| 変更ファイル数             | 66 ファイル（うち自動生成 \_templ.go: 13 ファイル）                         |
| 変更行数（実装）           | +927 / -216 行（.templ テンプレート含む、自動生成 \_templ.go・docs を除く） |
| 変更行数（テスト）         | +404 / -537 行                                                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/auth/token.go`（新規）
- [x] `go/internal/usecase/create_sign_in.go`（新規）
- [x] `go/internal/usecase/create_user_session.go`
- [x] `go/internal/handler/sign_in/handler.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/errors.go`
- [x] `go/internal/session/manager.go`
- [x] `go/internal/validator/sign_in.go`
- [x] `go/internal/validator/email_confirmation.go`
- [x] `go/internal/validator/password.go`
- [x] `go/internal/validator/password_reset.go`
- [x] `go/internal/validator/sign_in_two_factor.go`
- [x] `go/internal/validator/sign_in_two_factor_recovery.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/email_confirmation/update.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/handler/page_move/create.go`
- [x] `go/internal/handler/page_move/new.go`
- [x] `go/internal/handler/password/edit.go`
- [x] `go/internal/handler/password_reset/create.go`
- [x] `go/internal/handler/sign_in_two_factor/create.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/create.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/handler/suggestion/edit.go`
- [x] `go/internal/handler/suggestion/new.go`
- [x] `go/internal/handler/suggestion/update.go`
- [x] `go/internal/handler/suggestion_comment_edit/edit.go`
- [x] `go/internal/handler/suggestion_comment_edit/update.go`
- [x] `go/cmd/server/main.go`
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

- [x] `go/internal/usecase/create_sign_in_test.go`（新規）
- [x] `go/internal/handler/sign_in/create_test.go`
- [x] `go/internal/handler/sign_in/new_test.go`
- [x] `go/internal/usecase/create_user_session_test.go`
- [x] `go/internal/validator/sign_in_test.go`

### 設定・その他

- [x] `go/.golangci.yml`
- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`
- [ ] `docs/reviews/usecase-3-10-001.md`（既存レビュー、レビュー対象外）
- [ ] `docs/reviews/usecase-3-10-002.md`（既存レビュー、レビュー対象外）
- [ ] `docs/reviews/usecase-3-10-003.md`（既存レビュー、レビュー対象外）
- [ ] `docs/reviews/usecase-3-10-004.md`（既存レビュー、レビュー対象外）

### 自動生成ファイル（レビュー対象外）

- [x] `go/internal/templates/components/form_errors_templ.go`
- [x] `go/internal/templates/pages/email_confirmation/edit_templ.go`
- [x] `go/internal/templates/pages/page/edit_templ.go`
- [x] `go/internal/templates/pages/page_move/new_templ.go`
- [x] `go/internal/templates/pages/password/edit_templ.go`
- [x] `go/internal/templates/pages/password/reset_templ.go`
- [x] `go/internal/templates/pages/sign_in/new_templ.go`
- [x] `go/internal/templates/pages/sign_in_two_factor/new_templ.go`
- [x] `go/internal/templates/pages/sign_in_two_factor/recovery_new_templ.go`
- [x] `go/internal/templates/pages/sign_up/new_templ.go`
- [x] `go/internal/templates/pages/suggestion/edit_templ.go`
- [x] `go/internal/templates/pages/suggestion/new_templ.go`
- [x] `go/internal/templates/pages/suggestion_comment/edit_templ.go`

## ファイルごとのレビュー結果

問題なし。全ファイルのチェックが完了しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書に沿った質の高いリファクタリングです。以下の点が特に良い：

1. **UseCase オーケストレーションパターンの導入**: `CreateSignInUsecase` がバリデーション → 2FA 判定 → セッション作成の一連のフローを統括し、Handler を HTTP 入出力処理に集中させる設計が計画通り実現されている

2. **`session.FormErrors` → `model.ValidationError` の一括移行**: Handler、Validator、テンプレートの全箇所で一貫して型を置き換えており、漏れがない

3. **`auth.GenerateSecureToken` の抽出**: UseCase → session パッケージ（Presentation 層ヘルパー）への依存を解消し、depguard ルールで強制している。既存呼び出し元のためのラッパー関数も適切

4. **テストの改善**: `setupHandler` ヘルパーによりテストの重複コードが大幅に削減され、各テストケースの意図が明確になっている。新規の `CreateSignInUsecase` テストは正常系・異常系を網羅している

5. **段階的なアプローチ**: サインインの UseCase オーケストレーションを先行実装しつつ、他の Validator は型の移行（`session.FormErrors` → `model.ValidationError`）のみ行い、Result 構造体 →`(data, error)` パターンへの移行は将来のブランチに委ねている。一度に全てを変更せず、着実に進めている
