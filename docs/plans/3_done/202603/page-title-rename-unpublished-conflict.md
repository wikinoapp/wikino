# ページタイトルリネーム時の未公開ページ競合解消 作業計画書

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

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 仕様書

<!--
- 既存機能を変更する場合: 変更対象の仕様書へのリンクを記載してください
- 新しい機能の場合: タスク完了後に作成予定の仕様書のパスを記載してください
-->

- [ページ編集 仕様書](../specs/page/edit.md)
- [Wikiリンク 仕様書](../specs/page/wikilink.md)

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

ページ編集画面でページのタイトルをリネームする際、同じタイトルの未公開ページ（`pages.published_at` が NULL）が存在すると「すでに存在しています」というバリデーションエラーが表示される。この未公開ページは Wikiリンク記法の入力中に自動保存で自動作成されたものであり、ユーザーはその存在を認識していない。そのため、なぜそのタイトルが使えないのか理解できない。

例えば、ユーザーが `[[共同編集機能]]` というリンクを入力する途中で `[[共同編集]]` まで書いた段階で自動保存が実行されると、「共同編集」という未公開ページが自動作成される。最終的に「共同編集機能」ページを「共同編集」にリネームしようとすると、自動作成された未公開ページとの競合でエラーになる。

この問題を解決するため、リネーム先のタイトルが未公開かつ本文が空のページと競合する場合は、その競合ページを論理削除（タイトルをランダム値に変更し `discarded_at` を設定）してリネームを許可するように変更する。

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

- ページタイトルをリネームする際、同タイトルの未公開かつ本文が空のページ（`published_at IS NULL AND body = ''`）が存在する場合、システムはその競合ページを論理削除してリネームを許可する
- 同タイトルの公開済みページ（`published_at IS NOT NULL`）が存在する場合は、従来どおり重複エラーを表示する
- 同タイトルの未公開だが本文が空でないページ（`published_at IS NULL AND body != ''`）が存在する場合は、従来どおり重複エラーを表示する（本文消失を防ぐ安全策）
- 競合ページの論理削除は、タイトルをページ自身のID（ULID）に変更し、`discarded_at` に現在日時を設定することで行う

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

- **データ整合性**: 競合ページの論理削除とページ更新は同一トランザクション内で実行し、部分的な状態変更を防ぐ
- **セキュリティ**: 論理削除対象のページは同一スペース内のものに限定する（`space_id` によるスコープ）
- **パフォーマンス**: 論理削除方式により、関連レコードの削除処理をリネーム時に行わず、将来の定期クリーンアップタスクに委ねることでリネーム操作を軽量に保つ

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

### 処理フロー

現在のページ公開フロー（`page.Handler.Update` → `PageUpdateValidator` → `PublishPageUsecase`）を以下のように変更する:

```
[Handler.Update]
  ↓
[PageUpdateValidator.Validate]
  ├─ 競合ページなし → 従来どおり
  ├─ 競合ページが公開済み → 従来どおりエラー表示
  ├─ 競合ページが未公開だが本文あり → 従来どおりエラー表示
  └─ 競合ページが未公開かつ本文が空 → エラーなし、Result に競合ページIDを格納
  ↓
[Handler が競合ページIDを PublishPageInput に設定]
  ↓
[PublishPageUsecase.Execute]（トランザクション内）
  ├─ 競合する未公開ページを論理削除（タイトルをIDに変更 + discarded_at を設定）
  └─ ページを更新（タイトル変更・公開）
```

### 論理削除の方式

競合ページを論理削除する際は、以下の2つの更新を行う:

1. **タイトルの変更**: ページ自身のID（ULID）に変更する。IDはプライマリキーで一意なので、他のページタイトルと競合しない
2. **`discarded_at` の設定**: 現在日時を設定する。既存のクエリは `discarded_at IS NULL` でフィルタしているため、論理削除されたページはユーザーから見えなくなる

```sql
-- 競合ページを論理削除する
UPDATE pages
SET title = id::varchar,
    discarded_at = NOW()
WHERE id = @page_id
  AND space_id = @space_id;
```

### `linked_page_ids` の扱い

論理削除されたページのIDを `linked_page_ids` で参照している他のページ・下書きについては、リネーム時に即座に更新しない。理由:

- 既存のリンク一覧・バックリンク一覧のクエリは `discarded_at IS NULL` でフィルタしているため、論理削除されたページは表示されない
- 参照元ページが次回保存されたときに Wikiリンク再解析が実行され、`linked_page_ids` は自然に修正される（リネーム先のページが同じタイトルを持つため、正しいページIDに解決される）
- リネーム操作を軽量に保つ

### バリデーターの変更

`PageUpdateValidator` の結果に未公開の競合ページ情報を追加する。

```go
// PageUpdateValidatorResult はバリデーションの結果
type PageUpdateValidatorResult struct {
    FormErrors                     *session.FormErrors
    UnpublishedConflictingPageID   *model.PageID  // 未公開かつ本文が空の競合ページID（存在する場合）
}
```

バリデーションロジックの変更:

```go
existingPage, err := v.pageRepo.FindByTopicAndTitle(ctx, input.TopicID, input.Title, input.SpaceID)
if existingPage != nil && existingPage.ID != input.PageID {
    if existingPage.PublishedAt == nil && existingPage.Body == "" {
        // 未公開かつ本文が空のページとの競合 → エラーにせず、結果に格納
        result.UnpublishedConflictingPageID = &existingPage.ID
    } else {
        // 公開済みページ、または未公開だが本文があるページとの競合 → 従来どおりエラー
        editPath := fmt.Sprintf("/s/%s/pages/%d/edit", input.SpaceIdentifier, existingPage.Number)
        errorMsg := templates.T(ctx, "validation_page_title_uniqueness_html")
        formErrors.AddField("title", fmt.Sprintf(errorMsg, editPath))
    }
}
```

### ハンドラーの変更

`page.Handler.Update` で、バリデーション結果に未公開の競合ページIDがある場合、`PublishPageInput` に設定する。

```go
input := usecase.PublishPageInput{
    // 既存フィールド...
    UnpublishedConflictingPageID: validationResult.UnpublishedConflictingPageID,
}
```

### ユースケースの変更

`PublishPageUsecase.Execute` にて、トランザクション内で未公開ページの論理削除を追加する。

```go
// PublishPageInput に新規フィールドを追加
type PublishPageInput struct {
    // 既存フィールド...
    UnpublishedConflictingPageID *model.PageID  // 未公開かつ本文が空の競合ページID（存在する場合）
}
```

ページ更新処理（ステップ8: `pageRepo.Update`）の前に、競合ページの論理削除を実行する:

```go
// 競合する未公開ページが存在する場合、論理削除する
if input.UnpublishedConflictingPageID != nil {
    err := pageRepo.DiscardByID(ctx, *input.UnpublishedConflictingPageID, input.SpaceID, now)
    if err != nil {
        return nil, fmt.Errorf("競合する未公開ページの論理削除に失敗しました: %w", err)
    }
}
```

### 新規SQLクエリ

`go/db/queries/pages.sql` に以下のクエリを追加する:

```sql
-- name: DiscardPageByID :exec
-- 指定ページを論理削除する（タイトルをIDに変更し、discarded_at を設定する）
UPDATE pages
SET title = id::varchar,
    discarded_at = @discarded_at
WHERE id = @id
  AND space_id = @space_id;
```

### Repository メソッド

`PageRepository` に以下のメソッドを追加する:

```go
// DiscardByID は指定ページを論理削除する（タイトルをIDに変更し、discarded_at を設定する）
func (r *PageRepository) DiscardByID(ctx context.Context, pageID model.PageID, spaceID model.SpaceID, discardedAt time.Time) error
```

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### バリデーションクエリで `published_at IS NOT NULL` のページのみを重複チェック対象にする

`FindPageByTopicAndTitle` クエリに `published_at IS NOT NULL` 条件を追加して、未公開ページをそもそも重複チェックの対象外にする方式を検討した。

**不採用の理由**:

- このクエリは Wikiリンクのページ存在確認にも使用されており、条件を変えると Wikiリンク解析に影響する
- バリデーションでは通っても、DB のユニーク制約（`index_pages_on_topic_id_and_title`）で更新が拒否されるため、結局未公開ページの対処が必要になる
- バリデーターのレベルで判定し、ユースケースで論理削除する現在の設計のほうが、関心の分離が明確

### 未公開ページのタイトルを NULL に変更する

競合する未公開ページのタイトルを NULL に変更して、ユニーク制約を回避する方式を検討した。

**不採用の理由**:

- タイトルが NULL のページが残存すると、ページ一覧やリンク解析で意図しない振る舞いを引き起こす可能性がある
- 空のタイトルのページが蓄積されていくため、データのクリーンアップが別途必要になる
- ページ自身のIDをタイトルに設定するほうが、一意性が保証され安全

### 未公開ページをハードデリートし、関連レコードも即座に削除する

競合する未公開ページを物理削除（ハードデリート）し、外部キー制約に従って子レコード（`page_editors`, `draft_pages`, `draft_page_revisions`, `page_attachment_references`, `page_revisions`, `suggestion_pages`, `suggestion_page_revisions`）も同時に削除する方式を検討した。

**不採用の理由**:

- 多数のテーブルにまたがる削除処理をリネーム操作のクリティカルパスで実行するため、パフォーマンスが低下する
- 各テーブルの DELETE クエリと Repository メソッドを多数追加する必要があり、実装が複雑になる
- 論理削除方式（タイトル変更 + `discarded_at` 設定）であれば、1つの UPDATE クエリで完結し、関連レコードの物理削除は将来の定期クリーンアップタスクに委ねられる

### linked_page_ids を即座に更新する

論理削除されたページIDを参照している `linked_page_ids` を、リネーム先ページのIDに即座に置換する方式を検討した。

**不採用の理由**:

- 既存のリンク一覧・バックリンク一覧のクエリは `discarded_at IS NULL` でフィルタしているため、論理削除されたページは表示されず、実用上の問題は発生しない
- 参照元ページが次回保存されたときに Wikiリンク再解析により自然に修正される
- リネーム操作を軽量に保つ方針と整合する

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

フィーチャーフラグ:
- 新機能の開発ではフィーチャーフラグによる制御を検討してください
- フラグで制御することで、実装途中でも develop ブランチにマージできます
- フラグのセットアップ（フラグ名の定義、ルーティングパターンの追加など）をフェーズ1に含めてください
- 機能が安定した後のフラグ削除タスクも計画に含めてください
- 詳細は CLAUDE.md の「フィーチャーフラグによる開発」セクションを参照

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

### フェーズ 1: 実装

<!--
例: インフラ準備、基本機能実装、セキュリティ機能など
各タスクは1つのPull Requestで完結する粒度で記述してください
各タスクには想定サイズを明記してください
Go版/Rails版の両方を修正する場合は別タスクに分けてください
-->

- [x] **1-1**: [Go] 未公開ページ競合時の論理削除とバリデーション・ユースケースの変更
  - `go/db/queries/pages.sql` に `DiscardPageByID` クエリを追加
  - sqlc generate を実行
  - `internal/repository/page.go` に `DiscardByID` メソッドを追加
  - `internal/validator/page.go`: `PageUpdateValidatorResult` に `UnpublishedConflictingPageID` フィールドを追加し、未公開かつ本文が空のページとの競合時にエラーにしないロジックに変更
  - `internal/handler/page/update.go`: バリデーション結果の未公開競合ページIDを `PublishPageInput` に渡す
  - `internal/usecase/publish_page.go`: `PublishPageInput` に `UnpublishedConflictingPageID` フィールドを追加し、トランザクション内で競合ページの論理削除を実行
  - **想定ファイル数**: 約 7 ファイル（実装 5 + テスト 2）
  - **想定行数**: 約 200 行（実装 60 行 + テスト 140 行）

### フェーズ 2: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [x] **2-1**: 仕様書の更新
  - `docs/specs/page/edit.md` にタイトルリネーム時の未公開ページ競合解消の仕様を追記する
  - `docs/specs/page/wikilink.md` に未公開ページが他ページのリネーム時に論理削除される可能性があることを追記する
  - 作業計画書の採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **論理削除されたページの物理削除定期タスク**: `discarded_at` が設定されたページとその関連レコードを定期的に物理削除するタスク。本タスクで論理削除されたページが蓄積されるため、将来的に実装が必要
- **自動保存時の孤立した未公開ページのクリーンアップ**: Wikiリンク記法の入力中に作成された不要な未公開ページを自動保存のタイミングで削除する機能。現在の問題（リネーム時の競合）とは別の改善であり、影響範囲が広いため別タスクで検討する
- **DBユニーク制約への `WHERE published_at IS NOT NULL` の追加（partial unique index）**: ユニーク制約を公開済みページのみに限定する方式。DBスキーマの変更が必要であり、未公開ページ同士の同タイトルを許可するかどうかの設計判断を含むため、別途検討する

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [ページ編集 仕様書](/workspace/docs/specs/page/edit.md)
- [Wikiリンク 仕様書](/workspace/docs/specs/page/wikilink.md)
- [Go版 バリデーター](/workspace/go/internal/validator/page.go)
- [Go版 ページ公開ユースケース](/workspace/go/internal/usecase/publish_page.go)
- [Go版 ページ更新ハンドラー](/workspace/go/internal/handler/page/update.go)
