# コードレビュー: usecase-3-11

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3-11                                         |
| ベースブランチ             | usecase-3-10                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 16 ファイル                                          |
| 変更行数（実装）           | +134 / -160 行                                       |
| 変更行数（テスト）         | +247 / -1097 行                                      |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/email_confirmation/handler.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/email_confirmation/update.go`
- [x] `go/internal/usecase/mark_email_as_confirmed.go`
- [x] `go/internal/usecase/send_email_confirmation.go`
- [x] `go/internal/validator/email_confirmation.go`

### テストファイル

- [x] `go/internal/handler/email_confirmation/create_test.go`
- [x] `go/internal/handler/email_confirmation/edit_test.go`
- [x] `go/internal/handler/email_confirmation/update_test.go`
- [x] `go/internal/usecase/mark_email_as_confirmed_test.go`
- [x] `go/internal/usecase/send_email_confirmation_test.go`
- [x] `go/internal/validator/email_confirmation_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/i18n/locales/ja.toml` / `go/internal/i18n/locales/en.toml`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 翻訳ファイルの追加手順

**問題点・改善提案**:

- **[@go/docs/i18n-guide.md#descriptionを必ず記述]**: `validation_confirmation_already_succeeded` の翻訳に `description` フィールドがない

  ```toml
  # 問題のあるコード
  [validation_confirmation_already_succeeded]
  other = "This confirmation has already been completed"
  ```

  **修正案**:

  ```toml
  # ja.toml
  [validation_confirmation_already_succeeded]
  description = "既に確認済みの場合のエラー"
  other = "この確認は既に完了しています"

  # en.toml
  [validation_confirmation_already_succeeded]
  description = "Error when confirmation is already completed"
  other = "This confirmation has already been completed"
  ```

  **対応方針**:
  - [x] 修正案の通り description を追加する
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

タスク 3-11（mark_email_as_confirmed UseCase の移行）が作業計画書の設計通りに正しく実装されています。

**良かった点**:

- Handler から Validator の直接呼び出しが完全に除去され、UseCase 経由に統一されている
- `(data, error)` の 2 値返しパターンへの移行が正確に行われている（Result 型の廃止）
- `errors.As` / `model.AsValidationError` / `model.AsAppError` による型判別パターンが一貫している
- 既存のセンチネルエラー（`ErrEmailConfirmationNotFound` 等）が `model.ValidationError` / `model.AppError` に正しく置き換えられている
- テストコードのリファクタリング（`newTestHandlerForCreate` / `newTestHandlerForEdit` ヘルパーの導入）でコード量が大幅に削減されている（-1097 行）
- `handler.go` から `validator` パッケージの import が除去され、Handler → Validator の依存が排除されている
- `main.go` の DI 構成が正しく更新されている（Validator を UseCase に渡し、Handler からは除去）

**指摘事項**: 1 件（i18n の description 欠落のみ、軽微）
