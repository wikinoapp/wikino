# コードレビュー: sign-up-7-1

## レビュー情報

| 項目              | 内容                       |
| ----------------- | -------------------------- |
| レビュー日        | 2026-02-03                 |
| 対象ブランチ      | sign-up-7-1                |
| ベースブランチ    | go                         |
| 変更ファイル数    | 90 ファイル                |
| 変更行数（実装）  | 約 +10000 / -100 行        |
| 変更行数（テスト）| 約 +2500 行                |

## 参照するガイドライン

### 共通ガイドライン

- [@CLAUDE.md](/workspace/CLAUDE.md) - プロジェクト全体のガイド

### Go版ガイドライン

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル（主要なもの）

- [x] `go/internal/handler/account/handler.go`
- [x] `go/internal/handler/account/new.go`
- [x] `go/internal/handler/account/create.go`
- [x] `go/internal/handler/account/request.go`
- [x] `go/internal/handler/email_confirmation/handler.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/email_confirmation/edit.go`
- [x] `go/internal/handler/email_confirmation/update.go`
- [x] `go/internal/handler/email_confirmation/request.go`
- [x] `go/internal/handler/sign_up/handler.go`
- [x] `go/internal/handler/sign_up/new.go`
- [x] `go/internal/usecase/create_account.go`
- [x] `go/internal/usecase/send_email_confirmation.go`
- [x] `go/internal/usecase/verify_email_confirmation.go`
- [x] `go/internal/repository/email_confirmation_repository.go`
- [x] `go/internal/repository/user_repository.go`
- [x] `go/internal/repository/user_password_repository.go`
- [x] `go/internal/worker/client.go`
- [x] `go/internal/worker/send_email_confirmation.go`
- [x] `go/internal/ratelimit/limiter.go`
- [x] `go/internal/model/email_confirmation.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/handler/account/create_test.go`
- [x] `go/internal/handler/account/new_test.go`
- [x] `go/internal/handler/account/request_test.go`
- [x] `go/internal/handler/email_confirmation/create_test.go`
- [x] `go/internal/handler/email_confirmation/edit_test.go`
- [x] `go/internal/handler/email_confirmation/update_test.go`
- [x] `go/internal/usecase/create_account_test.go`
- [x] `go/internal/usecase/send_email_confirmation_test.go`
- [x] `go/internal/usecase/verify_email_confirmation_test.go`
- [x] `go/internal/repository/email_confirmation_repository_test.go`
- [x] `go/internal/worker/send_email_confirmation_test.go`
- [x] `go/internal/ratelimit/limiter_test.go`

### 設定・その他

- [x] `go/db/migrations/20260202060000_add_river_tables.sql`
- [x] `go/db/migrations/20260202160000_create_rate_limits.sql`
- [x] `go/db/queries/email_confirmations.sql`
- [x] `go/db/queries/rate_limits.sql`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/account/new.templ`
- [x] `go/internal/templates/pages/email_confirmation/edit.templ`

## ファイルごとのレビュー結果

### `go/internal/handler/account/handler.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - Handler構造体の定義

**問題点・改善提案**:

- 問題なし。Handler構造体と依存性注入が適切に定義されている。フィールド数も6個で8個以下のガイドラインに従っている。

---

### `go/internal/handler/account/new.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ファイル命名規則、メソッド命名規則
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - CSRF対策

**問題点・改善提案**:

- 問題なし。`new.go` ファイルに `New` メソッドが実装されており、命名規則に従っている。CSRFトークンも正しく取得・テンプレートに渡している。

---

### `go/internal/handler/account/create.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ファイル命名規則
- [@go/CLAUDE.md#ログ出力](/workspace/go/CLAUDE.md) - ログ出力
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - CSRF対策、エラーハンドリング

**問題点・改善提案**:

- 問題なし。`slog` を正しく使用している。エラー時のログ出力も適切。バリデーションエラー時に422ステータスを返すのも良い実装。

---

### `go/internal/handler/account/request.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - Request DTOパターン

**問題点・改善提案**:

- 問題なし。形式バリデーションのみ行い、DBアクセスは行っていない。i18nを使用してメッセージを国際化している。

---

### `go/internal/handler/email_confirmation/handler.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - Handler構造体

**問題点・改善提案**:

- 問題なし。フィールド数も7個で適切。

---

### `go/internal/handler/email_confirmation/create.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ファイル命名規則
- [@go/CLAUDE.md#ログ出力](/workspace/go/CLAUDE.md) - ログ出力
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - CSRF対策、Bot対策

**問題点・改善提案**:

- 問題なし。Turnstile検証によるBot対策が実装されている。ログ出力も `slog` を正しく使用。

---

### `go/internal/handler/email_confirmation/request.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - Request DTOの配置
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - Request DTOパターン

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#Request DTOの配置]**: `request.go` 内に `CreateRequest` と `UpdateRequest` の2つの構造体が定義されている。ガイドラインでは「1リソース1リクエスト構造体」とされているが、同一リソース（email_confirmation）の異なるアクション（Create、Update）に対する構造体なので許容範囲内。

  ただし、ガイドラインには「複数のリクエスト構造体が必要な場合は、**新しいリソースディレクトリを作成する**」とある。厳密に従うなら、`email_confirmation_verify/` などに分けることも検討できるが、現状の実装は実用的で問題ない。

---

### `go/internal/usecase/create_account.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - ユースケース、Repository の WithTx パターン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - パスワード管理

**問題点・改善提案**:

- 問題なし。トランザクション管理が適切で、`WithTx` パターンを正しく使用。bcryptでパスワードをハッシュ化している。エラー定義も明確。

---

### `go/internal/usecase/send_email_confirmation.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - ユースケース
- [@go/CLAUDE.md#ログ出力](/workspace/go/CLAUDE.md) - ログ出力

**問題点・改善提案**:

- 問題なし。インターフェース `EmailConfirmationEnqueuer` を使ってWorkerへの依存を抽象化している点が良い。

---

### `go/internal/usecase/verify_email_confirmation.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - ユースケース

**問題点・改善提案**:

- 問題なし。エラー定義が明確で、`errors.Is()` でのエラー判定が可能。確認コードの大文字小文字を区別しない実装（`strings.EqualFold`）は適切。

---

### `go/internal/repository/email_confirmation_repository.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Repository層
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - ModelとRepositoryは1:1の関係

**問題点・改善提案**:

- 問題なし。`WithTx` メソッドが実装されており、トランザクション対応している。`toModel` メソッドでQuery結果をModelに変換している。

---

### `go/internal/worker/send_email_confirmation.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - ワーカー（Worker）
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Worker の依存関係

**問題点・改善提案**:

- 問題なし。Workerがtemplates（`email_confirmation`）に依存しているが、これはメールレンダリングのための例外として許可されている。

---

### `go/internal/worker/client.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - ワーカー（Worker）

**問題点・改善提案**:

- 問題なし。Riverクライアントのラッパーとして適切に実装されている。`EnqueueEmailConfirmation` メソッドでジョブをエンキューする処理も問題なし。

---

### `go/internal/ratelimit/limiter.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - コーディング規約

**問題点・改善提案**:

- 問題なし。PostgreSQLベースのRate Limiterとして適切に実装。`WithTx` メソッドも実装されている。

---

### `go/internal/model/email_confirmation.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Model

**問題点・改善提案**:

- 問題なし。ドメインモデルとして純粋に実装されている。`IsExpired()` と `IsSucceeded()` メソッドはビジネスロジックを適切にカプセル化している。

---

### `go/internal/middleware/reverse_proxy.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - リバースプロキシによる段階的移行

**問題点・改善提案**:

- 問題なし。ホワイトリスト方式でGo版で処理するパスを管理している。`goHandledPaths` に新しいルート（`/email_confirmation`, `/accounts`）が追加されている。

---

### `go/cmd/server/main.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - エントリポイント

**問題点・改善提案**:

- 問題なし。リポジトリ、ユースケース、ハンドラーの初期化が適切に行われている。ルーティングも認証ミドルウェアを使って適切に保護されている。

---

### `go/internal/templates/pages/sign_up/new.templ`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレート
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - CSRF対策

**問題点・改善提案**:

- 問題なし。CSRFトークンがhiddenフィールドとして含まれている。Turnstileコンポーネントも使用されている。

---

### `go/internal/templates/pages/account/new.templ`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレート
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - CSRF対策

**問題点・改善提案**:

- 問題なし。構造体ベースのパターン（`NewPageData`）を使用している。CSRFトークンも含まれている。

---

### `go/db/migrations/20260202160000_create_rate_limits.sql`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#カラム定義のガイドライン](/workspace/go/CLAUDE.md) - カラム定義のガイドライン

**問題点・改善提案**:

- 問題なし。`VARCHAR`（長さ指定なし）と`TIMESTAMP WITH TIME ZONE`を使用しており、ガイドラインに従っている。

---

### `go/internal/usecase/create_account_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- 問題なし。実データベースを使用してテストしている。成功ケース、メール未確認ケース、アットネーム重複ケースなどが網羅されている。

---

### `go/internal/i18n/locales/ja.toml`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#国際化（I18n）](/workspace/go/CLAUDE.md) - 国際化

**問題点・改善提案**:

- 問題なし。サインアップ、メール確認、アカウント作成に関する翻訳が追加されている。

---

## 総合評価

**評価**: Approve

**総評**:

サインアップ機能の実装が包括的かつ高品質に行われています。

**良い点**:

1. **アーキテクチャの一貫性**: 3層アーキテクチャ（Presentation → Application → Domain/Infrastructure）が正しく守られている
2. **セキュリティ対策**: CSRF対策、Turnstile（Bot対策）、bcryptによるパスワードハッシュ化が適切に実装されている
3. **エラーハンドリング**: カスタムエラー（`ErrEmailNotConfirmed`, `ErrAtnameAlreadyTaken`など）を定義し、`errors.Is()` で判定できる設計
4. **トランザクション管理**: `WithTx` パターンを使用してRepository層でトランザクションを適切に管理
5. **テストカバレッジ**: 主要なユースケースとハンドラーに対してテストが実装されている
6. **国際化対応**: 日本語・英語の翻訳ファイルが追加されている
7. **コード品質**: `slog` の適切な使用、構造体ベースのテンプレート引数パターン、命名規則の遵守

**必須対応**:

- なし

**推奨対応**:

- [ ] **[@go/docs/handler-guide.md#Request DTOの配置]**: `email_confirmation/request.go` に2つのRequest構造体（`CreateRequest`, `UpdateRequest`）が定義されている。ガイドラインの厳密な解釈では別リソースに分けるべきだが、同一リソースの異なるアクションなので実用上は問題なし。将来的にリクエスト構造体が増える場合は分割を検討。
