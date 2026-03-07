# ページ編集画面Go版の全ユーザー展開とRails版削除 作業計画書

<!--
このテンプレートの使い方:
1. このファイルを `docs/plans/2_todo/` ディレクトリにコピー
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

- [ページ編集 仕様書](../specs/page/edit.md)

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

[ページ編集画面のGo移行](../1_doing/page-edit-go-migration.md) で実装したGo版ページ編集画面を全ユーザーに展開し、不要になったRails版の実装を削除する。

現在はフィーチャーフラグによる段階的ロールアウトでGo版を提供しているが、問題がないことを確認した後にフィーチャーフラグからホワイトリスト方式に切り替えて全ユーザーがGo版を使用するようにする。その後、Rails版のページ編集関連コードを削除してコードベースを整理する。

### 前提

- [ページ編集画面のGo移行](../1_doing/page-edit-go-migration.md) が完了し、フィーチャーフラグによるロールアウトで問題がないことが確認済みであること

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

- 全ユーザーがGo版のページ編集画面を使用する（フィーチャーフラグ不要）
- Rails版のページ編集関連コードが削除され、コードベースが整理される

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

### Rails版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - 全体的なコーディング規約
- [@rails/docs/architecture-guide.md](/workspace/rails/docs/architecture-guide.md) - アーキテクチャガイド（クラス設計と依存関係、サービスクラスのルール）
- [@rails/docs/testing-guide.md](/workspace/rails/docs/testing-guide.md) - テストガイド（RSpec のコーディング規約）
- [@rails/docs/security-guide.md](/workspace/rails/docs/security-guide.md) - セキュリティガイドライン

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

### フィーチャーフラグからホワイトリストへの移行

- `internal/middleware/reverse_proxy.go` の `featureFlaggedPatterns` からページ編集関連のパターンを削除
- `internal/middleware/reverse_proxy.go` に `goHandledRegexPatterns`（正規表現 + メソッドフィルタによるホワイトリスト）を追加し、ページ編集関連のパスを登録（全ユーザーがGo版を使用）
- `internal/middleware/reverse_proxy.go` の `goHandledPrefixPaths` に `/drafts`（下書き一覧）を追加
- `/s/` をプレフィックス一致に追加するとGo未実装のページ表示（`GET /s/:space_id/pages/:page_id`）もGoに送られてしまうため、正規表現パターンで個別に指定する

### フィーチャーフラグの整理

フィーチャーフラグの仕組み自体は今後の機能移行でも使用するため残す。ページ編集用のフラグのみ削除し、仕組みが壊れないようダミーのフラグ定数を追加する。

- `internal/model/feature_flag.go` の `FeatureFlagGoPageEdit` 定数を削除し、ダミーの `FeatureFlagExample` 定数を追加（仕組みの参照用）
- `internal/middleware/reverse_proxy.go` の `featureFlaggedPatterns` からページ編集関連のパターンをすべて削除（スライスは空にする）
- `e2e/tests/auth.setup.ts` から `createTestFeatureFlag` 呼び出しを削除（ページ編集がホワイトリストに移行するため不要）

### Rails版バックリンクパスの競合解消

Go版ホワイトリストに `^/s/[^/]+/pages/\d+/backlinks$` が全メソッド対象で登録されているため、Rails版ページ表示画面（`GET /s/:space_identifier/pages/:page_number` はRails処理）から `BacklinkListComponent` が発行するバックリンクページネーションリクエスト（`POST /s/:space_identifier/pages/:page_number/backlinks`）がGoに転送されてしまう。

- `config/routes.rb` の `page_backlink_list` ルートのパスを `/rails/s/:space_identifier/pages/:page_number/backlinks` に変更する
- `BacklinkListComponent` 等は `page_backlink_list_path` ヘルパーを使用しているため、ルート変更に自動的に追従する
- Go版リバースプロキシは `/rails/` プレフィックスをホワイトリストに持たないため、自動的にRailsに転送される

### Rails版の削除対象

- コントローラー: `app/controllers/pages/edit_controller.rb`, `app/controllers/pages/update_controller.rb`, `app/controllers/draft_pages/update_controller.rb`, `app/controllers/page_locations/index_controller.rb`
- サービス: `app/services/pages/update_service.rb`, `app/services/draft_pages/update_service.rb`
- フォーム: `app/forms/pages/edit_form.rb`
- ビュー: `app/views/pages/edit_view.html.erb`, `app/views/draft_pages/update_view.html.erb`
- 関連するルーティング・テスト・翻訳

### 削除しないもの

- レコード（`PageRecord`, `DraftPageRecord`）やPageable concernは他機能で使用されているため削除しない
- ページ表示関連（`ShowController`等）の削除は [ページ表示画面のGo移行](page-show-go-migration.md) で行う
- フィーチャーフラグの仕組み（`featureFlagChecker` インターフェース、`FeatureFlagRepository`、`featureFlaggedPattern` 構造体、関連メソッド、テストビルダー、E2Eヘルパー等）は今後の機能移行で再利用するため削除しない

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### `/s/` プレフィックスによるホワイトリスト登録

`goHandledPrefixPaths` に `/s/` を追加してスペース配下の全パスをGo版に送る方法を検討した。

**不採用の理由**: `/s/:space_id/pages/:page_id` のGETリクエスト（ページ表示）はまだGo版で実装されていないため、`/s/` をプレフィックスに追加するとページ表示が壊れる。また、`PATCH /s/:space_id/pages/:page_id`（ページ更新）はGoで処理するがGETはRailsに転送する必要があり、プレフィックス一致ではメソッドの区別ができない。そのため、正規表現 + メソッドフィルタによる `goHandledRegexPatterns` を導入した。

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

### フェーズ 0: バグ修正

- [x] **0-1**: [Go] ページ公開時に下書きリビジョンを削除してから下書きを削除するよう修正
  - **問題**: `PATCH /s/:space_id/pages/:page_id` でページを公開する際、`draft_page_revisions` テーブルが `draft_pages.id` を外部キー（`ON DELETE CASCADE` なし）で参照しているため、リビジョンが存在する下書きの削除が外部キー制約違反で失敗する
  - **原因**: `internal/usecase/publish_page.go` のステップ12で `draftPageRepo.Delete` を呼ぶ前に、関連する `draft_page_revisions` を削除していない
  - **対応**: `PublishPageUsecase` の下書き削除（ステップ12）の前に、`draft_page_revisions` の削除処理を追加する
    - `db/queries/draft_page_revisions.sql` に `DeleteDraftPageRevisionsByDraftPageID` クエリを追加
    - `internal/repository/` に DraftPageRevisionRepository を作成（または既存のリポジトリに追加）し、削除メソッドを実装
    - `internal/usecase/publish_page.go` に DraftPageRevisionRepository を注入し、下書き削除前にリビジョンを削除
  - **想定ファイル数**: 約 5 ファイル（実装 4 + テスト 1）
  - **想定行数**: 約 80 行（実装 ~50 行 + テスト ~30 行）

### フェーズ 1: フィーチャーフラグからホワイトリストへの移行（全ユーザー展開）

- [x] **1-1**: [Go] ページ編集パスをホワイトリストに追加し、フィーチャーフラグを整理
  - フィーチャーフラグによる段階的ロールアウトで問題がないことを確認した後に実施
  - `internal/middleware/reverse_proxy.go` に `goHandledRegexPatterns`（正規表現 + メソッドフィルタによるホワイトリスト）を追加し、ページ編集関連パスを登録
  - `internal/middleware/reverse_proxy.go` の `goHandledPrefixPaths` に `/drafts` を追加
  - `internal/middleware/reverse_proxy.go` の `featureFlaggedPatterns` からページ編集関連のパターンをすべて削除（スライスは空にする）
  - `internal/model/feature_flag.go` の `FeatureFlagGoPageEdit` 定数を削除し、ダミーの `FeatureFlagExample` 定数を追加
  - `e2e/tests/auth.setup.ts` から `createTestFeatureFlag` 呼び出しを削除
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 -50 行（パターン削除分）

### フェーズ 1a: Rails版バックリンクパスの競合解消

- [x] **1a-1**: [Rails] バックリンク一覧のパスに `/rails` プレフィックスを追加
  - Go版ホワイトリストに `^/s/[^/]+/pages/\d+/backlinks$` が登録されているため、Rails版ページ表示画面からのバックリンクページネーション（`POST /s/:space_identifier/pages/:page_number/backlinks`）がGoに転送されてしまう問題を解消する
  - `config/routes.rb` の `page_backlink_list` ルートのパスを `/rails/s/:space_identifier/pages/:page_number/backlinks` に変更
  - `BacklinkListComponent` が生成するパスは `page_backlink_list_path` ヘルパー経由のため、ルート変更で自動的に追従する
  - Go版リバースプロキシは `/rails/` プレフィックスのパスをホワイトリストに持たないため、自動的にRailsに転送される
  - **想定ファイル数**: 約 1 ファイル（実装 1）
  - **想定行数**: 約 5 行（ルート定義の変更のみ）
  - 依存: 1-1

### フェーズ 2: Rails版の実装の削除

- [ ] **2-1**: [Rails] ページ編集・公開関連のコントローラー・サービス・フォーム・ビューの削除
  - `app/controllers/pages/edit_controller.rb`, `app/controllers/pages/update_controller.rb` を削除
  - `app/controllers/draft_pages/update_controller.rb` を削除
  - `app/controllers/page_locations/index_controller.rb` を削除
  - `app/services/pages/update_service.rb`, `app/services/draft_pages/update_service.rb` を削除
  - `app/forms/pages/edit_form.rb` を削除
  - `app/views/pages/edit_view.html.erb`, `app/views/draft_pages/update_view.html.erb` を削除
  - 関連するルーティング・テスト・翻訳を削除
  - レコード（`PageRecord`, `DraftPageRecord`）やPageable concernは他機能で使用されているため削除しない
  - ページ表示関連（`ShowController`等）の削除は [ページ表示画面のGo移行](page-show-go-migration.md) で行う
  - **想定ファイル数**: 約 11 ファイル（実装 8 ファイル削除 + テスト 3 ファイル削除）
  - **想定行数**: 約 -800 行（実装 ~-500 行 + テスト ~-300 行）
  - 依存: 1-1

### フェーズ N: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [ ] **N-1**: 仕様書の更新
  - `docs/specs/page/edit.md` のページ編集仕様書を更新する（Go版への完全移行を反映）

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **ページ表示関連のRails版削除**: [ページ表示画面のGo移行](page-show-go-migration.md) で対応する
- **レコード・concernの削除**: `PageRecord`, `DraftPageRecord`, Pageable concernは他機能で使用されているため削除しない

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [ページ編集画面のGo移行 作業計画書](../1_doing/page-edit-go-migration.md)
