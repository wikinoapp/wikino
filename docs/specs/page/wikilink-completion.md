# Wikiリンク補完 仕様書

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

Wikiリンク補完は、ページ編集画面のCodeMirrorエディタで `[[` を入力するとスペース内の既存ページ名を候補として表示し、選択すると `[[トピック名/ページタイトル]]` 形式のWikiリンクが自動補完される機能。

バックエンドのページロケーション検索APIがキーワードに一致するページ候補を返し、フロントエンドのCodeMirror補完拡張が候補を表示・挿入する。

**目的**:

- Wikiリンクの記述時にページ名を正確に入力できる
- スペース内の既存ページを素早く検索・リンクできる

**背景**:

- Wikiリンクはトピック名とページタイトルの正確な入力が必要なため、補完機能でユーザーの入力負担を軽減する
- サーバーサイドでフィルタリングすることで、スペース内のページ数が多くても効率的に候補を絞り込める

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

### 補完の動作

- ユーザーがエディタで `[[` を入力すると補完候補が表示される
- `[[` 以降のテキストがキーワードとして使用される（`[[` 自体は除去される）
- `[[トピック名/` の形式で入力した場合、`/` 以降のテキストがキーワードとして使用される
- 候補を選択すると `[[トピック名/ページタイトル` が挿入される（閉じ括弧 `]]` はユーザーが入力する）
- 補完候補の表示は `displayLabel` で `トピック名/ページタイトル` 形式を使用する

### 検索条件

- ページタイトルの部分一致（ILIKE）で候補がフィルタリングされる
- キーワードをスペース区切りで分割し、各ワードに対してAND条件で検索する
- 公開済み（`published_at IS NOT NULL`）かつ未廃棄（`discarded_at IS NULL`）かつ未ゴミ箱（`trashed_at IS NULL`）のページのみ対象
- タイトルが `NULL` のページは対象外
- `modified_at` 降順でソートし、最大10件を返す
- サーバーサイドでフィルタリング済みの結果をそのまま表示する（フロントエンド側のフィルタリングは無効化）

### 認証・認可

- ログイン必須
- スペースメンバーであること（スペースメンバーは全アクティブページを検索可能）

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

### API設計

**エンドポイント**: `GET /s/{space_identifier}/page_locations?q={keyword}`

**レスポンス形式**:

```json
{
  "page_locations": [{ "key": "トピック名/ページタイトル" }]
}
```

### データベースクエリ

```sql
SELECT p.title, t.name AS topic_name
FROM pages p
INNER JOIN topics t ON p.topic_id = t.id AND t.discarded_at IS NULL
WHERE p.space_id = @space_id
  AND p.discarded_at IS NULL
  AND p.trashed_at IS NULL
  AND p.published_at IS NOT NULL
  AND p.title IS NOT NULL
  AND p.title ILIKE ALL(@keywords::text[])
ORDER BY p.modified_at DESC
LIMIT 10;
```

### ファイル構成

```
go/internal/handler/
└── page_location/
    ├── handler.go      # Handler構造体と依存性
    └── index.go        # GET /s/:space_identifier/page_locations?q=:keyword

go/web/markdown-editor/
└── wikilink-completions.ts  # CodeMirror補完拡張
```

### フロントエンド

- `wikilink-completions.ts` — CodeMirrorの `autocompletion` 拡張の override として登録
- `[[` の入力を正規表現 `/\[\[.*/` で検出し、`[[` 以降のテキストをキーワードとして補完候補を取得
- `fetch` でページロケーション検索APIを呼び出し、結果をCodeMirrorの補完候補に変換
- `filter: false` を設定し、サーバーサイドでフィルタリング済みの結果をそのまま表示
- 補完候補の `label`: `[[トピック名/ページタイトル`（挿入用）
- 補完候補の `displayLabel`: `トピック名/ページタイトル`（表示用）

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Rails版 PageLocations::IndexController](/workspace/rails/app/controllers/page_locations/index_controller.rb)
- [Rails版 wikilink-completions.ts](/workspace/rails/app/javascript/controllers/wikilink-completions.ts)
- [CodeMirror Autocompletion](https://codemirror.net/docs/ref/#autocomplete)
