# Datastar から htmx 4 への移行 作業計画書

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

- 該当なし（インフラ・ライブラリの変更のため、機能仕様への影響なし）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

Go 版 Wikino のハイパーメディアフレームワークを Datastar v1.0.0-RC.7 から htmx 4 に移行する。

移行の動機:

- **コミュニティの規模**: htmx のほうがコミュニティが大きく、情報や事例が豊富
- **バックエンドとの疎結合**: Datastar は Go SDK（`datastar-go`）に依存しており、バックエンドとの結合が密。htmx は「単に HTML を返す」だけで済む
- **SSE の冗長性**: 現在の使用箇所（ページネーション等）は単純な HTML フラグメントの取得で十分であり、SSE は過剰
- **依存関係のシンプルさ**: htmx 単体で現在の全ユースケースをカバーできる

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

- 既存の全 Datastar ベースの UI インタラクションが htmx 4 で同等に動作する
  - フォーム二重送信防止（7 箇所）
  - ページネーションの「もっと読み込む」ボタン（3 箇所）
  - 下書き自動保存後のフラグメント更新（1 箇所）
- Datastar の依存を完全に除去する（Go SDK、ベンダー JS ファイル、go.mod エントリ）

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

- ユーザーから見た挙動は変更前と同一であること（リグレッションなし）
- CSRF 保護が引き続き機能すること

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

### Go 版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン（**ファイル名は標準の 9 種類のみ**）
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
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

### 移行対象ファイルの一覧

#### Go ハンドラー（Datastar SDK 使用: 4 ファイル）

| ファイル                                      | 用途                               | 移行方針                           |
| --------------------------------------------- | ---------------------------------- | ---------------------------------- |
| `internal/handler/draft_page/show.go`         | 下書き自動保存後のフラグメント更新 | OOB スワップで 3 要素同時更新      |
| `internal/handler/page_link_list/show.go`     | リンク一覧ページネーション         | 通常の HTML レスポンス（SSE 不要） |
| `internal/handler/page_backlinks/show.go`     | バックリンクページネーション       | 通常の HTML レスポンス（SSE 不要） |
| `internal/handler/page_backlink_list/show.go` | リンク先ごとのバックリンク         | 通常の HTML レスポンス（SSE 不要） |

#### templ テンプレート（Datastar 属性使用: 11 ファイル）

| ファイル                                                 | 使用パターン               | 移行方針             |
| -------------------------------------------------------- | -------------------------- | -------------------- |
| `internal/templates/components/load_more_button.templ`   | `data-on:click` + `@get`   | `hx-get` + `hx-swap` |
| `internal/templates/components/backlink_list.templ`      | `@get` + `filterSignals`   | `hx-get` + `hx-swap` |
| `internal/templates/components/link_list.templ`          | `@get` + `filterSignals`   | `hx-get` + `hx-swap` |
| `internal/templates/components/page_backlink_list.templ` | `@get` + `filterSignals`   | `hx-get` + `hx-swap` |
| `internal/templates/pages/account/new.templ`             | `$isSubmitting` シグナル   | `hx-disable`         |
| `internal/templates/pages/email_confirmation/edit.templ` | `$isSubmitting` シグナル   | `hx-disable`         |
| `internal/templates/pages/page/edit.templ`               | `$isSubmitting` + 自動保存 | `hx-disable` + OOB   |
| `internal/templates/pages/password/edit.templ`           | `$isSubmitting` シグナル   | `hx-disable`         |
| `internal/templates/pages/password/reset.templ`          | `$isSubmitting` シグナル   | `hx-disable`         |
| `internal/templates/pages/sign_in/new.templ`             | `$isSubmitting` シグナル   | `hx-disable`         |
| `internal/templates/pages/sign_up/new.templ`             | `$isSubmitting` シグナル   | `hx-disable`         |

#### その他（2 ファイル）

| ファイル                                   | 用途                 | 移行方針           |
| ------------------------------------------ | -------------------- | ------------------ |
| `internal/templates/components/head.templ` | Datastar JS 読み込み | htmx JS に差し替え |
| `go.mod`                                   | `datastar-go` 依存   | 依存を削除         |

### パターン別の移行設計

#### パターン 1: フォーム二重送信防止（7 ファイル）

**現在（Datastar）**:

```templ
<form data-on:submit__passive="$isSubmitting = true">
  <button data-attr:disabled="$isSubmitting == true">送信</button>
</form>
```

**移行後（htmx 4）**:

```templ
<form hx-on:submit="disableSubmitButtons(this)" method="POST" action="/path">
  <button type="submit">送信</button>
</form>
```

`disableSubmitButtons` は `web/main.js` に定義したグローバル関数で、フォーム内の全送信ボタンを disabled にする。htmx 4 の `hx-on:submit` は通常の DOM イベントリスナーとして動作するため、htmx 経由でないフォーム送信でも機能する。通常の HTML フォーム送信（`method="POST" action="/path"`）を維持したまま、テンプレートを見るだけで送信時の挙動がわかる。

#### パターン 2: ページネーション「もっと読み込む」（3 ファイル）

**現在（Datastar）**:

```templ
<button data-on:click="@get('/path?page=2', {filterSignals: {include: /(?!)/}})">
  もっと読み込む
</button>
```

Go ハンドラー側で SSE + `PatchElementTempl` を使用。

**移行後（htmx 4）**:

```templ
<button hx-get="/path?page=2" hx-target="#list-container" hx-swap="beforeend">
  もっと読み込む
</button>
```

Go ハンドラー側は通常の HTML レスポンスを返すだけ（SSE 不要）。`datastar-go` SDK への依存がなくなる。

現在のハンドラーでは、Before モードで新しいカードを挿入し、Inner モードでページネーション要素を更新するという 2 段階の操作を行っている。htmx では OOB スワップを使って同等の処理を実現する。

#### パターン 3: 下書き自動保存後のフラグメント更新（1 ファイル）

**現在（Datastar）**:

```templ
<div data-on:draft-autosaved__window="@get('/s/{space}/pages/{page}/draft_page')">
```

Go ハンドラー側で SSE + 3 つの `PatchElementTempl` 呼び出し:

1. `#page-draft-saved-at`（Outer モード）
2. `#page-link-list`（Inner モード）
3. `#page-backlink-list`（Inner モード）

**移行後（htmx 4）**:

カスタムイベント `draft-autosaved` を `hx-trigger` で受信し、`hx-get` で HTML フラグメントを取得。レスポンスに OOB スワップ用の要素を含めて 3 要素を同時更新する。

```templ
<div hx-get="/s/{space}/pages/{page}/draft_page"
     hx-trigger="draft-autosaved from:window"
     hx-target="#page-draft-saved-at"
     hx-swap="outerHTML">
</div>
```

Go ハンドラーは通常の HTML レスポンスとして以下を返す:

```html
<!-- メインターゲット -->
<span id="page-draft-saved-at">保存済み: 12:34</span>

<!-- OOB スワップ -->
<div id="page-link-list" hx-swap-oob="innerHTML">...</div>
<div id="page-backlink-list" hx-swap-oob="innerHTML">...</div>
```

### JS ベンダーファイルの差し替え

- `static/js/vendor/datastar-v1.0.0-RC.7.js` を削除
- htmx 4 の JS ファイルを `static/js/vendor/htmx-4.0.0.js`（リリースバージョンに合わせる）として配置
- `head.templ` の `<script>` タグを差し替え

### CSRF 対策

Datastar では `@post()` 等のアクションがリクエストボディに JSON を送信するため、シグナルに CSRF トークンを含めて `ReadSignals` で検証する独自のパターンが必要だった。

htmx 4 では、通常の HTML フォーム送信と同様に `<input type="hidden" name="csrf_token">` でトークンを送信するため、既存の CSRF ミドルウェアがそのまま機能する。

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### htmx 2.x を使用する

htmx 2.x は安定版だが、SSE 対応が extension 依存であり、fetch API ベースの新しいアーキテクチャを持つ htmx 4 のほうが将来的に有利。htmx 4 はまだ alpha だが、Wikino の開発ペースでは安定版リリースまでに十分な時間がある。

### Alpine.js を併用する

htmx + Alpine.js の組み合わせは一般的だが、Wikino のクライアント側の動的 UI ニーズは限定的（フォーム二重送信防止程度）であり、htmx 単体で対応できる。依存を増やすメリットがない。

### フォーム送信を `hx-post` に変換して `hx-disable` を使用する

htmx 4 の `hx-disable` はリクエスト中に要素を自動で disabled にするが、htmx 経由（`hx-post` 等）でフォーム送信される場合のみ機能する。フォームを `hx-post` 化すれば `hx-disable` が使えるが、以下の理由で不採用とした：

- ハンドラー側の変更が必要（成功時は `HX-Redirect` ヘッダー、エラー時はフォーム HTML の再レンダリング）で、テンプレート変更だけでは完結しない
- `edit.templ` のフォームは `formaction` 属性で複数の送信先を切り替えており、`hx-post` 化すると各ボタンに異なる `hx-post` を設定する必要があり複雑になる
- 対象ハンドラーが多岐にわたり（sign_in, sign_up, password_reset, password, email_confirmation, account, page）、影響範囲が大きい

代わりに `hx-on:submit` による DOM イベントリスナー方式を採用した。通常の HTML フォーム送信を維持したまま、テンプレート変更のみで二重送信防止を実現できる。

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

### フェーズ 1: 準備

- [x] **1-1**: htmx 4 スキルファイルの作成
  - `.claude/skills/htmx4/SKILL.md` を作成
  - Datastar スキルの構成を参考に、htmx 4 の API リファレンス・テンプレートパターン・Go 統合パターンを記述
  - **想定ファイル数**: 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 200 行（実装 200 行 + テスト 0 行）

- [x] **1-2**: [Go] htmx JS ファイルの配置と読み込み設定
  - htmx 4 の JS ファイルを `static/js/vendor/` に配置
  - `head.templ` に htmx の `<script>` タグを追加（Datastar と並行して読み込み、段階的移行を可能にする）
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 10 行（実装 10 行 + テスト 0 行）

### フェーズ 2: ページネーション系ハンドラーの移行

<!--
SSEが不要になる3箇所を移行。ハンドラーをSSEレスポンスから通常HTMLレスポンスに変更し、
テンプレートのDatastar属性をhtmx属性に置き換える。
-->

- [x] **2-1**: [Go] ページネーション系ハンドラーの一括移行（リンク一覧 + バックリンク）
  - `/htmx4` スキルを使用して実装する
  - `LoadMoreButton` 共有コンポーネントを htmx 対応に変更するため、バックリンク系も同時に移行
  - `internal/templates/components/load_more_button.templ` を htmx 対応に変更（`data-on:click` → `hx-get` + `hx-target` + `hx-swap`）
  - `internal/templates/components/link_list.templ` の Datastar 属性を htmx 属性に置き換え、SSE 用コンポーネントを `LinkListResponse` に統合
  - `internal/templates/components/backlink_list.templ` の Datastar 属性を htmx 属性に置き換え、SSE 用コンポーネントを削除
  - `internal/templates/components/page_backlink_list.templ` の Datastar 属性を htmx 属性に置き換え、`PageBacklinkListResponse` を追加
  - `internal/handler/page_link_list/show.go` から `datastar-go` SDK を除去し、通常の HTML レスポンスに変更
  - `internal/handler/page_backlinks/show.go` から `datastar-go` SDK を除去し、通常の HTML レスポンスに変更
  - `internal/handler/page_backlink_list/show.go` から `datastar-go` SDK を除去し、通常の HTML レスポンスに変更
  - ハンドラーテストの更新（SSE → HTML レスポンスのアサーションに変更）
  - **想定ファイル数**: 約 13 ファイル（実装 10 + テスト 3）
  - **想定行数**: 約 250 行（実装 150 行 + テスト 100 行）

### フェーズ 3: 下書き自動保存フラグメント更新の移行

- [x] **3-1**: [Go] 下書き自動保存の SSE レスポンスを OOB スワップに移行
  - `/htmx4` スキルを使用して実装する
  - `internal/handler/draft_page/show.go` から `datastar-go` SDK を除去し、OOB スワップを含む HTML レスポンスに変更
  - `internal/templates/pages/page/edit.templ` の `data-on:draft-autosaved__window` を `hx-trigger="draft-autosaved from:window"` に置き換え
  - `edit.templ` の `$isSubmitting` シグナルはフェーズ 4 でまとめて対応する（edit.templ のフォームは複数の送信ボタンと `formaction` を持ち、htmx 化にはページ更新ハンドラーの変更も必要なため、他フォームと一括で対応するほうが効率的）
  - ハンドラーテストの更新
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 120 行（実装 70 行 + テスト 50 行）

### フェーズ 4: フォーム二重送信防止の移行

<!--
$isSubmittingシグナルをhtmxのフォーム送信制御に置き換える。
edit.templはフェーズ3で対応済みのため、残り6ファイルを移行。
-->

- [x] **4-1**: [Go] フォーム二重送信防止の移行
  - `/htmx4` スキルを使用して実装する
  - 以下 7 ファイルの `data-on:submit__passive` + `data-attr:disabled` を `hx-on:submit` による送信ボタン無効化に置き換え
  - フォームの `submit` イベントで全送信ボタンを disabled にする方式を採用（通常の HTML フォーム送信を維持、ハンドラー変更不要）
    - `internal/templates/pages/page/edit.templ`（フェーズ 3 で `$isSubmitting` を残した分）
    - `internal/templates/pages/account/new.templ`
    - `internal/templates/pages/email_confirmation/edit.templ`
    - `internal/templates/pages/password/edit.templ`
    - `internal/templates/pages/password/reset.templ`
    - `internal/templates/pages/sign_in/new.templ`
    - `internal/templates/pages/sign_up/new.templ`
  - **想定ファイル数**: 約 7 ファイル（実装 7 + テスト 0）
  - **想定行数**: 約 70 行（実装 70 行 + テスト 0 行）

### フェーズ 5: クリーンアップ

- [x] **5-1**: [Go] Datastar 依存の完全除去
  - `/htmx4` スキルを使用して実装する
  - `head.templ` から Datastar の `<script>` タグを削除
  - `static/js/vendor/datastar-v1.0.0-RC.7.js` を削除
  - `go.mod` / `go.sum` から `github.com/starfederation/datastar-go` を削除（`go mod tidy`）
  - CLAUDE.md、ガイドドキュメントから Datastar への言及を htmx に更新
  - **想定ファイル数**: 約 5 ファイル（実装 5 + テスト 0）
  - **想定行数**: 約 30 行（実装 30 行 + テスト 0 行）

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **htmx 4 の SSE extension の導入**: 現時点では SSE が必要なユースケースがないため、通常の HTML レスポンスで対応する。将来的にリアルタイム更新が必要になった場合に検討する
- **フォーム送信の htmx 化（`hx-post` 等）**: フォーム二重送信防止は `hx-on:submit` による DOM イベントリスナー方式を採用したため、フォーム送信の htmx 化は不要。将来的にフォームのプログレッシブエンハンスメントが必要になった場合に検討する

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- htmx 4 ソースコード・ドキュメント: `.claude/skills/htmx4/src/htmx-4.0.0-alpha8/`
- htmx 4 マイグレーションガイド: `.claude/skills/htmx4/src/htmx-4.0.0-alpha8/www/src/content/docs/01-get-started/02-migration.md`
