# コードレビュー: usecase-3-9

## レビュー情報

| 項目                       | 内容                                                              |
| -------------------------- | ----------------------------------------------------------------- |
| レビュー日                 | 2026-03-30                                                        |
| 対象ブランチ               | usecase-3-9                                                       |
| ベースブランチ             | usecase-3-8                                                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク3-9） |
| 変更ファイル数             | 14 ファイル                                                       |
| 変更行数（実装）           | +250 / -232 行                                                    |
| 変更行数（テスト）         | +213 / -575 行                                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/create_account.go`
- [x] `go/internal/validator/account.go`
- [x] `go/internal/handler/account/create.go`
- [x] `go/internal/handler/account/handler.go`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/templates/pages/account/new.templ`
- [x] `go/internal/templates/pages/account/new_templ.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/usecase/create_account_test.go`
- [x] `go/internal/validator/account_test.go`
- [x] `go/internal/handler/account/create_test.go`
- [x] `go/internal/handler/account/new_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/handler/account/new_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

**問題点・改善提案**:

- **[@go/docs/testing-guide.md]**: `create_test.go` では `setupHandler` ヘルパーと `createConfirmedEmailConfirmation` ヘルパーを導入してテストのボイラープレートを大幅に削減しているが、`new_test.go` は同じリファクタリングが行われておらず、各テスト関数（`TestNew`, `TestNew_NoEmailConfirmationID`, `TestNew_EmailConfirmationNotFound`, `TestNew_EmailNotVerified`, `TestNew_EnglishLocale`）がそれぞれ独自にハンドラーをセットアップしている。今回の変更は `create_test.go` の `setupHandler` から `createValidator` の削除に伴う修正が `new_test.go` にも波及しているが、`new_test.go` のヘルパー化は別のタスクとも考えられるため、確認したい。

  **修正案**:

  `new_test.go` も `create_test.go` と同じ `setupHandler` ヘルパーを共有するようにリファクタリングする（別PRでもよい）。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 今回のPRで `new_test.go` も `setupHandler` を使うようにリファクタリングする
  - [ ] 別PRでリファクタリングする（今回はスコープ外）
  - [ ] 現状のままにする（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3-9（create_account UseCase の移行）が作業計画書の設計方針に沿って正しく実装されている。

**良い点**:

- **作業計画書との整合性が高い**: UseCase がオーケストレーターとなり、データ取得 → バリデーション → 永続化の流れが UseCase 内に統合されている
- **エラーハンドリングパターンの一貫性**: `model.ValidationError` と `model.AppError` を使った `errors.As` パターンが、他の移行済み UseCase（create_suggestion 等）と一貫している
- **Handler の薄型化**: Handler は HTTP 入出力変換に徹しており、バリデーターやポリシーへの直接依存が完全に除去されている
- **Validator の `(data, error)` パターン**: Result 型が廃止され、Go の慣習的なエラー返却パターンに変更されている
- **テストの大幅な簡素化**: `create_test.go` で `setupHandler` と `createConfirmedEmailConfirmation` ヘルパーを導入し、テストのボイラープレートが大幅に削減されている（-575行）
- **テストカバレッジ**: UseCase テストに `EmailConfirmationNotFound`、`EmailNotConfirmed`、`ValidationError` のケースが追加されている
- **i18n 対応**: 新しいエラーメッセージ `error_email_not_confirmed` が日英両方のロケールファイルに追加されている

**軽微な指摘**:

- `new_test.go` が `create_test.go` と異なり未リファクタリングだが、今回のタスクのスコープとしては妥当
