# コードレビュー: suggestion-7-1

## レビュー情報

| 項目                       | 内容                               |
| -------------------------- | ---------------------------------- |
| レビュー日                 | 2026-03-19                         |
| 対象ブランチ               | suggestion-7-1                     |
| ベースブランチ             | suggestion-6-2                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md   |
| 変更ファイル数             | 7 ファイル（実装3、ドキュメント4） |
| 変更行数（実装）           | +214 行                            |
| 変更行数（テスト）         | +366 行                            |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [ ] `go/internal/usecase/apply_suggestion.go`
- [x] `go/internal/testutil/suggestion_page_builder.go`

### テストファイル

- [ ] `go/internal/usecase/apply_suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-7-1-001.md`
- [x] `docs/reviews/done/202603/suggestion-7-1-002.md`
- [x] `docs/reviews/done/202603/suggestion-7-1-003.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/apply_suggestion.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase、WithTxパターン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - スペースIDによるクエリスコープ
- [@CLAUDE.md](/workspace/CLAUDE.md) - コメントのガイドライン
- 作業計画書 docs/plans/1_doing/suggestion.md - タスク7-1の仕様

**問題点・改善提案**:

- **[作業計画書との乖離] LinkedPageIDsとFeaturedImageAttachmentIDの暫定処理**: 125-126行目で `page.LinkedPageIDs` と `page.FeaturedImageAttachmentID` を使用している。作業計画書のタスク7-1bで「`page.LinkedPageIDs` の代わりに `sp.LinkedPageIDs` を使用し、`sp.FeaturedImageAttachmentID` を `UpdatePageInput` に渡す」と記載されており、現在の実装は暫定的にページの既存値を引き継いでいる。これはタスク7-1aと7-1bが未実装（`suggestion_pages` テーブルにまだこれらのカラムが存在しない）ためであり、意図的な暫定対応と理解できる。ただし、現状の実装では編集提案のbody内のWikiリンクや画像が反映されず、元ページのLinkedPageIDsとFeaturedImageAttachmentIDがそのまま残る。

  ```go
  // 現在のコード（125-126行目）
  LinkedPageIDs:             page.LinkedPageIDs,
  FeaturedImageAttachmentID: page.FeaturedImageAttachmentID,
  ```

  **修正案**: タスク7-1aと7-1bで対応予定の内容なので、現状のままで問題ないが、将来の修正漏れを防ぐためTODOコメントを追加することを推奨する。

  ```go
  // TODO: タスク7-1bでsp.LinkedPageIDsとsp.FeaturedImageAttachmentIDに変更する
  LinkedPageIDs:             page.LinkedPageIDs,
  FeaturedImageAttachmentID: page.FeaturedImageAttachmentID,
  ```

  **対応方針**:
  - [x] TODOコメントを追加する
  - [ ] 現状のまま（タスク7-1bで対応するため不要）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[作業計画書との乖離] 添付ファイル参照の同期が未実装**: `publish_page.go` ではMarkdownレンダリング、Wikiリンク解析、添付ファイル参照の同期（`syncAttachmentReferences`）、添付ファイルフィルター（`markup.FilterAttachments`）、スタンドアロン画像のラッピング（`markup.WrapStandaloneImageLinks`）を実行している。`apply_suggestion.go` ではSuggestionPageの `body_html` をそのまま使用しており、これらの処理をスキップしている。作業計画書の設計方針（「反映時のMarkdownパイプライン再実行を避けるため、write time（編集提案作成時）に計算して保存する」）に基づく意図的な設計と理解できるが、添付ファイル参照の同期（`page_attachment_references` テーブルの更新）は現在スキップされている。編集提案のbodyに添付ファイルが含まれる場合、反映後にページと添付ファイルの参照関係が不整合になる可能性がある。

  **修正案**: 編集提案反映時にも `syncAttachmentReferences` を呼び出す、または将来のタスクとして明示的に管理する。

  **対応方針**:
  - [ ] 現在のタスクで `syncAttachmentReferences` の呼び出しを追加する
  - [x] 将来のタスク（7-1bなど）で対応する（作業計画書に追記）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/apply_suggestion_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略、GetTestDB
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#並行テスト]**: テストケースに `t.Parallel()` が付与されていない。テストガイドでは「`t.Parallel()` で並行実行可能なテストを高速化」と記載されている。UseCaseテストでは `GetTestDB()` を使用し各テストが独立したデータを作成しているため、並行実行が可能。

  ```go
  // 現在のコード
  t.Run("正常系: 1つのページの編集提案を反映できる", func(t *testing.T) {
  ```

  **修正案**:

  ```go
  t.Run("正常系: 1つのページの編集提案を反映できる", func(t *testing.T) {
      t.Parallel()
  ```

  **対応方針**:
  - [x] 全テストケースに `t.Parallel()` を追加する
  - [ ] 現状のまま（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[@go/docs/testing-guide.md]**: 正常系テスト「1つのページの編集提案を反映できる」（26-133行目）で、SuggestionPageBuilderDBのデフォルト値（`title="テスト提案ページ"`, `body="テスト本文"`）に依存してアサーションしている（114-119行目）。テストの意図を明確にするために、テスト内で明示的に値を設定する方が望ましい。複数ページの正常系テスト（135-239行目）では `WithTitle`/`WithBody`/`WithBodyHTML` を明示的に呼び出しており一貫性がない。

  ```go
  // 現在のコード（72-77行目）- デフォルト値に依存
  testutil.NewSuggestionPageBuilderDB(t, db).
      WithSpaceID(spaceID).
      WithSuggestionID(suggestionID).
      WithPageID(pageID).
      WithPageRevisionID(pageRevisionID).
      Build()
  ```

  **修正案**:

  ```go
  testutil.NewSuggestionPageBuilderDB(t, db).
      WithSpaceID(spaceID).
      WithSuggestionID(suggestionID).
      WithPageID(pageID).
      WithPageRevisionID(pageRevisionID).
      WithTitle("提案タイトル").
      WithBody("提案本文").
      WithBodyHTML("<p>提案本文</p>").
      Build()
  ```

  **対応方針**:
  - [x] 明示的に値を設定してアサーションと一致させる
  - [ ] 現状のまま（デフォルト値のテストとして意味がある）
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

タスク7-1（編集提案反映のUseCase）の実装として、全体的に良い品質のコードである。

**良い点**:

- `publish_page.go` の既存パターン（Page更新 → PageRevision作成 → PageEditor更新 → TopicMember更新）に忠実に従っている
- WithTxパターンが正しく使用されている
- エラーハンドリングが一貫している
- 異常系テスト（ステータス違い3パターン、存在しないID）が網羅的
- 複数ページの反映テストも含まれている

**軽微な指摘**:

- LinkedPageIDsとFeaturedImageAttachmentIDの暫定処理はタスク7-1a/7-1bで対応予定であり、計画通り
- 添付ファイル参照の同期については方針の確認が必要
- テストの `t.Parallel()` とビルダーのデフォルト値依存は軽微な改善点
