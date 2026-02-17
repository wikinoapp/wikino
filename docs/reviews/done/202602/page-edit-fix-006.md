# コードレビュー: page-edit-fix (006)

## レビュー情報

| 項目                         | 内容                                                            |
| ---------------------------- | --------------------------------------------------------------- |
| レビュー日                   | 2026-02-17                                                      |
| 対象ブランチ                 | page-edit-fix                                                   |
| ベースブランチ               | page-edit                                                       |
| 作業計画書（指定があれば）   | docs/plans/1_doing/page-edit-go-migration.md                    |
| 変更ファイル数               | 121 ファイル                                                    |
| 変更行数（実装）             | +3121 行（Go実装39ファイル、テスト・生成コード除く）            |
| 変更行数（テスト）           | +5351 行（テスト17ファイル）                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 変更ファイル一覧

### 実装ファイル

**モデル (`internal/model/`)**:

- [x] `go/internal/model/id.go`
- [x] `go/internal/model/page.go`
- [x] `go/internal/model/draft_page.go`
- [x] `go/internal/model/page_revision.go`
- [x] `go/internal/model/page_editor.go`
- [x] `go/internal/model/page_attachment_reference.go`
- [x] `go/internal/model/space.go`
- [x] `go/internal/model/space_member.go`
- [x] `go/internal/model/topic.go`
- [x] `go/internal/model/topic_member.go`
- [x] `go/internal/model/attachment.go`

**ポリシー (`internal/policy/`)**:

- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_guest.go`

**リポジトリ (`internal/repository/`)**:

- [x] `go/internal/repository/page.go`
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/repository/page_revision.go`
- [x] `go/internal/repository/page_editor.go`
- [x] `go/internal/repository/page_attachment_reference.go`
- [x] `go/internal/repository/space.go`
- [x] `go/internal/repository/space_member.go`
- [x] `go/internal/repository/topic.go`
- [x] `go/internal/repository/topic_member.go`
- [x] `go/internal/repository/attachment.go`

**Markupパッケージ (`internal/markup/`)**:

- [x] `go/internal/markup/markup.go`
- [x] `go/internal/markup/wikilink.go`
- [x] `go/internal/markup/attachment_extract.go`
- [x] `go/internal/markup/attachment_filter.go`
- [x] `go/internal/markup/batch.go`
- [x] `go/internal/markup/htmlutil.go`

**テストユーティリティ (`internal/testutil/`)**:

- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `go/internal/testutil/page_revision_builder.go`
- [x] `go/internal/testutil/space_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/topic_builder.go`
- [x] `go/internal/testutil/topic_member_builder.go`

### テストファイル

- [x] `go/internal/markup/markup_test.go`
- [x] `go/internal/markup/wikilink_test.go`
- [x] `go/internal/markup/attachment_filter_test.go`
- [x] `go/internal/markup/attachment_extract_test.go`
- [x] `go/internal/markup/batch_test.go`
- [x] `go/internal/markup/pipeline_integration_test.go`
- [x] `go/internal/policy/topic_test.go`
- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/repository/draft_page_test.go`
- [x] `go/internal/repository/page_revision_test.go`
- [x] `go/internal/repository/page_editor_test.go`
- [x] `go/internal/repository/page_attachment_reference_test.go`
- [x] `go/internal/repository/space_test.go`
- [x] `go/internal/repository/space_member_test.go`
- [x] `go/internal/repository/topic_test.go`
- [x] `go/internal/repository/topic_member_test.go`
- [x] `go/internal/repository/attachment_test.go`

### 設定・その他

- [x] `go/.golangci.yml`
- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/security-guide.md`

### SQLクエリファイル

- [x] `go/db/queries/pages.sql`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/db/queries/page_revisions.sql`
- [x] `go/db/queries/page_editors.sql`
- [x] `go/db/queries/page_attachment_references.sql`
- [x] `go/db/queries/topics.sql`
- [x] `go/db/queries/topic_members.sql`
- [x] `go/db/queries/spaces.sql`
- [x] `go/db/queries/space_members.sql`
- [x] `go/db/queries/attachments.sql`

### 生成コード（自動生成、レビュー対象外）

- [x] `go/internal/query/pages.sql.go`
- [x] `go/internal/query/draft_pages.sql.go`
- [x] `go/internal/query/page_revisions.sql.go`
- [x] `go/internal/query/page_editors.sql.go`
- [x] `go/internal/query/page_attachment_references.sql.go`
- [x] `go/internal/query/topics.sql.go`
- [x] `go/internal/query/topic_members.sql.go`
- [x] `go/internal/query/spaces.sql.go`
- [x] `go/internal/query/space_members.sql.go`
- [x] `go/internal/query/attachments.sql.go`

## ファイルごとのレビュー結果

### `go/internal/markup/attachment_extract.go`: `featuredHTMLImgRegex`の正規表現パターン不整合

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - XSS対策・入力バリデーション

**問題点・改善提案**:

- **[@go/docs/security-guide.md#入力バリデーション]**: `featuredHTMLImgRegex`（26行目）のIDキャプチャパターンが他の抽出用正規表現と不整合

  同ファイルの他の正規表現は `/` を除外するパターン `([^/"']+)` を使用しているが、`featuredHTMLImgRegex` のみ `([^"']+)` を使用しており、`/attachments/foo/bar` のようなサブパスが `foo/bar` としてマッチしてしまう。

  ```go
  // 問題のあるコード（26行目）
  featuredHTMLImgRegex = regexp.MustCompile(`(?i)<img[^>]+src=["']/attachments/([^"']+)["'][^>]*>`)
  ```

  **修正案**:

  ```go
  // 他の正規表現と同じパターンに統一
  featuredHTMLImgRegex = regexp.MustCompile(`(?i)<img[^>]+src=["']/attachments/([^/"']+)["'][^>]*>`)
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->

  - [x] 修正案の通り `([^/"']+)` に変更する
  - [ ] 現状のまま（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/repository/attachment.go`: `toModel`ヘルパーメソッドの未使用

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Repository層のパターン

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#ModelとRepositoryの1:1関係]**: 他のすべてのリポジトリは`toModel`ヘルパーメソッドを使用してクエリ結果をモデルに変換しているが、`attachment.go`のみインラインで`model.Attachment`を構築している（`FindByIDAndSpace`と`FindByIDsAndSpace`で同じ構築コードが重複）

  ```go
  // 現状: 同じ構築コードが2箇所に重複
  return &model.Attachment{
      ID:       model.AttachmentID(row.ID),
      SpaceID:  model.SpaceID(row.SpaceID),
      Filename: row.Filename,
  }, nil
  ```

  **修正案**:

  ```go
  // toModel ヘルパーを追加して重複を解消
  func toModel(row query.FindAttachmentByIDAndSpaceRow) *model.Attachment {
      return &model.Attachment{
          ID:       model.AttachmentID(row.ID),
          SpaceID:  model.SpaceID(row.SpaceID),
          Filename: row.Filename,
      }
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->

  - [x] 修正案の通り `toModel` を追加する
  - [ ] 現状のまま（フィールドが3つだけなので重複は許容範囲）
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

全体的に高品質な実装です。以下の点が特に優れています：

- **セキュリティ**: 全SQLクエリが`space_id`スコープを正しく適用しており、`page_attachment_references`テーブルのようにspace_idカラムを持たないテーブルもJOINベースのスコーピングを適切に実装しています
- **アーキテクチャ**: 3層アーキテクチャの依存関係ルールが厳守されており、depguard設定も新しいパッケージ（policy, markup）に正しく対応しています
- **ドメインID型**: 全モデル・リポジトリで一貫してドメインID型が使用されており、型安全性が確保されています
- **ポリシーパターン**: TopicPolicyのインターフェース + ロール別構造体の実装は、設計書通りの純粋ロジックで、DBアクセスなし・コンパイル時型安全を実現しています
- **テストカバレッジ**: markupパッケージで90以上のテストケース、リポジトリテストで正常系・異常系の網羅的なカバレッジがあり、非常に充実しています
- **XSS対策**: bluemondayサニタイゼーション、`golang.org/x/net/html`によるDOM操作、テストでのXSSベクター検証が適切に実装されています

指摘事項は2件で、いずれも軽微です。`featuredHTMLImgRegex`の正規表現パターン不整合は他の正規表現との一貫性のため推奨、`toModel`の追加はスタイルの統一のための提案です。
