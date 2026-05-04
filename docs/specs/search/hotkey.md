# 検索 グローバルホットキー 仕様書

## 概要

グローバルホットキーは、Go 版で配信されるサイドバー付きの主要なページ (`default.templ` 利用画面) から `s` キーまたは `/` キーひとつで検索ページに遷移できる機能である。スペース内のページにいる場合は、自動的に `q=space:{identifier}` クエリが付与された検索 URL に遷移し、現在のスペース内に絞り込んだ検索フォームが開く。

Rails 版では Stimulus + `@github/hotkey` で同等機能を提供しており、Go 版では素の TypeScript で実装している。Go 版は Stimulus を導入しておらず、`web/main.js` から `DOMContentLoaded` で初期化する素のフロントエンド構成を採っているため、依存ライブラリは追加せず自前実装としている。

判定ロジック (どの要素を「入力中」とみなすか) は Rails 版と同一にしており、両者の挙動は一致する。

**目的**:

- キーボード操作で検索ページに高速にアクセスできるようにし、ページ間移動の摩擦を減らす
- スペース内にいる場合は自動的にスコープ付き検索に遷移させ、ユーザーがクエリを毎回打ち込まなくて済むようにする
- Rails 版から Go 版に移行した画面でも、同じキーボード操作性を維持する

**背景**:

- Rails 版では `s` / `/` キーで検索ページに遷移できるホットキーが提供されていた。Go 版で再実装した画面 (リバースプロキシで Go 側に振られる画面) では、Stimulus コントローラが読み込まれないためホットキーが効かない状態になっていた
- 機能パリティを維持するため、Go 版にも同等のホットキーを実装した

## 仕様

### 対象画面

- `default.templ` レイアウトを使う画面でホットキーが有効になる (サイドバーの有無に関わらず動作する)
- `simple.templ` レイアウトを使う画面 (sign_in / sign_up / sign_in_two_factor / sign_in_two_factor_recovery / password_reset / email_confirmation / account / password 等の認証フロー画面) はホットキー対象外。Rails 版でも同様にこれらの画面ではホットキーが効かないため、挙動を揃えている

### キー操作

| キー操作                              | 挙動                                                                               |
| ------------------------------------- | ---------------------------------------------------------------------------------- |
| `s` または `/` (入力欄外でフォーカス) | スペース外なら `/search`、スペース内なら `/search?q=space:{identifier}` に遷移する |
| `s` または `/` (入力欄内でフォーカス) | 何もしない (通常の文字入力として扱われる)                                          |
| Ctrl / Meta / Alt + `s` などの併用    | 何もしない (ブラウザ既定の動作を妨げない)                                          |

- `Shift` キーは許可するが、`event.key` で大文字小文字や記号変換が反映されるため、`Shift + s` は `S`、`Shift + /` は `?` となり、いずれも `event.key === "s"` / `event.key === "/"` の判定で自然に弾かれる

### 入力欄判定

ユーザーが以下のいずれかの要素にフォーカスを置いている場合、システムはホットキーを発動させない (タイピングを邪魔しない)。

- `input` / `textarea` / `select` 要素
- `contenteditable="true"` を持つ要素
- `.cm-content` クラスを持つ要素 (CodeMirror エディタの編集領域)

IME 入力中の `isComposing` 判定は行わない。Rails 版と同様、入力欄判定でカバーされるためである。

### スペースフィルター付与

- ユーザーがスペース内のページ (例: `/s/{space}/topics/...`、`/s/{space}/pages/...`) にいる場合、システムは `/search?q=space:{identifier}` に遷移する
- スペース外のページ (ホーム、プロフィール、サイドバー無しの画面など) では、素の `/search` に遷移する
- 検索ページ自体に遷移したあとは、検索フォームにフォーカスがあるため `s` / `/` キーは通常の文字入力として扱われる (ホットキーとして再発動しない)

## 設計

### 全体方針

- Stimulus / `@github/hotkey` などの外部依存は追加せず、**素の TypeScript** で実装する
- 検索パスはサーバーサイド (`templates.SearchPathFor`) で組み立てて `<meta>` タグに埋め込み、フロントエンドはその値に遷移するだけにする (クエリ組み立てを JS 側で行わない)
- Rails 版とロジックを共有はしないが、入力欄判定のルールを揃えることで挙動の一貫性を保つ

### コード構成

```
go/web/
├── main.js                  # エントリポイント (initializeGlobalHotkey を呼ぶ)
└── global-hotkey.ts         # グローバルホットキー処理
```

`global-hotkey.ts` は `initializeGlobalHotkey(): void` をエクスポートし、`web/main.js` から `DOMContentLoaded` で 1 回だけ呼び出す。

### TypeScript 側の処理

`initializeGlobalHotkey` は `document` に `keydown` リスナを 1 つ登録し、以下の順で判定する:

1. 修飾キー (`event.ctrlKey` / `event.metaKey` / `event.altKey`) のいずれかが押されている場合は何もしない
2. `event.key` が `s` でも `/` でもない場合は何もしない
3. `document.activeElement` が「入力中」要素 (input / textarea / select / `contenteditable="true"` / `.cm-content`) であれば何もしない
4. `<meta name="wikino-search-path">` の `content` 属性から検索パスを取得する。値が空なら何もしない
5. `event.preventDefault()` を呼んでから `window.location.href = searchPath` で遷移する

### サーバーサイド側の処理

`internal/templates/layouts/default.templ` の `<head>` に以下の `<meta>` タグを 1 つ出力する:

```templ
<meta name="wikino-search-path" content={ string(templates.SearchPathFor(data.Meta.CurrentSpaceIdentifier)) }/>
```

`templates.SearchPathFor` は `internal/templates/path.go` で定義されており、引数の `spaceIdentifier` が空文字なら `/search`、非空なら `/search?q=space:{identifier}` を返す。スペース識別子は英数字とアンダースコアのみを想定しており、URL エンコードは行わない。

### `viewmodel.PageMeta.CurrentSpaceIdentifier`

`viewmodel.PageMeta` に `CurrentSpaceIdentifier string` フィールドを持たせ、ホットキー用の検索パスを組み立てるための識別子としてレイアウトに渡す。

- 「`<meta>` タグ用の情報は `PageMeta` に集約する」という方針に基づき、Title / Description / OG 系メタ情報と同じ場所で管理する
- スペース内画面のハンドラーで `meta := viewmodel.DefaultPageMeta(...)` を構築した直後に `meta.CurrentSpaceIdentifier = string(spaceIdentifier)` を設定する。`CurrentSpaceIdentifier` 専用のセッター (`Title` における `SetTitle` のようなもの) は現状未提供で、生のフィールド代入で運用している
- スペース外画面ではゼロ値の空文字のままにする
- サイドバーを表示しない画面 (`HideSidebar = true`) でもホットキーを動かす必要があるため、`SidebarData.SpaceIdentifier` ではなく `PageMeta` で受け渡す

## 採用しなかった方針

### `@github/hotkey` パッケージを Go 版にも導入する

Rails 版と同じ `@github/hotkey` パッケージを `go/package.json` に追加して使う案。Rails 版とロジックを共有でき、`data-hotkey` 属性ベースの宣言的な書き味が得られる。

**不採用の理由**:

- Go 版は Stimulus を使わず、素の TypeScript + DOMContentLoaded 初期化のシンプルな構成を採っている。`@github/hotkey` を導入してもメリットは小さく、依存と bundle サイズが増えるだけ
- 実装ロジックは「キー判定 + 入力欄判定 + 遷移」のごく短いコードで、自前実装の保守コストはほぼ無視できる
- YAGNI の観点からも、外部依存の追加は必要になってからで良い

### Stimulus を Go 版にも導入する

Rails 版と同じ Stimulus を Go 版にも導入し、コントローラを共有する案。

**不採用の理由**:

- Go 版は意図的に Stimulus を使わず素の TypeScript で構成している。本機能 1 つのためにフレームワークを導入するのは過剰
- 仮に将来的に Stimulus 導入を検討するとしても、それは別タスクで設計判断するべき範囲

### `<meta>` タグではなくグローバル変数で受け渡す

検索パスを `window.WIKINO = { ... }` のようなグローバル変数経由で渡す案。

**不採用の理由**:

- Rails 版が `<meta>` タグ + Stimulus value で受け渡しており、Wikino 全体の慣習に近い形を維持できる
- `<meta>` タグはサーバサイド (templ) で型を意識して埋め込めるため、グローバル変数より見通しが良い
- グローバル変数は他の JS との名前衝突や、状態管理の複雑化リスクがある

### `SidebarData.SpaceIdentifier` を直接参照する

レイアウトに新しいフィールドを足さず、`data.Sidebar.SpaceIdentifier` をホットキー用の `<meta>` タグに埋め込む案。

**不採用の理由**:

- `HideSidebar = true` の画面ではサイドバーがレンダリングされないが、ホットキーは引き続き動く必要がある
- 「サイドバーに表示するためのデータ」と「ホットキーで使うためのデータ」は意味的に別関心事であり、`<meta>` タグ用の値は `PageMeta` に集約するほうが意図が明確になる

### `DefaultLayoutData` 直下に `CurrentSpaceIdentifier` フィールドを追加する

`DefaultLayoutData` に直接 `CurrentSpaceIdentifier string` フィールドを足し、`data.CurrentSpaceIdentifier` として参照する案。

**不採用の理由**:

- `<meta>` タグとして出力する情報なので、`viewmodel.PageMeta` (Title・Description・OG 系などの head メタ情報の集約場所) に持たせるほうが責務として一貫する
- レイアウト直下にフィールドを増やすと、ホットキー以外でも将来的に「`<meta>` タグ用の値」が出てきた際に同じ層に並ぶことになり、`PageMeta` との分担が曖昧になる
- 各ハンドラで `meta := viewmodel.DefaultPageMeta(...)` → `meta.SetTitle(...)` → `meta.CurrentSpaceIdentifier = ...` という流れで Meta を組み立てるパターンに統一でき、書き味も自然

### 2 種類の `<meta>` タグを出力し JS 側でクエリを組み立てる

`<meta name="wikino-search-path">` (素の `/search`) と `<meta name="wikino-current-space-identifier">` の 2 つの `<meta>` タグを出力し、JS 側で空でない場合に `?q=space:{identifier}` を組み立てる案 (Rails 版と同じ構造)。

**不採用の理由**:

- Go 版にはサーバーサイドで検索パスを生成するヘルパー (`templates.SearchPathFor`) が既に存在しており、そこにスペース内 / 外の分岐ロジックが集約されている。JS 側で同じ分岐を再実装するのは重複になる
- `<meta>` タグが 1 つ減ることで `<head>` の出力もシンプルになる
- JS 側では「meta タグの値にそのまま遷移する」だけになり、テンプレートとフロントエンドの責務が明確に分離される
- スペース ID のエンコードや将来的なクエリ追加も、サーバーサイドのヘルパーに集約しておくほうが変更の影響範囲を局所化できる

### IME 入力中 (`isComposing`) の独自判定を追加する

`event.isComposing` を明示的にチェックして IME 入力中はホットキーを発動させない案。

**不採用の理由**:

- IME を起動している時点で入力欄 (input / textarea / contenteditable) にフォーカスが当たっており、入力欄判定で既にホットキー発動が抑制されている
- Rails 版コントローラにも `isComposing` の判定はなく、入力欄判定でカバーされている
- 重複した判定を追加するメリットがなく、Rails 版とのロジック差異も生じてしまう

## 参考資料

- [@/rails/app/javascript/controllers/global-hotkey-controller.ts](/workspace/rails/app/javascript/controllers/global-hotkey-controller.ts) — Rails 版の Stimulus コントローラ (実装の参照元)
- [@/go/web/global-hotkey.ts](/workspace/go/web/global-hotkey.ts) — Go 版のホットキー実装
- [@/go/web/main.js](/workspace/go/web/main.js) — Go 版フロントエンドのエントリポイント
- [@/go/internal/templates/layouts/default.templ](/workspace/go/internal/templates/layouts/default.templ) — Go 版のデフォルトレイアウト
- [@/go/internal/templates/path.go](/workspace/go/internal/templates/path.go) — 検索パスのヘルパー関数 (`SearchPath` / `SearchPathWithSpaceFilter` / `SearchPathFor`)
- [@/go/internal/viewmodel/page_meta.go](/workspace/go/internal/viewmodel/page_meta.go) — `PageMeta` の定義 (`CurrentSpaceIdentifier` フィールドを含む)
- [@github/hotkey](https://github.com/github/hotkey) — Rails 版で使用しているホットキーライブラリ (Go 版では使用しないが挙動の参考)
