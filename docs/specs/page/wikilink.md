# Wikiリンク 仕様書

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

Wikiリンクは、ページ本文中に `[[ページ名]]` または `[[トピック名/ページ名]]` と記述することで、同じスペース内の他のページへのリンクを作成する機能。下書きの自動保存時とページ公開時の両方でWikiリンクが解析され、リンク先ページの自動作成とHTML変換が行われる。

**目的**:

- ページ間のリンクを簡潔な記法で作成できる
- リンク先ページが存在しない場合に自動作成することで、ページ作成のハードルを下げる

**背景**:

- Wiki システムで一般的なリンク記法を採用し、ユーザーが直感的にページ間の関係を構築できるようにしている
- リンク先ページの自動作成により、「リンクを先に書いてからページを作成する」というワークフローをサポートする

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

### Wikiリンクの形式

| 形式                      | 例                | 意味                             |
| ------------------------- | ----------------- | -------------------------------- |
| `[[ページ名]]`            | `[[設計書]]`      | 現在のトピック内のページにリンク |
| `[[トピック名/ページ名]]` | `[[開発/設計書]]` | 指定トピック内のページにリンク   |

### パース処理

- Markdown本文（body）から正規表現 `\[\[(.*?)\]\]` でWikiリンクを抽出する
- 抽出した文字列を `/` で分割（最大2分割）し:
  - `トピック名/ページ名` 形式の場合: トピック名とページ名をそのまま使用
  - `ページ名` のみの場合: 現在のトピック名をトピック名として使用
- 前後の空白はトリムされ、空のWikiリンクはスキップされる

### 自動ページ作成

パースしたWikiリンクごとに以下の処理を行う:

1. スペース内で指定トピック名のトピックを検索する
2. トピックが存在する場合、そのトピック内で指定タイトルのページを first_or_create 方式で取得・作成する
3. 新規作成されたページは空の `body` / `bodyHtml` と空の `linkedPageIds` を持つ
4. 新規作成されたページの `published_at` は `NULL`（未公開状態）とする
5. 新規作成されたページには `page_editors` レコードも作成する
6. 指定トピックが存在しない場合、そのWikiリンクはスキップする（リンクは作成されない）
7. 全てのリンク先ページIDを収集し、Page / DraftPage の `linkedPageIds` を更新する

### HTML変換

bodyHtml の生成時に、WikiリンクをHTML `<a>` タグに変換する:

1. body からWikiリンクをパースする
2. パースしたキーに対応するページをDBから一括取得する
3. bodyHtml 内の `[[トピック名/ページ名]]` または `[[ページ名]]` を以下のHTMLに置換する:
   - ページが存在する場合: `<a href="/s/{space_identifier}/pages/{page_number}">{ページタイトル}</a>`
   - ページが存在しない場合: `[[...]]` のまま（プレーンテキストとして残す）
4. `<a>`, `<code>`, `<pre>`, `<script>`, `<style>` タグ内のWikiリンクは変換しない

### 処理タイミング

- **自動保存時**: DraftPage の body からWikiリンクを解析し、自動ページ作成と `linkedPageIds` 更新を行う。bodyHtml にもWikiリンクのHTML変換を含める
- **公開時**: Page の body からWikiリンクを解析し、自動ページ作成と `linkedPageIds` 更新を行う。bodyHtml にもWikiリンクのHTML変換を含める

### `published_at` の扱い

- Wikiリンクによる自動ページ作成時の `published_at` は `NULL` とする
- リンク記法の入力中に自動保存が実行されると不完全なページ名でページが作成される可能性があるため、`published_at` に日時を入れると不完全なページが一覧等に表示されてしまう
- 自動作成されたページは、ユーザーが内容を入力して公開するまで未公開状態を維持する

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

### ファイル構成

```
go/internal/
├── markup/
│   └── wikilink.go        # パース（ScanWikilinks）、HTML変換（ReplaceWikilinks）
├── usecase/
│   ├── auto_save_draft_page.go  # 自動保存時のWikiリンク処理
│   └── publish_page.go         # 公開時のWikiリンク処理
└── repository/
    ├── page.go            # FindByTopicAndTitle, CreateLinkedPage
    └── topic.go           # FindBySpaceAndNames
```

### 主要な構造体

```go
// WikilinkKey はパースしたWikiリンクのキー
type WikilinkKey struct {
    Raw       string  // 元のテキスト（例: "開発/設計書"）
    TopicName string  // トピック名
    PageTitle string  // ページタイトル
}

// PageLocation はWikiリンクキーに対応するページ情報
type PageLocation struct {
    Key        WikilinkKey
    TopicName  string
    PageID     model.PageID
    PageNumber int32
    PageTitle  string
}
```

### 主要な関数

- `ScanWikilinks(body, currentTopicName)` — Markdown本文からWikiリンクをパースし、`WikilinkKey` のリストを返す
- `ReplaceWikilinks(bodyHTML, currentTopicName, spaceIdentifier, pageLocations)` — bodyHTML内のWikiリンクをHTML `<a>` タグに変換する。DOMツリー走査により `<a>`, `<code>`, `<pre>`, `<script>`, `<style>` タグ内のWikiリンクは変換しない

### 自動ページ作成の処理フロー

```
[ScanWikilinks] Markdown本文からWikiリンクを抽出
    ↓
[FindBySpaceAndNames] スペース内でトピック名を一括検索
    ↓
[findOrCreateLinkedPage] 各Wikiリンクに対応するページをfirst_or_create方式で取得・作成
    ↓ (ページ番号重複時は最大3回リトライ)
[PageEditor作成] 新規作成されたページにpage_editorsレコードを作成
    ↓
[linkedPageIds更新] 全リンク先ページIDをPage/DraftPageに反映
    ↓
[ReplaceWikilinks] bodyHTML内のWikiリンクを<a>タグに変換
```

### Markdownレンダリングパイプラインにおける位置

Wikiリンク変換は以下のレンダリングパイプラインの中で実行される:

1. Markdown → HTML変換（goldmark）
2. HTMLサニタイズ（bluemonday）
3. **Wikiリンク変換**（`ReplaceWikilinks`）
4. 添付ファイルフィルター
5. 後処理（単独画像リンクのラッピング）

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### 自動作成されたページの `published_at` に現在日時を設定する

Wikiリンクによる自動ページ作成時に `published_at` を現在日時に設定する方式を検討した。

**不採用の理由**:

- リンク記法の入力中に自動保存が実行されると、不完全なページ名でページが作成される可能性がある
- `published_at` に日時を入れると不完全なページが一覧等に表示されてしまう
- 自動作成されたページは、ユーザーが内容を入力して公開するまで未公開状態を維持するべきである

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Rails版 Pageable#link! メソッド](/workspace/rails/app/records/concerns/pageable.rb)
- [Rails版 Markup::PageLinkFilter](/workspace/rails/app/models/markup/page_link_filter.rb)
