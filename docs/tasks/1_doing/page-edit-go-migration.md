# ページ編集画面のGo移行 タスクリスト

<!--
このテンプレートの使い方:
1. このファイルを `docs/tasks/` ディレクトリにコピー
   例: cp docs/tasks/template.md docs/tasks/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残してください

**タスクリストの性質**:
- タスクリストは「何をどう変えるか」という変更内容を記述するドキュメントです
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

- [@docs/tasks/2_todo/edit-suggestion.md](edit-suggestion.md) - 編集提案機能（本タスクが前提）
- [@docs/tasks/2_todo/page-move.md](page-move.md) - ページの移動機能（本タスクと並行可能）
- [@docs/tasks/2_todo/draft-page-revision.md](draft-page-revision.md) - DraftPageRevisionの実装（本タスクが前提）
- [@docs/tasks/2_todo/title-change-link-rewrite.md](title-change-link-rewrite.md) - タイトル変更時のリンク自動書き換え（本タスクが前提）
- [@docs/tasks/2_todo/page-revision-history.md](page-revision-history.md) - 編集履歴画面・ロールバック（本タスクが前提）
- [@docs/tasks/2_todo/draft-page-discard.md](draft-page-discard.md) - 下書き破棄機能（本タスクが前提）
- [@docs/tasks/2_todo/publish-diff-confirmation.md](publish-diff-confirmation.md) - 公開前の差分確認（本タスクが前提）

## 要件

<!--
ガイドライン:
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な要件（セキュリティ、パフォーマンスなど）も記述
-->

- ユーザーはページを表示できる
- ユーザーはページの編集画面を開ける
- ユーザーの編集内容はDraftPageに自動保存される
- ユーザーはDraftPageの内容を公開できる（PageRevisionの作成）
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

| 操作                          | 英語        | 対象モデル     | 意味                                                                                 |
| ----------------------------- | ----------- | -------------- | ------------------------------------------------------------------------------------ |
| **自動保存 (Auto-save)**      | Auto-save   | DraftPage      | 編集内容をDraftPageに自動的に永続化する。ユーザーが意識しない裏側の処理。              |
| **下書き保存 (Save Draft)**   | Save Draft  | DraftPageRevision  | 下書きのバージョン（DraftPageRevision）を作成する。ユーザーの明示的な操作（Ctrl+S相当）。 |
| **公開 (Publish)**            | Publish     | PageRevision   | 他の人に見えるようにする。このタイミングでバージョン（PageRevision）が作成される。     |

Gitとの対応:

| Wikino           | Git                          |
| ---------------- | ---------------------------- |
| 自動保存         | ファイル編集（working tree） |
| 下書き保存       | commit                       |
| 公開             | push                         |

### 日本語での表示名

内部モデル名とUI上の日本語表示名の対応を以下に定義する。

| モデル名（内部）      | 単数形（UI表示）         | 複数形・一覧（UI表示）       |
| --------------------- | ------------------------ | ---------------------------- |
| **PageRevision**      | バージョン               | 編集履歴                     |
| **DraftPageRevision**     | 下書きのバージョン       | 下書きの編集履歴             |

- **単数形（バージョン）**: 個々のPageRevision/DraftPageRevisionを指すときに使用する。例: 「バージョン 3」「このバージョンとの差分」「下書き保存する」
- **複数形・一覧（編集履歴）**: PageRevisionの一覧や履歴を表示するときに使用する。例: 「編集履歴を見る」「下書きの編集履歴」

### データモデル

以下のモデルをGo版で実装する:

- **Page** - ページのデータ管理
- **PageRevision** - 公開されたページのスナップショット
- **DraftPage** - ページの下書き（スペースメンバーごとに分離）

#### Page

ページのデータを管理する。最新のタイトル・本文は`pages`テーブルに直接格納しており、`page_revisions`テーブルをJOINせずにページ表示が可能な非正規化設計を採用している。これはページ表示が最も頻繁な操作であるため、読み取りパフォーマンスを優先した設計である。

| フィールド                  | 型                | 説明                                                          |
| --------------------------- | ----------------- | ------------------------------------------------------------- |
| `id`                        | `string`          | 一意の識別子（ULID）                                          |
| `spaceId`                   | `string`          | 所属するスペースのID                                          |
| `topicId`                   | `string`          | 所属するトピックのID                                          |
| `number`                    | `number`          | URLに使用されるページ番号（スペース内でユニーク）             |
| `title`                     | `string \| null` | ページタイトル（トピック内でユニーク、大文字小文字を区別しない）|
| `body`                      | `string`          | Markdownの本文                                                |
| `bodyHtml`                  | `string`          | HTMLに変換された本文                                          |
| `linkedPageIds`             | `string[]`        | 本文中のリンクで参照されているページIDのリスト                 |
| `modifiedAt`                | `Date`            | 内容の更新日時                                                |
| `publishedAt`               | `Date \| null`   | 公開日時（nullなら非公開）                                    |
| `trashedAt`                 | `Date \| null`   | ゴミ箱に入れた日時                                            |
| `createdAt`                 | `Date`            | 作成日時                                                      |
| `updatedAt`                 | `Date`            | レコードの更新日時                                            |
| `pinnedAt`                  | `Date \| null`   | ピン留め日時                                                  |
| `discardedAt`               | `Date \| null`   | 廃棄日時                                                      |
| `featuredImageAttachmentId` | `string \| null` | アイキャッチ画像の添付ファイルID                              |

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

| フィールド      | 型                | 説明                                          |
| --------------- | ----------------- | --------------------------------------------- |
| `id`            | `string`          | 一意の識別子（ULID）                          |
| `spaceId`       | `string`          | 所属するスペースのID                          |
| `pageId`        | `string`          | 対象ページのID                                |
| `spaceMemberId` | `string`          | スペースメンバーのID（メンバーごとに分離）    |
| `topicId`       | `string`          | トピックのID                                  |
| `title`         | `string \| null` | 編集中のページタイトル                        |
| `body`          | `string`          | 編集中のMarkdown本文                          |
| `bodyHtml`      | `string`          | 編集中のHTML本文                              |
| `linkedPageIds` | `string[]`        | 本文中のリンクで参照されているページIDのリスト |
| `modifiedAt`    | `Date`            | 内容の更新日時                                |
| `createdAt`     | `Date`            | 作成日時                                      |
| `updatedAt`     | `Date`            | レコードの更新日時                            |

DraftPageの設計意図:

- **スペースメンバーごとに分離**: 自分の下書きが他の人に見えない

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

// DraftPageRevision は計画中の機能
// 詳細は docs/tasks/2_todo/draft-page-revision.md を参照
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

- `GET /s/:space_identifier/pages/:page_number` - ページ表示
- `GET /s/:space_identifier/pages/:page_number/edit` - ページ編集画面
- `PATCH /s/:space_identifier/pages/:page_number/draft_page` - 下書き更新（自動保存）
- `PATCH /s/:space_identifier/pages/:page_number` - ページ公開

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

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

（タスクリストは未作成）

### フェーズ N: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針をタスクリストから転記・整理する
-->

- [ ] **N-1**: 仕様書の作成・更新
  - `docs/specs/page/edit.md` に仕様書を作成する
  - タスクリストの概要・要件・設計・採用しなかった方針を仕様書に反映する
