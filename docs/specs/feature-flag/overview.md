# フィーチャーフラグ 仕様書

<!--
このテンプレートの使い方:
1. 操作対象のモデルに対応するディレクトリを `docs/specs/` 配下に作成（例: `docs/specs/page/`）
2. このファイルをそのディレクトリにコピー（例: cp docs/specs/template.md docs/specs/page/create.md）
3. [機能名] などのプレースホルダーを実際の内容に置き換え
4. 各セクションのガイドラインに従って記述
5. コメント（ `\<!-- ... --\>` ）はガイドラインとして残してください

**ファイルの配置ルール**:
- 仕様書は操作対象のモデル（名詞）ごとにディレクトリを分け、機能（動詞）をファイル名にする
  - 例: `docs/specs/user/sign-up.md`、`docs/specs/page/create.md`
- モデルに分類しにくい横断的な機能は、その機能自体を名詞としてディレクトリにする
  - 例: `docs/specs/search/full-text.md`
- モデルの定義・状態遷移・他モデルとの関係を記述する場合は `overview.md` を作成する
  - `overview.md` はモデルの静的な性質（「これは何か」）を書く場所
  - 操作に紐づく仕様（バリデーション、権限など）は各機能の仕様書に書く
- 詳細は [@docs/README.md](/workspace/docs/README.md) を参照

**仕様書の性質**:
- 仕様書は「現在のシステムの状態」を記述するドキュメントです
- 実装が完了したら、仕様書を最新の状態に更新してください
- 過去の状態はGit履歴で参照できるため、仕様書には常に現在の状態のみを記述します

**作業計画書との関係**:
- 新しい機能の場合: `docs/plans/` の作業計画書に概要・要件・設計を記述し、タスク完了後にこの仕様書を作成します
- 既存機能の変更の場合: `docs/plans/` の作業計画書に変更内容を記述し、タスク完了後にこの仕様書を更新します

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 概要

<!--
ガイドライン:
- この機能が現在「どのように動いているか」を簡潔に説明
- なぜこの仕組みになっているかの背景も記述
- 2-3段落程度で簡潔に
-->

フィーチャーフラグは、RailsからGoへの段階的移行において、Go版で再実装した機能を特定のユーザーやデバイスにだけ先行公開するための仕組みである。`feature_flags` テーブルで `device_token`（Cookieに保存されるデバイス識別子）または `user_id` と `name` の組み合わせによりフラグの有効/無効を管理する。レコードが存在すればフラグ有効、存在しなければフラグ無効という単純なモデルで動作する。

リバースプロキシミドルウェアにフラグ判定機能が統合されており、URLパターンに応じてGoまたはRailsにルーティングする。フラグが有効なユーザー/デバイスはGo版のハンドラーで処理され、フラグが無効な場合やフラグ判定でエラーが発生した場合はRails版にプロキシされる。未ログインユーザーに対しても `device_token` 経由でフラグの制御が可能である。

**目的**:

- Go版で再実装した機能を、全ユーザーに公開する前に特定のユーザーやデバイスで検証できるようにする
- 通常のURL（`/@{space}/pages/{number}/edit` など）でGo版を提供しつつ、フラグが無効なユーザーには従来のRails版を提供する
- 未ログインユーザーに対しても、`device_token` Cookie経由でフラグを設定できるようにする

**背景**:

- Go版のページ編集画面は `/go/s/{space}/pages/{number}/edit` という開発用URLで提供されている。本番環境でGo版を使うためには、通常のURLでGo版を提供する仕組みが必要になる
- 段階的なロールアウト（まず開発者 → 次にベータユーザー → 最後に全ユーザー）のために、ユーザー単位・デバイス単位の制御が必要
- 未ログインユーザーがフィーチャーフラグ付きURLにアクセスした場合にも、Go版への切り替えを可能にする必要がある

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

### フラグの管理

- システムは `feature_flags` テーブルで `device_token` または `user_id` によるフラグの有効/無効を管理する
- レコードが存在 = フラグ有効、レコードが存在しない = フラグ無効
- 開発者はDBを直接操作（psqlやマイグレーション）してフラグを管理できる（`device_token` による設定、`user_id` による設定の両方）

### 2種類のフラグレコード

| レコード種類       | `device_token` | `user_id` | 対象                     | 用途                                       |
| ------------------ | -------------- | --------- | ------------------------ | ------------------------------------------ |
| デバイス単位フラグ | 設定あり       | NULL      | そのデバイスのみ         | 未ログインユーザー、特定デバイスでのテスト |
| ユーザー単位フラグ | NULL           | 設定あり  | そのユーザーの全デバイス | 開発者やベータユーザーへのロールアウト     |

### device_token Cookie

- システムはリバースプロキシミドルウェアで、`device_token` Cookieが存在しない場合に自動生成してレスポンスにセットする
- `device_token` はログイン状態に関わらずデバイス（ブラウザ）を識別するために使用する
- Cookie属性: `HttpOnly: true`, `Secure: true`（本番環境）, `SameSite: Lax`, `MaxAge: 10年`
- トークン生成には `session.GenerateSecureToken()` を使用する（24バイトのランダムデータをBase64 URL-safeエンコードした32文字の文字列）
- ログイン前後で同じCookieが維持されるため、フラグの移行処理は不要

### ルーティング判定

- システムはリクエストのURLパターンとフラグ状態に基づいて、GoまたはRailsにルーティングする
- リバースプロキシミドルウェアは以下の3段階でルーティングを判定する:
  1. **常にGoで処理するパス**: ホワイトリストに一致するパスはGoのハンドラーで処理される
  2. **フィーチャーフラグで制御するパス**: URLパターンに一致し、かつ `device_token` または `user_id` のフラグが有効ならGoのハンドラーで処理される
  3. **その他**: 上記以外はすべてRailsにプロキシされる
- フラグが無効な場合、両方のCookieが存在しない場合、またはフラグ判定でエラーが発生した場合は、Rails版にフォールバックする

### フラグ名の命名規則

- フラグ名はスネークケース（アンダースコア区切り）で命名する（例: `go_suggestion`, `go_page_edit`）
- Go版への移行で使用するフラグには `go_` プレフィックスを付ける
- フラグ名の定数は `internal/model/feature_flag.go` に定義する

### フラグ付きURLパターン

- フラグで制御するURLパターンは `featureFlaggedPatterns` スライスで管理する
- 現時点ではパターンリストは空であり、具体的なURLパターンはページ編集Go移行タスクで追加される

### パフォーマンス

- フィーチャーフラグ判定は、`featureFlaggedPatterns` に登録されたURLパターンにマッチするリクエストのみで実行される。非対象パスでは追加のDB問い合わせは発生しない
- 1クエリで `device_token` と `user_id` の両方をチェックする

### 信頼性

- フラグ判定でエラーが発生した場合は、Rails版にフォールバックする（Go版が表示されないほうがサービス断よりも安全）
- `featureFlagRepo` が `nil` の場合（テスト時やフラグ機能不要時）は、フィーチャーフラグ判定をスキップする

### セキュリティ

- `device_token` Cookieは `session.GenerateSecureToken()` による安全なランダム値を使用し、`HttpOnly` + `Secure` + `SameSite=Lax` で設定する

## 設計

<!--
ガイドライン:
- 現在の技術的な実装の詳細を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - 技術スタック（使用するライブラリ、フレームワーク、ツールなど）
  - アーキテクチャ（システム全体の構成、コンポーネント間の関係など）
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - セキュリティ設計（認証・認可、トークン管理、Rate Limitingなど）
  - コード設計（パッケージ構成、主要な構造体、インターフェースなど）
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### データベース設計

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

- `device_token` と `user_id` はどちらもnullableだが、CHECK制約で少なくとも一方がNOT NULLであることを保証する
- `(device_token, name)` と `(user_id, name)` の2つのユニーク制約で、同一対象に同一フラグが重複登録されることを防ぐ
- PostgreSQLではUNIQUE制約のNULL値は重複として扱われないため、`device_token` がNULLのレコードは `UNIQUE(device_token, name)` 制約に影響しない。同様に `user_id` がNULLのレコードは `UNIQUE(user_id, name)` 制約に影響しない
- `name` にはフラグ名を格納する（例: `go_page_edit`）

### ルーティングの流れ

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

- リバースプロキシはミドルウェアチェーンの**最初**に配置される
- 認証ミドルウェアはリバースプロキシの**後**に実行される
- そのため、フラグ判定には `device_token` Cookieと `user_session_tokens` Cookieの値を直接使用してDBに問い合わせる

### コード設計

#### Model

`internal/model/feature_flag.go`:

```go
type FeatureFlag struct {
    ID          FeatureFlagID
    DeviceToken *string       // nullable: デバイス単位フラグの場合に設定
    UserID      *UserID       // nullable: ユーザー単位フラグの場合に設定
    Name        FeatureFlagName
    CreatedAt   time.Time
}
```

`internal/model/id.go` にドメインID型を定義:

```go
type FeatureFlagID string
type FeatureFlagName string
```

#### Repository

`internal/repository/feature_flag_repository.go`:

```go
type FeatureFlagRepository struct {
    q *query.Queries
}

// IsEnabled は指定ユーザーに対してフラグが有効かどうかを返す（内部利用・テスト用）
func (r *FeatureFlagRepository) IsEnabled(ctx context.Context, userID model.UserID, name model.FeatureFlagName) (bool, error)

// IsEnabledForDevice はデバイストークンまたはログインセッション経由でフラグが有効かどうかを返す
func (r *FeatureFlagRepository) IsEnabledForDevice(ctx context.Context, deviceToken string, sessionToken string, name model.FeatureFlagName) (bool, error)
```

`IsEnabledForDevice` が使用するSQLクエリ:

```sql
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

- `$1` = `device_token` Cookieの値（空文字列は何にもマッチしない）
- `$2` = `user_session_tokens` Cookieの値（空文字列は何にもマッチしない）
- `$3` = フラグ名
- 1クエリで `device_token` と `user_id`（セッショントークン経由）の両方をチェックする

セッション管理は `user_sessions` テーブルで行われており、`user_session_tokens` Cookieの値が `user_sessions.token` カラムに対応する。Go版・Rails版の両方で同一のテーブルとCookieを共有している。

#### リバースプロキシミドルウェア

`internal/middleware/reverse_proxy.go` にフィーチャーフラグ判定を統合:

```go
// DeviceTokenCookieName はデバイス（ブラウザ）識別用のCookieキー名
const DeviceTokenCookieName = "device_token"

// featureFlagChecker はフィーチャーフラグの有効判定を行うインターフェース
type featureFlagChecker interface {
    IsEnabledForDevice(ctx context.Context, deviceToken string, sessionToken string, name model.FeatureFlagName) (bool, error)
}

// featureFlaggedPattern はフィーチャーフラグで制御するURLパターンを定義
type featureFlaggedPattern struct {
    pattern *regexp.Regexp
    flag    model.FeatureFlagName
}

// フィーチャーフラグで制御するURLパターンのリスト
var featureFlaggedPatterns []featureFlaggedPattern
```

`ReverseProxyMiddleware` 構造体に `featureFlagRepo featureFlagChecker` フィールドを持ち、`isFeatureFlagEnabled` メソッドで判定を行う。

`Middleware` メソッドの処理:

1. `device_token` Cookieが存在しない場合、`ensureDeviceToken` で自動生成してレスポンスにセットする
2. 常にGoのパスかどうかを判定する
3. フィーチャーフラグ付きパスの場合、`isFeatureFlagEnabled` で判定する

`isFeatureFlagEnabled` の処理:

1. `featureFlagRepo` が `nil` なら `false` を返す
2. リクエストから `device_token` Cookieと `user_session_tokens` Cookieを取得
3. どちらのCookieも存在しない場合は `false` を返す（Rails版にフォールバック）
4. `FeatureFlagRepository.IsEnabledForDevice` で1クエリで判定
5. エラー時は `false` を返す（Rails版にフォールバック）

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

### ファイル構成

```
go/
├── db/
│   ├── migrations/
│   │   ├── 20260301154347_create_feature_flags.sql              # 初期マイグレーション
│   │   └── 20260313062541_add_device_token_to_feature_flags.sql # device_token追加
│   └── queries/
│       └── feature_flags.sql                                    # sqlcクエリ定義
└── internal/
    ├── model/
    │   ├── feature_flag.go                          # FeatureFlagモデル
    │   └── id.go                                    # FeatureFlagID, FeatureFlagName型
    ├── query/
    │   └── feature_flags.sql.go                     # sqlc生成コード
    ├── repository/
    │   ├── feature_flag_repository.go               # リポジトリ
    │   └── feature_flag_repository_test.go          # リポジトリテスト
    ├── middleware/
    │   ├── reverse_proxy.go                         # フラグ判定統合済みリバースプロキシ
    │   └── reverse_proxy_test.go                    # ミドルウェアテスト
    └── testutil/
        └── feature_flag_builder.go                  # テスト用ビルダー
```

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### 環境変数/設定ファイルベースのフラグ管理

環境変数や設定ファイルでフラグを管理する方法を検討した。

**不採用の理由**: ユーザー単位の制御ができず、「全ユーザーに対してON/OFF」しかできない。段階的なロールアウト（まず開発者 → 次にベータユーザー → 最後に全ユーザー）ができないため不採用。

### 認証ミドルウェアをリバースプロキシの前に配置する

認証ミドルウェア（`SetUser`）をリバースプロキシの前に配置し、ユーザーIDを使ってフラグ判定する方法を検討した。

**不採用の理由**: 現在のミドルウェア順序は「リバースプロキシ → Method Override → CSRF → ...」となっており、リバースプロキシの前にbody-consumingなミドルウェアを配置するとRailsへのプロキシ時にリクエストボディが空になる問題がある。`SetUser`自体はbodyを消費しないが、ミドルウェア順序の変更は影響範囲が大きいため、リバースプロキシ内でCookieを直接読む方式を採用した。

### Railsでのリダイレクト方式

Rails側でフラグを判定し、Go版URLにリダイレクトする方法を検討した。

**不採用の理由**: 追加のHTTPリダイレクトが発生し、Rails・Go両方の変更が必要になる。リバースプロキシでの判定であれば、Go側のみの変更で完結する。

### インメモリキャッシュ

フラグ判定結果をインメモリにキャッシュする方法を検討した。

**不採用の理由**: YAGNI。フラグ判定はフィーチャーフラグ付きパスにマッチするリクエストのみで実行され、頻度はページ編集画面へのアクセス程度。単純なDBクエリ（インデックス付き）で十分な性能が得られる。パフォーマンスが問題になった場合に追加する。

### 管理UI

フラグの有効/無効を管理するWebUIを検討した。

**不採用の理由**: 現時点ではフラグの変更頻度が低く（開発者が手動で設定する程度）、psqlやマイグレーションで十分に管理できる。管理UIは必要になったタイミングで別タスクとして実装する。

### 匿名ユーザー専用テーブル（`feature_flags_anonymous`）の新設

匿名ユーザー用のフラグを別テーブル（`feature_flags_anonymous`）で管理する方式を検討した。

**不採用の理由**: フラグ管理テーブルが2つに分散し、管理の複雑さが増す。1つのテーブルで `device_token` と `user_id` の両方を管理するほうがシンプルで保守しやすい。

### `viewers` テーブルの新設

`device_token` と `user_id` のマッピングを管理する `viewers` テーブルを作成し、`feature_flags` から参照する方式を検討した。

**不採用の理由**: フィーチャーフラグの用途においては、`device_token` はCookieの値として十分に機能し、テーブルで管理する必要がない。追加のテーブルとJOINが必要になり、複雑さが増す。

### Viewer 概念モデルの導入

サイト訪問者を概念的にモデリングし、Viewer（全訪問者）/ User（ログイン済み）/ Visitor（未ログイン）と定義する方式を検討した。Cookie名・カラム名を `viewer_token` とし、概念モデルに紐づける設計だった。

**不採用の理由**: 将来的にWeb APIを提供する際、GitHubのGraphQL APIのように `viewer` をアクセストークンの発行者を指す用語として使いたい（例: `viewerCanCreatePage`）。フィーチャーフラグのCookie識別子とAPIの `viewer` 概念が衝突するため、Cookie/カラム名は実態に即した `device_token` とし、概念モデルは導入しないこととした。

### 匿名トークンを `user_session_tokens` Cookie に統合する

既存の `user_session_tokens` Cookieに匿名セッション用のトークンも格納する方式を検討した。

**不採用の理由**: `user_session_tokens` Cookieは `user_sessions` テーブルと密接に連携しており、未ログインユーザーのために `user_sessions` にレコードを作成すると、セッション管理全体（セッションクリーンアップ、認証ミドルウェアなど）に影響が及ぶ。`device_token` Cookieは独立した関心事として管理するほうが影響範囲が小さい。

### ログイン時の匿名フラグ移行処理

未ログインユーザーがログインした際に、`device_token` で設定されたフラグを `user_id` ベースのフラグに移行する処理を検討した。

**不採用の理由**: `device_token` Cookieはログイン前後で維持されるため、`device_token` で設定されたフラグはログイン後もそのまま有効。移行処理は不要であり、実装の複雑さを増すだけになる。ログインユーザーに全デバイス横断でフラグを設定したい場合は、`user_id` でフラグを別途設定する運用で対応する。

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [internal/middleware/reverse_proxy.go](/workspace/go/internal/middleware/reverse_proxy.go) - リバースプロキシミドルウェアの実装
- [internal/repository/feature_flag_repository.go](/workspace/go/internal/repository/feature_flag_repository.go) - フィーチャーフラグリポジトリの実装
- [internal/model/feature_flag.go](/workspace/go/internal/model/feature_flag.go) - フィーチャーフラグモデル
