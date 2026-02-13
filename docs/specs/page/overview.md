# ページ 仕様書

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

**タスクリストとの関係**:
- 仕様の変更が必要な場合、「何をどう変えるか」は `docs/tasks/` のタスクリストに記述します
- タスク完了後に、この仕様書を新しい状態に更新してください

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 実装ガイドラインの参照

<!--
**重要**: 仕様書を作成する前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
仕様書作成の段階でガイドラインに準拠していることを確認してください。
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

## 概要

<!--
ガイドライン:
- この機能が現在「どのように動いているか」を簡潔に説明
- なぜこの仕組みになっているかの背景も記述
- 2-3段落程度で簡潔に
-->

ページはWikinoにおけるコンテンツの基本単位である。各ページはPageRevision（UIでは「バージョン」と表示、一覧は「編集履歴」と表示）による変更管理を持ち、「下書き（DraftPage）」と「公開済みのコンテンツ（PageRevision）」を分離した編集フローを提供する。

編集内容はDraftPageに自動保存され、ユーザーが明示的に「下書き保存」すると下書きのバージョン（DraftPageRevision）が作成される。さらに「公開」するとバージョン（PageRevision）が作成されて他の人にも見えるようになり、編集履歴として追跡できるようになる。

**目的**:

- ページの変更履歴を追跡可能にする
- 編集中の内容が他のユーザーに影響を与えない安全な編集環境を提供する
- 将来的に複数ユーザーによるリアルタイム同時編集を可能にする

**背景**:

- Gitのコミットモデルを参考に、「自動保存」「下書き保存」「公開」の3段階の設計を採用した
- 将来的にはCRDTを使用してリアルタイム同時編集の実現を計画している

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

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

> **注**: DraftPageRevisionは計画中の機能であり、現在のDBスキーマにはまだテーブルが存在しない。実装時にDBマイグレーションが必要となる。

下書きのバージョン。ユーザーが明示的に保存操作を行ったタイミング、またはAIが下書きを編集したタイミングで作成される。DraftPageに紐づき、下書き内の変更を段階的に確認するために使用する。

| フィールド       | 型       | 説明                                   |
| ---------------- | -------- | -------------------------------------- |
| `id`             | `string` | 一意の識別子（ULID）                   |
| `draftPageId` | `string` | 所属するDraftPageのID               |
| `spaceMemberId`  | `string` | 作成したスペースメンバーのID           |
| `title`          | `string` | バージョン作成時点のページタイトル     |
| `body`           | `string` | バージョン作成時点のMarkdown本文       |
| `bodyHtml`       | `string` | バージョン作成時点のHTML本文           |
| `createdAt`      | `Date`   | 作成日時                               |

DraftPageRevisionの設計意図:

- **段階的な差分確認**: 公開版との差分だけでなく、DraftPageRevision間の差分を確認できる。変更量が多い場合でも、各DraftPageRevision間の差分を見ることで変更内容を追いやすくなる
- **明示的な作成**: 自動保存（DraftPageへの永続化）のたびに作成するのではなく、ユーザーの明示的な「下書き保存」操作またはAI編集時にのみ作成する。これにより不要なバージョンの増大を防ぐ
- **非公開**: DraftPageと同様、自分だけが見える。公開時に作成されるPageRevisionとは異なる

### ページを開く

- DraftPage（自分用）が存在しない場合、Pageのtitle・bodyを表示する
- 「編集する」をクリックすると自分用のDraftPageを作成する（Pageの現在のtitle・body・bodyHtmlをコピー）
- DraftPage（自分用）が存在する場合、自分のDraftPageの内容を表示する（編集中の状態）

### 編集

- ページの本文とタイトルの両方を編集できる
- タイトルはDraftPageの`title`フィールドとして保持される
- タイトルの変更は公開時に反映される（編集中は他のユーザーには見えない）

#### 単独編集

- 自分のDraftPageだけを更新する
- 他のユーザーには見えない
- 編集内容はDraftPageのbody・bodyHtml・titleに自動保存される

#### 同時編集【計画中】

> **注**: 同時編集はCRDTの導入により実現する計画中の機能。現在のDBスキーマでは対応していない。

- 同じページを編集中の他のユーザーとCRDTセッションを共有する
- リアルタイムで変更が同期される
- 編集終了時、各ユーザーのDraftPageにそれぞれ保存される

### 下書き保存する【計画中】

> **注**: DraftPageRevisionは計画中の機能であり、現在は未実装。

下書き保存操作により、下書きのバージョン（DraftPageRevision）を作成する。自動保存（DraftPageへの永続化）とは異なり、ユーザーの明示的な操作によって行われる。以下のタイミングで作成される。

#### ユーザーによる明示的な下書き保存

- ユーザーが編集画面で「下書き保存」操作を行うと、その時点の下書き内容でDraftPageRevisionが作成される
- 自動保存ではDraftPageRevisionは作成されない

#### AIによる編集時の自動作成

- AIがMCP経由で下書きを編集した場合、編集後に自動的にDraftPageRevisionが作成される
- これにより、AIが何を変更したかを下書きの編集履歴で差分として確認できる

#### 下書きの編集履歴での差分確認

- ユーザーは下書きの編集履歴から任意の2つのバージョン間の差分を確認できる
- 直前のバージョンとの差分を確認することで、各回の変更内容を把握できる
- 公開版（Pageの現在の内容）との差分も確認できる

### 公開する

#### 公開処理

1. DraftPageの内容とPageの現在の内容との差分を確認する（本文およびタイトル）
2. DraftPageの内容をPageに反映する（title、body、bodyHtml、linkedPageIds、modifiedAt）
3. 新しいPageRevisionを作成する（この時点のタイトルと本文のスナップショットを記録）
4. タイトルが変更されている場合:
   a. 同一スペース内の他のページの本文に含まれる `[[旧タイトル]]` を `[[新タイトル]]` に自動的に書き換える
5. PageのpublishedAtを更新する
6. 自分のDraftPageを削除する

### 破棄する

- 自分のDraftPageを削除するだけ
- 公開版（Pageの現在の内容）に戻る
- 他のユーザーのDraftPageには影響しない

### Claudeによる編集

Claude DesktopからMCP経由でページを編集できる。指定されたスペースメンバーのDraftPageを更新する。

- Claudeの編集は指定スペースメンバーのDraftPageを更新する
- DraftPageRevision実装後は、編集後にDraftPageRevisionが自動作成され、AIが何を変更したかを下書きの編集履歴で差分として確認できるようになる【計画中】
- 公開（Publish）はユーザーが明示的に行う
- 気に入らなければ破棄して公開版（Pageの現在の内容）に戻せる
- 他のスペースメンバーのDraftPageには影響しない

### 編集履歴

同一ページのPageRevisionを`createdAt`の降順に並べることで、編集履歴を時系列で表示できる。

表示例:

```
● バージョン 3    田中太郎     3分前
○ バージョン 2    鈴木花子     1時間前
○ バージョン 1    田中太郎     3日前
```

### ロールバック

特定のバージョンに戻す場合:

1. 対象のPageRevisionのtitle・body・bodyHtmlを取得する
2. Pageの内容を対象PageRevisionの内容で更新する（title、body、bodyHtml、modifiedAt）
3. 新しいPageRevisionを作成する（ロールバック後の状態をスナップショットとして記録、履歴は保持される）

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

// DraftPageRevision は計画中の機能（現在のDBスキーマには未存在）
type DraftPageRevision struct {
	ID             string
	DraftPageID string
	SpaceMemberID  string
	Title          string    // バージョン作成時点のページタイトル
	Body           string    // バージョン作成時点のMarkdown本文
	BodyHTML       string    // バージョン作成時点のHTML本文
	CreatedAt      time.Time
}
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
 │ ...         │        │ (スナップ    │  差分   │              │
 └──────────────┘        │  ショット)  │  確認   └──────────────┘
       ↑                  └──────────────┘               │
  自動保存                (自分だけ見える)           PageRevision作成
  (自分だけ見える)                                  (他の人に見える)
```

### 同時編集のデータフロー【計画中】

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

### Claudeによる編集のデータフロー

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

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### ページタイトルの変更を独立した操作として提供する方式

ページの本文編集とは別に、タイトル変更を独立した操作として提供する方式を検討したが、以下の理由から編集に統合する方式を採用した。

- 本文編集時にも `[[新しいページ]]` のようなリンク記法を入力するとページタイトルの参照が生まれるため、タイトル変更だけを「リンク参照の書き換えを伴う特別な操作」として分離する根拠が弱い
- リンク参照の書き換えをリビジョン公開時（直接編集の公開時・編集提案の承認時）に行うことで、書き換えタイミングの複雑さは解消される
- タイトル変更を別操作にすると、ページを整理する際に「本文を編集してから、別画面でタイトルも変更する」という二度手間が生じる
- Wikiの性質上、ページの分割・統合・改名は頻繁に行われるため、タイトルは気軽に変更できるほうがよい
- トピック移動とは異なり、タイトル変更には権限モデルの複雑化（移動先トピックの権限チェック等）が伴わないため、編集と分離する必然性が低い

### 自動保存のたびにDraftPageRevisionを作成する方式

Jujutsu（jj-vcs）のように、下書きが自動保存されるたびに自動でDraftPageRevisionを作成する方式を検討した。しかし、以下の理由から、ユーザーの明示的な「下書き保存」操作でのみDraftPageRevisionを作成する方式を採用した。

- 自動保存のたびにDraftPageRevisionを作成すると大量のレコードが生成される（将来的にCRDTによるリアルタイム保存を導入した場合、特に顕著になる）
- ユーザーが意図したタイミングで下書き保存するほうが、各DraftPageRevision間の差分が意味のある単位になる

AIによる編集時の自動作成は例外として許可した。AIの編集は1回の操作が明確な単位（1つの改善提案など）を持つため、自動作成しても不要なバージョンが増大しにくい。

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

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->
