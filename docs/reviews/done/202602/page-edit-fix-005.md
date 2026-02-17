# コードレビュー: page-edit-fix

## レビュー情報

| 項目                         | 内容                                                 |
| ---------------------------- | ---------------------------------------------------- |
| レビュー日                   | 2026-02-17                                           |
| 対象ブランチ                 | page-edit-fix                                        |
| ベースブランチ               | page-edit                                            |
| 作業計画書（指定があれば）   | docs/plans/1_doing/page-edit-go-migration.md         |
| 変更ファイル数               | 120 ファイル                                         |
| 変更行数（実装）             | +9,619 行                                            |
| 変更行数（テスト）           | +5,351 行                                            |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

## 変更ファイル一覧

### モデル

- [x] `go/internal/model/id.go`
- [x] `go/internal/model/space.go`
- [x] `go/internal/model/space_member.go`
- [x] `go/internal/model/topic.go`
- [x] `go/internal/model/topic_member.go`
- [x] `go/internal/model/page.go`
- [x] `go/internal/model/draft_page.go`
- [x] `go/internal/model/page_revision.go`
- [x] `go/internal/model/page_editor.go`
- [x] `go/internal/model/page_attachment_reference.go`
- [x] `go/internal/model/attachment.go`

### SQLクエリ

- [x] `go/db/queries/spaces.sql`
- [x] `go/db/queries/space_members.sql`
- [x] `go/db/queries/topics.sql`
- [x] `go/db/queries/topic_members.sql`
- [x] `go/db/queries/pages.sql`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/db/queries/page_revisions.sql`
- [x] `go/db/queries/page_editors.sql`
- [x] `go/db/queries/page_attachment_references.sql`
- [x] `go/db/queries/attachments.sql`

### sqlc生成コード（自動生成のためレビュー対象外）

- [x] `go/internal/query/spaces.sql.go`
- [x] `go/internal/query/space_members.sql.go`
- [x] `go/internal/query/topics.sql.go`
- [x] `go/internal/query/topic_members.sql.go`
- [x] `go/internal/query/pages.sql.go`
- [x] `go/internal/query/draft_pages.sql.go`
- [x] `go/internal/query/page_revisions.sql.go`
- [x] `go/internal/query/page_editors.sql.go`
- [x] `go/internal/query/page_attachment_references.sql.go`
- [x] `go/internal/query/topics.sql.go`
- [x] `go/internal/query/attachments.sql.go`

### リポジトリ

- [x] `go/internal/repository/space.go`
- [x] `go/internal/repository/space_member.go`
- [x] `go/internal/repository/topic.go`
- [x] `go/internal/repository/topic_member.go`
- [x] `go/internal/repository/page.go`
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/repository/page_revision.go`
- [x] `go/internal/repository/page_editor.go`
- [x] `go/internal/repository/page_attachment_reference.go`
- [x] `go/internal/repository/attachment.go`

### ポリシー

- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_guest.go`

### マークアップ

- [x] `go/internal/markup/markup.go`
- [x] `go/internal/markup/wikilink.go`
- [x] `go/internal/markup/attachment_filter.go`
- [x] `go/internal/markup/attachment_extract.go`
- [x] `go/internal/markup/batch.go`
- [x] `go/internal/markup/htmlutil.go`

### テストファイル

- [x] `go/internal/repository/space_test.go`
- [x] `go/internal/repository/space_member_test.go`
- [x] `go/internal/repository/topic_test.go`
- [x] `go/internal/repository/topic_member_test.go`
- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/repository/draft_page_test.go`
- [x] `go/internal/repository/page_revision_test.go`
- [x] `go/internal/repository/page_editor_test.go`
- [x] `go/internal/repository/page_attachment_reference_test.go`
- [x] `go/internal/repository/attachment_test.go`
- [x] `go/internal/policy/topic_test.go`
- [x] `go/internal/markup/markup_test.go`
- [x] `go/internal/markup/wikilink_test.go`
- [x] `go/internal/markup/attachment_filter_test.go`
- [x] `go/internal/markup/attachment_extract_test.go`
- [x] `go/internal/markup/batch_test.go`
- [x] `go/internal/markup/pipeline_integration_test.go`

### テストユーティリティ

- [x] `go/internal/testutil/space_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/topic_builder.go`
- [x] `go/internal/testutil/topic_member_builder.go`
- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/testutil/page_revision_builder.go`
- [x] `go/internal/testutil/draft_page_builder.go`

### 設定・ドキュメント

- [x] `go/.golangci.yml`
- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/security-guide.md`
- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] `docs/README.md`
- [x] `docs/plans/template.md`
- [x] `docs/reviews/template.md`
- [x] `docs/specs/template.md`
- [x] その他ドキュメントファイル（計画書・レビュー関連）

## ファイルごとのレビュー結果

### `go/.golangci.yml`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#golangci-lintの使い方](/workspace/go/CLAUDE.md) - depguardのアーキテクチャルール
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャの依存関係ルール

**問題点・改善提案**:

- **[@go/CLAUDE.md#golangci-lintの使い方]**: `format_validator.go`/`state_validator.go`のパターンがバリデーションガイドと不一致

  バリデーションガイド（`validation-guide.md`）では`validator.go`に統合する方式を採用しているが、depguardの`format-validator-layer`（`**/format_validator.go`）と`state-validator-layer`（`**/state_validator.go`）は分離された命名パターンを期待している。現在の実装にはこのパターンに一致するファイルが存在しないため、ルール自体は無害だが、ガイドラインとの不整合がある。

  **修正案**:

  depguardのバリデータールールのファイルパターンを現在の命名規約に合わせて`**/validator.go`に変更するか、不要なルールを削除する。

  **対応方針**:

  - [x] `format-validator-layer`と`state-validator-layer`を`validator-layer`に統合し、パターンを`**/validator.go`に変更する
  - [ ] 現状のまま（将来的にバリデーター分離を検討する可能性があるため残す）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 設計との整合性チェック

作業計画書（`docs/plans/1_doing/page-edit-go-migration.md`）との整合性を確認した。

### 実装済みフェーズ

作業計画書のタスクリストで`[x]`になっているフェーズ1〜2b（基盤モデル・リポジトリ、ページモデル・リポジトリ、TopicPolicy、スペースIDスコープ強化）が本diffの主要な変更内容であり、以下の通り整合している:

| タスク | 計画 | 実装 | 整合性 |
|--------|------|------|--------|
| 1-1: Space モデル・クエリ・リポジトリ | ✅ | ✅ | 一致 |
| 1-2: SpaceMember モデル・クエリ・リポジトリ | ✅ | ✅ | 一致 |
| 1-3: Topic モデル・クエリ・リポジトリ | ✅ | ✅ | 一致 |
| 1-4: TopicMember モデル・クエリ・リポジトリ | ✅ | ✅ | 一致 |
| 2-1: Page モデル・クエリ・リポジトリ | ✅ | ✅ | 一致 |
| 2-2: DraftPage モデル・クエリ・リポジトリ | ✅ | ✅ | 一致 |
| 2-2.5: TopicPolicy | ✅ | ✅ | 一致 |
| 2-3: PageRevision モデル・クエリ・リポジトリ | ✅ | ✅ | 一致 |
| 2-4: PageEditor モデル・クエリ・リポジトリ | ✅ | ✅ | 一致 |
| 2-5: PageAttachmentReference モデル・クエリ・リポジトリ | ✅ | ✅ | 一致 |
| 2-6: Attachment モデル・クエリ・リポジトリ | ✅ | ✅ | 一致 |
| 2b-1: スペースIDスコープ強化 | ✅ | ✅ | 一致 |
| マークアップパッケージ | ✅ | ✅ | 一致 |

### セキュリティ要件

- **space_idスコープ**: 全SQLクエリで適切にspace_idが条件に含まれている ✅
- **ドメインID型**: 全モデルで専用のドメインID型が使用されている ✅
- **TopicPolicy**: インターフェース + ロール別構造体パターンが計画通り実装されている ✅

### 設計との乖離

なし。実装は作業計画書の設計に忠実に従っている。

## 総合評価

**評価**: Approve

**総評**:

本diffは、ページ編集Go移行の基盤となるモデル・リポジトリ・ポリシー・マークアップパッケージの実装であり、非常に高品質なコードである。

**良かった点**:

- **セキュリティ**: 全SQLクエリにspace_idスコープが適用されており、防御的プログラミングが徹底されている
- **ドメインID型**: 全モデルで専用のドメインID型が使用され、型安全性が確保されている
- **アーキテクチャ**: 3層アーキテクチャの依存関係ルールが正しく守られている。Repository→Query→Modelの依存方向が一貫している
- **テスト**: TestMainパターン、トランザクション分離、ビルダーパターン、テーブル駆動テストが全テストで一貫して使用されている
- **TopicPolicy**: インターフェース + ファクトリパターンが計画通りに実装され、純粋ロジック（DBアクセスなし）が守られている
- **マークアップ**: XSS対策（bluemonday）、Wikiリンクの安全な処理（URLエスケープ、タグ内スキップ）、添付ファイルフィルターが適切に実装されている
- **作業計画書との整合性**: 全タスクが計画通りに実装されており、乖離がない

**指摘事項**:

- `.golangci.yml`のdepguardルールでバリデーターのファイルパターンがガイドラインと不整合（軽微、実害なし）
