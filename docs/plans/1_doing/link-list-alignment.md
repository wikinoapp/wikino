# LinkList のRails版との整合 作業計画書

<!--
このテンプレートの使い方:
1. このファイルを `docs/plans/` ディレクトリにコピー
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

- [ページ編集 仕様書](../specs/page/edit.md)（タスク完了後に更新予定）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

Go版の `internal/viewmodel/link_list.go` が Rails版の `app/models/link_list.rb` と扱うデータに差があるため、Rails版を参考にデータ構造を整合させる。

### 現状のデータ構造比較

**Rails版 `LinkList`**:

```
LinkList
├── links: [Link]
│   ├── page: Page（ページの全情報）
│   └── backlink_list: BacklinkList（リンク先ページへのバックリンク一覧）
│       ├── backlinks: [Backlink]
│       │   └── page: Page
│       └── pagination: Pagination
└── pagination: Pagination（カーソルベースのページネーション）
```

**Go版 `LinkList`（現状）**:

```
LinkList
├── Items: [LinkListItem]
│   ├── Title: string
│   └── Number: int32
└── SpaceIdentifier: string
```

### 主な差分

| 項目               | Rails版                               | Go版（現状）             |
| ------------------ | ------------------------------------- | ------------------------ |
| ページネーション   | カーソルベース（`Pagination` 構造体） | なし                     |
| バックリンク       | 各リンク先に `BacklinkList` がネスト  | なし                     |
| リンク項目のデータ | `Page` モデル全体                     | `Title` と `Number` のみ |

### 関連タスク

- [@docs/plans/1_doing/page-edit-go-migration.md](page-edit-go-migration.md) - 親タスク（ページ編集画面のGo移行）
  - タスク 8b-2（バックリンク一覧表示）が未実装

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

- Go版の `LinkList` ビューモデルが Rails版の `LinkList` モデルと同等のデータを保持する
- リンク一覧にページネーション（カーソルベース）を導入し、大量リンク時も表示できる
- 各リンク先ページにバックリンク一覧（そのリンク先にリンクしている他ページ）を表示する
- バックリンク一覧にもページネーション（カーソルベース）を導入する
- ページ編集画面のフッターでリンク一覧とバックリンク一覧がRails版と同等に表示される

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

- リンク・バックリンクのクエリには必ず `space_id` を条件に含める（セキュリティ）
- カーソルページネーションにより、ページ数が増えてもパフォーマンスが劣化しない

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

### ビューモデル設計

Go版のビューモデルをRails版のモデル構造に合わせて再設計する。

#### Pagination ビューモデル（新規）

```go
// internal/viewmodel/pagination.go

// Pagination はカーソルベースのページネーション情報です
type Pagination struct {
    NextCursor     *string
    HasNext        bool
    PreviousCursor *string
    HasPrevious    bool
}
```

Rails版の `Pagination` モデルと1:1対応。

#### BacklinkList ビューモデル（新規）

```go
// internal/viewmodel/backlink_list.go

// BacklinkListItem はバックリンクの個別ページ情報です
type BacklinkListItem struct {
    Title  string
    Number int32
}

// BacklinkList はバックリンク一覧の表示データです
type BacklinkList struct {
    Items      []BacklinkListItem
    Pagination Pagination
}
```

Rails版の `BacklinkList` モデルに対応。Rails版では各 `Backlink` が `Page` モデル全体を持つが、Go版ではテンプレートで必要な `Title` と `Number` のみ保持する（YAGNI）。表示に必要なフィールドが増えた場合は拡張する。

#### LinkList ビューモデル（変更）

```go
// internal/viewmodel/link_list.go

// LinkListItem はリンク一覧の個別リンク情報です
type LinkListItem struct {
    Title        string
    Number       int32
    BacklinkList BacklinkList
}

// LinkList はリンク一覧の表示データです
type LinkList struct {
    Items           []LinkListItem
    Pagination      Pagination
    SpaceIdentifier string
}
```

変更点:

- `LinkListItem` に `BacklinkList` を追加（Rails版の `Link.backlink_list` に対応）
- `LinkList` に `Pagination` を追加（Rails版の `LinkList.pagination` に対応）

#### コンストラクタ設計

`NewLinkList` の引数を変更し、ページネーション情報とバックリンクデータを受け取れるようにする。

```go
// NewLinkListInput はNewLinkListの入力パラメータです
type NewLinkListInput struct {
    Pages           []*model.Page
    BacklinkMap     map[model.PageID]BacklinkList // リンク先ページIDごとのバックリンク一覧
    Pagination      Pagination
    SpaceIdentifier string
}

// NewLinkList はリンク先ページの一覧からLinkListを生成します
func NewLinkList(input NewLinkListInput) LinkList {
    // ...
}
```

### カーソルページネーション設計

Rails版では `activerecord_cursor_paginate` gemを使用している。Go版では同等のカーソルページネーションをSQLクエリレベルで実装する。

#### ソート順

Rails版に合わせて `modified_at DESC, id DESC` でソートする。

#### カーソルの仕組み

- カーソルは `(modified_at, id)` のタプルをBase64エンコードした文字列
- 「次のページ」: `WHERE (modified_at, id) < (cursor_modified_at, cursor_id)` で取得
- `limit + 1` 件取得し、`limit` 件を超えていれば `has_next = true`

#### SQLクエリ

```sql
-- リンク先ページ一覧（カーソルページネーション付き）
-- name: FindLinkedPagesPaginated :many
SELECT id, space_id, topic_id, number, title, modified_at
FROM pages
WHERE id = ANY(@page_ids::uuid[])
  AND space_id = @space_id
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
  AND (modified_at, id) < (@cursor_modified_at, @cursor_id)
ORDER BY modified_at DESC, id DESC
LIMIT @page_limit;

-- バックリンクページ一覧（カーソルページネーション付き）
-- name: FindBacklinkedPagesPaginated :many
SELECT id, space_id, topic_id, number, title, modified_at
FROM pages
WHERE @target_page_id::varchar = ANY(linked_page_ids)
  AND space_id = @space_id
  AND published_at IS NOT NULL
  AND discarded_at IS NULL
  AND (modified_at, id) < (@cursor_modified_at, @cursor_id)
ORDER BY modified_at DESC, id DESC
LIMIT @page_limit;
```

カーソルなし（初回リクエスト）の場合は `(modified_at, id) <` 条件を省略したクエリを使用する。

#### ページネーションの上限

Rails版に合わせる:

- リンク一覧: 15件/ページ
- バックリンク一覧: 14件/ページ

### テンプレート設計

#### リンク一覧テンプレート（変更）

`internal/templates/components/link_list.templ` を更新し、以下を追加:

- 各リンク項目の下にバックリンク一覧を表示
- 「もっと見る」ボタン（`has_next` が `true` の場合）

#### バックリンク一覧テンプレート（新規）

`internal/templates/components/backlink_list.templ` を新規作成:

- バックリンク項目のカード表示
- 「もっと見る」ボタン（`has_next` が `true` の場合）

### ハンドラー設計

#### 既存ハンドラーの更新

- `internal/handler/page/edit.go`: LinkList構築時にページネーションとバックリンクデータを取得
- `internal/handler/page_link_list/show.go`: SSEレスポンスでページネーション付きLinkListを返す

#### 新規ハンドラー

- `internal/handler/page_backlink_list/handler.go`: バックリンク一覧SSEエンドポイント用Handler
- `internal/handler/page_backlink_list/show.go`: バックリンク一覧のSSEレスポンス

### リポジトリ設計

`internal/repository/page.go` に以下のメソッドを追加:

```go
// FindLinkedPagesPaginated はリンク先ページをカーソルページネーションで取得します
func (r *PageRepository) FindLinkedPagesPaginated(ctx context.Context, pageIDs []model.PageID, spaceID model.SpaceID, cursor *Cursor, limit int) (*PaginatedPages, error)

// FindBacklinkedPagesPaginated はバックリンクページをカーソルページネーションで取得します
func (r *PageRepository) FindBacklinkedPagesPaginated(ctx context.Context, pageID model.PageID, spaceID model.SpaceID, cursor *Cursor, limit int) (*PaginatedPages, error)
```

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### LinkListItemにPageモデル全体を持たせる

Rails版の `Link` は `Page` モデル全体を保持しているが、Go版では `Title` と `Number` のみとした。テンプレートで実際に表示しているのはタイトルとリンク先URLのみであり、YAGNI原則に従って必要最小限のフィールドのみ保持する。将来的にトピック名やアイコンなどの表示が必要になった場合は、その時点でフィールドを追加する。

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

### フェーズ 1: ビューモデルとテンプレートの基盤整備

<!--
Pagination ビューモデルの追加と LinkList の構造変更。テンプレートとハンドラーの更新。
-->

- [ ] **1-1**: [Go] Pagination ビューモデルの追加と LinkList 構造の変更
  - `internal/viewmodel/pagination.go` を新規作成（`Pagination` 構造体）
  - `internal/viewmodel/backlink_list.go` を新規作成（`BacklinkListItem`, `BacklinkList` 構造体、`NewBacklinkList` コンストラクタ）
  - `internal/viewmodel/link_list.go` を更新（`LinkListItem` に `BacklinkList` 追加、`LinkList` に `Pagination` 追加、`NewLinkList` のシグネチャ変更）
  - `internal/viewmodel/link_list_test.go` を更新（新しいシグネチャに合わせてテスト更新）
  - `internal/handler/page/edit.go` を更新（`NewLinkList` 呼び出しを新シグネチャに変更。初期実装ではページネーション・バックリンクは空で渡す）
  - `internal/handler/page_link_list/show.go` を更新（同上）
  - **想定ファイル数**: 約 7 ファイル（実装 5 + テスト 2）
  - **想定行数**: 約 200 行（実装 130 行 + テスト 70 行）

### フェーズ 2: カーソルページネーションの導入

<!--
SQLクエリとリポジトリにカーソルページネーション機能を追加する。
-->

- [ ] **2-1**: [Go] カーソルページネーション基盤とリンク一覧への適用
  - `internal/repository/cursor.go` を新規作成（カーソルのエンコード/デコード、`PaginatedPages` 構造体）
  - `internal/query/queries/pages.sql` にカーソルページネーション付きクエリを追加（初回用・カーソル指定用の2種類 × リンク先・バックリンクの2種類 = 4クエリ）
  - `internal/query/` にsqlc生成コードを更新（`sqlc generate`）
  - `internal/repository/page.go` に `FindLinkedPagesPaginated`, `FindBacklinkedPagesPaginated` メソッドを追加
  - `internal/repository/cursor_test.go` を新規作成（カーソルエンコード/デコードのテスト）
  - `internal/repository/page_test.go` にページネーションメソッドのテストを追加
  - **想定ファイル数**: 約 8 ファイル（実装 4 + 自動生成 2 + テスト 2）
  - **想定行数**: 約 300 行（実装 180 行 + テスト 120 行）

### フェーズ 3: ハンドラーとテンプレートの更新

<!--
ページネーションとバックリンクデータをハンドラーで取得し、テンプレートで表示する。
-->

- [ ] **3-1**: [Go] リンク一覧のページネーション表示とバックリンク取得の実装
  - `internal/handler/page/edit.go` を更新（ページネーション付きリンク先取得、バックリンクデータ取得）
  - `internal/handler/page_link_list/show.go` を更新（ページネーションパラメータ対応）
  - `internal/templates/components/link_list.templ` を更新（バックリンク表示の追加、「もっと見る」ボタンの追加）
  - `internal/templates/components/backlink_list.templ` を新規作成（バックリンク一覧コンポーネント）
  - 翻訳ファイル（`ja.toml`, `en.toml`）にバックリンク・ページネーション関連メッセージを追加
  - templ再生成（`make templ-generate`）
  - **想定ファイル数**: 約 8 ファイル（実装 6 + テスト 2）
  - **想定行数**: 約 280 行（実装 200 行 + テスト 80 行）

- [ ] **3-2**: [Go] バックリンク一覧のSSEエンドポイントの追加
  - `internal/handler/page_backlink_list/handler.go` を新規作成
  - `internal/handler/page_backlink_list/show.go` を新規作成（バックリンク一覧のSSEレスポンス）
  - `internal/templates/path.go` にパスヘルパー `GoPageBacklinkListPath` を追加
  - `internal/templates/pages/page/edit.templ` を更新（バックリンクセクションにDatastarリスナーを追加）
  - `cmd/server/main.go` にルーティング登録
  - `internal/middleware/reverse_proxy.go` にパス追加
  - **想定ファイル数**: 約 8 ファイル（実装 6 + テスト 2）
  - **想定行数**: 約 250 行（実装 170 行 + テスト 80 行）

### フェーズ 4: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [ ] **4-1**: 仕様書の作成・更新
  - `docs/specs/page/edit.md` を更新する
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **リッチなカード表示（トピック名、画像、ピンアイコン）**: Rails版の `CardLinks::PageComponent` にはトピック名表示やカード画像表示機能があるが、Go版のページ編集画面ではタイトルとリンクのみで十分。ページ表示画面のGo移行時に検討する
- **「前へ」ページネーション**: Rails版ではカーソルの「前へ」もサポートしているが、Go版ではリンク一覧の「もっと見る」（次へ）のみ実装する。「前へ」は必要性が低いため

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Rails版 LinkList モデル](/workspace/rails/app/models/link_list.rb)
- [Rails版 LinkListRepository](/workspace/rails/app/repositories/link_list_repository.rb)
- [Rails版 LinkListComponent](/workspace/rails/app/components/link_list_component.html.erb)
- [Rails版 BacklinkListComponent](/workspace/rails/app/components/backlink_list_component.html.erb)
- [ページ編集画面のGo移行 作業計画書](/workspace/docs/plans/1_doing/page-edit-go-migration.md) - タスク 8b-2
