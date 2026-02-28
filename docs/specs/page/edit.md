# ページ編集 仕様書

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

ページ編集画面（Go版）では、スペースメンバーが既存のページのタイトルと本文を編集できる。エディタ部分はMarkdownエディタ（CodeMirror 6）を使用し、下書きの自動保存に対応している。

編集画面のフッターには、そのページがリンクしている他ページの一覧（リンク一覧）が表示される。各リンク先ページには、そのリンク先にリンクしている他ページの一覧（バックリンク一覧）もネストして表示される。リンク一覧とバックリンク一覧はそれぞれオフセットベースのページネーションに対応しており、SSE（Server-Sent Events）を使用した非同期読み込みで表示される。

**目的**:

- ユーザーがページの内容を編集し、公開できる
- リンク先ページとバックリンクを確認することで、ページ間の関係を把握しながら編集できる

**背景**:

- Rails版からGo版への段階的移行の一環として実装
- リンク一覧・バックリンク一覧のデータ構造はRails版の `LinkList` / `BacklinkList` モデルに合わせて設計し、表示内容の整合性を保っている

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

### ページ編集

- ユーザーはページのタイトルと本文を編集し、「公開」ボタンで保存できる
- 本文の編集はMarkdownエディタ（CodeMirror 6）で行う
- 下書きは自動保存され、編集画面を再度開いたときに下書きの内容が復元される
- 編集権限はトピックポリシー（`TopicPolicy.CanUpdatePage`）で制御される

### リンク一覧

- 編集中のページがリンクしている他ページの一覧がフッターに表示される
- 下書きが存在する場合は下書きの `LinkedPageIDs` を、存在しない場合は公開済みページの `LinkedPageIDs` を使用する
- リンク一覧はオフセットベースのページネーションで表示される（1ページあたり15件）
- ソート順は `modified_at DESC, id DESC`（更新日時が新しい順、同一の場合はID降順）
- 「もっと見る」ボタンで次のページを読み込める
- 下書き自動保存時にリンク一覧が自動更新される（Datastarのカスタムイベント `draft-autosaved` を使用）

### バックリンク一覧

- 各リンク先ページに対して、そのリンク先にリンクしている他ページの一覧がバックリンクとして表示される
- バックリンク一覧はオフセットベースのページネーションで表示される（1ページあたり14件）
- ソート順は `modified_at DESC, id DESC`
- 「もっと見る」ボタンで次のページを読み込める
- バックリンク一覧は各リンク先ページごとに独立してページネーションできる

### 表示条件

- 公開済み（`published_at IS NOT NULL`）かつ未削除（`discarded_at IS NULL`）のページのみ表示される
- 同一スペース内のページのみ表示される（`space_id` による絞り込み）

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

### エンドポイント

| メソッド | パス                                                                                    | ハンドラー                        | 説明                            |
| -------- | --------------------------------------------------------------------------------------- | --------------------------------- | ------------------------------- |
| GET      | `/go/s/{space_identifier}/pages/{page_number}/edit`                                     | `page.Handler.Edit`               | ページ編集フォームの表示        |
| PATCH    | `/go/s/{space_identifier}/pages/{page_number}`                                          | `page.Handler.Update`             | ページの更新                    |
| GET      | `/go/s/{space_identifier}/pages/{page_number}/link_list`                                | `page_link_list.Handler.Show`     | リンク一覧のSSEレスポンス       |
| GET      | `/go/s/{space_identifier}/pages/{page_number}/links/{linked_page_number}/backlink_list` | `page_backlink_list.Handler.Show` | バックリンク一覧のSSEレスポンス |

リンク一覧とバックリンク一覧のエンドポイントは SSE（Server-Sent Events）でレスポンスを返し、Datastar によるフラグメント更新でDOMを部分更新する。

### ビューモデル

#### Pagination

```go
type Pagination struct {
    Current     int   // 現在のページ番号（1始まり）
    Total       int   // 総ページ数
    HasNext     bool  // 次のページが存在するか
    HasPrevious bool  // 前のページが存在するか
}
```

#### LinkList

```go
type LinkList struct {
    Items           []LinkListItem
    Pagination      Pagination
    SpaceIdentifier string
    PageNumber      int32   // 編集中のページ番号
}

type LinkListItem struct {
    Page         Page          // リンク先ページの表示情報
    BacklinkList BacklinkList  // そのリンク先ページへのバックリンク一覧
}
```

#### BacklinkList

```go
type BacklinkList struct {
    Items            []BacklinkListItem
    Pagination       Pagination
    SpaceIdentifier  string
    PageNumber       int32  // 編集中のページ番号
    LinkedPageNumber int32  // リンク先ページ番号
}

type BacklinkListItem struct {
    Page Page  // バックリンク元ページの表示情報
}
```

### ページネーション

| 対象         | 1ページあたりの件数 | ソート順                    |
| ------------ | ------------------- | --------------------------- |
| リンク一覧   | 15件                | `modified_at DESC, id DESC` |
| バックリンク | 14件                | `modified_at DESC, id DESC` |

ページネーションはオフセットベースで、`page` クエリパラメータ（1始まり）でページ番号を指定する。

### データベースクエリ

#### リンク先ページ取得

```sql
SELECT id, space_id, topic_id, number, title, modified_at
FROM pages
WHERE id = ANY(@page_ids::uuid[])
  AND space_id = @space_id
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
ORDER BY modified_at DESC, id DESC
LIMIT @page_limit OFFSET @page_offset;
```

#### バックリンクページ取得

```sql
SELECT id, space_id, topic_id, number, title, modified_at
FROM pages
WHERE @target_page_id::varchar = ANY(linked_page_ids)
  AND space_id = @space_id
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
ORDER BY modified_at DESC, id DESC
LIMIT @page_limit OFFSET @page_offset;
```

#### バックリンク一括取得（N+1最適化）

リンク一覧の初回表示時、全リンク先ページのバックリンクを1回のクエリで取得する。PostgreSQLの `CROSS JOIN LATERAL` を使用し、各リンク先ページに対して上限件数分のバックリンクを効率的に取得する。

### SSEフラグメント更新

| 対象                           | セレクタID                                 | トリガー                                                     |
| ------------------------------ | ------------------------------------------ | ------------------------------------------------------------ |
| リンク一覧                     | `#page-link-list`                          | ページ読み込み時、下書き自動保存時、「もっと見る」クリック時 |
| バックリンク一覧（各リンク先） | `#page-backlink-list-{linked_page_number}` | 「もっと見る」クリック時                                     |

### ファイル構成

```
go/internal/
├── viewmodel/
│   ├── pagination.go       # Pagination構造体、NewPagination関数、定数（LinkLimit, BacklinkLimit）
│   ├── link_list.go        # LinkList, LinkListItem構造体、NewLinkList関数
│   └── backlink_list.go    # BacklinkList, BacklinkListItem構造体、NewBacklinkList関数
├── handler/
│   ├── page/
│   │   ├── handler.go      # Handler構造体
│   │   └── edit.go         # ページ編集フォーム表示
│   ├── page_link_list/
│   │   ├── handler.go      # Handler構造体
│   │   └── show.go         # リンク一覧SSEエンドポイント
│   └── page_backlink_list/
│       ├── handler.go      # Handler構造体
│       └── show.go         # バックリンク一覧SSEエンドポイント
├── repository/
│   ├── page.go             # FindLinkedPagesPaginated, FindBacklinkedPagesPaginated, FindBacklinksForPages
│   └── pagination.go       # PaginatedPages構造体
├── templates/
│   ├── path.go             # GoPageLinkListPath, GoPageBacklinkListPath
│   ├── components/
│   │   ├── link_list.templ      # リンク一覧コンポーネント
│   │   └── backlink_list.templ  # バックリンク一覧コンポーネント
│   └── pages/page/
│       └── edit.templ           # ページ編集画面テンプレート
└── query/queries/
    └── pages.sql           # ページネーション付きクエリ
```

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### カーソルベースのページネーション

Rails版では `activerecord_cursor_paginate` gemを使用したカーソルベースのページネーションを採用しているが、Go版ではオフセットベースを採用した。

**不採用の理由**:

- リンク一覧・バックリンク一覧のデータ量は大量にはならず、オフセットベースでもパフォーマンス上の問題は発生しない
- カーソルベースはエンコード/デコード・初回/2回目以降のクエリ分岐などの実装が複雑になる
- オフセットベースは `?page=2` のようなURLパラメータで任意のページに直接アクセスでき、デバッグや開発時の利便性が高い

### BacklinkListItemのフィールドをTitleとNumberのみにする

`BacklinkListItem` を `Title` と `Number` のみのフラットな構造体にする案を検討した。しかし、バックリンク一覧表示で `Page` ビューモデルの情報がすぐに必要になるため、`BacklinkListItem` は `viewmodel.Page` を内包する構造とした。`LinkListItem` も同様のパターンで統一している。

### リッチなカード表示

Rails版の `CardLinks::PageComponent` にはトピック名表示やカード画像表示機能があるが、Go版のページ編集画面ではタイトルとリンクのみで十分と判断した。ページ表示画面のGo移行時に検討する。

### 「前へ」ページネーション

オフセットベースのページネーションでは「前へ」の実装は容易だが、リンク一覧のUIでは「もっと見る」（次ページのみ）で十分と判断した。「前へ」は必要性が低い。

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Rails版 LinkList モデル](/workspace/rails/app/models/link_list.rb)
- [Rails版 BacklinkListComponent](/workspace/rails/app/components/backlink_list_component.html.erb)
- [Datastar 公式サイト](https://data-star.dev/)
