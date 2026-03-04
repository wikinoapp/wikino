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

フィーチャーフラグは、RailsからGoへの段階的移行において、Go版で再実装した機能を特定のユーザーにだけ先行公開するための仕組みである。`feature_flags` テーブルで `user_id` と `name` の組み合わせによりユーザーごとのフラグ有効/無効を管理する。レコードが存在すればフラグ有効、存在しなければフラグ無効という単純なモデルで動作する。

リバースプロキシミドルウェアにフラグ判定機能が統合されており、URLパターンに応じてGoまたはRailsにルーティングする。フラグが有効なユーザーはGo版のハンドラーで処理され、フラグが無効なユーザー（または未ログインユーザー）はRails版にプロキシされる。

**目的**:

- Go版で再実装した機能を、全ユーザーに公開する前に特定のユーザーで検証できるようにする
- 通常のURL（`/@{space}/pages/{number}/edit` など）でGo版を提供しつつ、フラグが無効なユーザーには従来のRails版を提供する

**背景**:

- Go版のページ編集画面は `/go/s/{space}/pages/{number}/edit` という開発用URLで提供されている。本番環境でGo版を使うためには、通常のURLでGo版を提供する仕組みが必要になる
- 段階的なロールアウト（まず開発者 → 次にベータユーザー → 最後に全ユーザー）のために、ユーザー単位の制御が必要

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

### フラグの管理

- システムは `feature_flags` テーブルでユーザーごとのフラグ有効/無効を管理する
- レコードが存在 = フラグ有効、レコードが存在しない = フラグ無効
- 開発者はDBを直接操作（psqlやマイグレーション）してフラグを管理できる

### ルーティング判定

- システムはリクエストのURLパターンとユーザーのフラグ状態に基づいて、GoまたはRailsにルーティングする
- リバースプロキシミドルウェアは以下の3段階でルーティングを判定する:
  1. **常にGoで処理するパス**: ホワイトリストに一致するパスはGoのハンドラーで処理される
  2. **フィーチャーフラグで制御するパス**: URLパターンに一致し、かつフラグが有効なユーザーはGoのハンドラーで処理される
  3. **その他**: 上記以外はすべてRailsにプロキシされる
- フラグが無効なユーザー、未ログインユーザー、またはフラグ判定でエラーが発生した場合は、Rails版にフォールバックする

### フラグ付きURLパターン

- フラグで制御するURLパターンは `featureFlaggedPatterns` スライスで管理する
- 現時点ではパターンリストは空であり、具体的なURLパターンはページ編集Go移行タスクで追加される

### パフォーマンス

- フィーチャーフラグ判定は、`featureFlaggedPatterns` に登録されたURLパターンにマッチするリクエストのみで実行される。非対象パスでは追加のDB問い合わせは発生しない

### 信頼性

- フラグ判定でエラーが発生した場合は、Rails版にフォールバックする（Go版が表示されないほうがサービス断よりも安全）
- `featureFlagRepo` が `nil` の場合（テスト時やフラグ機能不要時）は、フィーチャーフラグ判定をスキップする

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
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_feature_flags_user_id ON feature_flags(user_id);
CREATE INDEX idx_feature_flags_name ON feature_flags(name);
```

- `user_id` + `name` のユニーク制約で、同一ユーザーに同一フラグが重複登録されることを防ぐ
- `name` にはフラグ名を格納する（例: `go_page_edit`）

### ルーティングの流れ

```
リクエスト到着
  ↓
[リバースプロキシミドルウェア]
  ├─ 常にGoのパス？ → Yes → Go側ミドルウェアチェーン → ハンドラー
  ├─ フィーチャーフラグ付きパス？
  │   ├─ Yes + フラグ有効 → Go側ミドルウェアチェーン → ハンドラー
  │   └─ Yes + フラグ無効/未ログイン/エラー → Railsにプロキシ
  └─ その他 → Railsにプロキシ
```

- リバースプロキシはミドルウェアチェーンの**最初**に配置される
- 認証ミドルウェアはリバースプロキシの**後**に実行される
- そのため、フラグ判定には `user_session_tokens` Cookieのトークン値を使用して `user_sessions` テーブル経由でDBに問い合わせる

### コード設計

#### Model

`internal/model/feature_flag.go`:

```go
type FeatureFlag struct {
    ID        FeatureFlagID
    UserID    UserID
    Name      FeatureFlagName
    CreatedAt time.Time
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

// IsEnabled は指定ユーザーに対してフラグが有効かどうかを返す
func (r *FeatureFlagRepository) IsEnabled(ctx context.Context, userID model.UserID, name model.FeatureFlagName) (bool, error)

// IsEnabledBySessionToken はセッショントークンからユーザーを特定し、フラグが有効かどうかを返す
func (r *FeatureFlagRepository) IsEnabledBySessionToken(ctx context.Context, sessionToken string, name model.FeatureFlagName) (bool, error)
```

`IsEnabledBySessionToken` が使用するSQLクエリ:

```sql
SELECT EXISTS(
    SELECT 1 FROM feature_flags ff
    INNER JOIN user_sessions us ON ff.user_id = us.user_id
    WHERE us.token = $1 AND ff.name = $2
);
```

セッション管理は `user_sessions` テーブルで行われており、`user_session_tokens` Cookieの値が `user_sessions.token` カラムに対応する。Go版・Rails版の両方で同一のテーブルとCookieを共有している。

#### リバースプロキシミドルウェア

`internal/middleware/reverse_proxy.go` にフィーチャーフラグ判定を統合:

```go
// featureFlagChecker はフィーチャーフラグの有効判定を行うインターフェース
type featureFlagChecker interface {
    IsEnabledBySessionToken(ctx context.Context, sessionToken string, name model.FeatureFlagName) (bool, error)
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

`isFeatureFlagEnabled` の処理:

1. `featureFlagRepo` が `nil` なら `false` を返す
2. リクエストから `user_session_tokens` Cookieを取得
3. `FeatureFlagRepository.IsEnabledBySessionToken` でDB問い合わせ
4. エラー時またはCookieなし時は `false` を返す（Rails版にフォールバック）

### ファイル構成

```
go/
├── db/
│   ├── migrations/
│   │   └── 20260301154347_create_feature_flags.sql  # マイグレーション
│   └── queries/
│       └── feature_flags.sql                        # sqlcクエリ定義
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

**不採用の理由**: 現在のミドルウェア順序は「リバースプロキシ → Method Override → CSRF → ...」となっており、リバースプロキシの前にbody-consumingなミドルウェアを配置するとRailsへのプロキシ時にリクエストボディが空になる問題がある。`SetUser`自体はbodyを消費しないが、ミドルウェア順序の変更は影響範囲が大きいため、リバースプロキシ内でセッションCookieを直接読む方式を採用した。

### Railsでのリダイレクト方式

Rails側でフラグを判定し、Go版URLにリダイレクトする方法を検討した。

**不採用の理由**: 追加のHTTPリダイレクトが発生し、Rails・Go両方の変更が必要になる。リバースプロキシでの判定であれば、Go側のみの変更で完結する。

### インメモリキャッシュ

フラグ判定結果をインメモリにキャッシュする方法を検討した。

**不採用の理由**: YAGNI。フラグ判定はフィーチャーフラグ付きパスにマッチするリクエストのみで実行され、頻度はページ編集画面へのアクセス程度。単純なDBクエリ（インデックス付き）で十分な性能が得られる。パフォーマンスが問題になった場合に追加する。

### 管理UI

フラグの有効/無効を管理するWebUIを検討した。

**不採用の理由**: 現時点ではフラグの変更頻度が低く（開発者が手動で設定する程度）、psqlやマイグレーションで十分に管理できる。管理UIは必要になったタイミングで別タスクとして実装する。

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [internal/middleware/reverse_proxy.go](/workspace/go/internal/middleware/reverse_proxy.go) - リバースプロキシミドルウェアの実装
- [internal/repository/feature_flag_repository.go](/workspace/go/internal/repository/feature_flag_repository.go) - フィーチャーフラグリポジトリの実装
- [internal/model/feature_flag.go](/workspace/go/internal/model/feature_flag.go) - フィーチャーフラグモデル
