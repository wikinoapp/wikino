# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                         |
| -------------------------- | -------------------------------------------- |
| レビュー日                 | 2026-02-17                                   |
| 対象ブランチ               | page-edit-fix                                |
| ベースブランチ             | page-edit                                    |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md |
| 変更ファイル数             | 119 ファイル                                 |
| 変更行数（実装）           | +4461 / -0 行                                |
| 変更行数（テスト）         | +5351 / -0 行                                |

## 参照するガイドライン

- [@CLAUDE.md](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/model/id.go`
- [x] `go/internal/model/space.go`
- [x] `go/internal/model/space_member.go`
- [x] `go/internal/model/topic.go`
- [ ] `go/internal/model/topic_member.go`
- [x] `go/internal/model/page.go`
- [x] `go/internal/model/draft_page.go`
- [x] `go/internal/model/page_revision.go`
- [x] `go/internal/model/page_editor.go`
- [x] `go/internal/model/page_attachment_reference.go`
- [x] `go/internal/model/attachment.go`
- [x] `go/internal/repository/space.go`
- [x] `go/internal/repository/space_member.go`
- [ ] `go/internal/repository/topic.go`
- [x] `go/internal/repository/topic_member.go`
- [x] `go/internal/repository/page.go`
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/repository/page_revision.go`
- [x] `go/internal/repository/page_editor.go`
- [x] `go/internal/repository/page_attachment_reference.go`
- [x] `go/internal/repository/attachment.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/markup/markup.go`
- [ ] `go/internal/markup/wikilink.go`
- [ ] `go/internal/markup/attachment_filter.go`
- [ ] `go/internal/markup/attachment_extract.go`
- [x] `go/internal/markup/htmlutil.go`
- [ ] `go/internal/markup/batch.go`
- [x] `go/.golangci.yml`

### SQLクエリファイル

- [x] `go/db/queries/spaces.sql`
- [x] `go/db/queries/space_members.sql`
- [ ] `go/db/queries/topics.sql`
- [x] `go/db/queries/topic_members.sql`
- [x] `go/db/queries/pages.sql`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/db/queries/page_revisions.sql`
- [x] `go/db/queries/page_editors.sql`
- [x] `go/db/queries/page_attachment_references.sql`
- [x] `go/db/queries/attachments.sql`

### sqlc生成ファイル

- [x] `go/internal/query/spaces.sql.go`
- [x] `go/internal/query/space_members.sql.go`
- [x] `go/internal/query/topics.sql.go`
- [x] `go/internal/query/topic_members.sql.go`
- [x] `go/internal/query/pages.sql.go`
- [x] `go/internal/query/draft_pages.sql.go`
- [x] `go/internal/query/page_revisions.sql.go`
- [x] `go/internal/query/page_editors.sql.go`
- [x] `go/internal/query/page_attachment_references.sql.go`
- [x] `go/internal/query/attachments.sql.go`

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

### ドキュメント・設定

- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/security-guide.md`
- [x] `docs/reviews/template.md`
- [x] `docs/plans/template.md`
- [x] `docs/specs/template.md`
- [x] `docs/README.md`
- [x] その他 `docs/plans/` 配下のファイル

## ファイルごとのレビュー結果

### `go/db/queries/topics.sql`: `ListTopicsJoinedBySpaceMember`にspace_idスコープが不足

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - スペースIDによるクエリスコープ
- [@go/CLAUDE.md#セキュリティガイドライン](/workspace/go/CLAUDE.md) - スペースIDスコープ

**問題点・改善提案**:

- **[@go/docs/security-guide.md#スペースIDによるクエリスコープ]**: `ListTopicsJoinedBySpaceMember`クエリに`space_id`条件が不足している

  ```sql
  -- 問題のあるコード (go/db/queries/topics.sql)
  -- name: ListTopicsJoinedBySpaceMember :many
  SELECT t.* FROM topics t
  INNER JOIN topic_members tm ON t.id = tm.topic_id
  WHERE tm.space_member_id = $1 AND t.discarded_at IS NULL
  ORDER BY t.number;
  ```

  `space_member_id`はスペースに紐づく識別子ではあるが、セキュリティガイドラインでは防御的プログラミングとしてクエリレベルでも`space_id`を条件に含めることが求められている。

  **修正案**:

  ```sql
  -- name: ListTopicsJoinedBySpaceMember :many
  SELECT t.* FROM topics t
  INNER JOIN topic_members tm ON t.id = tm.topic_id
  WHERE tm.space_member_id = $1 AND t.space_id = $2 AND t.discarded_at IS NULL
  ORDER BY t.number;
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り`space_id`条件を追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/model/topic_member.go`: DBスキーマに存在する`CreatedAt`/`UpdatedAt`フィールドが未定義

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md) - モデル定義の一貫性
- [docs/plans/1_doing/page-edit-go-migration.md](/workspace/docs/plans/1_doing/page-edit-go-migration.md) - 作業計画書

**問題点・改善提案**:

- **作業計画書との整合性**: DBスキーマの`topic_members`テーブルには`created_at`と`updated_at`カラムが存在するが、`model.TopicMember`にはこれらのフィールドが定義されていない

  ```go
  // 現在のコード (go/internal/model/topic_member.go)
  type TopicMember struct {
      ID                 TopicMemberID
      SpaceID            SpaceID
      TopicID            TopicID
      SpaceMemberID      SpaceMemberID
      Role               TopicMemberRole
      JoinedAt           time.Time
      LastPageModifiedAt *time.Time
      // CreatedAt, UpdatedAt が不足
  }
  ```

  DBスキーマ（`db/schema.sql`）:

  ```sql
  CREATE TABLE public.topic_members (
      ...
      created_at timestamp(6) without time zone NOT NULL,
      updated_at timestamp(6) without time zone NOT NULL
  );
  ```

  **修正案**:

  ```go
  type TopicMember struct {
      ID                 TopicMemberID
      SpaceID            SpaceID
      TopicID            TopicID
      SpaceMemberID      SpaceMemberID
      Role               TopicMemberRole
      JoinedAt           time.Time
      LastPageModifiedAt *time.Time
      CreatedAt          time.Time
      UpdatedAt          time.Time
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 修正案の通り`CreatedAt`/`UpdatedAt`を追加する
  - [x] 現時点では不要（使用箇所がないためYAGNI）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/markup/attachment_extract.go`: 正規表現パターンの不一致

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@CLAUDE.md#既存コードとの一貫性](/workspace/CLAUDE.md) - 一貫性

**問題点・改善提案**:

- **一貫性**: `extractMarkdownImgRegex`と`featuredMarkdownImgRegex`のキャプチャグループが異なる

  ```go
  // 問題のあるコード (go/internal/markup/attachment_extract.go)

  // extractMarkdownImgRegex: ([^/)]+) — "/" と ")" を除外
  extractMarkdownImgRegex = regexp.MustCompile(`!\[[^\]]*\]\(/attachments/([^/)]+)\)`)

  // featuredMarkdownImgRegex: ([^)]+) — ")" のみを除外（"/" を許容）
  featuredMarkdownImgRegex = regexp.MustCompile(`!\[[^\]]*\]\(/attachments/([^)]+)\)`)
  ```

  添付ファイルIDに`/`が含まれることは通常想定されないが、2つの正規表現でキャプチャグループの定義が異なるのは意図しない不一致の可能性がある。同じ対象（添付ファイルID）を抽出するパターンは統一すべき。

  **修正案**:

  ```go
  // 両方とも同じキャプチャグループに統一
  extractMarkdownImgRegex  = regexp.MustCompile(`!\[[^\]]*\]\(/attachments/([^/)]+)\)`)
  featuredMarkdownImgRegex = regexp.MustCompile(`!\[[^\]]*\]\(/attachments/([^/)]+)\)`)
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り`([^/)]+)`に統一する
  - [ ] `([^)]+)`に統一する（`/`を許容）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/markup/attachment_filter.go`: 動画フォールバックリンクにテキストコンテンツがない

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@CLAUDE.md#Railsからの移行について](/workspace/CLAUDE.md) - Rails版との一貫性

**問題点・改善提案**:

- **Rails版との一貫性**: `replaceWithInlineVideo`で生成されるフォールバック`<a>`タグにテキストコンテンツがない

  ```go
  // 問題のあるコード (go/internal/markup/attachment_filter.go)
  fallbackA := &html.Node{
      Type:     html.ElementNode,
      DataAtom: atom.A,
      Data:     "a",
      Attr: []html.Attribute{
          {Key: "href", Val: "#"},
          {Key: "data-attachment-id", Val: attachmentID},
          {Key: "data-attachment-link", Val: "true"},
          {Key: "target", Val: "_blank"},
      },
  }
  videoNode.AppendChild(fallbackA)
  ```

  `<video>`タグ未対応ブラウザではフォールバックの`<a>`タグが表示されるが、テキストが空のため何も表示されない。

  **修正案**:

  ```go
  fallbackA := &html.Node{
      Type:     html.ElementNode,
      DataAtom: atom.A,
      Data:     "a",
      Attr: []html.Attribute{
          {Key: "href", Val: "#"},
          {Key: "data-attachment-id", Val: attachmentID},
          {Key: "data-attachment-link", Val: "true"},
          {Key: "target", Val: "_blank"},
      },
  }
  // フォールバックテキストを追加
  fallbackText := &html.Node{
      Type: html.TextNode,
      Data: attachmentID,
  }
  fallbackA.AppendChild(fallbackText)
  videoNode.AppendChild(fallbackA)
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りフォールバックテキストを追加する
  - [ ] Rails版のフォールバック実装を確認してから判断する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/markup/wikilink.go`: `spaceIdentifier`がURLパスエスケープされていない

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#XSS対策](/workspace/go/docs/security-guide.md) - XSS対策

**問題点・改善提案**:

- **[@go/docs/security-guide.md#XSS対策]**: `buildWikilinkNode`で`spaceIdentifier`がURLパスエスケープされずに`href`に埋め込まれている

  ```go
  // 問題のあるコード (go/internal/markup/wikilink.go)
  func buildWikilinkNode(spaceIdentifier string, pl *PageLocation) *html.Node {
      href := fmt.Sprintf("/s/%s/pages/%d", spaceIdentifier, pl.PageNumber)
      // ...
  }
  ```

  `spaceIdentifier`に特殊文字（`/`, `?`, `#`, スペースなど）が含まれた場合、URL構造が壊れる可能性がある。現時点ではスペース識別子はバリデーション済みの英数字が想定されるが、防御的プログラミングとしてエスケープが望ましい。

  **修正案**:

  ```go
  import "net/url"

  func buildWikilinkNode(spaceIdentifier string, pl *PageLocation) *html.Node {
      href := fmt.Sprintf("/s/%s/pages/%d", url.PathEscape(spaceIdentifier), pl.PageNumber)
      // ...
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り`url.PathEscape`を追加する
  - [ ] スペース識別子は英数字のみであり対応不要
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/markup/batch.go`: `mapAttachmentFinder.FindByIDAndSpace`が`spaceID`パラメータを無視している

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - スペースIDスコープ

**問題点・改善提案**:

- **[@go/docs/security-guide.md#スペースIDによるクエリスコープ]**: `mapAttachmentFinder`の`FindByIDAndSpace`メソッドが`spaceID`パラメータを無視してIDのみで検索している

  ```go
  // 問題のあるコード (go/internal/markup/batch.go)
  func (f *mapAttachmentFinder) FindByIDAndSpace(_ context.Context, id model.AttachmentID, _ model.SpaceID) (*model.Attachment, error) {
      return f.attachments[id], nil
  }
  ```

  バッチレンダリング時にマップに事前ロードされた添付ファイルのみが対象となるため、実際にはスペース外の添付ファイルが混入するリスクは低い。しかし、インターフェースが`spaceID`を引数に取る以上、無視するのは防御的プログラミングに反する。

  **修正案**:

  マップのキーを`(AttachmentID, SpaceID)`のペアにするか、検索結果の`SpaceID`を検証する。

  ```go
  func (f *mapAttachmentFinder) FindByIDAndSpace(_ context.Context, id model.AttachmentID, spaceID model.SpaceID) (*model.Attachment, error) {
      att := f.attachments[id]
      if att != nil && att.SpaceID != spaceID {
          return nil, nil
      }
      return att, nil
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り`SpaceID`の検証を追加する
  - [ ] バッチレンダリングでは事前フィルタ済みのため対応不要
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

ページ編集機能のGo移行に向けた基盤実装として、モデル・リポジトリ・ポリシー・マークアップ処理が包括的に実装されています。テストカバレッジも充実しており（テストコード5351行 vs 実装コード4461行）、品質意識の高さが伺えます。

**良かった点**:

- ドメインID型（`SpaceID`, `PageID`等）の一貫した使用により型安全性が確保されている
- 3層アーキテクチャの依存関係ルールが`.golangci.yml`のdepguardで機械的に強制されている
- ポリシーパターンによるトピック権限管理が柔軟に設計されている
- マークアップ処理がgoldmark + bluemonday + DOM操作で安全に実装されている
- テストビルダーパターンによりテストデータの作成が簡潔

**修正が必要な点**:

1. **セキュリティ（必須）**: `ListTopicsJoinedBySpaceMember`クエリに`space_id`スコープが不足（セキュリティガイドライン違反）
2. **一貫性（必須）**: 正規表現パターンの不一致（`extractMarkdownImgRegex` vs `featuredMarkdownImgRegex`）

**確認が必要な点**:

3. `TopicMember`モデルに`CreatedAt`/`UpdatedAt`が不足（DBスキーマとの乖離）
4. 動画フォールバックリンクにテキストコンテンツがない（Rails版との一貫性）
5. `spaceIdentifier`のURLパスエスケープ（防御的プログラミング）
6. `mapAttachmentFinder.FindByIDAndSpace`の`spaceID`無視（防御的プログラミング）
