# フィーチャーフラグ 作業計画書

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

- [フィーチャーフラグ 仕様書](../specs/feature-flag/overview.md)（タスク完了後に作成予定）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

RailsからGoへの段階的移行において、Go版で再実装した機能を特定のユーザーにだけ先行公開するためのフィーチャーフラグ機能を導入する。

現在、Go版のページ編集画面は `/go/s/{space}/pages/{number}/edit` という開発用URLで提供されている。本番環境でGo版を使うためには、通常のURL（`/@{space}/pages/{number}/edit`）でGo版を提供しつつ、フラグが無効なユーザーには従来のRails版を提供する仕組みが必要になる。

フィーチャーフラグは `feature_flags` テーブルで管理し、`user_id` と `name` の組み合わせでレコードが存在すれば、そのユーザーに対してフラグが有効であることを意味する。リバースプロキシミドルウェアにフラグ判定機能を統合し、URLパターンに応じてGoまたはRailsにルーティングする。

### 関連タスク

- [@docs/plans/1_doing/page-edit-go-migration.md](../1_doing/page-edit-go-migration.md) - ページ編集画面のGo移行（本タスクの利用先）

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

- システムは `feature_flags` テーブルでユーザーごとのフラグ有効/無効を管理する
- システムはリクエストのURLパターンとユーザーのフラグ状態に基づいて、GoまたはRailsにルーティングする
- フラグが有効なユーザーが対象URLにアクセスした場合、Go版のハンドラーで処理される
- フラグが無効なユーザー（または未ログインユーザー）が対象URLにアクセスした場合、Rails版にプロキシされる
- 開発者はDBを直接操作（psqlやマイグレーション）してフラグを管理できる

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

- **パフォーマンス**: フィーチャーフラグ判定は対象URLパターンにマッチするリクエストのみで実行される。非対象パスでは追加のDB問い合わせは発生しない
- **信頼性**: フラグ判定でエラーが発生した場合は、安全側に倒してRails版にフォールバックする（Go版が表示されないほうがサービス断よりも安全）

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
- レコードが存在 = フラグ有効、レコードが存在しない = フラグ無効

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

// フィーチャーフラグ名の定数
// システム内で使用するフラグはここに定数として定義する
const (
    FeatureFlagGoPageEdit FeatureFlagName = "go_page_edit"
)
```

`internal/model/id.go` にドメインID型を追加:

```go
type FeatureFlagID string
func (id FeatureFlagID) String() string { return string(id) }

type FeatureFlagName string
func (n FeatureFlagName) String() string { return string(n) }
```

#### Repository

`internal/repository/feature_flag.go`:

```go
type FeatureFlagRepository struct {
    q *query.Queries
}

// IsEnabled は指定ユーザーに対してフラグが有効かどうかを返す
func (r *FeatureFlagRepository) IsEnabled(ctx context.Context, userID model.UserID, name model.FeatureFlagName) (bool, error)

// IsEnabledBySessionToken はセッショントークンからユーザーを特定し、フラグが有効かどうかを返す
// リバースプロキシミドルウェアから使用する（認証ミドルウェアの前に実行されるため、ユーザーIDが取得できない）
// セッショントークンは user_session_tokens Cookieの値
func (r *FeatureFlagRepository) IsEnabledBySessionToken(ctx context.Context, sessionToken string, name model.FeatureFlagName) (bool, error)
```

`IsEnabledBySessionToken` が使用するSQLクエリ:

```sql
-- name: IsFeatureFlagEnabledBySessionToken :one
SELECT EXISTS(
    SELECT 1 FROM feature_flags ff
    INNER JOIN user_sessions us ON ff.user_id = us.user_id
    WHERE us.token = $1 AND ff.name = $2
);
```

セッション管理は `user_sessions` テーブルで行われており、`user_session_tokens` Cookieの値が `user_sessions.token` カラムに対応する。Go版・Rails版の両方で同一のテーブルとCookieを共有している。

#### リバースプロキシミドルウェアの変更

`internal/middleware/reverse_proxy.go` を以下のように変更する:

**1. フィーチャーフラグ付きパスパターンの定義**

```go
// featureFlaggedPattern はフィーチャーフラグで制御するURLパターンを定義
type featureFlaggedPattern struct {
    pattern *regexp.Regexp
    flag    model.FeatureFlagName
}

// フィーチャーフラグで制御するURLパターンのリスト
// パターンを追加するには、このスライスに要素を追加する
var featureFlaggedPatterns []featureFlaggedPattern
```

初期状態ではパターンリストは空。ページ編集Go移行タスクで具体的なパターンを追加する。

**2. ルーティング判定の拡張**

`Middleware` メソッドに3段階のルーティング判定を追加:

```go
func (m *ReverseProxyMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. 常にGoで処理するパス（既存の動作）
        if m.isGoHandledPath(r.URL.Path) {
            next.ServeHTTP(w, r)
            return
        }

        // 2. フィーチャーフラグで制御するパス（新規追加）
        if flagName := m.getFeatureFlagForPath(r.URL.Path); flagName != model.FeatureFlagName("") {
            if m.isFeatureFlagEnabled(r, flagName) {
                next.ServeHTTP(w, r)
                return
            }
        }

        // 3. その他はすべてRailsにプロキシ（既存の動作）
        m.proxy.ServeHTTP(w, r)
    })
}
```

**3. フラグ判定メソッド**

```go
// getFeatureFlagForPath はURLパスに対応するフィーチャーフラグ名を返す
func (m *ReverseProxyMiddleware) getFeatureFlagForPath(path string) model.FeatureFlagName

// isFeatureFlagEnabled はリクエストのセッションCookieからユーザーを特定し、
// フィーチャーフラグが有効かどうかを判定する
func (m *ReverseProxyMiddleware) isFeatureFlagEnabled(r *http.Request, flagName model.FeatureFlagName) bool
```

`isFeatureFlagEnabled` の処理:

1. リクエストから `user_session_tokens` Cookieを取得
2. `FeatureFlagRepository.IsEnabledBySessionToken` でDB問い合わせ（`user_sessions.token` → `user_sessions.user_id` → `feature_flags` をJOIN）
3. エラー時またはCookieなし時は `false` を返す（Rails版にフォールバック）

**4. 依存関係の追加**

```go
type ReverseProxyMiddleware struct {
    railsURL        *url.URL
    proxy           *httputil.ReverseProxy
    cfg             *config.Config
    featureFlagRepo *repository.FeatureFlagRepository // 新規追加
}
```

`featureFlagRepo` が `nil` の場合（テスト時やフラグ機能不要時）は、フィーチャーフラグ判定をスキップする。

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

**ポイント**:

- リバースプロキシはミドルウェアチェーンの**最初**に配置される（既存の動作を維持）
- 認証ミドルウェアはリバースプロキシの**後**に実行される
- そのため、フラグ判定には `user_session_tokens` Cookieのトークン値を使用して `user_sessions` テーブル経由でDBに問い合わせる

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
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

### フェーズ 1: データベースとモデル層

<!--
マイグレーション、モデル、sqlcクエリ、リポジトリをまとめて実装する。
テーブル作成とリポジトリの実装は密接に関連するため、1つのPRにまとめる。
-->

- [x] **1-1**: [Go] feature_flagsテーブルのマイグレーションとモデル・リポジトリの実装
  - dbmateマイグレーションで `feature_flags` テーブルを作成
  - `internal/model/id.go` に `FeatureFlagID` 型を追加
  - `internal/model/feature_flag.go` に `FeatureFlag` モデルを追加
  - `internal/query/queries/feature_flags.sql` にsqlcクエリを追加（`IsEnabled`、`IsEnabledBySessionToken`）
  - sqlcコード生成を実行
  - `internal/repository/feature_flag.go` にリポジトリを実装
  - リポジトリのテストを追加
  - **想定ファイル数**: 約 10 ファイル（実装 6 + テスト 2 + マイグレーション 1 + スキーマ 1）
  - **想定行数**: 約 250 行（実装 150 行 + テスト 100 行）

### フェーズ 2: リバースプロキシ統合

<!--
リバースプロキシにフィーチャーフラグ判定を統合する。
初期状態ではフラグ付きパターンは空リスト。
具体的なURLパターンはページ編集Go移行タスクで追加する。
-->

- [x] **2-1**: [Go] リバースプロキシミドルウェアにフィーチャーフラグ判定を統合
  - `ReverseProxyMiddleware` に `FeatureFlagRepository` の依存を追加
  - `featureFlaggedPattern` 型とパターンリストを定義（初期状態は空リスト）
  - `getFeatureFlagForPath` メソッドを追加
  - `isFeatureFlagEnabled` メソッドを追加（`user_session_tokens` Cookie → `user_sessions` テーブル経由でDB問い合わせ）
  - `Middleware` メソッドのルーティング判定を拡張（3段階判定）
  - `cmd/server/main.go` のミドルウェア初期化を更新
  - テストを追加（パターンマッチング、フラグ有効/無効、エラー時のフォールバック）
  - **想定ファイル数**: 約 5 ファイル（実装 3 + テスト 2）
  - **想定行数**: 約 280 行（実装 150 行 + テスト 130 行）

### フェーズ 3: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [x] **3-1**: 仕様書の作成・更新
  - `docs/specs/feature-flag/overview.md` に仕様書を作成する
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **管理UI**: フラグの管理はpsqlまたはマイグレーションで行う。管理UIは必要になったタイミングで別タスクとして実装する
- **インメモリキャッシュ**: フラグ判定結果のキャッシュは、パフォーマンスが問題になった場合に追加する
- **パーセンテージベースのロールアウト**: ユーザー単位の手動フラグ管理で十分。自動ロールアウトは必要になったタイミングで検討する
- **Go版ハンドラーの通常URL登録**: ページ編集ハンドラーを `/@{space}/pages/{number}/edit` に登録する作業は、page-edit-go-migration計画のタスクとして実施する
- **フィーチャーフラグ付きURLパターンの追加**: 実際のURLパターン（`/@{space}/pages/{number}/edit` 等）の追加は、page-edit-go-migration計画のタスクとして実施する

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [internal/middleware/reverse_proxy.go](/workspace/go/internal/middleware/reverse_proxy.go) - 現在のリバースプロキシミドルウェアの実装
- [internal/middleware/auth.go](/workspace/go/internal/middleware/auth.go) - 認証ミドルウェアの実装
- [cmd/server/main.go](/workspace/go/cmd/server/main.go) - ミドルウェアの登録順序とルーティング構造
