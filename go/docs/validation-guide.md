# バリデーションガイド

このドキュメントは、Go 版 Wikino でのバリデーションのベストプラクティスを説明します。

## 概要

フォームからの入力値の検証は、`internal/validator/` パッケージの**バリデーター**で実装します。形式バリデーション（入力値の形式チェック）と状態バリデーション（DB を使った検証）を同じパッケージに配置することで、「どこに書くべきか」の判断コストをゼロにします。

### バリデーターの配置先

すべてのバリデーターは `internal/validator/` パッケージに配置します。形式バリデーションのみの場合も、状態バリデーションを含む場合も同じパッケージです。これにより：

- **判断コストがゼロ**: バリデーションは常に `internal/validator/` に配置する
- **進化に強い**: 形式バリデーションに後から DB チェックが追加されても、ファイル移動が不要
- **一箇所で把握できる**: バリデーションルールを確認したいとき `internal/validator/` だけ見ればよい
- **アーキテクチャの強制**: Handler パッケージから `repository` パッケージの import を完全に排除し、depguard で強制できる

### ファイル構成

```
internal/validator/
├── sign_in.go            # バリデーション（形式チェック + DBを使った検証）
├── sign_in_test.go       # バリデーションのテスト
├── password_reset.go     # バリデーション（形式チェックのみ）
└── password_reset_test.go # バリデーションのテスト

internal/handler/sign_in/
├── handler.go            # Handler構造体と依存性（Validatorを外部から受け取る）
├── new.go                # フォーム表示
└── create.go             # 作成処理

internal/handler/password_reset/
├── handler.go            # Handler構造体と依存性（Validatorを外部から受け取る）
├── new.go                # フォーム表示
└── create.go             # 作成処理
```

### バリデーションの分類

バリデーションは以下の 2 種類に分類されますが、同じファイル（`validator.go`）に実装します：

| 種類               | 責務                  | 特徴            |
| ------------------ | --------------------- | --------------- |
| 形式バリデーション | 入力値の形式チェック  | DB アクセス不要 |
| 状態バリデーション | DB の状態を使った検証 | DB アクセス必要 |

### 構造体の命名規則

- **命名規則**: `{Resource}{Action}Validator`（例: `SignInCreateValidator`, `PasswordResetCreateValidator`）
- **コンストラクタ**: `New{Resource}{Action}Validator`（例: `NewSignInCreateValidator`）
- **1 つの構造体で両方のバリデーションを担当**: 形式バリデーションと状態バリデーションを `Validate` メソッド内で順次実行

### 状態バリデーションの配置場所

状態バリデーションは `internal/validator/` パッケージ内のバリデーターまたは UseCase のどちらかに配置します。

**判断基準**: **「検証失敗時に DB を更新する必要があるか？」**

| 検証失敗時の DB 更新 | 配置場所     | 理由                                               |
| -------------------- | ------------ | -------------------------------------------------- |
| 不要                 | validator.go | UseCase をシンプルに保つため                       |
| 必要                 | UseCase      | トランザクション内で検証と更新を行う必要があるため |

**validator.go で行うべき検証**:

| 検証内容                   | 失敗時の動作     | 理由                   |
| -------------------------- | ---------------- | ---------------------- |
| ユーザー存在チェック       | エラーメッセージ | DB 更新なし            |
| メールアドレス重複チェック | エラーメッセージ | DB 更新なし            |
| アットネーム重複チェック   | エラーメッセージ | DB 更新なし            |
| メール確認完了チェック     | エラーメッセージ | DB 更新なし            |
| コード一致チェック         | エラーメッセージ | DB 更新なし（※注参照） |
| パスワード照合             | エラーメッセージ | DB 更新なし            |

※注: コード検証で「試行回数インクリメント」が必要な場合は UseCase で行う

**UseCase で行うべき検証**:

| 検証内容           | 失敗時の動作           | 理由                   |
| ------------------ | ---------------------- | ---------------------- |
| ログインコード検証 | 試行回数インクリメント | 失敗時に DB 更新が必要 |

※注: 「リカバリーコード消費」や「トークン使用済みマーク」は検証成功後の処理であり、バリデーションではなく UseCase の永続化処理として扱う。検証自体は validator.go で行い、成功後に UseCase を呼び出す。

### エラー表示方法の使い分け

| エラー種類           | 表示方法            | 使い分け                                           |
| -------------------- | ------------------- | -------------------------------------------------- |
| **フィールドエラー** | `FormErrors.Fields` | 特定の入力フィールドに関連するエラー               |
| **グローバルエラー** | `FormErrors.Global` | フォーム全体に関連するエラー（同じページに留まる） |
| **Flash メッセージ** | `session.Flash`     | リダイレクト後に表示するメッセージ（成功/エラー）  |
| **ログのみ**         | `slog.Error`        | 開発者向け情報（ユーザーには一般メッセージを表示） |

**判断フローチャート**:

```
フォームを再表示する？
├─ Yes → FormErrors（Fields または Global）
│    └─ 特定フィールドに関連？ → AddFieldError（例: ユーザー名重複）
│    └─ フォーム全体に関連？  → AddGlobalError（例: 確認コード不一致）
└─ No（リダイレクトする）→ Flash
     └─ 成功 → FlashSuccess
     └─ エラー → FlashError
```

### メッセージの国際化

バリデーションメッセージは必ず `templates.T(ctx, "message_id")` を使用します。

## 実装例

### 基本的なバリデーター（`internal/validator/` パッケージ）

```go
// internal/validator/sign_in.go
package validator

import (
    "context"
    "errors"
    "net/mail"

    "github.com/wikinoapp/wikino/go/internal/auth"
    "github.com/wikinoapp/wikino/go/internal/model"
    "github.com/wikinoapp/wikino/go/internal/repository"
    "github.com/wikinoapp/wikino/go/internal/session"
    "github.com/wikinoapp/wikino/go/internal/templates"
)

// バリデーションのエラー定義
var (
    ErrUserNotFound    = errors.New("ユーザーが見つかりません")
    ErrInvalidPassword = errors.New("パスワードが正しくありません")
)

// SignInCreateValidator はサインインのバリデーションを行う
type SignInCreateValidator struct {
    userRepo *repository.UserRepository
}

// NewSignInCreateValidator は SignInCreateValidator を生成する
func NewSignInCreateValidator(userRepo *repository.UserRepository) *SignInCreateValidator {
    return &SignInCreateValidator{
        userRepo: userRepo,
    }
}

// SignInCreateValidatorInput はバリデーションの入力パラメータ
type SignInCreateValidatorInput struct {
    Email    string
    Password string
}

// SignInCreateValidatorResult はバリデーションの結果
type SignInCreateValidatorResult struct {
    User       *model.User
    FormErrors *session.FormErrors
    Err        error
}

// Validate はバリデーションを行う
func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) *SignInCreateValidatorResult {
    // 1. 形式バリデーション
    formErrors := session.NewFormErrors()

    if input.Email == "" {
        formErrors.AddFieldError("email", templates.T(ctx, "error_required"))
    } else if !isValidEmail(input.Email) {
        formErrors.AddFieldError("email", templates.T(ctx, "error_invalid_email_format"))
    }

    if input.Password == "" {
        formErrors.AddFieldError("password", templates.T(ctx, "error_required"))
    }

    if formErrors.HasErrors() {
        return &SignInCreateValidatorResult{FormErrors: formErrors}
    }

    // 2. 状態バリデーション（DB検証）
    user, err := v.userRepo.GetByEmailForSignIn(ctx, input.Email)
    if err != nil {
        if err == repository.ErrNotFound {
            formErrors.AddGlobalError(templates.T(ctx, "error_invalid_credentials"))
            return &SignInCreateValidatorResult{FormErrors: formErrors, Err: ErrUserNotFound}
        }
        return &SignInCreateValidatorResult{Err: err}
    }

    // パスワード検証
    if err := auth.CheckPassword(user.PasswordDigest, input.Password); err != nil {
        formErrors.AddGlobalError(templates.T(ctx, "error_invalid_credentials"))
        return &SignInCreateValidatorResult{FormErrors: formErrors, Err: ErrInvalidPassword}
    }

    return &SignInCreateValidatorResult{User: user}
}

func isValidEmail(email string) bool {
    _, err := mail.ParseAddress(email)
    return err == nil
}
```

### 形式バリデーションのみのバリデーター

DB を使った検証が不要な場合でも、`internal/validator/` パッケージに配置します。

```go
// internal/validator/password_reset.go
package validator

import (
    "context"
    "regexp"

    "github.com/wikinoapp/wikino/go/internal/session"
    "github.com/wikinoapp/wikino/go/internal/templates"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// PasswordResetCreateValidator はパスワードリセット申請のバリデーションを行う
type PasswordResetCreateValidator struct{}

// NewPasswordResetCreateValidator は PasswordResetCreateValidator を生成する
func NewPasswordResetCreateValidator() *PasswordResetCreateValidator {
    return &PasswordResetCreateValidator{}
}

// PasswordResetCreateValidatorInput はバリデーションの入力パラメータ
type PasswordResetCreateValidatorInput struct {
    Email string
}

// PasswordResetCreateValidatorResult はバリデーションの結果
type PasswordResetCreateValidatorResult struct {
    FormErrors *session.FormErrors
}

// Validate はバリデーションを行う
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) *PasswordResetCreateValidatorResult {
    formErrors := session.NewFormErrors()

    // 必須チェック
    if input.Email == "" {
        formErrors.AddFieldError("email", templates.T(ctx, "error_required"))
        return &PasswordResetCreateValidatorResult{FormErrors: formErrors}
    }

    // フォーマットチェック
    if !emailRegex.MatchString(input.Email) {
        formErrors.AddFieldError("email", templates.T(ctx, "error_invalid_email_format"))
    }

    // 文字数制限
    if len(input.Email) > 255 {
        formErrors.AddFieldError("email", templates.T(ctx, "error_email_too_long"))
    }

    return &PasswordResetCreateValidatorResult{FormErrors: formErrors}
}
```

### 複数フィールドのバリデーター

```go
// internal/validator/password.go
package validator

import (
    "context"

    "github.com/wikinoapp/wikino/go/internal/session"
    "github.com/wikinoapp/wikino/go/internal/templates"
)

// PasswordUpdateValidator はパスワード更新のバリデーションを行う
type PasswordUpdateValidator struct{}

// NewPasswordUpdateValidator は PasswordUpdateValidator を生成する
func NewPasswordUpdateValidator() *PasswordUpdateValidator {
    return &PasswordUpdateValidator{}
}

// PasswordUpdateValidatorInput はバリデーションの入力パラメータ
type PasswordUpdateValidatorInput struct {
    Password             string
    PasswordConfirmation string
}

// PasswordUpdateValidatorResult はバリデーションの結果
type PasswordUpdateValidatorResult struct {
    FormErrors *session.FormErrors
}

// Validate はバリデーションを行う
func (v *PasswordUpdateValidator) Validate(ctx context.Context, input PasswordUpdateValidatorInput) *PasswordUpdateValidatorResult {
    formErrors := session.NewFormErrors()

    // 必須チェック
    if input.Password == "" {
        formErrors.AddFieldError("password", templates.T(ctx, "error_required"))
    }

    if input.PasswordConfirmation == "" {
        formErrors.AddFieldError("password_confirmation", templates.T(ctx, "error_required"))
    }

    // 文字数チェック
    if len(input.Password) > 0 && len(input.Password) < 8 {
        formErrors.AddFieldError("password", templates.T(ctx, "error_password_too_short"))
    }

    // パスワード一致チェック
    if input.Password != "" && input.PasswordConfirmation != "" && input.Password != input.PasswordConfirmation {
        formErrors.AddFieldError("password_confirmation", templates.T(ctx, "error_password_mismatch"))
    }

    return &PasswordUpdateValidatorResult{FormErrors: formErrors}
}
```

## ハンドラーでの使用

### 基本パターン

```go
// internal/handler/sign_in/create.go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // 入力データを作成
    input := validator.SignInCreateValidatorInput{
        Email:    r.FormValue("email"),
        Password: r.FormValue("password"),
    }

    // バリデーション実行
    result := h.validator.Validate(ctx, input)
    if result.FormErrors != nil && result.FormErrors.HasErrors() {
        h.renderForm(w, ctx, csrfToken, input.Email, result.FormErrors)
        return
    }
    if result.Err != nil {
        // システムエラー
        slog.ErrorContext(ctx, "バリデーションでエラーが発生", "error", result.Err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // 認証成功後の処理（UseCase）
    ucResult, err := h.createSessionUC.Execute(ctx, usecase.CreateSessionInput{
        ActorID: result.User.ActorID,
        // ...
    })
    // ...
}
```

### Handler の依存性

Handler は Validator を外部から受け取ります。`main.go` で Validator を構築し、Handler に渡します。

```go
// internal/handler/sign_in/handler.go
type Handler struct {
    cfg             *config.Config
    sessionMgr      *session.Manager
    validator       *validator.SignInCreateValidator  // internal/validator パッケージ
    createSessionUC *usecase.CreateSessionUsecase
}

func NewHandler(
    cfg *config.Config,
    sessionMgr *session.Manager,
    signInValidator *validator.SignInCreateValidator,
    createSessionUC *usecase.CreateSessionUsecase,
) *Handler {
    return &Handler{
        cfg:             cfg,
        sessionMgr:      sessionMgr,
        validator:       signInValidator,
        createSessionUC: createSessionUC,
    }
}
```

```go
// cmd/server/main.go での構築
signInValidator := validator.NewSignInCreateValidator(userRepo)
signInHandler := sign_in.NewHandler(cfg, sessionMgr, signInValidator, createSessionUC)

passwordResetValidator := validator.NewPasswordResetCreateValidator()
passwordResetHandler := password_reset.NewHandler(cfg, sessionMgr, passwordResetValidator, riverClient)
```

## テスト

### テスト方針

バリデーションのテストは `validator_test.go` に統合して実装します。

| テスト対象           | ファイル            | 特徴                                   |
| -------------------- | ------------------- | -------------------------------------- |
| バリデーション全体   | `validator_test.go` | 形式・状態バリデーションを統合テスト   |
| ハンドラーの振る舞い | `handler_test.go`   | E2E テスト、正常系・代表的な異常系のみ |

**理由**:

- **シンプルな構成**: 1 つのファイルにすべてのバリデーションテストを集約
- **問題の特定**: テスト失敗時にどの検証の問題か即座に分かる
- **保守性向上**: テストファイルの管理が容易

### バリデーションのテスト

```go
// internal/validator/sign_in_test.go
func TestSignInCreateValidator_Validate(t *testing.T) {
    // 形式バリデーションのテスト
    t.Run("形式バリデーション", func(t *testing.T) {
        tests := []struct {
            name          string
            input         SignInCreateValidatorInput
            wantErrors    bool
            expectedField string
        }{
            {
                name: "有効な入力",
                input: SignInCreateValidatorInput{
                    Email:    "user@example.com",
                    Password: "password123",
                },
                wantErrors: false,
            },
            {
                name: "メールアドレスが空",
                input: SignInCreateValidatorInput{
                    Email:    "",
                    Password: "password123",
                },
                wantErrors:    true,
                expectedField: "email",
            },
            {
                name: "パスワードが空",
                input: SignInCreateValidatorInput{
                    Email:    "user@example.com",
                    Password: "",
                },
                wantErrors:    true,
                expectedField: "password",
            },
        }

        // DBアクセスなしでテスト（形式バリデーションのみ）
        v := NewSignInCreateValidator(nil)

        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                ctx := context.Background()
                ctx = templates.WithLocale(ctx, "ja")

                result := v.Validate(ctx, tt.input)

                if tt.wantErrors {
                    if result.FormErrors == nil || !result.FormErrors.HasErrors() {
                        t.Error("expected errors, got none")
                    }
                    if tt.expectedField != "" && !result.FormErrors.HasFieldError(tt.expectedField) {
                        t.Errorf("expected field error for %q", tt.expectedField)
                    }
                } else if tt.input.Email != "" && tt.input.Password != "" {
                    // 形式バリデーションのみの場合、DBアクセスでエラーになるのでスキップ
                }
            })
        }
    })

    // 状態バリデーションのテスト（DB必要）
    t.Run("状態バリデーション", func(t *testing.T) {
        // 共有DB接続プールからトランザクションをセットアップ
        db, tx := testutil.SetupTx(t)

        // テストユーザーを作成
        testutil.NewUserBuilder(t, tx).
            WithEmail("test@example.com").
            WithPassword("password123").
            Build()

        userRepo := repository.NewUserRepository(db).WithTx(tx)
        v := NewSignInCreateValidator(userRepo)

        t.Run("有効な認証情報", func(t *testing.T) {
            ctx := context.Background()
            input := SignInCreateValidatorInput{
                Email:    "test@example.com",
                Password: "password123",
            }

            result := v.Validate(ctx, input)

            if result.FormErrors != nil && result.FormErrors.HasErrors() {
                t.Errorf("unexpected form errors: %v", result.FormErrors)
            }
            if result.User == nil {
                t.Error("expected user, got nil")
            }
        })

        t.Run("無効なパスワード", func(t *testing.T) {
            ctx := context.Background()
            input := SignInCreateValidatorInput{
                Email:    "test@example.com",
                Password: "wrongpassword",
            }

            result := v.Validate(ctx, input)

            if result.User != nil {
                t.Error("expected nil user")
            }
            if result.FormErrors == nil || !result.FormErrors.HasErrors() {
                t.Error("expected form errors")
            }
        })
    })
}
```

### 正規表現のテスト

```go
func TestEmailRegex(t *testing.T) {
    tests := []struct {
        email string
        valid bool
    }{
        {"user@example.com", true},
        {"user.name@example.co.jp", true},
        {"user+tag@example.com", true},
        {"", false},
        {"invalid", false},
        {"@example.com", false},
        {"user@", false},
        {"user@.com", false},
    }

    for _, tt := range tests {
        t.Run(tt.email, func(t *testing.T) {
            if got := emailRegex.MatchString(tt.email); got != tt.valid {
                t.Errorf("emailRegex.MatchString(%q) = %v, want %v", tt.email, got, tt.valid)
            }
        })
    }
}
```

## ベストプラクティス

### 1. バリデーションは 1 ファイルに統合

```go
// ✅ Good: 状態バリデーションは internal/validator/ パッケージに配置
// internal/validator/sign_in.go
type SignInCreateValidator struct {
    userRepo *repository.UserRepository
}

func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) *SignInCreateValidatorResult {
    // 1. 形式バリデーション（DB不要）
    formErrors := session.NewFormErrors()
    // ...

    // 2. 状態バリデーション（DB必要）
    user, err := v.userRepo.GetByEmailForSignIn(ctx, input.Email)
    // ...
}
```

### 2. 国際化を徹底

```go
// ❌ Bad: ハードコードされたメッセージ
formErrors.AddFieldError("email", "メールアドレスを入力してください")

// ✅ Good: 国際化された翻訳
formErrors.AddFieldError("email", templates.T(ctx, "error_required"))
```

### 3. 早期リターンでネストを減らす

```go
// ❌ Bad: ネストが深い
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) *PasswordResetCreateValidatorResult {
    formErrors := session.NewFormErrors()
    if input.Email != "" {
        if emailRegex.MatchString(input.Email) {
            // OK
        } else {
            formErrors.AddFieldError("email", "...")
        }
    } else {
        formErrors.AddFieldError("email", "...")
    }
    return &PasswordResetCreateValidatorResult{FormErrors: formErrors}
}

// ✅ Good: 早期リターンでシンプル
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) *PasswordResetCreateValidatorResult {
    formErrors := session.NewFormErrors()

    if input.Email == "" {
        formErrors.AddFieldError("email", templates.T(ctx, "error_required"))
        return &PasswordResetCreateValidatorResult{FormErrors: formErrors}
    }

    if !emailRegex.MatchString(input.Email) {
        formErrors.AddFieldError("email", templates.T(ctx, "error_invalid_email_format"))
    }

    return &PasswordResetCreateValidatorResult{FormErrors: formErrors}
}
```

### 4. 正規表現はパッケージレベルで定義

```go
// ✅ Good: 正規表現のコンパイルを1回だけ実行
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) *PasswordResetCreateValidatorResult {
    // emailRegexを使用
}
```

### 5. Result 構造体でバリデーション結果を返す

```go
// ✅ Good: 結果を構造体で返す
type SignInCreateValidatorResult struct {
    User       *model.User        // 成功時のデータ
    FormErrors *session.FormErrors // フォームエラー
    Err        error               // システムエラー
}

func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) *SignInCreateValidatorResult {
    // ...
    return &SignInCreateValidatorResult{User: user}
}
```

## 利点

1. **判断コストがゼロ**: バリデーションは常に `internal/validator/` に配置するため、「どこに書くべきか」を迷わない
2. **進化に強い**: 形式バリデーションに後から DB チェックが追加されても、ファイル移動が不要
3. **一箇所で把握できる**: バリデーションルールを確認したいとき `internal/validator/` だけ見ればよい
4. **アーキテクチャの強制**: Handler パッケージから repository の import を完全に排除でき、depguard で強制可能
5. **依存が明確**: バリデーターの依存関係が一目でわかる
6. **テストしやすい**: バリデーション全体をテストできる
7. **再利用可能**: Handler だけでなく Worker からもバリデーションを呼び出せる
