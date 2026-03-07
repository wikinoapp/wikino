# Rails版サイドバーをGo版と同じ仕様に統一する 作業計画書

## 仕様書

- 仕様書は現在存在しない。タスク完了後に `docs/specs/sidebar/overview.md` として作成予定

## 概要

Rails版とGo版の画面を行き来しても違和感がないよう、Rails版のサイドバーをGo版の仕様に合わせて更新する。

Go版のサイドバーは basecoat-css のサイドバーコンポーネント（`<aside class="sidebar">`）を採用し、下書きページ一覧・参加中トピック一覧・ナビゲーションを統合的に表示している。一方Rails版は下書きページ一覧がなく、サイドバーのHTML構造はStimulus + 独自Tailwindクラスで実装されており、basecoat-cssのサイドバーコンポーネントを使用していない。

この作業では、Rails版のサイドバーをGo版と同じbasecoat-cssサイドバーコンポーネントに置き換え、HTML構造・表示内容・開閉動作を統一する。また、Go版・Rails版の両方でサイドバーの開閉状態の保存をCookieからlocalStorageに移行する。

## 要件

### 機能要件

- Rails版のサイドバーをbasecoat-cssのサイドバーコンポーネント（`<aside class="sidebar">`）に置き換える
- Go版と同じ `data-initial-open` / `aria-hidden` 属性によるサイドバー開閉制御を実装する
- サイドバーの開閉状態をlocalStorage（`wikinoSidebarOpen`）に保存し、ページ遷移後も状態を維持する
- Go版のサイドバーもCookieからlocalStorageに移行する
- Rails版のサイドバーにGo版と同じ「下書きページ一覧」セクションを追加する
- 下書きページ一覧は最大5件表示し、それ以上ある場合は「すべてを表示」リンクを表示する
- 下書きページ一覧の各項目はトピック名・ページタイトル（無題の場合はフォールバック）・トピックの公開/非公開アイコンを表示する
- 下書きページ一覧の各項目にホバー時に表示される編集ボタンを設置する
- 参加中トピック一覧のHTML構造とスタイリングをGo版と合わせる
- ナビゲーションリンクのHTML構造とスタイリングをGo版と合わせる（basecoat-cssの `<ul>` / `<li>` / `<a>` セマンティクス）
- 未ログイン状態のナビゲーションをGo版と合わせる（ホーム・サインイン）
- セクション間の区切り線（`<hr>`）の配置をGo版と合わせる
- Go版にないフッターリンク（ヘルプ・ロードマップ・コミュニティ）はそのまま残す

### 非機能要件

- basecoat-cssのサイドバーJSを読み込み、開閉アニメーション・オーバーレイ・レスポンシブ動作をbasecoat-cssに委譲する
- 既存のStimulus `sidebar_controller.ts` は不要になるため削除する

## 実装ガイドラインの参照

### Rails版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - 全体的なコーディング規約
- [@rails/docs/architecture-guide.md](/workspace/rails/docs/architecture-guide.md) - アーキテクチャガイド（クラス設計と依存関係、サービスクラスのルール）
- [@rails/docs/testing-guide.md](/workspace/rails/docs/testing-guide.md) - テストガイド（RSpec のコーディング規約）
- [@rails/docs/security-guide.md](/workspace/rails/docs/security-guide.md) - セキュリティガイドライン

## 設計

### 現状の差分まとめ

| 項目                         | Go版                                           | Rails版（現状）                                               |
| ---------------------------- | ---------------------------------------------- | ------------------------------------------------------------- |
| サイドバーコンポーネント     | basecoat-css `<aside class="sidebar">`         | Stimulus + 独自Tailwindクラス                                 |
| 開閉制御                     | basecoat-css JS（`basecoat:sidebar` イベント） | Stimulus `sidebar_controller.ts`                              |
| 開閉状態の保存               | Cookie `wikino_sidebar_open`                   | なし（毎回リセット）                                          |
| 下書きページ一覧             | あり（最大5件 + 「すべて表示」）               | なし                                                          |
| 参加中トピック一覧           | インライン表示（最大10件）                     | Turbo Frameで遅延読み込み（最大10件）                         |
| ナビゲーション構造           | `<ul>` / `<li>` / `<a>`（basecoat-css）        | `<div>` + `link_to`（独自）                                   |
| ナビゲーションアイコンサイズ | `size-4`（16px）                               | `22px`                                                        |
| セクション見出しスタイル     | `text-xs font-bold text-muted-foreground`      | `text-sm font-bold`                                           |
| トピックアイコンサイズ       | `size-[16px]`                                  | `22px`                                                        |
| ホバー時の編集/作成ボタン    | `hover:bg-brand-300/40`                        | `hover:bg-brand-300/70`                                       |
| 編集/作成ボタンアイコン      | `pencil-simple-line-regular`                   | `pencil-simple-line`                                          |
| セクション間の区切り線       | `<hr class="border-gray-300 mx-2">`            | `<hr class="border-gray-300">` を `<div class="px-2">` で囲む |
| セクションの順序             | ナビ → 下書き → トピック                       | ナビ → トピック → フッター                                    |
| フッターリンク               | なし                                           | あり（ヘルプ・ロードマップ・コミュニティ）                    |

### 変更方針

1. **basecoat-cssサイドバーコンポーネントへの移行**: Rails版の `SidebarComponent` を `<aside class="sidebar" data-side="left" data-initial-open aria-hidden>` 構造に書き換える。既存の固定/スライドの2重実装とStimulus `sidebar_controller.ts` を削除し、basecoat-cssのサイドバーJSに置き換える
2. **サイドバー開閉状態のlocalStorage保存**: Go版・Rails版の両方で `basecoat:sidebar` イベントをリスンし、`wikinoSidebarOpen` をlocalStorageに保存する。初期状態はインラインスクリプトでlocalStorageから読み取り `data-initial-open` / `aria-hidden` を設定する（サーバーサイドのロジックは不要）
3. **下書きページ一覧の追加**: `Sidebar::DraftPagesComponent` を新規作成する。参加中トピック一覧のTurbo Frameと同様の遅延読み込み方式を採用する
4. **スタイリングの統一**: CSSクラスをGo版のbasecoat-css変数ベースに合わせる。Rails版のCSSに `--sidebar` / `--sidebar-accent` 等のCSS変数を追加する
5. **セクション順序の変更**: ナビ → 下書き → トピック → フッターの順に変更
6. **参加中トピック一覧はTurbo Frame方式を維持**: 遅延読み込みは維持し、見た目のみGo版に合わせる

### basecoat-cssサイドバーへの移行

**現状（Rails版）**:

```erb
<%# 固定サイドバー (xl以上) %>
<aside class="z-sidebar fixed top-0 left-0 hidden h-screen xl:block xl:max-w-[230px]">
  <%= render Sidebar::ContentComponent.new(variant: :fixed) %>
</aside>

<%# スライドサイドバー (xl未満) %>
<aside data-sidebar-target="panel" class="z-sidebar fixed ... -translate-x-full ...">
  <%= render Sidebar::ContentComponent.new(variant: :slide) %>
</aside>

<%# オーバーレイ %>
<div data-sidebar-target="overlay" data-action="click->sidebar#close" class="..."></div>
```

**変更後（basecoat-css準拠）**:

```erb
<aside class="sidebar" data-side="left">
  <nav>
    <section class="scrollbar gap-4 py-4">
      <%# ナビゲーション、下書き、トピック、フッター %>
    </section>
  </nav>
</aside>
```

basecoat-cssがレスポンシブ対応（デスクトップ: 固定表示、モバイル: スライド + オーバーレイ）を自動で処理するため、2重実装とStimulus制御が不要になる。

`data-initial-open` と `aria-hidden` の初期値はインラインスクリプトで設定するため、サーバーサイドでの属性設定は不要（詳細は「サイドバー開閉状態のlocalStorage保存」を参照）。

### サイドバー開閉状態のlocalStorage保存

サイドバーの開閉状態をlocalStorageに保存し、ページ遷移後も状態を維持する。Go版・Rails版の両方で同じ方式を採用する。

**インラインスクリプト（`<aside>` 開始タグ直後に配置）**: localStorageから初期状態を読み取り、FOUC（ちらつき）を防止する。`<aside>` にはデフォルトで `data-initial-open="false" aria-hidden="true"` を設定し、localStorageに `"true"` がある場合のみスクリプトで属性を上書きする。スクリプトは `<nav>` 等の可視コンテンツより前に配置することで、ブラウザがパースをブロックし、レンダリング前に属性が確定する。

```html
<aside class="sidebar" data-side="left" id="sidebar" data-initial-open="false" aria-hidden="true">
  <script>
    (function () {
      var s = document.getElementById("sidebar");
      if (s && localStorage.getItem("wikinoSidebarOpen") === "true") {
        s.setAttribute("data-initial-open", "true");
        s.setAttribute("aria-hidden", "false");
      }
    })();
  </script>
  <nav>...</nav>
</aside>
```

**状態保存（バンドルJS）**: `basecoat:sidebar` イベントをリスンし、localStorageに状態を保存する

```javascript
document.addEventListener("basecoat:sidebar", () => {
  const sidebar = document.querySelector(".sidebar");
  if (!sidebar) return;
  const isOpen = sidebar.getAttribute("aria-hidden") === "false";
  localStorage.setItem("wikinoSidebarOpen", String(isOpen));
});
```

**Go版の変更点**:

- `go/web/main.js`: Cookie保存ロジックをlocalStorage保存ロジックに置き換える
- `go/internal/templates/layouts/sidebar.go`: 削除する（サーバーサイドでのCookie読み取りが不要になるため）
- `go/internal/templates/components/sidebar.templ`: `DefaultClosed` フィールドの使い方を変更する（インラインスクリプトに委譲）
- レイアウトテンプレートにインラインスクリプトを追加する

### basecoat-css サイドバーJSの読み込み

Rails版では現在ドロップダウンメニューJSのみをCDN経由で読み込んでいる。サイドバーJSも同様に追加する。

```typescript
// application.ts に追加
script.src = "https://cdn.jsdelivr.net/npm/basecoat-css@0.2.8/dist/js/sidebar.min.js";
```

### サイドバートグルボタン

Go版ではトップナビに以下のボタンがある：

```html
<button onclick="document.dispatchEvent(new CustomEvent('basecoat:sidebar'))"></button>
```

Rails版でも同じ方式でサイドバーの開閉トリガーを実装する。既存のStimulus `data-action="sidebar#toggle"` を `onclick` イベントに置き換える。

### CSS変数の追加

Rails版の `application.css` にGo版と同じサイドバー用CSS変数を追加する：

```css
:root {
  --sidebar: var(--color-brand-50);
  --sidebar-accent: var(--color-brand-100);
  --sidebar-width: 200px;
}
```

### データ取得（下書きページ一覧）

- `DraftPageRepository` に `find_for_sidebar(user_record:, limit:)` メソッドを追加
- `DraftPageRecord` を `space_record`, `page_record`, `topic_record` と共にクエリ
- `modified_at DESC` でソート
- limit+1 件取得し、超過分があれば「すべて表示」リンクを出す

### コンポーネント構成（変更後）

```
SidebarComponent (basecoat-css <aside class="sidebar">)
└── <nav> / <section>
    ├── <ul> ナビゲーション項目 (basecoat-css セマンティクス)
    ├── <hr>
    ├── Sidebar::DraftPagesComponent (新規: 下書きページ一覧)
    │   └── Turbo Frameで遅延読み込み
    ├── <hr>
    ├── Sidebar::JoinedTopicsComponent (既存: 参加中トピック一覧)
    │   └── Turbo Frameで遅延読み込み
    └── フッターリンク
```

`Sidebar::ContentComponent` は不要になる（basecoat-cssが構造を提供するため、SidebarComponentに統合）。
`Sidebar::ItemLinkComponent` も不要になる（basecoat-cssの `<ul>` / `<li>` / `<a>` で直接記述）。

### 新規コントローラー

- `DraftPages::SidebarController` - サイドバー用の下書きページ一覧を返すコントローラー

## 採用しなかった方針

### 参加中トピック一覧をインライン表示に変更する

Go版では参加中トピック一覧をサイドバーデータと一緒にインラインで取得・表示しているが、Rails版ではTurbo Frameでの遅延読み込みを維持する。理由：

- Rails版の全ページでサイドバーの初期表示データにトピック一覧を含めると、全てのコントローラーでデータ取得が必要になり影響範囲が大きい
- Turbo Frameの遅延読み込みはページの初期表示速度を損なわない
- 見た目が統一されていれば、データ取得方式の違いはユーザーには分からない

### Cookieによるサイドバー開閉状態の保存

Go版の既存実装ではCookie（`wikino_sidebar_open`）を使用していたが、localStorageに移行する。理由：

- **サーバーサイドのロジックが不要**: localStorageはクライアントのみで完結するため、サーバーサイドでCookieを読み取るヘルパーメソッド（Go版の `SidebarDefaultClosed` 関数）が不要になり、実装がシンプルになる
- **HTTPリクエストへのオーバーヘッドがない**: Cookieは全リクエストに自動的に付与されるが、サイドバーの開閉状態はサーバーに送る必要がない
- **インラインスクリプトでFOUCを防止可能**: `<aside>` 直後にインラインスクリプトを配置することで、DOMContentLoadedを待たずに初期状態を設定でき、ちらつきを防止できる

## タスクリスト

### フェーズ 1: Go版のlocalStorage移行 + basecoat-cssサイドバーへの移行

- [x] **1-1**: [Go] サイドバー開閉状態の保存をCookieからlocalStorageに移行する
  - `go/web/main.js`: Cookie保存ロジック（`initSidebarCookiePersistence`）をlocalStorage保存ロジックに置き換える
  - `go/internal/templates/layouts/sidebar.go`: 削除する（サーバーサイドでのCookie読み取りが不要になるため）
  - `go/internal/templates/components/sidebar.templ`: `SidebarData.DefaultClosed` フィールドを削除し、初期状態をインラインスクリプトに委譲する
  - レイアウトテンプレートにインラインスクリプトを追加する（`<aside>` 直後に配置）
  - サイドバーデータの組み立て箇所から `DefaultClosed` の設定を削除する
  - **想定ファイル数**: 約 6 ファイル（実装 6 + テスト 0）
  - **想定行数**: 約 60 行（実装 60 行 + テスト 0 行）

- [x] **1-2**: [Rails] basecoat-cssサイドバーのCSS変数・JSを追加し、サイドバーのHTML構造を移行する
  - `application.css` に `--sidebar`, `--sidebar-accent` 等のCSS変数を追加
  - `application.ts` にbasecoat-cssサイドバーJSの読み込みを追加
  - `SidebarComponent` を `<aside class="sidebar" data-side="left">` 構造に書き換え
  - `Sidebar::ContentComponent` を削除し、`SidebarComponent` に統合
  - `Sidebar::ItemLinkComponent` を削除し、basecoat-cssの `<ul>` / `<li>` / `<a>` で直接記述
  - `Sidebar::JoinedTopicsComponent` の `variant` 引数を削除（2重実装が不要になるため）
  - ナビゲーション構造・アイコンサイズ・スタイリングをGo版に合わせる
  - **想定ファイル数**: 約 8 ファイル（実装 8 + テスト 0）
  - **想定行数**: 約 200 行（実装 200 行 + テスト 0 行）

- [x] **1-3**: [Rails] サイドバー開閉状態のlocalStorage保存とStimulus削除
  - レイアウトに `<aside>` 直後のインラインスクリプトを追加（localStorageから初期状態を読み取り `data-initial-open` / `aria-hidden` を設定）
  - `basecoat:sidebar` イベントリスナーを追加し、localStorageに状態を保存
  - 既存のサイドバートグルボタンを `onclick="document.dispatchEvent(new CustomEvent('basecoat:sidebar'))"` に置き換え
  - Stimulus `sidebar_controller.ts` を削除
  - **想定ファイル数**: 約 4 ファイル（実装 4 + テスト 0）
  - **想定行数**: 約 50 行（実装 50 行 + テスト 0 行）

### フェーズ 2: 下書きページ一覧の追加

- [ ] **2-1**: [Rails] DraftPageRepository にサイドバー用クエリメソッドを追加
  - `find_for_sidebar(user_record:, limit:)` メソッドを追加
  - `DraftPageRecord` を `space_record`, `page_record`, `topic_record` で preload
  - `modified_at DESC` でソート
  - テストを作成
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 80 行（実装 30 行 + テスト 50 行）

- [ ] **2-2**: [Rails] サイドバー用下書きページ一覧のコントローラー・ビュー・コンポーネントを作成
  - `DraftPages::SidebarController` を新規作成
  - `DraftPages::SidebarView` を新規作成
  - `Sidebar::DraftPagesComponent` を新規作成（Turbo Frameで遅延読み込み）
  - ルーティングを追加
  - Go版と同じHTML構造・スタイリングで下書きページ一覧を表示
  - 各項目にトピックアイコン・トピック名・ページタイトル・ホバー時の編集ボタンを表示
  - 下書きがない場合の空状態メッセージを表示
  - 5件を超える場合「すべてを表示」リンクを表示
  - I18n翻訳キーを追加（ja, en）
  - `SidebarComponent` に下書きセクションを追加（ナビの後、トピックの前）
  - **想定ファイル数**: 約 10 ファイル（実装 8 + テスト 2）
  - **想定行数**: 約 250 行（実装 180 行 + テスト 70 行）

### フェーズ 3: 既存セクションのスタイリング統一

- [ ] **3-1**: [Rails] 参加中トピック一覧のスタイリングをGo版に合わせる
  - `JoinedTopics::IndexController` と `IndexView` を更新（variant 引数の削除対応）
  - `JoinedTopics::IndexView` のHTML構造をGo版に合わせる
  - アイコンサイズを `22px` → `16px` に変更
  - セクション見出しのスタイルをGo版に合わせる
  - ホバー時のスタイルを統一
  - **想定ファイル数**: 約 3 ファイル（実装 3 + テスト 0）
  - **想定行数**: 約 40 行（実装 40 行 + テスト 0 行）

### フェーズ 4: 仕様書への反映

- [ ] **4-1**: 仕様書の作成・更新
  - `docs/specs/sidebar/overview.md` にサイドバーの仕様書を作成する
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

- **フッターリンクの削除**: Go版にはないが、Rails版固有の情報として残す

## 参考資料

- [basecoat-css サイドバーコンポーネント](https://basecoatui.com/components/sidebar/)
- Go版サイドバー実装: `/workspace/go/internal/templates/components/sidebar.templ`
- Go版下書きページ一覧: `/workspace/go/internal/templates/components/sidebar_draft_pages.templ`
- Go版参加中トピック一覧: `/workspace/go/internal/templates/components/sidebar_joined_topics.templ`
- Go版サイドバー状態保存JS: `/workspace/go/web/main.js`
