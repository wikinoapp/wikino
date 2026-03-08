# ページの移動 作業計画書

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
-->

## 仕様書

<!--
- 既存機能を変更する場合: 変更対象の仕様書へのリンクを記載してください
- 新しい機能の場合: タスク完了後に作成予定の仕様書のパスを記載してください
-->

- [ページの移動 仕様書](../specs/page/move.md)（タスク完了後に作成予定）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

ページの移動機能は、ページが所属するトピックを変更する機能である。ページの内容（リビジョン履歴を含む）はそのまま維持し、所属先のトピックのみを変更する。

編集提案機能の導入に伴い、「内容の変更（編集）」と「構造の変更（移動）」を明確に分離する設計を採用した。これにより、編集はページの内容変更のみを扱い、トピックの変更は独立した操作として提供する。

**目的**:

- ページの内容編集とトピック変更を分離し、それぞれの操作をシンプルに保つ
- リビジョン履歴やリンクを維持したままトピックを変更できるようにする
- 編集提案機能との整合性を確保する（編集提案は内容変更のみを扱うため）

**背景**:

- 従来はページ編集時にトピックも変更可能だったが、編集提案機能の導入により、トピック変更を含む編集提案の権限管理が複雑化する問題が生じた
- トピックごとにユーザーの権限が異なるため、トピックを横断する操作は権限チェックが複雑になる
- 編集（内容変更）と移動（トピック変更）を分離することで、各操作の権限モデルが明確になり、編集提案との整合性も自然に取れる

## 要件

<!--
ガイドライン:
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な要件（セキュリティ、パフォーマンスなど）も記述
-->

### 基本要件

- ユーザーはページを同一スペース内の別のトピックに移動できる
- 移動時にページの内容（リビジョン履歴を含む）は変更されない
- ページの移動は編集操作とは独立した操作であり、ページ編集中にトピックを変更することはできない

### 権限

- ページの移動には、移動先トピックへのページ作成権限が必要である
- 権限のないトピックへの移動はシステムが拒否する

### バリデーション

- 移動先のトピックは現在のトピックと異なる必要がある
- 移動先のトピックは同一スペース内に存在する必要がある
- 移動先のトピックに同名のページが存在しないこと（ページタイトルはトピック内で一意制約がある）

### 関連タスク・仕様

- [@docs/plans/1_doing/edit-suggestion.md](/workspace/docs/plans/1_doing/edit-suggestion.md) - 編集提案機能（ページモデルの詳細もこちらに記載）

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
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

### Rails版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - 全体的なコーディング規約

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
-->

### URL 設計

| HTTP メソッド | URL                                          | Handler          | 説明               |
| ------------- | -------------------------------------------- | ---------------- | ------------------ |
| GET           | /s/:space_identifier/pages/:page_number/move | page_move.New    | ページ移動フォーム |
| POST          | /s/:space_identifier/pages/:page_number/move | page_move.Create | ページ移動実行     |

### 権限設計

ページ移動には2段階の権限チェックが必要:

1. **移動元の権限**: 移動元トピックの `CanUpdatePage` が真であること
   - ページ移動画面へのアクセス制御に使用
2. **移動先の権限**: 移動先トピックの `CanCreatePage` が真であること
   - フォームのセレクトボックスに表示するトピックの絞り込みに使用
   - 移動実行時のサーバーサイドチェックにも使用

移動先トピックのリスト:

- **スペースオーナー**: スペース内の全アクティブトピック（`TopicRepository.ListActiveBySpace`）
- **トピックメンバー**: 所属トピックのみ（`TopicRepository.ListJoinedBySpaceMember`）
- いずれの場合も**現在のトピックを除外**して表示

### バリデーション設計

形式バリデーション:

- 移動先トピックが選択されていること（必須）

状態バリデーション:

- 移動先トピックが同一スペース内に存在すること
- 移動先トピックが現在のトピックと異なること
- 移動先トピックにページ作成権限があること（`CanCreatePage`）
- 移動先トピックに同名のページが存在しないこと（タイトルの一意制約）

### コード設計

#### Go 版

**Handler** (`go/internal/handler/page_move/`):

| ファイル     | 説明                                         |
| ------------ | -------------------------------------------- |
| handler.go   | Handler 構造体と依存性の定義                 |
| new.go       | New メソッド（移動フォーム表示）             |
| create.go    | Create メソッド（移動実行）                  |
| validator.go | CreateValidator（形式 + 状態バリデーション） |

Handler 構造体の依存性:

- `cfg *config.Config`
- `flashMgr *session.FlashManager`
- `spaceRepo *repository.SpaceRepository`
- `spaceMemberRepo *repository.SpaceMemberRepository`
- `pageRepo *repository.PageRepository`
- `topicRepo *repository.TopicRepository`
- `topicMemberRepo *repository.TopicMemberRepository`
- `movePageUC *usecase.MovePageUsecase`

**Policy** (`go/internal/policy/topic.go`):

`TopicPolicy` インターフェースに `CanCreatePage(topic *model.Topic) bool` を追加する。各ポリシーの実装:

| ポリシー          | CanCreatePage の判定                             |
| ----------------- | ------------------------------------------------ |
| topicOwnerPolicy  | `spaceMemberActive && spaceID == topic.Space.ID` |
| topicAdminPolicy  | `spaceMemberActive && topicID == topic.ID`       |
| topicMemberPolicy | `spaceMemberActive && topicID == topic.ID`       |
| topicGuestPolicy  | `false`（常に不可）                              |

**Repository** (`go/internal/repository/page.go`):

`PageRepository` に `MoveTopic` メソッドを追加。`topic_id` と `updated_at` のみを更新する専用 SQL クエリを作成する。

```sql
-- name: MovePageToTopic :one
UPDATE pages
SET topic_id = $2, updated_at = NOW()
WHERE id = $1 AND space_id = $3
RETURNING *;
```

**UseCase** (`go/internal/usecase/move_page.go`):

```go
type MovePageUsecase struct {
    db       *sql.DB
    pageRepo *repository.PageRepository
}

type MovePageInput struct {
    PageID      model.PageID
    SpaceID     model.SpaceID
    DestTopicID model.TopicID
}

type MovePageOutput struct {
    Page *model.Page
}
```

処理内容:

1. トランザクション開始
2. `pageRepo.MoveTopic` で `topic_id` を更新
3. コミット

**Template** (`go/internal/templates/pages/page_move/new.templ`):

移動フォームのテンプレート。構造体ベースのデータ受け渡しパターンを使用。

```go
type MovePageData struct {
    CSRFToken       string
    FormErrors      *session.FormErrors
    Page            viewmodel.Page
    Space           viewmodel.Space
    CurrentTopic    viewmodel.Topic
    AvailableTopics []viewmodel.TopicForSelect
}
```

**ViewModel** (`go/internal/viewmodel/topic.go`):

セレクトボックス用の ViewModel を追加:

```go
type TopicForSelect struct {
    Name   string
    Number int32
}
```

**Routing**:

- `cmd/server/main.go` に GET/POST `/s/{space_identifier}/pages/{page_number}/move` を登録
- `internal/middleware/reverse_proxy.go` の `featureFlaggedPatterns` に `^/s/[^/]+/pages/\d+/move$` パターンを追加（`FeatureFlagGoPageEdit` を再利用）

**I18n** (`go/internal/i18n/locales/`):

ja.toml / en.toml に以下を追加:

- `page_move_title`: ページタイトル
- `page_move_heading`: 見出し
- `page_move_current_topic_label`: 現在のトピック
- `page_move_destination_topic_label`: 移動先トピック
- `page_move_submit`: 移動ボタン
- `page_move_success`: 成功メッセージ
- `page_move_error_same_topic`: 同一トピックエラー
- `page_move_error_no_permission`: 権限エラー
- `page_move_error_title_exists`: タイトル重複エラー
- `page_move_select_topic_placeholder`: プレースホルダー

#### Rails 版

`Dropdowns::PageActionsComponent` に「移動」リンクを追加する。`page.can_update?` が真の場合にのみ表示する。

リンク先は `/s/:space_identifier/pages/:page_number/move` で、Go 版のページ移動画面に遷移する。リバースプロキシの `FeatureFlagGoPageEdit` フラグが有効なユーザーのみ Go 版が表示される。

### UI 設計

ページ移動画面の構成:

1. パンくずリスト（スペース > トピック）
2. 見出し「ページの移動」
3. ページタイトルの表示（読み取り専用）
4. 現在のトピック名の表示（読み取り専用）
5. 移動先トピックのセレクトボックス（現在のトピックを除外、権限のあるトピックのみ）
6. 「移動する」ボタン

成功時: フラッシュメッセージ「ページを移動しました」を表示し、ページ詳細画面にリダイレクト

移動先トピックが存在しない場合（ユーザーが1つのトピックにしか所属していない場合）: セレクトボックスが空になり、ボタンは無効化。「移動先のトピックがありません」というメッセージを表示。

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### ページ編集時にトピック変更を許可する方式

従来はページ編集時にトピックも変更可能だったが、以下の理由から編集と移動を分離する方式に変更した。

- 編集提案機能との整合性: 編集提案ではトピック変更を対象外としたため、編集中にトピック変更ができると「トピックを変更したページは編集提案として提出できない」という制限が生じ、ユーザー体験が複雑になる
- 権限モデルの複雑化: トピックごとに権限が異なるため、編集とトピック変更を同時に行うと権限チェックが複雑になる
- 操作の明確化: 「内容の変更」と「構造の変更」は異なる意図の操作であり、分離することで各操作の目的が明確になる

### 同タイトルのページを新規作成する方式

トピック変更の代わりに、移動先のトピックに同タイトルのページを新規作成する方式を検討したが、以下の理由から採用しなかった。

- リビジョン履歴が引き継がれず、変更の追跡が途切れる
- 他のページからのリンクが切れる
- 元のページを手動で削除する必要があり、元トピックの削除権限がないと残り続ける

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

### フェーズ 1: バックエンド基盤

- [x] **1-1**: [Go] TopicPolicy に CanCreatePage を追加し、PageRepository に MoveTopic メソッドを追加
  - `go/internal/policy/topic.go` に `CanCreatePage(topic *model.Topic) bool` を追加
  - `go/internal/query/queries/pages.sql` に `MovePageToTopic` クエリを追加
  - `go/internal/repository/page.go` に `MoveTopic` メソッドを追加
  - sqlc コード生成を実行
  - 想定ファイル数: 実装 4 ファイル / テスト 2 ファイル
  - 想定行数: 実装 ~80 行 / テスト ~120 行

- [x] **1-2**: [Go] MovePageUsecase の実装
  - `go/internal/usecase/move_page.go` を作成
  - トランザクション内で `pageRepo.MoveTopic` を呼び出す
  - 依存: 1-1
  - 想定ファイル数: 実装 1 ファイル / テスト 1 ファイル
  - 想定行数: 実装 ~80 行 / テスト ~100 行

### フェーズ 2: ページ移動画面

- [x] **2-1**: [Go] ページ移動画面の実装（Handler・Template・ViewModel・Routing・I18n・Validator）
  - `go/internal/handler/page_move/` ディレクトリを作成（handler.go, new.go, create.go, validator.go）
  - `go/internal/templates/pages/page_move/new.templ` を作成
  - `go/internal/viewmodel/topic.go` に `TopicForSelect` を追加
  - `cmd/server/main.go` にルーティングを追加
  - `go/internal/middleware/reverse_proxy.go` にパターンを追加
  - `go/internal/i18n/locales/ja.toml`, `en.toml` に翻訳を追加
  - 依存: 1-2
  - 想定ファイル数: 実装 ~12 ファイル / テスト ~4 ファイル
  - 想定行数: 実装 ~250 行 / テスト ~200 行

### フェーズ 3: Rails 側の変更

- [x] **3-1**: [Rails] PageActionsComponent に「移動」リンクを追加
  - `rails/app/components/dropdowns/page_actions_component.html.erb` に移動リンクを追加
  - `rails/config/locales/` に翻訳を追加（必要に応じて）
  - 依存: 2-1
  - 想定ファイル数: 実装 ~3 ファイル / テスト 0 ファイル
  - 想定行数: 実装 ~10 行 / テスト 0 行

### フェーズ 4: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [x] **4-1**: 仕様書の作成・更新
  - `docs/specs/page/move.md` に仕様書を作成する
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する
