# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                         |
| -------------------------- | -------------------------------------------- |
| レビュー日                 | 2026-02-16                                   |
| 対象ブランチ               | page-edit-fix                                |
| ベースブランチ             | page-edit                                    |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md |
| 変更ファイル数             | 101 ファイル                                 |
| 変更行数（実装）           | +1308 / -0 行                                |
| 変更行数（テスト）         | +3035 / -0 行                                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md) - ドメインID型の使用ルール
- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - スペースIDスコープ
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/model/id.go`
- [ ] `go/internal/model/attachment.go`
- [x] `go/internal/model/draft_page.go`
- [x] `go/internal/model/page.go`
- [ ] `go/internal/model/page_attachment_reference.go`
- [ ] `go/internal/model/page_editor.go`
- [ ] `go/internal/model/page_revision.go`
- [x] `go/internal/model/space.go`
- [x] `go/internal/model/space_member.go`
- [x] `go/internal/model/topic.go`
- [x] `go/internal/model/topic_member.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
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

### SQLクエリファイル

- [x] `go/db/queries/attachments.sql`
- [x] `go/db/queries/draft_pages.sql`
- [ ] `go/db/queries/page_attachment_references.sql`
- [x] `go/db/queries/page_editors.sql`
- [x] `go/db/queries/page_revisions.sql`
- [ ] `go/db/queries/pages.sql`
- [x] `go/db/queries/space_members.sql`
- [x] `go/db/queries/spaces.sql`
- [x] `go/db/queries/topic_members.sql`
- [x] `go/db/queries/topics.sql`

### sqlc生成ファイル（自動生成、レビュー対象外）

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

### テストファイル

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

### テストユーティリティ

- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `go/internal/testutil/page_builder.go`
- [ ] `go/internal/testutil/page_revision_builder.go`
- [x] `go/internal/testutil/space_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/topic_builder.go`
- [x] `go/internal/testutil/topic_member_builder.go`

### ドキュメント・設定

- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/security-guide.md`
- [x] `docs/README.md`
- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/template.md`
- [x] `docs/reviews/template.md`
- [x] `docs/specs/template.md`
- [x] `.claude/commands/review.md`
- [x] その他 `docs/plans/` 配下のファイル（作業計画書の整理）

## ファイルごとのレビュー結果

### `go/internal/model/id.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md) - 「新しいモデルを追加する場合は対応するID型も追加」

**問題点・改善提案**:

- **[@go/CLAUDE.md#ドメインID型]**: 今回追加されたモデル（PageRevision, PageEditor, PageAttachmentReference, Attachment）に対応するドメインID型が定義されていない

  現在定義されているID型: `SpaceID`, `TopicID`, `PageID`, `SpaceMemberID`, `TopicMemberID`, `DraftPageID`

  **修正案**: 以下のID型と `String()` メソッドを `id.go` に追加する

  ```go
  // PageRevisionID はページリビジョンのID型
  type PageRevisionID string

  // PageEditorID はページエディターのID型
  type PageEditorID string

  // PageAttachmentReferenceID はページ添付ファイル参照のID型
  type PageAttachmentReferenceID string

  // AttachmentID は添付ファイルのID型
  type AttachmentID string

  func (id PageRevisionID) String() string           { return string(id) }
  func (id PageEditorID) String() string              { return string(id) }
  func (id PageAttachmentReferenceID) String() string { return string(id) }
  func (id AttachmentID) String() string              { return string(id) }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り4つのID型を追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/model/page_revision.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md) - 「IDフィールドに `string` を使用しない」

**問題点・改善提案**:

- **[@go/CLAUDE.md#ドメインID型]**: `ID` フィールドが `string` 型になっている

  ```go
  // 現在のコード
  type PageRevision struct {
      ID            string  // ← stringを使用している
      SpaceID       SpaceID
      ...
  }
  ```

  **修正案**: `id.go` にID型を追加後、モデルのIDフィールドを変更する

  ```go
  type PageRevision struct {
      ID            PageRevisionID
      SpaceID       SpaceID
      ...
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `PageRevisionID` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/model/page_editor.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md) - 「IDフィールドに `string` を使用しない」

**問題点・改善提案**:

- **[@go/CLAUDE.md#ドメインID型]**: `ID` フィールドが `string` 型になっている

  ```go
  // 現在のコード
  type PageEditor struct {
      ID                 string  // ← stringを使用している
      ...
  }
  ```

  **修正案**:

  ```go
  type PageEditor struct {
      ID                 PageEditorID
      ...
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `PageEditorID` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/model/page_attachment_reference.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md) - 「IDフィールドに `string` を使用しない」「外部キーにも専用型を使用」

**問題点・改善提案**:

- **[@go/CLAUDE.md#ドメインID型]**: `ID` フィールドと `AttachmentID` フィールドが `string` 型になっている

  ```go
  // 現在のコード
  type PageAttachmentReference struct {
      ID           string  // ← stringを使用している
      AttachmentID string  // ← stringを使用している（外部キー）
      PageID       PageID
      ...
  }
  ```

  **修正案**:

  ```go
  type PageAttachmentReference struct {
      ID           PageAttachmentReferenceID
      AttachmentID AttachmentID
      PageID       PageID
      ...
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `PageAttachmentReferenceID` と `AttachmentID` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/model/attachment.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md) - 「IDフィールドに `string` を使用しない」

**問題点・改善提案**:

- **[@go/CLAUDE.md#ドメインID型]**: `ID` フィールドが `string` 型になっている

  ```go
  // 現在のコード
  type Attachment struct {
      ID       string  // ← stringを使用している
      SpaceID  SpaceID
      Filename string
  }
  ```

  **修正案**:

  ```go
  type Attachment struct {
      ID       AttachmentID
      SpaceID  SpaceID
      Filename string
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `AttachmentID` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/db/queries/pages.sql`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - 「スペース内リソースへのクエリには必ず `space_id` を WHERE 条件に含める」

**問題点・改善提案**:

- **[@go/docs/security-guide.md#スペースIDによるクエリスコープ]**: `FindPageByTopicAndTitle`（38〜43行目）に `space_id` 条件がない

  `pages` テーブルは `space_id` カラムを持つスペース内リソースであるため、防御的プログラミングとして `space_id` を条件に含める必要がある。

  ```sql
  -- 現在のコード（pages.sql:38-43）
  -- name: FindPageByTopicAndTitle :one
  SELECT * FROM pages
  WHERE topic_id = $1
    AND title = $2
    AND discarded_at IS NULL;
  ```

  **修正案**:

  ```sql
  -- name: FindPageByTopicAndTitle :one
  SELECT * FROM pages
  WHERE topic_id = $1
    AND title = $2
    AND space_id = $3
    AND discarded_at IS NULL;
  ```

  **対応方針**:
  - [x] 修正案の通り `space_id` 条件を追加する
  - [ ] `topic_id` が既にスペーススコープを暗黙的に保証しているため現状のままにする（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/db/queries/page_attachment_references.sql`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - 「テーブル自体に space_id がない場合は JOIN で検証」

**問題点・改善提案**:

- **[@go/docs/security-guide.md#スペースIDによるクエリスコープ]**: `DeletePageAttachmentReferencesByPageAndAttachmentIDs`（13〜16行目）にスペース検証がない

  `page_attachment_references` テーブルは `space_id` カラムを持たないが、スペース内リソースである `pages` を経由してスペースに紐づく。同ファイルの `ListPageAttachmentReferencesByPageID`（1〜5行目）では正しく `pages` との JOIN でスペース検証を行っているが、DELETE クエリにはこの検証がない。

  ```sql
  -- 現在のコード（page_attachment_references.sql:13-16）
  -- name: DeletePageAttachmentReferencesByPageAndAttachmentIDs :exec
  DELETE FROM page_attachment_references
  WHERE page_id = $1 AND attachment_id = ANY($2::uuid[]);
  ```

  **修正案**: `pages` テーブルとの JOIN で `space_id` を検証する

  ```sql
  -- name: DeletePageAttachmentReferencesByPageAndAttachmentIDs :exec
  DELETE FROM page_attachment_references par
  USING pages p
  WHERE par.page_id = p.id
    AND par.page_id = $1
    AND p.space_id = $2
    AND par.attachment_id = ANY($3::uuid[]);
  ```

  **対応方針**:
  - [x] 修正案の通り USING 句で `space_id` を検証する
  - [ ] ハンドラー/ユースケース側で事前にスペースチェック済みのため現状のままにする（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/testutil/page_revision_builder.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md) - 「テストビルダーの戻り値」にドメインID型を使用するパターン

**問題点・改善提案**:

- **[@go/CLAUDE.md#ドメインID型]**: `Build()` メソッドの戻り値が `string` になっている。他のビルダー（`SpaceBuilder`, `PageBuilder` など）が `model.SpaceID`, `model.PageID` を返しているのと一貫性がない。

  ```go
  // 現在のコード（page_revision_builder.go:74）
  func (b *PageRevisionBuilder) Build() string {
  ```

  **修正案**: `id.go` に `PageRevisionID` を追加後、戻り値を変更する

  ```go
  func (b *PageRevisionBuilder) Build() model.PageRevisionID {
      // ...
      return model.PageRevisionID(id)
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `model.PageRevisionID` を返すように変更する
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

ページ編集Go移行の基盤コードとして、全体的に非常に高品質な実装です。特に以下の点が優れています：

- **リポジトリ層の設計**: 全10リポジトリが `WithTx` パターンを正しく実装し、モデル変換も適切
- **TopicPolicy**: Interface + ロール別構造体のパターンが見通しよく実装されている
- **テストカバレッジ**: 全リポジトリ・ポリシーに対する包括的なテストが書かれており、TestMainパターン・ビルダーパターン・t.Parallel()も正しく使用
- **SQLクエリの大半**: ほとんどのクエリで `space_id` スコープが適切に適用されている

修正が必要な点は以下の2カテゴリです：

1. **ドメインID型の欠落**（必須対応 5件）: `PageRevisionID`, `PageEditorID`, `PageAttachmentReferenceID`, `AttachmentID` が `id.go` に未定義で、対応するモデルのIDフィールドが `string` のまま。ガイドライン「IDフィールドに `string` を使用しない」に違反。
2. **SQLクエリのspace_idスコープ不足**（要確認 2件）: `FindPageByTopicAndTitle` と `DeletePageAttachmentReferencesByPageAndAttachmentIDs` にスペース検証が欠けている。

ドメインID型の修正は機械的な変更で対応可能です。SQLクエリについては開発者の判断を仰ぐ項目があるため、対応方針の回答をお願いします。
