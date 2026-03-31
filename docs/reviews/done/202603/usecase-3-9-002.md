# コードレビュー: usecase-3-9

## レビュー情報

| 項目                       | 内容                                                               |
| -------------------------- | ------------------------------------------------------------------ |
| レビュー日                 | 2026-03-30                                                         |
| 対象ブランチ               | usecase-3-9                                                        |
| ベースブランチ             | usecase-3-8                                                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク 3-9） |
| 変更ファイル数             | 13 ファイル（レビュー済みドキュメントを除くと 12 ファイル）        |
| 変更行数（実装）           | +151 / -149 行                                                     |
| 変更行数（テスト）         | +226 / -843 行                                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/create_account.go`
- [x] `go/internal/validator/account.go`
- [x] `go/internal/handler/account/create.go`
- [x] `go/internal/handler/account/handler.go`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/templates/pages/account/new.templ`
- [x] `go/internal/templates/pages/account/new_templ.go`（自動生成）
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/usecase/create_account_test.go`
- [x] `go/internal/validator/account_test.go`
- [x] `go/internal/handler/account/create_test.go`
- [x] `go/internal/handler/account/new_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`（タスク 3-9 にチェック）
- [x] `docs/reviews/done/usecase-3-9-001.md`（前回レビュー）

## ファイルごとのレビュー結果

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3-9（create_account UseCase の移行）は、作業計画書の設計方針に正確に沿って実装されている。

**良かった点**:

- **設計との整合性**: 作業計画書の「検討事項 1〜5」で確定した方針（`model.ValidationError` / `model.AppError` のエラー型、Validator の `(data, error)` 返し、UseCase がオーケストレーターとして統括）が正確に反映されている
- **UseCase の処理順序**: 「1. データ取得 → 2. バリデーション → 3. 永続化」の順序が作業計画書の検討事項 4 のパターンに沿っている。トランザクション開始前にデータ取得とバリデーションを完了し、トランザクション内は永続化のみ行っている
- **Handler の薄さ**: Handler はフォームパース → UseCase 呼び出し → エラーハンドリング/リダイレクトのみに徹しており、「HTTP の入出力変換に徹する」方針に合致
- **エラーハンドリング**: `handleCreateError` メソッドで `errors.As` パターンを使い、`ValidationError` → フォーム再描画（422）、`AppError` → コードに応じた処理（リダイレクト）、素の `error` → 500 という使い分けが明確
- **Validator のシンプル化**: Result 型を廃止し `error` 返しに変更。`emailConfirmationRepo` への依存を Validator から UseCase に移動し、Validator の責務が形式バリデーション + アットネーム重複チェックのみに絞られた
- **テストの充実**: UseCase テストに `EmailConfirmationNotFound`、`EmailNotConfirmed`、`ValidationError` の異常系テストが追加され、移行前は Validator テストでカバーしていたケースが UseCase テストに正しく移動している
- **テストのリファクタリング**: Handler テストで `setupHandler` ヘルパーと `createConfirmedEmailConfirmation` ヘルパーを導入し、大量の重複コードを削除（-843 行）しつつテストカバレッジを維持
- **i18n 対応**: 新しいエラーメッセージ `error_email_not_confirmed` が日英両方に追加されている
