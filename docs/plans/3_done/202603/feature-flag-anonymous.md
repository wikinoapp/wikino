# フィーチャーフラグの未ログインユーザー対応 作業計画書

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

<!--
- 既存機能を変更する場合: 変更対象の仕様書へのリンクを記載してください
- 新しい機能の場合: タスク完了後に作成予定の仕様書のパスを記載してください
-->

- [フィーチャーフラグ 仕様書](../../specs/feature-flag/overview.md)（タスク完了後に更新）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

現在のフィーチャーフラグは `feature_flags` テーブルの `user_id` を通じてログインユーザー単位でフラグを管理しているため、未ログインユーザーに対してフラグの切り替えができない。未ログインユーザーがフィーチャーフラグ付きURLにアクセスした場合、常にRails版にフォールバックされる。

本タスクでは、`feature_flags` テーブルのスキーマを変更し、`user_id` に加えて `device_token`（Cookieに保存されるランダムな値）でもフラグを管理できるようにする。これにより、未ログインユーザーに対しても `device_token` 経由でフラグの切り替えが可能になる。ログインユーザーには `user_id` で全デバイス横断のフラグ設定ができ、未ログインユーザーには `device_token` でデバイス単位のフラグ設定ができる。

### 関連タスク

- [@docs/plans/3_done/202603/feature-flag.md](../3_done/202603/feature-flag.md) - フィーチャーフラグ（本タスクの前提）

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

- システムは `device_token`（Cookie値）によるフラグ設定で、未ログインユーザーに対してもフィーチャーフラグの有効/無効を管理できる
- システムは `user_id` によるフラグ設定で、ログインユーザーに対して全デバイス横断でフラグの有効/無効を管理できる
- 未ログインユーザーがフィーチャーフラグ付きURLにアクセスした場合、`device_token` のフラグが有効ならGo版で処理される
- 未ログインユーザーがログインした場合、`device_token` で設定されたフラグはそのまま有効であり続ける（同じCookieが維持されるため、移行処理は不要）
- 開発者はDBを直接操作してフラグを管理できる（`device_token` による設定、`user_id` による設定の両方）

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

- **パフォーマンス**: フラグ判定は、既存と同様にフィーチャーフラグ付きURLパターンにマッチするリクエストのみで実行される。1クエリで `device_token` と `user_id` の両方をチェックする
- **信頼性**: フラグ判定でエラーが発生した場合は、安全側に倒してRails版にフォールバックする
- **セキュリティ**: `device_token` Cookie は安全なランダム値を使用し、HttpOnly + Secure + SameSite=Lax で設定する

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
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

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

### データベース設計

既存の `feature_flags` テーブルを変更する。`feature_flags` テーブルは現在未使用のため、スキーマ変更による既存の挙動への影響はない。

#### 変更前

```sql
CREATE TABLE feature_flags (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);
```

#### 変更後

```sql
CREATE TABLE feature_flags (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    device_token VARCHAR,
    user_id UUID REFERENCES users(id),
    name VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (device_token IS NOT NULL OR user_id IS NOT NULL),
    UNIQUE(device_token, name),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_feature_flags_device_token ON feature_flags(device_token);
CREATE INDEX idx_feature_flags_user_id ON feature_flags(user_id);
CREATE INDEX idx_feature_flags_name ON feature_flags(name);
```

**変更点**:

- `user_id` を `NOT NULL` から nullable に変更し、外部キー制約は維持
- `device_token` カラム（nullable）を追加
- `CHECK` 制約で `device_token` と `user_id` の少なくとも一方が NOT NULL であることを保証
- ユニーク制約を `(user_id, name)` から `(device_token, name)` と `(user_id, name)` の2つに変更
- `device_token` 用のインデックスを追加

**2種類のフラグレコード**:

| レコード種類       | `device_token` | `user_id` | 対象                     | 用途                                       |
| ------------------ | -------------- | --------- | ------------------------ | ------------------------------------------ |
| デバイス単位フラグ | 設定あり       | NULL      | そのデバイスのみ         | 未ログインユーザー、特定デバイスでのテスト |
| ユーザー単位フラグ | NULL           | 設定あり  | そのユーザーの全デバイス | 開発者やベータユーザーへのロールアウト     |

PostgreSQLでは `UNIQUE` 制約のNULL値は重複として扱われないため、`device_token` が NULL のレコードは `UNIQUE(device_token, name)` 制約に影響しない。同様に `user_id` が NULL のレコードは `UNIQUE(user_id, name)` 制約に影響しない。

### コード設計

#### Cookie

デバイス（ブラウザ）識別用の新しいCookieを定義する。

- **Cookie名**: `device_token`
- カラム名と一致させ、対応関係を直感的にする
- 既存の `user_session_tokens` Cookie（ログインセッション用）とは独立
- `HttpOnly: true`, `Secure: true`（本番環境）, `SameSite: Lax`
- `MaxAge`: 10年（`user_session_tokens` と同じ長期間）
- ログイン前後で同じCookieが維持されるため、フラグの移行処理は不要
- **自動生成**: リバースプロキシミドルウェアで `device_token` Cookie が存在しない場合に自動生成してレスポンスにセットする
- **トークン生成**: 既存の `session.GenerateSecureToken()` を使用する（24バイトのランダムデータを Base64 URL-safe エンコードした32文字の文字列）

Cookie名の定数は `internal/middleware/reverse_proxy.go` に定義する:

```go
// DeviceTokenCookieName はデバイス（ブラウザ）識別用のCookieキー名
// ログイン状態に関わらずデバイス単位でサイト訪問者を識別するために使用する
const DeviceTokenCookieName = "device_token"
```

#### Model

`internal/model/feature_flag.go` の `FeatureFlag` モデルを変更:

```go
// FeatureFlag はフィーチャーフラグのドメインモデル
type FeatureFlag struct {
    ID          FeatureFlagID
    DeviceToken *string   // nullable: デバイス単位フラグの場合に設定
    UserID      *UserID   // nullable: ユーザー単位フラグの場合に設定
    Name        FeatureFlagName
    CreatedAt   time.Time
}
```

#### Repository

`internal/repository/feature_flag_repository.go` のメソッドを変更:

```go
// IsEnabledForDevice はデバイストークンまたはログインセッション経由で
// フラグが有効かどうかを返す
// deviceTokenとsessionTokenの両方を受け取り、1クエリで判定する
func (r *FeatureFlagRepository) IsEnabledForDevice(
    ctx context.Context,
    deviceToken string,
    sessionToken string,
    name model.FeatureFlagName,
) (bool, error)
```

既存の `IsEnabled`（user_id直接指定）は内部利用・テスト用に維持する。
`IsEnabledBySessionToken` は `IsEnabledForDevice` に置き換え、削除する。

**sqlcクエリ**:

```sql
-- name: IsFeatureFlagEnabledForDevice :one
-- デバイストークンまたはセッショントークン経由のuser_idでフラグが有効かを判定する
-- deviceToken: device_token Cookieの値（空文字列の場合はマッチしない）
-- sessionToken: user_session_tokens Cookieの値（空文字列の場合はマッチしない）
SELECT EXISTS(
    SELECT 1 FROM feature_flags ff
    WHERE ff.name = $3
    AND (
        (ff.device_token IS NOT NULL AND ff.device_token = $1)
        OR (ff.user_id IS NOT NULL AND ff.user_id = (
            SELECT us.user_id FROM user_sessions us WHERE us.token = $2
        ))
    )
);
```

- `$1` = `device_token` Cookie の値（空文字列は何にもマッチしない）
- `$2` = `user_session_tokens` Cookie の値（空文字列は何にもマッチしない）
- `$3` = フラグ名

#### リバースプロキシミドルウェアの変更

`featureFlagChecker` インターフェースを変更:

```go
// featureFlagChecker はフィーチャーフラグの有効判定を行うインターフェース
type featureFlagChecker interface {
    IsEnabledForDevice(ctx context.Context, deviceToken string, sessionToken string, name model.FeatureFlagName) (bool, error)
}
```

`Middleware` メソッドに `device_token` Cookie の自動生成を追加:

```go
func (m *ReverseProxyMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // device_token Cookie が存在しない場合は自動生成してセット
        m.ensureDeviceToken(w, r)

        // 以下は既存のルーティング判定ロジック（変更なし）
        // ...
    })
}

// ensureDeviceToken は device_token Cookie が存在しない場合に自動生成する
func (m *ReverseProxyMiddleware) ensureDeviceToken(w http.ResponseWriter, r *http.Request) {
    if _, err := r.Cookie(DeviceTokenCookieName); err == nil {
        return // 既にCookieが存在する
    }

    token, err := session.GenerateSecureToken()
    if err != nil {
        slog.WarnContext(r.Context(), "device_tokenの生成に失敗", "error", err)
        return
    }

    http.SetCookie(w, &http.Cookie{
        Name:     DeviceTokenCookieName,
        Value:    token,
        Path:     "/",
        Domain:   m.cfg.CookieDomain,
        MaxAge:   10 * 365 * 24 * 60 * 60, // 10年
        HttpOnly: true,
        Secure:   m.cfg.SecureCookie,
        SameSite: http.SameSiteLaxMode,
    })
}
```

`isFeatureFlagEnabled` メソッドを変更:

```go
func (m *ReverseProxyMiddleware) isFeatureFlagEnabled(r *http.Request, flagName model.FeatureFlagName) bool {
    if m.featureFlagRepo == nil {
        return false
    }

    // device_token Cookie の値を取得
    deviceToken := ""
    if cookie, err := r.Cookie(DeviceTokenCookieName); err == nil {
        deviceToken = cookie.Value
    }

    // user_session_tokens Cookie の値を取得
    sessionToken := ""
    if cookie, err := r.Cookie(session.CookieName); err == nil {
        sessionToken = cookie.Value
    }

    // どちらのCookieも存在しない場合はRails版にフォールバック
    if deviceToken == "" && sessionToken == "" {
        return false
    }

    // 1クエリでdevice_tokenとuser_idの両方をチェック
    enabled, err := m.featureFlagRepo.IsEnabledForDevice(r.Context(), deviceToken, sessionToken, flagName)
    if err != nil {
        slog.WarnContext(r.Context(), "フィーチャーフラグ判定でエラーが発生（Rails版にフォールバック）",
            "error", err,
            "flag", flagName,
            "path", r.URL.Path,
        )
        return false
    }

    return enabled
}
```

### ルーティングの流れ（変更後）

```
リクエスト到着
  ↓
[リバースプロキシミドルウェア]
  ├─ device_token Cookie なし？ → 自動生成してレスポンスにセット
  ↓
  ├─ 常にGoのパス？ → Yes → Go側ミドルウェアチェーン → ハンドラー
  ├─ フィーチャーフラグ付きパス？
  │   ├─ Yes + device_tokenのフラグ有効 → Go側ミドルウェアチェーン → ハンドラー
  │   ├─ Yes + user_idのフラグ有効（ログインユーザー全デバイス）→ Go側ミドルウェアチェーン → ハンドラー
  │   └─ Yes + フラグ無効/トークンなし/エラー → Railsにプロキシ
  └─ その他 → Railsにプロキシ
```

### フラグ管理の運用例

```sql
-- 未ログインユーザーの特定デバイスに対してフラグを設定
-- （device_token Cookieは自動生成されるため、ブラウザのDevToolsで値を確認する）
INSERT INTO feature_flags (device_token, name) VALUES ('cookie値', 'go_page_edit');

-- ログインユーザーの全デバイスに対してフラグを設定
INSERT INTO feature_flags (user_id, name) VALUES ('user-uuid', 'go_page_edit');

-- 両方を設定することも可能（別レコードとして）
INSERT INTO feature_flags (device_token, name) VALUES ('cookie値', 'go_page_edit');
INSERT INTO feature_flags (user_id, name) VALUES ('user-uuid', 'go_page_edit');
```

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### 匿名ユーザー専用テーブル（`feature_flags_anonymous`）の新設

匿名ユーザー用のフラグを別テーブル（`feature_flags_anonymous`）で管理する方式を検討した。

**不採用の理由**: フラグ管理テーブルが2つに分散し、管理の複雑さが増す。`feature_flags` テーブルは現在未使用のためスキーマ変更のリスクがなく、1つのテーブルで `device_token` と `user_id` の両方を管理するほうがシンプルで保守しやすい。

### `viewers` テーブルの新設

`device_token` と `user_id` のマッピングを管理する `viewers` テーブルを作成し、`feature_flags` から参照する方式を検討した。

**不採用の理由**: フィーチャーフラグの用途においては、`device_token` はCookieの値として十分に機能し、テーブルで管理する必要がない。追加のテーブルとJOINが必要になり、複雑さが増す。

### Viewer 概念モデルの導入

サイト訪問者を概念的にモデリングし、Viewer（全訪問者）/ User（ログイン済み）/ Visitor（未ログイン）と定義する方式を検討した。Cookie名・カラム名を `viewer_token` とし、概念モデルに紐づける設計だった。

**不採用の理由**: 将来的にWeb APIを提供する際、GitHubのGraphQL APIのように `viewer` をアクセストークンの発行者を指す用語として使いたい（例: `viewerCanCreatePage`）。フィーチャーフラグのCookie識別子と API の `viewer` 概念が衝突するため、Cookie/カラム名は実態に即した `device_token` とし、概念モデルは導入しないこととした。

### 匿名トークンを `user_session_tokens` Cookie に統合する

既存の `user_session_tokens` Cookie に匿名セッション用のトークンも格納する方式を検討した。

**不採用の理由**: `user_session_tokens` Cookie は `user_sessions` テーブルと密接に連携しており、未ログインユーザーのために `user_sessions` にレコードを作成すると、セッション管理全体（セッションクリーンアップ、認証ミドルウェアなど）に影響が及ぶ。`device_token` Cookie は独立した関心事として管理するほうが影響範囲が小さい。

### ログイン時の匿名フラグ移行処理

未ログインユーザーがログインした際に、`device_token` で設定されたフラグを `user_id` ベースのフラグに移行する処理を検討した。

**不採用の理由**: `device_token` Cookie はログイン前後で維持されるため、`device_token` で設定されたフラグはログイン後もそのまま有効。移行処理は不要であり、実装の複雑さを増すだけになる。ログインユーザーに全デバイス横断でフラグを設定したい場合は、`user_id` でフラグを別途設定する運用で対応する。

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

### フェーズ 1: データベースとリポジトリ層

<!--
feature_flagsテーブルのスキーマ変更とリポジトリの実装。
マイグレーション、モデル、sqlcクエリ、リポジトリをまとめて実装する。
-->

- [x] **1-1**: [Go] feature_flagsテーブルのスキーマ変更とリポジトリの実装
  - dbmateマイグレーションで `feature_flags` テーブルを変更（`user_id` をnullable化、`device_token` カラム追加、CHECK制約追加、ユニーク制約変更）
  - `internal/model/feature_flag.go` の `FeatureFlag` モデルを変更（`DeviceToken *string`、`UserID *UserID`）
  - `db/queries/feature_flags.sql` のsqlcクエリを変更（`IsFeatureFlagEnabledForDevice` を追加、既存クエリを更新）
  - sqlcコード生成を実行
  - `internal/repository/feature_flag_repository.go` のメソッドを変更（`IsEnabledForDevice` を追加、`IsEnabledBySessionToken` を削除）
  - リポジトリのテストを更新（`device_token` でのフラグ有効/無効、`user_id` でのフラグ有効/無効、両方のCookie確認の統合テスト）
  - テスト用ビルダー（`internal/testutil/feature_flag_builder.go`）を更新
  - **想定ファイル数**: 約 8 ファイル（実装 5 + テスト 1 + マイグレーション 1 + スキーマ 1）
  - **想定行数**: 約 250 行（実装 150 行 + テスト 100 行）

### フェーズ 2: リバースプロキシミドルウェア変更

<!--
ミドルウェアのフラグ判定ロジックを変更し、device_tokenとuser_idの両方に対応する。
-->

- [x] **2-1**: [Go] リバースプロキシミドルウェアのフラグ判定ロジック変更
  - `DeviceTokenCookieName` 定数を定義
  - `ensureDeviceToken` メソッドを追加（Cookie未設定時に `session.GenerateSecureToken()` で自動生成）
  - `featureFlagChecker` インターフェースを変更（`IsEnabledForDevice` に統合）
  - `isFeatureFlagEnabled` メソッドを変更（`device_token` Cookie と `user_session_tokens` Cookie の両方を読み取り、1クエリで判定）
  - テストを更新（`device_token` の自動生成、`device_token` でのフラグ有効/無効、`user_id` でのフラグ有効/無効、両Cookie無し時のフォールバック、エラー時のフォールバック）
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 1）
  - **想定行数**: 約 250 行（実装 100 行 + テスト 150 行）

### フェーズ 3: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [x] **3-1**: 仕様書の更新
  - `docs/specs/feature-flag/overview.md` を更新する
  - `device_token` / `user_id` 併用方式、採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **`viewers` テーブルの作成**: 現時点ではテーブルは不要。将来的にAPIで `viewer` 概念が必要になった場合に検討する
- **管理UI**: フラグの管理はpsqlまたはマイグレーションで行う
- **古いフラグレコードのクリーンアップ**: パフォーマンスが問題になった場合に検討する

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [internal/middleware/reverse_proxy.go](/workspace/go/internal/middleware/reverse_proxy.go) - 現在のリバースプロキシミドルウェアの実装
- [internal/repository/feature_flag_repository.go](/workspace/go/internal/repository/feature_flag_repository.go) - 現在のフィーチャーフラグリポジトリ
- [docs/specs/feature-flag/overview.md](/workspace/docs/specs/feature-flag/overview.md) - フィーチャーフラグ仕様書
- [docs/plans/3_done/202603/feature-flag.md](/workspace/docs/plans/3_done/202603/feature-flag.md) - フィーチャーフラグ作業計画書
