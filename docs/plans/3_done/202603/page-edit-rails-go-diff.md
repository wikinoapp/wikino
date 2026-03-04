# ページ編集 Rails版 vs Go版 差分分析

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

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 仕様書

<!--
- 既存機能を変更する場合: 変更対象の仕様書へのリンクを記載してください
- 新しい機能の場合: タスク完了後に作成予定の仕様書のパスを記載してください
-->

- [ページ編集 仕様書](../specs/page/edit.md)（タスク完了後に作成予定）
- [ページ編集画面のGo移行 作業計画書](page-edit-go-migration.md)（親タスク）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

ページ編集画面のRailsからGoへの移行にあたり、Rails版とGo版のデータベース操作の差分を洗い出し、データ不整合を防ぐための修正計画を策定する。

調査の結果、Go版で以下の差分が確認された。これらを修正し、Rails版と同等のデータ整合性を実現する。

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

- Go版のページ編集がRails版と同等のデータベース操作を行い、データ不整合が発生しないこと

### 非機能要件

<!--
必要に応じて以下のような項目を追加してください：
- セキュリティ（認証、認可、暗号化、監査ログなど）
- パフォーマンス（応答時間、スループット、リソース使用量など）
- ユーザビリティ（UX）（使いやすさ、わかりやすさ、アクセシビリティなど）
- 可用性・信頼性（稼働率、障害時の挙動、エラーハンドリングなど）
- 保守性（テストのしやすさ、コードの読みやすさ、ドキュメントなど）

不要な場合はこのセクション全体を削除してください。
-->

なし

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

### Go版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン（**ファイル名は標準の9種類のみ**）
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

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

### 差分調査結果

Rails版とGo版のページ編集フローにおけるデータベース操作を比較した結果、以下の差分が確認された。

#### 差分一覧

| #   | 差分                                                           | 重大度 | 操作タイミング       |
| --- | -------------------------------------------------------------- | ------ | -------------------- |
| 1   | Wikiリンクによるページ自動作成時に`page_editors`が作成されない | 高     | 自動保存・公開の両方 |
| 2   | 廃棄済みページと同名のWikiリンクでページ作成が失敗する         | 高     | 自動保存・公開の両方 |
| 3   | ページ番号の採番戦略が異なる（悲観ロック vs 楽観リトライ）     | 低     | 自動保存・公開の両方 |

#### 差分1: page_editorsが作成されない（重大度: 高）

**Rails版の挙動**:

Wikiリンク `[[ページ名]]` によりページが自動作成される際、`create_linked_page!` メソッド（`rails/app/records/space_member_record.rb`）で以下が行われる:

1. `pages` テーブルに新しいページを `first_or_create!` で作成
2. `page_editors` テーブルに編集者（= Wikiリンクを書いたユーザー）を `first_or_create!` で作成

これは自動保存時（`DraftPages::UpdateService`）と公開時（`Pages::UpdateService`）の両方で実行される。

**Go版の挙動**:

`resolveAndCreateLinkedPages`（`go/internal/usecase/auto_save_draft_page.go`）では:

1. `pages` テーブルに新しいページを `CreateLinkedPage` で作成
2. `page_editors` テーブルへの書き込みは**行われない**

`AutoSaveDraftPageUsecase` は `pageEditorRepo` を依存に持っていない。`PublishPageUsecase` は `pageEditorRepo` を持っているが、公開対象のページに対してのみ使用し、自動作成されたリンク先ページに対しては使用していない。

**影響**:

- 自動作成されたページに `page_editors` レコードが存在しない
- `page_editors` テーブルを参照する機能（編集者一覧表示など）で、自動作成ページの初期編集者が表示されない

**修正方針**:

`resolveAndCreateLinkedPages` 内で、ページを新規作成した場合に `page_editors` レコードも作成する。`AutoSaveDraftPageUsecase` と `PublishPageUsecase` の両方に `pageEditorRepo` を渡し、`resolveAndCreateLinkedPages` でページ新規作成時に `pageEditorRepo.FindOrCreate` を呼び出す。

#### 差分2: 廃棄済みページと同名のWikiリンクで失敗（重大度: 高）

**Rails版の挙動**:

`first_or_create!` は `discarded_at` でフィルタリングしない。つまり、廃棄済みページ（`discarded_at IS NOT NULL`）も検索対象に含まれる。同じ `topic_id + title` の廃棄済みページが存在する場合、そのページが返される（再利用される）。

**Go版の挙動**:

`FindByTopicAndTitle`（`go/db/queries/pages.sql`）のSQLに `AND discarded_at IS NULL` が含まれている:

```sql
SELECT * FROM pages
WHERE topic_id = $1 AND title = $2 AND space_id = $3 AND discarded_at IS NULL;
```

廃棄済みページは検索にヒットしないため、`CreateLinkedPage` による INSERT が実行される。しかし、DBのユニークインデックス `(topic_id, title)` は `discarded_at` を条件に含んでいないため、ユニーク制約違反が発生する。リトライ（最大3回）しても同じ結果になり、最終的にエラーとなる。

**影響**:

- 廃棄済みページと同名のWikiリンクを含む本文を自動保存・公開しようとするとエラーが発生する
- ユーザーは該当のWikiリンクを削除するか、異なるタイトルに変更する必要がある

**修正方針**:

Rails版と同様に、`FindByTopicAndTitle` クエリから `AND discarded_at IS NULL` 条件を削除する。廃棄済みページであっても同名のページが存在すればそれを返す（リンク先として使用する）。

注: これにより廃棄済みページへのWikiリンクが張られることになるが、Rails版と同じ挙動であり、廃棄済みページの扱いは別途検討する。

#### 差分3: ページ番号の採番戦略（重大度: 低）

**Rails版の挙動**:

`sequenced` gem が `LOCK TABLE pages IN EXCLUSIVE MODE` でテーブルレベルの排他ロックを取得し、`MAX(number) + 1` で採番する。

**Go版の挙動**:

ロックなしで `COALESCE(MAX(number), 0) + 1` を実行し、ユニーク制約違反が発生した場合は最大3回リトライする（楽観的並行制御）。

**影響**:

- 高並行性の環境では、Go版で3回のリトライを超えてエラーになる可能性がある
- 現状のワークロード（同時にページが大量作成されるケースは稀）では問題にならない

**修正方針**:

現状維持。楽観リトライで十分であり、テーブルレベルの排他ロックはパフォーマンスへの影響が大きい。問題が顕在化した場合に対応する。

### 操作別のデータベース操作比較

以下に、ページ編集フローの各操作におけるRails版とGo版のデータベース操作を詳細に比較する。

#### 編集画面を開く

読み取り操作のみ。Rails版とGo版で差分なし。

| テーブル                    | 操作   | Rails | Go  |
| --------------------------- | ------ | ----- | --- |
| `spaces`                    | SELECT | ○     | ○   |
| `space_members`             | SELECT | ○     | ○   |
| `pages`                     | SELECT | ○     | ○   |
| `topic_members`             | SELECT | ○     | ○   |
| `topics`                    | SELECT | ○     | ○   |
| `draft_pages`               | SELECT | ○     | ○   |
| `pages`（リンク一覧）       | SELECT | ○     | ○   |
| `pages`（バックリンク一覧） | SELECT | ○     | ○   |

#### 自動保存（DraftPage更新）

| テーブル           | 操作                                                           | Rails | Go    | 差分      |
| ------------------ | -------------------------------------------------------------- | ----- | ----- | --------- |
| `draft_pages`      | SELECT / INSERT（find-or-create）                              | ○     | ○     | なし      |
| `topics`           | SELECT（トピック解決）                                         | ○     | ○     | なし      |
| `draft_pages`      | UPDATE（title, body, body_html, linked_page_ids, modified_at） | ○     | ○     | なし      |
| `topics`           | SELECT（Wikiリンク内のトピック名解決）                         | ○     | ○     | なし      |
| `pages`            | SELECT / INSERT（Wikiリンクによるページ自動作成）              | ○     | ○     | なし      |
| **`page_editors`** | **SELECT / INSERT（自動作成ページの編集者登録）**              | **○** | **✗** | **差分1** |

#### 公開（Page更新）

| テーブル                     | 操作                                                                                                                 | Rails | Go    | 差分      |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------- | ----- | ----- | --------- |
| `pages`                      | UPDATE（title, body, body_html, topic_id, linked_page_ids, modified_at, published_at, featured_image_attachment_id） | ○     | ○     | なし      |
| `page_editors`               | SELECT / INSERT / UPDATE（公開対象ページの編集者）                                                                   | ○     | ○     | なし      |
| `page_revisions`             | INSERT（スナップショット作成）                                                                                       | ○     | ○     | なし      |
| `topics`                     | SELECT（Wikiリンク内のトピック名解決）                                                                               | ○     | ○     | なし      |
| `pages`                      | SELECT / INSERT（Wikiリンクによるページ自動作成）                                                                    | ○     | ○     | なし      |
| **`page_editors`**           | **SELECT / INSERT（自動作成ページの編集者登録）**                                                                    | **○** | **✗** | **差分1** |
| `draft_pages`                | DELETE（下書き削除）                                                                                                 | ○     | ○     | なし      |
| `topic_members`              | UPDATE（last_page_modified_at）                                                                                      | ○     | ○     | なし      |
| `page_attachment_references` | SELECT / INSERT / DELETE（添付ファイル参照の差分同期）                                                               | ○     | ○     | なし      |
| `attachments`                | SELECT（添付ファイル存在確認）                                                                                       | ○     | ○     | なし      |
| `pages`                      | UPDATE（featured_image_attachment_id）                                                                               | ○     | ○     | なし      |

### Wikiリンク処理の比較

| 処理                             | Rails                               | Go                                                          | 差分                                   |
| -------------------------------- | ----------------------------------- | ----------------------------------------------------------- | -------------------------------------- |
| `[[ページ名]]` パース            | `PageLocationKey.scan_text`         | `markup.ScanWikilinks`                                      | なし                                   |
| `[[トピック名/ページ名]]` パース | `/` で分割（最大2パーツ）           | `/` で分割（最大2パーツ）                                   | なし                                   |
| トピック名の一括解決             | 個別にクエリ                        | `FindBySpaceAndNames` で一括                                | パフォーマンス改善（Go版の方が効率的） |
| ページの find-or-create          | `first_or_create!`（discarded含む） | `FindByTopicAndTitle` + `CreateLinkedPage`（discarded除外） | **差分2**                              |
| page_editors 作成                | `first_or_create!`                  | なし                                                        | **差分1**                              |
| HTML内のWikiリンク置換           | Markupクラス内で実行                | `markup.ReplaceWikilinks`                                   | なし                                   |
| linkedPageIDs 更新               | `update!(linked_page_ids: ...)`     | DraftPage/Page の UPDATE に含む                             | なし                                   |

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### 差分2について: ユニークインデックスに `discarded_at IS NULL` の部分インデックスを使う方針

`FindByTopicAndTitle` は `discarded_at IS NULL` を維持しつつ、ユニークインデックスを部分インデックスに変更する方法を検討したが、以下の理由で採用しなかった:

- DBスキーマの変更が必要になり、Rails版にも影響が及ぶ
- Rails版では `first_or_create!` で廃棄済みページを含めて検索しており、この挙動に合わせる方がシンプル
- 廃棄済みページの扱いは別途検討すべきであり、今回のスコープではRails版と同じ挙動に揃える

### 差分3について: Go版でもテーブルロックを使う方針

Go版でもRails版と同様にテーブルレベルの排他ロック（`LOCK TABLE pages IN EXCLUSIVE MODE`）を使う方針を検討したが、以下の理由で採用しなかった:

- テーブルレベルのロックはパフォーマンスへの影響が大きい
- 現状のワークロードでは楽観リトライで十分
- 問題が顕在化した場合に `SELECT ... FOR UPDATE` によるアドバイザリーロックなどの軽量な方法を検討する

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

### フェーズ 1: 差分修正

- [x] **1-1**: [Go] Wikiリンクによるページ自動作成時にpage_editorsを作成する
  - `resolveAndCreateLinkedPages` に `pageEditorRepo` を渡す
  - ページ新規作成時に `pageEditorRepo.FindOrCreate` を呼び出す
  - `AutoSaveDraftPageUsecase` に `pageEditorRepo` を依存追加
  - `PublishPageUsecase` の `resolveAndCreateLinkedPages` 呼び出しにも `pageEditorRepo` を渡す
  - テスト: 自動保存・公開の両方で自動作成ページに `page_editors` が作成されることを検証
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 120 行（実装 60 行 + テスト 60 行）

- [x] **1-2**: [Go] FindByTopicAndTitleからdiscarded_atフィルタを削除する
  - `go/db/queries/pages.sql` の `FindPageByTopicAndTitle` クエリから `AND discarded_at IS NULL` を削除
  - `make sqlc-generate` でコード再生成
  - テスト: 廃棄済みページと同名のWikiリンクで自動保存・公開が成功することを検証
  - **想定ファイル数**: 約 4 ファイル（実装 2 + テスト 2）
  - **想定行数**: 約 80 行（実装 10 行 + テスト 70 行）

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **ページ番号採番戦略の変更**: Go版の楽観リトライ方式で現状問題ないため、テーブルロックへの変更は行わない
- **廃棄済みページの扱いの改善**: 廃棄済みページへのWikiリンクの表示方法などは別途検討する

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [ページ編集画面のGo移行 作業計画書](page-edit-go-migration.md) - 親タスク
- Rails版のページ編集関連ソースコード:
  - `rails/app/services/draft_pages/update_service.rb` - 自動保存サービス
  - `rails/app/services/pages/update_service.rb` - 公開サービス
  - `rails/app/records/record_concerns/pageable.rb` - Wikiリンク処理（`link!`メソッド）
  - `rails/app/records/space_member_record.rb` - ページ自動作成（`create_linked_page!`メソッド）
- Go版のページ編集関連ソースコード:
  - `go/internal/usecase/auto_save_draft_page.go` - 自動保存ユースケース
  - `go/internal/usecase/publish_page.go` - 公開ユースケース
  - `go/internal/markup/wikilink.go` - Wikiリンク解析
  - `go/db/queries/pages.sql` - ページ関連SQLクエリ
