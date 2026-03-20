# コードレビュー: suggestion-7-1

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-19                       |
| 対象ブランチ               | suggestion-7-1                   |
| ベースブランチ             | suggestion-6-2                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 4 ファイル                       |
| 変更行数（実装）           | +213 / -1 行                     |
| 変更行数（テスト）         | +323 / -0 行                     |

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

## ファイルごとのレビュー結果

### `go/internal/usecase/apply_suggestion.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase のパターン、WithTx パターン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コメント、ログ出力
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - スペースIDによるクエリスコープ

**問題点・改善提案**:

- **[設計との整合性] LinkedPageIDsが再計算されていない**

  `publish_page.go` ではページを更新する際に `resolveAndCreateLinkedPages()` を呼び出してbody内のWikiリンクからLinkedPageIDsを再計算している（publish_page.go:101-106）。しかし `apply_suggestion.go` では旧ページのLinkedPageIDsをそのまま使用している:

  ```go
  // apply_suggestion.go:118-128（問題のあるコード）
  _, err = pageRepo.Update(ctx, repository.UpdatePageInput{
      // ...
      Body:          sp.Body,
      BodyHTML:      sp.BodyHTML,
      LinkedPageIDs: page.LinkedPageIDs, // ← 旧ページのLinkedPageIDs
      // ...
  })
  ```

  編集提案のbodyに新しいWikiリンクが含まれる場合や、既存のWikiリンクが削除されている場合、LinkedPageIDsが実際のbody内容と不整合になる。

  **修正案**:

  `publish_page.go` と同様に `resolveAndCreateLinkedPages()` を呼び出してLinkedPageIDsを再計算する。または、SuggestionPage作成時にLinkedPageIDsを事前計算して保存しておき、反映時にそれを使用する。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] `resolveAndCreateLinkedPages()` を呼び出してLinkedPageIDsを再計算する
  - [ ] SuggestionPage作成時にLinkedPageIDsを事前計算する方式にする
  - [ ] 初期リリースでは対応しない（理由を回答欄に記入）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  `pages.linked_page_ids`, `pages.featured_image_attachment_id` と対になるカラムを `draft_pages`, `suggestion_pages` にも
  追加したほうが良い気がしてきました。
  下書きを保存するときに `(draft_pages|suggestion_pages).linked_page_ids`,
  `(draft_pages|suggestion_pages).draft_pages.featured_image_attachment_id` を更新しておき、
  `apply_suggestion.go` を実行して編集提案を適用するときには `suggestion_pages` レコードに格納されている値を
  そのまま `pages` レコードに反映するイメージです。
  `apply_suggestion.go` でMarkdownの再変換を行うと対象のページ数が多いときに負荷が多くなりそうですし、
  下書きの時点でプレビュー的に表示したいときのためにも各テーブルにカラムを持っておくべきでは？と思いました。
  どう思いますか？懸念点などあれば教えてください
  ```

- **[設計との整合性] body/bodyHTMLのMarkdownパイプライン処理が省略されている**

  `publish_page.go` ではbodyの反映時に以下のパイプラインを実行している:
  1. Markdownレンダリング (`markup.RenderMarkdown`)
  2. Wikiリンク解析・リンク先ページの自動作成 (`resolveAndCreateLinkedPages`)
  3. bodyHTML内のWikiリンクを`<a>`タグに変換 (`markup.ReplaceWikilinks`)
  4. 添付ファイル参照の同期 (`syncAttachmentReferences`)
  5. アイキャッチ画像の抽出 (`extractFeaturedImageAttachmentID`)
  6. 添付ファイルフィルター (`markup.FilterAttachments`)
  7. スタンドアロン画像のラッピング (`markup.WrapStandaloneImageLinks`)

  `apply_suggestion.go` ではSuggestionPageに保存済みのbody/bodyHTMLをそのまま使用しており、これらの処理が行われない。作業計画書に「ベースが変わっていても強制的に上書き反映」とあるため意図的な簡略化の可能性もあるが、少なくともLinkedPageIDsとFeaturedImageAttachmentIDは正しく設定すべきである。

  **修正案**:

  `publish_page.go` と同じMarkdownパイプラインを `apply_suggestion.go` でも実行する。`resolveAndCreateLinkedPages` や `extractFeaturedImageAttachmentID` はパッケージ内のプライベート関数として既に利用可能。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] フルパイプライン処理を追加する（publish_page.goと同等）
  - [ ] LinkedPageIDsとFeaturedImageAttachmentIDのみ対応する
  - [ ] 初期リリースでは対応しない（理由を回答欄に記入）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  上で書いたように、編集提案を適用するタイミングではMarkdownパイプラインを実行しない設計にしたほうが良いかなと思いました。
  ```

- **[設計との整合性] FeaturedImageAttachmentIDが未設定**

  `UpdatePageInput` に `FeaturedImageAttachmentID` が渡されていない。`publish_page.go:156` では `extractFeaturedImageAttachmentID()` で抽出した値を設定している。未設定のまま更新すると、既存ページのアイキャッチ画像がnilに上書きされる可能性がある。

  ```go
  // apply_suggestion.go:118-128（FeaturedImageAttachmentIDが欠落）
  _, err = pageRepo.Update(ctx, repository.UpdatePageInput{
      ID:            sp.PageID,
      SpaceID:       input.SpaceID,
      TopicID:       page.TopicID,
      Title:         sp.Title,
      Body:          sp.Body,
      BodyHTML:      sp.BodyHTML,
      LinkedPageIDs: page.LinkedPageIDs,
      ModifiedAt:    now,
      PublishedAt:   &now,
      // FeaturedImageAttachmentID が欠落している
  })
  ```

  **修正案**:

  `publish_page.go` と同様に `extractFeaturedImageAttachmentID()` を呼び出すか、既存ページの `FeaturedImageAttachmentID` を引き継ぐ。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] `extractFeaturedImageAttachmentID()` を呼び出して新しいbodyから抽出する
  - [ ] 既存ページの `page.FeaturedImageAttachmentID` をそのまま引き継ぐ
  - [ ] 初期リリースでは対応しない（理由を回答欄に記入）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  上で書いたように、 `suggestion_pages.draft_pages.featured_image_attachment_id` の値をそのまま `pages` に渡す設計が良いかなと思いました。
  ```

### `go/internal/usecase/apply_suggestion_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略、テストヘルパー

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#テストヘルパーの使用] 既存のPageRevisionBuilderDBを使用していない**

  テストファイル末尾にローカルヘルパー `createPageRevisionViaRepoDB` が定義されている（apply_suggestion_test.go:307-323）が、`internal/testutil/page_revision_builder.go` に `PageRevisionBuilderDB` が既に存在する。既存のビルダーパターンを使用することでテストコードの一貫性が向上する。

  ```go
  // apply_suggestion_test.go:307-323（現在のコード）
  func createPageRevisionViaRepoDB(t *testing.T, q *query.Queries, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, pageID model.PageID) model.PageRevisionID {
      t.Helper()
      repo := repository.NewPageRevisionRepository(q)
      rev, err := repo.Create(context.Background(), repository.CreatePageRevisionInput{...})
      // ...
  }
  ```

  **修正案**:

  ```go
  // 既存の PageRevisionBuilderDB を使用
  pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
      WithSpaceID(spaceID).
      WithSpaceMemberID(spaceMemberID).
      WithPageID(pageID).
      Build()
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り既存の `PageRevisionBuilderDB` を使用する
  - [ ] 現状のまま（理由を回答欄に記入）
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

編集提案反映UseCaseの基本的な実装は、WithTxパターン、スペースIDによるクエリスコープ、エラーハンドリング、コメント記述など、既存のコーディング規約やアーキテクチャガイドラインに概ね従っている。テストも正常系・異常系を網羅しており、テストデータのセットアップも適切である。

ただし、`publish_page.go` で行われているMarkdownパイプライン処理（Wikiリンク解析、LinkedPageIDs計算、FeaturedImageAttachmentID抽出）が `apply_suggestion.go` では省略されており、ページ更新時にLinkedPageIDsやFeaturedImageAttachmentIDが正しく設定されない問題がある。これらはデータの整合性に影響するため、初期リリースの範囲であっても対応が望ましい。
