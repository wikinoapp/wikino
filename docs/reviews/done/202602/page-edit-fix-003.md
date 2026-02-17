# コードレビュー: page-edit-fix

## レビュー情報

| 項目                         | 内容                                                      |
| ---------------------------- | --------------------------------------------------------- |
| レビュー日                   | 2026-02-17                                                |
| 対象ブランチ                 | page-edit-fix                                             |
| ベースブランチ               | page-edit                                                 |
| 作業計画書（指定があれば）   | docs/plans/1_doing/page-edit-go-migration.md              |
| 変更ファイル数               | 118 ファイル                                              |
| 変更行数（実装）             | +3107 行                                                  |
| 変更行数（テスト）           | +5351 行                                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/.golangci.yml`
- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/security-guide.md`
- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] `go/db/queries/attachments.sql`
- [ ] `go/db/queries/draft_pages.sql`
- [x] `go/db/queries/page_attachment_references.sql`
- [x] `go/db/queries/page_editors.sql`
- [x] `go/db/queries/page_revisions.sql`
- [x] `go/db/queries/pages.sql`
- [x] `go/db/queries/space_members.sql`
- [x] `go/db/queries/spaces.sql`
- [x] `go/db/queries/topic_members.sql`
- [x] `go/db/queries/topics.sql`
- [x] `go/internal/markup/attachment_extract.go`
- [x] `go/internal/markup/attachment_filter.go`
- [x] `go/internal/markup/batch.go`
- [x] `go/internal/markup/htmlutil.go`
- [x] `go/internal/markup/markup.go`
- [x] `go/internal/markup/wikilink.go`
- [x] `go/internal/model/attachment.go`
- [x] `go/internal/model/draft_page.go`
- [x] `go/internal/model/id.go`
- [x] `go/internal/model/page.go`
- [x] `go/internal/model/page_attachment_reference.go`
- [x] `go/internal/model/page_editor.go`
- [x] `go/internal/model/page_revision.go`
- [x] `go/internal/model/space.go`
- [x] `go/internal/model/space_member.go`
- [x] `go/internal/model/topic.go`
- [x] `go/internal/model/topic_member.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
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
- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/testutil/page_revision_builder.go`
- [x] `go/internal/testutil/space_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/topic_builder.go`
- [x] `go/internal/testutil/topic_member_builder.go`

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

### 設定・その他

- [x] `.claude/commands/review.md`
- [x] `docs/README.md`
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
- [x] `docs/plans/template.md`
- [x] `docs/reviews/done/202602/page-edit-fix-001.md`
- [x] `docs/reviews/done/202602/page-edit-fix-002.md`
- [x] `docs/reviews/template.md`
- [x] `docs/specs/template.md`

## ファイルごとのレビュー結果

### `go/db/queries/draft_pages.sql`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - スペースIDによるクエリスコープ

**問題点・改善提案**:

- **[@go/docs/security-guide.md#スペースIDによるクエリスコープ]**: `FindDraftPageByPageAndMember` クエリに `space_id` が WHERE 条件に含まれていない

  ```sql
  -- 現在のコード
  SELECT * FROM draft_pages WHERE page_id = $1 AND space_member_id = $2;
  ```

  `draft_pages` テーブルは `space_id` カラムを持つスペーススコープのリソースです。セキュリティガイドラインでは、スペース内リソースへのクエリには必ず `space_id` を WHERE 条件に含めることが求められています。

  `page_id` と `space_member_id` の組み合わせにより間接的にスペーススコープが保証されますが、防御的プログラミングの観点から明示的に `space_id` を含めるべきです。同ファイル内の `UpdateDraftPage` や `DeleteDraftPage` は正しく `space_id` を含んでいます。

  **修正案**:

  ```sql
  -- name: FindDraftPageByPageAndMember :one
  -- ページIDとスペースメンバーIDで下書きを取得する
  SELECT * FROM draft_pages WHERE page_id = $1 AND space_member_id = $2 AND space_id = $3;
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->

  - [x] 修正案の通り `space_id` を WHERE 条件に追加する
  - [ ] 現状のまま（間接的にスコープされているため問題ない）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

全体として非常に高品質な実装です。フェーズ1〜3（基盤モデル・リポジトリ、ページモデル・リポジトリ、Markdownレンダリング・マークアップ処理）が作業計画書の設計通りに実装されています。

**良かった点**:

- **ドメインID型の一貫した使用**: すべてのモデルで専用のID型が正しく使用されており、型安全性が確保されている
- **Model-Repository 1:1関係の遵守**: 各モデルに対応するリポジトリが作成され、WithTxパターンも一貫して実装されている
- **Policyパターンの設計**: TopicPolicyのインターフェース+ロール別構造体の設計が優れており、テストも網羅的
- **Markupパイプラインの設計**: インターフェース（AttachmentFinder, PageLocationResolver）を使用した依存性の抽象化が適切で、レイヤー違反を回避している
- **golangci-lintルールの追加**: Policy層とMarkup層の依存関係ルールが正しく追加されている
- **テストカバレッジ**: 全リポジトリと全マークアップ処理に対して包括的なテストが書かれている（テストコード5351行）
- **テストビルダーの一貫性**: 新規ビルダーがすべてドメインID型を返すパターンで統一されている

**修正が必要な点**:

- `FindDraftPageByPageAndMember` クエリに `space_id` を追加する（セキュリティガイドライン準拠）: 1件
