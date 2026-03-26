# 日時表示 仕様書

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

Go 版において、日時をユーザーに表示する際の共通的な仕組みである。サーバーサイドでタイムゾーン変換とフォーマットを統一的に行うヘルパーと templ コンポーネントを提供し、すべての日時表示箇所で使用する。

タイムゾーンはミドルウェアでリクエストごとに解決され、context に格納される。テンプレートヘルパーが context からタイムゾーンを取得してフォーマットを行うため、ハンドラーや ViewModel でタイムゾーンを意識する必要がない。

**目的**:

- 日時のフォーマットとタイムゾーンの扱いを統一し、1 箇所の変更で全体に反映できるようにする
- 相対時間（「3 分前」など）と絶対時間（「2026/03/25 14:14」）の 2 種類の表示を共通コンポーネントとして提供する
- ログインユーザーも未ログインユーザーも、それぞれのタイムゾーンに合った日時を表示する

**背景**:

- サーバーサイドレンダリングを基本方針とするプロジェクトであり、日時のフォーマットもサーバーサイドで行う
- ミドルウェアで context にタイムゾーンを格納し、テンプレートヘルパーで参照する方式を採用した。これにより、ハンドラーや UseCase、ViewModel にタイムゾーンを引き回す必要がない

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

### タイムゾーン解決

システムはリクエストごとに以下の優先順位でタイムゾーンを解決し、context に格納する:

| 優先順位 | ソース                         | 条件                                                       |
| -------- | ------------------------------ | ---------------------------------------------------------- |
| 1        | ログインユーザーの `time_zone` | ユーザーが認証済みで、`users.time_zone` が空でない場合     |
| 2        | クッキー `wikino_time_zone`    | クッキーが存在し、`time.LoadLocation` で検証に成功した場合 |
| 3        | UTC                            | 上記のいずれにも該当しない場合                             |

- クッキーの値は `time.LoadLocation` で検証され、不正な値（存在しない IANA タイムゾーン名など）は無視される
- タイムゾーンミドルウェアは認証ミドルウェアより後に配置される（ユーザー情報を参照するため）

### タイムゾーンクッキー

- システムはブラウザの JavaScript で `Intl.DateTimeFormat().resolvedOptions().timeZone` を使用してタイムゾーンを検出し、クッキーに保存する
- クッキー名: `wikino_time_zone`
- 初回アクセス時にのみセット（既にクッキーが存在する場合はスキップ）
- 有効期限: 1 年
- `SameSite=Lax`、`path=/`

### 日時フォーマット

システムは以下の 3 種類のフォーマットヘルパーを提供する:

| ヘルパー           | フォーマット例     | 用途                             |
| ------------------ | ------------------ | -------------------------------- |
| `FormatDateTime()` | `2026/03/25 14:14` | 日付と時刻の完全表示             |
| `FormatTime()`     | `14:14`            | 時刻のみの表示                   |
| `RelativeTime()`   | `3分前`            | 相対時間（3 日超は絶対時間表示） |

すべてのヘルパーは context からタイムゾーンを取得し、ユーザーのタイムゾーンに変換してからフォーマットする。

### 相対時間の表示ルール

| 経過時間   | 日本語                                                  | 英語          |
| ---------- | ------------------------------------------------------- | ------------- |
| 1 分未満   | たった今                                                | just now      |
| 1〜59 分   | N 分前                                                  | N minutes ago |
| 1〜23 時間 | N 時間前                                                | N hours ago   |
| 1〜3 日    | N 日前                                                  | N days ago    |
| 3 日超     | 絶対時間にフォールバック（`FormatDateTime` と同じ形式） |               |

- `IsRelativeTime()` ヘルパーで 72 時間（3 日）以内かどうかを判定できる
- 相対時間のメッセージはすべて i18n 対応されている（`datetime_just_now`、`datetime_minutes_ago`、`datetime_hours_ago`、`datetime_days_ago`）

### templ コンポーネント

テンプレートから呼び出せる 2 種類の日時表示コンポーネントを提供する:

**RelativeTime コンポーネント**:

- 相対時間を表示する（例: 「3 分前」）
- 3 日以内の場合: `<time>` 要素の `title` 属性に絶対時間を設定する（マウスオーバーで確認可能）
- 3 日超の場合: 絶対時間にフォールバックし、`title` 属性は付与しない
- `datetime` 属性には UTC の RFC3339 形式を設定する

**AbsoluteTime コンポーネント**:

- 絶対時間を表示する（例: 「2026/03/25 14:14」）
- `<time>` 要素の `datetime` 属性に UTC の RFC3339 形式を設定する
- `title` 属性は付与しない

### 日時表示の使用箇所

| 箇所                 | コンポーネント/ヘルパー       | 表示形式                 |
| -------------------- | ----------------------------- | ------------------------ |
| 編集提案・コメント   | `RelativeTime` コンポーネント | 相対時間（3 日超は絶対） |
| 下書き自動保存時刻   | `FormatTime()` ヘルパー       | 時刻のみ（`14:14`）      |
| 下書き一覧の更新日時 | `FormatDateTime()` ヘルパー   | 絶対時間                 |

### パフォーマンス

- タイムゾーン変換はサーバーサイドで行い、クライアントサイドの JS による追加のランタイムコストは発生しない
- タイムゾーンクッキーの設定 JS は初回のみ実行される

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

### アーキテクチャ

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
       - AbsoluteTime  → <time datetime="...">2026/03/25 14:14</time>
```

### コード設計

#### パッケージ構成

| パッケージ                      | 責務                                                      |
| ------------------------------- | --------------------------------------------------------- |
| `internal/timezone`             | タイムゾーンの context 操作（`ToContext`、`FromContext`） |
| `internal/middleware`           | タイムゾーン解決ミドルウェア（`timezone.go`）             |
| `internal/templates`            | フォーマットヘルパー（`helper.go`）                       |
| `internal/templates/components` | 日時表示 templ コンポーネント（`datetime.templ`）         |

`internal/timezone` パッケージは `middleware` と `templates` の両方から参照される共通パッケージである。循環インポートを回避するために `middleware` からタイムゾーンの context 操作を分離した。

#### タイムゾーン context 操作

```go
// internal/timezone/context.go
package timezone

// ToContext はタイムゾーン文字列を context に格納する
func ToContext(ctx context.Context, tz string) context.Context

// FromContext は context からタイムゾーン文字列を取得する（デフォルト: "UTC"）
func FromContext(ctx context.Context) string
```

#### テンプレートヘルパー

```go
// internal/templates/helper.go

// FormatDateTime は日時を "2006/01/02 15:04" 形式でフォーマットする
func FormatDateTime(ctx context.Context, t time.Time) string

// FormatTime は時刻を "15:04" 形式でフォーマットする
func FormatTime(ctx context.Context, t time.Time) string

// RelativeTime は相対時間文字列を返す
func RelativeTime(ctx context.Context, t time.Time) string

// IsRelativeTime は 72 時間以内かどうかを判定する
func IsRelativeTime(t time.Time) bool
```

#### templ コンポーネント

```go
// internal/templates/components/datetime.templ

type RelativeTimeData struct {
    Time time.Time
}

type AbsoluteTimeData struct {
    Time time.Time
}
```

### ファイル構成

```
go/
├── web/
│   └── main.js                              # タイムゾーンクッキー設定 JS
└── internal/
    ├── timezone/
    │   └── context.go                       # context 操作（ToContext, FromContext）
    ├── middleware/
    │   ├── timezone.go                      # タイムゾーン解決ミドルウェア
    │   └── timezone_test.go                 # ミドルウェアテスト
    ├── templates/
    │   ├── helper.go                        # フォーマットヘルパー
    │   ├── helper_test.go                   # ヘルパーテスト
    │   └── components/
    │       ├── datetime.templ               # 日時表示コンポーネント
    │       └── datetime_test.go             # コンポーネントテスト
    └── i18n/
        └── locales/
            ├── ja.toml                      # 日本語翻訳（datetime_* エントリ）
            └── en.toml                      # 英語翻訳（datetime_* エントリ）
```

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### クライアントサイド JS のみで日時をフォーマットする

`<time datetime="...">` に UTC の ISO 8601 値を埋め込み、JS の `toLocaleString()` で表示を書き換える方式。

**不採用の理由**: JS が無効またはロード前に UTC の生文字列が一瞬表示される（FOUC）。相対時間のロジックを JS にも持つ必要があり、Go 側と二重管理になる。サーバーサイドレンダリングを基本方針とするプロジェクトに合わない。

### ViewModel で日時を事前フォーマット済み文字列にする

ViewModel のコンストラクタ内で `time.Time` → `string` に変換し、テンプレートはフォーマット済み文字列をそのまま表示する方式。

**不採用の理由**: 相対時間と絶対時間を切り替えるには ViewModel に両方のフォーマットを持たせる必要があり、肥大化する。タイムゾーンを ViewModel のコンストラクタに渡す必要があり、すべての UseCase → Handler → ViewModel の経路でタイムゾーンを引き回す必要がある。ミドルウェアで context にタイムゾーンを格納し、テンプレートヘルパーで参照する方がシンプル。

### テンプレートヘルパーから `middleware.TimeZoneFromContext` を直接呼び出す

テンプレートヘルパーの `loadLocationFromContext` から `middleware` パッケージの `TimeZoneFromContext` を直接呼び出す方式。

**不採用の理由**: `middleware` → `templates/pages/errors` → `templates` の循環インポートが発生する。`internal/timezone` パッケージにコンテキスト操作関数を切り出し、`middleware` と `templates` の両方から参照する方式に変更した。

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [internal/timezone/context.go](/workspace/go/internal/timezone/context.go) - タイムゾーン context 操作
- [internal/middleware/timezone.go](/workspace/go/internal/middleware/timezone.go) - タイムゾーン解決ミドルウェア
- [internal/templates/helper.go](/workspace/go/internal/templates/helper.go) - テンプレートヘルパー
- [internal/templates/components/datetime.templ](/workspace/go/internal/templates/components/datetime.templ) - 日時表示コンポーネント
