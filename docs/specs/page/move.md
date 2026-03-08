# ページの移動 仕様書

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

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

### 基本仕様

- ユーザーはページを同一スペース内の別のトピックに移動できる
- 移動時にページの内容（リビジョン履歴を含む）は変更されない
- ページの移動は編集操作とは独立した操作であり、ページ編集中にトピックを変更することはできない

### 権限

- ページの移動には、移動元トピックの `CanUpdatePage` 権限が必要である
- 移動先トピックの `CanCreatePage` 権限が必要である
- 権限のないトピックへの移動はシステムが拒否する

### `CanCreatePage` の判定

| ポリシー          | CanCreatePage の判定                             |
| ----------------- | ------------------------------------------------ |
| topicOwnerPolicy  | `spaceMemberActive && spaceID == topic.Space.ID` |
| topicAdminPolicy  | `spaceMemberActive && topicID == topic.ID`       |
| topicMemberPolicy | `spaceMemberActive && topicID == topic.ID`       |
| topicGuestPolicy  | `false`（常に不可）                              |

### 移動先トピックの選択肢

- スペースオーナーの場合: スペース内の全アクティブトピック
- トピックメンバーの場合: 所属トピックのみ
- いずれの場合も現在のトピックを除外して表示する

### バリデーション

形式バリデーション:

- 移動先トピックが選択されていること（必須）

状態バリデーション:

- 移動先トピックが同一スペース内に存在すること
- 移動先トピックが現在のトピックと異なること
- 移動先トピックにページ作成権限があること（`CanCreatePage`）
- 移動先トピックに同名のページが存在しないこと（タイトルの一意制約）

### UI

- ページ移動画面は、パンくずリスト、見出し「ページの移動」、ページタイトル（読み取り専用）、現在のトピック名（読み取り専用）、移動先トピックのセレクトボックス、「移動する」ボタンで構成される
- 成功時はフラッシュメッセージ「ページを移動しました」を表示し、ページ詳細画面にリダイレクトする
- 移動先トピックが存在しない場合（ユーザーが1つのトピックにしか所属していない場合）は、セレクトボックスが空になり、ボタンは無効化される。「移動先のトピックがありません」というメッセージを表示する

### Rails 版との連携

- Rails 版の `Dropdowns::PageActionsComponent` に「移動」リンクが追加されている
- `page.can_update?` が真の場合にのみ表示される
- リンク先は Go 版のページ移動画面に遷移する

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

| メソッド | パス                                             | ハンドラー         | 説明               |
| -------- | ------------------------------------------------ | ------------------ | ------------------ |
| GET      | `/s/{space_identifier}/pages/{page_number}/move` | `page_move.New`    | ページ移動フォーム |
| POST     | `/s/{space_identifier}/pages/{page_number}/move` | `page_move.Create` | ページ移動実行     |

リバースプロキシの `FeatureFlagGoPageEdit` フラグで Go 版と Rails 版を切り替える。

### コード設計

#### Handler (`go/internal/handler/page_move/`)

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

#### UseCase (`go/internal/usecase/move_page.go`)

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

トランザクション内で `pageRepo.MoveTopic` を呼び出し、`topic_id` と `updated_at` を更新する。

#### Repository (`go/internal/repository/page.go`)

`PageRepository` に `MoveTopic` メソッドを追加。`topic_id` と `updated_at` のみを更新する専用 SQL クエリを使用する。

```sql
-- name: MovePageToTopic :one
UPDATE pages
SET topic_id = $2, updated_at = NOW()
WHERE id = $1 AND space_id = $3
RETURNING *;
```

#### ViewModel (`go/internal/viewmodel/topic.go`)

セレクトボックス用の ViewModel:

```go
type TopicForSelect struct {
    Name   string
    Number int32
}
```

#### Template (`go/internal/templates/pages/page_move/new.templ`)

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

### I18n

ja.toml / en.toml に以下のキーを定義:

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

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
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

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [ページの移動 作業計画書](/workspace/docs/plans/1_doing/page-move.md)
- [編集提案機能 作業計画書](/workspace/docs/plans/1_doing/edit-suggestion.md)
