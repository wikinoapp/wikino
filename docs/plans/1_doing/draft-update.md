# 下書き機能のアップデート 作業計画書

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

- [ページ編集 仕様書](../specs/page/edit.md)（タスク完了後に更新予定）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

下書き機能を拡張し、DraftPageRevision（下書きのバージョン）と下書き一覧画面を実装する。

**DraftPageRevision**: ユーザーが明示的に「下書き保存」操作を行ったタイミング、またはAIが下書きを編集したタイミングで作成される。現在のDBスキーマにはDraftPageRevisionのテーブルが存在しないため、DBマイグレーションから新規に実装する。

**下書き一覧画面** (`GET /drafts`): 参加しているすべてのスペースのトピックごとに、下書き保存しているページを一覧表示する。編集提案機能の前提となる画面。

**目的**:

- 下書き内での段階的な差分確認を可能にする（DraftPageRevision間の差分表示）
- AIによる編集内容を下書きの編集履歴で差分として確認できるようにする
- 編集提案機能の前提となる（下書き一覧画面から編集提案を作成できるようにする）

### 依存タスク

- 前提: [@docs/plans/2_todo/page-edit-go-migration.md](page-edit-go-migration.md) - ページ編集画面のGo移行

### 関連タスク

- [@docs/plans/1_doing/edit-suggestion.md](/workspace/docs/plans/1_doing/edit-suggestion.md) - 編集提案機能（本タスクが前提）

## 要件

<!--
ガイドライン:
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な要件（セキュリティ、パフォーマンスなど）も記述
-->

### DraftPageRevision

- ユーザーが編集画面で「下書き保存」操作を行うと、DraftPageRevisionが作成される
- 自動保存ではDraftPageRevisionは作成されない
- AIがMCP経由で下書きを編集した場合、編集後に自動的にDraftPageRevisionが作成される
- DraftPageRevisionはDraftPageと同様、作成者本人のみが閲覧できる（非公開）

### 下書き一覧画面

- ユーザーは参加しているスペースのトピックごとに下書きページを一覧表示できる
- スペース・トピックでグルーピングされて表示される

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

ページモデル全体の概要（Page, PageRevision, DraftPage, 用語、編集フローなど）は [ページ編集画面のGo移行](page-edit-go-migration.md) の「設計」セクションを参照。

### DraftPageRevision データモデル

下書きのバージョン。ユーザーが明示的に保存操作を行ったタイミング、またはAIが下書きを編集したタイミングで作成される。DraftPageに紐づき、下書き内の変更を段階的に確認するために使用する。

| フィールド      | 型       | 説明                               |
| --------------- | -------- | ---------------------------------- |
| `id`            | `string` | 一意の識別子（ULID）               |
| `draftPageId`   | `string` | 所属するDraftPageのID              |
| `spaceMemberId` | `string` | 作成したスペースメンバーのID       |
| `title`         | `string` | バージョン作成時点のページタイトル |
| `body`          | `string` | バージョン作成時点のMarkdown本文   |
| `bodyHtml`      | `string` | バージョン作成時点のHTML本文       |
| `createdAt`     | `Date`   | 作成日時                           |

DraftPageRevisionの設計意図:

- **段階的な差分確認**: 公開版との差分だけでなく、DraftPageRevision間の差分を確認できる。変更量が多い場合でも、各DraftPageRevision間の差分を見ることで変更内容を追いやすくなる
- **明示的な作成**: 自動保存（DraftPageへの永続化）のたびに作成するのではなく、ユーザーの明示的な「下書き保存」操作またはAI編集時にのみ作成する。これにより不要なバージョンの増大を防ぐ
- **非公開**: DraftPageと同様、自分だけが見える。公開時に作成されるPageRevisionとは異なる

```go
type DraftPageRevision struct {
	ID            DraftPageRevisionID
	DraftPageID   DraftPageID
	SpaceMemberID SpaceMemberID
	Title         string    // バージョン作成時点のページタイトル
	Body          string    // バージョン作成時点のMarkdown本文
	BodyHTML      string    // バージョン作成時点のHTML本文
	CreatedAt     time.Time
}
```

### テーブル設計

- テーブル名: `draft_page_revisions`
- インデックス: `[draft_page_id, created_at]`

### 下書き保存フロー

下書き保存操作により、下書きのバージョン（DraftPageRevision）を作成する。自動保存（DraftPageへの永続化）とは異なり、ユーザーの明示的な操作によって行われる。以下のタイミングで作成される。

- ユーザーが編集画面で「下書き保存」操作を行うと、その時点の下書き内容でDraftPageRevisionが作成される
- 自動保存ではDraftPageRevisionは作成されない
- AIがMCP経由で下書きを編集した場合、編集後に自動的にDraftPageRevisionが作成される（AIが何を変更したかを下書きの編集履歴で差分として確認できる）

### Usecase命名の方針

自動保存と手動保存（下書き保存）を対にした命名とする。

| 操作     | Usecase                      | ファイル名                  |
| -------- | ---------------------------- | --------------------------- |
| 自動保存 | `AutoSaveDraftPageUsecase`   | `auto_save_draft_page.go`   |
| 手動保存 | `ManualSaveDraftPageUsecase` | `manual_save_draft_page.go` |

### 下書き一覧画面

- エンドポイント: `GET /drafts`
- 参加しているすべてのスペースのトピックごとに下書き保存しているページを一覧表示する

```
/drafts (下書き一覧画面)
  スペースA / トピック1
    - ページX の下書き
    - ページY の下書き
  スペースA / トピック2
    - ページZ の下書き
```

NOTE: 「編集提案する...」ボタンは編集提案機能のタスク（edit-suggestion.md）で追加する。

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### 自動保存のたびにDraftPageRevisionを作成する方式

Jujutsu（jj-vcs）のように、下書きが自動保存されるたびに自動でDraftPageRevisionを作成する方式を検討した。しかし、以下の理由から、ユーザーの明示的な「下書き保存」操作でのみDraftPageRevisionを作成する方式を採用した。

- 自動保存のたびにDraftPageRevisionを作成すると大量のレコードが生成される（将来的にCRDTによるリアルタイム保存を導入した場合、特に顕著になる）
- ユーザーが意図したタイミングで下書き保存するほうが、各DraftPageRevision間の差分が意味のある単位になる

AIによる編集時の自動作成は例外として許可した。AIの編集は1回の操作が明確な単位（1つの改善提案など）を持つため、自動作成しても不要なバージョンが増大しにくい。

### 下書き一覧画面にチェックボックスを配置する案

編集提案機能（edit-suggestion.md）の「採用しなかった方針」を参照。

### 下書き詳細画面を本タスクに含める案

下書き詳細画面（`GET /s/:space_id/topics/:topic_id/draft`）を本タスクのスコープに含めることを検討した。下書き詳細画面ではトピック内の下書きページの一覧、編集履歴（DraftPageRevision一覧）、差分表示が可能になる。

**不採用の理由**:

- 編集提案機能の前提として最低限必要なのは下書き一覧画面であり、下書き詳細画面は必須ではない
- 下書き詳細画面は差分表示コンポーネント（[@docs/plans/2_todo/diff-component.md](diff-component.md)）が前提となり、依存が増える
- 下書き詳細画面は独立したタスクとして後から追加できる

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

### フェーズ 1: DraftPageRevision のデータ層

- [x] **1-1**: [Go] DraftPageRevision のDBマイグレーションとモデル定義
  - `draft_page_revisions` テーブルを作成するマイグレーション
  - `internal/model/id.go` に `DraftPageRevisionID` 型を追加
  - `internal/model/draft_page_revision.go` にモデル定義
  - 想定ファイル数: 実装 3, テスト 0
  - 想定行数: 実装 60, テスト 0

- [x] **1-2**: [Go] DraftPageRevision の sqlc クエリとリポジトリ
  - `db/queries/draft_page_revisions.sql` に Create クエリを追加
  - `internal/repository/draft_page_revision.go` にリポジトリ実装（Create, WithTx）
  - sqlc コード生成
  - 想定ファイル数: 実装 3, テスト 1
  - 想定行数: 実装 80, テスト 60
  - 依存: 1-1

### フェーズ 2: 下書き保存 Usecase とハンドラー

- [x] **2-1**: [Go] ManualSaveDraftPage Usecase の実装
  - `internal/usecase/manual_save_draft_page.go` に Usecase 実装
  - DraftPage の現在の内容でスナップショット（DraftPageRevision）を作成する
  - トランザクション内で DraftPageRevision を作成
  - 想定ファイル数: 実装 1, テスト 1
  - 想定行数: 実装 80, テスト 100
  - 依存: 1-2

- [x] **2-2**: [Go] 編集画面に「下書き保存」ボタンを追加
  - `internal/handler/draft_page/create.go` に POST ハンドラーを追加（DraftPageRevision の作成）
  - `internal/handler/draft_page/handler.go` に Usecase の依存を追加
  - `internal/templates/pages/page/edit.templ` に「下書き保存」ボタンを追加
  - `internal/i18n/locales/ja.toml`, `en.toml` に翻訳キーを追加
  - ルーティング追加（`cmd/server/main.go`）
  - 想定ファイル数: 実装 7, テスト 1
  - 想定行数: 実装 120, テスト 80
  - 依存: 2-1

### フェーズ 3: 下書き一覧画面

- [ ] **3-1**: [Go] 下書き一覧用のクエリとリポジトリメソッド追加
  - `db/queries/joined_draft_pages.sql` に下書き一覧用クエリを追加（スペース名・トピック名を含む、スペース・トピック順にソート）
  - `internal/repository/draft_page.go` に `ListByUserForIndex` メソッドを追加
  - sqlc コード生成
  - 想定ファイル数: 実装 3, テスト 1
  - 想定行数: 実装 80, テスト 60

- [ ] **3-2**: [Go] 下書き一覧画面の ViewModel・テンプレート・ハンドラー
  - `internal/viewmodel/draft_page_for_index.go` に一覧用 ViewModel（グルーピングロジック含む）
  - `internal/templates/pages/draft_page/index.templ` にテンプレート
  - `internal/handler/draft_page_index/handler.go`, `index.go` にハンドラー
  - `internal/i18n/locales/ja.toml`, `en.toml` に翻訳キーを追加
  - ルーティング追加（`cmd/server/main.go`）
  - 想定ファイル数: 実装 8, テスト 1
  - 想定行数: 実装 200, テスト 80
  - 依存: 3-1

### フェーズ 4: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [ ] **4-1**: 仕様書の作成・更新
  - `docs/specs/page/edit.md` の仕様書にDraftPageRevisionの内容を追記する
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

以下の機能は今回の実装では**実装しません**：

- **下書き詳細画面**: 差分表示コンポーネントが前提となるため、独立したタスクとして後から実装する
