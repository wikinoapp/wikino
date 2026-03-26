# 日時表示の統一 作業計画書

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

- タスク完了後に `docs/specs/datetime/overview.md` に仕様書を作成予定

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

Go 版において、日時をユーザーに表示する際の共通的な仕組みを整備する。
現在、日時表示はページごとに個別のフォーマット関数やクライアントサイド JS で対応しており、タイムゾーンの扱いやフォーマットに一貫性がない。

本計画では、サーバーサイドでタイムゾーン変換とフォーマットを統一的に行うヘルパーと templ コンポーネントを整備し、既存の日時表示箇所をすべて移行する。

### 現状の課題

Go 版には日時を表示する箇所が 3 つあり、それぞれ異なるアプローチをとっている:

| 箇所                                      | ファイル                               | 現在のアプローチ                                      |
| ----------------------------------------- | -------------------------------------- | ----------------------------------------------------- |
| 編集提案・コメント（Post コンポーネント） | `components/post.templ`                | UTC 固定表示 + クライアント JS で変換                 |
| 下書き自動保存時刻                        | `components/draft_page_response.templ` | `formatTimeInZone()` でサーバーサイド変換（時刻のみ） |
| 下書き一覧の更新日時                      | `viewmodel/draft_page_for_index.go`    | ViewModel 内で事前フォーマット済み文字列に変換        |

問題点:

- タイムゾーン取得方法がバラバラ（ログインユーザーの `user.TimeZone` のみで、未ログイン時の考慮なし）
- フォーマットが統一されていない（`15:04`、`2006-01-02 15:04`、`2006-01-02 15:04 UTC`）
- 相対時間（「3 分前」など）の表示機能がない
- `formatTimeInZone()` が `components` パッケージ内のローカル関数で再利用しにくい

## 要件

<!--
ガイドライン:
- 機能要件: 「何ができるべきか」を記述
- 非機能要件: 「どのように動くべきか」を必要に応じて記述
-->

### 機能要件

- **タイムゾーン解決**: ログイン中は `users.time_zone` を使用し、未ログイン時はクッキーに保存されたブラウザタイムゾーンを使用する
- **相対時間表示**: 「3 分前」「1 時間前」「昨日」のような相対時間で表示でき、`title` 属性に `2026/03/25 14:14` 形式の絶対時間を設定する
- **絶対時間表示**: `2026/03/25 14:14` 形式で表示する。`title` 属性は不要
- **コンポーネント化**: 上記 2 種類の表示をテンプレートから呼び出せる共通コンポーネントとして提供する
- **既存箇所の移行**: 現在日時を表示している 3 箇所すべてを新しい仕組みに移行する

### 非機能要件

- **パフォーマンス**: タイムゾーン変換はサーバーサイドで行い、追加の JS ランタイムコストを発生させない
- **保守性**: 日時フォーマットの変更が 1 箇所の修正で済むようにする

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
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン（**ファイル名は標準の9種類のみ**）
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
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

### 全体像

```
[ブラウザ]
  ├─ 初回アクセス時: JS で Intl.DateTimeFormat().resolvedOptions().timeZone を取得
  └─ クッキー `wikino_time_zone` にタイムゾーン文字列を保存（例: "Asia/Tokyo"）

[サーバー]
  ├─ ミドルウェア: リクエストからタイムゾーンを解決し context に格納
  │    1. ログイン中 → users.time_zone
  │    2. 未ログイン → クッキー `wikino_time_zone` の値
  │    3. どちらもなし → UTC
  │
  ├─ テンプレートヘルパー: context からタイムゾーンを取得し日時をフォーマット
  │    - FormatDateTime(ctx, t)     → "2026/03/25 14:14"
  │    - RelativeTime(ctx, t)      → "3分前" / "1時間前" / "昨日" / "2026/03/25 14:14"
  │    - FormatTime(ctx, t)        → "14:14"
  │
  └─ templ コンポーネント: 表示モードに応じた HTML を生成
       - RelativeTime  → <time title="2026/03/25 14:14" datetime="...">3分前</time>
       - AbsoluteTime  → <span>2026/03/25 14:14</span>
```

### タイムゾーンクッキーの設計

未ログインユーザーのタイムゾーンをサーバーサイドで利用するため、ブラウザのタイムゾーンをクッキーに保存する。

**JS（`main.js`）:**

```js
function setTimeZoneCookie() {
  if (document.cookie.includes("wikino_time_zone=")) return;
  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
  if (tz) {
    document.cookie = `wikino_time_zone=${tz};path=/;max-age=${60 * 60 * 24 * 365};SameSite=Lax`;
  }
}
```

- 初回アクセス時にのみセット（既にクッキーがあればスキップ）
- `SameSite=Lax` で CSRF リスクを最小化
- 有効期限 1 年
- クッキー名: `wikino_time_zone`

### タイムゾーン解決ミドルウェア

`internal/middleware/timezone.go` に配置する。

```go
type contextKey string
const timeZoneKey contextKey = "timezone"

func TimeZone(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tz := resolveTimeZone(r)
        ctx := context.WithValue(r.Context(), timeZoneKey, tz)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func resolveTimeZone(r *http.Request) string {
    // 1. ログインユーザーの設定を優先
    if user := UserFromContext(r.Context()); user != nil && user.TimeZone != "" {
        return user.TimeZone
    }
    // 2. クッキーから取得
    if c, err := r.Cookie("wikino_time_zone"); err == nil && c.Value != "" {
        // time.LoadLocation で検証して不正な値を除外
        if _, err := time.LoadLocation(c.Value); err == nil {
            return c.Value
        }
    }
    // 3. フォールバック
    return "UTC"
}

func TimeZoneFromContext(ctx context.Context) string {
    if tz, ok := ctx.Value(timeZoneKey).(string); ok {
        return tz
    }
    return "UTC"
}
```

- 認証ミドルウェアより後に配置（`UserFromContext` を使うため）
- `time.LoadLocation` でクッキーの値を検証し、不正な値は無視する

### テンプレートヘルパー

`internal/templates/helper.go` に日時フォーマット関数を追加する。

```go
// FormatDateTime は日時を "2026/03/25 14:14" 形式でフォーマットする
func FormatDateTime(ctx context.Context, t time.Time) string {
    loc := loadLocationFromContext(ctx)
    return t.In(loc).Format("2006/01/02 15:04")
}

// FormatTime は時刻を "14:14" 形式でフォーマットする
func FormatTime(ctx context.Context, t time.Time) string {
    loc := loadLocationFromContext(ctx)
    return t.In(loc).Format("15:04")
}

// RelativeTime は相対時間文字列を返す
// 1分未満: "たった今"
// 1〜59分: "N分前"
// 1〜23時間: "N時間前"
// 1〜3日: "N日前"
// 3日超: "2026/03/25 14:14"（絶対時間にフォールバック）
func RelativeTime(ctx context.Context, t time.Time) string { ... }

func loadLocationFromContext(ctx context.Context) *time.Location {
    tz := timezone.FromContext(ctx)
    loc, err := time.LoadLocation(tz)
    if err != nil {
        return time.UTC
    }
    return loc
}
```

### templ コンポーネント

`internal/templates/components/datetime.templ` に配置する。

```templ
// RelativeTimeData は相対時間コンポーネントのデータです
type RelativeTimeData struct {
    Time time.Time
}

// 相対時間を表示するコンポーネント
// title 属性に絶対時間を設定する
templ RelativeTime(data RelativeTimeData) {
    <time
        datetime={ data.Time.UTC().Format(time.RFC3339) }
        title={ templates.FormatDateTime(ctx, data.Time) }
    >
        { templates.RelativeTime(ctx, data.Time) }
    </time>
}

// AbsoluteTimeData は絶対時間コンポーネントのデータです
type AbsoluteTimeData struct {
    Time time.Time
}

// 絶対時間を表示するコンポーネント
templ AbsoluteTime(data AbsoluteTimeData) {
    <time datetime={ data.Time.UTC().Format(time.RFC3339) }>
        { templates.FormatDateTime(ctx, data.Time) }
    </time>
}
```

### 相対時間の表示ルール

| 経過時間   | 日本語                                       | 英語          |
| ---------- | -------------------------------------------- | ------------- |
| 1 分未満   | たった今                                     | just now      |
| 1〜59 分   | N 分前                                       | N minutes ago |
| 1〜23 時間 | N 時間前                                     | N hours ago   |
| 1〜3 日    | N 日前                                       | N days ago    |
| 3 日超     | 絶対時間にフォールバック（2026/03/25 14:14） | 同左          |

### 既存箇所の移行計画

#### 1. Post コンポーネント（`components/post.templ`）

**変更前:**

```templ
<time datetime={ ... } class="text-muted-foreground" data-local-time>
    { data.CreatedAt.UTC().Format("2006-01-02 15:04 UTC") }
</time>
```

**変更後:** `components.RelativeTime` コンポーネントを使用する。
`data-local-time` 属性と対応する JS（`formatLocalTimes()`）は不要になるため削除する。

#### 2. 下書き自動保存時刻（`components/draft_page_response.templ`）

**変更前:** ローカル関数 `formatTimeInZone()` でサーバーサイド変換。`TimeZone` を `DraftPageShowResponseData` のフィールドとして受け取る。

**変更後:** `templates.FormatTime(ctx, t)` ヘルパーを使用する。
タイムゾーンは context から取得するため、`DraftPageShowResponseData.TimeZone` フィールドは削除する。
`formatTimeInZone()` 関数も削除する。

#### 3. 下書き一覧（`viewmodel/draft_page_for_index.go`）

**変更前:** ViewModel 内で `ModifiedAt` を `string` 型に事前フォーマットする。`loadLocation()` 関数を ViewModel 内に持つ。

**変更後:** `ModifiedAt` を `time.Time` 型に変更し、テンプレート側で `templates.FormatDateTime(ctx, t)` を使ってフォーマットする。
`loadLocation()` 関数と `NewDraftPageGroupsForIndex` の `timeZone` 引数は削除する。

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### A. クライアントサイド JS のみで日時をフォーマットする

`<time datetime="...">` に UTC の ISO 8601 値を埋め込み、JS の `toLocaleString()` で表示を書き換える方式。

**不採用の理由:**

- JS が無効またはロード前に UTC の生文字列が一瞬表示される（FOUC）
- 相対時間のロジックを JS にも持つ必要があり、Go 側と二重管理になる
- サーバーサイドレンダリングを基本方針とするプロジェクトに合わない

### B. ViewModel で日時を事前フォーマット済み文字列にする（現行の下書き一覧方式）

ViewModel のコンストラクタ内で `time.Time` → `string` に変換し、テンプレートはフォーマット済み文字列をそのまま表示する方式。

**不採用の理由:**

- 相対時間と絶対時間を切り替えるには ViewModel に両方のフォーマットを持たせる必要があり、肥大化する
- タイムゾーンを ViewModel のコンストラクタに渡す必要があり、すべての UseCase → Handler → ViewModel の経路でタイムゾーンを引き回す必要がある
- ミドルウェアで context にタイムゾーンを格納し、テンプレートヘルパーで参照する方がシンプル

### C. テンプレートヘルパーから `middleware.TimeZoneFromContext` を直接呼び出す

当初の設計では `loadLocationFromContext` から `middleware.TimeZoneFromContext` を直接呼び出す方針だった。

**不採用の理由:**

- `middleware` → `templates/pages/errors` → `templates` の循環インポートが発生する
- `internal/timezone` パッケージにコンテキスト操作関数を切り出し、`middleware` と `templates` の両方から参照する方式に変更した
- `middleware.SetTimeZoneToContext` / `middleware.TimeZoneFromContext` は削除し、全呼び出し元を `timezone.ToContext` / `timezone.FromContext` に移行した

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

### フェーズ 1: タイムゾーン解決の基盤

<!--
例: インフラ準備、基本機能実装、セキュリティ機能など
各タスクは1つのPull Requestで完結する粒度で記述してください
各タスクには想定サイズを明記してください
Go版/Rails版の両方を修正する場合は別タスクに分けてください
-->

- [x] **1-1**: [Go] タイムゾーンクッキーの設定とミドルウェアの実装
  - `web/main.js` にタイムゾーンクッキー設定 JS を追加し、前回追加した `formatLocalTimes()` を削除する
  - `internal/middleware/timezone.go` にタイムゾーン解決ミドルウェアを実装
  - `internal/middleware/timezone.go` に `TimeZoneFromContext()` を実装
  - `cmd/server/main.go` のミドルウェアチェーンにタイムゾーンミドルウェアを登録（認証ミドルウェアの後）
  - **想定ファイル数**: 約 5 ファイル（実装 3 + テスト 2）
  - **想定行数**: 約 200 行（実装 80 行 + テスト 120 行）

- [x] **1-2**: [Go] アカウント作成時にクッキーからタイムゾーンを保存
  - `internal/handler/account/create.go` でハードコードされている `TimeZone: "Asia/Tokyo"` を `middleware.TimeZoneFromContext(ctx)` に置き換える
  - 既存テストを更新
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 20 行（実装 5 行 + テスト 15 行）

### フェーズ 2: ヘルパーとコンポーネントの実装

- [x] **2-1**: [Go] テンプレートヘルパーの実装
  - `internal/templates/helper.go` に `FormatDateTime(ctx, t)`、`FormatTime(ctx, t)`、`RelativeTime(ctx, t)` を追加
  - `internal/templates/helper.go` に内部関数 `loadLocationFromContext(ctx)` を追加
  - 相対時間の i18n エントリを `ja.toml`、`en.toml` に追加（`datetime_just_now`、`datetime_minutes_ago`、`datetime_hours_ago`、`datetime_days_ago`）
  - 循環インポート回避のため `internal/timezone` パッケージを新設（タイムゾーンのコンテキスト操作を `middleware` から分離）
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 250 行（実装 100 行 + テスト 150 行）

- [x] **2-2**: [Go] 日時表示 templ コンポーネントの実装
  - `internal/templates/components/datetime.templ` に `RelativeTime` コンポーネントと `AbsoluteTime` コンポーネントを実装
  - **想定ファイル数**: 約 3 ファイル（実装 1 + テスト 2）
  - **想定行数**: 約 150 行（実装 40 行 + テスト 110 行）

### フェーズ 3: 既存箇所の移行

- [x] **3-1**: [Go] Post コンポーネントの移行
  - `components/post.templ` の `<time data-local-time>` を `components.RelativeTime` に置き換え
  - `PostData` 構造体から `CreatedAt time.Time` はそのまま維持
  - `web/main.js` から `formatLocalTimes()` の呼び出しを削除（1-1 で JS 自体は削除済み）
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 20 行（実装 20 行 + テスト 0 行）

- [x] **3-2**: [Go] 下書き自動保存時刻の移行
  - `components/draft_page_response.templ` の `formatTimeInZone()` 呼び出しを `templates.FormatTime(ctx, t)` に置き換え
  - `DraftPageShowResponseData` から `TimeZone` フィールドを削除
  - `formatTimeInZone()` 関数を削除
  - `internal/handler/draft_page/show.go` から `responseData.TimeZone = user.TimeZone` の行を削除
  - 既存テスト（`draft_page_response_test.go`）を新しい方式に合わせて更新
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 60 行（実装 30 行 + テスト 30 行）

- [x] **3-2a**: [Go] RelativeTime コンポーネントの title 属性の条件付き表示
  - `templates.IsRelativeTime(t)` ヘルパーを追加し、72 時間以内かどうかを判定
  - `components/datetime.templ` の `RelativeTime` で、絶対時間フォールバック時は `title` 属性を付与しないように修正
  - 既存テストを更新し、`IsRelativeTime` のテストを追加
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 2）
  - **想定行数**: 約 60 行（実装 10 行 + テスト 50 行）

- [x] **3-3**: [Go] 下書き一覧の移行
  - `viewmodel/draft_page_for_index.go` の `ModifiedAt` を `string` → `time.Time` に変更
  - `NewDraftPageGroupsForIndex` の `timeZone` 引数を削除し、`loadLocation()` を削除
  - `pages/draft_page/index.templ` で `templates.FormatDateTime(ctx, draft.ModifiedAt)` を使用
  - `internal/handler/draft_page_index/index.go` から `user.TimeZone` の引き渡しを削除
  - 既存テストを更新
  - **想定ファイル数**: 約 5 ファイル（実装 4 + テスト 1）
  - **想定行数**: 約 80 行（実装 40 行 + テスト 40 行）

### フェーズ 4: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [x] **4-1**: 仕様書の作成
  - `docs/specs/datetime/overview.md` に仕様書を作成する
  - タイムゾーン解決の優先順位、フォーマット仕様、相対時間の表示ルール、採用しなかった方針を記載する

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **ユーザー設定画面でのタイムゾーン変更 UI**: 既存の `users.time_zone` カラムへの書き込みは Rails 版で行う。Go 版でのタイムゾーン設定 UI は別タスクとする
- **秒単位の表示**: 分単位までの表示とし、秒は表示しない
- **日付のみの表示（時刻なし）**: 現在のユースケースでは不要。必要になった時点で `FormatDate()` ヘルパーを追加する
