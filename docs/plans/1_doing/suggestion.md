# 編集提案 作業計画書

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

- [編集提案 仕様書](../specs/suggestion/overview.md)（タスク完了後に作成予定）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

編集提案機能は、GitHubのPull Requestのように、ページに対して変更案を提出し、レビューを経て反映できる機能である。直接編集する代わりに「こう変更してはどうでしょうか」という形で提案できる。

トピック内でページの作成や既存ページの編集を提案でき、複数のページの変更を一つの編集提案にまとめることもできる。

**目的**:

- 編集に対する心理的ハードルを下げ、気軽に貢献できるようにする
- AIによる文書更新を積極的に活用し、人間が変更箇所を確認して取捨選択できるようにする
- レビュープロセスにより、ドキュメントの品質を維持・向上する
- 非技術者でもGit的なワークフローを利用できるようにする

**背景**:

- 「たぶんこの記述は間違っているけど、自分の修正も合ってるか自信が持てない」という場面で、直接編集は心理的ハードルが高い。「提案」という形であれば、却下されても精神的ダメージが少なく、気軽に貢献できる
- 企業の公式ドキュメントやOSSの技術仕様書など、正確性が重要な文書では、誤った修正の影響を防ぐためにレビュープロセスが求められている
- AIに文書をガンガン更新してもらいたいが、すべてを無条件に反映するのではなく、変更内容を確認してから取捨選択したい。編集提案はそのレビューの仕組みとして機能する

### 利用シーン

#### 個人利用

今後提供予定のCLIやMCPサーバーと組み合わせることで、生成AIにドキュメントの改善提案を作ってもらい、差分を確認して良さそうなら適用するというワークフローが実現できる。Gitを使わずに、AIが生成したテキストの差分確認・適用・破棄ができるようになる。

#### チーム利用

- メンバー間でドキュメントの改善案を提案し合える
- レビュープロセスを通じて、ドキュメントの品質を保てる
- 責任の所在を明確にしながら、協働的な編集ができる

#### OSSプロジェクト

- 初心者コントリビューターが参加しやすくなる
- ドキュメントへの貢献のハードルが下がる
- レビューを通じて品質を維持しながら、多様な貢献を受け入れられる

## 要件

<!--
ガイドライン:
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な要件（セキュリティ、パフォーマンスなど）も記述
-->

- ユーザーはトピック内でページの作成や既存ページの編集を提案できる
- 編集提案は1つのトピック内のページに限定される。複数トピックにまたがる編集提案は作成できない
- 複数のページの変更を一つの編集提案にまとめることができる
- 編集提案ではページのトピック変更は対象外とする。トピックの変更（[@docs/plans/page-move.md](/workspace/docs/plans/page-move.md)）は独立した操作として提供する
- 編集提案にはタイトルの変更を含めることができる。タイトル変更に伴うリンク参照の書き換え（`[[旧タイトル]]` → `[[新タイトル]]`）は、編集提案の承認時に自動的に行われる
- 編集提案の作成はスペースメンバーのみ可能。公開トピックであってもスペースへの参加が必要となる
- 編集提案作成者がスペースから退会しても、作成済みの編集提案は保持される
- ページ編集画面の「下書き保存」ボタンの右側にドロップダウンメニューを表示するアイコンがあり、クリックすると「下書き保存して編集提案を作成する...」アクションが選択できる。このアクションを実行すると、下書き保存後に下書き一覧画面にはリダイレクトせず、保存した下書きページが選択された状態で編集提案作成画面に直接遷移する

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

### 命名

- **モデル名（英語）**: Suggestion, SuggestionPage, SuggestionPageRevision, SuggestionComment
- **テーブル名**: `suggestions`, `suggestion_pages`, `suggestion_page_revisions`, `suggestion_comments`
- **日本語UI表示名**: 編集提案

Google Docsの「Suggestions」と同じ用語を採用した。日本語名は「提案」だと一般名詞すぎて固有名詞感がないため、「編集提案」を維持する。

### 関連モデル

編集提案機能は、ページのリビジョン管理システム（Page, PageRevision, DraftPage）を基盤として構築する。ページモデルの詳細（データモデル、編集フロー、データ構造、データフローなど）は [ページ編集画面のGo移行](page-edit-go-migration.md) の「設計」セクションを参照。

### Gitモデルとの対応

編集提案機能はGitHubのPull Requestモデルを参考に設計する。Gitの概念とWikinoのモデルの対応は以下の通り:

| Git                   | Wikino                 | 説明                                     |
| --------------------- | ---------------------- | ---------------------------------------- |
| remote main branch    | Page                   | 公開済みのページ（最新の正式な内容）     |
| local working tree    | DraftPage              | スペースメンバーごとの個人の下書き       |
| remote feature branch | SuggestionPage         | 編集提案に紐づくページの変更内容（共有） |
| commits on branch     | SuggestionPageRevision | 編集提案ページへの変更履歴               |
| Pull Request          | Suggestion             | 変更のレビュー・反映を管理する単位       |

**重要な設計判断**: SuggestionPageがGitの「リモートフィーチャーブランチ」に相当するリソースとして機能する。これにより、複数のスペースメンバーが同じ編集提案に対して変更を加えることが可能になる。DraftPageはあくまで個人のワーキングツリーであり、編集提案の共有リソースとは分離する。

### 編集提案の共同編集

編集提案は複数のスペースメンバーが共同で作成・編集できる設計とする。

#### DraftPageと編集提案の連携

編集提案のページを編集する際は、既存のDraftPageの仕組みを再利用する。`draft_pages` テーブルに `suggestion_page_id`（nullable FK → `suggestion_pages`）を追加し、DraftPageが「どのブランチをチェックアウトしているか」を表現する。

| `suggestion_page_id` | DraftPageの役割      | 自動保存先 | 内容の初期化元                         |
| -------------------- | -------------------- | ---------- | -------------------------------------- |
| NULL                 | 通常のページ編集     | DraftPage  | Pageの現在のコンテンツ                 |
| NOT NULL             | 編集提案のページ編集 | DraftPage  | SuggestionPageRevisionの最新リビジョン |

DraftPageのユニーク制約は `[space_member_id, page_id]` のままとする（`suggestion_page_id` を含めない）。これにより、同一ページに対して通常編集と編集提案の編集を同時に持つことはできない。Gitの「ワーキングツリーは同時に1つのブランチしかチェックアウトできない」のと同じ制約であり、概念モデルがシンプルに保たれる。

#### 通常編集 → 編集提案の編集への切り替え

編集提案からページを編集しようとしたとき、既に通常編集のDraftPage（`suggestion_page_id` がNULL）が存在する場合は確認画面を表示する:

- **編集提案の内容で編集を続ける**: 既存のDraftPageの内容を破棄し、SuggestionPageRevisionの最新リビジョンの内容でDraftPageを初期化する。`suggestion_page_id` を設定する
- **もとの下書きを保持する**: 編集提案の編集を開始しない。既存のDraftPageはそのまま

#### 編集提案の編集 → 通常編集への切り替え

編集提案のページ編集が完了（SuggestionPageRevisionを作成）した後、DraftPageを削除するか `suggestion_page_id` をNULLにクリアする。次回の通常編集時にはPageの内容から新しいDraftPageが作られる。

#### 編集提案ページ編集時のページ編集画面

DraftPageが編集提案にリンクされている場合（`suggestion_page_id` がNOT NULL）、ページ編集画面の表示が変わる:

- 「この下書きへの保存は編集提案 #12345 に反映されます」というメッセージを表示
- 「トピックに公開」ボタンは非表示にする（編集提案にリンク中は直接公開できない）
- 代わりに「編集提案を更新」ボタンを表示 → SuggestionPageRevision作成
- 「下書き保存」ボタンは通常通り表示 → DraftPageRevision作成（編集提案の文脈での下書き保存）

#### 初期リリースの範囲

初期リリースでは以下のフローで運用する:

1. 提案者が自分のDraftPageから編集提案を作成する
2. レビュアーはコメントでフィードバックする
3. 提案者（またはスペースメンバー）が編集提案詳細画面からページの編集を開始する
4. ページ編集画面でDraftPageに自動保存されつつ編集し、「編集提案を更新」でSuggestionPageRevisionを作成する

#### 将来の拡張

- Botメンバーによる編集: API経由でSuggestionPageRevisionを直接作成し、AIによる編集提案の更新を可能にする
- 同時編集: CRDTの導入後、複数のスペースメンバーが同じ編集提案ページをリアルタイムで同時編集できるようにする

### フィーチャーフラグ

編集提案機能はフィーチャーフラグで制御し、フラグが有効なユーザー/デバイスのみがアクセスできるようにする。これにより、実装途中でも develop ブランチにマージでき、段階的に機能を公開できる。

- **フラグ名**: `go_suggestion`
- **制御方式**: リバースプロキシミドルウェアの `featureFlaggedPatterns` にURLパターンを追加し、フラグが有効な場合のみ Go 版で処理する
- **対象URLパターン**: `/s/{space}/topics/{topic}/suggestions` 配下のすべてのパス
- **フラグ無効時の挙動**: Rails 版にプロキシされる（Rails 版に該当機能がないため 404 になる。編集提案は Go 版の新機能であり、Rails 版への移行ではないため問題ない）
- **クリーンアップ**: 機能が安定し全ユーザーに公開した後、フラグを削除し `goHandledPrefixPaths` または `goHandledRegexPatterns` に移動する

### 未決定事項

#### ベースページの乖離（マージコンフリクト）

編集提案作成後にベースとなるPageが更新された場合の扱い。`suggestion_pages.page_revision_id` でベースとなるリビジョンを記録しているため、ベースが古くなったことは検出できる。

- 初期リリースでは「ベースが変わっていても強制的に上書き反映」とする
- 将来的にはコンフリクト検出・手動解決のUIを追加する可能性がある

#### 同一編集提案ページの同時編集

同じ編集提案のページを複数のスペースメンバーが同時に編集した場合の競合。通常のページの同時編集と同じ問題であり、将来的にはCRDTで解決する計画。初期は「最後に保存した人が勝つ（last-write-wins）」とする。

### ステータス

編集提案には以下の4つのステータスを設ける:

1. **下書き** - 作成中の編集提案
2. **オープン** - レビュー待ちの編集提案
3. **反映済み** - トピックに反映された編集提案
4. **クローズ** - 反映されずに閉じられた編集提案

### テーブル設計

- 編集提案テーブル (`suggestions`)
  - id, space_id, topic_id, created_space_member_id, title, body, body_html, status, applied_at, created_at, updated_at
  - bodyはMarkdownで記述し、保存時にページと同じMarkdownパイプライン（Wikiリンク解決含む）でbody_htmlを生成する
  - インデックス: status, [topic_id, status]
- 編集提案ページリビジョンテーブル (`suggestion_page_revisions`)
  - id, space_id, suggestion_page_id, editor_space_member_id, title, body, body_html, created_at, updated_at
  - インデックス: [suggestion_page_id, created_at]
- 編集提案ページテーブル (`suggestion_pages`)
  - id, space_id, suggestion_id, page_id, page_revision_id, latest_revision_id, created_at, updated_at
  - page_id, page_revision_idはoptional（新規ページ作成の場合）
  - ユニークインデックス: [suggestion_id, page_id]
- 編集提案コメントテーブル (`suggestion_comments`)
  - id, space_id, suggestion_id, created_space_member_id, body, body_html, created_at, updated_at

#### 既存テーブルの変更

- 下書きページテーブル (`draft_pages`) にカラムを追加
  - `suggestion_page_id` (nullable, FK → `suggestion_pages`): 編集提案のページ編集時にリンクする。NULLなら通常のページ編集、NOT NULLなら編集提案のページ編集
  - ユニーク制約 `[space_member_id, page_id]` は変更しない

### UI設計

トピック詳細画面:

- トピック画面上部に「ページ」と「編集提案」のタブを表示
- GitHubのCode/Pull requestsタブと同様のUI
- デフォルトは「ページ」が選択されており、トピック内のページが一覧で表示される (現在のトピック詳細画面と同じ)
- 「編集提案」を選択すると、編集提案一覧画面が表示される

編集提案一覧画面:

- スペースメンバーが作成した編集提案をリスト表示
- GitHubのようにオープン/クローズで絞り込み可能
  - オープン表示：下書き・オープンステータスの編集提案
  - クローズ表示：反映済み・クローズステータスの編集提案

編集提案詳細画面:

- 「会話」「編集したページ」の2つのタブ
- デフォルトは「会話」タブがアクティブ
- 「会話」タブ：編集提案の概要とコメント表示
- 「編集したページ」タブ：変更差分の表示
- 「反映する」ボタン（権限がある場合）

ページ編集画面:

- エディタの下に「トピックに公開」と「下書き保存」ボタンが表示されている
- 「トピックに公開」ボタンを押すと:
  - 新しいPageRevisionが作成されて編集内容が反映される
  - 編集したページの詳細画面にリダイレクトする
- 「下書き保存」ボタンを押すと:
  - 新しいDraftPageRevisionが作成されてDraftPageも更新される
  - 下書き詳細画面にリダイレクトする
- 「下書き保存」ボタンの右側にドロップダウンアイコンがあり、クリックすると以下のアクションが表示される:
  - 「下書き保存して編集提案を作成する...」: 下書き保存後、保存した下書きページが選択された状態で編集提案作成画面に直接遷移する

下書き一覧画面 (`GET /drafts`):

- 参加しているスペースのトピックごとに下書き保存しているページが一覧表示される
- 各トピックグループに「編集提案する...」ボタンが表示される
- 「編集提案する...」を押すと、そのトピックにスコープされた編集提案作成画面に遷移する

編集提案作成画面:

- トピック内の自分の下書きページがチェックボックス付きで表示される
- 編集提案したい下書きページを選択し、タイトルと概要を入力し「作成する」ボタンを押すと編集提案が作成される

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

ページモデルの命名・設計に関する採用しなかった方針は [ページ編集画面のGo移行](page-edit-go-migration.md) の「採用しなかった方針」セクションを参照。

### 編集提案でのトピック変更

編集提案にトピック変更を含めることを検討したが、以下の理由から対象外とした。

- トピックごとにユーザーの権限が異なるため、トピックを横断する提案のレビュー権限が曖昧になる（移動元と移動先のどちらのトピック権限で判断すべきか）
- 「内容の変更」と「トピックの移動」が1つの提案に混在すると、レビュアーの判断が複雑になる
- 編集提案の主目的は「ページの内容改善を気軽に提案できること」であり、トピック移動は性質の異なる操作である
- トピック変更が必要な場合は、権限のあるユーザーが直接編集で対応すれば十分である

将来的にトピック変更対応を追加することは可能だが、初期段階では内容変更に集中し、シンプルな設計を維持する。

### DraftPageの変更を暗黙的に編集提案に反映する案

DraftPageがスペースメンバーごとに作成される設計を活かし、「DraftPageが編集されたら暗黙的にSuggestionPageRevisionを自動作成する」という案を検討した。DraftPageと編集提案の間に明示的なリンクを持たず、DraftPageの更新をトリガーとして編集提案のリビジョンを作成する方式。

**不採用の理由**:

- DraftPageの更新が通常の編集フロー用なのか編集提案フロー用なのかを区別できない。明示的なリンク（`suggestion_page_id`）がないため、「この下書きの変更はどこに反映されるのか」が不明確になる
- 代わりに、`draft_pages` テーブルに `suggestion_page_id`（nullable FK）を追加し、DraftPageが「どのブランチをチェックアウトしているか」を明示する方式を採用した（設計セクションの「DraftPageと編集提案の連携」を参照）

### suggestion_pages と draft_pages の中間テーブルを設ける案

`suggestion_pages` と `draft_pages` の間に多対多の中間テーブルを設け、スペースメンバーごとにレコードを作成する案を検討した。

**不採用の理由**:

- DraftPageから見ると、同時に複数の編集提案にリンクされるケースは実運用で起きづらい（下書きを保存したときに「どの編集提案に反映されるのか」が曖昧になるため）。実質的にDraftPage側は1対1の関係になる
- 1対1であれば、中間テーブルを設けるよりも `draft_pages` に nullable FK（`suggestion_page_id`）を追加する方がシンプル
- nullable FK方式でも、SuggestionPage側からは複数のDraftPage（異なるスペースメンバー）がリンクされる多対1の関係を表現できる

### DraftPageのユニーク制約を拡張する案

`draft_pages` のユニーク制約を `[space_member_id, page_id]` から `[space_member_id, page_id, suggestion_page_id]` に変更し、同一ページに対して通常編集と編集提案の編集を同時に持てるようにする案を検討した。

**不採用の理由**:

- 同一ページに対して複数のDraftPageが存在すると、ユーザーにとって「どの下書きがどの目的か」の管理が複雑になる
- Gitでも1つのワーキングツリーは同時に1つのブランチしかチェックアウトできない。同じ制約をWikinoにも適用する方が概念モデルがシンプルになる
- 編集提案の編集を始める際に確認画面を表示し、既存の下書きを保持するか編集提案の内容に切り替えるかをユーザーに選択してもらうことで、ユニーク制約を変更せずに対応できる

### `edit_suggestions` というテーブル名

当初テーブル名を `edit_suggestions` としていたが、`suggestions` にリネームした。

**リネームの理由**:

- `edit_suggestions` は正確だが冗長。関連テーブル名（`edit_suggestion_pages`, `edit_suggestion_page_revisions`）も長くなる
- Google Docsが同種の機能に「Suggestions」という用語を使っており、英語圏でも通じやすい
- Wikinoの文脈では「suggestions」= ページ編集の提案であることが明確なため、`edit_` プレフィックスがなくても曖昧にならない
- 日本語名は「編集提案」を維持する。「提案」だけでは一般名詞すぎて固有名詞感がなく、会話の中で機能名として認識しづらいため

### `suggested_pages` というテーブル名

`suggestion_pages` を `suggested_pages`（形容詞+名詞）とする案を検討した。「提案されたページ」という意味では英語として自然に読める。

**不採用の理由**:

- `suggestion_comments`（suggestionに属するcomments）は所有関係を表す複合名詞であり、`suggested_comments`（提案されたコメント）とするのは意味的に不自然。プレフィックスの文法パターンが混在する
- `suggestion_pages`、`suggestion_comments` はいずれも「suggestionに属するもの」という所有関係を表す複合名詞パターンで統一できる。`order_items`、`project_members` と同じ慣習
- GitHubのAPIでも `pull_request_reviews`、`pull_request_comments` のように名詞の複合形が使われている

### ページ編集画面のボタン名「公開する」「更新する」

ページ編集画面の送信ボタンの文言として「公開する」と「更新する」を検討したが、いずれも採用せず「トピックに公開」とした。

**「公開する」を不採用にした理由**:

- トピックの公開/非公開設定（visibility）と混同しやすい。「公開する」を押すとトピック自体が公開されるように見えてしまう
- 元々は下書きの「保存する」との差別化のために「公開する」としていたが、スコープが曖昧なため不採用とした

**「更新する」を不採用にした理由**:

- 新規ページ作成時（`/s/:space/topics/:topic/pages/new`）にページレコードが空の状態で作成され編集画面にリダイレクトされるが、まだ何も入力していない画面で「更新する」は違和感がある
- 「更新」は既存の内容を変更するニュアンスが強く、新規作成のケースに合わない

**「トピックに公開」を採用した理由**:

- 「公開」の対象をトピックに限定することで、トピック自体の公開/非公開との混同を回避できる
- 新規ページでも既存ページでも自然に読める（「トピックに公開する」という動作として一貫）
- 将来の編集提案機能で「編集提案を更新」ボタンに置き換わる際にも、「トピックに公開」と「編集提案を更新」で操作対象が明確に区別できる

### 「編集リクエスト」という名称

最初「編集リクエスト」という名前を検討したが、「編集提案」のほうが気軽・柔らかい印象があるため「編集提案」を採用した。

- 「編集リクエスト」のニュアンス
  - 「変更してください」というやや能動的なイメージ
  - 受け手にアクションを求めるイメージ
  - 編集を取り込むことを前提としたイメージ
- 「編集提案」のニュアンス
  - 「こうしたらどうだろうか？」という控えめなイメージ
  - 受け手に判断を委ねる受動的なイメージ
  - あくまでアイデアを提示したまでで、取り込まれなくても問題ないというイメージ

### 編集提案をページの亜種として扱う案

`pages` テーブルに `type` カラムを追加し、通常のページを `type: note`、編集提案を `type: suggestion` として管理する案を検討した。「すべてのドキュメントはページ」という思想に基づき、編集提案の本文でWikiリンク記法を自然に使えるようにする狙いがあった。

**不採用の理由**:

- ページモデルの責務が肥大化する。既存・今後のすべてのページ関連ロジック（検索、一覧、ページ番号、Wikiリンクの解決先など）でtypeの考慮が必要になる
- ページ一覧やトピック内ページ数など、既存のクエリすべてに `WHERE type = 'note'` が必要になり、漏れるとバグになる
- `[[編集提案のタイトル]]` でリンクできてしまうが、リンク先が編集提案の説明文ページになるのは意味的に不自然
- 編集提案の説明文は「変更に関するメタデータ」であり、Wikiのコンテンツではない。GitHubのPR descriptionが「ファイル」ではないのと同じ
- Wikiリンク記法のサポートはMarkdownレンダリングパイプラインの機能であり、ページモデルに依存しない。`suggestions.body` を保存時にページと同じMarkdownパイプライン（Wikiリンク解決含む）で処理すれば、ページモデルを変更せずに目的を達成できる

## 依存タスク

編集提案機能の実装前に、以下のタスクを完了する必要がある。

### 依存関係図

```
1. ページ編集画面のGo移行        2. ページの移動機能
   ↓                                (1と並行可能)
   ├─→ 3. 下書き機能のアップデート (DraftPageRevision + 下書き一覧画面)
   ├─→ 4. 差分表示コンポーネント（本タスクリスト内で対応）
   └─→ 6. トピック詳細画面のGo移行 (1と並行可能)
         ↓
   編集提案機能（本タスク）
      (1〜4, 6 が前提。5は後続タスクに変更)
```

### 前提タスク一覧

1. [ページ編集画面のGo移行](../3_done/202603/page-edit-go-migration.md) - Page/DraftPage/PageRevisionのGoへの移行
2. [ページの移動機能](../3_done/202603/page-move.md) - トピック変更の独立操作化（1と並行可能）
3. [下書き機能のアップデート](../3_done/202603/draft-update.md) - DraftPageRevision・下書き一覧画面（1が前提）
4. 差分表示コンポーネント - テキスト間の差分表示（1が前提）。編集提案の作業計画書内のタスクリストで対応する
5. ~~[タイトル変更時のWikiリンク自動書き換え](title-change-link-rewrite.md)~~ - 編集提案の後続タスクとして対応予定。編集提案の前提からは外す
6. [トピック詳細画面のGo移行](topic-show-go-migration.md) - トピック詳細画面のGo実装（1と並行可能）。編集提案でトピック画面にタブを追加するための前提

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

### フェーズ 0: フィーチャーフラグのセットアップ

- [x] **0-1**: [Go] フィーチャーフラグのセットアップ
  - `internal/model/feature_flag.go` に `FeatureFlagSuggestion FeatureFlagName = "go_suggestion"` を追加
  - `internal/middleware/reverse_proxy.go` の `featureFlaggedPatterns` に編集提案関連のURLパターンを追加
  - **想定ファイル数**: 約 2 ファイル（実装 2）
  - **想定行数**: 約 20 行（実装 20 行）

### フェーズ 1: データベース・モデル・リポジトリの基盤構築

#### フェーズ 1a: suggestionsテーブル

- [x] **1a-1**: [Go] suggestionsテーブルのマイグレーションとモデル定義
  - `go/db/migrations/` に `suggestions` テーブルのマイグレーション作成
  - カラム: id (ULID), space_id, topic_id, created_space_member_id, title, body, body_html, status (integer), applied_at, created_at, updated_at
  - インデックス: `[topic_id, status]`, `[space_id]`, `[created_space_member_id]`
  - `internal/model/id.go` に `SuggestionID` 型を追加
  - `internal/model/suggestion.go` に `Suggestion` モデルと `SuggestionStatus` 型（Draft=0, Open=1, Applied=2, Closed=3）を定義
  - **想定ファイル数**: 実装 3, テスト 0
  - **想定行数**: 実装 約80行

- [x] **1a-2**: [Go] suggestionsテーブルのsqlcクエリとリポジトリ
  - `internal/query/queries/suggestions.sql` にCRUDクエリを作成（Create, GetByID, ListByTopicAndStatus, UpdateStatus, Count系）
  - `internal/repository/suggestion.go` に `SuggestionRepository` を作成（WithTx, toModel, Create, FindByID, ListByTopicAndStatus, UpdateStatus）
  - `make sqlc-generate` でコード生成
  - **想定ファイル数**: 実装 2（queries 1 + repository 1）, テスト 1
  - **想定行数**: 実装 約150行, テスト 約150行

#### フェーズ 1b: suggestion_pagesテーブル

- [ ] **1b-1**: [Go] suggestion_pagesテーブルのマイグレーションとモデル定義
  - `go/db/migrations/` に `suggestion_pages` テーブルのマイグレーション作成
  - カラム: id (ULID), space_id, suggestion_id, page_id (nullable), page_revision_id (nullable), latest_revision_id (nullable), created_at, updated_at
  - ユニークインデックス: `[suggestion_id, page_id]`
  - インデックス: `[space_id]`
  - `internal/model/id.go` に `SuggestionPageID` 型を追加
  - `internal/model/suggestion_page.go` に `SuggestionPage` モデルを定義
  - **想定ファイル数**: 実装 3, テスト 0
  - **想定行数**: 実装 約70行

- [ ] **1b-2**: [Go] suggestion_pagesテーブルのsqlcクエリとリポジトリ
  - `internal/query/queries/suggestion_pages.sql` にCRUDクエリを作成
  - `internal/repository/suggestion_page.go` に `SuggestionPageRepository` を作成
  - **想定ファイル数**: 実装 2, テスト 1
  - **想定行数**: 実装 約130行, テスト 約130行

#### フェーズ 1c: suggestion_page_revisionsテーブル

- [ ] **1c-1**: [Go] suggestion_page_revisionsテーブルのマイグレーションとモデル定義
  - `go/db/migrations/` に `suggestion_page_revisions` テーブルのマイグレーション作成
  - カラム: id (ULID), space_id, suggestion_page_id, editor_space_member_id, title, body, body_html, created_at, updated_at
  - インデックス: `[suggestion_page_id, created_at]`, `[space_id]`
  - `internal/model/id.go` に `SuggestionPageRevisionID` 型を追加
  - `internal/model/suggestion_page_revision.go` に `SuggestionPageRevision` モデルを定義
  - **想定ファイル数**: 実装 3, テスト 0
  - **想定行数**: 実装 約60行

- [ ] **1c-2**: [Go] suggestion_page_revisionsテーブルのsqlcクエリとリポジトリ
  - `internal/query/queries/suggestion_page_revisions.sql` にCRUDクエリを作成
  - `internal/repository/suggestion_page_revision.go` に `SuggestionPageRevisionRepository` を作成
  - **想定ファイル数**: 実装 2, テスト 1
  - **想定行数**: 実装 約120行, テスト 約120行

#### フェーズ 1d: suggestion_commentsテーブル

- [ ] **1d-1**: [Go] suggestion_commentsテーブルのマイグレーションとモデル定義
  - `go/db/migrations/` に `suggestion_comments` テーブルのマイグレーション作成
  - カラム: id (ULID), space_id, suggestion_id, created_space_member_id, body, body_html, created_at, updated_at
  - インデックス: `[suggestion_id, created_at]`, `[space_id]`
  - `internal/model/id.go` に `SuggestionCommentID` 型を追加
  - `internal/model/suggestion_comment.go` に `SuggestionComment` モデルを定義
  - **想定ファイル数**: 実装 3, テスト 0
  - **想定行数**: 実装 約50行

- [ ] **1d-2**: [Go] suggestion_commentsテーブルのsqlcクエリとリポジトリ
  - `internal/query/queries/suggestion_comments.sql` にCRUDクエリを作成
  - `internal/repository/suggestion_comment.go` に `SuggestionCommentRepository` を作成
  - **想定ファイル数**: 実装 2, テスト 1
  - **想定行数**: 実装 約100行, テスト 約100行

#### フェーズ 1e: draft_pagesテーブルの変更

- [ ] **1e-1**: [Go] draft_pagesテーブルへのsuggestion_page_idカラム追加
  - `go/db/migrations/` に `draft_pages` テーブルへの `suggestion_page_id` (nullable FK → `suggestion_pages`) カラム追加マイグレーション作成
  - `internal/model/draft_page.go` に `SuggestionPageID *SuggestionPageID` フィールドを追加
  - sqlcクエリの更新（suggestion_page_idを含むselect/update）
  - `internal/repository/draft_page.go` の `toModel` を更新
  - **想定ファイル数**: 実装 4, テスト 1
  - **想定行数**: 実装 約60行, テスト 約50行

### フェーズ 2: 編集提案一覧画面

- [ ] **2-1**: [Go] 編集提案一覧のUseCase・ViewModel
  - `internal/usecase/get_suggestion_list.go` に `GetSuggestionListUsecase` を作成（トピック内の編集提案一覧取得、オープン/クローズのフィルタリング）
  - `internal/viewmodel/suggestion.go` に `SuggestionForList` ViewModel を作成（タイトル、ステータス、作成者名、作成日時）
  - **想定ファイル数**: 実装 2, テスト 1
  - **想定行数**: 実装 約120行, テスト 約100行

- [ ] **2-2**: [Go] 編集提案一覧のハンドラーとテンプレート
  - `internal/handler/suggestion/handler.go` に Handler 構造体を定義
  - `internal/handler/suggestion/index.go` に `Index` メソッドを実装（GET /s/{space}/topics/{topic}/suggestions）
  - `internal/templates/pages/suggestion/index.templ` に一覧テンプレートを作成（オープン/クローズのタブ切り替え）
  - `cmd/server/main.go` にルーティング登録
  - 翻訳ファイル（ja.toml, en.toml）にメッセージ追加
  - **想定ファイル数**: 実装 5, テスト 1
  - **想定行数**: 実装 約200行, テスト 約100行

- [ ] **2-3**: [Go] トピック詳細画面に「編集提案」タブを追加
  - `internal/templates/pages/topic/show.templ` に「ページ」「編集提案」のタブUIを追加
  - `internal/templates/components/` にタブコンポーネントを作成（必要に応じて）
  - タブクリック時に編集提案一覧画面（`/s/{space}/topics/{topic}/suggestions`）にナビゲーション
  - 翻訳ファイルにタブラベルのメッセージ追加
  - **想定ファイル数**: 実装 3, テスト 1
  - **想定行数**: 実装 約80行, テスト 約50行

### フェーズ 3: 編集提案作成

- [ ] **3-1**: [Go] 編集提案作成のValidator・UseCase
  - `internal/validator/suggestion.go` に `SuggestionCreateValidator` を作成（タイトル必須、長さ制限、選択された下書きページの存在確認）
  - `internal/usecase/create_suggestion.go` に `CreateSuggestionUsecase` を作成（トランザクション: Suggestion作成 → 選択された下書きページからSuggestionPage・SuggestionPageRevision作成）
  - **想定ファイル数**: 実装 2, テスト 2
  - **想定行数**: 実装 約200行, テスト 約250行

- [ ] **3-2**: [Go] 編集提案作成のハンドラーとテンプレート
  - `internal/handler/suggestion/new.go` に `New` メソッドを実装（GET /s/{space}/topics/{topic}/suggestions/new）
  - `internal/handler/suggestion/create.go` に `Create` メソッドを実装（POST /s/{space}/topics/{topic}/suggestions）
  - `internal/usecase/get_suggestion_new.go` に作成画面用データ取得UseCase（トピック内の自分の下書きページ一覧）
  - `internal/templates/pages/suggestion/new.templ` に作成フォームテンプレート（下書きページのチェックボックス、タイトル・概要入力）
  - 翻訳ファイルにメッセージ追加
  - **想定ファイル数**: 実装 5, テスト 2
  - **想定行数**: 実装 約250行, テスト 約200行

### フェーズ 4: 編集提案詳細画面（会話タブ）

- [ ] **4-1**: [Go] 編集提案詳細のUseCase・ViewModel
  - `internal/usecase/get_suggestion_detail.go` に `GetSuggestionDetailUsecase` を作成（編集提案 + コメント一覧 + 編集ページ一覧取得）
  - `internal/viewmodel/suggestion.go` に `SuggestionForDetail`, `SuggestionCommentForList` ViewModel を追加
  - **想定ファイル数**: 実装 2, テスト 1
  - **想定行数**: 実装 約150行, テスト 約120行

- [ ] **4-2**: [Go] 編集提案詳細のハンドラーとテンプレート
  - `internal/handler/suggestion/show.go` に `Show` メソッドを実装（GET /s/{space}/topics/{topic}/suggestions/{suggestion_id}）
  - `internal/templates/pages/suggestion/show.templ` に詳細テンプレート（「会話」「編集したページ」のタブ、ステータスバッジ、概要表示）
  - `cmd/server/main.go` にルーティング登録
  - 翻訳ファイルにメッセージ追加
  - **想定ファイル数**: 実装 4, テスト 1
  - **想定行数**: 実装 約200行, テスト 約100行

### フェーズ 5: コメント機能

- [ ] **5-1**: [Go] コメント作成のValidator・UseCase・ハンドラー
  - `internal/validator/suggestion_comment.go` に `SuggestionCommentCreateValidator` を作成（本文必須、長さ制限）
  - `internal/usecase/create_suggestion_comment.go` に `CreateSuggestionCommentUsecase` を作成
  - `internal/handler/suggestion_comment/handler.go` に Handler 構造体を定義
  - `internal/handler/suggestion_comment/create.go` に `Create` メソッドを実装（POST /s/{space}/topics/{topic}/suggestions/{suggestion_id}/comments）
  - `internal/templates/pages/suggestion/` にコメントフォーム・コメント一覧の部分テンプレートを追加
  - `cmd/server/main.go` にルーティング登録
  - 翻訳ファイルにメッセージ追加
  - **想定ファイル数**: 実装 6, テスト 2
  - **想定行数**: 実装 約250行, テスト 約200行

### フェーズ 6: 差分表示（「編集したページ」タブ）

- [ ] **6-1**: [Go] 差分表示コンポーネントの実装
  - テキストの差分計算ライブラリの導入（`github.com/sergi/go-diff` 等）
  - `internal/viewmodel/diff.go` に差分表示用ViewModel（DiffLine, DiffBlock等）を定義
  - `internal/templates/components/diff.templ` に差分表示コンポーネントを作成（追加行・削除行・変更行のスタイリング）
  - **想定ファイル数**: 実装 3, テスト 1
  - **想定行数**: 実装 約200行, テスト 約100行

- [ ] **6-2**: [Go] 「編集したページ」タブの実装
  - `internal/usecase/get_suggestion_diff.go` に差分取得UseCase（各SuggestionPageの最新リビジョンとベースページの差分を計算）
  - `internal/templates/pages/suggestion/diff.templ` に差分表示テンプレートを作成
  - `internal/handler/suggestion/show.go` を更新し、タブに応じた表示を切り替え
  - **想定ファイル数**: 実装 3, テスト 1
  - **想定行数**: 実装 約180行, テスト 約120行

### フェーズ 7: 編集提案の反映（マージ）

- [ ] **7-1**: [Go] 編集提案反映のUseCase
  - `internal/usecase/apply_suggestion.go` に `ApplySuggestionUsecase` を作成
  - トランザクション内で: 各SuggestionPageの最新リビジョンの内容でPageを更新 → PageRevision作成 → Suggestionのステータスを「反映済み」に変更 → applied_atを設定
  - 新規ページ作成の場合はPage作成処理も含む
  - **想定ファイル数**: 実装 1, テスト 1
  - **想定行数**: 実装 約200行, テスト 約250行

- [ ] **7-2**: [Go] 編集提案反映のハンドラーとPolicy
  - `internal/policy/suggestion.go` に `SuggestionPolicy` を作成（反映権限: スペースオーナーまたはトピック管理者）
  - `internal/handler/suggestion/update.go` に `Update` メソッドを実装（PATCH /s/{space}/topics/{topic}/suggestions/{suggestion_id}）
  - 反映ボタンと確認UIをテンプレートに追加
  - `cmd/server/main.go` にルーティング登録
  - 翻訳ファイルにメッセージ追加
  - **想定ファイル数**: 実装 4, テスト 2
  - **想定行数**: 実装 約180行, テスト 約200行

### フェーズ 8: 編集提案のクローズ

- [ ] **8-1**: [Go] 編集提案クローズのUseCase・ハンドラー
  - `internal/usecase/close_suggestion.go` に `CloseSuggestionUsecase` を作成（ステータスを「クローズ」に変更）
  - `internal/handler/suggestion/` にクローズ用ハンドラーを追加（`update.go` 内でアクション分岐、またはDELETEを活用）
  - クローズボタンをテンプレートに追加（権限: 作成者またはスペースオーナー/トピック管理者）
  - `internal/policy/suggestion.go` にクローズ権限判定を追加
  - 翻訳ファイルにメッセージ追加
  - **想定ファイル数**: 実装 3, テスト 2
  - **想定行数**: 実装 約120行, テスト 約150行

### フェーズ 9: 編集提案ページの編集

- [ ] **9-1**: [Go] 編集提案ページ編集開始のUseCase・ハンドラー
  - 編集提案詳細画面から「編集する」ボタンで編集開始
  - `internal/usecase/start_suggestion_page_edit.go` に UseCase を作成（DraftPageの `suggestion_page_id` を設定し、SuggestionPageRevisionの最新内容でDraftPageを初期化）
  - 既存DraftPageがある場合の確認画面テンプレートを作成
  - 通常編集 → 編集提案編集の切り替えフロー実装
  - **想定ファイル数**: 実装 4, テスト 2
  - **想定行数**: 実装 約200行, テスト 約200行

- [ ] **9-2**: [Go] ページ編集画面の編集提案モード対応
  - ページ編集画面（`internal/handler/page/edit.go`）で `DraftPage.SuggestionPageID` がNOT NULLの場合の表示切り替え
  - 「トピックに公開」ボタンを非表示 → 「編集提案を更新」ボタンを表示
  - 「この下書きへの保存は編集提案 #xxx に反映されます」メッセージ表示
  - `internal/templates/pages/page/edit.templ` の更新
  - **想定ファイル数**: 実装 3, テスト 1
  - **想定行数**: 実装 約100行, テスト 約80行

- [ ] **9-3**: [Go] 「編集提案を更新」アクションの実装
  - `internal/usecase/update_suggestion_page.go` に UseCase を作成（DraftPageの内容からSuggestionPageRevision作成 → SuggestionPageのlatest_revision_id更新 → DraftPageのsuggestion_page_idをクリア）
  - 既存のページ更新ハンドラーに編集提案更新のルートを追加、またはsuggestion_page用の新ハンドラーを作成
  - **想定ファイル数**: 実装 3, テスト 2
  - **想定行数**: 実装 約180行, テスト 約200行

### フェーズ 10: 下書き保存からの編集提案作成フロー

- [ ] **10-1**: [Go] ページ編集画面の「下書き保存して編集提案を作成する...」アクション
  - ページ編集画面の「下書き保存」ボタン右側のドロップダウンに「下書き保存して編集提案を作成する...」アクションを追加
  - `internal/templates/pages/page/edit.templ` にドロップダウンUIを追加
  - アクション実行時: 下書き保存後、保存した下書きページが選択された状態で編集提案作成画面にリダイレクト
  - **想定ファイル数**: 実装 2, テスト 1
  - **想定行数**: 実装 約80行, テスト 約60行

- [ ] **10-2**: [Go] 下書き一覧画面の「編集提案する...」ボタン
  - 下書き一覧画面の各トピックグループに「編集提案する...」ボタンを追加
  - クリック時にそのトピックにスコープされた編集提案作成画面に遷移
  - `internal/templates/pages/draft_page/` の既存テンプレートを更新
  - **想定ファイル数**: 実装 2, テスト 1
  - **想定行数**: 実装 約60行, テスト 約40行

### フェーズ N: フィーチャーフラグの削除

<!--
機能が安定し全ユーザーに公開した後に実施する。
-->

- [ ] **N-0**: [Go] フィーチャーフラグの削除
  - `featureFlaggedPatterns` から編集提案のパターンを削除し、`goHandledPrefixPaths` または `goHandledRegexPatterns` に移動
  - `FeatureFlagSuggestion` 定数を削除
  - DBから `go_suggestion` フラグレコードを削除するマイグレーション（必要に応じて）

### フェーズ N+1: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [ ] **N-1**: 仕様書の作成・更新
  - `docs/specs/suggestion/overview.md` に仕様書を作成する
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する
