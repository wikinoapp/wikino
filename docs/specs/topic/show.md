# トピック詳細画面 仕様書

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

トピック詳細画面（`GET /s/:space_identifier/topics/:topic_number`）は、トピック内のページを一覧表示する画面である。ピン留めページと通常ページをグリッド表示し、オフセットベースのページネーションを提供する。Go 版で実装されており、全ユーザーが Go 版を使用する。

**目的**:

- ユーザーがトピック内のページ一覧を閲覧できる
- 編集提案機能で「ページ」と「編集提案」のタブを追加する前提として、Go 版でトピック詳細画面を先に実装しておく

**背景**:

- Rails から Go への段階的移行の一環として実装した
- Go 版で Domain/Infrastructure 層（Model、Repository、Query）は既に実装済みだったため、Presentation 層（Handler、ViewModel、Template）を追加する形で対応した
- Rails 版のトピック詳細画面専用コード（コントローラー・ビュー・ヘッダーコンポーネント・リクエストテスト）は削除済み。ルート定義（`config/routes.rb` の `topic` named route）は `topic_path` が他のコントローラー・コンポーネントで使用されているため維持している

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

### ページ一覧表示

- ユーザーはトピック詳細画面（`/s/:space_identifier/topics/:topic_number`）にアクセスし、トピック内のページ一覧を表示できる
- ピン留めされたページは画面上部にグリッド表示される（`pinned_at DESC` でソート、最大 100 件）
- 通常ページはピン留めページの下にグリッド表示され、`modified_at DESC, id DESC` でソートされる
- 通常ページが 100 件を超える場合はオフセットベースのページネーションで移動できる
- グリッドは 3 カラム（`md:grid-cols-3`）で表示される
- ページが 1 件も存在しない場合は空状態コンポーネントが表示される

### ページカード

- 各ページカードにはタイトルが表示される（未設定の場合は「無題」）
- アイキャッチ画像が存在する場合はカード内に表示される
- ピン留めページにはピン留めアイコン（琥珀色のプッシュピン）がカード右上に表示される
- トピックメンバーには編集ボタン（ペンシルアイコン）が表示される

### 権限

- 公開トピックは全員が閲覧できる
- 非公開トピックはトピックメンバーまたはスペースオーナーのみが閲覧できる。アクセス権のないユーザーがアクセスした場合は 404 を返す
- トピックメンバーまたはスペースオーナーは「新規ページ」ボタンからページ作成画面に遷移できる
- トピック管理者（Admin ロール）またはスペースオーナーはドロップダウンメニューから「トピック設定」画面へのリンクにアクセスできる

### ナビゲーション

- パンくずリスト: ホーム → スペース名 → トピック名
- モバイルではボトムナビゲーションが表示される（メニュー、ホーム、検索/ログインボタン）
- サイドバーにはユーザーが参加しているトピックと下書きページが表示される

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

### ルーティング

リバースプロキシの `goHandledRegexPatterns` にパターン `^/s/[^/]+/topics/\d+$` が登録されており、全ユーザーが Go 版を使用する。

### エンドポイント

| メソッド | パス                                          | ハンドラー           | 説明             |
| -------- | --------------------------------------------- | -------------------- | ---------------- |
| GET      | `/s/{space_identifier}/topics/{topic_number}` | `topic.Handler.Show` | トピック詳細画面 |

### 処理フロー

1. URL パラメータから `space_identifier` と `topic_number` を取得
2. `SpaceRepository` でスペースを取得
3. ログインユーザーの `SpaceMember` を取得（未ログインなら nil）
4. `TopicRepository` でトピックを取得
5. 権限チェック: トピックが非公開の場合、トピックメンバーまたはスペースオーナーでなければ 404
6. ピン留めページを取得（`pinned_at IS NOT NULL`、`pinned_at DESC` でソート、最大 100 件）
7. 通常ページをオフセットベースページネーションで取得（`pinned_at IS NULL`、`modified_at DESC, id DESC`、100 件/ページ）
8. ViewModel に変換してテンプレートをレンダリング

### ビューモデル

#### TopicForShow

```go
type TopicForShow struct {
    Name          string
    Number        int32
    Description   string
    IconName      IconName   // 公開: "globe-regular"、非公開: "lock-regular"
    CanUpdate     bool
    CanCreatePage bool
}
```

#### Pagination

```go
type Pagination struct {
    Current     int   // 現在のページ番号（1始まり）
    Total       int   // 総ページ数
    HasNext     bool  // 次のページが存在するか
    HasPrevious bool  // 前のページが存在するか
}
```

ページカード用には既存の `viewmodel.CardLinkPage` を使用している。

### データベースクエリ

#### ピン留めページ取得

```sql
SELECT ...
FROM pages
WHERE topic_id = @topic_id
  AND space_id = @space_id
  AND pinned_at IS NOT NULL
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
  AND trashed_at IS NULL
ORDER BY pinned_at DESC
LIMIT @page_limit;
```

#### 通常ページ取得（オフセットベースページネーション）

```sql
SELECT ...
FROM pages
WHERE topic_id = @topic_id
  AND space_id = @space_id
  AND pinned_at IS NULL
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
  AND trashed_at IS NULL
ORDER BY modified_at DESC, id DESC
LIMIT @page_limit OFFSET @page_offset;
```

### ファイル構成

```
go/internal/
├── handler/topic/
│   ├── handler.go           # Handler構造体と依存性
│   ├── show.go              # Show ハンドラー（権限チェック含む）
│   ├── main_test.go         # テストセットアップ
│   └── show_test.go         # ハンドラーの統合テスト
├── viewmodel/
│   ├── topic.go             # TopicForShow構造体
│   ├── pagination.go        # Pagination構造体
│   └── card_link_page.go    # CardLinkPage構造体（アイキャッチ画像対応）
├── repository/
│   ├── page.go              # FindPinnedByTopic, FindRegularByTopicPaginated
│   └── pagination.go        # PaginatedPages構造体
├── templates/
│   ├── components/
│   │   ├── bottom_nav.templ     # モバイル用ボトムナビ
│   │   ├── card_link_page.templ # ページカードコンポーネント
│   │   └── pagination.templ     # ページネーションコンポーネント
│   └── pages/topic/
│       └── show.templ           # トピック詳細画面テンプレート
├── query/queries/
│   └── pages.sql                # ピン留め・ページネーション取得クエリ
└── i18n/locales/
    ├── ja.toml                  # 翻訳
    └── en.toml                  # 翻訳
```

### i18n キー

| キー                              | 日本語                             | 用途                       |
| --------------------------------- | ---------------------------------- | -------------------------- |
| `topic_show_title`                | `{{.TopicName}} \| {{.SpaceName}}` | ページタイトル             |
| `topic_show_no_pages_message`     | ページはありません                 | 空状態メッセージ           |
| `topic_show_no_pages_description` | 最初の1ページ目を作成しましょう    | 空状態説明文               |
| `topic_show_new_page`             | 新規ページ                         | 新規ページボタン           |
| `topic_show_settings`             | トピック設定                       | 設定リンク                 |
| `pagination_previous`             | 前へ                               | ページネーション前ボタン   |
| `pagination_next`                 | 次へ                               | ページネーション次ボタン   |
| `bottom_nav_menu`                 | メニュー                           | ボトムナビのメニューボタン |
| `bottom_nav_home`                 | ホーム                             | ボトムナビのホームボタン   |
| `bottom_nav_search`               | 検索                               | ボトムナビの検索ボタン     |
| `bottom_nav_sign_in`              | ログイン                           | ボトムナビのログインボタン |

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### カーソルベースページネーション

Rails 版ではカーソルベースページネーション（`cursor_paginate` gem、`(modified_at, id)` の組み合わせを Base64 エンコード）を使用していた。Go 版ではオフセットベースページネーションを採用した。

**不採用の理由**:

- トピック内のページ一覧は頻繁にデータが挿入・削除される性質ではなく、オフセットベースでも実用上の問題が少ない
- オフセットベースの方が実装がシンプルで、ページ番号による直感的なナビゲーションが可能
- 将来的にカーソルベースが必要になった場合は、ページネーションの実装を差し替えれば対応できる

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [トピック詳細画面のGo移行 作業計画書](/workspace/docs/plans/1_doing/topic-show-go-migration.md)
