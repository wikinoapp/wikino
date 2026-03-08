# コードレビュー: page-edit-fix (7回目 - モデル・リポジトリ・ポリシー・マークアップ・テスト全体)

## レビュー情報

| 項目                           | 内容                                         |
| ------------------------------ | -------------------------------------------- |
| レビュー日                     | 2026-02-17                                   |
| 対象ブランチ                   | page-edit-fix                                |
| ベースブランチ                 | page-edit                                    |
| 作業計画書（指定があれば）     | docs/plans/1_doing/page-edit-go-migration.md |
| 変更ファイル数                 | 76 ファイル（Go実装・テスト・SQL・設定）     |
| 変更行数（実装）               | +3291 行                                     |
| 変更行数（テスト）             | +5351 行                                     |
| 変更行数（設定・ドキュメント） | +166 行                                      |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 変更ファイル一覧

### モデル

- [x] `go/internal/model/id.go`
- [x] `go/internal/model/page.go`
- [x] `go/internal/model/draft_page.go`
- [x] `go/internal/model/page_revision.go`
- [x] `go/internal/model/page_attachment_reference.go`
- [x] `go/internal/model/page_editor.go`
- [x] `go/internal/model/space.go`
- [ ] `go/internal/model/space_member.go`
- [x] `go/internal/model/topic.go`
- [x] `go/internal/model/topic_member.go`
- [x] `go/internal/model/attachment.go`

### リポジトリ

- [x] `go/internal/repository/page.go`
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/repository/page_revision.go`
- [ ] `go/internal/repository/page_attachment_reference.go`
- [x] `go/internal/repository/page_editor.go`
- [x] `go/internal/repository/space.go`
- [x] `go/internal/repository/space_member.go`
- [x] `go/internal/repository/topic.go`
- [x] `go/internal/repository/topic_member.go`
- [x] `go/internal/repository/attachment.go`

### ポリシー

- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`

### マークアップ

- [x] `go/internal/markup/markup.go`
- [x] `go/internal/markup/wikilink.go`
- [x] `go/internal/markup/attachment_extract.go`
- [x] `go/internal/markup/attachment_filter.go`
- [x] `go/internal/markup/batch.go`
- [x] `go/internal/markup/htmlutil.go`

### SQLクエリ

- [x] `go/db/queries/pages.sql`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/db/queries/page_revisions.sql`
- [x] `go/db/queries/page_editors.sql`
- [ ] `go/db/queries/page_attachment_references.sql`
- [x] `go/db/queries/attachments.sql`
- [x] `go/db/queries/spaces.sql`
- [x] `go/db/queries/space_members.sql`
- [x] `go/db/queries/topics.sql`
- [x] `go/db/queries/topic_members.sql`

### テストファイル

- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/repository/draft_page_test.go`
- [x] `go/internal/repository/page_revision_test.go`
- [x] `go/internal/repository/page_attachment_reference_test.go`
- [ ] `go/internal/repository/page_editor_test.go`
- [x] `go/internal/repository/space_test.go`
- [x] `go/internal/repository/space_member_test.go`
- [x] `go/internal/repository/topic_test.go`
- [x] `go/internal/repository/topic_member_test.go`
- [ ] `go/internal/repository/attachment_test.go`
- [x] `go/internal/policy/topic_test.go`
- [x] `go/internal/markup/markup_test.go`
- [x] `go/internal/markup/wikilink_test.go`
- [x] `go/internal/markup/attachment_extract_test.go`
- [x] `go/internal/markup/attachment_filter_test.go`
- [x] `go/internal/markup/batch_test.go`
- [x] `go/internal/markup/pipeline_integration_test.go`

### テストユーティリティ

- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `go/internal/testutil/page_revision_builder.go`
- [x] `go/internal/testutil/space_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/topic_builder.go`
- [x] `go/internal/testutil/topic_member_builder.go`

### 設定・その他

- [x] `go/.golangci.yml`
- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/security-guide.md`

## ファイルごとのレビュー結果

### `go/internal/model/space_member.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md) - ドメインID型の使用

**問題点・改善提案**:

- **[@go/CLAUDE.md#ドメインID型]**: `UserID` フィールドが `string` を使用している

  ```go
  // 現在のコード
  type SpaceMember struct {
      ID      SpaceMemberID
      SpaceID SpaceID
      UserID  string  // ← plain string
      // ...
  }
  ```

  ガイドラインでは「IDフィールドに `string` を使用しない」「外部キーにも専用型を使用: `UserID model.UserID`」と定められている。ただし、既存の `User` モデル自体が `ID string` を使用しており、`id.go` に `UserID` 型が未定義のため、コードベース全体で一貫した問題である。

  **修正案**:

  `id.go` に `type UserID string` と `String()` メソッドを追加し、`SpaceMember.UserID` を `UserID` 型に変更する。既存の `User.ID` や他のモデルの `UserID` フィールドの移行は別PRで対応可能。

  **対応方針**:
  - [ ] 本PRで `UserID` 型を追加し `SpaceMember.UserID` を修正する
  - [x] 別PRで `UserID` 型の導入を一括で行う（本PRでは現状維持）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/db/queries/page_attachment_references.sql` / `go/internal/repository/page_attachment_reference.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - space_idスコープ

**問題点・改善提案**:

- **[@go/docs/security-guide.md#スペースIDによるクエリスコープ]**: `CreatePageAttachmentReference` クエリに space_id の検証がない

  同ファイルの `ListPageAttachmentReferencesByPageID` と `DeletePageAttachmentReferencesByPageAndAttachmentIDs` は `pages` テーブルとのJOINで `space_id` を検証しているが、`CreatePageAttachmentReference` のINSERTにはスペース境界の検証がない。

  ```sql
  -- 現在のコード
  INSERT INTO page_attachment_references (attachment_id, page_id, created_at, updated_at)
  VALUES ($1, $2, $3, $4)
  RETURNING id;
  ```

  セキュリティガイドラインでは「テーブル自体に space_id がない場合は JOIN で検証する」と定められている。防御的プログラミングの一環として、INSERT時にもページと添付ファイルが同一スペースに属することをクエリレベルで検証すべき。

  **修正案**:

  ```sql
  -- name: CreatePageAttachmentReference :one
  INSERT INTO page_attachment_references (attachment_id, page_id, created_at, updated_at)
  SELECT $1, $2, $3, $4
  FROM pages p
  INNER JOIN attachments a ON a.space_id = p.space_id
  WHERE p.id = $2 AND a.id = $1 AND p.space_id = $5
  RETURNING id;
  ```

  リポジトリ側の `CreateBatch` メソッドにも `spaceID model.SpaceID` パラメータを追加する。

  **対応方針**:
  - [x] 修正案の通り、INSERTにspace_id検証を追加する
  - [ ] アプリケーション層（UseCase）で検証しているため現状維持
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/repository/page_editor_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テストカバレッジ

**問題点・改善提案**:

- **[@go/CLAUDE.md#テスト戦略]**: `UpdateLastPageModifiedAt` メソッドのテストが不足している

  `page_editor.go` には `FindOrCreate` と `UpdateLastPageModifiedAt` の2つのメソッドがあるが、テストは `FindOrCreate` のみをカバーしている。`UpdateLastPageModifiedAt` はページ公開時に呼ばれる重要なメソッドであり、テストが必要。

  **修正案**:

  ```go
  func TestPageEditorRepository_UpdateLastPageModifiedAt(t *testing.T) {
      t.Parallel()
      _, tx := testutil.SetupTx(t)
      // ... セットアップ ...
      // FindOrCreateで作成後、UpdateLastPageModifiedAtを呼び出し、
      // FindOrCreateで再取得して更新後の値を検証
  }
  ```

  **対応方針**:
  - [x] テストを追加する
  - [ ] 別PRで対応する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/repository/attachment_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テストカバレッジ

**問題点・改善提案**:

- **[@go/CLAUDE.md#テスト戦略]**: `FindByIDsAndSpace` メソッドのテストが不足している

  `attachment.go` には `ExistsByIDAndSpace`、`FindByIDAndSpace`、`FindByIDsAndSpace` の3つのメソッドがあるが、テストは最初の2つのみをカバーしている。`FindByIDsAndSpace` は添付ファイル参照同期で使用されるバッチ取得メソッドであり、テストが必要。

  **修正案**:

  ```go
  func TestAttachmentRepository_FindByIDsAndSpace(t *testing.T) {
      t.Parallel()
      _, tx := testutil.SetupTx(t)
      // ... 複数の添付ファイルを作成し、バッチ取得を検証
      // 異なるスペースの添付ファイルが含まれないことも検証
  }
  ```

  **対応方針**:
  - [x] テストを追加する
  - [ ] 別PRで対応する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/markup/htmlutil.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#コーディング規約](/workspace/go/CLAUDE.md) - エラーハンドリング

**問題点・改善提案**:

- **エラーハンドリング**: `renderContainerChildren` 関数で `html.Render` のエラーを `continue` で無視している

  ```go
  // 現在のコード
  if err := html.Render(&b, c); err != nil {
      continue
  }
  ```

  正常なDOMツリーに対しては実質的に発生しないエラーだが、デバッグ容易性の観点からログ出力を検討する価値がある。

  **修正案**:

  ```go
  if err := html.Render(&b, c); err != nil {
      slog.Warn("HTMLノードのレンダリングに失敗", "error", err)
      continue
  }
  ```

  **対応方針**:
  - [x] slog.Warnでログ出力を追加する
  - [ ] 現状のまま（実質的に発生しないエラーのため）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/markup/markup.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#コーディング規約](/workspace/go/CLAUDE.md) - エラーハンドリング

**問題点・改善提案**:

- **エラーハンドリング**: `RenderMarkdown` 関数で `md.Convert` のエラーが空文字列として返される

  ```go
  // 現在のコード
  if err := md.Convert([]byte(processed), &buf); err != nil {
      return ""
  }
  ```

  goldmarkの `Convert` は通常の入力に対してエラーを返すことはほぼないが、万が一の場合にデバッグが困難になる。

  **修正案**:

  ```go
  if err := md.Convert([]byte(processed), &buf); err != nil {
      slog.Warn("Markdown変換に失敗", "error", err)
      return ""
  }
  ```

  **対応方針**:
  - [x] slog.Warnでログ出力を追加する
  - [ ] 現状のまま（goldmarkのConvertは通常エラーを返さないため）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### テストヘルパーの重複: `createTestAttachment` / `createTestAttachmentForRepo`

**ステータス**: 要確認

**現状**:

`page_attachment_reference_test.go` の `createTestAttachment` と `attachment_test.go` の `createTestAttachmentForRepo` が、ほぼ同一の実装（`active_storage_blobs`、`active_storage_attachments`、`attachments` への直接INSERT）を持っている。

**提案**:

`testutil.NewAttachmentBuilder` を作成し、他のビルダー（`NewPageBuilder` 等）と同じパターンで添付ファイルのテストデータ作成を統合する。

**メリット**:

- テストコードの重複を排除
- 添付ファイル作成ロジックの一元管理
- 他のテストパッケージからも再利用可能

**トレードオフ**:

- `active_storage_*` テーブルはRails由来の構造であり、ビルダーの実装がやや複雑になる
- 現時点での使用箇所は2ファイルのみ

**対応方針**:

- [x] `testutil.NewAttachmentBuilder` を作成する
- [ ] 現状のまま（使用箇所が限定的なため）
- [ ] 別PRで対応する
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

### テストでのraw SQL使用

**ステータス**: 要確認

**現状**:

`page_test.go` で廃棄済みページの作成、`topic_test.go` で廃棄済みトピックの作成やトピックメンバーの作成にraw SQLを使用している。`topic_member_test.go` では `NewTopicMemberBuilder` を使用しているのに対し、`topic_test.go` 内ではraw SQLで `topic_members` に直接INSERTしている。

**提案**:

各ビルダーに `WithDiscarded()` メソッドを追加し、raw SQLを排除する。`topic_test.go` では既存の `NewTopicMemberBuilder` を使用する。

**メリット**:

- テストデータ作成パターンの統一
- ビルダーの機能拡充

**トレードオフ**:

- 既存のビルダー実装に変更が必要

**対応方針**:

- [x] ビルダーを拡充して統一する
- [ ] 現状のまま（テストの動作に問題はないため）
- [ ] 別PRで対応する
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 設計との整合性チェック

作業計画書（`docs/plans/1_doing/page-edit-go-migration.md`）に記載された本PRスコープの要件との整合性を確認した。

### 実装済みの要素

| 作業計画書の要件                               | 実装状況                                       |
| ---------------------------------------------- | ---------------------------------------------- |
| モデル: Page, DraftPage, PageRevision等        | 実装済み（全フィールド網羅、ドメインID型使用） |
| モデル: Space, SpaceMember, Topic, TopicMember | 実装済み                                       |
| モデル: Attachment, PageAttachmentReference    | 実装済み                                       |
| モデル: PageEditor                             | 実装済み                                       |
| リポジトリ: 作業計画書記載の全メソッド         | 実装済み（SearchPageLocationsを除く）          |
| ポリシー: TopicPolicy（CanUpdatePage等）       | 実装済み（Strategy パターンで4ロール実装）     |
| マークアップ: goldmark + bluemonday            | 実装済み                                       |
| マークアップ: Wikiリンク解析・HTML変換         | 実装済み                                       |
| マークアップ: 添付ファイルフィルター           | 実装済み                                       |
| マークアップ: 添付ファイルID抽出・アイキャッチ | 実装済み                                       |
| マークアップ: バッチ処理                       | 実装済み                                       |
| SQLクエリ: space_idスコープ                    | ほぼ全て実装済み                               |
| テストビルダー: 全モデル対応                   | 実装済み                                       |
| テスト: リポジトリ・ポリシー・マークアップ     | 実装済み（高いカバレッジ）                     |

### 未実装・次PRの要素

以下は本PRのスコープ外（モデル・リポジトリ層のみの実装であり、ハンドラー・ユースケース・テンプレートは後続PRで実装予定）:

- ハンドラー: `page/`, `draft_page/`, `page_location/`
- ユースケース: `update_draft_page.go`, `publish_page.go`
- テンプレート: 編集画面、フロントエンドアセット

### 乖離

作業計画書のデータ構造では全フィールドが `string` 型で記載されているが、実装ではドメインID型（`PageID`, `SpaceID` 等）を使用している。これはガイドライン準拠の改善であり、正しい乖離。

## 総合評価

**評価**: Approve

**総評**:

モデル・リポジトリ・ポリシー・マークアップの4層にわたる76ファイルの実装をレビューした。全体的に高品質であり、プロジェクトのガイドラインに一貫して準拠している。

**良かった点**:

1. **ドメインID型の徹底**: 新規モデル全てでドメインID型を使用。リポジトリ層でのstring⇔ドメインID型変換も正しく実装されている。
2. **セキュリティ**: SQLクエリのspace_idスコープがほぼ全クエリで実装されている。markupパッケージではbluemondayサニタイズ、DOM操作によるWikiリンク置換、url.PathEscapeによるURL構築など、多層的なXSS対策が施されている。
3. **アーキテクチャ準拠**: 全パッケージが3層アーキテクチャの依存ルールに従っている。リポジトリは `model` + `query` のみに依存、ポリシーは `model` のみ、マークアップは `model` + 外部ライブラリのみ。
4. **テスト品質**: 5351行のテストコードで高いカバレッジ。テーブル駆動テスト、`t.Parallel()`、SetupTxパターンが一貫して使用されている。XSSベクターのテストも含まれている。
5. **日本語コメント**: 全ファイルで日本語コメントが統一されている。
6. **WithTxパターン**: 全リポジトリでWithTxが実装されており、ユースケース層でのトランザクション管理に対応済み。

**指摘事項サマリー**:

- 必須対応: 0件
- 要確認: 6件（開発者の判断を要するもの）
  1. `space_member.go`: `UserID` が `string` 型（既存パターンとの一貫性 vs ガイドライン準拠）
  2. `page_attachment_references.sql`: `CreatePageAttachmentReference` のspace_id検証欠如
  3. `page_editor_test.go`: `UpdateLastPageModifiedAt` のテスト不足
  4. `attachment_test.go`: `FindByIDsAndSpace` のテスト不足
  5. `htmlutil.go`: エラーのサイレント無視
  6. `markup.go`: エラーのサイレント無視
- 設計改善提案: 2件
  1. テストヘルパーの重複統合
  2. テストでのraw SQL排除
