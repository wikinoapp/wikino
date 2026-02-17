# コードレビュー: page-edit-fix

## レビュー情報

| 項目                         | 内容                                           |
| ---------------------------- | ---------------------------------------------- |
| レビュー日                   | 2026-02-17                                     |
| 対象ブランチ                 | page-edit-fix                                  |
| ベースブランチ               | page-edit                                      |
| 作業計画書（指定があれば）   | docs/plans/1_doing/page-edit-go-migration.md   |
| 変更ファイル数               | 78 ファイル（実装・テスト・設定、ドキュメント除く） |
| 変更行数（実装）             | +2597 / -0 行                                  |
| 変更行数（テスト）           | +5377 / -0 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 変更ファイル一覧

### 実装ファイル

#### モデル

- [x] `go/internal/model/id.go`
- [x] `go/internal/model/page.go`
- [x] `go/internal/model/page_revision.go`
- [x] `go/internal/model/draft_page.go`
- [x] `go/internal/model/page_attachment_reference.go`
- [x] `go/internal/model/page_editor.go`
- [x] `go/internal/model/topic.go`
- [x] `go/internal/model/topic_member.go`
- [x] `go/internal/model/space.go`
- [x] `go/internal/model/space_member.go`
- [x] `go/internal/model/attachment.go`

#### ポリシー

- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`

#### リポジトリ

- [x] `go/internal/repository/page.go`
- [x] `go/internal/repository/page_revision.go`
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/repository/page_attachment_reference.go`
- [x] `go/internal/repository/page_editor.go`
- [x] `go/internal/repository/topic.go`
- [x] `go/internal/repository/topic_member.go`
- [x] `go/internal/repository/space.go`
- [x] `go/internal/repository/space_member.go`
- [x] `go/internal/repository/attachment.go`

#### マークアップ

- [x] `go/internal/markup/markup.go`
- [ ] `go/internal/markup/wikilink.go`
- [x] `go/internal/markup/attachment_extract.go`
- [x] `go/internal/markup/attachment_filter.go`
- [x] `go/internal/markup/batch.go`
- [x] `go/internal/markup/htmlutil.go`

#### SQLクエリ

- [x] `go/db/queries/pages.sql`
- [x] `go/db/queries/page_revisions.sql`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/db/queries/page_attachment_references.sql`
- [x] `go/db/queries/page_editors.sql`
- [x] `go/db/queries/topics.sql`
- [x] `go/db/queries/topic_members.sql`
- [x] `go/db/queries/spaces.sql`
- [x] `go/db/queries/space_members.sql`
- [x] `go/db/queries/attachments.sql`

#### sqlc生成コード（自動生成）

- [x] `go/internal/query/pages.sql.go`
- [x] `go/internal/query/page_revisions.sql.go`
- [x] `go/internal/query/draft_pages.sql.go`
- [x] `go/internal/query/page_attachment_references.sql.go`
- [x] `go/internal/query/page_editors.sql.go`
- [x] `go/internal/query/topics.sql.go`
- [x] `go/internal/query/topic_members.sql.go`
- [x] `go/internal/query/spaces.sql.go`
- [x] `go/internal/query/space_members.sql.go`
- [x] `go/internal/query/attachments.sql.go`

### テストファイル

- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/repository/page_revision_test.go`
- [x] `go/internal/repository/draft_page_test.go`
- [x] `go/internal/repository/page_attachment_reference_test.go`
- [x] `go/internal/repository/page_editor_test.go`
- [ ] `go/internal/repository/topic_test.go`
- [x] `go/internal/repository/topic_member_test.go`
- [x] `go/internal/repository/space_test.go`
- [x] `go/internal/repository/space_member_test.go`
- [x] `go/internal/repository/attachment_test.go`
- [x] `go/internal/policy/topic_test.go`
- [x] `go/internal/markup/markup_test.go`
- [x] `go/internal/markup/wikilink_test.go`
- [x] `go/internal/markup/attachment_extract_test.go`
- [x] `go/internal/markup/attachment_filter_test.go`
- [x] `go/internal/markup/batch_test.go`
- [x] `go/internal/markup/pipeline_integration_test.go`

### テストユーティリティ

- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/testutil/page_revision_builder.go`
- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `go/internal/testutil/attachment_builder.go`
- [x] `go/internal/testutil/space_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/topic_builder.go`
- [x] `go/internal/testutil/topic_member_builder.go`

### 設定・その他

- [x] `go/.golangci.yml`
- [x] `go/go.mod`
- [x] `go/go.sum`

## ファイルごとのレビュー結果

### `go/internal/repository/topic_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テストヘルパーの活用、ビルダーパターン

**問題点・改善提案**:

- **[@go/CLAUDE.md#テストヘルパーの使用]**: 256-267行目でtopic_membersレコードの作成に生SQLを使用している。`testutil.NewTopicMemberBuilder`が利用可能であるため、ビルダーパターンを使用すべき。

  ```go
  // 問題のあるコード（256-267行目）
  now := time.Now()
  for _, topicID := range []model.TopicID{topicID1, topicID2} {
      _, err := tx.ExecContext(
          context.Background(),
          `INSERT INTO topic_members (space_id, topic_id, space_member_id, role, joined_at, created_at, updated_at)
           VALUES ($1, $2, $3, $4, $5, $6, $7)`,
          string(spaceID), string(topicID), string(spaceMemberID), 0, now, now, now,
      )
      if err != nil {
          t.Fatalf("topic_member作成に失敗: %v", err)
      }
  }
  ```

  **修正案**:

  ```go
  for _, topicID := range []model.TopicID{topicID1, topicID2} {
      testutil.NewTopicMemberBuilder(t, tx).
          WithSpaceID(spaceID).
          WithTopicID(topicID).
          WithSpaceMemberID(spaceMemberID).
          Build()
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->

  - [x] 修正案の通りビルダーパターンに変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/markup/wikilink.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ログ出力](/workspace/go/CLAUDE.md) - slog.Warnの使用

**問題点・改善提案**:

- **[@go/CLAUDE.md#ログ出力]**: 96-98行目で`parseHTMLFragmentWithContainer()`のエラーを無視して元のHTMLを返している。同一パッケージの`attachment_filter.go`ではHTMLパースエラー時に`error`を返しているため、一貫性がない。パースに失敗した場合、`slog.Warn`でログ出力するか、パターンを統一すべき。

  ```go
  // 現在のコード（96-98行目）
  container, err := parseHTMLFragmentWithContainer(bodyHTML)
  if err != nil {
      return bodyHTML
  }
  ```

  **修正案**:

  ```go
  container, err := parseHTMLFragmentWithContainer(bodyHTML)
  if err != nil {
      slog.Warn("Wikiリンク変換時のHTMLパースに失敗", "error", err)
      return bodyHTML
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->

  - [x] 修正案の通りslog.Warnを追加する
  - [ ] 現状のまま（エラー時は安全にフォールバックしており問題なし）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 設計との整合性チェック

作業計画書で定義されている要件・設計に対する実装の整合性を確認した。

### 実装済み（本PR範囲）

- [x] **ドメインモデル**: Page, PageRevision, DraftPage, PageAttachmentReference, PageEditor, Topic, TopicMember, Space, SpaceMember, Attachment — 全モデルが作業計画書のデータ構造通りに実装されている
- [x] **ドメインID型**: 全モデルのIDフィールドに専用型を使用（UserIDは既存コードベース全体でstring型のため、現状は許容）
- [x] **リポジトリ**: 作業計画書に記載された全リポジトリメソッドが実装されている（FindBySpaceAndNumber, FindByIDs, FindBacklinkedByPageID, Update, FindByTopicAndTitle, CreateLinkedPage, SearchPageLocations 等）
- [x] **ポリシー**: TopicPolicyが正しく実装されている（Owner/Admin/Member/Guestのロール分離、CanUpdatePage/CanUpdateDraftPage）
- [x] **Markdownレンダリング**: goldmark + bluemondayによるパイプラインが実装されている
- [x] **Wikiリンク解析**: `[[ページ名]]`と`[[トピック名/ページ名]]`の両形式をサポート
- [x] **添付ファイルフィルター**: HTML img/aタグの変換、ファイル種別判定、プレースホルダー方式
- [x] **添付ファイルID抽出**: 4パターン（HTML img/a、Markdown画像/リンク）での検出
- [x] **アイキャッチ画像抽出**: body 1行目からの画像ID抽出
- [x] **バッチレンダリング**: N+1問題回避のためのバッチ処理基盤
- [x] **SQLクエリのspace_idスコープ**: 全クエリでspace_idが条件に含まれている
- [x] **golangci-lintルール**: PolicyとMarkup層の依存関係ルールが追加されている
- [x] **テスト**: 全リポジトリ、ポリシー、マークアップパッケージにテストが実装されている

### 未実装（本PR範囲外、作業計画書で別フェーズ/別タスク）

- ハンドラー（page/, draft_page/, page_location/）— 後続PRで実装予定
- ユースケース（update_draft_page, publish_page）— 後続PRで実装予定
- テンプレート・フロントエンド — 後続PRで実装予定
- I18n翻訳 — 後続PRで実装予定

### 乖離

設計との乖離はなし。

## 総合評価

**評価**: Comment

**総評**:

全体的に高品質な実装。作業計画書のデータモデル設計に忠実に従い、ドメインID型の一貫した使用、全SQLクエリでのspace_idスコープ、包括的なテストカバレッジが確認できた。

**良かった点**:

- テスト行数（5377行）が実装行数（2597行）の2倍以上と、テストが充実している
- 全SQLクエリでspace_idによるスコープが適切に設定されている（セキュリティガイドライン準拠）
- golangci-lintにPolicy層とMarkup層の依存関係ルールを追加し、アーキテクチャルールを強制している
- MarkupパッケージがRepository/Queryに依存せずインターフェースで抽象化されており、テスタビリティが高い
- ビルダーパターンによるテストデータ作成が一貫して使われている

**指摘事項**:

- topic_test.goの1箇所で生SQLが使われている（ビルダーに置き換え可能）— 軽微
- wikilink.goのHTMLパースエラー時にログ出力がない — 軽微だが一貫性のため推奨
