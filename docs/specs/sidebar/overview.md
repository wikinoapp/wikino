# サイドバー 仕様書

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

サイドバーは画面左側に配置されるナビゲーション領域で、ナビゲーションリンク・下書きページ一覧・参加中トピック一覧を表示する。Go版・Rails版の両方で basecoat-css のサイドバーコンポーネント（`<aside class="sidebar">`）を使用しており、HTML構造・表示内容・開閉動作が統一されている。

サイドバーの開閉状態はlocalStorage（`wikinoSidebarOpen`）に保存され、ページ遷移後も状態が維持される。デスクトップでは固定表示、モバイルではスライド+オーバーレイとして表示され、レスポンシブ動作は basecoat-css が自動で処理する。

**目的**:

- ユーザーがアプリケーション内を素早くナビゲーションできる
- 下書きページや参加中トピックに直接アクセスできる
- Go版とRails版の画面を行き来しても違和感がない統一的なUIを提供する

**背景**:

- Go版とRails版で同じ basecoat-css サイドバーコンポーネントを採用することで、HTML構造・開閉動作・レスポンシブ対応を統一している
- 開閉状態の保存には、サーバーサイドのロジックが不要でHTTPリクエストへのオーバーヘッドもないlocalStorageを採用している

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

### サイドバーの表示

- サイドバーは basecoat-css の `<aside class="sidebar" data-side="left">` コンポーネントで実装されている
- デスクトップ（xl以上）では固定表示、モバイルではスライド+オーバーレイとして表示される
- レスポンシブ動作（固定/スライド切り替え、オーバーレイ）は basecoat-css のJSが処理する

### サイドバーの開閉

- ユーザーはトップナビのトグルボタンでサイドバーを開閉できる
- トグルボタンは `basecoat:sidebar` カスタムイベントを dispatch してサイドバーの開閉を制御する
- 開閉状態はlocalStorage（キー: `wikinoSidebarOpen`）に保存され、ページ遷移後も維持される
- 初期状態はインラインスクリプトでlocalStorageから読み取り、`data-initial-open` / `aria-hidden` 属性を設定する（FOUC防止のため `<nav>` より前に配置）
- デフォルトはサイドバー閉（`data-initial-open="false"` / `aria-hidden="true"`）

### ナビゲーション（ログイン時）

- ホーム: ホームページへのリンク（アイコン: `house-regular` / `house-fill`）
- 検索: 検索ページへのリンク（アイコン: `magnifying-glass-regular` / `magnifying-glass-fill`）。スペース内にいる場合はスペースフィルター付き検索パスになる
- プロフィール: ユーザーのプロフィールページへのリンク（アイコン: `user-circle-regular` / `user-circle-fill`）
- 現在のページに対応するナビゲーション項目はアクティブアイコン（fill版）で表示される

### ナビゲーション（未ログイン時）

- ホーム: トップページへのリンク（アイコン: `house-regular` / `house-fill`）
- サインイン: サインインページへのリンク（アイコン: `sign-in-regular`）

### 下書きページ一覧

- ログイン時のみ表示される
- ナビゲーションの後、参加中トピック一覧の前に配置される
- 最大5件の下書きページを `modified_at DESC` の順で表示する
- 各項目にはトピックのアイコン・トピック名・ページタイトルが表示される
- ページタイトルが未設定の場合は「無題」というフォールバックテキストが表示される
- 各項目にはホバー時に表示される編集ボタン（`pencil-simple-line-regular`）が設置されている
- 5件を超える下書きがある場合は「すべてを表示」リンクが表示され、下書き一覧画面に遷移する
- 下書きがない場合は空状態メッセージが表示される
- Rails版ではTurbo Frameによる遅延読み込みで取得する

### 参加中トピック一覧

- ログイン時のみ表示される
- 下書きページ一覧の後に配置される
- 最大10件の参加中トピックを表示する
- 各項目にはトピックのアイコン・スペース名・トピック名が表示される
- 各項目にはホバー時に表示されるページ作成ボタン（`pencil-simple-line-regular`）が設置されている
- Rails版ではTurbo Frameによる遅延読み込みで取得する

### セクション間の区切り

- ナビゲーションと下書きページ一覧の間に `<hr>` を表示する
- 下書きページ一覧と参加中トピック一覧の間に `<hr>` を表示する
- 区切り線のスタイルは `border-gray-300 mx-2`

### フッター

- サイドバーにはフッターリンクを含めない
- フッターはレイアウトに配置され、利用規約（`/terms`）・プライバシーポリシー（`/privacy`）・著作権表示（© 2025-2026 Wikino）を含む

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

### HTML構造

Go版・Rails版で共通の basecoat-css サイドバー構造を使用する。

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
  <nav>
    <section class="scrollbar gap-4 py-4">
      <ul class="px-2">
        <!-- ナビゲーション項目 -->
      </ul>

      <hr class="border-gray-300 mx-2" />
      <!-- 下書きページ一覧 -->

      <hr class="border-gray-300 mx-2" />
      <!-- 参加中トピック一覧 -->
    </section>
  </nav>
</aside>
```

### 開閉状態の保存

インラインスクリプト（`<aside>` 直後、`<nav>` より前に配置）でlocalStorageから初期状態を読み取り、`data-initial-open` / `aria-hidden` 属性を設定する。ブラウザが `<script>` タグをパースする際にレンダリングをブロックするため、可視コンテンツがレンダリングされる前に属性が確定し、FOUCを防止できる。

状態の保存はバンドルJS内で `basecoat:sidebar` イベントをリスンし、`aria-hidden` の値をlocalStorageに書き込む。

```javascript
document.addEventListener("basecoat:sidebar", () => {
  const sidebar = document.querySelector(".sidebar");
  if (!sidebar) return;
  const isOpen = sidebar.getAttribute("aria-hidden") === "false";
  localStorage.setItem("wikinoSidebarOpen", String(isOpen));
});
```

### CSS変数

basecoat-css サイドバーが使用するCSS変数を定義する。

```css
:root {
  --sidebar: var(--color-brand-50);
  --sidebar-accent: var(--color-brand-100);
  --sidebar-width: 200px;
}
```

### スタイリング

| 要素                      | CSSクラス / スタイル                      |
| ------------------------- | ----------------------------------------- |
| セクション見出し          | `text-xs font-bold text-muted-foreground` |
| ナビゲーションアイコン    | `size-4`（16px）                          |
| トピック/下書きアイコン   | `size-[16px]`                             |
| ホバー時の背景            | `hover:bg-sidebar-accent`                 |
| ホバー時の編集/作成ボタン | `hover:bg-brand-300/40`                   |
| 編集/作成ボタンアイコン   | `pencil-simple-line-regular`              |
| 区切り線                  | `<hr class="border-gray-300 mx-2">`       |

### コンポーネント構成

#### Go版

```
go/internal/templates/components/
├── sidebar.templ              # Sidebarコンポーネント（basecoat-css <aside>）
│   ├── <nav> / <section>
│   │   ├── <ul> ナビゲーション項目
│   │   ├── <hr>
│   │   ├── SidebarDraftPages   # 下書きページ一覧
│   │   ├── <hr>
│   │   └── SidebarJoinedTopics # 参加中トピック一覧
├── sidebar_draft_pages.templ  # 下書きページ一覧コンポーネント
├── sidebar_joined_topics.templ # 参加中トピック一覧コンポーネント
└── footer.templ               # フッターコンポーネント
```

Go版のサイドバーデータは `SidebarData` 構造体で渡される。下書きページ一覧・参加中トピック一覧はインラインでレンダリングする。

#### Rails版

```
rails/app/components/
├── sidebar_component.rb       # SidebarComponent（basecoat-css <aside>）
├── sidebar_component.html.erb
├── sidebar/
│   ├── draft_pages_component.rb       # 下書きページ一覧（Turbo Frame遅延読み込み）
│   ├── draft_pages_component.html.erb
│   ├── joined_topics_component.rb     # 参加中トピック一覧（Turbo Frame遅延読み込み）
│   └── joined_topics_component.html.erb
└── layouts/
    └── column1_component.html.erb     # レイアウト（フッターを含む）
```

Rails版の下書きページ一覧・参加中トピック一覧はTurbo Frameによる遅延読み込みで取得する。

### データ取得（下書きページ一覧）

下書きページ一覧のデータは `modified_at DESC` でソートし、limit+1件取得する。超過分がある場合は「すべてを表示」リンクを表示する。

各下書きページにはトピック名・ページタイトル・トピックアイコンの情報が必要なため、関連テーブル（space, page, topic）を結合して取得する。

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### Cookieによるサイドバー開閉状態の保存

Go版の初期実装ではCookie（`wikino_sidebar_open`）を使用していたが、localStorageに移行した。

**不採用の理由**:

- **サーバーサイドのロジックが不要**: localStorageはクライアントのみで完結するため、サーバーサイドでCookieを読み取るヘルパーが不要になり実装がシンプルになる
- **HTTPリクエストへのオーバーヘッドがない**: Cookieは全リクエストに自動的に付与されるが、サイドバーの開閉状態はサーバーに送る必要がない
- **インラインスクリプトでFOUCを防止可能**: `<aside>` 直後にインラインスクリプトを配置することで、DOMContentLoadedを待たずに初期状態を設定できる

### Rails版の参加中トピック一覧をインライン表示に変更する

Go版では参加中トピック一覧をインラインで取得・表示しているが、Rails版ではTurbo Frameでの遅延読み込みを維持している。

**不採用の理由**:

- Rails版の全ページでサイドバーの初期表示データにトピック一覧を含めると、すべてのコントローラーでデータ取得が必要になり影響範囲が大きい
- Turbo Frameの遅延読み込みはページの初期表示速度を損なわない
- 見た目が統一されていれば、データ取得方式の違いはユーザーには分からない

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [basecoat-css サイドバーコンポーネント](https://basecoatui.com/components/sidebar/)
- Go版サイドバー実装: `/workspace/go/internal/templates/components/sidebar.templ`
- Go版下書きページ一覧: `/workspace/go/internal/templates/components/sidebar_draft_pages.templ`
- Go版参加中トピック一覧: `/workspace/go/internal/templates/components/sidebar_joined_topics.templ`
- Rails版サイドバー実装: `/workspace/rails/app/components/sidebar_component.html.erb`
