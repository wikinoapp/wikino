# UseCase オーケストレーション リファクタリング 作業計画書

<!--
このテンプレートの使い方:
1. このファイルを `docs/plans/2_todo/` ディレクトリにコピー
   例: cp docs/plans/template.md docs/plans/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残してください

**作業計画書の性質**:
- 作業計画書は「何をどう変えるか」という変更内容を記述するドキュメントです
- 新しい機能の場合は、概要・要件・設計もこのドキュメントに記述します
- 現在のシステムの状態は `docs/specs/` の仕様書に記述されています
- タスク完了後は、仕様書を新しい状態に更新してください（設計判断や採用しなかった方針も含める）

**仕様書との関係**:
- 新しい機能の場合: タスク完了後に `docs/specs/` に仕様書を作成する
- 既存機能の変更の場合: 「仕様書」セクションに対応する仕様書へのリンクを記載し、タスク完了後に仕様書を更新する

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 仕様書

- [アーキテクチャガイド](/workspace/go/docs/architecture-guide.md)
- [ハンドラーガイド](/workspace/go/docs/handler-guide.md)
- [バリデーションガイド](/workspace/go/docs/validation-guide.md)

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

Go 版 Wikino のアーキテクチャにおける処理のオーケストレーション責務を、Handler から UseCase に移動するリファクタリング。

### 現状の課題

現在は Handler がオーケストレーターとして、読み取り UseCase → Policy → Validator → 書き込み UseCase の流れを制御している。この設計には以下の課題がある：

- **エントリーポイントが増えた場合の漏れリスク**: 将来 Web API などのエントリーポイントが追加された場合、認可・バリデーションの呼び出しを各エントリーポイントで再現する必要があり、漏れが発生しやすい
- **Handler にドメイン固有の判断が集中**: 外部世界との接点である Handler にビジネスロジックの制御フローが書かれており、関心の分離が不十分

### 変更後の方針

- **UseCase をオーケストレーターにする**: バリデーション・認可・ビジネスロジック・永続化を UseCase 内部で統括する
- **Handler は HTTP の入出力変換に徹する**: リクエストのパース → UseCase 呼び出し → レスポンス（リダイレクト or テンプレート描画）
- **Worker を Presentation 層に移動**: Handler と同じく、薄い Adapter として UseCase を呼ぶだけにする

## 要件

<!--
ガイドライン:
- 機能要件: 「何ができるべきか」を記述
- 非機能要件: 「どのように動くべきか」を必要に応じて記述
-->

### 機能要件

- UseCase がバリデーション・認可・ビジネスロジック・永続化を統括する
- Handler は HTTP リクエストのパース、UseCase の呼び出し、レスポンスの生成のみを行う
- Worker は UseCase を呼ぶだけの薄い Adapter となる
- 外部からの振る舞いは変わらない（リファクタリングのみ）

### 非機能要件

- **一貫性**: すべての Handler・Worker が同じパターンに従う。ケースバイケースの判断を排除する
- **保守性**: エントリーポイントが増えても認可・バリデーションが漏れない構造にする

### 追加要件（フェーズ 5a）

- email パッケージにメール種別ごとの Sender（`ConfirmationSender`, `PasswordResetSender`）を新設し、テンプレートレンダリングを UseCase から移動する
- UseCase → templates の例外依存を解消し、application-layer に `templates` deny を追加する
- `SendRaw` / `SendRawInput` を `Sender` インターフェースから削除し、`Send` のみに統一する
- 未使用の `SendEmailWorker` / `SendEmailUsecase` / `SendEmailArgs` / `EnqueueSendEmail` を削除する

## 実装ガイドラインの参照

<!--
**重要**: 作業計画書を作成する前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
作業計画書作成の段階でガイドラインに準拠していることを確認してください。
-->

### Go版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン（**ファイル名は標準の9種類のみ**）
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 設計

<!--
ガイドライン:
- 技術的な実装の設計を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - UI設計（画面構成、ユーザーフローなど）
  - セキュリティ設計（認証・認可、トークン管理など）
  - コード設計（パッケージ構成、主要な構造体など）

**重要: 設計は実装中に更新する**:
- 作業計画書内の設計は初期の方針であり、完璧ではない
- 実装中により良いアプローチが見つかった場合は、設計を積極的に更新する
- 設計に固執して実装の質を下げるよりも、実装で得た知見を設計に反映する方が重要
- 変更した場合は「採用しなかった方針」セクションに変更前の方針と変更理由を記録する
-->

### 変更前後のアーキテクチャ比較

**変更前**（Handler がオーケストレーター）:

```
Handler (オーケストレーター)
  ├── 読み取り UseCase（データ取得）
  ├── Policy（認可チェック）
  ├── Validator（バリデーション）
  └── 書き込み UseCase（永続化のみ）
```

**変更後**（UseCase がオーケストレーター）:

```
Handler / Worker (薄い Adapter)
  └── UseCase (オーケストレーター)
        ├── Validator（バリデーション）
        ├── Policy（認可チェック）
        ├── ビジネスロジック
        └── Repository（データアクセス・永続化）
```

### 検討事項 1: UseCase からのエラーの返し方 【確定】

#### 課題

現状 Validator は `session.FormErrors`（Presentation 層の型）を含む Result を返し、Handler がフォームを再描画している。UseCase にバリデーションが移ると、UseCase が Presentation 層の型に依存してしまう。

#### 現状のパターン

```go
// internal/validator/suggestion.go（現状）
type SuggestionCreateValidatorResult struct {
    FormErrors *session.FormErrors  // Presentation層の型
    DraftPages []*model.DraftPage
    Err        error
}

// internal/handler/suggestion/create.go（現状）
result := h.createValidator.Validate(ctx, input)
if result.FormErrors.HasErrors() {
    w.WriteHeader(http.StatusUnprocessableEntity)
    h.renderNewForm(w, r, ..., result.FormErrors, ...)
    return
}
```

#### 確定方針: Domain/Infrastructure 層（Model）にエラー型を定義する

`internal/model/` に `ValidationError` と `AppError` を定義する。Model は依存グラフの最下層にあるため、すべての層から自然に参照でき、循環依存が起きない。

**ValidationError**: バリデーションエラー（ユーザーが入力を修正できるエラー）。Handler はフォームを再描画する。

**AppError**: アプリケーションエラー（[SafeError パターン](https://blog.jetbrains.com/go/2026/03/02/secure-go-error-handling-best-practices/)を参考）。`Error()` メソッドはユーザー安全なメッセージのみを返し、内部エラーの露出を構造的に防止する。

```go
// internal/model/errors.go

// ValidationError はバリデーションエラーを表す。
// Handler はこのエラーを受け取ったらフォームを再描画する（422）。
type ValidationError struct {
    Global []string
    Fields map[string][]string
}

func (e *ValidationError) Error() string { return "validation failed" }

func (e *ValidationError) AddGlobal(message string) { ... }
func (e *ValidationError) AddField(field, message string) { ... }
func (e *ValidationError) HasErrors() bool { ... }

// AppErrorCode はアプリケーションエラーの種別を表す型
type AppErrorCode int

const (
    AppErrCodeResourceNotFound AppErrorCode = iota + 1
    AppErrCodeForbidden
    AppErrCodeConflict
    AppErrCodeInternal
)

// AppError はアプリケーションエラーを表す（SafeError パターン）。
// Error() はユーザー安全なメッセージのみを返す。
type AppError struct {
    Code     AppErrorCode         // エラー種別（定数で一元管理）
    UserMsg  string            // ユーザーに表示する安全なメッセージ
    Internal error             // 内部エラー（ログ用、ユーザーには非公開）
    Metadata map[string]string // 構造化ログ用のメタデータ
}

func (e *AppError) Error() string { return e.UserMsg }

// LogString はログ出力用の詳細文字列を返す
func (e *AppError) LogString() string {
    return fmt.Sprintf("Code: %d | Msg: %s | Cause: %v | Meta: %v",
        e.Code, e.UserMsg, e.Internal, e.Metadata)
}
```

**依存の方向**:

```
Handler (Presentation) → errors.As で ValidationError / AppError を判別
UseCase (Application)  → ValidationError / AppError を return
Validator (Application) → ValidationError を生成
Model (Domain/Infra)    → ValidationError / AppError を定義
```

**Handler での使用パターン**:

```go
// Handler は UseCase のエラーを型で判別してレスポンスを決定する
output, err := h.createSuggestionUC.Execute(ctx, input)
if err != nil {
    var ve *model.ValidationError
    if errors.As(err, &ve) {
        // バリデーションエラー → フォーム再描画（422）
        w.WriteHeader(http.StatusUnprocessableEntity)
        h.renderNewForm(w, r, ..., ve, ...)
        return
    }
    var ae *model.AppError
    if errors.As(err, &ae) {
        // アプリケーションエラー → ログ + ユーザー向けメッセージ
        slog.ErrorContext(ctx, ae.LogString())
        // ae.Error() は安全なメッセージのみ返す
    }
    // 予期しないエラー → 500
    slog.ErrorContext(ctx, "予期しないエラー", "error", err)
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    return
}
```

**Validator での使用パターン**:

```go
// Validator は ValidationError を生成して返す
func (v *SuggestionCreateValidator) Validate(ctx context.Context, input Input) error {
    ve := &model.ValidationError{}

    if input.Title == "" {
        ve.AddField("title", templates.T(ctx, "error_required"))
    }

    if ve.HasErrors() {
        return ve
    }

    // 状態バリデーション（DB検証）
    ...

    return nil
}
```

**エラー型の使い分け**:

| エラー型                 | 生成元    | 意味                             | Handler の対応                          |
| ------------------------ | --------- | -------------------------------- | --------------------------------------- |
| `*model.ValidationError` | Validator | 入力が不正（ユーザーが修正可能） | フォーム再描画（422）                   |
| `*model.AppError`        | UseCase   | 業務レベルの既知の失敗           | エラーコードに応じた処理（403, 404 等） |
| 素の `error`             | どこでも  | 予期しないシステムエラー         | 500                                     |

- **ValidationError**: Validator が生成する。UseCase は Validator から受け取ったエラーをそのまま返す
- **AppError**: UseCase が生成する。業務文脈を知る UseCase だけが「この失敗はユーザーにどう伝えるべきか」を判断できる
- **素の error**: どの層でも発生しうる。Handler は詳細をログに記録し、ユーザーには汎用的なエラーメッセージを表示する

**UseCase でのエラー生成パターン**:

```go
func (uc *CreateSuggestionUsecase) Execute(ctx context.Context, input Input) (*Output, error) {
    // 1. データ取得 → 見つからない場合は AppError
    space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return nil, &model.AppError{
                Code:     model.AppErrCodeResourceNotFound,
                UserMsg:  i18n.T(ctx, "error_space_not_found"),
                Internal: err,
            }
        }
        return nil, err  // 予期しないエラー → そのまま返す
    }

    // 2. 認可チェック → 権限なしは AppError
    if !policy.NewTopicPolicy(spaceMember).CanCreateSuggestion() {
        return nil, &model.AppError{
            Code:    model.AppErrCodeForbidden,
            UserMsg: i18n.T(ctx, "error_forbidden"),
        }
    }

    // 3. バリデーション → ValidationError or システムエラー
    draftPages, err := uc.validator.Validate(ctx, input)
    if err != nil {
        return nil, err  // ValidationError か素の error がそのまま上がる
    }

    // 4. 永続化
    ...
}
```

**既存の `session.FormErrors` との関係**:

- `session.FormErrors` は廃止し、`model.ValidationError` に置き換える
- テンプレートは `model.ValidationError` を直接受け取る（構造は同じ `Global` + `Fields`）
- `session` パッケージから `form_errors.go` を削除する

**変更の影響範囲**:

- `internal/session/form_errors.go` → 削除
- `internal/model/errors.go` → 新規作成
- `internal/validator/*.go` → Result 型を廃止し、`error` を返すように変更
- `internal/handler/*/create.go`, `update.go` 等 → `errors.As` パターンに変更
- `internal/templates/**/*.templ` → `session.FormErrors` → `model.ValidationError` に変更

### 検討事項 2: Read UseCase と Write UseCase の統合 【確定】

#### 課題

現状は「読み取り UseCase」と「書き込み UseCase」が分離されている。認可・バリデーションが UseCase に移ると、書き込み UseCase 内でデータ取得も行うことになり、読み取り UseCase との役割が重複する可能性がある。

#### 現状のパターン

```go
// Handler が2つの UseCase を使い分ける（現状）
output, _ := h.getSuggestionNewUsecase.Execute(ctx, ...)  // 読み取り
// → Policy チェック
// → Validator
createOutput, _ := h.createSuggestionUsecase.Execute(ctx, ...)  // 書き込み
```

#### 確定方針: 書き込み UseCase に統合する

書き込み操作の UseCase がデータ取得・認可・バリデーション・永続化を一貫して行う。読み取り UseCase はフォーム表示（GET リクエスト）専用として残す。

```go
// 書き込み UseCase（変更後）
func (uc *CreateSuggestionUsecase) Execute(ctx context.Context, input Input) (*Output, error) {
    // 1. データ取得（認可・バリデーションに必要なデータ）
    space, _ := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
    spaceMember, _ := uc.spaceMemberRepo.Find(ctx, space.ID, input.UserID)

    // 2. 認可チェック
    if !policy.NewTopicPolicy(spaceMember, ...).CanCreateSuggestion() {
        return nil, ErrForbidden
    }

    // 3. バリデーション
    if err := uc.validator.Validate(ctx, validatorInput); err != nil {
        return nil, err  // ValidationError
    }

    // 4. 永続化（トランザクション）
    ...
}
```

**読み取り UseCase の扱い**:

- フォーム表示（GET リクエスト）用の読み取り UseCase はそのまま残す
- 読み取り UseCase のプレフィックス `Get` は維持（`Get` = 読み取り、それ以外 = 書き込みの判別ルール）

### 検討事項 3: Worker のメールレンダリング 【確定】

#### 課題

現状 Worker は `templates` パッケージへの依存が例外として許可されており、メールの HTML レンダリングを行っている。Worker を Presentation 層の薄い Adapter に変更すると、メールレンダリングの配置先を決める必要がある。

#### 現状のパターン

```go
// internal/worker/send_email_confirmation.go（現状）
func (w *SendEmailConfirmationWorker) Work(ctx context.Context, job *river.Job[...]) error {
    // テンプレートレンダリング（Worker 内で直接実行）
    htmlBody := renderEmailTemplate(ctx, job.Args)
    // メール送信
    w.sender.SendRaw(ctx, ...)
}
```

#### 確定方針: メール送信 UseCase を作成し、テンプレートレンダリングも UseCase 内で行う

UseCase → templates の依存を例外として許可する（現状の Worker → templates と同じ例外が UseCase に移るだけ）。将来テンプレート選択にビジネスロジックが必要になった場合にも UseCase 内で完結できる。

```go
// Worker（変更後）- 薄い Adapter
func (w *SendEmailConfirmationWorker) Work(ctx context.Context, job *river.Job[...]) error {
    return w.sendEmailConfirmationUC.Execute(ctx, usecase.SendEmailConfirmationInput{
        Email:  job.Args.Email,
        Code:   job.Args.Code,
        Locale: job.Args.Locale,
    })
}

// UseCase - テンプレートレンダリング + メール送信
type SendEmailConfirmationUsecase struct {
    sender *email.Sender
}
func (uc *SendEmailConfirmationUsecase) Execute(ctx context.Context, input ...) error {
    htmlBody := renderEmailTemplate(ctx, input)
    return uc.sender.SendRaw(ctx, ...)
}
```

**依存の例外ルール**:

- 現状: Worker → templates（メールレンダリングのため例外として許可）
- 変更後: UseCase → templates（メールレンダリングのため例外として許可）
- Worker → templates の例外は廃止（Worker は UseCase を呼ぶだけ）

### 検討事項 3a: UseCase と Worker 間の循環依存の解消 【確定】

#### 課題

UseCase がジョブをキューに投入し、Worker が UseCase を呼び出すと、パッケージ間の循環依存が発生する。

```
internal/usecase/ → internal/worker/（enqueue 時に Args 型を参照）
internal/worker/  → internal/usecase/（UseCase を呼び出し）
→ 循環依存！
```

#### 確定方針: Dispatcher パッケージを新設する

`internal/dispatcher/` パッケージを Domain/Infrastructure 層に新設し、ジョブキューへの投入を抽象化する。Repository がデータベースアクセスを抽象化するのと同じ発想で、Dispatcher がジョブキューアクセスを抽象化する。

```go
// internal/dispatcher/dispatcher.go（Domain/Infrastructure 層）
package dispatcher

type Dispatcher struct {
    riverClient *river.Client
}

func NewDispatcher(riverClient *river.Client) *Dispatcher {
    return &Dispatcher{riverClient: riverClient}
}

func (d *Dispatcher) EnqueueEmailConfirmation(ctx context.Context, email, code, locale string) error {
    _, err := d.riverClient.Insert(ctx, &sendEmailConfirmationArgs{
        Email: email, Code: code, Locale: locale,
    }, nil)
    return err
}

// Args 型もこのパッケージ内に定義する
type SendEmailConfirmationArgs struct {
    Email  string `json:"email"`
    Code   string `json:"code"`
    Locale string `json:"locale"`
}
func (SendEmailConfirmationArgs) Kind() string { return "send_email_confirmation" }
```

**UseCase での使用**:

```go
// UseCase は River の存在を知らない
func (uc *CreateAccountUsecase) Execute(ctx context.Context, input Input) error {
    // ... アカウント作成 ...
    return uc.dispatcher.EnqueueEmailConfirmation(ctx, input.Email, code, locale)
}
```

**Worker での使用**:

```go
// Worker は Args 型を dispatcher から参照し、UseCase を呼ぶ
type SendEmailConfirmationWorker struct {
    river.WorkerDefaults[dispatcher.SendEmailConfirmationArgs]
    uc *usecase.SendEmailConfirmationUsecase
}

func (w *SendEmailConfirmationWorker) Work(ctx context.Context, job *river.Job[dispatcher.SendEmailConfirmationArgs]) error {
    return w.uc.Execute(ctx, usecase.SendEmailConfirmationInput{
        Email: job.Args.Email, Code: job.Args.Code, Locale: job.Args.Locale,
    })
}
```

**依存の方向**:

```
Worker (Presentation)      → dispatcher + usecase
UseCase (Application)      → dispatcher
Dispatcher (Domain/Infra)  → river（外部ライブラリ）
```

循環なし。UseCase は River やジョブ Args 型の存在を知らない。

**Repository との対比**:

|                          | Repository                                       | Dispatcher                                      |
| ------------------------ | ------------------------------------------------ | ----------------------------------------------- |
| **抽象化する対象**       | データベースアクセス（同期的なデータ永続化）     | ジョブキューへの投入（非同期タスク委譲）        |
| **層**                   | Domain/Infrastructure                            | Domain/Infrastructure                           |
| **UseCase からの見え方** | `repo.FindByID(ctx, id)`                         | `dispatcher.EnqueueEmailConfirmation(ctx, ...)` |
| **分離の基準**           | インフラの種類ではなく、**操作の性質**で分離する |

### 検討事項 4: 書き込み UseCase のルール見直し 【確定】

#### 課題

現状の書き込み UseCase には以下の3つのルールがある：

1. データの検証処理を書かない
2. トランザクション開始後はデータの取得や計算処理を行わない（永続化のみ）
3. Execute 内にロジックを直接書かない

認可・バリデーションが UseCase に移ることで、ルール1は廃止が必要。ルール2・3は維持可能。

#### 確定方針

変更後のルール:

1. ~~データの検証処理を書かない~~ → **バリデーション・認可は UseCase の責務として実行する**
2. トランザクション開始後はデータの取得や計算処理を行わない（永続化のみ） → **維持**
3. Execute 内にロジックを直接書かない → **維持**

UseCase 内の処理順序:

```go
func (uc *CreateSuggestionUsecase) Execute(ctx context.Context, input Input) (*Output, error) {
    // 1. データ取得（トランザクション外）
    // 2. 認可チェック
    // 3. バリデーション
    // 4. ビジネスロジック（計算、変換等）
    // 5. トランザクション（永続化のみ）
}
```

### 検討事項 5: Validator パッケージの位置づけ 【確定】

#### 課題

Validator は引き続き `internal/validator/` パッケージに配置するが、呼び出し元が Handler から UseCase に変わる。Validator の Result 型に `session.FormErrors` が含まれている点を解決する必要がある（検討事項 1 と関連）。

#### 確定方針

- Validator パッケージは `internal/validator/` に維持する（「判断コストゼロ」の原則を維持）
- 呼び出し元が Handler → UseCase に変わるだけで、Validator 自体の責務は変わらない
- **Result 型を廃止し、Go の慣習に従った `(data, error)` の2値返しに変更する**

**変更後の Validator パターン**:

```go
// データを返す Validator
func (v *SuggestionCreateValidator) Validate(ctx context.Context, input Input) ([]*model.DraftPage, error) {
    ve := &model.ValidationError{}

    if input.Title == "" {
        ve.AddField("title", templates.T(ctx, "error_required"))
    }
    if ve.HasErrors() {
        return nil, ve  // *model.ValidationError は error を満たす
    }

    // 状態バリデーションで取得したデータを返す
    draftPages, err := v.draftPageRepo.FindByIDs(ctx, input.DraftPageIDs)
    if err != nil {
        return nil, err  // システムエラー
    }

    return draftPages, nil
}

// データを返さない Validator
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input Input) error {
    ve := &model.ValidationError{}
    // ...
    if ve.HasErrors() {
        return ve
    }
    return nil
}
```

**変更の効果**:

- Validator ごとに専用の Result 構造体を定義する手間がなくなる
- `FormErrors` と `Err` の2つのエラー経路が `error` 1つに統合される
- Go の標準的なエラーハンドリングパターン（`errors.As`）で扱える

### 検討事項 6: depguard ルールの更新 【確定】

現状の depguard ルールを変更後のアーキテクチャに合わせて更新する。

| 変更対象               | 現状             | 変更後                                     |
| ---------------------- | ---------------- | ------------------------------------------ |
| UseCase → Policy       | 禁止             | 許可                                       |
| UseCase → Validator    | （暗黙的に許可） | 明示的に許可                               |
| UseCase → templates    | 禁止             | 例外として許可（メールレンダリングのため） |
| UseCase → Dispatcher   | （未存在）       | 許可                                       |
| Handler → Validator    | 許可             | 禁止（UseCase 経由に統一）                 |
| Handler → Policy       | 許可             | 禁止（UseCase 経由に統一）                 |
| Worker → templates     | 例外として許可   | 禁止（UseCase 経由に統一）                 |
| Worker → UseCase       | （未存在）       | 許可                                       |
| Worker → Dispatcher    | （未存在）       | 許可（Args 型の参照）                      |
| Dispatcher → UseCase   | -                | 禁止                                       |
| Dispatcher → Handler   | -                | 禁止                                       |
| Dispatcher → Worker    | -                | 禁止                                       |
| Dispatcher → Validator | -                | 禁止                                       |
| Dispatcher → Policy    | -                | 禁止                                       |

**Dispatcher の依存先**: River（外部ライブラリ）のみ。上位層（UseCase, Handler, Worker）や同レイヤーの他パッケージ（Validator, Policy）には依存しない。

### 検討事項 7: UseCase → templates 例外依存の解消 【確定】

#### 課題

検討事項 3 で確定した方針により、メールのテンプレートレンダリングは UseCase 内で行い、UseCase → templates の依存を「例外として許可」している。この例外ルールは以下の問題がある：

- **例外の存在自体がルールを弱める**: depguard のコメントに「例外として許可」と書くと、将来他の例外が追加されやすい
- **Mewst との一貫性**: Mewst では email パッケージにレンダリング責務を移すことで例外を解消しており、Wikino でも同じパターンを適用したい
- **関心の分離**: テンプレートレンダリング（Presentation 層の関心）が Application 層の UseCase に存在している

#### 確定方針: email パッケージにテンプレートレンダリングを移動する

`internal/email/` にメール種別ごとの Sender を新設し、テンプレートレンダリング + i18n 件名取得を email パッケージに閉じ込める。UseCase は自前で定義した小さい interface に依存するだけで、`internal/templates` を import しない。

```go
// internal/email/confirmation_sender.go（email パッケージ）
type ConfirmationSender struct {
    sender Sender
}

func (s *ConfirmationSender) Send(ctx context.Context, to, code, appURL, locale string) error {
    ctx = i18n.SetLocale(ctx, locale)
    subject := i18n.T(ctx, "email_confirmation_subject")

    data := email_confirmation.Data{Email: to, Code: code, AppURL: appURL}
    var htmlBody, textBody templ.Component
    switch locale {
    case "ja":
        htmlBody = email_confirmation.JaHTML(data)
        textBody = email_confirmation.JaText(data)
    default:
        htmlBody = email_confirmation.EnHTML(data)
        textBody = email_confirmation.EnText(data)
    }

    return s.sender.Send(ctx, SendInput{
        To: to, Subject: subject, HTMLBody: htmlBody, TextBody: textBody,
    })
}
```

```go
// internal/usecase/send_email_confirmation.go（UseCase）
// UseCase 側で interface を定義（templates に依存しない）
type EmailConfirmationSender interface {
    Send(ctx context.Context, to, code, appURL, locale string) error
}

type SendEmailConfirmationUsecase struct {
    sender EmailConfirmationSender
}

func (uc *SendEmailConfirmationUsecase) Execute(ctx context.Context, input SendEmailConfirmationInput) error {
    if input.Email == "" {
        return fmt.Errorf("メールアドレスが空です")
    }
    return uc.sender.Send(ctx, input.Email, input.Code, input.AppURL, input.Locale)
}
```

**「採用しなかった方針 C」との違い**:

- 方針 C は「テンプレートファイルを `internal/email/templates/` に移動する」話であり、不採用
- 本方針は「テンプレートレンダリングの責務を email パッケージに移す」話であり、テンプレートファイルは `internal/templates/emails/` に残る
- templ のツールチェーンはそのまま維持できる

**`SendRaw` / `SendRawInput` の削除**:

email パッケージ内の Sender が `Send`（`templ.Component` ベース）を直接使うようになるため、`SendRaw`（文字列ベース）は不要になる。`Sender` インターフェースを `Send` のみに統一することで、インターフェースがシンプルになる。

**email-layer の depguard ルール**:

```yaml
email-layer:
  files:
    - "**/internal/email/**"
  deny:
    - pkg: .../internal/query
    - pkg: .../internal/repository
    - pkg: .../internal/handler
    - pkg: .../internal/middleware
    - pkg: .../internal/viewmodel
    - pkg: .../internal/usecase
    - pkg: .../internal/validator
    - pkg: .../internal/worker
    - pkg: .../internal/dispatcher
    - pkg: .../internal/session
```

許可される依存先: `templates`, `i18n`, `model`, `config`, 外部パッケージ（`templ`, `resend`）

**依存の方向**:

```
Worker (Presentation)      → usecase
UseCase (Application)      → email (interface 経由)
email (Presentation helper) → templates, i18n, Sender
```

### 変更対象のファイル一覧（参考）

#### UseCase（書き込み）

| ファイル                        | 対応する Validator                          | 対応する Policy |
| ------------------------------- | ------------------------------------------- | --------------- |
| `create_suggestion.go`          | `suggestion.go` (SuggestionCreateValidator) | `topic.go`      |
| `update_suggestion.go`          | `suggestion.go` (SuggestionUpdateValidator) | `topic.go`      |
| `create_suggestion_comment.go`  | `suggestion_comment.go`                     | `topic.go`      |
| `update_suggestion_comment.go`  | `suggestion_comment.go`                     | `topic.go`      |
| `publish_page.go`               | `page.go`                                   | `topic.go`      |
| `manual_save_draft_page.go`     | `page.go`                                   | `topic.go`      |
| `auto_save_draft_page.go`       | `page.go`                                   | `topic.go`      |
| `move_page.go`                  | `page_move.go`                              | `topic.go`      |
| `create_account.go`             | `account.go`                                | なし            |
| `mark_email_as_confirmed.go`    | `email_confirmation.go`                     | なし            |
| `create_user_session.go`        | `sign_in.go`                                | なし            |
| `update_password_reset.go`      | `password_reset.go` / `password.go`         | なし            |
| `apply_suggestion.go`           | `suggestion.go`                             | `topic.go`      |
| `close_suggestion.go`           | なし                                        | `topic.go`      |
| `start_suggestion_page_edit.go` | なし                                        | `topic.go`      |
| `update_suggestion_page.go`     | `suggestion_page.go`                        | `topic.go`      |

#### Worker

| ファイル                     | 変更内容               |
| ---------------------------- | ---------------------- |
| `send_email_confirmation.go` | UseCase 呼び出しに変更 |
| `send_password_reset.go`     | UseCase 呼び出しに変更 |
| `send_email.go`              | UseCase 呼び出しに変更 |
| `cleanup_rate_limits.go`     | UseCase 呼び出しに変更 |

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### A. Read UseCase を廃止し UseCase を1つに統合する（検討事項 2 案B）

GET と POST で同じ UseCase を呼び、引数で動作を切り替える方針を検討した。

**不採用の理由**:

- GET（フォーム表示）と POST（作成処理）で責務が異なるため、1つの UseCase に統合すると不自然になる
- 読み取り UseCase はフォーム表示専用として残すほうが、責務が明確でシンプル

### B. Worker（Presentation 層）でテンプレートをレンダリングし UseCase に渡す（検討事項 3 案B）

Worker がメールテンプレートをレンダリングし、レンダリング済み HTML を UseCase に渡す方針を検討した。Handler がテンプレートを描画するのと同じパターンで、アーキテクチャの例外が不要という利点があった。

**不採用の理由**:

- 将来テンプレート選択にビジネスロジック（例: ユーザーのプランに応じた内容の分岐）が必要になった場合、Worker 側でその判断ができない
- その時点で案A（UseCase 内でレンダリング）に変更する手間が発生する
- 最初から UseCase にレンダリングを配置しておけば、判断コストが不要

### C. メールテンプレートを独立パッケージに分離する（検討事項 3 案C）

メールテンプレートを `internal/email/templates/` のような独立パッケージに移動する方針を検討した。

**不採用の理由**:

- パッケージが増えて複雑になる
- メールテンプレートも templ で記述しており、HTTP レスポンス用テンプレートと同じツールチェーンを使用しているため、分離するメリットが薄い

### D. ジョブの enqueue を Repository に含める

Repository がデータベースの抽象化層であるため、ジョブキューへの投入も Repository に含める方針を検討した。

**不採用の理由**:

- Repository は同期的なデータ永続化（CRUD）を担当し、Model と 1:1 で対応する。ジョブキューへの投入（非同期タスク委譲）は操作の性質が異なる
- Repository は `WithTx` パターンでトランザクションに参加するが、ジョブ投入はトランザクションとは別のライフサイクルを持つ
- `EnqueueEmailConfirmation` をどの Repository に置くかという判断コストが発生する
- 分離の基準はインフラの種類（PostgreSQL vs Redis vs River）ではなく、操作の性質（同期的データ永続化 vs 非同期タスク委譲）とする

### E. ValidationError と AppError を Application 層に配置する

`internal/usecase/errors.go` または新設の `internal/apperror/` にエラー型を定義する方針を検討した。

**不採用の理由**:

- Validator（Application 層）が `ValidationError` を生成するために `usecase` パッケージを import すると、UseCase → Validator という依存の方向に対して Validator → UseCase の逆方向依存が発生し、循環依存のリスクがある
- 新設パッケージ（`internal/apperror/`）を作ると、パッケージが増えて複雑になる
- Model（Domain/Infrastructure 層）は依存グラフの最下層にあり、すべての層から自然に参照できるため、エラー型の配置先として適切

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

フィーチャーフラグ:
- 新機能の開発ではフィーチャーフラグによる制御を検討してください
- フラグで制御することで、実装途中でも develop ブランチにマージできます
- フラグのセットアップ（フラグ名の定義、ルーティングパターンの追加など）をフェーズ1に含めてください
- 機能が安定した後のフラグ削除タスクも計画に含めてください
- 詳細は CLAUDE.md の「フィーチャーフラグによる開発」セクションを参照

タスク番号の付け方:
- 各タスクには階層的な番号を付与します（例: 1-1, 1-2, 2-1, 2-2）
- フォーマット: **フェーズ番号-タスク番号**: タスク名
- **フェーズ番号は半角英数字とハイフンのみで表記**してください（ブランチ名に使用するため）
  - 例: フェーズ 1, フェーズ 2, フェーズ 5a（フェーズ 5 と 6 の間に追加する場合）
  - NG: フェーズ 5.5（ドットは使用不可）
- タスクの前に別のタスクを追加する場合は、サブ番号を使用します
  - 例: タスク 2-1 の前にタスクを追加する場合 → 2-0
  - 例: タスク 2-0 の前にタスクを追加する場合 → 2-0-1
- この番号はブランチ名の一部として使用されます（例: feature-1-1, feature-2-0）

プラットフォームプレフィックス:
- Go版またはRails版の修正を行うタスクには、タスク名の先頭にプラットフォームを示すプレフィックスを付けてください
- フォーマット: **フェーズ番号-タスク番号**: [Go] タスク名 または **フェーズ番号-タスク番号**: [Rails] タスク名
- Go版とRails版の両方を修正する場合は、別々のタスクに分けてください
- 例:
  - `- [ ] **1-1**: [Go] マイグレーション作成`
  - `- [ ] **1-2**: [Rails] モデルへのコールバック追加`
-->

### フェーズ 1: 基盤整備

- [x] **1-1**: [Go] エラー型の定義（`model.ValidationError`, `model.AppError`, `model.AppErrorCode`）
  - `internal/model/errors.go` を新規作成
  - `ValidationError`（Global + Fields）、`AppError`（SafeError パターン）、`ErrorCode`（iota 定数）を定義
  - ヘルパー関数（`AsValidationError`, `AsAppError` 等）を定義
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行）

- [x] **1-2**: [Go] Dispatcher パッケージの新設
  - `internal/dispatcher/dispatcher.go` を新規作成
  - 既存の Worker Args 型を `internal/worker/` から `internal/dispatcher/` に移動
  - Enqueue メソッド（`EnqueueEmailConfirmation`, `EnqueuePasswordReset` 等）を定義
  - 既存の UseCase が `worker` パッケージ経由で River に投入している箇所を `dispatcher` 経由に変更
  - **想定ファイル数**: 約 4 ファイル（実装 2 + テスト 2）
  - **想定行数**: 約 250 行（実装 150 行 + テスト 100 行）

- [x] **1-3**: [Go] depguard ルールの更新（許可ルール + Dispatcher ルール）
  - UseCase → Policy, templates の禁止ルールを削除（依存を許可）
  - Dispatcher の上位層依存を禁止するルールを新設
  - Worker → UseCase, Dispatcher は現状で禁止されていないため変更不要
  - Handler → Policy, Validator の禁止ルール追加はフェーズ 3a-2 に延期
  - Worker → templates の禁止ルール追加はフェーズ 4-4 に延期
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 30 行（実装 30 行 + テスト 0 行）

### フェーズ 2: パイロット移行（1機能で検証）

<!--
比較的シンプルな機能で新パターンを検証し、確立する。
パイロット対象: suggestion_comment の create（Validator + Policy 両方あり、シンプルな構成）
-->

- [x] **2-1**: [Go] パイロット: create_suggestion_comment UseCase にバリデーション・認可を統合
  - `internal/usecase/create_suggestion_comment.go` に Validator・Policy の呼び出しを統合
  - `internal/validator/suggestion_comment.go` の Result 型を廃止し `(data, error)` 返しに変更
  - `internal/handler/suggestion_comment/create.go` から Validator・Policy の直接呼び出しを削除し、UseCase の `errors.As` パターンに変更
  - テンプレートの `session.FormErrors` 参照を `model.ValidationError` に変更（対象テンプレート）
  - `main.go` の DI 構成を更新
  - 全テストを更新
  - **想定ファイル数**: 約 12 ファイル（実装 6 + テスト 6）
  - **想定行数**: 約 500 行（実装 250 行 + テスト 250 行）

### フェーズ 3: 残りの UseCase の移行

<!--
パイロット移行の確立したパターンに従い、残りの UseCase を移行する。
関連する Validator・Policy・Handler ごとにグルーピングし、1タスク = 1 PR の粒度で分割。
認可（Policy）のみの UseCase と、バリデーション + 認可の UseCase を分けて記載。
-->

#### 認可 + バリデーション

- [x] **3-1**: [Go] create_suggestion UseCase の移行
  - UseCase に SuggestionCreateValidator + TopicPolicy を統合
  - Handler（suggestion/create.go, suggestion/new.go）を更新
  - **想定ファイル数**: 約 10 ファイル（実装 5 + テスト 5）
  - **想定行数**: 約 500 行（実装 250 行 + テスト 250 行）

- [x] **3-2**: [Go] update_suggestion UseCase の移行
  - UseCase に SuggestionUpdateValidator + TopicPolicy を統合
  - Handler（suggestion/update.go, suggestion/edit.go）を更新
  - **想定ファイル数**: 約 10 ファイル（実装 5 + テスト 5）
  - **想定行数**: 約 500 行（実装 250 行 + テスト 250 行）

- [x] **3-3**: [Go] update_suggestion_comment UseCase の移行
  - UseCase に SuggestionCommentUpdateValidator + TopicPolicy を統合
  - Handler（suggestion_comment_edit/edit.go, update.go）を更新
  - **想定ファイル数**: 約 10 ファイル（実装 5 + テスト 5）
  - **想定行数**: 約 500 行（実装 250 行 + テスト 250 行）

- [x] **3-4**: [Go] publish_page, manual_save_draft_page, auto_save_draft_page UseCase の移行
  - 3つの UseCase に PageValidator + TopicPolicy を統合（共通の Validator を使用）
  - Handler（page/update.go 等）を更新
  - **想定ファイル数**: 約 12 ファイル（実装 6 + テスト 6）
  - **想定行数**: 約 600 行（実装 300 行 + テスト 300 行）

- [x] **3-5**: [Go] move_page UseCase の移行
  - UseCase に PageMoveValidator + TopicPolicy を統合
  - Handler（page_move/create.go, page_move/new.go）を更新
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

- [x] **3-6**: [Go] update_suggestion_page UseCase の移行
  - UseCase に SuggestionPageValidator + TopicPolicy を統合
  - Handler（suggestion_page/update.go）を更新
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

- [x] **3-7**: [Go] apply_suggestion UseCase の移行
  - UseCase に TopicPolicy を統合（Validator あり）
  - Handler（suggestion_apply/create.go）を更新
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

#### 認可のみ（バリデーションなし）

- [x] **3-8**: [Go] close_suggestion, start_suggestion_page_edit UseCase の移行
  - 2つの UseCase に TopicPolicy を統合
  - Handler（suggestion_close/create.go, suggestion_page_edit/create.go）を更新
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 300 行（実装 150 行 + テスト 150 行）

#### バリデーションのみ（認可なし）

- [x] **3-9**: [Go] create_account UseCase の移行
  - UseCase に AccountCreateValidator を統合
  - Handler（account/create.go）を更新
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

- [x] **3-10**: [Go] create_user_session UseCase の移行
  - UseCase に SignInCreateValidator を統合
  - Handler（sign_in/create.go）を更新
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

- [x] **3-11**: [Go] mark_email_as_confirmed UseCase の移行
  - UseCase に EmailConfirmationValidator を統合
  - Handler（email_confirmation/update.go, create.go）を更新
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

- [x] **3-12**: [Go] update_password_reset UseCase の移行
  - UseCase に PasswordResetValidator / PasswordValidator を統合
  - Handler（password_reset/create.go, password/edit.go）を更新
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

### フェーズ 3a: テンプレートの session.FormErrors 除去

<!--
フェーズ 3 の移行で個別に対応しきれなかったテンプレート・コンポーネントの残存参照を一括で除去する。
-->

- [x] **3a-1**: [Go] `session.FormErrors` の完全除去
  - `internal/templates/components/form_errors.templ` を `model.ValidationError` に変更
  - 残存する全テンプレートの `session.FormErrors` 参照を除去
  - `internal/session/form_errors.go` を削除
  - templ generate を実行し、生成ファイルを更新
  - **想定ファイル数**: 約 16 ファイル（実装 16 + テスト 0）
  - **想定行数**: 約 200 行（実装 200 行 + テスト 0 行）

- [x] **3a-2**: [Go] depguard 禁止ルールの追加（Handler → Policy, Validator）
  - Handler から Policy への依存を禁止するルールを追加
  - Handler から Validator への依存を禁止するルールを追加
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 10 行（実装 10 行 + テスト 0 行）

- [x] **3a-3**: [Go] フォーム表示用の読み取り UseCase の整理
  - 書き込み Handler の edit.go が show ページ用の読み取り UseCase（例: `GetSuggestionDetailUsecase`）を流用しており、不要なデータ（`SuggestionPages`, `Pages`, `Comments` 等）まで取得している
  - edit 用の専用読み取り UseCase を新設し、必要なデータのみ取得するように変更する
  - 読み取り UseCase に認可チェックを含めて Handler から Policy の直接呼び出しを除去するかどうかも検討する
  - 対象の edit.go を洗い出し、対応する
  - **想定ファイル数**: 未定（対象の洗い出し後に見積もり）
  - **想定行数**: 未定（対象の洗い出し後に見積もり）

### フェーズ 4: Worker の Presentation 層への移動

- [x] **4-1**: [Go] メール送信 Worker 用の UseCase を新設
  - `send_email_confirmation`, `send_password_reset`, `send_email` の3つの UseCase を新規作成
  - テンプレートレンダリング + メール送信ロジックを Worker から UseCase に移動
  - **想定ファイル数**: 約 6 ファイル（実装 3 + テスト 3）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

- [x] **4-2**: [Go] cleanup_rate_limits Worker 用の UseCase を新設
  - `cleanup_rate_limits` の UseCase を新規作成
  - ロジックを Worker から UseCase に移動
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 100 行（実装 50 行 + テスト 50 行）

- [x] **4-3**: [Go] Worker を薄い Adapter に変更
  - 4つの Worker から ビジネスロジックを削除し、UseCase を呼ぶだけに変更
  - Worker を Presentation 層として位置づける（ドキュメントはフェーズ 5 で更新）
  - `main.go` の DI 構成を更新
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行）

- [x] **4-4**: [Go] depguard 禁止ルールの追加（Worker → templates）
  - Worker から templates への依存を禁止するルールを追加
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 10 行（実装 10 行 + テスト 0 行）

### フェーズ 5: ドキュメント更新

- [x] **5-1**: [Go] アーキテクチャガイドの更新
  - 3層アーキテクチャの図を更新（Worker を Presentation 層に、Dispatcher を Domain/Infrastructure 層に追加）
  - UseCase のオーケストレーション責務、処理順序を反映
  - 書き込み UseCase のルール（旧ルール1の廃止）を反映
  - 依存関係ルールの更新（depguard ルールとの整合性）
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 200 行（実装 200 行 + テスト 0 行）

- [x] **5-2**: [Go] ハンドラーガイド・バリデーションガイドの更新
  - Handler の責務変更（オーケストレーター → 薄い Adapter）を反映
  - Validator の呼び出し元変更（Handler → UseCase）と Result 型廃止を反映
  - エラー型の使い分け（ValidationError, AppError, 素の error）を追記
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 200 行（実装 200 行 + テスト 0 行）

### フェーズ 5a: email パッケージのリファクタリング

<!--
UseCase → templates の例外依存を解消する。
email パッケージにテンプレートレンダリング責務を移動し、
Sender インターフェースを簡素化する。
Mewst の depguard-alignment 作業計画書のタスク 4-1a, 4-2 に対応。
-->

- [x] **5a-1**: [Go] email パッケージへのテンプレートレンダリング移動 + depguard 更新
  - `internal/email/confirmation_sender.go` に `ConfirmationSender` を新設（テンプレートレンダリング + i18n 件名取得）
  - `internal/email/password_reset_sender.go` に `PasswordResetSender` を新設（テンプレートレンダリング + i18n 件名取得）
  - `internal/usecase/send_email_confirmation.go` から `internal/templates` import を削除し、interface パターンに変更
  - `internal/usecase/send_password_reset.go` から `internal/templates` import を削除し、interface パターンに変更
  - application-layer に `templates` deny を追加
  - email-layer の depguard ルールを新設
  - `main.go` の DI 構成を更新
  - テスト更新
  - `make lint` で違反がないことを確認
  - **想定ファイル数**: 約 10 ファイル（実装 5 + テスト 5）
  - **想定行数**: 約 300 行（実装 150 行 + テスト 150 行）

- [x] **5a-2**: [Go] SendRaw 削除 + 未使用コード削除
  - `Sender` インターフェースから `SendRaw` / `SendRawInput` を削除（`Send` のみに統一）
  - `NoopSender` から `SentRawEmails` フィールドを削除
  - 未使用の `SendEmailWorker` / `SendEmailUsecase` / `SendEmailArgs` / `EnqueueSendEmail` を削除
  - `internal/worker/send_email.go`, `internal/worker/send_email_test.go` を削除
  - `internal/usecase/send_email.go`, `internal/usecase/send_email_test.go` を削除（5a-1 で `mockSender` がこのファイルに移動されたため、ファイルごと削除すること）
  - `main.go` の DI 構成を更新（`SendEmailWorker` の登録を削除）
  - `make lint` で違反がないことを確認
  - **想定ファイル数**: 約 8 ファイル（実装 5 + テスト 3、削除行多め）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行）

### フェーズ 6: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [x] **6-1**: 仕様書の作成・更新
  - `docs/specs/` に仕様書を作成または更新する
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **Web API エントリーポイントの追加**: リファクタリングのみ行い、新しいエントリーポイントは追加しない
- **Rails 版のアーキテクチャ変更**: Go 版のみを対象とする

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [The Clean Architecture (Uncle Bob)](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Hexagonal Architecture (Alistair Cockburn)](https://alistair.cockburn.us/hexagonal-architecture/)
