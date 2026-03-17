# コードレビュー: suggestion-3-1

## レビュー情報

| 項目                       | 内容                                                      |
| -------------------------- | --------------------------------------------------------- |
| レビュー日                 | 2026-03-17                                                |
| 対象ブランチ               | suggestion-3-1                                            |
| ベースブランチ             | suggestion-2-3                                            |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md                          |
| 変更ファイル数             | 13 ファイル                                               |
| 変更行数（実装）           | +288 / -0 行（validator + usecase + repo + query + i18n） |
| 変更行数（テスト）         | +442 / -0 行                                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャ
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーション
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティ

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/validator/suggestion.go`
- [x] `go/internal/usecase/create_suggestion.go`
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/repository/page_revision.go`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/db/queries/page_revisions.sql`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/validator/suggestion_test.go`
- [x] `go/internal/usecase/create_suggestion_test.go`

### 設定・その他

- [x] `go/internal/query/draft_pages.sql.go` （自動生成）
- [x] `go/internal/query/page_revisions.sql.go` （自動生成）
- [x] `docs/plans/1_doing/suggestion.md` （タスクチェック更新のみ）

## ファイルごとのレビュー結果

### `go/internal/usecase/create_suggestion.go`

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase のトランザクションパターン
- [作業計画書](/workspace/docs/plans/1_doing/suggestion.md) - 設計との整合性

**問題点・改善提案**:

- **[設計との整合性]**: 作業計画書のテーブル設計セクションで「bodyはMarkdownで記述し、保存時にページと同じMarkdownパイプライン（Wikiリンク解決含む）でbody_htmlを生成する」と記述されているが、実装では `markup.RenderMarkdown(input.Body)` のみを使用しており、Wikiリンク解決（`markup.ReplaceWikilinks` 等）が含まれていない。`publish_page.go` では `RenderMarkdown` → `ReplaceWikilinks` → `FilterAttachments` → `WrapStandaloneImageLinks` の順で処理している

  ```go
  // 現在の実装（76行目）
  bodyHTML := markup.RenderMarkdown(input.Body)
  ```

  **修正案**:

  ```go
  // 案A: publish_page.go と同様のパイプラインを適用する
  bodyHTML := markup.RenderMarkdown(input.Body)
  // Wikiリンク解決等のパイプラインを追加

  // 案B: 初期リリースでは基本的なMarkdownレンダリングのみとし、
  // Wikiリンク解決は後続タスクで対応する（その場合、作業計画書にその旨を記載する）
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 案Aの通り、Wikiリンク解決を含むフルパイプラインを実装する
  - [ ] 案Bの通り、初期リリースでは基本レンダリングのみとし、作業計画書を更新する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/create_suggestion_test.go`

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストヘルパーの活用

**問題点・改善提案**:

- **[テストヘルパー]**: `createPageRevisionDB` ヘルパー関数（17-33行目）がリポジトリを介さず生SQLで直接 `page_revisions` テーブルに挿入している。`PageRevisionRepository.Create` が既に存在するため、リポジトリ経由でのテストデータ作成が可能。あるいは `testutil` パッケージに `NewPageRevisionBuilderDB` ヘルパーを作成するのが望ましい

  ```go
  // 現在の実装
  func createPageRevisionDB(t *testing.T, db *sql.DB, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, pageID model.PageID) model.PageRevisionID {
      t.Helper()
      now := time.Now()
      var id string
      err := db.QueryRowContext(
          context.Background(),
          `INSERT INTO page_revisions ...`,
      // ...
  ```

  **修正案**:

  ```go
  // 案A: 既存の PageRevisionRepository.Create を使用
  func createPageRevisionDB(t *testing.T, db *sql.DB, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, pageID model.PageID) model.PageRevisionID {
      t.Helper()
      q := query.New(db)
      repo := repository.NewPageRevisionRepository(q)
      rev, err := repo.Create(context.Background(), repository.CreatePageRevisionInput{
          SpaceID:       spaceID,
          SpaceMemberID: spaceMemberID,
          PageID:        pageID,
          Title:         "Revision Title",
          Body:          "Revision body",
          BodyHTML:      "<p>Revision body</p>",
      })
      if err != nil {
          t.Fatalf("ページリビジョン作成に失敗: %v", err)
      }
      return rev.ID
  }

  // 案B: testutil に NewPageRevisionBuilderDB を作成
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 案Aの通り、リポジトリ経由に変更する
  - [ ] 案Bの通り、testutil にビルダーを追加する
  - [ ] 現状のまま（テストヘルパーとしては動作するため）
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

タスク 3-1（編集提案作成のValidator・UseCase）の実装は全体的に高品質で、アーキテクチャガイドラインに忠実に従っている。

**良かった点**:

- 3層アーキテクチャの依存関係ルール（UseCase → Repository、Validator → Repository）が正しく守られている
- WithTx パターンによるトランザクション管理が適切に実装されている
- SQLクエリにすべて `space_id` 条件が含まれており、セキュリティガイドラインに準拠している
- バリデーションが形式チェック → 状態チェックの順序で早期リターンするパターンに従っている
- i18n の翻訳キーが日英両方に適切に追加されている
- テストが正常系・異常系を適切にカバーしている

**要確認事項**:

- Suggestion body の Markdown レンダリングで Wikiリンク解決を含めるかどうか（作業計画書との整合性）
- テストヘルパーの `createPageRevisionDB` をリポジトリ経由に変更するかどうか（軽微）
