# 編集提案 仕様書

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

編集提案機能は、GitHub の Pull Request のように、ページに対して変更案を提出し、レビューを経て反映できる機能である。直接編集する代わりに「こう変更してはどうでしょうか」という形で提案できる。トピック内のページの変更（既存ページの編集、新規ページの追加）を 1 つの編集提案にまとめ、コメントによるレビューを経て反映・クローズできる。

編集提案は GitHub の Pull Request モデルを参考に設計しており、`Page` を「リモート main ブランチ」、`DraftPage` を「ローカルのワーキングツリー」、`SuggestionPage` を「リモートのフィーチャーブランチ」、`SuggestionPageRevision` を「ブランチ上のコミット」、`Suggestion` を「Pull Request」に対応させている。これにより、複数のスペースメンバーが同じ編集提案ページに変更を加えることが可能になっている。

Go 版で実装されており、現在は `go_suggestion` フィーチャーフラグで制御されている。フラグが有効なユーザー/デバイスのみがアクセスできる。Rails 版に対応する機能はなく、フラグ無効時は Rails 版にプロキシされて 404 となる。

**目的**:

- 編集に対する心理的ハードルを下げ、気軽に貢献できるようにする
- AI による文書更新を積極的に活用し、人間が変更箇所を確認して取捨選択できるようにする
- レビュープロセスにより、ドキュメントの品質を維持・向上する
- 非技術者でも Git 的なワークフローを利用できるようにする

**背景**:

- 「たぶんこの記述は間違っているけど、自分の修正も合ってるか自信が持てない」という場面で、直接編集は心理的ハードルが高い。「提案」という形であれば、却下されても精神的ダメージが少なく、気軽に貢献できる
- 企業の公式ドキュメントや OSS の技術仕様書など、正確性が重要な文書では、誤った修正の影響を防ぐためにレビュープロセスが求められている
- AI に文書をガンガン更新してもらいたいが、すべてを無条件に反映するのではなく、変更内容を確認してから取捨選択したい。編集提案はそのレビューの仕組みとして機能する

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

### モデル

編集提案は以下の 4 つのモデルで構成されている。

| モデル                   | 役割                                                                               | Git の対応概念        |
| ------------------------ | ---------------------------------------------------------------------------------- | --------------------- |
| `Suggestion`             | 編集提案そのもの。タイトル・本文・ステータス・所属トピックを保持する               | Pull Request          |
| `SuggestionPage`         | 編集提案に含まれる個別ページの最新内容。複数メンバーが共同で編集できる共有リソース | remote feature branch |
| `SuggestionPageRevision` | `SuggestionPage` への変更履歴のスナップショット                                    | commits on branch     |
| `SuggestionComment`      | 編集提案に紐づくコメント                                                           | PR comment            |

`Page` / `DraftPage` / `PageRevision` の関係性については [ページ編集 仕様書](../page/edit.md) を参照。

### ステータスと状態遷移

`Suggestion` には以下の 4 つのステータスがある（`SuggestionStatus` 型、整数値）。

| 値  | 名称                      | 説明                       |
| --- | ------------------------- | -------------------------- |
| 0   | `SuggestionStatusDraft`   | 下書き（作成中の編集提案） |
| 1   | `SuggestionStatusOpen`    | オープン（レビュー待ち）   |
| 2   | `SuggestionStatusApplied` | 反映済み                   |
| 3   | `SuggestionStatusClosed`  | クローズ                   |

```
作成 ──→ Open ──┬─→ Applied
                └─→ Closed
```

現在の実装では、編集提案は作成時に `Open` ステータスで開始する。`Draft` ステータスは `SuggestionStatus` 型として将来の拡張のために定義されているものの、UI からのドラフト保存フローはまだ提供していない。

#### ステータス変更のべき等性

反映・クローズのリクエストは**べき等**に振る舞う。既に目的のステータスになっている場合はエラーを返さず、何もせずに成功レスポンスを返す。

- 反映済みの編集提案に対する反映リクエスト → 何もせず成功（リダイレクト）
- クローズ済みの編集提案に対するクローズリクエスト → 何もせず成功（リダイレクト）

ユーザーの意図は「この編集提案を反映したい」であり、既に反映済みなら目的は達成されている。エラーを返しても混乱を招くだけであり、2 人が同時に反映ボタンを押した場合も両方に成功レスポンスを返す方が UX として自然である。

### 編集提案の作成

- ユーザーはトピック内でページの作成や既存ページの編集を提案できる
- 新規ページ（まだ一度も公開されていないページ）の下書きも編集提案に含めることができる
- 編集提案は 1 つのトピック内のページに限定される。複数トピックにまたがる編集提案は作成できない
- 複数のページの変更を 1 つの編集提案にまとめることができる
- 編集提案ではページのトピック変更は対象外とする。トピックの変更は [ページの移動 仕様書](../page/move.md) を参照
- 編集提案の本文・編集提案コメントの本文はプレーンテキストで記述する。改行は表示時に CSS の `white-space: pre-wrap` で保持し、URL は表示時にヘルパーで自動的にリンク化する。Markdown 記法・Wiki リンク記法はサポートしない
- 編集提案作成画面では、対象トピック内の自分の下書きページがチェックボックス付きで表示される。チェックを入れた下書きページが編集提案ページ（`SuggestionPage`）として登録される
- 編集提案作成時、各下書きページから `linked_page_ids` と `featured_image_attachment_id` をそのままコピーして `SuggestionPage` に保存する。反映時の Markdown パイプライン再実行を避けるため、write time に計算した値を保持する
- 編集提案作成時、各下書きページの `suggestion_page_id` に作成した `SuggestionPage` の ID を設定し、作成者がそのまま編集提案モードでページ編集を継続できるようにする
- 編集提案の作成はスペースメンバー（アクティブ）のみ可能。公開トピックであってもスペースへの参加が必要となる
- 作成者がスペースから退会しても、作成済みの編集提案は保持される

### 編集提案の編集

- オープン状態の編集提案のタイトルと本文を編集できる
- 編集はスペースメンバー（アクティブ）であれば誰でも可能（作成者以外も可）
- 反映済み・クローズ済みの編集提案は編集できない

### コメント

- ユーザーは編集提案にコメントを投稿できる（スペースメンバー（アクティブ）のみ）
- オープン状態の編集提案のコメントは編集できる。編集権限はスペースメンバー（アクティブ）であれば誰でも可能（作成者以外も可）
- コメントは編集提案内で連番（`number`）を持ち、URL のキーとして使用される（`/s/{space}/suggestions/{number}/comments/{comment_number}`）

### 編集提案ページの追加・削除

- オープン状態の編集提案に対して、下書きページを追加できる
- 既存の編集提案ページを編集提案から削除できる
- 編集提案に含まれるページが 1 つの場合は削除不可（空の編集提案は許可しない）
- 追加・削除の権限は `CanAddSuggestionPage` / `CanRemoveSuggestionPage`（オープン状態のスペースメンバー（アクティブ））

### 編集提案ページの編集

- 編集提案の変更差分画面（`/s/{space}/suggestions/{number}/changes`）の各ページごとに「編集する」ボタンを配置している
- 「編集する」ボタンを押すと、`DraftPage` の `suggestion_page_id` に対象の `SuggestionPage` ID を設定し、`SuggestionPage` の最新内容で `DraftPage` を初期化してページ編集画面に遷移する
- 既存の `DraftPage` が、対象の `SuggestionPage` 以外を指している場合（別の編集提案にリンクされている、または通常編集である）は確認画面を表示する
  - **編集を続ける**: 既存の `DraftPage` を `SuggestionPage` の最新内容で上書きし、`suggestion_page_id` を対象の `SuggestionPage` に切り替える
  - **下書きを保持する**: 編集提案の編集を開始しない。既存の `DraftPage` はそのまま
  - 確認画面のメッセージ・ボタン文言は下書きの種類（通常 / 別の編集提案）に依らず単一バリアントを表示する
- 既存の `DraftPage` が既に対象の `SuggestionPage` にリンクされている場合は、確認画面をスキップしてそのままページ編集画面に遷移する
- 編集提案作成者の場合は作成時に `suggestion_page_id` が設定済みのため、確認画面をスキップしてそのままページ編集画面に遷移する
- ページ編集画面では `DraftPage.SuggestionPageID` が NOT NULL の場合に表示が変わる
  - 「編集提案 #xxx のページを編集中です」というメッセージを表示
  - 「トピックに公開」ボタンを非表示にする
  - 代わりに「編集提案を更新」ボタンを表示する
- 「編集提案を更新」を押すと `DraftPage` の内容で `SuggestionPage` のコンテンツを更新し、新しい `SuggestionPageRevision` を作成する
- 反映済み・クローズ済みの編集提案のページは更新できない

### 編集提案の反映

- 反映権限を持つユーザー（スペースオーナー、トピック管理者）はオープンな編集提案を反映できる
- 反映時は各 `SuggestionPage` の最新内容で対応する `Page` を更新し、`PageRevision` を作成する
- 反映時に `SuggestionPage` の `linked_page_ids` と `featured_image_attachment_id` を `Page` にコピーする（write time に計算済みの値を再利用）
- 反映時に `page_attachment_references` テーブルを更新する
- 反映時に `PageEditor` を作成・更新し、最終編集時刻を更新する
- 反映時にトピックメンバーの `last_page_modified_at` を更新する
- 反映後、編集提案のステータスを `Applied` に変更し `applied_at` に現在時刻を設定する
- 反映後、関連する `DraftPage` の `suggestion_page_id` をクリアする
- 新規ページ（`page_revision_id` が NULL の `SuggestionPage`）を含む編集提案も反映できる。新規ページの場合は `Page` を初めて公開する（`published_at` を設定する）
- ベースリビジョンが古くなっている（編集提案作成後にページが直接更新された）場合でも、強制的に上書き反映される

### 編集提案のクローズ

- 編集提案の作成者またはスペースオーナー / トピック管理者はオープンな編集提案をクローズできる
- クローズ後、編集提案のステータスを `Closed` に変更する
- クローズ後、関連する `DraftPage` の `suggestion_page_id` をクリアする
- 反映済みの編集提案はクローズできない

### ページ移動の制限

- オープンな編集提案で参照されているページは別のトピックに移動できない
- ページ移動のバリデーター（`PageMoveCreateValidator`）が `SuggestionPageRepository.ExistsByPageIDAndOpenStatus` を呼び出してチェックする
- 編集提案の `topic_id` は作成時のトピックを指し続けるため、ページが別トピックに移動すると不整合が生じるのを防ぐための制約

### 認可ポリシー

`TopicPolicy` インターフェースに定義された編集提案関連の権限。

| メソッド                     | 判定対象     | 概要                                                                                                                                |
| ---------------------------- | ------------ | ----------------------------------------------------------------------------------------------------------------------------------- |
| `CanCreateSuggestion`        | `Topic`      | 編集提案の作成権限                                                                                                                  |
| `CanApplySuggestion`         | `Suggestion` | 編集提案の反映権限（スペースオーナー / トピック管理者のみ）                                                                         |
| `CanCloseSuggestion`         | `Suggestion` | 編集提案のクローズ権限（作成者 + スペースオーナー / トピック管理者）                                                                |
| `CanUpdateSuggestion`        | `Suggestion` | 編集提案のタイトル・本文の編集権限（オープン状態のみ）                                                                              |
| `CanAddSuggestionPage`       | `Suggestion` | 編集提案へのページ追加権限（オープン状態のみ）                                                                                      |
| `CanRemoveSuggestionPage`    | `Suggestion` | 編集提案からのページ削除権限（オープン状態のみ）                                                                                    |
| `CanEditSuggestionPage`      | `Suggestion` | 編集提案ページの編集権限（`Policy` 自体は状態をチェックしない。オープン状態の制約は `StartSuggestionPageEditUsecase` 側で強制する） |
| `CanCreateSuggestionComment` | `Suggestion` | コメント作成権限                                                                                                                    |
| `CanUpdateSuggestionComment` | `Suggestion` | コメント編集権限（オープン状態のみ）                                                                                                |

ロールごとの判定の概要（主要なアクションを抜粋。すべてのアクションでアクティブなスペースメンバーであることが前提）:

| ロール       | 編集提案作成     | 反映 | クローズ             | 更新（タイトル・本文・ページ追加・ページ削除）           |
| ------------ | ---------------- | ---- | -------------------- | -------------------------------------------------------- |
| Space Owner  | 同一スペース     | 可   | 可                   | 可                                                       |
| Topic Admin  | 同一トピック     | 可   | 作成者 or 同トピック | 同一トピック                                             |
| Topic Member | 同一トピック     | 不可 | 作成者のみ           | 同一トピック                                             |
| Topic Guest  | 公開トピックのみ | 不可 | 作成者のみ           | スペースメンバーであれば可能（トピック所属チェックなし） |

「Topic Guest」はトピックメンバーではないがアクティブなスペースメンバーであるユーザーを表す。コメント作成・コメント更新・編集提案ページの編集も上記の「更新」と同じ条件で判定される。

### UI

#### トピック詳細画面

- 画面上部に「ページ」と「編集提案」のタブを表示する
- GitHub の Code / Pull requests タブと同様の UI
- デフォルトは「ページ」が選択されており、トピック内のページが一覧で表示される
- 「編集提案」を選択すると、編集提案一覧画面に遷移する
- 「編集提案」タブはフィーチャーフラグ `go_suggestion` が有効な場合のみ表示される

#### 編集提案一覧画面

- スペースメンバーが作成した編集提案をリスト表示する
- GitHub のようにオープン / クローズで絞り込み可能
  - オープン表示: 下書き・オープンステータスの編集提案
  - クローズ表示: 反映済み・クローズステータスの編集提案

#### 編集提案詳細画面

- 「会話」「編集したページ」の 2 つのタブ
- デフォルトは「会話」タブがアクティブ
- 「会話」タブ: 編集提案の概要とコメント表示
- 「編集したページ」タブ: 各 `SuggestionPage` の差分表示（変更差分画面 `/s/{space}/suggestions/{number}/changes`）
- 「反映する」ボタン（`CanApplySuggestion` が true の場合）
- 「クローズする」ボタン（`CanCloseSuggestion` が true の場合）
- タイトル横に「編集する」ボタンを配置（`@components.MainTitle` の `Actions`、トピック画面の「新規ページ」ボタンと同じパターン）。`CanUpdateSuggestion` が true かつオープン状態の場合のみ表示
- 各コメントの投稿ヘッダー（`@atname` + 日時の行）の右端に「...」ドロップダウンメニューを配置。メニュー内に「編集する」アイテムがあり、`CanUpdateSuggestionComment` が true の場合のみ表示
- 公開トピックの編集提案は未ログインでも閲覧可能。非公開トピックは未ログインで 404 を返す

#### 編集提案作成画面

- トピック内の自分の下書きページがチェックボックス付きで表示される
- 編集提案したい下書きページを選択し、タイトルと概要を入力して「作成する」ボタンを押すと編集提案が作成される

#### ページ編集画面

- 通常時は「トピックに公開」と「下書き保存」ボタンが表示されている
- 「下書き保存」ボタンの右側にドロップダウンアイコンがあり、「下書き保存して編集提案を作成する...」アクションを選択できる
  - 実行すると下書き保存後、保存した下書きページが選択された状態で編集提案作成画面に直接遷移する
  - フィーチャーフラグ `go_suggestion` が有効な場合のみ表示
- `DraftPage.SuggestionPageID` が NOT NULL の場合は編集提案モードに切り替わり、「トピックに公開」ボタンが「編集提案を更新」ボタンに置き換わる

#### 下書き一覧画面（`GET /drafts`）

- 各トピックグループに「編集提案する...」ボタンが表示される（フィーチャーフラグが有効な場合のみ）
- 「編集提案する...」を押すと、そのトピックにスコープされた編集提案作成画面に遷移する

## 設計

<!--
ガイドライン:
- 現在の技術的な実装の詳細を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - 技術スタック（使用するライブラリ、フレームワーク、ツールなど）
  - アーキテクチャ（システム全体の構成、コンポーネント間の関係など)
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - セキュリティ設計（認証・認可、トークン管理、Rate Limitingなど）
  - コード設計（パッケージ構成、主要な構造体、インターフェースなど）
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### 命名

- **モデル名（英語）**: `Suggestion`, `SuggestionPage`, `SuggestionPageRevision`, `SuggestionComment`
- **テーブル名**: `suggestions`, `suggestion_pages`, `suggestion_page_revisions`, `suggestion_comments`
- **日本語 UI 表示名**: 編集提案

Google Docs の「Suggestions」と同じ用語を採用した。日本語名は「提案」だと一般名詞すぎて固有名詞感がないため、「編集提案」を維持する。

### 関連モデル

編集提案機能は、ページのリビジョン管理システム（`Page`、`PageRevision`、`DraftPage`）を基盤として構築されている。`Page` モデルとリビジョン管理の詳細は [ページ編集 仕様書](../page/edit.md) を参照。

### Git モデルとの対応

| Git                   | Wikino                   | 説明                                     |
| --------------------- | ------------------------ | ---------------------------------------- |
| remote main branch    | `Page`                   | 公開済みのページ（最新の正式な内容）     |
| local working tree    | `DraftPage`              | スペースメンバーごとの個人の下書き       |
| remote feature branch | `SuggestionPage`         | 編集提案に紐づくページの変更内容（共有） |
| commits on branch     | `SuggestionPageRevision` | 編集提案ページへの変更履歴               |
| Pull Request          | `Suggestion`             | 変更のレビュー・反映を管理する単位       |

`SuggestionPage` が「リモートフィーチャーブランチ」に相当するリソースとして機能する。これにより、複数のスペースメンバーが同じ編集提案に対して変更を加えることが可能になる。`DraftPage` はあくまで個人のワーキングツリーであり、編集提案の共有リソースとは分離されている。

### DraftPage と編集提案の連携

編集提案のページを編集する際は、既存の `DraftPage` の仕組みを再利用する。`draft_pages` テーブルに `suggestion_page_id`（nullable FK → `suggestion_pages`）が存在し、`DraftPage` が「どのブランチをチェックアウトしているか」を表現する。

| `suggestion_page_id` | `DraftPage` の役割   | 自動保存先  | 内容の初期化元                    |
| -------------------- | -------------------- | ----------- | --------------------------------- |
| NULL                 | 通常のページ編集     | `DraftPage` | `Page` の現在のコンテンツ         |
| NOT NULL             | 編集提案のページ編集 | `DraftPage` | `SuggestionPage` の最新コンテンツ |

`DraftPage` のユニーク制約は `[space_member_id, page_id]` であり、`suggestion_page_id` を含めない。これにより、同一ページに対して通常編集と編集提案の編集を同時に持つことはできない。Git の「ワーキングツリーは同時に 1 つのブランチしかチェックアウトできない」のと同じ制約であり、概念モデルがシンプルに保たれる。

### URL 設計

| 操作                 | HTTP メソッド | URL                                                                     | ハンドラー                               |
| -------------------- | ------------- | ----------------------------------------------------------------------- | ---------------------------------------- |
| 一覧                 | GET           | `/s/{space}/topics/{topic}/suggestions`                                 | `suggestion.Handler.Index`               |
| 作成フォーム         | GET           | `/s/{space}/topics/{topic}/suggestions/new`                             | `suggestion.Handler.New`                 |
| 作成                 | POST          | `/s/{space}/topics/{topic}/suggestions`                                 | `suggestion.Handler.Create`              |
| 詳細                 | GET           | `/s/{space}/suggestions/{number}`                                       | `suggestion.Handler.Show`                |
| 編集フォーム         | GET           | `/s/{space}/suggestions/{number}/edit`                                  | `suggestion.Handler.Edit`                |
| 更新                 | PATCH         | `/s/{space}/suggestions/{number}`                                       | `suggestion.Handler.Update`              |
| 変更差分画面         | GET           | `/s/{space}/suggestions/{number}/changes`                               | `suggestion_change.Handler.Index`        |
| 反映                 | POST          | `/s/{space}/suggestions/{number}/apply`                                 | `suggestion_apply.Handler.Create`        |
| クローズ             | POST          | `/s/{space}/suggestions/{number}/close`                                 | `suggestion_close.Handler.Create`        |
| コメント作成         | POST          | `/s/{space}/suggestions/{number}/comments`                              | `suggestion_comment.Handler.Create`      |
| コメント編集フォーム | GET           | `/s/{space}/suggestions/{number}/comments/{comment_number}/edit`        | `suggestion_comment_edit.Handler.Edit`   |
| コメント更新         | PATCH         | `/s/{space}/suggestions/{number}/comments/{comment_number}`             | `suggestion_comment_edit.Handler.Update` |
| ページ追加フォーム   | GET           | `/s/{space}/suggestions/{number}/suggestion_pages/new`                  | `suggestion_page.Handler.New`            |
| ページ追加           | POST          | `/s/{space}/suggestions/{number}/suggestion_pages`                      | `suggestion_page.Handler.Create`         |
| ページ更新           | PATCH         | `/s/{space}/suggestions/{number}/suggestion_pages/{suggestion_page_id}` | `suggestion_page.Handler.Update`         |
| ページ削除           | DELETE        | `/s/{space}/suggestions/{number}/suggestion_pages/{suggestion_page_id}` | `suggestion_page.Handler.Delete`         |
| ページ編集確認画面   | GET           | `/s/{space}/suggestions/{number}/page_edits/{suggestion_page_id}`       | `suggestion_page_edit.Handler.Show`      |
| ページ編集開始       | POST          | `/s/{space}/suggestions/{number}/page_edits`                            | `suggestion_page_edit.Handler.Create`    |

反映・クローズを独立したリソース（`suggestion_apply`、`suggestion_close`）として切り出すことで、`PATCH /s/{space}/suggestions/{number}` を編集提案のタイトル・本文の更新に使用できる。

コメントの URL には `suggestion_comments.id`（UUID）ではなく、編集提案内の連番（`comment_number`）を使用する。これにより、編集提案の `number`（スペース内連番）と統一感のある URL になる。

### フィーチャーフラグ

編集提案機能は `go_suggestion` フィーチャーフラグで制御されている（`model.FeatureFlagSuggestion`）。

- **制御方式**: リバースプロキシミドルウェア（`internal/middleware/reverse_proxy.go`）の `featureFlaggedPatterns` に編集提案関連の URL パターンを登録している
- **対象 URL パターン**:
  - `^/s/[^/]+/topics/\d+/suggestions`（一覧・作成）
  - `^/s/[^/]+/suggestions/\d+`（詳細・反映・クローズ・コメント・ページ操作）
- **フラグ無効時の挙動**: Rails 版にプロキシされる。Rails 版に該当機能がないため 404 になる
- **クリーンアップ**: 機能が安定し全ユーザーに公開した後、フラグを削除し `goHandledPrefixPaths` または `goHandledRegexPatterns` に移動する予定

### テーブル設計

#### `suggestions`

```sql
CREATE TABLE suggestions (
    id uuid DEFAULT generate_ulid() NOT NULL PRIMARY KEY,
    space_id uuid NOT NULL,
    topic_id uuid NOT NULL,
    created_space_member_id uuid NOT NULL,
    title character varying NOT NULL,
    body character varying NOT NULL,
    status integer DEFAULT 0 NOT NULL,
    applied_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    number integer DEFAULT 0 NOT NULL
);

CREATE UNIQUE INDEX idx_suggestions_space_id_number ON suggestions(space_id, number);
CREATE INDEX idx_suggestions_topic_id_status ON suggestions(topic_id, status);
CREATE INDEX idx_suggestions_space_id ON suggestions(space_id);
CREATE INDEX idx_suggestions_created_space_member_id ON suggestions(created_space_member_id);
```

- `number` はスペース内での連番（`pages.number` と同じパターン）。URL のキーとして使用する
- `body` はプレーンテキストとして保存する。表示時に CSS の `white-space: pre-wrap` で改行を保持し、ヘルパーで URL を自動的にリンク化する

#### `suggestion_pages`

```sql
CREATE TABLE suggestion_pages (
    id uuid DEFAULT generate_ulid() NOT NULL PRIMARY KEY,
    space_id uuid NOT NULL,
    suggestion_id uuid NOT NULL,
    page_id uuid NOT NULL,
    page_revision_id uuid,
    title character varying,
    body character varying DEFAULT '' NOT NULL,
    body_html character varying DEFAULT '' NOT NULL,
    linked_page_ids character varying[] DEFAULT '{}' NOT NULL,
    featured_image_attachment_id uuid REFERENCES attachments(id),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE UNIQUE INDEX idx_suggestion_pages_suggestion_id_page_id ON suggestion_pages(suggestion_id, page_id);
CREATE INDEX idx_suggestion_pages_space_id ON suggestion_pages(space_id);
```

- `pages` や `draft_pages` と同じパターンで、コンテンツ（`title`、`body`、`body_html`）を直接保持する。変更履歴は `suggestion_page_revisions` に記録する
- `page_id` は NOT NULL（ページは事前に作成されている）
- `page_revision_id` は nullable。新規ページ（まだ一度も公開されていないページ）の場合は NULL になる。値がある場合は編集提案作成時点のベースリビジョンを表す
- `linked_page_ids` は `body` 内の Wiki リンクから解決されたページ ID の配列。編集提案作成時に `DraftPage` からコピーし、反映時にそのまま `pages.linked_page_ids` にコピーする。反映時の Markdown パイプライン再実行を避けるため、write time（編集提案作成時）に計算して保存する
- `featured_image_attachment_id` は `body` 内の最初の画像から抽出されたアイキャッチ画像の添付ファイル ID

#### `suggestion_page_revisions`

```sql
CREATE TABLE suggestion_page_revisions (
    id uuid DEFAULT generate_ulid() NOT NULL PRIMARY KEY,
    space_id uuid NOT NULL,
    suggestion_page_id uuid NOT NULL,
    editor_space_member_id uuid NOT NULL,
    title character varying,
    body character varying DEFAULT '' NOT NULL,
    body_html character varying DEFAULT '' NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX idx_suggestion_page_revisions_suggestion_page_id_created_at
    ON suggestion_page_revisions(suggestion_page_id, created_at);
CREATE INDEX idx_suggestion_page_revisions_space_id ON suggestion_page_revisions(space_id);
```

#### `suggestion_comments`

```sql
CREATE TABLE suggestion_comments (
    id uuid DEFAULT generate_ulid() NOT NULL PRIMARY KEY,
    space_id uuid NOT NULL,
    suggestion_id uuid NOT NULL,
    created_space_member_id uuid NOT NULL,
    body character varying DEFAULT '' NOT NULL,
    number integer NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE UNIQUE INDEX idx_suggestion_comments_suggestion_id_number ON suggestion_comments(suggestion_id, number);
CREATE INDEX idx_suggestion_comments_suggestion_id_created_at ON suggestion_comments(suggestion_id, created_at);
CREATE INDEX idx_suggestion_comments_space_id ON suggestion_comments(space_id);
```

- `number` は編集提案内での連番。URL のキーとして使用する
- `body` はプレーンテキストとして保存する。表示時に CSS の `white-space: pre-wrap` で改行を保持し、ヘルパーで URL を自動的にリンク化する

#### `draft_pages` への変更

```sql
ALTER TABLE draft_pages ADD COLUMN suggestion_page_id uuid REFERENCES suggestion_pages(id);
ALTER TABLE draft_pages ADD COLUMN featured_image_attachment_id uuid REFERENCES attachments(id);

CREATE INDEX index_draft_pages_on_suggestion_page_id
    ON draft_pages(suggestion_page_id) WHERE suggestion_page_id IS NOT NULL;
```

- `suggestion_page_id`: 編集提案のページ編集時にリンクする。NULL なら通常のページ編集、NOT NULL なら編集提案のページ編集
- `featured_image_attachment_id`: 編集提案作成時に `DraftPage` から `SuggestionPage` にコピーするために追加（既に存在する `linked_page_ids` と同じパターン）
- 既存のユニーク制約 `[space_member_id, page_id]` は変更しない

### モデル定義

```go
// internal/model/suggestion.go
type SuggestionStatus int32

const (
    SuggestionStatusDraft   SuggestionStatus = 0
    SuggestionStatusOpen    SuggestionStatus = 1
    SuggestionStatusApplied SuggestionStatus = 2
    SuggestionStatusClosed  SuggestionStatus = 3
)

type Suggestion struct {
    ID                   SuggestionID
    SpaceID              SpaceID
    TopicID              TopicID
    CreatedSpaceMemberID SpaceMemberID
    Number               SuggestionNumber
    Title                string
    Body                 string
    Status               SuggestionStatus
    AppliedAt            *time.Time
    CreatedAt            time.Time
    UpdatedAt            time.Time
}

// internal/model/suggestion_page.go
type SuggestionPage struct {
    ID                        SuggestionPageID
    SpaceID                   SpaceID
    SuggestionID              SuggestionID
    PageID                    PageID
    PageRevisionID            *PageRevisionID
    Title                     *string
    Body                      string
    BodyHTML                  string
    LinkedPageIDs             []PageID
    FeaturedImageAttachmentID *AttachmentID
    CreatedAt                 time.Time
    UpdatedAt                 time.Time
}

// internal/model/suggestion_page_revision.go
type SuggestionPageRevision struct {
    ID                  SuggestionPageRevisionID
    SpaceID             SpaceID
    SuggestionPageID    SuggestionPageID
    EditorSpaceMemberID SpaceMemberID
    Title               *string
    Body                string
    BodyHTML            string
    CreatedAt           time.Time
    UpdatedAt           time.Time
}

// internal/model/suggestion_comment.go
type SuggestionComment struct {
    ID                   SuggestionCommentID
    SpaceID              SpaceID
    SuggestionID         SuggestionID
    CreatedSpaceMemberID SpaceMemberID
    Number               SuggestionCommentNumber
    Body                 string
    CreatedAt            time.Time
    UpdatedAt            time.Time
}
```

### コード設計

#### Handler

| パッケージ                                 | 主な責務                                                     |
| ------------------------------------------ | ------------------------------------------------------------ |
| `internal/handler/suggestion`              | 編集提案の一覧・詳細・作成フォーム・作成・編集フォーム・更新 |
| `internal/handler/suggestion_change`       | 編集提案の変更差分画面                                       |
| `internal/handler/suggestion_apply`        | 編集提案の反映                                               |
| `internal/handler/suggestion_close`        | 編集提案のクローズ                                           |
| `internal/handler/suggestion_comment`      | コメントの作成                                               |
| `internal/handler/suggestion_comment_edit` | コメントの編集フォーム・更新                                 |
| `internal/handler/suggestion_page`         | 編集提案ページの追加フォーム・追加・更新・削除               |
| `internal/handler/suggestion_page_edit`    | 編集提案ページの編集開始（確認画面 + 編集開始処理）          |

#### UseCase

| UseCase                          | 責務                                                      |
| -------------------------------- | --------------------------------------------------------- |
| `GetSuggestionListUsecase`       | 編集提案一覧の取得                                        |
| `GetSuggestionDetailUsecase`     | 編集提案詳細の取得                                        |
| `GetSuggestionEditUsecase`       | 編集提案編集フォーム用データの取得                        |
| `GetSuggestionNewUsecase`        | 編集提案作成フォーム用データの取得                        |
| `GetSuggestionDiffUsecase`       | 変更差分の取得                                            |
| `GetSuggestionPageNewUsecase`    | ページ追加フォーム用データの取得                          |
| `GetSuggestionCommentUsecase`    | コメント取得                                              |
| `CreateSuggestionUsecase`        | 編集提案の作成                                            |
| `UpdateSuggestionUsecase`        | 編集提案のタイトル・本文の更新                            |
| `ApplySuggestionUsecase`         | 編集提案の反映                                            |
| `CloseSuggestionUsecase`         | 編集提案のクローズ                                        |
| `AddSuggestionPageUsecase`       | 編集提案へのページ追加                                    |
| `RemoveSuggestionPageUsecase`    | 編集提案からのページ削除                                  |
| `UpdateSuggestionPageUsecase`    | 編集提案ページの内容更新（`SuggestionPageRevision` 作成） |
| `StartSuggestionPageEditUsecase` | 編集提案ページの編集開始（`DraftPage` の初期化）          |
| `CreateSuggestionCommentUsecase` | コメントの作成                                            |
| `UpdateSuggestionCommentUsecase` | コメントの更新                                            |

#### Validator

- `SuggestionCreateValidator`: 編集提案作成のバリデーション
- `SuggestionUpdateValidator`: 編集提案更新のバリデーション
- `SuggestionPageCreateValidator`: 編集提案ページ追加のバリデーション
- `SuggestionPageUpdateValidator`: 編集提案ページ更新のバリデーション
- `SuggestionCommentCreateValidator`: コメント作成のバリデーション
- `SuggestionCommentUpdateValidator`: コメント更新のバリデーション

#### Repository

- `SuggestionRepository`: `suggestions` テーブルのアクセス
- `SuggestionPageRepository`: `suggestion_pages` テーブルのアクセス（`ExistsByPageIDAndOpenStatus` を含む）
- `SuggestionPageRevisionRepository`: `suggestion_page_revisions` テーブルのアクセス
- `SuggestionCommentRepository`: `suggestion_comments` テーブルのアクセス

### 差分表示

`internal/usecase/get_suggestion_diff.go` の `GetSuggestionDiffUsecase` が各 `SuggestionPage` の最新内容とベースの `PageRevision` の差分を計算する。`PageRevisionID` が NULL の場合（新規ページ）は空文字列をベースとして差分を計算する。

差分表示は `internal/templates/components/diff.templ` の汎用差分コンポーネントで描画する。

### マージコンフリクト（ベースリビジョンの乖離）

編集提案作成後にベースとなる `Page` が更新された場合の扱い:

- `suggestion_pages.page_revision_id` でベースとなるリビジョンを記録しているため、ベースが古くなったことは検出できる
- 現在の実装では「ベースが変わっていても強制的に上書き反映」とする
- 将来的にはコンフリクト検出・手動解決の UI を追加する可能性がある

### 同時編集

同一編集提案ページの同時編集に対する競合は、初期実装では「最後に保存した人が勝つ（last-write-wins）」とする。将来的には CRDT で解決する計画。

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

ページモデルの命名・設計に関する採用しなかった方針は [ページ編集 仕様書](../page/edit.md) の「採用しなかった方針」セクションを参照。

### 編集提案でのトピック変更

編集提案にトピック変更を含めることを検討したが、以下の理由から対象外とした。

- トピックごとにユーザーの権限が異なるため、トピックを横断する提案のレビュー権限が曖昧になる（移動元と移動先のどちらのトピック権限で判断すべきか）
- 「内容の変更」と「トピックの移動」が 1 つの提案に混在すると、レビュアーの判断が複雑になる
- 編集提案の主目的は「ページの内容改善を気軽に提案できること」であり、トピック移動は性質の異なる操作である
- トピック変更が必要な場合は、権限のあるユーザーが直接編集で対応すれば十分である

将来的にトピック変更対応を追加することは可能だが、初期段階では内容変更に集中し、シンプルな設計を維持する。

### `DraftPage` の変更を暗黙的に編集提案に反映する案

`DraftPage` がスペースメンバーごとに作成される設計を活かし、「`DraftPage` が編集されたら暗黙的に `SuggestionPageRevision` を自動作成する」という案を検討した。`DraftPage` と編集提案の間に明示的なリンクを持たず、`DraftPage` の更新をトリガーとして編集提案のリビジョンを作成する方式。

**不採用の理由**:

- `DraftPage` の更新が通常の編集フロー用なのか編集提案フロー用なのかを区別できない。明示的なリンク（`suggestion_page_id`）がないため、「この下書きの変更はどこに反映されるのか」が不明確になる
- 代わりに、`draft_pages` テーブルに `suggestion_page_id`（nullable FK）を追加し、`DraftPage` が「どのブランチをチェックアウトしているか」を明示する方式を採用した

### `suggestion_pages` と `draft_pages` の中間テーブルを設ける案

`suggestion_pages` と `draft_pages` の間に多対多の中間テーブルを設け、スペースメンバーごとにレコードを作成する案を検討した。

**不採用の理由**:

- `DraftPage` から見ると、同時に複数の編集提案にリンクされるケースは実運用で起きづらい（下書きを保存したときに「どの編集提案に反映されるのか」が曖昧になるため）。実質的に `DraftPage` 側は 1 対 1 の関係になる
- 1 対 1 であれば、中間テーブルを設けるよりも `draft_pages` に nullable FK（`suggestion_page_id`）を追加する方がシンプル
- nullable FK 方式でも、`SuggestionPage` 側からは複数の `DraftPage`（異なるスペースメンバー）がリンクされる多対 1 の関係を表現できる

### `DraftPage` のユニーク制約を拡張する案

`draft_pages` のユニーク制約を `[space_member_id, page_id]` から `[space_member_id, page_id, suggestion_page_id]` に変更し、同一ページに対して通常編集と編集提案の編集を同時に持てるようにする案を検討した。

**不採用の理由**:

- 同一ページに対して複数の `DraftPage` が存在すると、ユーザーにとって「どの下書きがどの目的か」の管理が複雑になる
- Git でも 1 つのワーキングツリーは同時に 1 つのブランチしかチェックアウトできない。同じ制約を Wikino にも適用する方が概念モデルがシンプルになる
- 編集提案の編集を始める際に確認画面を表示し、既存の下書きを保持するか編集提案の内容に切り替えるかをユーザーに選択してもらうことで、ユニーク制約を変更せずに対応できる
- PostgreSQL では NULL は一意制約で「異なる値」として扱われるため、通常編集（`suggestion_page_id = NULL`）の `DraftPage` が無制限に作れてしまう。部分ユニークインデックスが必要になりスキーマが複雑化する
- 下書き一覧画面で同じページが複数表示され、「この下書きはどの編集提案のものか」をユーザーが管理する負担が増える
- ページ編集画面での自動保存先がどの `DraftPage` になるのかが曖昧になる

### 編集提案ページ編集の確認画面で下書きの種類によってメッセージを出し分ける案

編集提案ページの編集を開始する際の確認画面で、既存の下書きが「通常の下書き（`suggestion_page_id` が NULL）」なのか「別の編集提案にリンクされた下書き」なのかを区別し、それぞれ別の見出し・本文・ボタン文言を表示する案を検討した。`UseCase` の出力に下書きの種類を表す `ConflictDraftKind` を含め、ハンドラーがクエリパラメータに `draft_kind=other_suggestion` を付与し、テンプレート側で `IsOtherSuggestionDraft` フラグに応じて文言を切り替える実装を一度試した。

**不採用の理由**:

- ユーザーが取りうる選択肢は「編集を続ける（上書き）」「下書きを保持する」の二択であり、下書きの種類に依らず同じである。文言を出し分けても判断材料が増えない
- メッセージのバリアントが増えると i18n キーが分岐し、メンテナンスコストが高くなる
- ハンドラー → テンプレート間でクエリパラメータ経由のフラグを引き回す必要があり、実装が複雑になる
- 単一の文言（「既存の下書きがあります」）で「別の下書きと差し替えるか保持するか」というユーザー判断は十分伝わる

### `edit_suggestions` というテーブル名

当初テーブル名を `edit_suggestions` としていたが、`suggestions` にリネームした。

**リネームの理由**:

- `edit_suggestions` は正確だが冗長。関連テーブル名（`edit_suggestion_pages`、`edit_suggestion_page_revisions`）も長くなる
- Google Docs が同種の機能に「Suggestions」という用語を使っており、英語圏でも通じやすい
- Wikino の文脈では「suggestions」= ページ編集の提案であることが明確なため、`edit_` プレフィックスがなくても曖昧にならない
- 日本語名は「編集提案」を維持する

### `suggested_pages` というテーブル名

`suggestion_pages` を `suggested_pages`（形容詞 + 名詞）とする案を検討した。

**不採用の理由**:

- `suggestion_comments`（`suggestion` に属する `comments`）は所有関係を表す複合名詞であり、`suggested_comments`（提案されたコメント）とするのは意味的に不自然。プレフィックスの文法パターンが混在する
- `suggestion_pages`、`suggestion_comments` はいずれも「`suggestion` に属するもの」という所有関係を表す複合名詞パターンで統一できる。`order_items`、`project_members` と同じ慣習
- GitHub の API でも `pull_request_reviews`、`pull_request_comments` のように名詞の複合形が使われている

### 「編集リクエスト」という名称

最初「編集リクエスト」という名前を検討したが、「編集提案」のほうが気軽・柔らかい印象があるため「編集提案」を採用した。

- 「編集リクエスト」のニュアンス: 「変更してください」というやや能動的なイメージ。受け手にアクションを求めるイメージ。編集を取り込むことを前提としたイメージ
- 「編集提案」のニュアンス: 「こうしたらどうだろうか？」という控えめなイメージ。受け手に判断を委ねる受動的なイメージ。あくまでアイデアを提示したまでで、取り込まれなくても問題ないというイメージ

### 編集提案・編集提案コメントで Markdown をサポートする案

当初、編集提案の本文 (`suggestions.body`) と編集提案コメントの本文 (`suggestion_comments.body`) は Markdown で記述し、保存時にページと同じ Markdown パイプライン（Wiki リンク解決を含む）で `body_html` を生成して保存していた。その後、両方の Markdown サポートを廃止し、プレーンテキスト入力に変更した。

**不採用の理由**:

- 編集提案の本文に Markdown が必要になるほどの情報量を書くのは Wiki の責務分離として不自然である。詳細はページ本体に書き、編集提案では「なぜこの変更を提案するか」に集中するスタイルが望ましい
- Claude などの AI と既存テキストを修正するワークフローでも、編集提案や編集提案コメントの本文を Markdown で書きたい場面はほとんどない
- YAGNI: まずは Markdown なしでリリースし、本当に必要になったときに改めて検討する
- `body_html` カラムを保持し続けると、保存時にレンダリングするための Markdown パイプライン依存が UseCase に残り、Wikiリンク解決のための Repository 依存も増えて編集提案系の UseCase が肥大化する

**採用した方針**:

- `suggestions.body_html` / `suggestion_comments.body_html` カラムは削除する。`body` のみをプレーンテキストとして保存する
- 改行は表示時に CSS の `white-space: pre-wrap` で保持する（DB に `<br>` を入れない）
- URL は表示時にヘルパー（`internal/markup/linkify.go` の `LinkifyPlainText`）で都度リンク化する
- Wiki リンク `[[...]]` はサポートしない（バックリンクを生成する性質上、編集提案のような一時的なリソースから使えるのは違和感があるため）
- コードブロック・インラインコードはサポートしない（行ごとのコメント機能で対応予定）
- 影響範囲は `suggestions.body` / `suggestion_comments.body` のみ。`pages.body`、`suggestion_pages.body`、`suggestion_page_revisions.body` を含むページ系の Markdown サポートには触れない（`suggestion_pages` / `suggestion_page_revisions` は反映時にページ本体の系統に流れるため、`body_html` カラムを引き続き保持する）

### 編集提案をページの亜種として扱う案

`pages` テーブルに `type` カラムを追加し、通常のページを `type: note`、編集提案を `type: suggestion` として管理する案を検討した。「すべてのドキュメントはページ」という思想に基づき、編集提案の本文で Wiki リンク記法を自然に使えるようにする狙いがあった。

**不採用の理由**:

- ページモデルの責務が肥大化する。既存・今後のすべてのページ関連ロジック（検索、一覧、ページ番号、Wiki リンクの解決先など）で type の考慮が必要になる
- ページ一覧やトピック内ページ数など、既存のクエリすべてに `WHERE type = 'note'` が必要になり、漏れるとバグになる
- `[[編集提案のタイトル]]` でリンクできてしまうが、リンク先が編集提案の説明文ページになるのは意味的に不自然
- 編集提案の説明文は「変更に関するメタデータ」であり、Wiki のコンテンツではない。GitHub の PR description が「ファイル」ではないのと同じ
- そもそも編集提案・編集提案コメントの本文は Markdown サポートを廃止したため（前項参照）、Wiki リンク記法を自然に使えるようにする目的そのものが消滅した

### 編集提案番号をトピック内で採番する案

当初、編集提案の番号をトピック内で一意になるように採番していた（ユニークインデックス: `[topic_id, number]`）。その後、スペース内で一意になるように変更した（ユニークインデックス: `[space_id, number]`）。

**スペース内採番に変更した理由**:

- ページの番号がスペース内で一意に採番される設計と合わせることで、URL の形式に統一感が出る（`/s/{space}/pages/{number}` と `/s/{space}/suggestions/{number}`）
- 編集提案を将来的に別のトピックに移動する場合でも、URL が変わらない
- スペース内で番号が一意になるため、「提案 #5」のように番号だけで曖昧さなく参照できる（GitHub の Issue/PR がリポジトリ単位で採番されるのと同じ）
- 編集提案詳細の URL からトピックを省略でき、URL が簡潔になる

### PATCH エンドポイントで反映・クローズをアクション分岐する案

`PATCH /s/{space}/suggestions/{number}` の 1 つのエンドポイントで、リクエストボディの `action` パラメータにより反映（apply）・クローズ（close）・内容更新を分岐する案を検討した。

**不採用の理由**:

- 1 つのエンドポイントに複数の責務が混在し、ハンドラーの見通しが悪くなる
- ハンドラーガイドの「標準ファイル名 8 種類のみ」の原則に沿わない。`update.go` 内で分岐するとファイルが肥大化する
- 反映・クローズ・内容更新はそれぞれ必要な権限やバリデーションが異なるため、独立したエンドポイントの方が責務が明確になる
- 将来 `PATCH` で編集提案のタイトル・本文を更新したい場合に、反映・クローズのロジックと競合する

代わりに、反映は `POST /s/{space}/suggestions/{number}/apply`（`suggestion_apply.Handler.Create`）、クローズは `POST /s/{space}/suggestions/{number}/close`（`suggestion_close.Handler.Create`）として独立したリソースに切り出している。

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [ページ編集 仕様書](../page/edit.md)
- [ページの移動 仕様書](../page/move.md)
- [編集提案 作業計画書](/workspace/docs/plans/1_doing/suggestion.md)
- [GitHub: About pull requests](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/about-pull-requests)
