# ページ編集画面のGo移行 作業計画書

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

- [ページ編集 仕様書](../specs/page/edit.md)（タスク完了後に作成予定）
- [Wikiリンク 仕様書](../specs/page/wikilink.md)（タスク完了後に作成予定）
- [Wikiリンク補完 仕様書](../specs/page/wikilink-completion.md)（タスク完了後に作成予定）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

ページ編集画面および関連するモデル（Page, PageRevision, DraftPage）をRailsからGoに移行する。現在Rails側で実装されているページの表示・編集・公開・破棄のフローをGo版で再実装する。

これは編集提案機能をはじめとする今後の機能開発の基盤となる。ページのリビジョン管理システム（PageRevision）、下書きシステム（DraftPage）のGo版実装が必要。

### 関連タスク

- [@docs/plans/2_todo/edit-suggestion.md](edit-suggestion.md) - 編集提案機能（本タスクが前提）
- [@docs/plans/2_todo/page-move.md](page-move.md) - ページの移動機能（本タスクと並行可能）
- [@docs/plans/2_todo/draft-page-revision.md](draft-page-revision.md) - DraftPageRevisionの実装（本タスクが前提）
- [@docs/plans/2_todo/title-change-link-rewrite.md](title-change-link-rewrite.md) - タイトル変更時のリンク自動書き換え（本タスクが前提）
- [@docs/plans/2_todo/page-revision-history.md](page-revision-history.md) - 編集履歴画面・ロールバック（本タスクが前提）
- [@docs/plans/2_todo/draft-page-discard.md](draft-page-discard.md) - 下書き破棄機能（本タスクが前提）
- [@docs/plans/2_todo/publish-diff-confirmation.md](publish-diff-confirmation.md) - 公開前の差分確認（本タスクが前提）
- [@docs/plans/2_todo/page-show-go-migration.md](../2_todo/page-show-go-migration.md) - ページ表示画面のGo移行（本タスクが前提）
- [@docs/plans/2_todo/page-ogp-meta.md](../2_todo/page-ogp-meta.md) - ページのOGPメタタグ設定（本タスクが前提）
- ~~[@docs/plans/2_todo/wikilink-completion.md](../2_todo/wikilink-completion.md)~~ - Wikiリンク補完（本タスクに統合済み）
- ~~[@docs/plans/2_todo/topic-member-timestamp.md](../2_todo/topic-member-timestamp.md)~~ - TopicMemberのタイムスタンプ更新（本タスクに統合済み）
- ~~[@docs/plans/2_todo/wikilink-auto-creation.md](../2_todo/wikilink-auto-creation.md)~~ - Wikiリンクによるページ自動作成（本タスクに統合済み）

## 要件

<!--
ガイドライン:
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な要件（セキュリティ、パフォーマンスなど）も記述
-->

- ユーザーはページの編集画面を開ける
- ユーザーの編集内容はDraftPageに自動保存される
- ユーザーはDraftPageの内容を公開できる（PageRevisionの作成）
- ページ公開時に本文中の添付ファイル参照を検出し、PageAttachmentReferenceを同期する（追加・削除）
- ページ公開時に本文1行目から画像IDを抽出し、featured_image_attachment_idを設定する
- ページ公開時にTopicMemberの`last_page_modified_at`を公開日時で更新する（トピック一覧の「最終更新日時」ソートに使用）
- システムはページ本文中の`[[ページ名]]`および`[[トピック名/ページ名]]`形式のWikiリンクを解析し、リンク先ページが存在しない場合は空ページを自動作成する
- システムは解析結果をPage/DraftPageの`linkedPageIds`フィールドに格納する
- システムはWikiリンクをHTML内で`<a>`タグに変換する（リンク先ページが存在する場合のみ）
- 自動保存時と公開時の両方でWikiリンク解析が実行される
- ユーザーはページ編集画面のフッターでリンク一覧（そのページからリンクしている他ページ）を確認できる
- ユーザーはページ編集画面のフッターでバックリンク一覧（そのページにリンクしている他ページ）を確認できる
- ユーザーはエディタにファイルをドラッグ&ドロップしてアップロードできる
- ユーザーはクリップボードから画像をペーストしてアップロードできる
- アップロード完了後、ファイルのMarkdownリンク（またはimgタグ）がエディタに挿入される
- トピックの可視性（public/private）とトピックメンバーシップに基づくページ編集の権限チェックが行われる
- スペースオーナーは全トピックのページを編集・公開できる
- トピックメンバー（admin/member）は所属トピックのページを編集・公開できる
- 非トピックメンバーはプライベートトピックのページを編集できない
- ユーザーがエディタで`[[`を入力するとWikiリンク補完候補が表示される
- 候補はスペース内の既存ページ名（トピック名/ページタイトル形式）から検索される
- 候補を選択すると`[[トピック名/ページタイトル]]`形式のWikiリンクが挿入される
- ページタイトルの部分一致（AND条件）で候補がフィルタリングされる
- Rails版と同等の機能を提供する

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

**重要: 設計は実装中に更新する**:
- 作業計画書内の設計は初期の方針であり、完璧ではない
- 実装中により良いアプローチが見つかった場合は、設計を積極的に更新する
- 設計に固執して実装の質を下げるよりも、実装で得た知見を設計に反映する方が重要
- 変更した場合は「採用しなかった方針」セクションに変更前の方針と変更理由を記録する
-->

### ページモデルの概要

ページはWikinoにおけるコンテンツの基本単位である。各ページはPageRevision（UIでは「バージョン」と表示、一覧は「編集履歴」と表示）による変更管理を持ち、「下書き（DraftPage）」と「公開済みのコンテンツ（PageRevision）」を分離した編集フローを提供する。

編集内容はDraftPageに自動保存され、ユーザーが明示的に「下書き保存」すると下書きのバージョン（DraftPageRevision）が作成される。さらに「公開」するとバージョン（PageRevision）が作成されて他の人にも見えるようになり、編集履歴として追跡できるようになる。

**目的**:

- ページの変更履歴を追跡可能にする
- 編集中の内容が他のユーザーに影響を与えない安全な編集環境を提供する
- 将来的に複数ユーザーによるリアルタイム同時編集を可能にする

**背景**:

- Gitのコミットモデルを参考に、「自動保存」「下書き保存」「公開」の3段階の設計を採用した
- 将来的にはCRDTを使用してリアルタイム同時編集の実現を計画している

### 用語

| 操作                        | 英語       | 対象モデル        | 意味                                                                                      |
| --------------------------- | ---------- | ----------------- | ----------------------------------------------------------------------------------------- |
| **自動保存 (Auto-save)**    | Auto-save  | DraftPage         | 編集内容をDraftPageに自動的に永続化する。ユーザーが意識しない裏側の処理。                 |
| **下書き保存 (Save Draft)** | Save Draft | DraftPageRevision | 下書きのバージョン（DraftPageRevision）を作成する。ユーザーの明示的な操作（Ctrl+S相当）。 |
| **公開 (Publish)**          | Publish    | PageRevision      | 他の人に見えるようにする。このタイミングでバージョン（PageRevision）が作成される。        |

Gitとの対応:

| Wikino     | Git                          |
| ---------- | ---------------------------- |
| 自動保存   | ファイル編集（working tree） |
| 下書き保存 | commit                       |
| 公開       | push                         |

### 日本語での表示名

内部モデル名とUI上の日本語表示名の対応を以下に定義する。

| モデル名（内部）      | 単数形（UI表示）   | 複数形・一覧（UI表示） |
| --------------------- | ------------------ | ---------------------- |
| **PageRevision**      | バージョン         | 編集履歴               |
| **DraftPageRevision** | 下書きのバージョン | 下書きの編集履歴       |

- **単数形（バージョン）**: 個々のPageRevision/DraftPageRevisionを指すときに使用する。例: 「バージョン 3」「このバージョンとの差分」「下書き保存する」
- **複数形・一覧（編集履歴）**: PageRevisionの一覧や履歴を表示するときに使用する。例: 「編集履歴を見る」「下書きの編集履歴」

### データモデル

以下のモデルをGo版で実装する:

- **Page** - ページのデータ管理
- **PageRevision** - 公開されたページのスナップショット
- **DraftPage** - ページの下書き（スペースメンバーごとに分離）
- **PageAttachmentReference** - ページと添付ファイルの参照関係

#### Page

ページのデータを管理する。最新のタイトル・本文は`pages`テーブルに直接格納しており、`page_revisions`テーブルをJOINせずにページ表示が可能な非正規化設計を採用している。これはページ表示が最も頻繁な操作であるため、読み取りパフォーマンスを優先した設計である。

| フィールド                  | 型               | 説明                                                             |
| --------------------------- | ---------------- | ---------------------------------------------------------------- |
| `id`                        | `string`         | 一意の識別子（ULID）                                             |
| `spaceId`                   | `string`         | 所属するスペースのID                                             |
| `topicId`                   | `string`         | 所属するトピックのID                                             |
| `number`                    | `number`         | URLに使用されるページ番号（スペース内でユニーク）                |
| `title`                     | `string \| null` | ページタイトル（トピック内でユニーク、大文字小文字を区別しない） |
| `body`                      | `string`         | Markdownの本文                                                   |
| `bodyHtml`                  | `string`         | HTMLに変換された本文                                             |
| `linkedPageIds`             | `string[]`       | 本文中のリンクで参照されているページIDのリスト                   |
| `modifiedAt`                | `Date`           | 内容の更新日時                                                   |
| `publishedAt`               | `Date \| null`   | 公開日時（nullなら非公開）                                       |
| `trashedAt`                 | `Date \| null`   | ゴミ箱に入れた日時                                               |
| `createdAt`                 | `Date`           | 作成日時                                                         |
| `updatedAt`                 | `Date`           | レコードの更新日時                                               |
| `pinnedAt`                  | `Date \| null`   | ピン留め日時                                                     |
| `discardedAt`               | `Date \| null`   | 廃棄日時                                                         |
| `featuredImageAttachmentId` | `string \| null` | アイキャッチ画像の添付ファイルID                                 |

#### PageRevision（UI表示: バージョン / 編集履歴）

公開されたページのスナップショット。公開のたびに作成され、その時点のタイトル・本文を記録する。UIでは個々のPageRevisionを「バージョン」、PageRevisionの一覧を「編集履歴」と表示する。

| フィールド      | 型       | 説明                         |
| --------------- | -------- | ---------------------------- |
| `id`            | `string` | 一意の識別子（ULID）         |
| `spaceId`       | `string` | 所属するスペースのID         |
| `spaceMemberId` | `string` | 作成したスペースメンバーのID |
| `pageId`        | `string` | 所属するページのID           |
| `title`         | `string` | この時点のページタイトル     |
| `body`          | `string` | この時点のMarkdown本文       |
| `bodyHtml`      | `string` | この時点のHTML本文           |
| `createdAt`     | `Date`   | 作成日時                     |
| `updatedAt`     | `Date`   | レコードの更新日時           |

PageRevisionの設計意図:

- **非正規化との併用**: Pageテーブルには最新の内容が格納されているため、PageRevisionは変更履歴の保持が主目的
- **`createdAt`による順序管理**: 同一ページのPageRevisionを`createdAt`で昇順に並べることで編集履歴を表示する

#### DraftPage

ページの下書き。スペースメンバー × ページごとに1つ存在する。`spaceMemberId + pageId` でユニーク制約を持つ。DBテーブル名は`draft_pages`。

| フィールド      | 型               | 説明                                           |
| --------------- | ---------------- | ---------------------------------------------- |
| `id`            | `string`         | 一意の識別子（ULID）                           |
| `spaceId`       | `string`         | 所属するスペースのID                           |
| `pageId`        | `string`         | 対象ページのID                                 |
| `spaceMemberId` | `string`         | スペースメンバーのID（メンバーごとに分離）     |
| `topicId`       | `string`         | トピックのID                                   |
| `title`         | `string \| null` | 編集中のページタイトル                         |
| `body`          | `string`         | 編集中のMarkdown本文                           |
| `bodyHtml`      | `string`         | 編集中のHTML本文                               |
| `linkedPageIds` | `string[]`       | 本文中のリンクで参照されているページIDのリスト |
| `modifiedAt`    | `Date`           | 内容の更新日時                                 |
| `createdAt`     | `Date`           | 作成日時                                       |
| `updatedAt`     | `Date`           | レコードの更新日時                             |

DraftPageの設計意図:

- **スペースメンバーごとに分離**: 自分の下書きが他の人に見えない

#### PageAttachmentReference

ページ本文中で参照されている添付ファイルの追跡レコード。ページ公開時にbodyHtmlを解析して同期する。`page_id + attachment_id` でユニーク制約を持つ。DBテーブル名は`page_attachment_references`。

| フィールド     | 型       | 説明                 |
| -------------- | -------- | -------------------- |
| `id`           | `string` | 一意の識別子（ULID） |
| `attachmentId` | `string` | 添付ファイルのID     |
| `pageId`       | `string` | 所属するページのID   |
| `createdAt`    | `Date`   | 作成日時             |
| `updatedAt`    | `Date`   | レコードの更新日時   |

PageAttachmentReferenceの設計意図:

- **参照追跡**: ページがどの添付ファイルを使用しているかを追跡する
- **差分同期**: 公開のたびに、追加された参照と削除された参照を差分更新する
- **添付ファイル存在確認**: 参照追加時に、指定IDの添付ファイルが同じスペースに存在するか検証する

#### TopicMember

トピックのメンバーシップ管理。スペースメンバーがどのトピックに所属しているかを追跡する。`space_member_id + topic_id` でユニーク制約を持つ。DBテーブル名は`topic_members`。

| フィールド           | 型             | 説明                          |
| -------------------- | -------------- | ----------------------------- |
| `id`                 | `string`       | 一意の識別子（ULID）          |
| `spaceId`            | `string`       | 所属するスペースのID          |
| `topicId`            | `string`       | 所属するトピックのID          |
| `spaceMemberId`      | `string`       | スペースメンバーのID          |
| `role`               | `int`          | ロール（0: admin, 1: member） |
| `joinedAt`           | `Date`         | トピック参加日時              |
| `lastPageModifiedAt` | `Date \| null` | 最後にページを更新した日時    |
| `createdAt`          | `Date`         | 作成日時                      |
| `updatedAt`          | `Date`         | レコードの更新日時            |

TopicMemberの設計意図:

- **トピックレベルのアクセス制御**: プライベートトピックのページはトピックメンバーのみが編集可能
- **ロール管理**: admin（トピック管理権限あり）とmember（ページ編集のみ）の2種類

#### DraftPageRevision（UI表示: 下書きのバージョン / 下書きの編集履歴）【計画中】

DraftPageRevisionの詳細（データモデル、テーブル設計、下書き保存フローなど）は [DraftPageRevisionの実装](draft-page-revision.md) の「設計」セクションを参照。

### ページの編集フロー

#### ページを開く

- 編集画面を開くと、自分のDraftPageが存在すればその内容を表示し、存在しなければPageの現在の内容を表示する
- DraftPageは初回の自動保存時に作成する（find_or_create方式）。編集画面を開いた時点ではDraftPageを作成しない（Rails版の挙動を踏襲）

#### 編集

- ページの本文とタイトルの両方を編集できる
- タイトルはDraftPageの`title`フィールドとして保持される
- タイトルの変更は公開時に反映される（編集中は他のユーザーには見えない）
- トピックの変更は別機能（page-move）として切り出すため、ページ編集画面では対応しない

単独編集:

- 自分のDraftPageだけを更新する
- 他のユーザーには見えない
- 編集内容はDraftPageのbody・bodyHtml・titleに自動保存される

同時編集【計画中】:

> **注**: 同時編集はCRDTの導入により実現する計画中の機能。現在のDBスキーマでは対応していない。

- 同じページを編集中の他のユーザーとCRDTセッションを共有する
- リアルタイムで変更が同期される
- 編集終了時、各ユーザーのDraftPageにそれぞれ保存される

#### 下書き保存する【計画中】

下書き保存フローの詳細は [DraftPageRevisionの実装](draft-page-revision.md) の「設計」セクションを参照。

#### 公開する

1. DraftPageの内容をPageに反映する（title、body、bodyHtml、linkedPageIds、modifiedAt）
2. 新しいPageRevisionを作成する（この時点のタイトルと本文のスナップショットを記録）
3. PageのpublishedAtを更新する
4. 自分のDraftPageを削除する
5. TopicMemberの`last_page_modified_at`を公開日時で更新する
6. bodyHtmlから添付ファイルIDを検出し、PageAttachmentReferenceを差分同期する（追加・削除）
7. bodyの1行目から画像IDを抽出し、Pageのfeatured_image_attachment_idを更新する（見つからない場合はnullに設定）

> **注**: 公開前の差分確認は [公開前の差分確認](publish-diff-confirmation.md) で別途実装する。タイトル変更時のリンク自動書き換えは [タイトル変更時のWikiリンク自動書き換え](title-change-link-rewrite.md) で別途実装する。

#### 破棄する

Rails版では「キャンセル」リンクがページ表示に戻るのみで、DraftPageは削除されない（DraftPageは公開時にのみ削除される）。Go版でもこの挙動を踏襲する。

明示的なDraftPage破棄機能は [下書き破棄機能](draft-page-discard.md) で別途実装する。

#### Claudeによる編集

Claude DesktopからMCP経由でページを編集できる。指定されたスペースメンバーのDraftPageを更新する。

- Claudeの編集は指定スペースメンバーのDraftPageを更新する
- DraftPageRevision実装後は、編集後にDraftPageRevisionが自動作成され、AIが何を変更したかを下書きの編集履歴で差分として確認できるようになる【計画中】
- 公開（Publish）はユーザーが明示的に行う
- 気に入らなければ破棄して公開版（Pageの現在の内容）に戻せる
- 他のスペースメンバーのDraftPageには影響しない

#### 編集履歴・ロールバック

編集履歴画面およびロールバック機能の詳細は [編集履歴画面・ロールバック](page-revision-history.md) を参照。

### データ構造

```go
type Page struct {
	ID                        string
	SpaceID                   string
	TopicID                   string
	Number                    int
	Title                     *string    // nilの場合もある（大文字小文字を区別しない）
	Body                      string     // Markdown本文
	BodyHTML                  string     // HTML変換済み本文
	LinkedPageIDs             []string   // 本文中のリンクで参照されているページID
	ModifiedAt                time.Time  // 内容の更新日時
	PublishedAt               *time.Time // nilなら非公開
	TrashedAt                 *time.Time // ゴミ箱に入れた日時
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	PinnedAt                  *time.Time // ピン留め日時
	DiscardedAt               *time.Time // 廃棄日時
	FeaturedImageAttachmentID *string    // アイキャッチ画像
}

type PageRevision struct {
	ID            string
	SpaceID       string
	SpaceMemberID string
	PageID        string
	Title         string     // この時点のページタイトル
	Body          string     // この時点のMarkdown本文
	BodyHTML      string     // この時点のHTML本文
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type DraftPage struct {
	ID            string
	SpaceID       string
	PageID        string
	SpaceMemberID string     // スペースメンバーごとに分離
	TopicID       string
	Title         *string    // 編集中のページタイトル
	Body          string     // 編集中のMarkdown本文
	BodyHTML      string     // 編集中のHTML本文
	LinkedPageIDs []string   // 本文中のリンクで参照されているページID
	ModifiedAt    time.Time  // 内容の更新日時
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type PageAttachmentReference struct {
	ID           string
	AttachmentID string
	PageID       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TopicMember struct {
	ID                 string
	SpaceID            string
	TopicID            string
	SpaceMemberID      string
	Role               int        // 0: admin, 1: member
	JoinedAt           time.Time
	LastPageModifiedAt *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// DraftPageRevision は計画中の機能
// 詳細は docs/plans/2_todo/draft-page-revision.md を参照
```

### データフロー

```
自動保存                    下書き保存                公開
━━━━━━                    ━━━━━━━━                ━━━━

 DraftPage               DraftPageRevision            Page + PageRevision
 ┌──────────────┐        ┌──────────────┐        ┌──────────────┐
 │ # タイトル   │  [Save │ # タイトル   │        │ # タイトル   │
 │              │  Draft]│              │[Publish]│              │
 │ 本文を編集中 │──────→│ 本文         │──────→│ 本文         │
 │ ...         │        │ (スナップ    │        │              │
 └──────────────┘        │  ショット)  │        └──────────────┘
       ↑                  └──────────────┘               │
  自動保存                (自分だけ見える)           PageRevision作成
  (自分だけ見える)                                  (他の人に見える)
```

#### 同時編集のデータフロー【計画中】

```
ユーザーA                          ユーザーB
    │                                  │
    ▼                                  ▼
DraftPage (A)                DraftPage (B)
    │                                  │
    │   [同時編集を開始]               │
    │                                  │
    └────────────┬─────────────────────┘
                 ▼
        共有 CRDT セッション
        (リアルタイム同期)
                 │
                 │ 編集終了時
    ┌────────────┴─────────────────────┐
    ▼                                  ▼
DraftPage (A)                DraftPage (B)
に保存                            に保存
```

#### Claudeによる編集のデータフロー

```
Claude Desktop
      │
      │ MCP: edit_page(pageId, spaceMemberId, newContent)
      ▼
Wikino Server
      │
      │ 対象スペースメンバーの DraftPage を取得（なければ作成）
      │ DraftPage の内容を更新
      ▼
DraftPage 更新
      │
      ├──→ DraftPageRevision 自動作成【計画中】
      │    （AI編集の差分を編集履歴で確認するため）
      ▼
ユーザーが公開操作を行うまで下書きとして保持
```

### エンドポイント

Rails版のURL構造を踏襲する。ページのURL階層にトピックは含めない（トピックはページの分類であり、URLの階層構造には反映しない）。

> **Note**: 開発中はGo版とRails版の両方にアクセスできるよう、Go版のエンドポイントには一時的に `/go` プレフィックスを付与する（例: `GET /go/s/:space_identifier/pages/:page_number/edit`）。フェーズ 9 でプレフィックスを除去し、リバースプロキシの設定を更新してGo版に切り替える。

- `GET /s/:space_identifier/pages/:page_number` - ページ表示（本タスクのスコープ外、Rails版にルーティング）
- `GET /go/s/:space_identifier/pages/:page_number/edit` - ページ編集画面
- `PATCH /go/s/:space_identifier/pages/:page_number/draft_page` - 下書き更新（自動保存）
- `PATCH /go/s/:space_identifier/pages/:page_number` - ページ公開
- `GET /go/s/:space_identifier/page_locations?q=:keyword` - ページロケーション検索（Wikiリンク補完用）

### Go版の実装設計

#### 初回スコープ

本タスクで実装する範囲:

- ページ編集画面（タイトル・本文、CodeMirrorエディタ）
- 下書き自動保存（DraftPageの作成・更新、500msデバウンス）
- ページ公開（Page更新、PageRevision作成、PageEditor追加、DraftPage削除、TopicMemberのlast_page_modified_at更新）
- 添付ファイル参照追跡・アイキャッチ画像抽出（公開時にbodyHtmlから添付ファイルIDを検出し、PageAttachmentReferenceを同期。bodyの1行目から画像IDを抽出してfeatured_image_attachment_idを設定）
- Markdownレンダリング（goldmark + bluemonday + 添付ファイルフィルター）
- Wikiリンク解析・自動ページ作成・HTML変換（`[[ページ名]]`および`[[トピック名/ページ名]]`のパース、リンク先ページの自動作成、HTML `<a>`タグへの変換、`linkedPageIds`の更新）
- Wikiリンク補完（`[[`入力時にスペース内のページ名を候補表示するCodeMirror拡張、バックエンドのページロケーション検索API）
- エディタのファイルアップロード（ドラッグ&ドロップ、ペースト、プレースホルダー表示、Markdownリンク挿入）
- リンク一覧・バックリンク表示（編集画面フッターに表示。`linkedPageIds`に基づくリンク先ページ一覧と、バックリンク一覧）
- トピックアクセス制御（トピックの可視性とトピックメンバーシップに基づくページ編集権限チェック）

以下は別タスクに委譲し、初回スコープには含めない（各作業計画書は「関連タスク」セクションを参照）:

- [ページ表示画面のGo移行](../2_todo/page-show-go-migration.md) — ページ詳細画面の表示はリバースプロキシでRails版にルーティングする
- [ページのOGPメタタグ設定](../2_todo/page-ogp-meta.md)

#### ハンドラー構成

```
internal/handler/
├── page/
│   ├── handler.go      # Handler構造体と依存性
│   ├── edit.go         # GET /go/s/:space_identifier/pages/:page_number/edit
│   ├── update.go       # PATCH /go/s/:space_identifier/pages/:page_number（公開）
│   └── validator.go    # ページ公開のバリデーション（タイトル重複チェック等）
├── draft_page/
│   ├── handler.go      # Handler構造体と依存性
│   └── update.go       # PATCH /go/s/:space_identifier/pages/:page_number/draft_page（自動保存）
└── page_location/
    ├── handler.go      # Handler構造体と依存性
    └── index.go        # GET /go/s/:space_identifier/page_locations?q=:keyword（Wikiリンク補完用）
```

#### 新規パッケージ

**モデル** (`internal/model/`):

- `space.go` — ID, Identifier, Name, Plan, JoinedAt, DiscardedAt
- `space_member.go` — ID, SpaceID, UserID, Role, JoinedAt, Active
- `topic.go` — ID, SpaceID, Number, Name, Description, Visibility, DiscardedAt
- `page.go` — 設計セクションのデータ構造を実装
- `draft_page.go` — 同上
- `page_revision.go` — 同上
- `page_editor.go` — ID, SpaceID, PageID, SpaceMemberID, LastPageModifiedAt
- `page_attachment_reference.go` — ID, AttachmentID, PageID
- `topic_member.go` — ID, SpaceID, TopicID, SpaceMemberID, Role, JoinedAt, LastPageModifiedAt
- `attachment.go` — ID, SpaceID, Filename（添付ファイル存在確認・ファイル種別判定用のモデル）

**ポリシー** (`internal/policy/`):

- `topic.go` — TopicPolicy構造体（CanUpdatePage, CanUpdateDraftPage メソッド）

**リポジトリ** (`internal/repository/`):

- `space.go` — FindByIdentifier
- `space_member.go` — FindActiveBySpaceAndUser
- `topic.go` — FindBySpaceAndNumber, ListActiveBySpace, FindBySpaceAndNames
- `topic_member.go` — FindBySpaceMemberAndTopic, UpdateLastPageModifiedAt
- `page.go` — FindBySpaceAndNumber, FindByIDs, FindBacklinkedByPageID, Update, FindByTopicAndTitle, CreateLinkedPage, SearchPageLocations
- `draft_page.go` — FindByPageAndMember, Create, Update, Delete
- `page_revision.go` — Create
- `page_editor.go` — FindOrCreate
- `page_attachment_reference.go` — ListByPage, CreateBatch, DeleteByPageAndAttachmentIDs
- `attachment.go` — ExistsByIDAndSpace, FindByIDAndSpace（AttachmentFilter用にID・Filename取得）

**ユースケース** (`internal/usecase/`):

- `auto_save_draft_page.go` — 下書き自動保存（find_or_create + Markdownレンダリング + 添付ファイルフィルター + Wikiリンク解析・自動ページ作成 + 更新）
- `publish_page.go` — ページ公開（トランザクション内でPage更新、PageRevision作成、PageEditor追加、DraftPage削除、TopicMemberのlast_page_modified_at更新、Wikiリンク解析・自動ページ作成、添付ファイルフィルター、添付ファイル参照同期、アイキャッチ画像抽出。成功時はフラッシュメッセージを設定しページ表示画面にリダイレクト）

**Markdownレンダリング** (`internal/markup/`):

- `markup.go` — goldmarkでMarkdown→HTML変換、bluemondayでHTMLサニタイズ。レンダリングパイプライン: Markdown→HTML変換 → サニタイズ → Wikiリンク変換 → 添付ファイルフィルター → 後処理（単独画像リンクのラッピング）
- `attachment_filter.go` — 添付ファイルフィルター（Rails版の`Markup::AttachmentFilter`に相当。画像・動画・ダウンロードリンクの変換）
- `attachment_extract.go` — 添付ファイルID抽出、アイキャッチ画像ID抽出
- `wikilink.go` — Wikiリンクのパース（`[[ページ名]]`、`[[トピック名/ページ名]]`）、HTML `<a>`タグへの変換

#### Markdownレンダリング

Rails版は`Markup`クラスでMarkdown→HTML変換を行っている。Go版では以下のライブラリを使用する:

- **goldmark**: Markdown→HTML変換（GitHub Flavored Markdown対応）
- **bluemonday**: HTMLサニタイズ（XSS対策）

**Markdownオプション**:

Rails版では`html: true`（HTMLタグの解析を有効化）と`unsafe: true`（HTMLタグのレンダリングを有効化）を設定しており、Markdown本文中にHTMLタグを直接記述可能。Go版のgoldmarkでも`html.WithUnsafe()`オプションで同等の動作を実現する。

**HTMLサニタイズの設定**:

Rails版の`sanitization_config`では以下のカスタマイズを行っている。Go版のbluemondayでも同等の設定を行う:

- `input`要素を許可（タスクリスト記法 `- [ ]` のチェックボックス表示に必要）
- `img`要素に`width`属性と`height`属性を許可（アップロード画像のサイズ指定に使用）

**添付ファイルフィルター（AttachmentFilter）**:

Rails版は`Markup::AttachmentFilter`で、bodyHTML内の`/attachments/{id}`パターンのURLを持つ`<img>`タグと`<a>`タグを変換する。この変換はHTMLレンダリングの最終段階で行われ、署名付きURLの生成はクライアント側のJavaScriptに委譲する（プレースホルダー方式）。Go版でも同等のHTML変換ロジックを実装する。

変換ルール:

| 対象要素                        | ファイル種別                                     | 変換結果                                                                                                                                                                                                           |
| ------------------------------- | ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `<img src="/attachments/{id}">` | インライン画像（jpg, jpeg, png, gif, svg, webp） | `<a class="wikino-attachment-image-link" href="#" data-attachment-id="{id}" data-attachment-link="true"><img src="" data-attachment-id="{id}" data-attachment-type="image" class="wikino-attachment-image" /></a>` |
| `<img src="/attachments/{id}">` | 非画像ファイル                                   | ダウンロードリンク（SVGアイコン付き`<a>`タグ、`data-attachment-id`・`data-attachment-link`属性付き）                                                                                                               |
| `<a href="/attachments/{id}">`  | 動画（mp4, webm, ogg, mov）                      | `<video src="" data-attachment-id="{id}" data-attachment-type="video" class="wikino-attachment-video" controls>`                                                                                                   |
| `<a href="/attachments/{id}">`  | その他                                           | `<a href="#" data-attachment-id="{id}" data-attachment-link="true" target="_blank">`                                                                                                                               |

- 署名付きURLは`src=""`・`href="#"`のプレースホルダーで出力し、フロントエンドのJavaScriptが`data-attachment-id`属性を基に署名付きURLに差し替える
- 添付ファイルの存在確認: 同じスペースの`attachments`テーブルにレコードが存在するか確認し、存在しない場合は変換をスキップ
- ファイル種別の判定: `AttachmentRecord`の`filename`から拡張子を取得して判定

**単独画像リンクの`<p>`要素ラッピング**:

Rails版ではHTMLパイプラインの後処理として、`<p>`要素で囲まれていない単独の画像リンク（`<a class="wikino-attachment-image-link">`）を`<p>`要素で囲む処理を行っている。Go版でも同等の後処理を実装する。

**単独`<img>`タグのゼロ幅非接合子ワークアラウンド**:

Rails版では、Markdown本文中に単独行で記述された`<img>`タグがMarkdownパーサーによってHTMLブロックとして扱われる問題を回避するため、`<img>`タグの前にゼロ幅非接合子（`\u200C`）を挿入し、レンダリング後に除去する処理を行っている。Go版のgoldmarkでも同様の問題が発生する場合は同等のワークアラウンドを実装する。

**バッチレンダリング（将来的な最適化）**:

Rails版には`render_html_batch`メソッドがあり、複数テキストのWikiリンクを一括でDB検索してN+1問題を回避している。Go版でも将来的にバッチ処理が必要になった場合に対応できるよう、レンダリングパイプラインを設計する。初回スコープでは単一テキストのレンダリングのみ実装する。

#### Wikiリンク解析・自動ページ作成・HTML変換

Rails版の`Pageable#link!`メソッドと`Markup::PageLinkFilter`に相当する機能をGo版で実装する。

**Wikiリンクの形式**:

| 形式                      | 例                | 意味                             |
| ------------------------- | ----------------- | -------------------------------- |
| `[[ページ名]]`            | `[[設計書]]`      | 現在のトピック内のページにリンク |
| `[[トピック名/ページ名]]` | `[[開発/設計書]]` | 指定トピック内のページにリンク   |

**パース処理**:

Markdown本文（body）から正規表現`\[\[(.*?)\]\]`でWikiリンクを抽出する。抽出した文字列を`/`で分割（最大2分割）し:

- `トピック名/ページ名`形式の場合: トピック名とページ名をそのまま使用
- `ページ名`のみの場合: 現在のトピック名をトピック名として使用

**自動ページ作成（リンク処理）**:

パースしたWikiリンクごとに:

1. スペース内で指定トピック名のトピックを検索する
2. トピックが存在する場合、そのトピック内で指定タイトルのページをfirst_or_create方式で取得・作成する
3. 新規作成されたページは空のbody/bodyHtmlと空のlinkedPageIdsを持つ
4. 新規作成されたページにはPageEditorレコードも作成する
5. 指定トピックが存在しない場合、そのWikiリンクはスキップする（リンクは作成されない）
6. 全てのリンク先ページIDを収集し、Page/DraftPageの`linkedPageIds`を更新する

**HTML変換**:

bodyHtmlの生成時に、WikiリンクをHTML `<a>`タグに変換する:

1. bodyからWikiリンクをパースする
2. パースしたキーに対応するページをDBから一括取得する
3. bodyHtml内の`[[トピック名/ページ名]]`または`[[ページ名]]`を以下のHTMLに置換する:
   - ページが存在する場合: `<a href="/s/{space_identifier}/pages/{page_number}">{ページタイトル}</a>`
   - ページが存在しない場合: `[[...]]`のまま（プレーンテキストとして残す）
4. `<a>`, `<code>`, `<pre>`, `<script>`, `<style>`タグ内のWikiリンクは変換しない

**処理タイミング**:

- **自動保存時**: DraftPageのbodyからWikiリンクを解析し、自動ページ作成とlinkedPageIds更新を行う。bodyHtmlにもWikiリンクのHTML変換を含める
- **公開時**: PageのbodyからWikiリンクを解析し、自動ページ作成とlinkedPageIds更新を行う。bodyHtmlにもWikiリンクのHTML変換を含める

**実装の配置**:

- `internal/markup/wikilink.go` — Wikiリンクのパース（`ScanWikilinks`関数）、HTML変換（`ReplaceWikilinks`関数）
- `internal/usecase/auto_save_draft_page.go` — 自動保存時のWikiリンク処理を追加
- `internal/usecase/publish_page.go` — 公開時のWikiリンク処理を追加
- `internal/repository/page.go` — `FindByTopicAndTitle`、`CreateLinkedPage`を追加
- `internal/repository/topic.go` — `FindBySpaceAndNames`を追加

#### 添付ファイル参照追跡・アイキャッチ画像抽出

ページ公開時に、本文中の添付ファイル参照を追跡し、アイキャッチ画像を抽出する。Rails版の`PageRecord#update_attachment_references!`と`PageRecord#extract_featured_image_id`に相当する処理をGo版で実装する。

**添付ファイルID検出ロジック**:

bodyHtml（HTML変換後の本文）から以下の4パターンで添付ファイルIDを検出する:

| パターン       | 正規表現                                            | マッチ例                          |
| -------------- | --------------------------------------------------- | --------------------------------- |
| HTML imgタグ   | `<img[^>]+src=["']/attachments/([^/"']+)["'][^>]*>` | `<img src="/attachments/abc123">` |
| HTML aタグ     | `<a[^>]+href=["']/attachments/([^/"']+)["'][^>]*>`  | `<a href="/attachments/abc123">`  |
| Markdown画像   | `!\[[^\]]*\]\(/attachments/([^/)]+)\)`              | `![alt](/attachments/abc123)`     |
| Markdownリンク | `(?<!!)\\[[^\]]+\\]\(/attachments/([^/)]+)\)`       | `[text](/attachments/abc123)`     |

検出されたIDに対して:

- 同じスペースの添付ファイルが存在するか確認する（`attachments`テーブル）
- 存在する場合のみ`page_attachment_references`に参照レコードを作成する
- 前回の公開時から削除された参照は`page_attachment_references`から削除する（差分同期）

**アイキャッチ画像抽出ロジック**:

Markdown本文（body）の1行目から画像IDを抽出する:

1. bodyの1行目を取得（空行はスキップしない、1行目が空ならnull）
2. Markdown画像形式をチェック: `![alt](/attachments/{id})`
3. HTML img形式をチェック: `<img src="/attachments/{id}">`
4. マッチしたIDの添付ファイルが同じスペースに存在するか確認する
5. 存在すればPageの`featured_image_attachment_id`に設定、存在しなければnullに設定

**実装の配置**:

抽出ロジックは`internal/markup/`パッケージ内に配置する（Markdownレンダリングと関連するため）:

- `markup.go` — Markdown→HTML変換、HTMLサニタイズ（既存）
- `attachment.go` — 添付ファイルID抽出、アイキャッチ画像ID抽出

#### リンク一覧・バックリンク表示

ページ編集画面のフッターに、リンク一覧（そのページからリンクしている他ページ）とバックリンク一覧（そのページにリンクしている他ページ）を表示する。Rails版では`LinkListRepository`と`BacklinkListRepository`で実装されている機能をGo版で再実装する。

**リンク一覧**:

- 編集中のページ（DraftPage存在時はDraftPage、なければPage）の`linkedPageIds`を基に、リンク先ページの一覧を取得・表示する
- `pages`テーブルから`linkedPageIds`に含まれるIDのページを取得する
- 同じスペース内の公開済み（`published_at IS NOT NULL`）かつ未廃棄（`discarded_at IS NULL`）のページのみ表示する

**バックリンク一覧**:

- `pages`テーブルの`linked_page_ids`カラムにこのページのIDが含まれるページの一覧を取得・表示する（常にPageの公開済みデータを基にする）
- 同じスペース内の公開済みかつ未廃棄のページのみ表示する

**Rails版との違い**:

- Rails版ではTurbo Streamで自動保存のたびにフッターを動的更新している。Go版では[Datastar](https://data-star.dev/)を使用して、自動保存時にリンク一覧・バックリンク一覧を動的に更新する
- Rails版ではリンク先ページのバックリンク（ネスト表示）もあるが、初回スコープではフラットな一覧のみ表示する
- ページネーションは初回スコープに含めない

**Datastarによるリアルタイム更新**:

- 下書き自動保存のレスポンスにリンク一覧・バックリンク一覧のHTMLフラグメントを含める
- Datastarのシグナルとイベントを使用して、自動保存完了時にフッター部分を差し替える
- CodeMirrorエディタでの動作確認ができてからリンク・バックリンク表示を実装する（フェーズ 8b で実装）

#### 自動保存のレスポンス形式

Rails版ではTurbo Streamで部分更新（保存時刻表示、フッターのリンク一覧更新）を行っている。Go版ではDatastarを使用してフッターのリンク一覧・バックリンク一覧を動的に更新する。

自動保存のレスポンスはDatastarのSSEイベント形式で返す。Datastarのマージモードを使用して、保存時刻の更新とフッター（リンク一覧・バックリンク一覧）の差し替えを行う。具体的なレスポンス形式はDatastarのドキュメントを参照しながらフェーズ 8b で設計する。

#### フロントエンド

Rails版のStimulus + CodeMirror構成をGo版のpnpm/esbuild環境に移植する。

**移植するもの**:

- CodeMirror 6エディタの初期化（Markdown構文ハイライト、履歴、括弧マッチング）
- 自動保存コントローラー（500msデバウンス、fetch APIでPATCHリクエスト送信）
- キーバインド（Enter: リスト続行、Tab/Shift-Tab: インデント、Cmd/Ctrl+Enter: フォーム送信）
- エディタ内容とhidden textareaの同期
- Wikiリンク補完（`wikilink-completions.ts`）
- ファイルドラッグ&ドロップ（`file-drop-handler.ts`）
- ペーストによるファイルアップロード（`paste-handler.ts`）
- ファイルアップロードハンドラー（`file-upload-handler.ts`）
- アップロードプレースホルダー管理（`upload-placeholder.ts`）
- S3直接アップロード（`direct-upload.ts`）

#### Wikiリンク補完

ページ編集画面のCodeMirrorエディタで`[[`を入力するとスペース内の既存ページ名を候補として表示し、選択するとWikiリンクが自動補完される機能。Rails版の`wikilink-completions.ts`とPageLocations::IndexControllerに相当する。

**バックエンド: ページロケーション検索API**:

`GET /go/s/:space_identifier/page_locations?q=:keyword`

- 認証・認可: ログイン必須、スペースメンバーであること（`joined_space?`チェック）
- **トピック可視性フィルタリング**: Rails版では`space_policy.showable_pages(space_record:)`を使用しており、スペースオーナー・メンバーは全アクティブページを閲覧可能。Go版でも同等のフィルタリングを行う（スペースメンバーは全アクティブページを検索可能）
- クエリパラメータ`q`をスペース区切りで分割し、各ワードに対してページタイトルをILIKE（大文字小文字を区別しない部分一致）で検索する（AND条件）
- 公開済み（`published_at IS NOT NULL`）かつ未廃棄（`discarded_at IS NULL`）のページのみ対象
- `modified_at`降順でソートし、最大10件を返す
- レスポンス形式:

```json
{
  "page_locations": [{ "key": "トピック名/ページタイトル" }]
}
```

**フロントエンド: Wikiリンク補完拡張**:

- `wikilink-completions.ts` — CodeMirrorの`autocompletion`拡張のoverrideとして登録
- `[[`の入力を正規表現`/\[\[.*/`で検出し、`[[`以降のテキストをキーワードとして補完候補を取得
- `fetch`でページロケーション検索APIを呼び出し、結果をCodeMirrorの補完候補に変換
- `filter: false`を設定し、サーバーサイドでフィルタリング済みの結果をそのまま表示
- 補完候補のlabel: `[[トピック名/ページタイトル`、displayLabel: `トピック名/ページタイトル`

**フロントエンドのファイル構成**:

| ファイル                  | 責務                                                          |
| ------------------------- | ------------------------------------------------------------- |
| `wikilink-completions.ts` | CodeMirror補完拡張。`[[`入力検出、API呼び出し、補完候補の構築 |

#### ファイルアップロードの設計

エディタ内でのファイルアップロード機能。ドラッグ&ドロップとペーストの2つの入力方法をサポートする。

**アップロードフロー**:

1. ユーザーがファイルをドラッグ&ドロップまたはペースト
2. エディタにプレースホルダー（`<!-- Uploading "fileName"... -->`）を挿入
3. クライアント側でファイルバリデーション（サイズ、MIMEタイプ、画像サイズ）
4. MD5チェックサムを計算
5. Rails版のPresignエンドポイント（`/s/:space_identifier/attachments/presign`）にリクエストし、署名付きアップロードURLを取得
6. S3互換ストレージに直接アップロード（XMLHttpRequestでプログレス追跡）
7. アップロード完了後、プレースホルダーをMarkdownリンクまたはimgタグに置換
8. エラー時はプレースホルダーを削除し、トースト通知を表示

**Presignエンドポイント**:

添付ファイルのPresignエンドポイント（`/s/:space_identifier/attachments/presign`）と表示エンドポイント（`/attachments/:id`）は引き続きRails版にリバースプロキシでルーティングする。Go版での再実装は本タスクのスコープ外。

**ファイルバリデーション**:

| ファイル種別 | サイズ上限 |
| ------------ | ---------- |
| 画像         | 10 MB      |
| 動画         | 100 MB     |
| その他       | 25 MB      |

- MIMEタイプ: ホワイトリスト方式（画像、動画、PDF、Office文書、テキスト、アーカイブ等）
- 画像サイズ: 最大 10,000 × 10,000 ピクセル

**Markdown出力**:

- 画像: `<img width="{width}" alt="{fileName}" src="/attachments/{id}">` （heightは省略しアスペクト比を維持）
- その他: `[{fileName}](/attachments/{id})`

**フロントエンドのファイル構成**:

| ファイル                 | 責務                                                                                                    |
| ------------------------ | ------------------------------------------------------------------------------------------------------- |
| `file-drop-handler.ts`   | CodeMirror ViewPlugin。ドラッグ&ドロップイベントの検出、ドロップゾーンの表示、カーソル位置の計算        |
| `paste-handler.ts`       | クリップボードのペーストイベントを検知し、MIMEタイプに応じてカスタムイベントをディスパッチ              |
| `file-upload-handler.ts` | アップロードのオーケストレーター。バリデーション、チェックサム計算、Presignリクエスト、アップロード実行 |
| `upload-placeholder.ts`  | アップロード中のプレースホルダーテキストの挿入・追跡・置換・削除                                        |
| `direct-upload.ts`       | XMLHttpRequestラッパー。S3互換ストレージへのPUTリクエストとプログレス追跡                               |

#### レイアウト

現在Go版には`Plain`（ウェルカムページ用）と`Simple`（認証ページ用）の2つのレイアウトのみ存在する。ページ表示・編集にはスペースのコンテキストを持つレイアウトが必要。

スペースページ用の簡易レイアウトを新規作成する:

- スペース名のヘッダー表示
- メインコンテンツエリア
- 基本的なフッター

フルナビゲーション（サイドバー、検索バー等）は別タスクで実装する。

#### 認証・認可

- **認証**: `RequireAuth`ミドルウェアを使用（認証必須）。ハンドラー内で`SpaceMemberRepository.FindActiveBySpaceAndUser`によりアクティブなスペースメンバーであることを確認
- **認可（トピックアクセス制御）**: ページの編集・下書き保存・公開時に、トピックレベルの権限チェックを行う

**トピックアクセス制御の設計**:

Rails版では`TopicPolicy`パターン（Factory + ロール別ポリシークラス）で実装されている。Go版でも今後各リソースの権限チェックが増えていくことを見据え、**インターフェース + ロール別構造体 + ファクトリ関数**のパターンで実装する。

**設計方針**:

- `TopicPolicy`をインターフェースとして定義し、ロール別の具象構造体（`topicOwnerPolicy`, `topicAdminPolicy`, `topicMemberPolicy`, `topicGuestPolicy`）で実装する
- ファクトリ関数`NewTopicPolicy`が`SpaceMemberRole`、`spaceID`、`*TopicMember`、`active bool`を受け取り、適切なポリシーを返す
- `active`パラメータはSpaceMemberのアクティブ状態を示す。各ポリシー（guest以外）はメソッド内で`active`をチェックし、非アクティブの場合はすべての操作を拒否する
- ポリシーは**純粋なロジック**（DBアクセスなし）。データ取得はハンドラー層で事前に行い、ポリシーにはモデルのみを渡す
- `CanUpdatePage(*model.Page)` / `CanUpdateDraftPage(*model.DraftPage)` でページのスペース・トピックとの一致をポリシー内で検証する
- これにより、一覧表示時にバッチクエリで`TopicMember`を取得した後、メモリ上でポリシーを構築でき、N+1問題を回避できる

**ファイル構成** (`internal/policy/`):

| ファイル          | 内容                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------ |
| `topic.go`        | `TopicPolicy`インターフェース定義 + `NewTopicPolicy`ファクトリ関数                         |
| `topic_owner.go`  | `topicOwnerPolicy`構造体（スペースオーナー用、同スペース内の全トピックのページを編集可能） |
| `topic_admin.go`  | `topicAdminPolicy`構造体（トピックAdmin用、所属トピックのページを編集可能）                |
| `topic_member.go` | `topicMemberPolicy`構造体（トピックMember用、所属トピックのページを編集可能）              |
| `topic_guest.go`  | `topicGuestPolicy`構造体（非メンバー用、編集不可）                                         |
| `topic_test.go`   | 全ロール・全メソッドのテスト                                                               |

**インターフェース定義と主要な型**:

```go
// internal/policy/topic.go
package policy

// TopicPolicy はトピック内のリソースに対する権限を判定するインターフェース
type TopicPolicy interface {
	CanUpdatePage(page *model.Page) bool
	CanUpdateDraftPage(draftPage *model.DraftPage) bool
}

// NewTopicPolicy はスペースメンバーのロールとトピックメンバー情報から適切なポリシーを生成する
// active はスペースメンバーが有効かどうかを示す（SpaceMember.Active に対応）
func NewTopicPolicy(spaceMemberRole model.SpaceMemberRole, spaceID string, topicMember *model.TopicMember, active bool) TopicPolicy {
	if spaceMemberRole == model.SpaceMemberRoleOwner {
		return &topicOwnerPolicy{spaceID: spaceID, active: active}
	}
	if topicMember == nil {
		return &topicGuestPolicy{}
	}
	if topicMember.Role == model.TopicMemberRoleAdmin {
		return &topicAdminPolicy{topicID: topicMember.TopicID, active: active}
	}
	return &topicMemberPolicy{topicID: topicMember.TopicID, active: active}
}
```

**Rails版とのマッピング**:

| Rails版                      | Go版                                |
| ---------------------------- | ----------------------------------- |
| `TopicPermissions`モジュール | `TopicPolicy`インターフェース       |
| `TopicPolicyFactory`         | `NewTopicPolicy`ファクトリ関数      |
| `TopicOwnerPolicy`クラス     | `topicOwnerPolicy`構造体（非公開）  |
| `TopicAdminPolicy`クラス     | `topicAdminPolicy`構造体（非公開）  |
| `TopicMemberPolicy`クラス    | `topicMemberPolicy`構造体（非公開） |
| `TopicGuestPolicy`クラス     | `topicGuestPolicy`構造体（非公開）  |

**判定ロジックの詳細**（Rails版`TopicPermissions`に準拠）:

| ロール             | ページ編集 | 下書き更新 | 判定条件                                             |
| ------------------ | ---------- | ---------- | ---------------------------------------------------- |
| スペースオーナー   | ✅         | ✅         | `active && spaceID == page.SpaceID`                  |
| トピックAdmin      | ✅         | ✅         | `active && topicID == page.TopicID`                  |
| トピックMember     | ✅         | ✅         | `active && topicID == page.TopicID`                  |
| 非トピックメンバー | ❌         | ❌         | 常に`false`（topicMemberがnilの場合）                |
| 非スペースメンバー | ❌         | ❌         | スペースメンバーでない場合はハンドラー層で事前に拒否 |

**N+1問題への対策**:

ポリシーはDBアクセスを行わない純粋なロジックであるため、一覧表示時は以下の流れでN+1を回避する:

1. バッチクエリで対象トピックの`TopicMember`レコードをまとめて取得
2. メモリ上で各ページに対応する`TopicPolicy`を`NewTopicPolicy`で構築
3. 各ポリシーで権限チェック

**ハンドラーでの使用**:

各ハンドラー（edit.go, draft_page/update.go, page/update.go）で以下の流れで認可チェックを行う:

1. `SpaceMemberRepository.FindBySpaceAndUser`でスペースメンバーを取得（存在しない場合は404）
2. `TopicMemberRepository.FindBySpaceMemberAndTopic`でトピックメンバーを取得（存在しない場合はnil）
3. `NewTopicPolicy(role, spaceID, topicMember, spaceMember.Active)`でポリシーを生成し、`CanUpdatePage(page)`/`CanUpdateDraftPage(draftPage)`で権限チェック
4. 権限がない場合は404を返す（Rails版と同じ挙動）

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### TopicPolicyをシンプルな構造体で実装する方式

Go版のTopicPolicyを単一の構造体（`TopicPolicy`）にロールの判定ロジックとメソッドをすべて集約する方式を検討した。YAGNI原則に従い、最もシンプルな実装として有力だった。

**不採用の理由**:

- Rails版の権限チェック改善（[permission-improvement.md](/workspace/docs/plans/3_done/202508/permission-improvement.md)）で、ロール別にポリシークラスを分けるFactory + Policyパターンが有効であることが実証されている
- 今後各リソース（ページ、トピック、スペースなど）の権限チェックが増えていくことが明らかであり、単一構造体では条件分岐が肥大化する
- インターフェース + ロール別構造体のパターンであれば、新しいロールやリソースの追加時にOpen-Closed Principleに従った拡張が可能
- 具象構造体を非公開（小文字）にすることで、外部からはインターフェース経由のみのアクセスに制限でき、Goの慣習に沿った設計になる
- ポリシーをDBアクセスなしの純粋ロジックとすることで、N+1問題のリスクをアーキテクチャレベルで排除できる

### ページタイトルの変更を独立した操作として提供する方式

[タイトル変更時のWikiリンク自動書き換え](title-change-link-rewrite.md) の「採用しなかった方針」セクションを参照。

### 自動保存のたびにDraftPageRevisionを作成する方式

[DraftPageRevisionの実装](draft-page-revision.md) の「採用しなかった方針」セクションを参照。

### UI上で「リビジョン」という表示名を使用すること

内部モデル名の `PageRevision` をそのままカタカナ表記した「リビジョン」をUI上の表示名として使用することを検討したが、以下の理由から採用しなかった。

- 「リビジョン」はソフトウェアエンジニアには馴染みのある用語だが、非技術者にとっては見慣れない専門用語である
- Wikinoは非技術者でもGit的なワークフローを利用できることを目指しており、UIの用語もわかりやすさを優先すべきである
- 「バージョン」はソフトウェアに限らず広く使われている一般的なカタカナ語であり、「ある時点の状態」というPageRevisionの本質を十分に表現できる
- 一覧表示には「編集履歴」を採用した。「バージョン一覧」よりも「編集履歴」のほうが「過去の変更を振り返る」という操作の意図に合致する

### DraftPageRevision作成操作を「保存 (Save)」と呼ぶこと

DraftPageRevisionを作成する操作の名称として「保存 (Save)」を検討したが、以下の理由から「下書き保存 (Save Draft)」を採用した。

- 「保存」とだけ書くと、他の人にも見える形で保存されるというニュアンスに受け取るユーザーがいる可能性がある
- 「下書き保存」であれば「自分だけの下書き領域に保存する」という意図が名前から明確に伝わる
- 「自動保存」「下書き保存」「公開」の3段階において、「下書き保存」は中間のステップであることが名前から読み取れる

### `Revision` / `DraftRevision` という名称

当初は公開版のリビジョンを `Revision`、下書き版を `DraftRevision` と呼んでいたが、既存のDBテーブル名 `page_revisions` に合わせて `PageRevision` / `DraftPageRevision` に変更した。`Page` プレフィックスを付けることで、将来的に他のモデル（例: スペース設定など）にもリビジョン管理を導入する場合に名前の衝突を避けられる。

### `DraftSnapshot` という名称

下書きのリビジョンの名称として `DraftSnapshot` を検討したが、公開版の `PageRevision` と「Snapshot」という類似した別の用語が並存すると、両者の違いが分かりにくくなる。`DraftPageRevision` であれば「どちらもリビジョン（ある時点の記録）だが、公開/非公開のスコープが違う」という関係が名前から読み取れるため、`DraftPageRevision` を採用した。

### `WorkingContent` という名称

「作業中の内容」という意味だが、何に対する作業なのかが名前から読み取りにくい。「下書き（Draft）」のほうが、公開前の暫定版という状態を端的に表現でき、UIで「下書きのページ」と表示する際にも自然に対応するため、`DraftPage` を採用した。

### `DraftContent` という名称

当初は `DraftContent`（下書きの内容）という名称を使用していたが、実態として`linkedPageIds`や`topicId`などページとしての属性を持つモデルであるため、`DraftPage`に変更した。DBテーブル名が`draft_pages`であることとも一致する。

### `EditingContent` という名称

「編集中の内容」という意味だが、ユーザーがブラウザを閉じて後日再開する場合など、実際には編集操作をしていない状態でもデータは残り続ける。「編集中」は動作の状態を表す言葉であり、永続化されたデータの名前としては不正確であるため採用しなかった。`DraftPage` の「下書き」は動作ではなく状態を表す名詞であり、ユーザーが操作していない間もデータの性質を正しく表現できる。

### リンク一覧・バックリンクをページ初回表示時のみ静的に表示する方式

当初はGo版ではTurboを使用しないため、ページ初回表示時のみフッターにリンク一覧・バックリンクを静的に表示し、自動保存時の動的更新は行わない方針を検討した。

**不採用の理由**:

- Rails版では自動保存のたびにリンク一覧が動的に更新されており、ユーザーが編集中にリンク先の確認をリアルタイムで行えるUXが重要である
- [Datastar](https://data-star.dev/)を使用することで、Turboと同等の部分更新をGo版でも実現できる
- Datastarはhtml/templateやtemplとの親和性が高く、Go版のアーキテクチャに適合する
- 静的表示ではWikiリンクを記述しても結果がページリロードまで反映されず、編集体験が劣化する

### E2EテストでGoプロセスからPlaywrightを起動する方式

E2Eテストのテストデータ作成にGo版の`testutil`パッケージを流用するため、以下の方式を検討した。

1. **playwright-go（Go用Playwrightバインディング）を使用**: Goテスト内でブラウザを直接操作
2. **Goプロセスから`npx playwright test`を子プロセスとして起動**: テストデータ作成はGoで行い、Playwrightは子プロセスで実行
3. **テスト専用APIを用意**: Go側にテストデータ作成用のHTTPエンドポイントを実装し、Playwrightテストからそれを呼び出す

**不採用の理由**:

- **トランザクション分離が不可能**: E2Eテストではブラウザ → アプリサーバー → DBという経路でデータにアクセスするため、テストプロセスのトランザクション内で作成したデータはアプリサーバーから見えない。Go testutilの`SetupTx(t)`によるトランザクション分離・自動ロールバックのメリットが活かせない
- **playwright-go**: コミュニティライブラリであり、Playwright公式（Node.js）の新機能・修正への追従が遅れる可能性がある
- **子プロセス方式**: テストデータの値を環境変数でPlaywrightに受け渡す必要があり、煩雑になる
- **テスト専用API**: テストデータ作成APIは認証なしでデータを自由に操作できるエンドポイントとなり、本番環境への混入リスクがある。また、テストAPIはバリデーションをスキップする必要があるため公開APIとの流用範囲が限定的で、追加の実装・保守コストに見合わない

**採用した方式（Node.js + DB直接操作）の設計判断**:

- `go/e2e/helpers/database.ts`でPostgreSQLに直接アクセスしてテストデータを作成・削除する
- Go版の`testutil`ビルダーパターンと同等のINSERT文をTypeScriptで実装する
- テスト後はDELETEで明示的にクリーンアップする（トランザクション分離ではなく）
- Playwright公式のNode.jsサポートを最大限活用できる

**トレードオフ**:

- `database.ts`とGo版`testutil`の二重管理が必要（DBスキーマ変更時に両方を更新する必要がある）
- `database.ts`のINSERT文はバリデーションをバイパスするため、本番では存在し得ないデータを作成するリスクがある。バリデーションの正しさは単体・統合テスト（Go側）で保証し、E2Eテストではテストデータの作成を「前提条件の準備」として扱う
- ただし、`database.ts`のINSERT文がDBスキーマと乖離した場合、Go側のバリデーションテストを修正しても`database.ts`は直らないため、E2Eテストが不正なデータで実行される可能性がある。これはDB直接操作方式の固有のリスクとして認識している

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

### フェーズ 1: 基盤モデルとリポジトリ

ページ機能の前提となるSpace・SpaceMember・Topic・TopicMemberのモデル・クエリ・リポジトリとトピックアクセス制御ポリシーを追加する。

- [x] **1-1**: [Go] Space モデル・クエリ・リポジトリの追加
  - `internal/model/space.go` を追加（ID, Identifier, Name, Plan, JoinedAt, DiscardedAt）
  - `internal/query/queries/spaces.sql` を追加（FindByIdentifier）
  - `internal/repository/space.go` を追加（FindByIdentifier）
  - `internal/testutil/space_builder.go` を追加
  - 想定ファイル数: 実装 4 ファイル（+ sqlc生成 1 ファイル）, テスト 2 ファイル
  - 想定行数: 実装 ~120 行, テスト ~100 行
  - 依存: なし

- [x] **1-2**: [Go] SpaceMember モデル・クエリ・リポジトリの追加
  - `internal/model/space_member.go` を追加（ID, SpaceID, UserID, Role, JoinedAt, Active）
  - `internal/query/queries/space_members.sql` を追加（FindActiveBySpaceAndUser）
  - `internal/repository/space_member.go` を追加（FindActiveBySpaceAndUser）
  - `internal/testutil/space_member_builder.go` を追加
  - 想定ファイル数: 実装 4 ファイル（+ sqlc生成 1 ファイル）, テスト 2 ファイル
  - 想定行数: 実装 ~130 行, テスト ~100 行
  - 依存: 1-1

- [x] **1-3**: [Go] Topic モデル・クエリ・リポジトリの追加
  - `internal/model/topic.go` を追加（ID, SpaceID, Number, Name, Description, Visibility, DiscardedAt）
  - `internal/query/queries/topics.sql` を追加（FindBySpaceAndNumber, ListActiveBySpace, FindBySpaceAndNames）
  - `internal/repository/topic.go` を追加（FindBySpaceAndNumber, ListActiveBySpace, FindBySpaceAndNames）
  - `internal/testutil/topic_builder.go` を追加
  - FindBySpaceAndNames: スペース内で指定されたトピック名の一覧に一致するトピックを取得する（Wikiリンク解析時のトピック一括検索用）
  - 想定ファイル数: 実装 4 ファイル（+ sqlc生成 1 ファイル）, テスト 2 ファイル
  - 想定行数: 実装 ~150 行, テスト ~120 行
  - 依存: 1-1

- [x] **1-4**: [Go] TopicMember モデル・クエリ・リポジトリの追加
  - `internal/model/topic_member.go` を追加（ID, SpaceID, TopicID, SpaceMemberID, Role, JoinedAt, LastPageModifiedAt）
  - `internal/query/queries/topic_members.sql` を追加（FindBySpaceMemberAndTopic, UpdateLastPageModifiedAt）
  - `internal/repository/topic_member.go` を追加（FindBySpaceMemberAndTopic, UpdateLastPageModifiedAt, WithTx）
  - `internal/testutil/topic_member_builder.go` を追加
  - UpdateLastPageModifiedAt: トピックIDとスペースメンバーIDで特定されるTopicMemberの`last_page_modified_at`を指定日時で更新する（ページ公開時に使用）
  - 想定ファイル数: 実装 4 ファイル（+ sqlc生成 1 ファイル）, テスト 2 ファイル
  - 想定行数: 実装 ~140 行, テスト ~120 行
  - 依存: 1-2, 1-3

### フェーズ 2: ページモデルとリポジトリ

ページ関連のモデル・クエリ・リポジトリを追加する。

- [x] **2-1**: [Go] Page モデル・クエリ・リポジトリの追加
  - `internal/model/page.go` を追加（設計セクションのデータ構造に準拠）
  - `internal/query/queries/pages.sql` を追加（FindBySpaceAndNumber, FindByIDs, FindBacklinkedByPageID, Update, FindByTopicAndTitle, CreateLinkedPage）
  - `internal/repository/page.go` を追加（FindBySpaceAndNumber, FindByIDs, FindBacklinkedByPageID, Update, FindByTopicAndTitle, CreateLinkedPage, WithTx）
  - `internal/testutil/page_builder.go` を追加
  - FindByIDs: `linkedPageIds`に含まれるIDのページを取得（リンク一覧表示用。同スペース・公開済み・未廃棄のページのみ）
  - FindBacklinkedByPageID: `linked_page_ids`カラムに指定ページIDが含まれるページを取得（バックリンク一覧表示用。同スペース・公開済み・未廃棄のページのみ）
  - FindByTopicAndTitle: 指定トピック内で指定タイトルのページを取得する（Wikiリンクのページ存在確認とHTML変換用）
  - CreateLinkedPage: Wikiリンクから参照されるページをfirst_or_create方式で作成する（空のbody/bodyHtml、空のlinkedPageIds、modified_atを現在日時で設定。PageEditorレコードも作成）
  - 想定ファイル数: 実装 4 ファイル（+ sqlc生成 1 ファイル）, テスト 2 ファイル
  - 想定行数: 実装 ~250 行, テスト ~200 行
  - 依存: 1-1, 1-3

- [x] **2-2**: [Go] DraftPage モデル・クエリ・リポジトリの追加
  - `internal/model/draft_page.go` を追加（設計セクションのデータ構造に準拠）
  - `internal/query/queries/draft_pages.sql` を追加（FindByPageAndMember, Create, Update, Delete）
  - `internal/repository/draft_page.go` を追加（FindByPageAndMember, Create, Update, Delete, WithTx）
  - `internal/testutil/draft_page_builder.go` を追加
  - 想定ファイル数: 実装 4 ファイル（+ sqlc生成 1 ファイル）, テスト 2 ファイル
  - 想定行数: 実装 ~160 行, テスト ~120 行
  - 依存: 1-2, 2-1

- [x] **2-2.5**: [Go] TopicPolicy の追加
  - `internal/policy/topic.go` を追加（TopicPolicyインターフェース、NewTopicPolicyファクトリ関数）
  - `internal/policy/topic_owner.go` を追加（topicOwnerPolicy構造体、スペースオーナー用）
  - `internal/policy/topic_admin.go` を追加（topicAdminPolicy構造体、トピックAdmin用）
  - `internal/policy/topic_member.go` を追加（topicMemberPolicy構造体、トピックMember用）
  - `internal/policy/topic_guest.go` を追加（topicGuestPolicy構造体、非メンバー用）
  - ファクトリ関数は`active bool`パラメータを受け取り、各ポリシーがactiveチェックを行う
  - スペースオーナーは同じスペース内の全トピックのページを編集可能
  - トピックAdmin/Memberは所属トピックのページを編集可能
  - 非アクティブなスペースメンバーは全操作が拒否される
  - 非トピックメンバーは編集不可
  - 想定ファイル数: 実装 5 ファイル, テスト 1 ファイル
  - 想定行数: 実装 ~80 行, テスト ~180 行
  - 依存: 1-2, 1-4, 2-1, 2-2

- [x] **2-3**: [Go] PageRevision モデル・クエリ・リポジトリの追加
  - `internal/model/page_revision.go` を追加
  - `internal/query/queries/page_revisions.sql` を追加（Create）
  - `internal/repository/page_revision.go` を追加（Create, WithTx）
  - 想定ファイル数: 実装 3 ファイル（+ sqlc生成 1 ファイル）, テスト 1 ファイル
  - 想定行数: 実装 ~100 行, テスト ~80 行
  - 依存: 1-2, 2-1

- [x] **2-4**: [Go] PageEditor モデル・クエリ・リポジトリの追加
  - `internal/model/page_editor.go` を追加
  - `internal/query/queries/page_editors.sql` を追加（FindOrCreate）
  - `internal/repository/page_editor.go` を追加（FindOrCreate, WithTx）
  - 想定ファイル数: 実装 3 ファイル（+ sqlc生成 1 ファイル）, テスト 1 ファイル
  - 想定行数: 実装 ~100 行, テスト ~70 行
  - 依存: 1-2, 2-1

- [x] **2-5**: [Go] PageAttachmentReference モデル・クエリ・リポジトリの追加
  - `internal/model/page_attachment_reference.go` を追加（ID, AttachmentID, PageID）
  - `internal/query/queries/page_attachment_references.sql` を追加（ListByPage, CreateBatch, DeleteByPageAndAttachmentIDs）
  - `internal/repository/page_attachment_reference.go` を追加（ListByPage, CreateBatch, DeleteByPageAndAttachmentIDs, WithTx）
  - 想定ファイル数: 実装 3 ファイル（+ sqlc生成 1 ファイル）, テスト 1 ファイル
  - 想定行数: 実装 ~120 行, テスト ~100 行
  - 依存: 2-1

- [x] **2-6**: [Go] Attachment モデル・クエリ・リポジトリの追加
  - `internal/model/attachment.go` を追加（ID, SpaceID, Filename — 添付ファイル存在確認・ファイル種別判定用のモデル）
  - `internal/query/queries/attachments.sql` を追加（ExistsByIDAndSpace, FindByIDAndSpace）
  - `internal/repository/attachment.go` を追加（ExistsByIDAndSpace, FindByIDAndSpace）
  - FindByIDAndSpace: AttachmentFilter用。IDとスペースIDで添付ファイルを取得し、Filenameからファイル種別を判定する
  - 想定ファイル数: 実装 3 ファイル（+ sqlc生成 1 ファイル）, テスト 1 ファイル
  - 想定行数: 実装 ~60 行, テスト ~50 行
  - 依存: 1-1

### フェーズ 2b: スペースIDによるクエリスコープの強化

スペース内のリソースに対するUPDATE/DELETEクエリにspace_idを条件として追加し、スペースをまたいだ操作を防止する（防御的プログラミング）。

- [x] **2b-1**: [Go] スペースIDによるクエリスコープの強化
  - `db/queries/pages.sql` の `UpdatePage` にspace_idをWHERE条件に追加
  - `db/queries/draft_pages.sql` の `UpdateDraftPage` にspace_idをWHERE条件に追加
  - `db/queries/draft_pages.sql` の `DeleteDraftPage` にspace_idをWHERE条件に追加
  - `db/queries/page_editors.sql` の `UpdatePageEditorLastPageModifiedAt` にspace_idをWHERE条件に追加
  - sqlc再生成後、対応するリポジトリのメソッドシグネチャにspaceIDを追加
  - 呼び出し元（handler, usecase）の修正
  - 想定ファイル数: 実装 ~8 ファイル（SQL 3 + リポジトリ 3 + 呼び出し元 2）, テスト ~3 ファイル
  - 想定行数: 実装 ~50 行（変更）, テスト ~30 行（変更）
  - 依存: なし（独立して実施可能）

### フェーズ 2a: ドメインID型の導入

モデル層のIDフィールドを`string`から専用型（`type SpaceID string`等）に変更し、IDの取り違えをコンパイル時に検出できるようにする。

- [x] **2a-1**: [Go] ドメインID型の定義とモデル・リポジトリ・ポリシーへの適用
  - `internal/model/id.go` を追加（SpaceID, TopicID, PageID, SpaceMemberID, TopicMemberID, DraftPageID）
  - `internal/model/` 配下の各モデルのIDフィールドの型を `string` から専用型に変更
  - `internal/repository/` 配下の各リポジトリで、sqlcの`string`から専用型への変換を追加
  - `internal/policy/` 配下のポリシーで`string`を専用型に変更
  - テストヘルパー（`internal/testutil/`）の対応する型も更新
  - 想定ファイル数: 実装 ~15 ファイル（id.go新規 + 既存ファイル更新）, テスト ~8 ファイル（既存テストの修正）
  - 想定行数: 実装 ~150 行（変更）, テスト ~80 行（変更）
  - 依存: 1-1〜1-4, 2-1〜2-6

### フェーズ 3: Markdownレンダリングとマークアップ処理

- [x] **3-1**: [Go] Markdownレンダリングパッケージの追加
  - `internal/markup/markup.go` を追加（goldmark + bluemonday）
  - goldmark: GitHub Flavored Markdown対応でMarkdown→HTML変換。`html.WithUnsafe()`オプションでHTMLタグの記述を許可（Rails版の`html: true`, `unsafe: true`に相当）
  - bluemonday: HTMLサニタイズ（XSS対策）。`input`要素の許可（タスクリスト用）、`img`要素の`width`/`height`属性の許可（Rails版の`sanitization_config`に相当）
  - 単独`<img>`タグのゼロ幅非接合子ワークアラウンド（Rails版と同様の問題が発生する場合）
  - `go.mod` に `github.com/yuin/goldmark`, `github.com/microcosm-cc/bluemonday` を追加
  - 想定ファイル数: 実装 1 ファイル, テスト 1 ファイル
  - 想定行数: 実装 ~100 行, テスト ~150 行
  - 依存: なし

- [x] **3-2**: [Go] 添付ファイルフィルター（AttachmentFilter）の追加
  - `internal/markup/attachment_filter.go` を追加
  - `FilterAttachments(bodyHTML string, spaceID string, attachmentRepo) string` — bodyHTML内の`/attachments/{id}`パターンのURLを持つ`<img>`・`<a>`タグを変換する
  - インライン画像（jpg, jpeg, png, gif, svg, webp）: `<img>`を`<a class="wikino-attachment-image-link"><img data-attachment-id="{id}" /></a>`に変換
  - インライン動画（mp4, webm, ogg, mov）: `<a>`を`<video data-attachment-id="{id}" controls>`に変換
  - その他のファイル: ダウンロードリンク（SVGアイコン付き`<a>`タグ）に変換
  - 全てのURLはプレースホルダー（`src=""`、`href="#"`）で出力し、`data-attachment-id`・`data-attachment-link`属性を付与
  - 同じスペースの添付ファイルが存在しない場合は変換をスキップ
  - 単独画像リンクの`<p>`要素ラッピング（後処理）
  - 想定ファイル数: 実装 1 ファイル, テスト 1 ファイル
  - 想定行数: 実装 ~180 行, テスト ~200 行
  - 依存: 2-6

- [x] **3-3**: [Go] 添付ファイルID抽出・アイキャッチ画像ID抽出ロジックの追加
  - `internal/markup/attachment_extract.go` を追加
  - `ExtractAttachmentIDs(bodyHTML string) []string` — bodyHTMLから添付ファイルIDを抽出（HTML img/aタグ、Markdown画像/リンクの4パターン）
  - `ExtractFeaturedImageID(body string) *string` — Markdown本文の1行目から画像IDを抽出
  - 想定ファイル数: 実装 1 ファイル, テスト 1 ファイル
  - 想定行数: 実装 ~80 行, テスト ~150 行
  - 依存: なし

- [x] **3-4**: [Go] Wikiリンクパース・HTML変換ロジックの追加
  - `internal/markup/wikilink.go` を追加
  - `ScanWikilinks(body string, currentTopicName string) []WikilinkKey` — Markdown本文からWikiリンク（`[[ページ名]]`、`[[トピック名/ページ名]]`）をパースし、トピック名とページタイトルのペアのリストを返す
  - `ReplaceWikilinks(bodyHTML string, currentTopicName string, spaceIdentifier string, pageLocations []PageLocation) string` — bodyHTML内のWikiリンクをHTML `<a>`タグに変換する。`<a>`, `<code>`, `<pre>`, `<script>`, `<style>`タグ内のWikiリンクは変換しない
  - `WikilinkKey` — トピック名とページタイトルのペア（Raw, TopicName, PageTitle）
  - `PageLocation` — WikilinkKeyに対応するページ情報（Key, TopicName, PageID, PageNumber, PageTitle）
  - パースロジック: 正規表現`\[\[(.*?)\]\]`で抽出し、`/`で最大2分割。`トピック名/ページ名`形式の場合はそのまま、`ページ名`のみの場合は現在のトピック名を使用
  - HTML変換ロジック: ページが存在する場合は`<a href="/s/{space_identifier}/pages/{page_number}">{ページタイトル}</a>`に変換、存在しない場合はプレーンテキストのまま残す
  - 想定ファイル数: 実装 1 ファイル, テスト 1 ファイル
  - 想定行数: 実装 ~120 行, テスト ~200 行
  - 依存: なし

- [x] **3-5**: [Go] HTML処理を正規表現からDOMパーサーに移行
  - `attachment_filter.go` と `wikilink.go` の HTML処理を `golang.org/x/net/html` ベースに置き換える
  - 現状の正規表現ベースの処理は、ネストしたタグ・不正なHTML・シングルクォート属性値など、エッジケースでの誤動作リスクがある
  - Rails版はSelma（DOMパーサー）を使用しており、構造的に堅牢
  - `wikilink.go`: スキップ範囲検出（`findSkipRanges`）をDOMツリー走査に置き換え、テキストノード単位でWikiリンクを変換する
  - `attachment_filter.go`: img/a要素の属性操作をDOMノード操作に置き換える
  - `go.mod` に `golang.org/x/net` を追加
  - 想定ファイル数: 実装 2 ファイル（既存ファイルの修正）, テスト 2 ファイル（既存テストの修正）
  - 想定行数: 実装 ~300 行（差分）, テスト ~50 行（差分）
  - 依存: 3-2, 3-4

- [x] **3-6**: [Go] スタンドアロン画像の`<p>`要素ラッピング後処理の追加
  - `internal/markup/markup.go` または `internal/markup/attachment_filter.go` に後処理を追加
  - Rails版 `markup.rb:98-113` に相当する処理: `<p>`要素で囲まれていない独立した `a.wikino-attachment-image-link` を `<p>` 要素で囲む
  - 画像リンクの直後にインライン要素（`<br>`, `<em>` 等）が続く場合はラッピングしない
  - 想定ファイル数: 実装 1 ファイル（既存ファイルの修正）, テスト 1 ファイル（既存テストの修正）
  - 想定行数: 実装 ~30 行, テスト ~50 行
  - 依存: 3-2

- [x] **3-7**: [Go] AttachmentFilterのテスト拡充
  - `internal/markup/attachment_filter_test.go` にテストケースを追加
  - Rails版 `attachment_filter_spec.rb`（18ケース）との差分を埋める
  - 追加すべきテストケース:
    - クロススペースの添付ファイルアクセスがブロックされること
    - 特殊文字を含むファイル名のXSS防止（`<script>` を含むファイル名等）
    - 複数HTML imgタグの一括変換
    - 画像の後にemphasis記法（`*caption*`）が続くケース
    - スタンドアロン画像の`<p>`ラッピングの検証（3-6の実装後）
  - 想定ファイル数: テスト 1 ファイル（既存テストの修正）
  - 想定行数: テスト ~100 行
  - 依存: 3-2, 3-6

- [x] **3-8**: [Go] Wikilinkの三重括弧エッジケーステストの追加
  - `internal/markup/wikilink_test.go` にテストケースを追加
  - `[[[a]]]`（三重括弧）の動作を検証（Rails版はObsidian互換で `[a` として処理）
  - 想定ファイル数: テスト 1 ファイル（既存テストの修正）
  - 想定行数: テスト ~10 行
  - 依存: 3-4

- [x] **3-9**: [Go] バッチレンダリングの実装
  - Rails版の`Markup.render_html_batch`に相当する機能をGo版に追加
  - 複数ページのMarkdownを一括レンダリングする際、Wikiリンクのページ検索や添付ファイル検索をバッチ化してN+1クエリを防止する
  - 全ページのWikiリンクキーを事前に収集し、1回のDBクエリでPageLocationを取得する
  - 全ページの添付ファイルIDを事前に収集し、1回のDBクエリで存在確認を行う
  - 想定ファイル数: 実装 1〜2 ファイル, テスト 1 ファイル
  - 想定行数: 実装 ~150 行, テスト ~100 行
  - 依存: 3-1, 3-2, 3-4

- [x] **3-10**: [Go] マークアップパイプライン統合関数の追加
  - Rails版の`Markup.render_html`のように、Markdownレンダリング → HTMLサニタイズ → 添付ファイルフィルター → Wikiリンク変換 → スタンドアロン画像ラッピングの一連の処理を統合する関数を追加
  - 現状は各処理が独立した関数として存在し、呼び出し側でパイプラインを組み立てる必要がある
  - 統合関数により処理順序の一貫性を保証し、呼び出し側のコードを簡潔にする
  - 想定ファイル数: 実装 1 ファイル（既存ファイルの修正）, テスト 1 ファイル
  - 想定行数: 実装 ~50 行, テスト ~80 行
  - 依存: 3-1, 3-2, 3-4, 3-6

- [x] **3-11**: [Go] Wikiリンク置換時のHTMLエンティティ処理の追加
  - Rails版のPageLinkFilterでは`CGI.unescapeHTML`でHTMLエンティティをデコードしてからWikiリンクのマッチングを行っている
  - Go版ではHTMLパーサー（`golang.org/x/net/html`）がテキストノードを自動デコードするため基本的に問題ないが、`&amp;`や`&#91;`等のエンティティを含むWikiリンクのエッジケースをテストで検証する
  - 問題が見つかった場合はデコード処理を追加する
  - 想定ファイル数: テスト 1 ファイル（既存テストの修正）, 必要に応じて実装 1 ファイル
  - 想定行数: テスト ~30 行, 実装 ~20 行（必要な場合のみ）
  - 依存: 3-4, 3-5

- [x] **3-12**: [Go] フルパイプライン統合テストの追加
  - Rails版の`markup_spec.rb`・`page_link_filter_spec.rb`・`attachment_filter_spec.rb`のように、Markdownテキストを入力として最終的なHTML出力を検証する統合テストを追加
  - 個別の関数テスト（単体テスト）は充実しているが、パイプライン全体を通した統合テストが不足している
  - Wikiリンクと添付ファイルが混在するMarkdown、タスクリストとWikiリンクの組み合わせなど、複合的なケースを検証する
  - 想定ファイル数: テスト 1 ファイル
  - 想定行数: テスト ~150 行
  - 依存: 3-10

### フェーズ 4: スペースページ用レイアウト

- [x] **4-1**: [Go] スペースページ用レイアウトの追加
  - `internal/templates/layouts/space.templ` を追加（スペース名ヘッダー、メインコンテンツエリア、フッター）
  - `internal/viewmodel/space.go` を追加（レイアウトに渡すスペース情報の構造体）
  - 翻訳ファイル（`ja.toml`, `en.toml`）にスペースレイアウト関連のメッセージを追加
  - フルナビゲーション（サイドバー、検索バー等）は含めない
  - 想定ファイル数: 実装 4 ファイル, テスト 1 ファイル
  - 想定行数: 実装 ~150 行, テスト ~60 行
  - 依存: 1-1

### フェーズ 5: ページ編集画面

- [x] **5-1**: [Go] ページ編集ハンドラーとテンプレートの追加
  - `internal/handler/page/edit.go` を追加（DraftPage存在確認、フォーム表示）
  - `internal/templates/pages/page/edit.templ` を追加（タイトル入力、本文テキストエリア、CSRFトークン、公開ボタン）
  - 翻訳ファイルにページ編集関連のメッセージを追加
  - `cmd/server/main.go` にルーティングを登録（`/go` プレフィックス付き: `GET /go/s/:space_identifier/pages/:page_number/edit`）
  - `RequireAuth`ミドルウェアを使用、ハンドラー内でスペースメンバーを確認
  - `TopicPolicy.CanUpdatePage()`によるトピックアクセス制御（権限がない場合は404）
  - DraftPageが存在すればその内容を、なければPageの内容をフォームに表示
  - **オートフォーカスロジック**: タイトルが空の場合はタイトル入力にオートフォーカス、タイトルがある場合は本文エディタにオートフォーカス（Rails版の`autofocus_title?`/`autofocus_body?`に相当）
  - 想定ファイル数: 実装 4 ファイル, テスト 2 ファイル
  - 想定行数: 実装 ~270 行, テスト ~180 行
  - 依存: 1-5, 2-2, 4-1

- [x] **5-4**: [Go] ページロケーション検索APIの追加（Wikiリンク補完用）
  - `internal/handler/page_location/handler.go` を追加（Handler構造体と依存性）
  - `internal/handler/page_location/index.go` を追加（GET /go/s/:space_identifier/page_locations?q=:keyword）
  - `internal/query/queries/pages.sql` に`SearchPageLocations`クエリを追加（pagesテーブルとtopicsテーブルをJOINし、タイトルのILIKE検索、modified_at降順、LIMIT 10）
  - `internal/repository/page.go` に`SearchPageLocations`メソッドを追加（クエリパラメータ`q`をスペース区切りで分割し、各ワードに対してILIKEのAND条件で検索）
  - `cmd/server/main.go` にルーティングを登録（`/go` プレフィックス付き: `GET /go/s/:space_identifier/page_locations?q=:keyword`）
  - `RequireAuth`ミドルウェアを使用、ハンドラー内でアクティブなスペースメンバーであることを確認（スペースメンバーでない場合は404）
  - JSONレスポンス: `{"page_locations": [{"key": "トピック名/ページタイトル"}]}`
  - 想定ファイル数: 実装 3 ファイル（+ sqlc生成 1 ファイル）, テスト 2 ファイル
  - 想定行数: 実装 ~150 行, テスト ~150 行
  - 依存: 1-1, 1-2, 2-1

### フェーズ 6: 下書き自動保存

- [x] **6-1**: [Go] 下書き自動保存ユースケースの追加
  - `internal/usecase/auto_save_draft_page.go` を追加（find_or_create方式でDraftPageを取得・作成し、Markdownレンダリング + 添付ファイルフィルター + Wikiリンク解析・自動ページ作成・HTML変換後に更新）
  - **find_or_createの競合対策**: DraftPageのユニーク制約（`space_member_id + page_id`）違反時はリトライする（Rails版の`rescue RecordNotUnique; retry`に相当）
  - Wikiリンク処理: bodyからWikiリンクをパース → トピック検索 → リンク先ページの自動作成 → DraftPageのlinkedPageIdsを更新 → bodyHTML内のWikiリンクを`<a>`タグに変換
  - 想定ファイル数: 実装 1 ファイル, テスト 1 ファイル
  - 想定行数: 実装 ~180 行, テスト ~200 行
  - 依存: 2-2, 3-1, 3-2, 3-4

- [x] **6-2**: [Go] 下書き自動保存ハンドラーの追加
  - `internal/handler/draft_page/handler.go`, `internal/handler/draft_page/update.go` を追加
  - JSONレスポンス（`{"modified_at": "2024-01-01T12:34:56Z"}`）
  - `cmd/server/main.go` にルーティングを登録（`/go` プレフィックス付き: `PATCH /go/s/:space_identifier/pages/:page_number/draft_page`）
  - `RequireAuth`ミドルウェアを使用、ハンドラー内でスペースメンバーを確認
  - `TopicPolicy.CanUpdateDraftPage()`によるトピックアクセス制御（権限がない場合は404）
  - 想定ファイル数: 実装 3 ファイル, テスト 2 ファイル
  - 想定行数: 実装 ~120 行, テスト ~150 行
  - 依存: 1-5, 6-1

### フェーズ 7: ページ公開

- [x] **7-1**: [Go] ページ公開ユースケースの追加
  - `internal/usecase/publish_page.go` を追加（トランザクション内でPage更新、PageRevision作成、PageEditor追加、DraftPage削除、TopicMemberのlast_page_modified_at更新、Wikiリンク解析・自動ページ作成）
  - TopicMemberのlast_page_modified_at更新: トランザクション内でTopicMemberRepository.UpdateLastPageModifiedAtを呼び出し、公開日時で更新する
  - Wikiリンク処理: bodyからWikiリンクをパース → トピック検索 → リンク先ページの自動作成 → PageのlinkedPageIdsを更新 → bodyHTML内のWikiリンクを`<a>`タグに変換
  - 想定ファイル数: 実装 1 ファイル, テスト 2 ファイル
  - 想定行数: 実装 ~200 行, テスト ~220 行
  - 依存: 1-4, 2-3, 2-4, 3-2, 3-4, 6-1

- [x] **7-2**: [Go] ページ公開ハンドラー・バリデーターの追加
  - `internal/handler/page/update.go` を追加
  - `internal/handler/page/validator.go` を追加（タイトルの必須チェック、トピック存在チェック、トピック内のタイトル重複チェック）
  - タイトル重複チェック: 同じトピック内で同一タイトルのページが存在する場合、そのページの編集リンク付きのエラーメッセージを返す（Rails版の`title_uniqueness`バリデーションに相当）
  - bodyがnilの場合は空文字に変換する（Rails版の`convert_nil_to_empty_string`に相当）
  - 翻訳ファイルにバリデーションエラーメッセージを追加
  - `cmd/server/main.go` にルーティングを登録（`/go` プレフィックス付き: `PATCH /go/s/:space_identifier/pages/:page_number`）
  - `TopicPolicy.CanUpdatePage()`によるトピックアクセス制御（権限がない場合は404）
  - **バリデーション失敗時**: 編集画面を再表示する（HTTPステータス422）。リンク一覧・バックリンク一覧はフェーズ 8b でDatastarを使用して動的表示するため、初期実装ではフッターなしで再表示する
  - **公開成功時**: フラッシュメッセージ（`messages.pages.saved`相当）を設定し、ページ表示画面（`/s/{space_identifier}/pages/{page_number}`）にリダイレクト（ページ表示はRails版のため`/go`プレフィックスなし）
  - 想定ファイル数: 実装 4 ファイル, テスト 2 ファイル
  - 想定行数: 実装 ~150 行, テスト ~180 行
  - 依存: 1-5, 5-1, 7-1

- [x] **7-3**: [Go] ページ公開時の添付ファイル参照同期・アイキャッチ画像設定の追加
  - `internal/usecase/publish_page.go` を更新（添付ファイル参照同期、アイキャッチ画像抽出をトランザクション内に追加）
  - 公開時のフロー: bodyHTMLから添付ファイルIDを抽出 → 既存参照との差分を計算 → 追加分は添付ファイル存在確認後にPageAttachmentReference作成 → 削除分はPageAttachmentReference削除
  - 公開時のフロー: bodyの1行目から画像IDを抽出 → 添付ファイル存在確認 → Pageのfeatured_image_attachment_idを更新
  - 想定ファイル数: 実装 1 ファイル（更新）, テスト 2 ファイル
  - 想定行数: 実装 ~100 行, テスト ~200 行
  - 依存: 2-5, 2-6, 3-3, 7-1

### フェーズ 8: フロントエンド

- [x] **8-1**: [Go] Playwright E2Eテスト基盤のセットアップ
  - `go/e2e/` ディレクトリにPlaywrightプロジェクトを作成
  - `go/e2e/package.json` にPlaywright関連パッケージを追加（`@playwright/test`）
  - `go/e2e/playwright.config.ts` を作成（ベースURL、ブラウザ設定、タイムアウト設定）
  - `go/e2e/helpers/auth.ts` を作成（テスト用ユーザーのサインインヘルパー）
  - `go/e2e/helpers/editor.ts` を作成（CodeMirrorエディタ操作のヘルパー: テキスト入力、カーソル制御、コンテンツ取得、選択操作）
  - `go/e2e/helpers/database.ts` を作成（テストデータのセットアップ・クリーンアップ）
  - `go/Makefile` にE2Eテスト実行タスクを追加（`make e2e`, `make e2e-file`）
  - Rails版の `spec/system/components/markdown_editor_component/shared_helpers.rb` と同等のヘルパーを移植
  - 想定ファイル数: 5 ファイル
  - 想定行数: ~300 行

- [x] **8-2**: [Go] CodeMirrorエディタと自動保存JSの追加 + E2Eテスト（タブインデント・リスト続行）
  - CodeMirror 6の初期化（Markdown構文ハイライト、履歴、括弧マッチング）
  - 自動保存コントローラー（500msデバウンス、fetch APIでPATCHリクエスト送信、保存時刻の表示更新）
  - キーバインド（Enter: リスト続行、Tab/Shift-Tab: インデント、Cmd/Ctrl+Enter: フォーム送信）
  - エディタ内容とhidden textareaの同期
  - `package.json` にCodeMirror関連パッケージを追加
  - esbuildの設定を更新
  - `go/e2e/tests/tab-indent.spec.ts` を作成（Tab/Shift-Tabによるインデント・アンインデント、選択範囲のインデント）
  - `go/e2e/tests/list-continuation.spec.ts` を作成（Enter押下時のリストマーカー続行、空リスト項目でのマーカー削除、タスクリスト対応）
  - Rails版の `tab_indent_spec.rb`, `list_continuation_spec.rb` と同等のテストケースをカバー
  - 想定ファイル数: 実装 4 ファイル, テスト 2 ファイル
  - 想定行数: 実装 ~300 行, テスト ~300 行
  - 依存: 5-1, 6-2, 8-1

- [x] **8-2b**: `make e2e` だけでE2Eテストを実行できるようにする
  - **課題1: E2Eテスト用サーバーの自動起動・停止**
    - `make e2e` 実行時にE2E用の設定でGoサーバーを自動起動し、テスト完了後に停止する
    - E2E用設定: `WIKINO_PORT=4201`, `WIKINO_COOKIE_DOMAIN=localhost`, `WIKINO_TURNSTILE_SECRET_KEY=""`, `WIKINO_TURNSTILE_SITE_KEY=""`
    - Makefile内でバックグラウンド起動 → ヘルスチェック待機 → テスト実行 → プロセス停止の一連の流れを実装
  - **課題2: `APP_ENV=test_e2e` でopコマンド経由の環境変数取得**
    - `APP_ENV=test_e2e` を使用して1Password経由でE2Eテスト用の環境変数（`DATABASE_URL` 等）を取得する
    - Makefile の `e2e` / `e2e-file` ターゲットで `APP_ENV=test_e2e` を指定する
  - **課題3: テスト用DBのセットアップ**
    - E2Eテスト実行前に `make db-setup-test` を自動実行してスキーマを最新に保つ
  - **課題4: JSバンドルの自動ビルド**
    - E2Eテスト実行前に `pnpm build:js` を自動実行して最新のフロントエンドコードを使用する
  - **課題5: Playwrightブラウザの自動インストール**
    - 初回実行時に `pnpm exec playwright install chromium` を自動実行する（または手順をドキュメント化）
  - **課題6: `.env.example` の更新**
    - `WIKINO_DATABASE_E2E_URL` を `.env.example` に追加
  - 依存: 8-2

- [x] **8-3**: [Go] ファイルアップロードのコア機能の追加
  - `file-upload-handler.ts` を追加（バリデーション、MD5チェックサム計算、Presignリクエスト、アップロード実行、プレースホルダー置換）
  - `direct-upload.ts` を追加（XMLHttpRequestラッパー、S3直接アップロード、プログレス追跡）
  - `upload-placeholder.ts` を追加（プレースホルダーの挿入・追跡・置換・削除）
  - 添付ファイルのPresignエンドポイント・表示エンドポイントは引き続きRails版にプロキシ
  - 想定ファイル数: 実装 3 ファイル
  - 想定行数: 実装 ~250 行
  - 依存: 8-2

- [x] **8-4**: [Go] エディタのドラッグ&ドロップ・ペーストアップロードの追加 + E2Eテスト（ファイルアップロード・ペースト）
  - `file-drop-handler.ts` を追加（CodeMirror ViewPlugin、ドラッグ&ドロップイベント検出、ドロップゾーン表示）
  - `paste-handler.ts` を追加（クリップボードペースト検知、MIMEタイプ判定、カスタムイベントディスパッチ）
  - 8-2のエディタ初期化にファイルアップロードハンドラーを統合（イベントリスナー登録）
  - `go/e2e/tests/file-upload.spec.ts` を作成（ドラッグ&ドロップ、ペーストによるファイルアップロード、ドロップゾーン表示、プレースホルダー挿入・置換）
  - `go/e2e/tests/paste.spec.ts` を作成（テキストペースト、画像ペースト）
  - Rails版の `file_upload_spec.rb`, `paste_spec.rb` と同等のテストケースをカバー
  - 想定ファイル数: 実装 2 ファイル（+ 1 ファイル更新）, テスト 2 ファイル
  - 想定行数: 実装 ~150 行, テスト ~250 行
  - 依存: 8-1, 8-3

- [x] **8-5**: [Go] Wikiリンク補完フロントエンドの追加 + E2Eテスト
  - `wikilink-completions.ts` を追加（CodeMirror autocompletion override、`[[`入力検出、ページロケーション検索API呼び出し、補完候補の構築）
  - 8-2のCodeMirror初期化に`autocompletion({ override: [wikilinkCompletions(spaceIdentifier)] })`を統合
  - テンプレートの`data-space-identifier`属性からスペース識別子を取得
  - 補完トリガー: 正規表現`/\[\[.*/`で`[[`以降のテキストを検出
  - API呼び出し: `fetch(/go/s/${spaceIdentifier}/page_locations?q=${keyword})`でページロケーションを取得
  - 補完候補: label=`[[トピック名/ページタイトル`, displayLabel=`トピック名/ページタイトル`, filter=false
  - `go/e2e/tests/wiki-link-autocomplete.spec.ts` を作成（`[[`入力でのオートコンプリート表示、候補選択、補完結果の挿入）
  - Rails版の `wiki_link_autocomplete_spec.rb` と同等のテストケースをカバー
  - 想定ファイル数: 実装 1 ファイル（+ 1 ファイル更新）, テスト 1 ファイル
  - 想定行数: 実装 ~80 行, テスト ~150 行
  - 依存: 5-4, 8-1, 8-2

### フェーズ 8b: Datastarによるリンク一覧・バックリンク一覧の動的表示

CodeMirrorエディタと下書き自動保存の動作確認ができた後、Datastarを使用してリンク一覧・バックリンク一覧をリアルタイムに更新する機能を実装する。Rails版ではTurbo Streamで実現していた部分を、Go版ではDatastarで再実装する。

- [x] **8b-1**: [Go] ページ編集画面のリンク一覧表示の追加
  - Datastarはvendor JSとして導入済みのため、`package.json`やesbuild設定の変更は不要
  - リンク一覧: DraftPage存在時はDraftPageの`linkedPageIds`、なければPageの`linkedPageIds`を基に表示
  - 初期表示時: ハンドラーでリンク先ページを取得し、templコンポーネントでサーバーサイドレンダリング
  - 自動保存時: auto-save APIのJSONレスポンスに`link_list_html`フィールドを追加し、JSがDOMを更新
  - ページネーションは含めない
  - 依存: 2-1, 5-1, 6-2, 8-2
  - **サブタスク**:
    - [x] **8b-1a**: リンク一覧のビューモデルとtemplコンポーネントの追加
      - `internal/viewmodel/link_list.go` を追加（`LinkListItem`, `LinkList`構造体、`NewLinkList`コンストラクタ）
      - `internal/templates/components/link_list.templ` を追加（リンク一覧コンポーネント。リンクが空の場合は非表示）
      - 翻訳ファイルにリンク一覧関連のメッセージを追加（`page_edit_links_heading`）
      - 想定ファイル数: 3 ファイル（実装2 + 翻訳2）
    - [x] **8b-1b**: ページ編集ハンドラーとテンプレートの更新
      - `internal/handler/page/edit.go` を更新（`pageRepo.FindByIDs()`でリンク先ページ取得、`EditPageData`に`LinkList`追加）
      - `internal/templates/pages/page/edit.templ` を更新（フォーム下に`<div id="page-link-list">`でリンク一覧セクション追加）
      - 想定ファイル数: 2 ファイル
    - [x] **8b-1c**: 下書き自動保存レスポンスとJSの更新
      - `internal/handler/draft_page/update.go` を更新（auto-save後にリンク先ページを取得し、templコンポーネントをバッファにレンダリングしてHTMLフラグメントをJSONレスポンスに含める）
      - `web/markdown-editor/markdown-editor.ts` を更新（`link_list_html`をDOM更新）
      - 想定ファイル数: 2 ファイル

- [ ] **8b-2**: [Go] ページ編集画面のバックリンク一覧表示の追加
  - `internal/handler/page/edit.go` を更新（バックリンク一覧のデータ取得を追加）
  - `internal/templates/pages/page/edit.templ` を更新（フッターにバックリンク一覧セクションを追加）
  - `internal/templates/components/backlink_list.templ` を追加（バックリンク一覧コンポーネント。ページタイトルとリンクを表示）
  - `internal/viewmodel/backlink_list.go` を追加（バックリンク一覧のビューモデル）
  - 翻訳ファイルにバックリンク関連のメッセージを追加
  - バックリンク一覧: 常にPageの公開済みデータを基に表示
  - 下書き自動保存のレスポンスにバックリンク一覧のHTMLフラグメントも含める
  - ページネーションは含めない
  - 想定ファイル数: 実装 4 ファイル, テスト 1 ファイル
  - 想定行数: 実装 ~120 行, テスト ~80 行
  - 依存: 2-1, 5-1, 8b-1

### フェーズ 9: `/go` プレフィックスの除去とリバースプロキシの更新

- [ ] **9-1**: [Go] `/go` プレフィックスの除去とリバースプロキシ設定の追加
  - `cmd/server/main.go` のルーティングから `/go` プレフィックスを除去し、本番用のパスに変更:
    - `GET /go/s/:space_identifier/pages/:page_number/edit` → `GET /s/:space_identifier/pages/:page_number/edit`
    - `PATCH /go/s/:space_identifier/pages/:page_number/draft_page` → `PATCH /s/:space_identifier/pages/:page_number/draft_page`
    - `PATCH /go/s/:space_identifier/pages/:page_number` → `PATCH /s/:space_identifier/pages/:page_number`
    - `GET /go/s/:space_identifier/page_locations?q=:keyword` → `GET /s/:space_identifier/page_locations?q=:keyword`
  - フロントエンドのAPI呼び出しURL（自動保存、Wikiリンク補完）から `/go` プレフィックスを除去
  - `internal/middleware/reverse_proxy.go` のホワイトリストにページ編集・自動保存・公開・ページロケーション検索のパスパターンを追加
  - ページ表示（`GET /s/:space_identifier/pages/:page_number`）は含めない（引き続きRails版にルーティング）
  - この変更により、ページ編集関連のリクエストがRails版ではなくGo版で処理されるようになる
  - 想定ファイル数: 実装 ~5 ファイル（ルーティング 1 + リバースプロキシ 1 + テンプレート 1 + フロントエンド 2）, テスト 1 ファイル
  - 想定行数: 実装 ~50 行（変更）, テスト ~80 行
  - 依存: 5-1, 5-4, 6-2, 7-2, 8-2, 8-4, 8-5, 8b-2

### フェーズ 10: Rails版の実装の削除

- [ ] **10-1**: [Rails] ページ編集・公開関連のコントローラー・サービス・フォーム・ビューの削除
  - `app/controllers/pages/edit_controller.rb`, `app/controllers/pages/update_controller.rb` を削除
  - `app/controllers/draft_pages/update_controller.rb` を削除
  - `app/controllers/page_locations/index_controller.rb` を削除
  - `app/services/pages/update_service.rb`, `app/services/draft_pages/update_service.rb` を削除
  - `app/forms/pages/edit_form.rb` を削除
  - `app/views/pages/edit_view.html.erb`, `app/views/draft_pages/update_view.html.erb` を削除
  - 関連するルーティング・テスト・翻訳を削除
  - レコード（`PageRecord`, `DraftPageRecord`）やPageable concernは他機能で使用されているため削除しない
  - ページ表示関連（`ShowController`等）の削除は [ページ表示画面のGo移行](../2_todo/page-show-go-migration.md) で行う
  - 想定ファイル数: 実装 8 ファイル削除, テスト 3 ファイル削除
  - 想定行数: 実装 ~-500 行, テスト ~-300 行
  - 依存: 9-1

### フェーズ N: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [ ] **N-1**: 仕様書の作成・更新
  - `docs/specs/page/edit.md` にページ編集の仕様書を作成する
  - `docs/specs/page/wikilink.md` にWikiリンクの仕様書を作成する
  - `docs/specs/page/wikilink-completion.md` にWikiリンク補完の仕様書を作成する
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する
