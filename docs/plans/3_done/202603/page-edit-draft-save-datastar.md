# ページ編集画面の下書き保存時刻表示を Datastar SSE に移行 作業計画書

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

- [ページ編集 仕様書](../specs/page/edit.md)

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

ページ編集画面の下書き自動保存において、保存時刻の表示をサーバーサイドレンダリングに移行し、リンク一覧の更新と統合する。

現在の実装では、`saveAsDraft` 関数が `fetch` で PATCH リクエストを送り、JSON レスポンスから `modified_at` を取得して JavaScript 側で `toLocaleTimeString` を使って時刻文字列を生成し、DOM を直接操作して表示している。リンク一覧の更新は `draft-autosaved` カスタムイベントを dispatch し、Datastar の `@get()` で別途 SSE エンドポイントを呼び出す 2 リクエスト構成になっている。

PATCH レスポンスを 204 No Content（保存のみ）に簡素化し、保存成功後に `draft-autosaved` カスタムイベントを dispatch する。Datastar の `@get()` が SSE エンドポイントを呼び出し、保存時刻とリンク一覧の両方を SSE フラグメントとして返す。これにより、JS 側の JSON パースと DOM 操作を削除でき、UI 更新のレンダリングロジックを SSE エンドポイントに一元化できる。

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

- 下書きが自動保存されると、保存時刻が画面に表示される（現在と同等の動作）
- 保存時刻の表示位置は、保存/キャンセルボタンの下（Rails 版の `#markdown-editor-draft-saved-time` と同様）
- 保存時刻のフォーマットは templ テンプレート側で管理し、国際化に対応する
- リンク一覧の更新は現在と同等に動作する

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

### 全体アーキテクチャ

自動保存は CodeMirror の `EditorView.updateListener` から発火する命令的な処理のため、Datastar の `@patch()` は使えない（`@patch()` はシグナルを JSON で送信し SSE レスポンスを期待するが、自動保存は FormData で送信する必要がある）。そこで以下の設計とする：

1. **PATCH は保存のみに専念**し、204 No Content を返す（JSON レスポンスを廃止）
2. JS は保存成功後に `draft-autosaved` カスタムイベントを dispatch する（既存の仕組みを維持）
3. Datastar の `data-on:draft-autosaved__window="@get(...)"` が SSE エンドポイントを呼び出す
4. SSE エンドポイントが保存時刻とリンク一覧の HTML フラグメントを返し、Datastar が DOM を更新する

PATCH と GET の責務を分離することで、JS を薄く保ち（DOMParser 不要）、UI 更新のレンダリングロジックを SSE エンドポイントに一元化できる。

### 処理フロー

```
[CodeMirror change]
    ↓ (debounce 500ms)
[JS: fetch PATCH /go/s/{id}/pages/{num}/draft_page]
    ↓ (204 No Content)
[JS: window.dispatchEvent(new CustomEvent("draft-autosaved"))]
    ↓
[Datastar: data-on:draft-autosaved__window="@get('/go/s/{id}/pages/{num}/draft_page')"]
    ↓ (SSE レスポンス)
[Datastar: SSE フラグメントで DOM を更新]
    ├─ #page-draft-saved-at を更新（保存時刻）
    └─ #page-link-list を更新（リンク一覧）
```

### API 設計

#### PATCH `/go/s/{space_identifier}/pages/{page_number}/draft_page` の変更

- **Before**: JSON レスポンス `{"modified_at": "2024-01-01T00:00:00Z"}`
- **After**: 204 No Content（レスポンスボディなし）

#### GET `/go/s/{space_identifier}/pages/{page_number}/draft_page` の新設

既存の GET `link_list` エンドポイントを GET `draft_page` にリネームし、保存時刻フラグメントを追加する。

SSE レスポンスで以下の 2 つのフラグメントを返す：

1. `#page-draft-saved-at`（保存時刻、`outer` モードで要素全体を置換）
2. `#page-link-list`（リンク一覧、既存と同様に `inner` モードで内部 HTML を置換）

#### GET `/go/s/{space_identifier}/pages/{page_number}/link_list` の廃止

GET `draft_page` に統合するため、`link_list` エンドポイントは廃止する。`edit.templ` の `data-on:intersect` と `data-on:draft-autosaved__window` の両方が GET `draft_page` を呼び出すように変更する。

### テンプレート設計

#### 新規コンポーネント: `components/draft_saved_time.templ`

`time.Time` を受け取り、「自動保存: **HH:MM**」形式で表示する。

```templ
templ DraftSavedTime(modifiedAt time.Time) {
    <div class="text-xs">
        @templ.Raw(templates.T(ctx, "page_edit_draft_saved_time", map[string]any{"Time": modifiedAt.Format("15:04")}))
    </div>
}
```

#### SSE エンドポイントのレスポンス

GET `draft_page` の SSE ハンドラーで、保存時刻とリンク一覧の 2 つのフラグメントを `sse.PatchElementTempl()` で送信する。専用のレスポンステンプレートファイルは不要（既存の `page_link_list/show.go` と同様に、ハンドラー内で個別にフラグメントを送信する）。

#### `edit.templ` の変更

- 保存/キャンセルボタンの下に `#page-draft-saved-at` 要素を追加
- `data-markdown-editor-saved-at` 属性を削除
- `data-on:draft-autosaved__window` の URL を `link_list` → `draft_page` に変更
- `data-on:intersect` の URL を `link_list` → `draft_page` に変更

```templ
<div class="flex items-center gap-2">
    <!-- 保存・キャンセルボタン（既存） -->
</div>

<div id="page-draft-saved-at" class="text-right"></div>
```

### i18n 設計

```toml
# ja.toml
[page_edit_draft_saved_time]
description = "自動保存時刻の表示"
other = "自動保存: <span class=\"font-bold\">{{.Time}}</span>"

# en.toml
[page_edit_draft_saved_time]
description = "Auto-saved time display"
other = "Auto-saved: <span class=\"font-bold\">{{.Time}}</span>"
```

### 変更対象ファイル一覧

| ファイル                                                  | 変更種別 | 変更内容                                                                                                                                |
| --------------------------------------------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `go/internal/i18n/locales/ja.toml`                        | 修正     | `page_edit_draft_saved_time` を追加                                                                                                     |
| `go/internal/i18n/locales/en.toml`                        | 修正     | `page_edit_draft_saved_time` を追加                                                                                                     |
| `go/internal/templates/components/draft_saved_time.templ` | 新規     | 自動保存時刻コンポーネント                                                                                                              |
| `go/internal/templates/pages/page/edit.templ`             | 修正     | `#page-draft-saved-at` 追加、`data-markdown-editor-saved-at` 削除、`data-on:draft-autosaved__window` と `data-on:intersect` の URL 変更 |
| `go/internal/handler/draft_page/update.go`                | 修正     | 204 No Content を返すように変更（JSON レスポンスの廃止）                                                                                |
| `go/internal/handler/draft_page/show.go`                  | 新規     | GET SSE エンドポイント（保存時刻 + リンク一覧のフラグメントを返す）                                                                     |
| `go/internal/handler/draft_page/handler.go`               | 修正     | Show メソッド用の依存性を追加                                                                                                           |
| `go/internal/handler/page_link_list/`                     | 削除     | `draft_page` に統合するため廃止                                                                                                         |
| `go/cmd/server/main.go`                                   | 修正     | ルーティング変更（`link_list` → `draft_page` の GET 追加）                                                                              |
| `go/web/markdown-editor/markdown-editor.ts`               | 修正     | JSON パースと DOM 操作の削除、`savedAtEl` の削除（`draft-autosaved` イベント dispatch は維持）                                          |

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### PATCH エンドポイントを SSE に変更する

PATCH `/go/s/{id}/pages/{num}/draft_page` 自体を SSE レスポンスにし、フラグメントで時刻とリンク一覧を返す案を検討した。

**不採用の理由**:

- 自動保存は CodeMirror の `updateListener` から命令的に呼ばれるため、Datastar の `@patch()` 属性が使えない（`@patch()` はシグナルを JSON で送信するが、自動保存は FormData で送信する必要がある）
- `fetch` で PATCH を送る部分は残す必要があり、SSE レスポンスを受け取っても Datastar が自動的に処理しない

### PATCH が `text/html` で保存時刻とリンク一覧を返す（1 リクエスト方式）

PATCH レスポンスを `text/html` に変更し、保存時刻とリンク一覧の HTML フラグメントを含めることで、1 回のリクエストで両方の UI 更新を完結させる案を検討した。JS 側で `DOMParser` を使ってレスポンス HTML をパースし、ID で対象要素を特定して DOM を更新する。

**不採用の理由**:

- GET `link_list`（または `draft_page`）SSE エンドポイントは初回表示時の遅延読み込みで引き続き必要であり、リンク一覧のレンダリングロジックが PATCH ハンドラーと SSE ハンドラーの 2 箇所に重複する
- JS に `DOMParser` によるパースロジックが必要になり、JS が厚くなる
- Datastar の SSE フラグメント更新パターンを活用できず、独自の DOM 更新ロジックを実装する必要がある

### 保存時刻を専用の SSE エンドポイントで返す

`GET /go/s/{id}/pages/{num}/draft_saved_time` のような専用エンドポイントを作り、別途 SSE フラグメントを返す案を検討した。

**不採用の理由**:

- リンク一覧の更新と保存時刻の表示は常に同じタイミング（下書き保存後）で行われる
- エンドポイントを分けると、同じイベントで 2 つの GET リクエストが飛ぶことになり非効率

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

### フェーズ 1: 下書き保存の Datastar SSE 移行

- [x] **1-1**: [Go] GET `draft_page` SSE エンドポイントの新設と PATCH の簡素化
  - i18n 翻訳の追加（`page_edit_draft_saved_time` を ja.toml、en.toml に追加）
  - `components/draft_saved_time.templ` の新規作成
  - `handler/draft_page/show.go` の新規作成（GET SSE エンドポイント、保存時刻 + リンク一覧フラグメントを返す）
  - `handler/draft_page/handler.go` の修正（Show メソッド用の依存性を追加）
  - `handler/draft_page/update.go` の修正（204 No Content を返すように変更）
  - `pages/page/edit.templ` の修正（`#page-draft-saved-at` 追加、`data-markdown-editor-saved-at` 削除、URL を `link_list` → `draft_page` に変更）
  - `handler/page_link_list/` の削除（`draft_page` に統合）
  - `cmd/server/main.go` のルーティング変更
  - `web/markdown-editor/markdown-editor.ts` の修正（JSON パースと DOM 操作の削除、`savedAtEl` の削除）
  - **想定ファイル数**: 約 10 ファイル（実装 8 + 自動生成 2）
  - **想定行数**: 約 100 行（実装 100 行、テストは既存テストで担保）

### フェーズ 2: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [x] **2-1**: 仕様書の更新
  - `docs/specs/page/edit.md` を更新する
  - 下書き保存時刻表示の仕様を追加
  - 採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **保存失敗時のエラー表示**: 現在と同様、自動保存の失敗は静かに無視する。ユーザー通知が必要になった場合に別途検討する
- **タイトル変更時の自動保存トリガー**: 現在は本文変更時のみ自動保存される。タイトル変更時のトリガーは別の課題として扱う

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Rails 版 edit_view.html.erb の `#markdown-editor-draft-saved-time`](/workspace/rails/app/views/pages/edit_view.html.erb)
- [Rails 版 draft_pages/update_view.html.erb](/workspace/rails/app/views/draft_pages/update_view.html.erb)
- [Datastar 公式サイト](https://data-star.dev/)
