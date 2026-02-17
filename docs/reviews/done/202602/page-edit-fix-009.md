# コードレビュー: page-edit-fix

## レビュー情報

| 項目                         | 内容                                              |
| ---------------------------- | ------------------------------------------------- |
| レビュー日                   | 2026-02-17                                        |
| 対象ブランチ                 | page-edit-fix                                     |
| ベースブランチ               | page-edit                                         |
| 作業計画書（指定があれば）   | docs/plans/1_doing/page-edit-go-migration.md      |
| 変更ファイル数               | 125 ファイル                                      |
| 変更行数（実装）             | +3477 / -20 行                                    |
| 変更行数（テスト）           | +5371 / -0 行                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

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
- [x] `go/internal/markup/markup.go`
- [ ] `go/internal/markup/wikilink.go`
- [x] `go/internal/markup/batch.go`
- [x] `go/internal/markup/attachment_extract.go`
- [x] `go/internal/markup/attachment_filter.go`
- [x] `go/internal/markup/htmlutil.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/testutil/attachment_builder.go`
- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/testutil/page_revision_builder.go`
- [x] `go/internal/testutil/space_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/topic_builder.go`
- [x] `go/internal/testutil/topic_member_builder.go`
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

### 設定・その他

- [x] `go/.golangci.yml`
- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] `go/internal/query/attachments.sql.go` (自動生成)
- [x] `go/internal/query/draft_pages.sql.go` (自動生成)
- [x] `go/internal/query/page_attachment_references.sql.go` (自動生成)
- [x] `go/internal/query/page_editors.sql.go` (自動生成)
- [x] `go/internal/query/page_revisions.sql.go` (自動生成)
- [x] `go/internal/query/pages.sql.go` (自動生成)
- [x] `go/internal/query/space_members.sql.go` (自動生成)
- [x] `go/internal/query/spaces.sql.go` (自動生成)
- [x] `go/internal/query/topic_members.sql.go` (自動生成)
- [x] `go/internal/query/topics.sql.go` (自動生成)

### ドキュメント

- [x] `docs/README.md`
- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/template.md`
- [x] `docs/reviews/template.md`
- [x] `docs/specs/template.md`
- [x] `docs/reviews/done/202602/page-edit-fix-001.md` 〜 `page-edit-fix-008.md`
- [x] 各種 `docs/plans/` ファイル
- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/security-guide.md`
- [x] `.claude/commands/review.md`

## ファイルごとのレビュー結果

### `go/internal/markup/wikilink.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md) - ドメインID型の使用ルール

**問題点・改善提案**:

- **[@go/CLAUDE.md#ドメインID型]**: `PageLocation.PageID` フィールドが `string` 型を使用している（43行目）

  同じパッケージ内の `batch.go` と `attachment_filter.go` は既に `model` パッケージをインポートしてドメインID型（`model.SpaceID`, `model.AttachmentID`）を使用している。`wikilink.go` の `PageLocation.PageID` だけが `string` のままになっており、ガイドラインの「IDフィールドに `string` を使用しない」に違反している。

  ```go
  // 現在のコード (wikilink.go:37-48)
  type PageLocation struct {
      Key        WikilinkKey
      TopicName  string
      PageID     string    // ← string を使用
      PageNumber int
      PageTitle  string
  }
  ```

  **修正案**:

  ```go
  import (
      // 既存のインポートに追加
      "github.com/wikinoapp/wikino/go/internal/model"
  )

  type PageLocation struct {
      Key        WikilinkKey
      TopicName  string
      PageID     model.PageID  // ← ドメインID型を使用
      PageNumber int
      PageTitle  string
  }
  ```

  なお、`PageLocation.PageID` は `wikilink.go` 内では参照されておらず、`batch.go` の `PageLocationResolver` インターフェース経由で外部から設定されるフィールドのため、テストコードの `PageID: "..."` を `PageID: model.PageID("...")` に変更する必要がある。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->

  - [x] 修正案の通り `model.PageID` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

ページ編集機能のGo移行に必要なDomain/Infrastructure層（Model、Repository、Query）、markupパッケージ、policyパッケージ、テストビルダーが包括的に実装されている。

**良かった点**:

- 全SQLクエリで `space_id` によるスコープが徹底されており、セキュリティガイドラインに準拠している
- 全モデルでドメインID型が正しく使用されている（`model.PageID`, `model.SpaceID` 等）
- 全リポジトリに `WithTx` メソッドが実装されており、トランザクション対応が万全
- golangci-lintの設定に `policy-layer` と `markup-layer` のdepguardルールが追加されており、アーキテクチャルールが静的に強制される
- テストが充実しており、正常系・異常系・境界値が網羅されている（+5371行）
- テストビルダーパターンが一貫しており、全ビルダーがドメインID型を返却している
- `page_attachment_references` のように自テーブルに `space_id` がない場合も、JOINで適切にスコープしている

**指摘事項**:

- `wikilink.go` の `PageLocation.PageID` が `string` 型になっている点のみ。同パッケージ内の他ファイルは既に `model` パッケージのドメインID型を使用しているため、一貫性の観点から修正が望ましい
