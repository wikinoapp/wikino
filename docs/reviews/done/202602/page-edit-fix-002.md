# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                         |
| -------------------------- | -------------------------------------------- |
| レビュー日                 | 2026-02-17                                   |
| 対象ブランチ               | page-edit-fix                                |
| ベースブランチ             | page-edit                                    |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md |
| 変更ファイル数             | 117 ファイル                                 |
| 変更行数（実装）           | +4583 / -12 行                               |
| 変更行数（テスト）         | +5351 / -0 行                                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/model/id.go`
- [x] `go/internal/model/attachment.go`
- [x] `go/internal/model/draft_page.go`
- [ ] `go/internal/model/page.go`
- [x] `go/internal/model/page_attachment_reference.go`
- [x] `go/internal/model/page_editor.go`
- [x] `go/internal/model/page_revision.go`
- [x] `go/internal/model/space.go`
- [x] `go/internal/model/space_member.go`
- [x] `go/internal/model/topic.go`
- [x] `go/internal/model/topic_member.go`
- [x] `go/db/queries/attachments.sql`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/db/queries/page_attachment_references.sql`
- [ ] `go/db/queries/page_editors.sql`
- [x] `go/db/queries/page_revisions.sql`
- [x] `go/db/queries/pages.sql`
- [x] `go/db/queries/space_members.sql`
- [x] `go/db/queries/spaces.sql`
- [ ] `go/db/queries/topic_members.sql`
- [x] `go/db/queries/topics.sql`
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
- [ ] `go/internal/repository/topic_member.go`
- [x] `go/internal/markup/markup.go`
- [x] `go/internal/markup/wikilink.go`
- [x] `go/internal/markup/attachment_extract.go`
- [x] `go/internal/markup/attachment_filter.go`
- [x] `go/internal/markup/batch.go`
- [x] `go/internal/markup/htmlutil.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/testutil/page_revision_builder.go`
- [x] `go/internal/testutil/space_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/topic_builder.go`
- [x] `go/internal/testutil/topic_member_builder.go`

### テストファイル

- [x] `go/internal/markup/markup_test.go`
- [x] `go/internal/markup/wikilink_test.go`
- [x] `go/internal/markup/attachment_extract_test.go`
- [x] `go/internal/markup/attachment_filter_test.go`
- [x] `go/internal/markup/batch_test.go`
- [x] `go/internal/markup/pipeline_integration_test.go`
- [x] `go/internal/policy/topic_test.go`
- [x] `go/internal/repository/attachment_test.go`
- [x] `go/internal/repository/draft_page_test.go`
- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/repository/page_attachment_reference_test.go`
- [x] `go/internal/repository/page_editor_test.go`
- [x] `go/internal/repository/page_revision_test.go`
- [x] `go/internal/repository/space_test.go`
- [x] `go/internal/repository/space_member_test.go`
- [x] `go/internal/repository/topic_test.go`
- [x] `go/internal/repository/topic_member_test.go`

### 設定・その他

- [x] `go/.golangci.yml`
- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/security-guide.md`
- [x] `docs/README.md`
- [x] `docs/reviews/template.md`
- [x] `docs/plans/template.md`
- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/2_todo/diff-component.md`
- [x] `docs/plans/2_todo/domain-id-types-remaining.md`
- [x] `docs/plans/2_todo/draft-page-discard.md`
- [x] `docs/plans/2_todo/draft-page-revision.md`
- [x] `docs/plans/2_todo/draft-screens.md`
- [x] `docs/plans/2_todo/edit-suggestion.md`
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
- [x] `docs/specs/template.md`
- [x] `.claude/commands/review.md`

## ファイルごとのレビュー結果

### `go/db/queries/topic_members.sql`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - space_id によるクエリスコープ

**問題点・改善提案**:

- **[@go/docs/security-guide.md#スペースIDによるクエリスコープ]**: `FindTopicMemberBySpaceMemberAndTopic` クエリ（3行目）に `space_id` が WHERE 条件に含まれていない

  ```sql
  -- 問題のあるコード（3行目）
  SELECT * FROM topic_members WHERE space_member_id = $1 AND topic_id = $2;
  ```

  **修正案**:

  ```sql
  SELECT * FROM topic_members WHERE space_member_id = $1 AND topic_id = $2 AND space_id = $3;
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `space_id` を WHERE 条件に追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[@go/docs/security-guide.md#スペースIDによるクエリスコープ]**: `UpdateTopicMemberLastPageModifiedAt` クエリ（7行目）に `space_id` が WHERE 条件に含まれていない

  ```sql
  -- 問題のあるコード（7行目）
  UPDATE topic_members SET last_page_modified_at = $1, updated_at = $2 WHERE topic_id = $3 AND space_member_id = $4;
  ```

  **修正案**:

  ```sql
  UPDATE topic_members SET last_page_modified_at = $1, updated_at = $2 WHERE topic_id = $3 AND space_member_id = $4 AND space_id = $5;
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `space_id` を WHERE 条件に追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/db/queries/page_editors.sql`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - space_id によるクエリスコープ

**問題点・改善提案**:

- **[@go/docs/security-guide.md#スペースIDによるクエリスコープ]**: `FindPageEditorByPageAndSpaceMember` クエリ（3行目）に `space_id` が WHERE 条件に含まれていない。なお、同ファイル内の `UpdatePageEditorLastPageModifiedAt`（13-16行目）は正しく `space_id` を含んでいる

  ```sql
  -- 問題のあるコード（3行目）
  SELECT * FROM page_editors WHERE page_id = $1 AND space_member_id = $2;
  ```

  **修正案**:

  ```sql
  SELECT * FROM page_editors WHERE page_id = $1 AND space_member_id = $2 AND space_id = $3;
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `space_id` を WHERE 条件に追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/repository/topic_member.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - space_id によるクエリスコープ

**問題点・改善提案**:

- **[@go/docs/security-guide.md#スペースIDによるクエリスコープ]**: 上記のSQLクエリ修正に伴い、`FindBySpaceMemberAndTopic`（29-41行目）と `UpdateLastPageModifiedAt`（44-51行目）のメソッドシグネチャおよびクエリ呼び出しに `spaceID model.SpaceID` パラメータを追加する必要がある

  ```go
  // 問題のあるコード（29-41行目）- FindBySpaceMemberAndTopic に spaceID が渡されていない
  func (r *TopicMemberRepository) FindBySpaceMemberAndTopic(ctx context.Context, spaceMemberID model.SpaceMemberID, topicID model.TopicID) (*model.TopicMember, error) {
      row, err := r.q.FindTopicMemberBySpaceMemberAndTopic(ctx, query.FindTopicMemberBySpaceMemberAndTopicParams{
          SpaceMemberID: string(spaceMemberID),
          TopicID:       string(topicID),
      })
  ```

  **修正案**:

  ```go
  func (r *TopicMemberRepository) FindBySpaceMemberAndTopic(ctx context.Context, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, topicID model.TopicID) (*model.TopicMember, error) {
      row, err := r.q.FindTopicMemberBySpaceMemberAndTopic(ctx, query.FindTopicMemberBySpaceMemberAndTopicParams{
          SpaceMemberID: string(spaceMemberID),
          TopicID:       string(topicID),
          SpaceID:       string(spaceID),
      })
  ```

  同様に `UpdateLastPageModifiedAt` にも `spaceID` パラメータを追加する。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] SQLクエリ修正に合わせてリポジトリのメソッドシグネチャも修正する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/model/page.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md) - ドメインID型の使用

**問題点・改善提案**:

- **[@go/CLAUDE.md#ドメインID型]**: `FeaturedImageAttachmentID` フィールド（24行目）が `*string` 型を使用しているが、ドメインID型ガイドラインでは「IDフィールドに `string` を使用しない」とされている

  ```go
  // 問題のあるコード（24行目）
  FeaturedImageAttachmentID *string
  ```

  **修正案**:

  ```go
  FeaturedImageAttachmentID *AttachmentID
  ```

  ただし、`AttachmentID` 型が `id.go` に既に定義されているか、また `*AttachmentID` 型への変換が既存コードに影響を与えないかの確認が必要。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] `*AttachmentID` に変更する
  - [ ] 現状のまま（既存の `docs/plans/2_todo/domain-id-types-remaining.md` で対応予定のため）
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

全体として非常に高品質な実装です。3層アーキテクチャの依存関係ルール、ドメインID型の使用、Repository の WithTx パターン、テストの TestMain パターンなど、すべてのガイドラインに忠実に従っています。特に以下の点が優れています：

- **markupパッケージ**: インターフェースベースのDIでアーキテクチャ境界を適切に保っており、パイプラインパターンの設計が優秀
- **policyパッケージ**: ロールごとのポリシー実装がインターフェースで統一されており、拡張性が高い
- **テスト**: 全リポジトリに対するテストが網羅的で、ビルダーパターンの活用も一貫している
- **golangci-lint設定**: depguardルールで新しいパッケージの依存関係も適切に制御されている

修正が必要な点は **セキュリティ（space_id によるクエリスコープ）** に関する3箇所のSQLクエリのみです。`topic_members.sql` の SELECT/UPDATE と `page_editors.sql` の SELECT に `space_id` が WHERE 条件として含まれていません。同ファイル内の他のクエリ（`page_editors.sql` の UPDATE）は正しく `space_id` を含んでおり、一部のクエリで漏れている状態です。
