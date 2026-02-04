# Go への移行 (トップページ編) 設計書

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

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
設計書作成の段階でガイドラインに準拠していることを確認してください。
-->

### Go 版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン（**ファイル名は標準の 8 種類のみ**）
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

## 概要

Go 版 Wikino にトップページ（ウェルカムページ）を実装します。未ログインユーザーがルート URL（`/`）にアクセスした際に表示されるランディングページです。

**目的**:

- Rails 版から Go 版への段階的移行として、トップページを実装する
- サービスの機能紹介、サインアップ・サインインへの導線を提供する

**背景**:

- トップページはサービスの顔であり、新規ユーザーの第一印象を決める重要なページ
- Go 版への移行において、認証関連ページ（サインイン、サインアップ、パスワードリセット）の実装が完了したため、次のステップとしてトップページを移行する

## 要件

### 機能要件

- ユーザーはルート URL（`/`）でトップページにアクセスできる
- トップページでは以下の内容を表示する：
  - ヒーローセクション（タイトル、説明文、「無料で始められる」、サインアップ・サインインボタン、スクリーンショット）
  - 機能紹介セクション（Markdown、リンク記法、リンク機能、公開・共有機能）
  - CTA（Call to Action）セクション
  - 開発者・コミュニティセクション（オープンソース、開発者について、各種ソーシャルメディアへのリンク）
  - フッター（利用規約、プライバシーポリシーへのリンクのみ）
- 以下は表示しない：
  - サイドバー
  - モバイル向けの下部ナビゲーションバー
- ログイン済みユーザーの場合は `/home` にリダイレクトする

### 非機能要件

#### 国際化

- 日本語と英語の両言語に対応
- すべてのテキストは翻訳ファイルから取得する

#### パフォーマンス

- 画像は `loading="lazy"` で遅延読み込みを行う

## 設計

### 技術スタック

- **HTTP ルーター**: `chi/v5`
- **テンプレート**: `templ`
- **CSS**: Tailwind CSS v4

### API 設計（ルーティング）

| URL | メソッド | ハンドラー     | 説明                                 |
| --- | -------- | -------------- | ------------------------------------ |
| `/` | GET      | `welcome.Show` | トップページ表示（未ログイン時のみ） |

### コード設計

#### ディレクトリ構造

```
internal/
├── handler/
│   └── welcome/
│       ├── handler.go      # Handler構造体と依存性
│       └── show.go         # GET /
└── templates/
    ├── layouts/
    │   └── plain.templ     # 素のレイアウト（新規）
    ├── components/
    │   └── footer.templ    # フッターコンポーネント（新規）
    └── pages/
        └── welcome/
            └── show.templ  # トップページテンプレート
```

#### レイアウト設計

##### plain.templ（新規）

素のレイアウトを新規作成します。既存の `simple.templ` との違いは以下の通りです：

| レイアウト     | 用途                                         | 特徴                                 |
| -------------- | -------------------------------------------- | ------------------------------------ |
| `simple.templ` | エラーページ、最小限のページ                 | 中央寄せ、ヘッダーなし、フッターなし |
| `plain.templ`  | トップページ、利用規約、プライバシーポリシー | 通常の縦方向レイアウト、フッターあり |

**plain.templ の特徴**:

- ヘッダーなし（トップページは独自のヒーローセクションを持つため）
- フッターあり（利用規約、プライバシーポリシーへのリンク）
- サイドバーなし
- モバイル向け下部ナビゲーションバーなし
- コンテンツは縦方向に配置

```go
// plain.templ の引数
type PlainLayoutData struct {
    Meta    viewmodel.PageMeta
    Flash   *session.FlashMessage
}

templ Plain(data PlainLayoutData, content templ.Component) {
    <!DOCTYPE html>
    <html lang={ templates.Locale(ctx) }>
        <head>
            @components.Head(data.Meta)
        </head>
        <body class="min-h-screen flex flex-col">
            @components.Flash(data.Flash)
            <main class="flex-1">
                @content
            </main>
            @components.Footer()
        </body>
    </html>
}
```

##### footer.templ（新規）

利用規約とプライバシーポリシーへのリンクのみを含む最小限のフッターを新規作成します。

```go
// フッターコンポーネント
templ Footer() {
    <footer class="mt-auto px-4 pt-12 pb-12 text-center flex items-center justify-center flex-wrap gap-4">
        <ul class="flex items-center justify-center gap-4">
            <li>
                <a class="link-muted text-sm" href="/terms">
                    { templates.T(ctx, "footer_terms") }
                </a>
            </li>
            <li>
                <a class="link-muted text-sm" href="/privacy">
                    { templates.T(ctx, "footer_privacy") }
                </a>
            </li>
        </ul>
        <div class="text-sm text-muted-foreground">
            &copy; 2025 Wikino
        </div>
    </footer>
}
```

#### 主要な構造体

**Handler（welcome/handler.go）**

```go
type Handler struct {
    cfg            *config.Config
    sessionManager *session.Manager
}

func NewHandler(cfg *config.Config, sessionManager *session.Manager) *Handler {
    return &Handler{
        cfg:            cfg,
        sessionManager: sessionManager,
    }
}
```

**Show（welcome/show.go）**

```go
// Show GET / - トップページ
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // ログイン済みの場合は /home にリダイレクト
    if user := authMiddleware.GetUserFromContext(ctx); user != nil {
        http.Redirect(w, r, "/home", http.StatusSeeOther)
        return
    }

    // ページメタデータを作成
    meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
    meta.SetTitle(ctx, "welcome_title")
    meta.SetDescription(ctx, "welcome_description")

    // フラッシュメッセージを取得
    flash := h.sessionManager.GetFlash(r)

    // テンプレートをレンダリング
    layoutData := layouts.PlainLayoutData{
        Meta:  meta,
        Flash: flash,
    }
    pageData := welcome.ShowPageData{
        // 必要なデータを渡す
    }
    layouts.Plain(layoutData, pages.Show(pageData)).Render(ctx, w)
}
```

### ページ構成

トップページは以下のセクションで構成されます：

1. **ヒーローセクション**
   - サービスのタイトル、説明文
   - 「無料で始められる」の明示
   - サインアップ・サインインボタン
   - 実際の画面スクリーンショット
2. **機能紹介セクション** - 4 つの機能を画像付きで紹介（ベネフィット中心の表現）
   - **Markdown でシンプルに書ける** - プレーンテキストで書ける手軽さ
   - **ページ同士を簡単につなげる** - リンク記法（`[[ページ名]]`）で知識のネットワークを構築
   - **つながりが見える** - バックリンクで関連ページを発見、知識の整理
   - **スペースを公開して共有できる** - 作成したスペースを公開して共有可能
     - 注釈：「非公開機能は現在無料で使えますが、今後提供する有料プラン限定の機能になる予定です」
3. **CTA セクション** - サインアップへの導線
4. **開発者・コミュニティセクション** - 運営者情報とコミュニティへの導線
   - オープンソースで開発しています（機能紹介から移動）
   - 開発者についてへのリンク
   - 各種ソーシャルメディアへのリンク（Bluesky、Discord、Mastodon、Mewst、mixi2、X）
5. **フッター** - 利用規約、プライバシーポリシーへのリンクのみ

### 画像の取り扱い

#### ヒーローセクションのスクリーンショット

実際の画面スクリーンショットを新規作成し、ヒーローセクションに配置します。

- `/assets/welcome/screenshot.png`（新規作成）

#### 機能紹介セクションの画像

機能紹介セクションで使用する画像は、Rails 版と同じパスを使用します：

- `/assets/welcome/image_1.png`
- `/assets/welcome/image_2.png`
- `/assets/welcome/image_3.png`
- `/assets/welcome/image_4.png`

Go 版では、これらの画像を `static/` ディレクトリから配信します。

### テスト戦略

- **ハンドラーテスト**: HTTP リクエスト・レスポンスの統合テスト
  - 未ログイン時にトップページが表示されること
  - ログイン済み時に `/home` にリダイレクトされること
  - 日本語・英語の両言語で正しく表示されること

## タスクリスト

### フェーズ 1: レイアウトとコンポーネントの実装

- [x] **1-1**: [Go] 素のレイアウト（plain.templ）の実装
  - `internal/templates/layouts/plain.templ`
  - `internal/templates/layouts/plain_templ.go`（自動生成）
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 30 行（実装 30 行 + テスト 0 行）

- [x] **1-2**: [Go] フッターコンポーネント（footer.templ）の実装
  - `internal/templates/components/footer.templ`
  - `internal/templates/components/footer_templ.go`（自動生成）
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 30 行（実装 30 行 + テスト 0 行）

### フェーズ 2: トップページの実装

- [x] **2-1**: [Go] トップページテンプレートの実装
  - `internal/templates/pages/welcome/show.templ`
  - `internal/templates/pages/welcome/show_templ.go`（自動生成）
  - Rails 版の `welcome/show_view.html.erb` を参考に実装
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 200 行（実装 200 行 + テスト 0 行）

- [ ] **2-2**: [Go] トップページハンドラーの実装
  - `internal/handler/welcome/handler.go`
  - `internal/handler/welcome/show.go`
  - `internal/handler/welcome/show_test.go`
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 1）
  - **想定行数**: 約 150 行（実装 50 行 + テスト 100 行）

### フェーズ 3: ルーティングと国際化

- [ ] **3-1**: [Go] ルーティング設定とリバースプロキシの更新
  - `cmd/server/main.go` にルーティング追加
  - `internal/middleware/reverse_proxy.go` のホワイトリスト更新
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 20 行（実装 20 行 + テスト 0 行）

- [ ] **3-2**: [Go] 国際化（I18n）の追加
  - トップページ関連の翻訳キーを追加
  - `internal/i18n/locales/ja.toml`
  - `internal/i18n/locales/en.toml`
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 80 行（実装 80 行 + テスト 0 行）

### 実装しない機能（スコープ外）

以下の機能は今回の実装では**実装しません**：

- **サイドバー**: トップページではサイドバーを表示しない
- **モバイル向け下部ナビゲーションバー**: トップページでは表示しない
- **ログイン済みユーザー向けのホームページ**: 別途 `/home` で実装予定

## 参考資料

- **Rails 版トップページ**: `/workspace/rails/app/views/welcome/show_view.html.erb`
- **Annict Go 版フッター**: `/annict/go/internal/templates/components/footer.templ`
- **Go CLAUDE.md**: `/workspace/go/CLAUDE.md`
