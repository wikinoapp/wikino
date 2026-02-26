# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                                          |
| -------------------------- | ------------------------------------------------------------- |
| レビュー日                 | 2026-02-17                                                    |
| 対象ブランチ               | page-edit-fix                                                 |
| ベースブランチ             | page-edit                                                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md                  |
| 変更ファイル数             | 126 ファイル                                                  |
| 変更行数（実装）           | +3,259 行（Go実装40ファイル）/ +172 行（SQLクエリ10ファイル） |
| 変更行数（テスト）         | +5,373 行（17ファイル）                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル（モデル）

- [x] `go/internal/model/id.go`
- [x] `go/internal/model/attachment.go`
- [x] `go/internal/model/draft_page.go`
- [x] `go/internal/model/page.go`
- [x] `go/internal/model/page_attachment_reference.go`
- [x] `go/internal/model/page_editor.go`
- [x] `go/internal/model/page_revision.go`
- [x] `go/internal/model/space.go`
- [x] `go/internal/model/space_member.go`
- [x] `go/internal/model/topic.go`
- [x] `go/internal/model/topic_member.go`

### 実装ファイル（ポリシー）

- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`

### 実装ファイル（リポジトリ）

- [x] `go/internal/repository/attachment.go`
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/repository/page.go`
- [x] `go/internal/repository/page_attachment_reference.go`
- [x] `go/internal/repository/page_editor.go`
- [x] `go/internal/repository/page_revision.go`
- [x] `go/internal/repository/space.go`
- [x] `go/internal/repository/space_member.go`
- [x] `go/internal/repository/topic.go`
- [x] `go/internal/repository/topic_member.go`

### 実装ファイル（マークアップ）

- [x] `go/internal/markup/attachment_extract.go`
- [x] `go/internal/markup/attachment_filter.go`
- [x] `go/internal/markup/batch.go`
- [x] `go/internal/markup/htmlutil.go`
- [x] `go/internal/markup/markup.go`
- [x] `go/internal/markup/wikilink.go`

### 実装ファイル（テストユーティリティ）

- [x] `go/internal/testutil/attachment_builder.go`
- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/testutil/page_revision_builder.go`
- [x] `go/internal/testutil/space_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/topic_builder.go`
- [x] `go/internal/testutil/topic_member_builder.go`

### SQLクエリファイル

- [x] `go/db/queries/attachments.sql`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/db/queries/page_attachment_references.sql`
- [x] `go/db/queries/page_editors.sql`
- [x] `go/db/queries/page_revisions.sql`
- [x] `go/db/queries/pages.sql`
- [x] `go/db/queries/space_members.sql`
- [x] `go/db/queries/spaces.sql`
- [x] `go/db/queries/topic_members.sql`
- [x] `go/db/queries/topics.sql`

### テストファイル

- [x] `go/internal/markup/attachment_extract_test.go`
- [x] `go/internal/markup/attachment_filter_test.go`
- [x] `go/internal/markup/batch_test.go`
- [x] `go/internal/markup/markup_test.go`
- [x] `go/internal/markup/pipeline_integration_test.go`
- [x] `go/internal/markup/wikilink_test.go`
- [x] `go/internal/policy/topic_test.go`
- [x] `go/internal/repository/attachment_test.go`
- [x] `go/internal/repository/draft_page_test.go`
- [x] `go/internal/repository/page_attachment_reference_test.go`
- [x] `go/internal/repository/page_editor_test.go`
- [x] `go/internal/repository/page_revision_test.go`
- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/repository/space_member_test.go`
- [x] `go/internal/repository/space_test.go`
- [x] `go/internal/repository/topic_member_test.go`
- [x] `go/internal/repository/topic_test.go`

### 自動生成ファイル（sqlc）

- [x] `go/internal/query/attachments.sql.go`
- [x] `go/internal/query/draft_pages.sql.go`
- [x] `go/internal/query/page_attachment_references.sql.go`
- [x] `go/internal/query/page_editors.sql.go`
- [x] `go/internal/query/page_revisions.sql.go`
- [x] `go/internal/query/pages.sql.go`
- [x] `go/internal/query/space_members.sql.go`
- [x] `go/internal/query/spaces.sql.go`
- [x] `go/internal/query/topic_members.sql.go`
- [x] `go/internal/query/topics.sql.go`

### 設定・ドキュメント

- [x] `go/.golangci.yml`
- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/security-guide.md`
- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] `.claude/commands/review.md`
- [x] `docs/README.md`
- [x] `docs/plans/template.md`
- [x] `docs/reviews/template.md`
- [x] `docs/specs/template.md`
- [x] `docs/plans/1_doing/edit-suggestion.md`
- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/2_todo/diff-component.md`
- [x] `docs/plans/2_todo/domain-id-types-remaining.md`
- [x] `docs/plans/2_todo/draft-page-discard.md`
- [x] `docs/plans/2_todo/draft-page-revision.md`
- [x] `docs/plans/2_todo/draft-screens.md`
- [x] `docs/plans/2_todo/go-rate-limit-advisory-lock.md`
- [x] `docs/plans/2_todo/page-move.md`
- [x] `docs/plans/2_todo/page-ogp-meta.md`
- [x] `docs/plans/2_todo/page-revision-history.md`
- [x] `docs/plans/2_todo/page-show-go-migration.md`
- [x] `docs/plans/2_todo/publish-diff-confirmation.md`
- [x] `docs/plans/2_todo/title-change-link-rewrite.md`
- [x] `docs/plans/3_done/202508/file-attachment.md`
- [x] `docs/plans/3_done/202508/page-thumbnail-and-og-image.md`
- [x] `docs/plans/3_done/202508/permission-improvement.md`
- [x] `docs/plans/3_done/202509/fix-types.md`
- [x] `docs/plans/3_done/202509/topic-links.md`
- [x] `docs/plans/3_done/202512/guideline-violations-fix.md`
- [x] `docs/plans/3_done/202602/go-maintenance-mode.md`
- [x] `docs/plans/3_done/202602/go-password-reset.md`
- [x] `docs/plans/3_done/202602/go-rate-limit-repository.md`
- [x] `docs/plans/3_done/202602/go-sign-up.md`
- [x] `docs/plans/3_done/202602/go-welcome.md`
- [x] `docs/plans/3_done/202602/go.md`
- [x] `docs/plans/3_done/202602/postgresql-18-upgrade.md`
- [x] `docs/plans/3_done/202602/rails-cleanup-go-migrated-endpoints.md`
- [x] `docs/plans/3_done/202602/unified-dev-container.md`
- [x] `docs/reviews/done/202602/page-edit-fix-001.md`
- [x] `docs/reviews/done/202602/page-edit-fix-002.md`
- [x] `docs/reviews/done/202602/page-edit-fix-003.md`
- [x] `docs/reviews/done/202602/page-edit-fix-004.md`
- [x] `docs/reviews/done/202602/page-edit-fix-005.md`
- [x] `docs/reviews/done/202602/page-edit-fix-006.md`
- [x] `docs/reviews/done/202602/page-edit-fix-007.md`
- [x] `docs/reviews/done/202602/page-edit-fix-008.md`
- [x] `docs/reviews/done/202602/page-edit-fix-009.md`

## ファイルごとのレビュー結果

問題のあるファイルはありませんでした。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

ページ編集機能のGo移行に向けた基盤実装として、モデル・リポジトリ・ポリシー・マークアップパイプラインが包括的かつ高品質に実装されています。以下の点が特に優れています。

**良かった点**:

1. **セキュリティ**: 全SQLクエリで `space_id` によるスコープが徹底されており、防御的プログラミングが実践されています。マークアップパイプラインではbluemonday sanitizerによるXSS対策、`html.RawNode`を活用したSVG安全注入など、セキュリティに配慮した実装です。

2. **アーキテクチャ準拠**: 3層アーキテクチャの依存関係ルールが厳守されています。Model→Repository→Queryの1:1関係、ドメインID型の一貫した使用、WithTxパターンによるトランザクション対応など、ガイドラインに忠実な実装です。

3. **TopicPolicyの設計**: インターフェース + ファクトリ + ロール別具象実装のパターンが作業計画書の設計通りに実装されており、TopicPolicyインターフェースに`model.SpaceID`を使用するなど、計画を超える改善も含まれています。

4. **マークアップパイプライン**: goldmark（Markdown→HTML変換）→ bluemonday（サニタイズ）→ wikilink変換 → attachment filter の多段パイプラインが正確に実装されています。ウィキリンク `[[page]]` / `[[topic/page]]` のパース、添付ファイルの種別判定（画像/動画/ダウンロード）が仕様通りです。

5. **テストカバレッジ**: 実装3,259行に対してテスト5,373行と、テストが実装を大きく上回る充実した品質保証です。全テストパッケージでTestMain/SetupTxパターン、t.Parallel()、ビルダーパターンが一貫して使用されています。

6. **作業計画書との整合性**: 作業計画書に記載されたデータモデル（Page, PageRevision, DraftPage, PageAttachmentReference, TopicMember）、リポジトリ、ポリシー、マークアップ処理がすべて仕様通りに実装されています。
