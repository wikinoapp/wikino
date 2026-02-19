# Topic ViewModel 構造体の導入

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
-->

## 仕様書

<!--
- 既存機能を変更する場合: 変更対象の仕様書へのリンクを記載してください
- 新しい機能の場合: タスク完了後に作成予定の仕様書のパスを記載してください
-->

- 内部リファクタリングのため、仕様書の作成・更新は不要

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

現在 `viewmodel/topic.go` には `TopicVisibilityIconName()` というスタンドアロン関数のみが定義されている。一方、同じパッケージ内の `SpaceHeader` はモデルから変換する構造体として定義されており、パターンに一貫性がない。

また、`handler/page/edit.go` ではトピック関連のデータ（名前、パス、アイコン名）をハンドラー内で個別に取り出し、`EditPageData` に1つずつ渡している。これをトピックの ViewModel 構造体に集約することで、ハンドラーがシンプルになり、テンプレート側でもどのデータが使えるか明確になる。

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

- `viewmodel.Topic` 構造体を定義し、テンプレートで必要なフィールド（Name, Number, IconName）を持たせる
- `model.Topic` から `viewmodel.Topic` への変換コンストラクタを提供する
- ハンドラーではモデルから ViewModel への変換を行い、`EditPageData` に ViewModel を渡す
- テンプレートは ViewModel のフィールドを参照し、パスは既存の `templates.TopicPath()` で生成する
- 既存の動作（HTML 出力）に変更はない

## 実装ガイドラインの参照

<!--
**重要**: 設計を行う前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
設計の段階でガイドラインに準拠していることを確認してください。
-->

### Go版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン（**ファイル名は標準の8種類のみ**）
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

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

### 設計方針

ViewModel にはパスを持たせず、テンプレートで必要なデータ（Name, Number, IconName）のみ保持する。パス生成はテンプレート層の関心として、既存の `templates.TopicPath()` をテンプレート内で呼び出す。

この方針により、以下を実現する:

- ViewModel が `templates` パッケージに依存しないため、循環依存が発生しない
- トピックに関するパスが今後増えても（一覧パス等）、ViewModel の変更が不要
- パス生成ロジックが `templates/path.go` に集約される

### 現状

#### viewmodel/topic.go

```go
func TopicVisibilityIconName(v model.TopicVisibility) IconName {
    if v == model.TopicVisibilityPublic {
        return "globe-regular"
    }
    return "lock-regular"
}
```

スタンドアロン関数のみ。`SpaceHeader` のような構造体パターンとの一貫性がない。

#### handler/page/edit.go

```go
topicName := topic.Name
topicPath := templates.TopicPath(space.Identifier, topic.Number)
topicIconName := viewmodel.TopicVisibilityIconName(topic.Visibility)

content := pagepages.Edit(pagepages.EditPageData{
    // ...
    TopicName:       topicName,
    TopicPath:       topicPath,
    TopicIconName:   topicIconName,
})
```

トピック関連のフィールドを個別に取り出して渡している。

#### templates/pages/page/edit.templ

```go
type EditPageData struct {
    // ...
    TopicName       string
    TopicPath       templates.Path
    TopicIconName   viewmodel.IconName
}
```

トピック関連の3フィールドが独立して定義されている。

### 変更後の設計

#### viewmodel/topic.go

`viewmodel.Topic` 構造体と変換コンストラクタを定義する。パスは持たず、テンプレート側で生成する。

```go
package viewmodel

import (
    "github.com/wikinoapp/wikino/go/internal/model"
)

// Topic はテンプレートで表示するトピック情報です
type Topic struct {
    Name     string
    Number   int32
    IconName IconName
}

// NewTopic はモデルからTopicを生成します
func NewTopic(topic *model.Topic) Topic {
    return Topic{
        Name:     topic.Name,
        Number:   topic.Number,
        IconName: topicVisibilityIconName(topic.Visibility),
    }
}

// topicVisibilityIconName はトピックの公開範囲に対応するアイコン名を返します
func topicVisibilityIconName(v model.TopicVisibility) IconName {
    if v == model.TopicVisibilityPublic {
        return "globe-regular"
    }
    return "lock-regular"
}
```

**設計判断**:

- `TopicVisibilityIconName` を非公開関数 `topicVisibilityIconName` に変更する。外部から呼ぶ必要がなくなるため
- パスは ViewModel に含めない。テンプレート側で `templates.TopicPath()` を使って生成する
- コンストラクタは `model.Topic` のみを引数に取る。`spaceIdentifier` はパス生成に必要だがパスを持たないため不要
- コンストラクタ名は `NewTopic` とする。`SpaceHeader` の `NewSpaceHeader` と同様のパターン

#### handler/page/edit.go

```go
topicVM := viewmodel.NewTopic(topic)

content := pagepages.Edit(pagepages.EditPageData{
    // ...
    Topic: topicVM,
})
```

`templates.TopicPath()` の呼び出しがハンドラーから消え、テンプレート側に移動する。ハンドラーの import から `templates` パッケージの参照が減る（`TopicPath` 用の参照が不要に）。

#### templates/pages/page/edit.templ

```go
type EditPageData struct {
    CSRFToken       string
    Title           string
    Body            string
    AutofocusTitle  bool
    PageNumber      int32
    SpaceIdentifier string
    SpaceName       string
    Topic           viewmodel.Topic
}
```

パンくずリスト構築時に `templates.TopicPath()` でパスを生成する:

```templ
@components.TopNav(components.TopNavData{
    Items: []components.BreadcrumbItem{
        {
            Label: templates.T(ctx, "breadcrumb_home"),
            Path:  templates.HomePath(),
        },
        {
            Label: data.SpaceName,
            Path:  templates.SpacePath(data.SpaceIdentifier),
        },
        {
            Label:    data.Topic.Name,
            Path:     templates.TopicPath(data.SpaceIdentifier, data.Topic.Number),
            IconName: data.Topic.IconName,
        },
    },
})
```

### 循環依存の回避

`templates` パッケージは `viewmodel` パッケージを import している（`helper.go` の `Icon` 関数が `viewmodel.IconName` を使用）。そのため、`viewmodel` から `templates` を import すると循環依存になる。

```
templates/helper.go → viewmodel (IconName を使用)
viewmodel/topic.go → templates (Path, TopicPath を使用) ← 循環!
```

**解決策**: ViewModel にはパスを持たせず、パス生成に必要なデータ（`Number`）のみ保持する。パスの生成はテンプレート側で `templates.TopicPath()` を呼び出して行う。これにより `viewmodel` は `templates` に依存せず、循環依存を回避できる。

### 変更対象ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `internal/viewmodel/topic.go` | `Topic` 構造体、`NewTopic` コンストラクタを追加。`TopicVisibilityIconName` を非公開化 |
| `internal/viewmodel/topic_test.go` | `NewTopic` のテストを追加 |
| `internal/handler/page/edit.go` | `viewmodel.NewTopic` を使用するように変更 |
| `internal/templates/pages/page/edit.templ` | `EditPageData` の Topic 関連フィールドを `viewmodel.Topic` に統合。パンくずリストで `templates.TopicPath()` を直接呼び出し |

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### ViewModel にパスを含める（`string` 型で `fmt.Sprintf` 生成）

ViewModel の `Path` フィールドに `string` 型でパスを保持し、`fmt.Sprintf` で直接パスを生成する案。循環依存は回避できるが、以下の理由で採用しない:

- トピックに関するパスは詳細ページ以外にも増える可能性がある（一覧パス等）。すべてのパスを ViewModel で管理するのは煩雑
- パス生成ロジックが `viewmodel` と `templates/path.go` に分散し、URL パターンの変更時に修正箇所が増える
- テンプレート側で `templates.Path()` にキャストする手間が発生する

### `internal/urlpath/` パッケージを新設して Path 型を移動する

`Path` 型とパス生成関数を `templates` から独立した `internal/urlpath/` パッケージに移動する案。`viewmodel` と `templates` の両方から import でき、循環依存を根本的に解消できる。しかし、以下の理由で今回は採用しない:

- `templates.Path` を使っている全箇所（`top_nav.templ`、`edit.templ` 等）の import 変更が必要で影響範囲が大きい
- 今回のリファクタリングのスコープを超える
- 必要性が明確になった時点で別タスクとして実施できる

### TopicVisibilityIconName を公開関数のまま残す

`NewTopic` を経由せずに直接アイコン名だけ取得したいケースが将来的にあるかもしれないが、YAGNI 原則に従い、現時点で不要なインターフェースは公開しない。必要になった時点で公開すれば良い。

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

### フェーズ 1: Topic ViewModel の導入

<!--
変更が小規模なため、1つの PR にまとめる
-->

- [x] **1-1**: [Go] viewmodel.Topic 構造体の導入とハンドラー・テンプレートの更新

  - `viewmodel/topic.go` に `Topic` 構造体と `NewTopic` コンストラクタを追加
  - `TopicVisibilityIconName` を非公開関数 `topicVisibilityIconName` に変更
  - `viewmodel/topic_test.go` に `NewTopic` のテストを追加
  - `handler/page/edit.go` で `viewmodel.NewTopic` を使用するように変更
  - `templates/pages/page/edit.templ` の `EditPageData` を更新し、Topic 関連フィールドを `viewmodel.Topic` に統合。パンくずリストで `templates.TopicPath()` を直接呼び出し
  - templ 再生成、フォーマット、リント、ビルド、テストの実行
  - **想定ファイル数**: 約 5 ファイル（実装 3 + テスト 1 + 自動生成 1）
  - **想定行数**: 約 80 行（実装 50 行 + テスト 30 行）

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **Page ViewModel 構造体の導入**: ページ編集画面の Title/Body フォールバックロジック（下書き → 公開版）の ViewModel 化は、Topic ViewModel の導入とは独立した変更であり、別途検討する
- **`internal/urlpath/` パッケージの新設**: Path 型の独立パッケージ化は影響範囲が大きいため、必要性が明確になった時点で別タスクとして実施する

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - ビューモデルの設計パターン（`NewWorkFromXXX` 等）
- `viewmodel/space.go` の `SpaceHeader` - 既存の ViewModel 構造体の実装例
