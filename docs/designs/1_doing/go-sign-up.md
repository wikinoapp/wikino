# Go への移行 (新規ユーザー登録機能編) 設計書

<!--
このテンプレートの使い方:
1. このファイルを `docs/designs/2_todo/` ディレクトリにコピー
   例: cp docs/designs/template.md docs/designs/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残しておくことを推奨
-->

## 実装ガイドラインの参照

<!--
**重要**: 設計書を作成する前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
設計書作成の段階でガイドラインに準拠していることを確認してください。
-->

### Go版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン（**ファイル名は標準の8種類のみ**）
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

## 概要

<!--
ガイドライン:
- この機能が「何を」実現するのかを簡潔に説明
- ユーザーにとっての価値や背景を記述
- 2-3段落程度で簡潔に
-->

Go 版 Wikino に新規ユーザー登録（サインアップ）機能を実装します。ユーザーはメールアドレスを入力し、確認コードを受け取り、アットネームとパスワードを設定してアカウントを作成できます。

Rails 版 Wikino と同じ DB を共有するため、Go 版で作成されたユーザーアカウントは Rails 版でも有効です。また、サインアップ完了後は自動的にログイン状態になります。

**目的**:

- Rails 版から Go 版への段階的移行の一環として、新規ユーザー登録機能を実装する
- ユーザーが Go 版でアカウントを作成できるようにする
- Rails 版と同じメール確認フローを維持し、セキュリティを確保する

**背景**:

- ログイン機能は Go 版で実装済み（[Go への移行 (ログイン機能編)](./3_done/202602/go.md)）
- 新規ユーザー登録機能は、ログイン機能と並んで認証基盤の重要な要素
- Rails 版の既存フロー（メール確認→アカウント作成）を Go 版でも再現する

**関連仕様書**:

- [Go への移行 (ログイン機能編)](./3_done/202602/go.md) - ログイン機能の仕様書（完了済み）
- [Go への移行 (パスワードリセット機能編)](./go-password-reset.md) - パスワードリセット機能の仕様書

## 要件

<!--
ガイドライン:
- 機能要件: 「何ができるべきか」を記述
- 非機能要件: 「どのように動くべきか」を必要に応じて記述
-->

### 機能要件

<!--
「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
箇条書きで簡潔に
-->

- ユーザーはメールアドレスを入力してサインアップを開始できる
- システムは入力されたメールアドレスに確認コード（6桁の英数字）を送信する
- ユーザーは確認コードを入力してメールアドレスを確認できる
- 確認コードは15分間有効で、有効期限切れの場合は再送信が必要
- メール確認後、ユーザーはアットネームとパスワードを入力してアカウントを作成できる
- アカウント作成後、自動的にログイン状態になる
- ログイン済みユーザーがサインアップページにアクセスした場合、ホームページにリダイレクトする
- 既に登録済みのメールアドレスでサインアップしようとした場合、エラーメッセージを表示する
- 既に使用されているアットネームでアカウントを作成しようとした場合、エラーメッセージを表示する

### 非機能要件

<!--
必要に応じて以下のような項目を追加してください：
- セキュリティ（認証、認可、暗号化、監査ログなど）
- パフォーマンス（応答時間、スループット、リソース使用量など）
- ユーザビリティ（UX）（使いやすさ、わかりやすさ、アクセシビリティなど）
- 可用性・信頼性（稼働率、障害時の挙動、エラーハンドリングなど）
- 保守性（テストのしやすさ、コードの読みやすさ、ドキュメントなど）

不要な場合はこのセクション全体を削除してください。
-->

#### セキュリティ

- CSRF 対策を実施する（全フォームに CSRF トークンを含める）
- Cloudflare Turnstile による Bot 対策を実施（メール入力フォームで検証）
- パスワードは bcrypt でハッシュ化して保存
- 確認コードは6文字の大文字英数字（A-Z0-9）でランダム生成
- 確認コードの有効期限は15分
- Rate Limiting を実施（IP単位、メールアドレス単位）

#### 国際化

- 日本語と英語の両言語に対応
- エラーメッセージ、フラッシュメッセージ、フォームラベルを国際化

#### Rails 互換性

- Rails 版と同じ `email_confirmations` テーブルを使用
- Rails 版と同じ `users`, `user_passwords`, `user_sessions` テーブルを使用
- Rails 版で作成されたメール確認コードは Go 版でも検証可能
- Go 版で作成されたアカウントは Rails 版でも有効

## 設計

<!--
ガイドライン:
- 技術的な実装の詳細を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - 技術スタック（使用するライブラリ、フレームワーク、ツールなど）
  - アーキテクチャ（システム全体の構成、コンポーネント間の関係など）
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - セキュリティ設計（認証・認可、トークン管理、Rate Limitingなど）
  - コード設計（パッケージ構成、主要な構造体、インターフェースなど）
  - テスト戦略（単体テスト、統合テスト、E2Eテストの方針）
  - マイグレーション管理（データベースマイグレーションの方針）
  - 実装方針（特記事項、既存システムとの関係、制約など）

不要な場合はこのセクション全体を削除してください。
-->

### 技術スタック

- **パスワードハッシュ化**: `golang.org/x/crypto/bcrypt`
- **確認コード生成**: `crypto/rand`
- **HTTP ルーター**: `chi/v5`
- **テンプレート**: `templ`
- **DB アクセス**: `sqlc`
- **メール送信**: `resend-go/v2`（Resend API）
- **Bot 対策**: Cloudflare Turnstile

### データベース設計

既存の Rails 版テーブルをそのまま使用します。

#### email_confirmations テーブル（既存）

```sql
CREATE TABLE public.email_confirmations (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    email character varying NOT NULL,
    event integer NOT NULL,                      -- 0=signup, 1=email_update, 2=password_reset
    code character varying NOT NULL,             -- 6文字コード
    started_at timestamp without time zone NOT NULL,
    succeeded_at timestamp without time zone,    -- NULLなら未確認
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);
```

#### users テーブル（既存）

```sql
CREATE TABLE public.users (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    email character varying NOT NULL,
    atname public.citext NOT NULL,              -- 大文字小文字区別なし
    name character varying NOT NULL,
    description character varying NOT NULL,
    locale integer NOT NULL,                     -- enum: 0=en, 1=ja
    time_zone character varying NOT NULL,
    joined_at timestamp without time zone NOT NULL,
    discarded_at timestamp without time zone,   -- 論理削除用
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);
```

#### user_passwords テーブル（既存）

```sql
CREATE TABLE public.user_passwords (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    user_id uuid NOT NULL,
    password_digest character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);
```

### API 設計（ルーティング）

| URL                       | メソッド | ハンドラー                    | 説明                           |
| ------------------------- | -------- | ----------------------------- | ------------------------------ |
| `/sign_up`                | GET      | `sign_up.New`                 | サインアップフォーム表示（メール入力） |
| `/email_confirmation`     | POST     | `email_confirmation.Create`   | メール確認コード送信           |
| `/email_confirmation/edit`| GET      | `email_confirmation.Edit`     | 確認コード入力フォーム表示     |
| `/email_confirmation`     | PATCH    | `email_confirmation.Update`   | 確認コード検証                 |
| `/accounts/new`           | GET      | `account.New`                 | アカウント作成フォーム表示     |
| `/accounts`               | POST     | `account.Create`              | アカウント作成（ユーザー登録完了） |

### コード設計

#### ディレクトリ構造

```
internal/
├── handler/
│   ├── sign_up/
│   │   ├── handler.go      # Handler構造体と依存性
│   │   └── new.go          # GET /sign_up
│   ├── email_confirmation/
│   │   ├── handler.go      # Handler構造体と依存性
│   │   ├── create.go       # POST /email_confirmation
│   │   ├── edit.go         # GET /email_confirmation/edit
│   │   ├── update.go       # PATCH /email_confirmation
│   │   └── request.go      # CreateRequest, UpdateRequest バリデーション
│   └── account/
│       ├── handler.go      # Handler構造体と依存性
│       ├── new.go          # GET /accounts/new
│       ├── create.go       # POST /accounts
│       └── request.go      # CreateRequest バリデーション
├── usecase/
│   ├── send_email_confirmation.go    # メール確認コード送信ユースケース
│   ├── verify_email_confirmation.go  # メール確認コード検証ユースケース
│   └── create_account.go             # アカウント作成ユースケース
├── repository/
│   └── email_confirmation_repository.go  # メール確認 CRUD
├── model/
│   └── email_confirmation.go         # メール確認ドメインモデル
└── templates/
    └── pages/
        ├── sign_up/
        │   └── new.templ             # サインアップフォーム（メール入力）
        ├── email_confirmation/
        │   └── edit.templ            # 確認コード入力フォーム
        └── account/
            └── new.templ             # アカウント作成フォーム
```

#### 主要な構造体

**Handler（sign_up/handler.go）**

```go
type Handler struct {
    cfg             *config.Config
    sessionMgr      *session.Manager
    turnstileClient *turnstile.Client
}
```

**Handler（email_confirmation/handler.go）**

```go
type Handler struct {
    cfg                     *config.Config
    sessionMgr              *session.Manager
    flashMgr                *session.FlashManager
    userRepo                *repository.UserRepository
    sendEmailConfirmationUC *usecase.SendEmailConfirmationUsecase
    verifyEmailConfirmationUC *usecase.VerifyEmailConfirmationUsecase
    turnstileClient         *turnstile.Client
}
```

**Handler（account/handler.go）**

```go
type Handler struct {
    cfg                *config.Config
    sessionMgr         *session.Manager
    flashMgr           *session.FlashManager
    createAccountUC    *usecase.CreateAccountUsecase
    createUserSessionUC *usecase.CreateUserSessionUsecase
}
```

**SendEmailConfirmationUsecase（usecase/send_email_confirmation.go）**

```go
type SendEmailConfirmationUsecase struct {
    emailConfirmationRepo *repository.EmailConfirmationRepository
    emailClient           *email.Client
}

type SendEmailConfirmationInput struct {
    Email  string
    Locale string
}

func (uc *SendEmailConfirmationUsecase) Execute(ctx context.Context, input SendEmailConfirmationInput) error
```

**VerifyEmailConfirmationUsecase（usecase/verify_email_confirmation.go）**

```go
type VerifyEmailConfirmationUsecase struct {
    emailConfirmationRepo *repository.EmailConfirmationRepository
}

type VerifyEmailConfirmationInput struct {
    Email string
    Code  string
}

func (uc *VerifyEmailConfirmationUsecase) Execute(ctx context.Context, input VerifyEmailConfirmationInput) error
```

**CreateAccountUsecase（usecase/create_account.go）**

```go
type CreateAccountUsecase struct {
    db                    *sql.DB
    userRepo              *repository.UserRepository
    userPasswordRepo      *repository.UserPasswordRepository
    emailConfirmationRepo *repository.EmailConfirmationRepository
}

type CreateAccountInput struct {
    Email    string
    Atname   string
    Password string
    Locale   int
    TimeZone string
}

type CreateAccountOutput struct {
    UserID string
}

func (uc *CreateAccountUsecase) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountOutput, error)
```

### 認証フロー

```
1. ユーザーが GET /sign_up にアクセス
   ├── 認証済みの場合 → /home にリダイレクト
   └── 未認証の場合 → メール入力フォーム表示

2. ユーザーがメールアドレスを送信 (POST /email_confirmation)
   ├── CSRF トークン検証
   ├── Turnstile 検証
   ├── Rate Limiting チェック（IP単位: 5回/時間）
   ├── Rate Limiting チェック（メールアドレス単位: 3回/時間）
   ├── フォームバリデーション
   │   └── メールアドレス: 必須、形式チェック
   ├── メールアドレス重複チェック
   │   └── 既に登録済みの場合 → エラー表示
   ├── 確認コード生成（6文字、A-Z0-9）
   ├── email_confirmations テーブルに INSERT
   ├── 確認メール送信（Resend API）
   ├── セッションに email_confirmation_id を保存
   └── /email_confirmation/edit にリダイレクト

3. ユーザーが GET /email_confirmation/edit にアクセス
   ├── セッションに email_confirmation_id がない場合 → /sign_up にリダイレクト
   └── 確認コード入力フォーム表示

4. ユーザーが確認コードを送信 (PATCH /email_confirmation)
   ├── CSRF トークン検証
   ├── セッションから email_confirmation_id を取得
   ├── 確認コード検証（VerifyEmailConfirmationUsecase）
   │   ├── コードが正しいかチェック
   │   ├── 有効期限（15分）チェック
   │   └── 検証失敗の場合 → エラー表示
   ├── email_confirmations.succeeded_at を更新
   └── /accounts/new にリダイレクト

5. ユーザーが GET /accounts/new にアクセス
   ├── セッションに email_confirmation_id がない場合 → /sign_up にリダイレクト
   ├── メール確認が完了していない場合 → /email_confirmation/edit にリダイレクト
   └── アカウント作成フォーム表示（メールアドレスは確認済みで表示のみ）

6. ユーザーがアカウント情報を送信 (POST /accounts)
   ├── CSRF トークン検証
   ├── フォームバリデーション
   │   ├── アットネーム: 必須、形式チェック（[A-Za-z0-9_]+）、最大20文字
   │   └── パスワード: 必須、最小8文字
   ├── アットネーム重複チェック
   │   └── 既に使用されている場合 → エラー表示
   ├── アカウント作成（CreateAccountUsecase）
   │   ├── users テーブルに INSERT
   │   └── user_passwords テーブルに INSERT（bcryptハッシュ化）
   ├── セッション作成（CreateUserSessionUsecase）
   ├── Cookie にセッショントークンを設定
   ├── セッションから email_confirmation_id を削除
   ├── フラッシュメッセージ設定
   └── /home にリダイレクト
```

### セキュリティ設計

#### 確認コード生成

```go
func GenerateConfirmationCode() (string, error) {
    // 6文字のランダムな大文字英数字を生成
    const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    code := make([]byte, 6)
    for i := range code {
        n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        if err != nil {
            return "", err
        }
        code[i] = charset[n.Int64()]
    }
    return string(code), nil
}
```

#### Rate Limiting

- **IP単位**: 5回/時間（同一IPからの連続リクエストを制限）
- **メールアドレス単位**: 3回/時間（同一メールアドレスへの連続送信を制限）

#### パスワードバリデーション

```go
const (
    PasswordMinLength = 8   // 最小8文字
    PasswordMaxLength = 72  // bcryptの制限
)

func ValidatePassword(password string) error {
    if len(password) < PasswordMinLength {
        return errors.New("password_too_short")
    }
    if len(password) > PasswordMaxLength {
        return errors.New("password_too_long")
    }
    return nil
}
```

#### アットネームバリデーション

```go
const (
    AtnameMaxLength = 20
    AtnamePattern   = `^[A-Za-z0-9_]+$`
)

func ValidateAtname(atname string) error {
    if len(atname) == 0 {
        return errors.New("atname_required")
    }
    if len(atname) > AtnameMaxLength {
        return errors.New("atname_too_long")
    }
    if !regexp.MustCompile(AtnamePattern).MatchString(atname) {
        return errors.New("atname_invalid_format")
    }
    return nil
}
```

### テスト戦略

- **ハンドラーテスト**: HTTP リクエスト・レスポンスの統合テスト
- **ユースケーステスト**: メール確認、アカウント作成ロジックのテスト
- **リポジトリテスト**: DB 操作のテスト
- **バリデーションテスト**: アットネーム、パスワード形式のテスト

テストでは実際の PostgreSQL データベースを使用し、トランザクションで分離します。

## タスクリスト

<!--
ガイドライン:
- フェーズごとに段階的な実装計画を記述
- チェックボックスで進捗を管理
- **重要**: 1タスク = 1 Pull Request の粒度で作成してください
- **重要**: 各タスクには想定ファイル数と想定行数を明記してください（PRサイズの見積もりのため）
- 想定ファイル数は「実装」と「テスト」に分けて記載してください
- 想定行数も「実装」と「テスト」に分けて記載してください
- 依存関係を明確に
- Pull Requestのガイドラインは CLAUDE.md を参照（変更ファイル数20以下、変更行数300行以下）

タスク番号の付け方:
- 各タスクには階層的な番号を付与します（例: 1-1, 1-2, 2-1, 2-2）
- フォーマット: **フェーズ番号-タスク番号**: タスク名
- タスクの前に別のタスクを追加する場合は、サブ番号を使用します
  - 例: タスク 2-1 の前にタスクを追加する場合 → 2-0
  - 例: タスク 2-0 の前にタスクを追加する場合 → 2-0-1
- この番号はブランチ名の一部として使用されます（例: feature-1-1, feature-2-0）
-->

### フェーズ 1: リポジトリ層の実装

<!--
例: インフラ準備、基本機能実装、セキュリティ機能など
各タスクは1つのPull Requestで完結する粒度で記述してください
各タスクには想定サイズを明記してください
-->

- [x] **1-1**: [Go] EmailConfirmationRepository の実装

  - `internal/repository/email_confirmation_repository.go` の作成
  - `internal/model/email_confirmation.go` の作成
  - `db/queries/email_confirmations.sql` の作成
  - sqlc コード生成
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

### フェーズ 2: ユースケース層の実装

- [x] **2-1**: [Go] SendEmailConfirmationUsecase の実装

  - `internal/usecase/send_email_confirmation.go` の作成
  - 確認コード生成ロジック
  - メール送信処理（Resend API）
  - メールテンプレートの作成
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

- [ ] **2-2**: [Go] VerifyEmailConfirmationUsecase の実装

  - `internal/usecase/verify_email_confirmation.go` の作成
  - 確認コード検証ロジック
  - 有効期限チェック
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 200 行（実装 80 行 + テスト 120 行）

- [ ] **2-3**: [Go] CreateAccountUsecase の実装

  - `internal/usecase/create_account.go` の作成
  - ユーザー作成ロジック（トランザクション管理）
  - パスワードハッシュ化
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 300 行（実装 120 行 + テスト 180 行）

### フェーズ 3: サインアップフォームの実装

- [ ] **3-1**: [Go] サインアップフォームハンドラーの実装

  - `internal/handler/sign_up/handler.go` の作成
  - `internal/handler/sign_up/new.go` の作成
  - `internal/templates/pages/sign_up/new.templ` の作成
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 250 行（実装 150 行 + テスト 100 行）

### フェーズ 4: メール確認機能の実装

- [ ] **4-1**: [Go] メール確認コード送信ハンドラーの実装

  - `internal/handler/email_confirmation/handler.go` の作成
  - `internal/handler/email_confirmation/create.go` の作成
  - `internal/handler/email_confirmation/request.go` の作成（CreateRequest）
  - Turnstile 検証、Rate Limiting
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 500 行（実装 250 行 + テスト 250 行）

- [ ] **4-2**: [Go] 確認コード入力フォームハンドラーの実装

  - `internal/handler/email_confirmation/edit.go` の作成
  - `internal/templates/pages/email_confirmation/edit.templ` の作成
  - **想定ファイル数**: 約 4 ファイル（実装 2 + テスト 2）
  - **想定行数**: 約 250 行（実装 100 行 + テスト 150 行）

- [ ] **4-3**: [Go] 確認コード検証ハンドラーの実装

  - `internal/handler/email_confirmation/update.go` の作成
  - `internal/handler/email_confirmation/request.go` の更新（UpdateRequest 追加）
  - **想定ファイル数**: 約 4 ファイル（実装 2 + テスト 2）
  - **想定行数**: 約 300 行（実装 120 行 + テスト 180 行）

### フェーズ 5: アカウント作成機能の実装

- [ ] **5-1**: [Go] アカウント作成フォームハンドラーの実装

  - `internal/handler/account/handler.go` の作成
  - `internal/handler/account/new.go` の作成
  - `internal/templates/pages/account/new.templ` の作成
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 300 行（実装 180 行 + テスト 120 行）

- [ ] **5-2**: [Go] アカウント作成ハンドラーの実装

  - `internal/handler/account/create.go` の作成
  - `internal/handler/account/request.go` の作成
  - アットネーム・パスワードバリデーション
  - セッション作成、Cookie 設定
  - **想定ファイル数**: 約 6 ファイル（実装 3 + テスト 3）
  - **想定行数**: 約 500 行（実装 200 行 + テスト 300 行）

### フェーズ 6: 統合とルーティング

- [ ] **6-1**: [Go] ルーティング設定とリバースプロキシミドルウェアの更新

  - ルーティング設定（`/sign_up`, `/email_confirmation/*`, `/accounts/*`）
  - リバースプロキシのホワイトリスト更新
  - **想定ファイル数**: 約 4 ファイル（実装 2 + テスト 2）
  - **想定行数**: 約 200 行（実装 80 行 + テスト 120 行）

- [ ] **6-2**: [Go] 国際化（I18n）メッセージの追加

  - `internal/i18n/locales/ja.toml` の更新
  - `internal/i18n/locales/en.toml` の更新
  - サインアップ関連のすべてのメッセージを追加
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 100 行（実装 100 行 + テスト 0 行）

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **ソーシャルログイン（OAuth）**: Wikino は現在未対応
- **メールアドレスの変更**: 別タスクで実装予定
- **確認メールの再送信UI**: 有効期限切れ時は再度 /sign_up から開始
- **アットネームの予約語チェック**: Rails 版の実装を参考に、必要であれば後で追加
- **招待制サインアップ**: 現在は誰でもサインアップ可能

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- **Rails 版 Wikino サインアップ実装**: `/workspace/rails/app/controllers/sign_up/`, `/workspace/rails/app/controllers/email_confirmations/`, `/workspace/rails/app/controllers/accounts/`
- **Annict Go 版サインアップ実装**: `/annict/go/internal/handler/sign_up/`, `/annict/go/internal/handler/sign_up_code/`, `/annict/go/internal/handler/sign_up_username/`
- **Wikino Go 版ログイン実装**: `/workspace/go/internal/handler/sign_in/`, `/workspace/go/internal/handler/user_session/`
- **Go CLAUDE.md**: `/workspace/go/CLAUDE.md`
- [chi ルーター](https://github.com/go-chi/chi)
- [sqlc](https://docs.sqlc.dev/)
- [templ](https://templ.guide/)
- [Cloudflare Turnstile](https://developers.cloudflare.com/turnstile/)
- [Resend API](https://resend.com/docs)
