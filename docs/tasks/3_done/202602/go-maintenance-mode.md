# メンテナンスモード 設計書

<!--
このテンプレートの使い方:
1. このファイルを `docs/designs/2_todo/` ディレクトリにコピー
   例: cp docs/designs/template.md docs/designs/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残しておくことを推奨

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 実装ガイドラインの参照

<!--
**重要**: 設計書を作成する前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン
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

### Rails版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - 全体的なコーディング規約

## 概要

Go版とRails版の両方にメンテナンスモード機能を実装します。メンテナンスモードが有効な場合、管理者IP以外のすべてのアクセスに対して503 Service Unavailableレスポンスとメンテナンスページを返します。

Go版はリバースプロキシを経由してRails版にリクエストを転送するため、Go版のメンテナンスモードはGo版で処理するリクエストにのみ適用されます。Rails版にプロキシされるリクエストはRails版のメンテナンスモードで処理されるため、両方の実装が必要です。

**目的**:

- サーバーメンテナンスやデプロイ時に、ユーザーに対して適切なメンテナンスページを表示する
- 管理者がメンテナンス中でもサイトにアクセスして動作確認できるようにする

**背景**:

- Annictではすでにメンテナンスモードが実装されており（Go版ミドルウェア + Rails版Rack::Rewrite）、実運用で活用されている
- Wikinoでは`config.go`にメンテナンスモード関連の設定（`MaintenanceMode`、`AdminIPs`）が定義済みだが、ミドルウェアが未実装
- Rails版にはメンテナンスモードの仕組みが存在しない
- デプロイやDB マイグレーション時など、サービスを一時停止する場面で必要

**参考実装**:

- Annict Go版: `/annict/go/internal/middleware/maintenance.go`
- Annict Rails版: `/annict/rails/config/application.rb`（Rack::Rewrite による `maintenance.html` 配信）

## 要件

### 機能要件

- 環境変数 `WIKINO_MAINTENANCE_MODE` を `on` に設定するとメンテナンスモードが有効になる
- メンテナンスモード中は、すべてのHTTPリクエストに対して503 Service Unavailableレスポンスを返す
- 環境変数 `WIKINO_ADMIN_IP` にカンマ区切りで指定したIPアドレスからのアクセスは、メンテナンスモード中でも通常通り処理される
- メンテナンスページにはサービス名と「メンテナンス中」のメッセージを表示する
- ヘルスチェックエンドポイント (`/health`) はメンテナンスモード中でも通常通り応答する（ロードバランサーの監視のため）
- Go版とRails版で同じ環境変数（`WIKINO_MAINTENANCE_MODE`、`WIKINO_ADMIN_IP`）を使用する
- Go版とRails版のメンテナンスページは同じデザイン・内容とする

### 非機能要件

#### セキュリティ

- 管理者IPの判定は、CF-Connecting-IP → X-Forwarded-For → X-Real-IP → RemoteAddr の優先順位で行う
  - Go版: 既存の `clientip.GetClientIP` を使用
  - Rails版: `HTTP_CF_CONNECTING_IP` → `HTTP_X_FORWARDED_FOR` → `HTTP_X_REAL_IP` → `REMOTE_ADDR` の順で取得（既存の `authenticatable.rb` のパターンを拡張）

#### パフォーマンス

- メンテナンスモードの判定はメモリ上の設定値チェックのみで、DBアクセスは発生しない

#### ユーザビリティ（UX）

- メンテナンスページは日本語で表示する（運用者向けサービスのため英語対応は不要）
- `Retry-After` レスポンスヘッダーで1時間後のリトライを推奨する（クローラー・ブラウザ向け）

## 設計

### Go版

#### 技術スタック

- **ミドルウェア**: Chi v5ミドルウェアとして実装
- **テンプレート**: templ（メンテナンスページ用）
- **クライアントIP取得**: 既存の `internal/clientip` パッケージを使用

#### API設計（ルーティング）

ルーティングの追加はありません。ミドルウェアとして全リクエストに適用されます。

#### ミドルウェアの配置順序

メンテナンスミドルウェアは、**リバースプロキシミドルウェアの直後、Method Overrideミドルウェアの直前**に配置します。

```
リバースプロキシ → メンテナンス → Method Override → Logger → ...
```

**配置理由**:

- **リバースプロキシの後**: Go版で処理するリクエストのみメンテナンスモードを適用する。Rails版にプロキシされるリクエストはRails版のメンテナンスモードで処理される
- **Method Overrideの前**: メンテナンスモード時にリクエストボディの解析（`r.ParseForm()`）は不要

```go
// main.go のミドルウェアチェーン
r := chi.NewRouter()

// 1. リバースプロキシ
if cfg.RailsAppURL != "" {
    r.Use(reverseProxyMiddleware.Middleware)
}

// 2. メンテナンスミドルウェア（新規追加）
maintenanceMW := middleware.NewMaintenanceMiddleware(cfg)
r.Use(maintenanceMW.Middleware)

// 3. 以下は既存のミドルウェアチェーン
r.Use(middleware.MethodOverride)
r.Use(chimiddleware.Logger)
// ...
```

#### コード設計

##### ディレクトリ構造

```
go/internal/
├── middleware/
│   ├── maintenance.go         # メンテナンスミドルウェア（新規）
│   └── maintenance_test.go    # テスト（新規）
└── templates/
    └── pages/
        └── maintenance/
            └── maintenance.templ   # メンテナンスページテンプレート（新規）
```

##### MaintenanceMiddleware

```go
// middleware/maintenance.go

// MaintenanceMiddleware はメンテナンスモード時にアクセスを制限するミドルウェア
type MaintenanceMiddleware struct {
    cfg *config.Config
}

func NewMaintenanceMiddleware(cfg *config.Config) *MaintenanceMiddleware

// Middleware はHTTPミドルウェアを返す
// メンテナンスモードが有効で、管理者IP以外からのアクセスの場合は503を返す
// ヘルスチェックエンドポイントはメンテナンスモード中でも通常処理する
func (m *MaintenanceMiddleware) Middleware(next http.Handler) http.Handler

// isAdminIP はリクエスト元IPが管理者IPかどうかをチェック
func (m *MaintenanceMiddleware) isAdminIP(r *http.Request) bool
```

**処理フロー**:

1. メンテナンスモードが無効 → 通常処理（`next.ServeHTTP`）
2. ヘルスチェックパス（`/health`）→ 通常処理（`next.ServeHTTP`）
3. 管理者IPからのアクセス → 通常処理（`next.ServeHTTP`）
4. 上記以外 → 503レスポンス + メンテナンスページ

##### メンテナンスページテンプレート

```go
// templates/pages/maintenance/maintenance.templ

// Page はメンテナンスページを表示するテンプレート
// 503 Service Unavailable と共に表示される
templ Page()
```

テンプレートは完全なHTMLドキュメントとして生成します（レイアウトテンプレートを使用しない）。メンテナンス中は他のシステムコンポーネント（セッション、DB等）が利用できない可能性があるため、依存を最小限にします。

**表示内容**:

- サービス名（Wikino）
- メンテナンス中であることを示すメッセージ
- しばらく待ってからアクセスしてほしいというお願い

#### テスト戦略

##### ミドルウェアテスト（`maintenance_test.go`）

以下のケースをテストします：

- メンテナンスモード無効時: 通常の200レスポンスが返ること
- メンテナンスモード有効 + 管理者IP: 通常の200レスポンスが返ること
- メンテナンスモード有効 + 一般IP: 503レスポンスが返ること
- メンテナンスモード有効 + ヘルスチェック: 通常の200レスポンスが返ること
- 複数の管理者IP: それぞれの管理者IPからアクセスできること
- X-Forwarded-For経由のIP判定
- CF-Connecting-IP経由のIP判定
- X-Real-IP経由のIP判定
- 管理者IP未設定時: すべてのアクセスで503が返ること
- レスポンスヘッダーの確認（Content-Type, Retry-After）
- メンテナンスページの内容確認

### Rails版

#### 技術スタック

- **ミドルウェア**: カスタムRackミドルウェアとして実装（gem追加不要）
- **メンテナンスページ**: 静的HTMLファイル（`public/maintenance.html`）
- **クライアントIP取得**: Rackの`env`ハッシュからHTTPヘッダーを直接取得

#### コード設計

##### ディレクトリ構造

```
rails/
├── lib/
│   └── maintenance_middleware.rb     # カスタムRackミドルウェア（新規）
├── public/
│   └── maintenance.html              # 静的メンテナンスページ（新規）
├── config/
│   └── application.rb                # ミドルウェアスタックに追加（修正）
└── spec/
    └── lib/
        └── maintenance_middleware_spec.rb  # テスト（新規）
```

##### MaintenanceMiddleware（Rackミドルウェア）

```ruby
# lib/maintenance_middleware.rb

class MaintenanceMiddleware
  def initialize(app)
  def call(env)

  private

  def maintenance_mode?
  def admin_ip?(env)
  def health_check?(env)
  def client_ip(env)
  def admin_ips
  def maintenance_response
end
```

**処理フロー**（Go版と同一）:

1. メンテナンスモードが無効 → 通常処理（`@app.call(env)`）
2. ヘルスチェックパス（`/health`）→ 通常処理（`@app.call(env)`）
3. 管理者IPからのアクセス → 通常処理（`@app.call(env)`）
4. 上記以外 → 503レスポンス + `public/maintenance.html` の内容を返す

**IP判定**: Rackの`env`ハッシュを使用

```ruby
def client_ip(env)
  env["HTTP_CF_CONNECTING_IP"] ||
    env["HTTP_X_FORWARDED_FOR"]&.split(",")&.first&.strip ||
    env["HTTP_X_REAL_IP"] ||
    env["REMOTE_ADDR"]
end
```

**環境変数の読み取り**: Go版の`config.go`のように起動時にパースするのではなく、`ENV`から直接読み取ります（Rackミドルウェアはアプリケーション設定に依存しないため）。

```ruby
def maintenance_mode?
  ENV["WIKINO_MAINTENANCE_MODE"] == "on"
end

def admin_ips
  ENV.fetch("WIKINO_ADMIN_IP", "").split(",").map(&:strip).reject(&:empty?)
end
```

##### ミドルウェアの配置

`config/application.rb` でミドルウェアスタックの**最初**に配置します。

```ruby
# config/application.rb
require_relative "../lib/maintenance_middleware"

module Wikino
  class Application < Rails::Application
    # ...

    config.middleware.insert_before 0, MaintenanceMiddleware
  end
end
```

**配置理由**: 全リクエストをメンテナンスモードで捕捉するため、他のミドルウェア（認証、CSRF等）より前に配置します。

##### メンテナンスページ（静的HTML）

`public/maintenance.html` に静的HTMLファイルとして配置します。Go版のtemplテンプレートと同じデザイン・内容を使用します。

- 完全な静的HTML（Ruby処理不要）
- インラインスタイル（外部CSS依存なし）
- Go版と同一のデザインとメッセージ
- 既存の `public/500.html` 等と同じパターン

#### テスト戦略

##### ミドルウェアテスト（`maintenance_middleware_spec.rb`）

RSpecでRackミドルウェアを直接テストします。Go版と同等のテストケースをカバーします：

- メンテナンスモード無効時: 通常のレスポンスが返ること
- メンテナンスモード有効 + 管理者IP: 通常のレスポンスが返ること
- メンテナンスモード有効 + 一般IP: 503レスポンスが返ること
- メンテナンスモード有効 + ヘルスチェック: 通常のレスポンスが返ること
- 複数の管理者IP: それぞれの管理者IPからアクセスできること
- 各IPヘッダー経由のIP判定（HTTP_CF_CONNECTING_IP、HTTP_X_FORWARDED_FOR、HTTP_X_REAL_IP）
- 管理者IP未設定時: すべてのアクセスで503が返ること
- レスポンスヘッダーの確認（Content-Type, Retry-After）
- メンテナンスページの内容確認

### 環境変数

| 環境変数                  | 説明                             | 必須   | 例                     |
| ------------------------- | -------------------------------- | ------ | ---------------------- |
| `WIKINO_MAINTENANCE_MODE` | `on` でメンテナンスモード有効化  | いいえ | `on`                   |
| `WIKINO_ADMIN_IP`         | 管理者IPアドレス（カンマ区切り） | いいえ | `192.168.1.1,10.0.0.1` |

Go版では `config.go` ですでに定義・パース済みです。Rails版では `ENV` から直接読み取ります。

## タスクリスト

### フェーズ 1: Go版メンテナンスモードの実装

- [x] **1-1**: [Go] メンテナンスページテンプレートの作成
  - depguardのアーキテクチャルール（MiddlewareはTemplatesに依存不可）に従い、templテンプレートではなくミドルウェア内のインラインHTMLとして実装
  - `internal/middleware/maintenance.go` 内の `renderMaintenanceHTML()` 関数として実装
  - 完全なHTMLドキュメント（レイアウト不使用、外部依存なし）
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 60 行（実装 60 行 + テスト 0 行）

- [x] **1-2**: [Go] メンテナンスミドルウェアの実装
  - `internal/middleware/maintenance.go` を作成
  - `MaintenanceMiddleware` 構造体と `Middleware` メソッドを実装
  - `isAdminIP` メソッドで管理者IP判定を実装（`clientip.GetClientIP` を使用）
  - ヘルスチェックエンドポイント (`/health`) のバイパス処理
  - `Retry-After` レスポンスヘッダーの設定
  - `internal/middleware/maintenance_test.go` にテストを作成
  - `cmd/server/main.go` にミドルウェアを追加
  - **想定ファイル数**: 約 4 ファイル（実装 2 + テスト 1 + main.go修正 1）
  - **想定行数**: 約 280 行（実装 80 行 + テスト 200 行）

### フェーズ 2: Rails版メンテナンスモードの実装

- [x] **2-1**: [Rails] メンテナンスミドルウェアの実装
  - `lib/maintenance_middleware.rb` にカスタムRackミドルウェアを作成
  - `public/maintenance.html` に静的メンテナンスページを作成（Go版と同じデザイン）
  - `config/application.rb` にミドルウェアを登録
  - `spec/lib/maintenance_middleware_spec.rb` にテストを作成
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 280 行（実装 80 行 + メンテナンスHTML 60 行 + テスト 140 行）

### 実装しない機能（スコープ外）

以下の機能は今回の実装では**実装しません**：

- **メンテナンスページの多言語対応**: 運用者向けサービスのため、日本語のみで十分
- **メンテナンスモードのWeb UI管理**: 環境変数による制御で十分。Web UIは過剰な機能
- **メンテナンス終了予定時刻の表示**: 運用開始後に必要性を検討
- **静的ファイル（CSS/JS）のバイパス**: メンテナンスページは外部依存なしのインラインスタイルで実装するため不要

## 参考資料

- Annict Go版メンテナンスミドルウェア: `/annict/go/internal/middleware/maintenance.go`
- Annict Rails版メンテナンスモード: `/annict/rails/config/application.rb`（97-104行目）
- Wikino 既存の設定パーサー: `/workspace/go/internal/config/config.go`（42-44行目、117-124行目）
- Wikino クライアントIP取得: `/workspace/go/internal/clientip/clientip.go`
- Wikino Rails版IP取得パターン: `/workspace/rails/app/controllers/controller_concerns/authenticatable.rb`（48-49行目）
- Wikino Rails版既存エラーページ: `/workspace/rails/public/500.html`
