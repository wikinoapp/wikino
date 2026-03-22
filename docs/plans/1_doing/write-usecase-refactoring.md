# 書き込みUseCaseのリファクタリング 作業計画書

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

- [アーキテクチャガイド](../../go/docs/architecture-guide.md)（タスク完了後に必要があれば更新）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

書き込みUseCaseのアーキテクチャ方針（「Handler での処理フロー」ガイドライン）に沿っていない既存実装をリファクタリングする。

現在、複数の書き込みUseCaseが Execute メソッド内でデータの取得（`FindByID`, `FindByXxx` 等）や状態の検証（ステータスチェック、存在確認等）を行っている。これらはHandlerが読み取りUseCaseやValidatorを通じて事前に行い、書き込みUseCaseには取得・検証済みのデータのみを渡すべきである。

**目的**:

- 書き込みUseCaseをトランザクション内の永続化処理に専念させ、責務を明確にする
- 検証ロジックをValidatorに集約し、一覧性を高める
- トランザクションの保持時間を短縮する

## 要件

<!--
ガイドライン:
- 機能要件: 「何ができるべきか」を記述
- 非機能要件: 「どのように動くべきか」を必要に応じて記述
-->

### 機能要件

- 各書き込みUseCaseの Execute メソッドからデータ取得（Find/Get/List呼び出し）を削除する
- 状態の検証はValidatorまたはHandler内で行い、書き込みUseCaseの入力には検証済みのモデルを渡す
- 外部から見た挙動（HTTPレスポンス、永続化結果）は変わらない

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
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（特に「Handler での処理フロー」セクション）
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン（**ファイル名は標準の9種類のみ**）
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

### リファクタリング方針

各書き込みUseCaseに対して以下のパターンでリファクタリングする：

1. **UseCase内のFind/Get/List呼び出しを特定する**
2. **呼び出し側（Handler）で事前にデータを取得する** — 既存の読み取りUseCaseで取得済みなら再利用、なければValidatorで取得
3. **UseCase の Input に取得済みモデルを追加し、Find呼び出しを削除する**
4. **UseCase内の状態検証をValidatorに移動する**

### 対象UseCaseと違反内容

#### 1. `close_suggestion.go`（軽微）

**違反内容**: トランザクション内で `suggestionRepo.FindByID` を呼び出し、ステータスを検証している。

**変更方針**: Handler（`suggestion_close/create.go`）は既に `getSuggestionDetailUsecase` でSuggestionを取得済み。Input に `*model.Suggestion` を追加し、UseCase内のFindByIDとステータス検証を削除する。

#### 2. `apply_suggestion.go`（中規模）

**違反内容**: トランザクション内で以下を行っている：

- `suggestionRepo.FindByID` でSuggestionを取得
- `suggestion.Status != Open` のステータス検証
- `suggestionPageRepo.ListBySuggestionID` でSuggestionPagesを取得
- `pageRepo.FindByIDs` でPagesを取得

**変更方針**: Handler（`suggestion_apply/create.go`）は既に `getSuggestionDetailUsecase` でSuggestion/SuggestionPagesを取得済み。Pagesの取得はValidatorまたは読み取りUseCaseに移動する。Input に取得済みモデルを追加する。

#### 3. `create_account.go`（軽微）

**違反内容**: トランザクション内で `emailConfirmationRepo.FindByID` を呼び出し、`IsSucceeded()` を検証している。

**変更方針**: Handler（`account/create.go`）の既存Validator（`AccountCreateValidator`）にメール確認の検証を追加し、UseCaseからは削除する。

#### 4. `create_suggestion.go`（中規模）

**違反内容**: トランザクション内で以下を行っている：

- `resolveLinkedPages` ヘルパー内で `topicRepo.FindBySpaceAndNames` と `pageRepo.FindByTopicAndTitle` を呼び出し
- `pageRevisionRepo.FindLatestByPageID` で各ページの最新リビジョンを取得

**変更方針**: Wikiリンク解決は読み取りUseCase（`GetSuggestionBodyHTMLUsecase`）で実行し、レンダリング済みの `BodyHTML` を書き込みUseCaseの Input に渡す。ページリビジョンの取得も読み取りUseCase（`GetLatestPageRevisionsUsecase`）で事前に行い、結果を Input に渡す。`resolveLinkedPages` は `get_suggestion_body_html.go` に移動。

#### 5. `publish_page.go`（中規模）

**違反内容**: トランザクション内で以下を行っている：

- `resolveAndCreateLinkedPages` ヘルパー内でトピック・ページの検索
- `syncAttachmentReferences` 内で既存参照の取得
- `extractFeaturedImageAttachmentID` 内で添付ファイルの存在確認
- `markup.FilterAttachments` 内で添付ファイルの検索

**変更方針**: `resolveAndCreateLinkedPages` はリンク先ページの自動作成（Write）を含むためトランザクション内に残す。添付ファイル関連の処理（`syncAttachmentReferences` の読み取り部分、`extractFeaturedImageAttachmentID`、`markup.FilterAttachments`）とMarkdownレンダリング・画像ラッピングは読み取りUseCase（`GetPagePublishDataUsecase`）で事前に実行し、結果を `PublishPageInput` に渡す。`syncAttachmentReferences` は `calculateAttachmentRefDiff`（読み取り）と `applyAttachmentRefChanges`（書き込み）に分離する。

#### 6. `auto_save_draft_page.go` / `manual_save_draft_page.go`（大規模）

**違反内容**: 共通ヘルパー `saveDraftPageContent` 内で以下を行っている：

- `findOrCreateDraftPage` でのDraftPage取得/作成
- `resolveAndCreateLinkedPages` でのトピック・ページ検索
- `markup.FilterAttachments` での添付ファイル検索
- `extractFeaturedImageAttachmentID` での添付ファイル検証

**変更方針**: `publish_page.go` と同様のアプローチ。ただし `auto_save_draft_page` はエディタの自動保存で頻繁に呼ばれるため、パフォーマンスへの影響を考慮する。

**注意**: 自動保存は `PATCH /s/{space}/pages/{page_number}/draft_page` で呼ばれ、レスポンス速度が重要。リファクタリングにより呼び出し回数が増えないよう注意が必要。

### 優先順位

| 優先度 | UseCase                                                 | 理由                                                 |
| ------ | ------------------------------------------------------- | ---------------------------------------------------- |
| 高     | `close_suggestion.go`                                   | 変更が小さく、パターンの実証に適する                 |
| 高     | `apply_suggestion.go`                                   | 編集提案機能の一部であり、早期に整合性を保ちたい     |
| 中     | `create_account.go`                                     | 変更が小さい                                         |
| 中     | `create_suggestion.go`                                  | 編集提案機能の一部                                   |
| 低     | `publish_page.go`                                       | リンク先ページの自動作成がありリファクタリングが複雑 |
| 低     | `auto_save_draft_page.go` / `manual_save_draft_page.go` | 共通ヘルパーの大規模な分離が必要                     |

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### `publish_page.go` の `resolveAndCreateLinkedPages` を完全にUseCaseから分離する案

リンク解決（Read）とリンク先ページ自動作成（Write）が `resolveAndCreateLinkedPages` に混在しているため、完全に分離するにはこの関数の分割が必要。しかし、リンク先ページの自動作成はトランザクション内で行う必要があり、解決結果に依存する。

リンク解決を事前に行い、結果をUseCaseに渡し、UseCase内では「未解決のリンク先ページの自動作成」のみを行う設計も可能だが、リンク解決時点で存在しなかったページがトランザクション開始までに他のユーザーに作成される可能性がある。実用上は問題にならないが、設計が複雑になるため、段階的に対応する。

### `auto_save_draft_page.go` / `manual_save_draft_page.go` を同時にリファクタリングする案

両者は `saveDraftPageContent` ヘルパーを共有しているため、同時にリファクタリングするのが理想。しかし変更規模が大きくなるため、フェーズを分けて対応する。

### `create_password_reset_token.go` をリファクタリング対象にする案

このUseCaseはトランザクション前にユーザーの存在確認（`FindByEmail`）を行っているが、これはセキュリティ上の設計判断（ユーザーの存在を外部に漏らさないため、存在しない場合は何もせず成功を返す）であり、Handlerに移すとセキュリティリスクが生じる可能性がある。そのため対象外とする。

### `start_suggestion_page_edit.go` をリファクタリング対象にする案

このUseCaseはトランザクション前にデータ取得と条件分岐を行っているが、これは「トランザクションが不要なケース（既にリンク済み、コンフリクト）」を早期リターンするための設計であり、合理的である。トランザクションの保持時間を最小化するという方針にも合致しているため、対象外とする。

### 書き込みUseCaseからすべてのデータ取得を外出しし、読み取りUseCaseで行う案

フェーズ1〜4では「書き込みUseCaseはデータの取得・検証を行わない。必要なデータはすべて入力として受け取る」という方針で、書き込みUseCaseのためのデータ取得を読み取りUseCaseやValidatorに移動した。しかし、書き込みUseCaseのためだけに読み取りUseCaseを新設すると、処理が分散して見通しが悪くなることが判明した。

**不採用の理由**:

- Handlerが書き込みUseCaseの内部実装を知る必要が生じる（どんなデータを事前に用意すべきか）
- 書き込みUseCaseのために読み取りUseCaseを作ると、両者が強く結合し、分離のメリットが薄い
- `GetDraftPageSaveDataUsecase` と `GetSaveDraftPageDataUsecase` のように命名が酷似し混同しやすくなる

**代替として採用した方針**: 書き込みUseCase内であっても、トランザクション開始前であればデータ取得を行ってよい。ただし以下のルールを守る: (1) 検証処理を書かない（エラー返り値はサーバーエラーのみ）、(2) トランザクション内は永続化のみ、(3) Execute内にロジックを直接書かず関数に切り出す。

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

### フェーズ 1: 編集提案関連のUseCaseリファクタリング

#### `close_suggestion.go`

- [x] **1-1**: [Go] `close_suggestion.go` のリファクタリング
  - `CloseSuggestionInput` に `Suggestion *model.Suggestion` を追加
  - UseCase内の `FindByID` とステータス検証を削除（Handlerが既にgetSuggestionDetailで取得・検証済み）
  - Handler（`suggestion_close/create.go`）の呼び出しを更新
  - 関連テストの更新
  - **想定ファイル数**: 約 4 ファイル（実装 2 + テスト 2）
  - **想定行数**: 約 60 行（実装 30 行 + テスト 30 行）

#### `apply_suggestion.go`

- [x] **1-2**: [Go] `apply_suggestion.go` のリファクタリング
  - `ApplySuggestionInput` に `Suggestion`, `SuggestionPages`, `Pages` を追加
  - UseCase内の `FindByID`, `ListBySuggestionID`, `FindByIDs` とステータス検証を削除
  - Pagesの取得をValidatorまたは読み取りUseCaseで事前に行う
  - Handler（`suggestion_apply/create.go`）の呼び出しを更新
  - 関連テストの更新
  - **想定ファイル数**: 約 6 ファイル（実装 3 + テスト 3）
  - **想定行数**: 約 150 行（実装 80 行 + テスト 70 行）

### フェーズ 2: アカウント・認証関連のUseCaseリファクタリング

- [x] **2-1**: [Go] `create_account.go` のリファクタリング
  - 既存の `AccountCreateValidator` にメール確認の検証（`FindByID` + `IsSucceeded`）を追加
  - `CreateAccountInput` に `EmailConfirmation *model.EmailConfirmation` を追加
  - UseCase内の `FindByID` と `IsSucceeded` 検証を削除
  - Handler（`account/create.go`）の呼び出しを更新
  - 関連テストの更新
  - **想定ファイル数**: 約 6 ファイル（実装 3 + テスト 3）
  - **想定行数**: 約 100 行（実装 50 行 + テスト 50 行）

### フェーズ 3: 編集提案作成のUseCaseリファクタリング

- [x] **3-1**: [Go] `create_suggestion.go` のリファクタリング
  - Wikiリンク解決（`resolveLinkedPages`）をUseCase外で実行するように変更
  - `resolveLinkedPages` をHandler側で呼び出し、結果を Input に渡す
  - ページリビジョン取得（`FindLatestByPageID`）を読み取りUseCase（`GetLatestPageRevisionsUsecase`）で事前に行い、結果を Input に含める
  - `CreateSuggestionInput` に `BodyHTML`, `PageLocations`, `PageRevisions` を追加
  - 関連テストの更新
  - **想定ファイル数**: 約 6 ファイル（実装 3 + テスト 3）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行）

### フェーズ 4: ページ公開・下書き保存のUseCaseリファクタリング

- [x] **4-1**: [Go] `publish_page.go` のリファクタリング
  - 読み取りUseCase `GetPagePublishDataUsecase` を新規作成し、Markdownレンダリング・添付ファイル参照の差分計算・アイキャッチ画像抽出・添付ファイルフィルター・画像ラッピングをトランザクション外に移動
  - `syncAttachmentReferences` を `calculateAttachmentRefDiff`（読み取り）と `applyAttachmentRefChanges`（書き込み）に分離
  - `PublishPageInput` に `BodyHTML`, `FeaturedImageAttachmentID`, `AttachmentRefsToAdd`, `AttachmentRefsToRemove` を追加
  - `PublishPageUsecase` から `attachmentRepo` を削除
  - `resolveAndCreateLinkedPages` はリンク先ページの自動作成を含むためトランザクション内に残す
  - Handler（`page/update.go`）で `GetPagePublishDataUsecase` を呼び出してから `PublishPageUsecase` に渡すように変更
  - 複数UseCaseで共通利用されるWikiリンク関連の関数（`resolveAndCreateLinkedPages`, `findOrCreateLinkedPage`, `uniqueTopicNames`, `isUniqueViolation`, `findOrCreateRetryLimit`）を `auto_save_draft_page.go` から `linked_page.go` に切り出し
  - 関連テストの更新
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 2 + 既存テスト更新 2）
  - **想定行数**: 約 250 行（実装 130 行 + テスト 120 行）

- [x] **4-2**: [Go] `auto_save_draft_page.go` / `manual_save_draft_page.go` のリファクタリング
  - `resolveAndCreateLinkedPages` から純粋な計算とDB読み取りをトランザクション前に分離
    - `markup.ScanWikilinks`、`uniqueTopicNames`（純粋な計算）をトランザクション前に移動
    - `topicRepo.FindBySpaceAndNames`（DB読み取り）をトランザクション前に移動
    - `findOrCreateLinkedPage`、`pageEditorRepo.FindOrCreate`（find-or-create）はトランザクション内に残す
    - 事前計算の結果（`WikilinkKey`のリスト、トピックのMap）を `resolveAndCreateLinkedPages` に引数として渡す
  - `saveDraftPageContent` からトランザクション外に出せる処理を分離
    - Markdownレンダリング（`markup.RenderMarkdown`）をトランザクション前に移動
    - 添付ファイルフィルター（`markup.FilterAttachments`）をトランザクション前に移動
    - 画像ラッピング（`markup.WrapStandaloneImageLinks`）をトランザクション前に移動
    - アイキャッチ画像抽出（`extractFeaturedImageAttachmentID`）をトランザクション前に移動
  - `auto_save_draft_page.go` と `manual_save_draft_page.go` の Input を更新
  - 自動保存のパフォーマンスに影響がないことを確認
  - 関連テストの更新
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 300 行（実装 150 行 + テスト 150 行）

### フェーズ 5: 書き込みUseCaseへの読み取りUseCase統合

フェーズ3・4で書き込みUseCaseのデータ取得を外出しするために新設した読み取りUseCaseを廃止し、データ取得ロジックを書き込みUseCaseのトランザクション前に統合する。

**背景**: 書き込みUseCaseのためだけに読み取りUseCaseを定義すると、処理が分散して見通しが悪くなる。書き込みUseCase内であっても、トランザクション開始前であればデータ取得を行ってよいという方針に変更する（詳細はフェーズ5最後のアーキテクチャガイド更新タスクを参照）。

**注意**: フェーズ1・2でリファクタリング済みの `close_suggestion.go`、`apply_suggestion.go`、`create_account.go` は変更不要。これらはHandlerが既存の読み取りUseCaseやValidatorで自然に取得済みのデータを書き込みUseCaseに渡しており、書き込みUseCaseのために新しい読み取りUseCaseを作成していない。

#### `CreateSuggestionUsecase`

- [x] **5-1**: [Go] `GetSuggestionBodyHTMLUsecase` と `GetLatestPageRevisionsUsecase` を `CreateSuggestionUsecase` に統合
  - `CreateSuggestionUsecase` に `topicRepo`, `pageRepo`, `pageRevisionRepo` を追加
  - `GetSuggestionBodyHTMLUsecase` のWikiリンク解決・Markdownレンダリングロジックを `CreateSuggestionUsecase` のトランザクション前に移動
  - `GetLatestPageRevisionsUsecase` のページリビジョン取得を `CreateSuggestionUsecase` のトランザクション前に移動
  - `CreateSuggestionInput` から `BodyHTML`, `PageRevisions` を削除し、必要なパラメータ（`Body`, `CurrentTopicName`, `SpaceIdentifier`）を追加
  - `get_suggestion_body_html.go`, `get_latest_page_revisions.go` を削除
  - Handler（`suggestion/create.go`）から2つの読み取りUseCaseの呼び出しを削除
  - Handler（`suggestion/handler.go`）から依存を削除
  - `cmd/server/main.go` のUseCase構築と依存注入を更新
  - 関連テストの更新
  - **想定ファイル数**: 約 10 ファイル（実装 5 + テスト 3 + 削除 2）
  - **想定行数**: 約 250 行（実装 130 行 + テスト 120 行）

#### `PublishPageUsecase`

- [ ] **5-2**: [Go] `GetPagePublishDataUsecase` を `PublishPageUsecase` に統合
  - `PublishPageUsecase` に `attachmentRepo` を追加
  - `GetPagePublishDataUsecase` のMarkdownレンダリング・添付ファイル参照差分計算・アイキャッチ画像抽出・添付ファイルフィルター・画像ラッピングを `PublishPageUsecase` のトランザクション前に移動
  - `PublishPageInput` から `BodyHTML`, `FeaturedImageAttachmentID`, `AttachmentRefsToAdd`, `AttachmentRefsToRemove` を削除し、必要なパラメータ（`Body` 等）を追加
  - `get_page_publish_data.go` を削除
  - Handler（`page/update.go`）から `GetPagePublishDataUsecase` の呼び出しを削除
  - Handler（`page/handler.go`）から依存を削除
  - `cmd/server/main.go` のUseCase構築と依存注入を更新
  - 関連テストの更新
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 3 + 削除 1）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行）

#### `AutoSaveDraftPageUsecase` / `ManualSaveDraftPageUsecase`

- [ ] **5-3**: [Go] `GetDraftPageSaveDataUsecase` を `AutoSaveDraftPageUsecase` / `ManualSaveDraftPageUsecase` に統合
  - Markdownレンダリング・Wikiリンクスキャン・トピック検索・アイキャッチ画像抽出・添付ファイルフィルター・画像ラッピングを共通ヘルパーまたは各UseCaseのトランザクション前に移動
  - `AutoSaveDraftPageUsecase`, `ManualSaveDraftPageUsecase` に `topicRepo`, `attachmentRepo` を追加
  - `AutoSaveDraftPageInput`, `ManualSaveDraftPageInput` から `BodyHTML`, `FeaturedImageAttachmentID`, `WikilinkKeys`, `TopicMap` を削除し、必要なパラメータ（`Body` 等）を追加
  - `get_draft_page_save_data.go` を削除
  - Handler（`draft_page/update.go`, `draft_page_revision/update.go`）から `GetDraftPageSaveDataUsecase` の呼び出しを削除
  - Handler（`draft_page/handler.go`, `draft_page_revision/handler.go`）から依存を削除
  - `cmd/server/main.go` のUseCase構築と依存注入を更新
  - 関連テストの更新
  - **想定ファイル数**: 約 10 ファイル（実装 5 + テスト 4 + 削除 1）
  - **想定行数**: 約 250 行（実装 130 行 + テスト 120 行）

#### アーキテクチャガイドの更新

- [ ] **5-4**: [Go] アーキテクチャガイドの「Handler での処理フロー」セクションを更新
  - 書き込みUseCaseのルールを以下の3つに整理:
    1. データの検証処理（ユーザーに表示するエラーを判別するもの）を書かない。書き込みUseCaseのエラー返り値はサーバーエラーとする
    2. トランザクション開始後はデータの取得や計算処理を行わない。永続化処理のみ行う（トランザクション前のデータ取得は許可）
    3. Execute内にロジックを直接書かない。ロジックは関数やメソッドとして定義し、Execute内ではそれを呼び出すだけにする
  - usecaseパッケージ内のプライベート関数の配置ルールを追加: あるUseCaseファイルに定義されたプライベート関数を別のUseCaseファイルから呼び出す必要が生じた場合は、その関数を専用のファイルに切り出す（例: `linked_page.go`）
  - 「書き込みUseCaseのために読み取りUseCaseを新設する方針」を「採用しなかった方針」に追記
  - **想定ファイル数**: 1 ファイル
  - **想定行数**: 約 60 行
