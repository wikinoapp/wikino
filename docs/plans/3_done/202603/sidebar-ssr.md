# サイドバーのサーバーサイドレンダリング化 作業計画書

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

- 該当する仕様書なし（UI の実装詳細の変更であり、機能仕様の変更ではないため、仕様書の作成・更新は不要）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

サイドバーの「参加中トピック」と「下書きページ」セクションは、現在 Datastar の `data-on-intersect` を使って SSE エンドポイントから非同期で読み込んでいる。MPA のページ遷移のたびにスケルトン → コンテンツの切り替えが発生し、チラつきが生じている。

Rails 版では Turbo の `data-turbo-permanent` で DOM をページ遷移間で保持できていたが、Datastar には同等の機能がない。

この変更では、サイドバーのデータをページの初期レンダリング時にサーバーサイドで取得・描画するように変更し、チラつきを解消する。Wikino は submit 時にページ全体を更新するスタイルを採用しているため、動的更新用の SSE エンドポイントは不要となり削除する。

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

- サイドバーの「参加中トピック」と「下書きページ」がページ読み込み時に即座に表示される
- スケルトンの表示なしで、最初からコンテンツが描画される
- 既存の表示内容・表示順序・件数制限（トピック 10 件、下書き 5 件）は変更なし

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
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

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

### SidebarData 構造体の拡張

`SidebarData` に参加中トピックと下書きページの ViewModel スライスを追加する。

```go
type SidebarData struct {
    DefaultClosed   bool
    CurrentPageName templates.PageName
    SignedIn        bool
    UserAtname      string
    SpaceIdentifier string
    JoinedTopics    []viewmodel.TopicForSidebar     // 追加
    DraftPages      []viewmodel.DraftPageForSidebar  // 追加
}
```

### テンプレートの変更

`sidebar.templ` の `data-on-intersect` + スケルトンを、直接のコンポーネント呼び出しに置き換える。

変更前:

```templ
<div id="sidebar-joined-topics" data-on-intersect="@get('/sidebar/joined_topics')">
    @sidebarJoinedTopicsSkeleton()
</div>
```

変更後:

```templ
@SidebarJoinedTopics(data.JoinedTopics)
```

スケルトンコンポーネント（`sidebarJoinedTopicsSkeleton`, `sidebarDraftPagesSkeleton`）は削除する。

### ハンドラーでのデータ取得

`page.Handler` に `sidebarContent` ヘルパーメソッドを追加し、`edit.go` と `update.go`（`renderEditWithErrors`）で使用する。

```go
func (h *Handler) sidebarContent(ctx context.Context, userID model.UserID) ([]viewmodel.TopicForSidebar, []viewmodel.DraftPageForSidebar) {
    // topicRepo.ListJoinedByUser + draftPageRepo.ListByUser を呼び出し
    // エラー時は空スライスを返す（ページ表示は継続）
}
```

`welcome/show.go` は未ログインユーザー専用のため、サイドバーコンテンツの取得は不要（`SignedIn: false` なのでトピック・下書きセクション自体が描画されない）。

### 変更対象ファイル一覧

| ファイル                                      | 変更内容                                                                       |
| --------------------------------------------- | ------------------------------------------------------------------------------ |
| `internal/templates/components/sidebar.templ` | `SidebarData` にフィールド追加、スケルトンを直接描画に置換、スケルトン関数削除 |
| `internal/handler/page/handler.go`            | `sidebarContent` ヘルパーメソッド追加                                          |
| `internal/handler/page/edit.go`               | `SidebarData` 構築時にサイドバーコンテンツを設定                               |
| `internal/handler/page/update.go`             | `renderEditWithErrors` でサイドバーコンテンツを設定                            |
| `cmd/server/main.go`                          | サイドバー SSE ハンドラーの初期化・ルート登録を削除                            |
| `internal/middleware/reverse_proxy.go`        | `/sidebar` をホワイトリストから削除                                            |

### 削除対象ファイル一覧

| ディレクトリ / ファイル                               | 内容           |
| ----------------------------------------------------- | -------------- |
| `internal/handler/sidebar_joined_topic/handler.go`    | Handler 構造体 |
| `internal/handler/sidebar_joined_topic/index.go`      | SSE ハンドラー |
| `internal/handler/sidebar_joined_topic/index_test.go` | テスト         |
| `internal/handler/sidebar_joined_topic/main_test.go`  | TestMain       |
| `internal/handler/sidebar_draft_page/handler.go`      | Handler 構造体 |
| `internal/handler/sidebar_draft_page/index.go`        | SSE ハンドラー |
| `internal/handler/sidebar_draft_page/index_test.go`   | テスト         |
| `internal/handler/sidebar_draft_page/main_test.go`    | TestMain       |

### 変更しないファイル

| ファイル                                                    | 理由                                                                           |
| ----------------------------------------------------------- | ------------------------------------------------------------------------------ |
| `internal/templates/components/sidebar_joined_topics.templ` | 描画コンポーネントはそのまま再利用                                             |
| `internal/templates/components/sidebar_draft_pages.templ`   | 描画コンポーネントはそのまま再利用                                             |
| `internal/viewmodel/topic_for_sidebar.go`                   | ViewModel はそのまま再利用                                                     |
| `internal/viewmodel/draft_page_for_sidebar.go`              | ViewModel はそのまま再利用                                                     |
| `internal/repository/topic.go`                              | `ListJoinedByUser` はそのまま再利用                                            |
| `internal/repository/draft_page.go`                         | `ListByUser` はそのまま再利用                                                  |
| `internal/handler/welcome/show.go`                          | 未ログインユーザー専用。`SignedIn: false` のため新フィールドのゼロ値で問題なし |

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### クライアントサイドキャッシュ

`localStorage` にサイドバーの HTML をキャッシュし、ページ遷移時に即座に表示する方法を検討した。

**不採用の理由**: JS の実行までにチラつきが残る可能性があること、キャッシュ管理の複雑さ、古いデータの表示リスクがあるため。サーバーサイドレンダリングの方がシンプルで確実。

### サーバーサイドキャッシュ

クエリ結果をユーザーごとにインメモリキャッシュし、トピック参加・下書き作成時にキャッシュを無効化する方法を検討した。

**不採用の理由**: YAGNI 原則。対象のクエリはインデックス付き JOIN + LIMIT で高速に返るため、現時点ではキャッシュ不要。将来パフォーマンスが問題になった場合に検討する。

### SSE エンドポイントの存続

動的更新のために SSE エンドポイントを残す方法を検討した。

**不採用の理由**: Wikino は submit 時にページ全体を更新するスタイルを採用しており、サイドバーの動的更新が必要になる場面がない。不要なエンドポイントとコードの維持コストを避ける。

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

### フェーズ 1: サイドバーの SSR 化と SSE エンドポイント削除

<!--
変更は 1 つの PR に収まるサイズ（実装コード約 30 行追加、約 430 行削除）のため、1 タスクにまとめる
-->

- [x] **1-1**: [Go] サイドバーをサーバーサイドレンダリングに変更し、SSE エンドポイントを削除
  - `SidebarData` にフィールド追加（`sidebar.templ`）
  - テンプレートをスケルトンから直接描画に変更（`sidebar.templ`）
  - `page/handler.go` に `sidebarContent` ヘルパーメソッド追加
  - `page/edit.go`, `page/update.go` でサイドバーコンテンツを設定
  - `cmd/server/main.go` からサイドバー SSE ルートを削除
  - `reverse_proxy.go` からホワイトリストエントリを削除
  - `internal/handler/sidebar_joined_topic/` ディレクトリを削除
  - `internal/handler/sidebar_draft_page/` ディレクトリを削除
  - **想定ファイル数**: 約 14 ファイル（実装 6 変更 + 削除 8）
  - **想定行数**: 約 30 行追加 + 約 430 行削除

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **サーバーサイドキャッシュ**: クエリ結果のキャッシュは、パフォーマンスが問題になってから検討する
- **他のハンドラーへのサイドバー追加**: 現在 `layouts.Default` を使用しているのはページ編集画面とウェルカムページのみ。他のページへのサイドバー追加は別タスク
