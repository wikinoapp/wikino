# Go への移行 (パスワードリセット機能編) 仕様書

## 概要

Go 版 Wikino にパスワードリセット機能を実装します。ユーザーはメールアドレスを入力してパスワードリセットを申請し、メールで送信されたリンクから新しいパスワードを設定できます。

**目的**:

- Rails 版から Go 版への段階的移行として、パスワードリセット機能を実装する
- ユーザーが Go 版でパスワードをリセットできるようにする

**背景**:

- パスワードリセット機能はログイン機能と密接に関連している
- ログイン機能（[go.md](./go.md)）の実装完了後に着手する

**関連仕様書**:

- [Go への移行 (ログイン機能編)](./go.md) - ログイン機能の仕様書

## 要件

### 機能要件

- ユーザーはメールアドレスを入力してパスワードリセットを申請できる
- 申請後、「メールを送信しました」というメッセージを表示する（ユーザーが存在しない場合も同じメッセージを表示）
- 登録済みのメールアドレスには、パスワードリセット用のリンクがメールで送信される
- ユーザーはメールのリンクをクリックして新しいパスワードを設定できる
- パスワードリセットトークンは 1 時間で有効期限切れになる
- パスワードリセット成功後、ログインページにリダイレクトされる
- 無効なトークン（期限切れ、使用済み、存在しない）の場合はエラーページを表示する

### 非機能要件

#### セキュリティ

- パスワードリセットトークンは SHA256 でハッシュ化してデータベースに保存
- パスワードリセットの Rate Limiting を実施（IP 単位: 5 回/時間、メールアドレス単位: 3 回/時間）
- パスワードリセット申請時、ユーザーの存在有無を漏らさない（常に成功メッセージを表示）
- Cloudflare Turnstile による Bot 対策を実施
- CSRF 対策を実施する（全フォームに CSRF トークンを含める）

#### 国際化

- 日本語と英語の両言語に対応
- エラーメッセージ、フラッシュメッセージ、フォームラベルを国際化

## 設計

### 技術スタック

- **パスワードリセットトークン**: `crypto/rand` + `crypto/sha256`
- **パスワードハッシュ化**: `golang.org/x/crypto/bcrypt`
- **HTTP ルーター**: `chi/v5`
- **テンプレート**: `templ`
- **DB アクセス**: `sqlc`
- **Bot 対策**: Cloudflare Turnstile
- **メール送信**: `resend-go/v2`（Resend API）
- **Rate Limiting**: PostgreSQL（スライディングウィンドウ方式、実装済み）

### データベース設計

#### password_reset_tokens テーブル（新規）

パスワードリセット機能用のトークンを管理するテーブルです。Go 版で新規作成します。

```sql
CREATE TABLE public.password_reset_tokens (
    id uuid DEFAULT public.generate_ulid() NOT NULL PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES public.users(id),
    token_digest character varying NOT NULL,
    expires_at timestamp(6) without time zone NOT NULL,
    used_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);

CREATE UNIQUE INDEX password_reset_tokens_token_digest_idx ON public.password_reset_tokens (token_digest);
CREATE INDEX password_reset_tokens_user_id_idx ON public.password_reset_tokens (user_id);
```

**設計ポイント**:

- `token_digest`: 平文トークンを SHA256 でハッシュ化して保存（セキュリティ対策）
- `expires_at`: トークンの有効期限（作成から 1 時間後）
- `used_at`: トークン使用日時（NULL の場合は未使用）
- 平文トークンはメールで送信し、DB にはハッシュ化した値のみを保存

### API 設計（ルーティング）

| URL               | メソッド | ハンドラー              | 説明                           |
| ----------------- | -------- | ----------------------- | ------------------------------ |
| `/password/reset` | GET      | `password_reset.New`    | パスワードリセット申請フォーム |
| `/password/reset` | POST     | `password_reset.Create` | パスワードリセット申請処理     |
| `/password/edit`  | GET      | `password.Edit`         | 新パスワード入力フォーム       |
| `/password`       | PATCH    | `password.Update`       | パスワード更新処理             |

### コード設計

#### ディレクトリ構造

```
internal/
├── handler/
│   ├── password_reset/
│   │   ├── handler.go      # Handler構造体と依存性
│   │   ├── new.go          # GET /password/reset
│   │   ├── create.go       # POST /password/reset
│   │   ├── validator.go    # バリデーション（形式チェック）
│   │   └── validator_test.go
│   └── password/
│       ├── handler.go      # Handler構造体と依存性
│       ├── edit.go         # GET /password/edit
│       ├── update.go       # PATCH /password
│       ├── validator.go    # バリデーション（形式チェック + トークン検証）
│       └── validator_test.go
├── usecase/
│   ├── create_password_reset_token.go # パスワードリセットトークン作成
│   └── update_password_reset.go       # パスワード更新（リセット経由）
├── repository/
│   └── password_reset_token_repository.go  # パスワードリセットトークン CRUD
├── ratelimit/
│   └── limiter.go          # Rate Limiting（Redis ベース）
├── email/
│   └── sender.go           # メール送信（Sender インターフェース、ResendSender）
├── password_reset/
│   └── token.go            # トークン生成・ハッシュ化ユーティリティ
└── templates/
    └── pages/
        └── password/
            ├── reset.templ      # パスワードリセット申請フォーム
            ├── reset_sent.templ # 申請完了ページ
            └── edit.templ       # 新パスワード入力フォーム
```

#### 主要な構造体

**Handler（password_reset/handler.go）**

```go
type Handler struct {
    cfg                *config.Config
    userRepo           *repository.UserRepository
    sessionManager     *session.Manager
    limiter            *ratelimit.Limiter
    turnstileClient    *turnstile.Client
    createTokenUseCase *usecase.CreatePasswordResetTokenUsecase
}
```

**CreateValidator（password_reset/validator.go）**

```go
// CreateValidator はパスワードリセット申請のバリデーションを行う
type CreateValidator struct{}

// NewCreateValidator は CreateValidator を生成する
func NewCreateValidator() *CreateValidator {
    return &CreateValidator{}
}

// CreateValidatorInput はバリデーションの入力パラメータ
type CreateValidatorInput struct {
    Email string
}

// CreateValidatorResult はバリデーションの結果
type CreateValidatorResult struct {
    FormErrors *session.FormErrors
}

// Validate はバリデーションを行う
func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
    formErrors := session.NewFormErrors()

    // メールアドレス必須チェック
    if input.Email == "" {
        formErrors.AddFieldError("email", templates.T(ctx, "error_required"))
        return &CreateValidatorResult{FormErrors: formErrors}
    }

    // フォーマットチェック
    if !emailRegex.MatchString(input.Email) {
        formErrors.AddFieldError("email", templates.T(ctx, "error_invalid_email_format"))
    }

    return &CreateValidatorResult{FormErrors: formErrors}
}
```

**Handler（password/handler.go）**

```go
type Handler struct {
    cfg                          *config.Config
    passwordResetTokenRepository *repository.PasswordResetTokenRepository
    sessionManager               *session.Manager
    limiter                      *ratelimit.Limiter
    updatePasswordUseCase        *usecase.UpdatePasswordResetUsecase
}
```

**UpdateValidator（password/validator.go）**

```go
// UpdateValidator はパスワード更新のバリデーションを行う
// 形式バリデーションとトークン検証（DB検索）を統合
type UpdateValidator struct {
    passwordResetTokenRepo *repository.PasswordResetTokenRepository
}

// NewUpdateValidator は UpdateValidator を生成する
func NewUpdateValidator(passwordResetTokenRepo *repository.PasswordResetTokenRepository) *UpdateValidator {
    return &UpdateValidator{
        passwordResetTokenRepo: passwordResetTokenRepo,
    }
}

// UpdateValidatorInput はバリデーションの入力パラメータ
type UpdateValidatorInput struct {
    Token                string
    Password             string
    PasswordConfirmation string
}

// UpdateValidatorResult はバリデーションの結果
type UpdateValidatorResult struct {
    TokenID    string               // 検証成功時のトークンID（UseCaseに渡す）
    UserID     string               // 検証成功時のユーザーID（UseCaseに渡す）
    FormErrors *session.FormErrors
}

// Validate はバリデーションを行う
func (v *UpdateValidator) Validate(ctx context.Context, input UpdateValidatorInput) *UpdateValidatorResult {
    formErrors := session.NewFormErrors()

    // 形式バリデーション
    // トークン必須チェック
    if input.Token == "" {
        formErrors.AddFieldError("token", templates.T(ctx, "error_required"))
    }

    // パスワード必須チェック
    if input.Password == "" {
        formErrors.AddFieldError("password", templates.T(ctx, "error_required"))
    }

    // パスワード確認必須チェック
    if input.PasswordConfirmation == "" {
        formErrors.AddFieldError("password_confirmation", templates.T(ctx, "error_required"))
    }

    // パスワード文字数チェック
    if len(input.Password) > 0 && len(input.Password) < 8 {
        formErrors.AddFieldError("password", templates.T(ctx, "error_password_too_short"))
    }

    // パスワード確認一致チェック
    if input.Password != "" && input.PasswordConfirmation != "" && input.Password != input.PasswordConfirmation {
        formErrors.AddFieldError("password_confirmation", templates.T(ctx, "error_password_mismatch"))
    }

    if formErrors.HasErrors() {
        return &UpdateValidatorResult{FormErrors: formErrors}
    }

    // トークン検証（状態バリデーション）
    tokenDigest := password_reset.HashToken(input.Token)
    token, err := v.passwordResetTokenRepo.FindByTokenDigest(ctx, tokenDigest)
    if err != nil {
        // DBエラーはログに記録し、ユーザーにはシステムエラーを表示
        formErrors.AddGlobalError(templates.T(ctx, "error_system"))
        return &UpdateValidatorResult{FormErrors: formErrors}
    }
    if token == nil {
        formErrors.AddGlobalError(templates.T(ctx, "error_token_invalid"))
        return &UpdateValidatorResult{FormErrors: formErrors}
    }
    if token.IsUsed() {
        formErrors.AddGlobalError(templates.T(ctx, "error_token_used"))
        return &UpdateValidatorResult{FormErrors: formErrors}
    }
    if token.IsExpired() {
        formErrors.AddGlobalError(templates.T(ctx, "error_token_expired"))
        return &UpdateValidatorResult{FormErrors: formErrors}
    }

    return &UpdateValidatorResult{
        TokenID: token.ID,
        UserID:  token.UserID,
    }
}
```

**CreatePasswordResetTokenUsecase（usecase/create_password_reset_token.go）**

```go
type CreatePasswordResetTokenUsecase struct {
    cfg                        *config.Config
    db                         *sql.DB
    passwordResetTokenRepo     *repository.PasswordResetTokenRepository
    inserter                   JobInserter // river ジョブのエンキュー用
}

// NewCreatePasswordResetTokenUsecase は CreatePasswordResetTokenUsecase を生成する
func NewCreatePasswordResetTokenUsecase(
    cfg *config.Config,
    db *sql.DB,
    passwordResetTokenRepo *repository.PasswordResetTokenRepository,
    inserter JobInserter,
) *CreatePasswordResetTokenUsecase

// CreatePasswordResetTokenInput は入力パラメータ
type CreatePasswordResetTokenInput struct {
    UserID string
    Email  string
    Locale string
}

// CreatePasswordResetTokenOutput は出力パラメータ
type CreatePasswordResetTokenOutput struct {
    TokenID string
}

// Execute はパスワードリセットトークンを生成し、メール送信ジョブをエンキューする
func (uc *CreatePasswordResetTokenUsecase) Execute(ctx context.Context, input CreatePasswordResetTokenInput) (*CreatePasswordResetTokenOutput, error)
```

**UpdatePasswordResetUsecase（usecase/update_password_reset.go）**

```go
type UpdatePasswordResetUsecase struct {
    db                         *sql.DB
    passwordResetTokenRepo     *repository.PasswordResetTokenRepository
    userPasswordRepo           *repository.UserPasswordRepository
}

// NewUpdatePasswordResetUsecase は UpdatePasswordResetUsecase を生成する
func NewUpdatePasswordResetUsecase(
    db *sql.DB,
    passwordResetTokenRepo *repository.PasswordResetTokenRepository,
    userPasswordRepo *repository.UserPasswordRepository,
) *UpdatePasswordResetUsecase

// UpdatePasswordResetInput は入力パラメータ
// トークン検証は validator.go で行い、検証済みの TokenID と UserID を受け取る
type UpdatePasswordResetInput struct {
    TokenID     string  // validator.go で検証済みのトークンID
    UserID      string  // validator.go で検証済みのユーザーID
    NewPassword string
}

// UpdatePasswordResetOutput は出力パラメータ
type UpdatePasswordResetOutput struct {
    UserID string
}

// Execute はパスワードを更新し、トークンを使用済みにマークする
// トークン検証は validator.go で行うため、UseCase では永続化のみ行う
func (uc *UpdatePasswordResetUsecase) Execute(ctx context.Context, input UpdatePasswordResetInput) (*UpdatePasswordResetOutput, error)
```

### パスワードリセットフロー

```
1. ユーザーが GET /password/reset にアクセス
   ├── 認証済みの場合 → /home にリダイレクト
   └── 未認証の場合 → パスワードリセット申請フォーム表示

2. ユーザーがフォーム送信 (POST /password/reset)
   ├── CSRF トークン検証
   ├── Turnstile 検証
   │   └── 失敗の場合 → エラー表示
   ├── フォームバリデーション
   │   └── メールアドレス: 必須
   ├── Rate Limiting チェック
   │   ├── IP 単位: 5 回/時間
   │   └── メールアドレス単位: 3 回/時間
   ├── ユーザー検索 (email で検索、discarded_at IS NULL)
   │   └── 見つからない場合も成功ページを表示（セキュリティ対策）
   ├── ユーザーが存在する場合のみ:
   │   ├── 既存の未使用トークンを削除
   │   ├── 新しいトークンを生成 (32バイト + Base64 URL-safe)
   │   ├── トークンを SHA256 でハッシュ化して DB に保存
   │   ├── 有効期限: 1 時間
   │   └── パスワードリセットメールを送信
   └── 常に成功ページを表示（ユーザーの存在を明かさない）

3. ユーザーがメールのリンクをクリック (GET /password/edit?token=xxx)
   ├── Rate Limiting チェック (トークン検証: 10 回/時間/IP)
   ├── トークンをハッシュ化して DB から検索
   │   └── 見つからない場合 → エラーページ表示
   ├── トークンの有効性チェック
   │   ├── 使用済みの場合 → エラーページ表示
   │   └── 有効期限切れの場合 → エラーページ表示
   └── 新パスワード入力フォーム表示

4. ユーザーがフォーム送信 (PATCH /password)
   ├── CSRF トークン検証
   ├── validator.go でバリデーション
   │   ├── 形式バリデーション
   │   │   ├── トークン: 必須
   │   │   ├── パスワード: 必須、8文字以上
   │   │   └── パスワード確認: 必須、パスワードと一致
   │   └── トークン検証（状態バリデーション）
   │       ├── トークンをハッシュ化して DB から検索
   │       ├── トークンの存在チェック
   │       ├── トークンの使用済みチェック
   │       └── トークンの有効期限チェック
   ├── UseCase で永続化（検証済みの TokenID, UserID を受け取る）
   │   ├── パスワードを bcrypt でハッシュ化
   │   ├── ユーザーのパスワードを更新
   │   └── トークンを使用済みにマーク
   ├── フラッシュメッセージ設定
   └── /sign_in にリダイレクト
```

### セキュリティ設計

#### パスワードリセットトークン生成

パスワードリセット用のトークンを生成し、SHA256 でハッシュ化して保存：

```go
// トークン生成（32バイト + Base64 URL-safe エンコード = 約43文字）
func GenerateToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}

// トークンのハッシュ化（DB保存用）
func HashToken(token string) string {
    hash := sha256.Sum256([]byte(token))
    return hex.EncodeToString(hash[:])
}
```

**セキュリティポイント**:

- **平文トークン**: メールで送信（URL パラメータとして）
- **ハッシュ化トークン**: データベースに保存（平文は保存しない）
- **有効期限**: 1 時間
- **使用回数**: 1 回のみ（使用後は `used_at` を設定）
- **Rate Limiting**: IP 単位・メールアドレス単位で制限

### テスト戦略

- **ハンドラーテスト**: HTTP リクエスト・レスポンスの統合テスト
- **ユースケーステスト**: トークン生成・パスワード更新ロジックのテスト
- **リポジトリテスト**: DB 操作のテスト
- **Rate Limiting テスト**: レート制限の動作テスト

テストでは実際の PostgreSQL データベースを使用し、トランザクションで分離します。

## タスクリスト

### フェーズ 8: パスワードリセット機能の実装

- [x] **8-1**: Rate Limiting の実装（実装済み）

  - `internal/ratelimit/limiter.go` - PostgreSQL ベースのスライディングウィンドウ方式
  - `db/migrations/20260202160000_create_rate_limits.sql` - マイグレーション
  - `db/queries/rate_limits.sql` - sqlc クエリ定義
  - IP 単位・メールアドレス単位のレート制限に対応
  - **実装済みファイル**: `/workspace/go/internal/ratelimit/limiter.go`

- [x] **8-2**: パスワードリセットトークンのユーティリティ実装

  - `internal/password_reset/token.go`
  - トークン生成（32 バイト + Base64 URL-safe）
  - トークンハッシュ化（SHA256）
  - **実装済みファイル**: `/workspace/go/internal/password_reset/token.go`
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 80 行（実装 30 行 + テスト 50 行）

- [x] **8-3**: メール送信機能の実装（実装済み）

  - `internal/email/sender.go` - `Sender` インターフェース、`ResendSender`、`NoopSender`
  - Resend API を使用したメール送信
  - メールテンプレートは `internal/templates/emails/` に配置
  - **実装済みファイル**: `/workspace/go/internal/email/sender.go`

- [x] **8-4**: password_reset_tokens テーブルのマイグレーション

  - `db/migrations/20260204160000_create_password_reset_tokens.sql`
  - `db/queries/password_reset_tokens.sql`
  - `internal/model/password_reset_token.go`
  - `internal/repository/password_reset_token_repository.go`
  - `internal/repository/password_reset_token_repository_test.go`
  - `internal/testutil/password_reset_token_builder.go`
  - **実装済みファイル数**: 6 ファイル（実装 5 + テスト 1）

- [x] **8-5**: パスワードリセットトークン作成ユースケースの実装

  - `internal/usecase/create_password_reset_token.go`
  - `internal/usecase/create_password_reset_token_test.go`
  - `internal/worker/send_password_reset.go`（ワーカー引数定義）
  - 既存の未使用トークンを削除
  - 新しいトークンを生成・保存
  - パスワードリセットメールを送信（ジョブをエンキュー）
  - **実装済みファイル数**: 3 ファイル（実装 2 + テスト 1）

- [x] **8-6**: パスワード更新ユースケースの実装

  - `internal/usecase/update_password_reset.go`
  - `internal/usecase/update_password_reset_test.go`
  - `db/queries/user_passwords.sql`（UpdateUserPasswordDigest クエリ追加）
  - `internal/repository/user_password_repository.go`（UpdatePasswordDigest メソッド追加）
  - `internal/testutil/user_password_builder.go`
  - トークン検証（有効期限、使用済みチェック）
  - パスワードを bcrypt でハッシュ化して更新
  - トークンを使用済みにマーク
  - **実装済みファイル数**: 5 ファイル（実装 4 + テスト 1）

- [x] **8-7**: パスワードリセット申請フォームテンプレートの実装

  - `internal/templates/pages/password/reset.templ`
  - `internal/templates/pages/password/reset_sent.templ`
  - **参考ファイル**: `/annict/go/internal/templates/pages/password/reset.templ`
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 100 行（実装 100 行 + テスト 0 行）
  - **実装済みファイル**: `/workspace/go/internal/templates/pages/password/reset.templ`, `/workspace/go/internal/templates/pages/password/reset_sent.templ`

- [x] **8-8**: 新パスワード入力フォームテンプレートの実装

  - `internal/templates/pages/password/edit.templ`
  - **参考ファイル**: `/annict/go/internal/templates/pages/password/edit.templ`
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 80 行（実装 80 行 + テスト 0 行）
  - **実装済みファイル**: `/workspace/go/internal/templates/pages/password/edit.templ`

- [x] **8-9**: パスワードリセット申請ハンドラーの実装

  - `internal/handler/password_reset/handler.go`
  - `internal/handler/password_reset/new.go`
  - `internal/handler/password_reset/create.go`
  - `internal/handler/password_reset/validator.go`
  - `internal/handler/password_reset/validator_test.go`
  - `internal/handler/password_reset/new_test.go`
  - `internal/handler/password_reset/create_test.go`
  - Turnstile 検証、Rate Limiting を実装
  - **参考ファイル**: `/annict/go/internal/handler/password_reset/`
  - **実装済みファイル数**: 7 ファイル（実装 4 + テスト 3）

- [x] **8-10**: パスワード更新ハンドラーの実装

  - `internal/handler/password/handler.go`
  - `internal/handler/password/edit.go`
  - `internal/handler/password/update.go`
  - `internal/handler/password/validator.go`
  - `internal/handler/password/validator_test.go`
  - `internal/handler/password/edit_test.go`
  - `internal/handler/password/update_test.go`
  - **validator.go でのトークン検証**（UseCase から分離）:
    - トークン存在チェック（平文トークンをハッシュ化して DB 検索）
    - トークン使用済みチェック（`IsUsed()`）
    - トークン有効期限チェック（`IsExpired()`）
    - 検証成功時は TokenID と UserID を返し、UseCase に渡す
  - パスワード形式バリデーション（必須、8 文字以上、確認一致）
  - **参考ファイル**: `/annict/go/internal/handler/password/`
  - **実装済みファイル数**: 7 ファイル（実装 4 + テスト 3）

- [x] **8-11**: ルーティング設定とリバースプロキシの更新

  - `/password/reset`, `/password/edit`, `/password` のルーティング追加
  - リバースプロキシのホワイトリスト更新
  - **実装済みファイル**: `cmd/server/main.go`, `internal/middleware/reverse_proxy.go`

- [x] **8-12**: 国際化（I18n）の追加

  - パスワードリセット関連の翻訳キーを追加
  - `internal/i18n/locales/ja.toml`
  - `internal/i18n/locales/en.toml`
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 60 行（実装 60 行 + テスト 0 行）

## 参考資料

- **Annict Go 版パスワードリセット実装**: `/annict/go/internal/handler/password_reset/`, `/annict/go/internal/handler/password/`
- **Rails 版 Wikino パスワードリセット実装**: `/workspace/rails/app/controllers/password_resets/`, `/workspace/rails/app/controllers/passwords/`
- **Go CLAUDE.md**: `/workspace/go/CLAUDE.md`
- [Resend API](https://resend.com/docs)
