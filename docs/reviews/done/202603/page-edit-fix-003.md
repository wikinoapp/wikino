# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                                                                        |
| -------------------------- | ------------------------------------------------------------------------------------------- |
| レビュー日                 | 2026-03-04                                                                                  |
| 対象ブランチ               | page-edit-fix                                                                               |
| ベースブランチ             | page-edit                                                                                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md, docs/plans/1_doing/page-edit-rails-go-diff.md |
| 変更ファイル数             | 40 ファイル                                                                                 |
| 変更行数（実装）           | 約 +1100 / -150 行（実装ファイル）                                                          |
| 変更行数（テスト）         | 約 +700 行（テストファイル）                                                                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/pages.sql`
- [x] `go/internal/handler/draft_page/show.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page_backlink_list/show.go`
- [x] `go/internal/handler/page_backlinks/handler.go`
- [x] `go/internal/handler/page_backlinks/show.go`
- [x] `go/internal/handler/page_link_list/handler.go`
- [x] `go/internal/handler/page_link_list/show.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/query/pages.sql.go`
- [x] `go/internal/repository/page.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/publish_page.go`
- [x] `go/internal/templates/components/backlink_list.templ`
- [x] `go/internal/templates/components/backlink_list_templ.go`
- [x] `go/internal/templates/components/link_list.templ`
- [x] `go/internal/templates/components/link_list_templ.go`
- [x] `go/internal/templates/components/page_backlink_list.templ`
- [x] `go/internal/templates/components/page_backlink_list_templ.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/viewmodel/link_list.go`
- [x] `go/web/style.css`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `rails/app/controllers/attachments/presigns/create_controller.rb`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/page_backlink_list/show_test.go`
- [x] `go/internal/handler/page_backlinks/main_test.go`
- [x] `go/internal/handler/page_backlinks/show_test.go`
- [x] `go/internal/templates/components/page_backlink_list_test.go`
- [x] `go/internal/usecase/auto_save_draft_page_test.go`
- [x] `go/internal/usecase/publish_page_test.go`
- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/middleware/reverse_proxy_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/1_doing/page-edit-rails-go-diff.md`
- [x] `docs/plans/3_done/202603/e2e-ci-fix.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/auto_save_draft_page_test.go`

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Query への依存ルール
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#重要なルール]**: テストのアサーションで `query` パッケージ（`q.FindPageEditorByPageAndSpaceMember`）を直接使用している。アーキテクチャガイドでは「Queryへの依存はRepositoryのみ」と定められており、テストコードでもこの原則に従うべき。同ファイルの既存テストでは `query` パッケージを直接使用したアサーションは行っておらず、一貫性もない。

  ```go
  // 問題のあるコード（テスト内）
  q := query.New(db)
  pageEditor, err := q.FindPageEditorByPageAndSpaceMember(ctx, query.FindPageEditorByPageAndSpaceMemberParams{...})
  ```

  **修正案**:

  `PageEditorRepository` のメソッドを使用してアサーションを行う。

  ```go
  pageEditorRepo := repository.NewPageEditorRepository(query.New(db))
  pageEditor, err := pageEditorRepo.FindByPageAndSpaceMember(ctx, ...)
  ```

  **対応方針**:
  - [x] Repository経由のアサーションに変更する
  - [ ] テストコードでは直接Query参照を許容する（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/publish_page_test.go`

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Query への依存ルール

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#重要なルール]**: `auto_save_draft_page_test.go` と同様、テストのアサーションで `q.FindPageEditorByPageAndSpaceMember(...)` を直接使用している。

  **修正案**:

  `auto_save_draft_page_test.go` と同様に、`PageEditorRepository` 経由のアサーションに変更する。

  **対応方針**:
  - [x] Repository経由のアサーションに変更する
  - [ ] テストコードでは直接Query参照を許容する（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/web/style.css`

**ステータス**: 対応不要（ユーザー判断）

**チェックしたガイドライン**:

- [@CLAUDE.md](/workspace/CLAUDE.md) - YAGNI 原則
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - YAGNI 原則

**問題点・改善提案**:

- **[@go/CLAUDE.md#YAGNI原則]**: `brand-green` と `brand-beige` の2つのカラーパレットが新規追加されているが、コードベース内のどこからも参照されていない。また、`brand-beige` の値は既存の `brand` パレットと完全に同一である。YAGNI原則に反する。

  **修正案**:

  未使用のカラーパレットを削除する。将来必要になった場合にそのタスクで追加すべき。

  **対応方針**:
  - [ ] 未使用のカラーパレットを削除する
  - [x] 今後使う予定があるため残す（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  ベージュをセカンダリに、緑をプライマリにした配色にしようかなと思っているので、残しておいてください
  ```

### `go/internal/templates/components/backlink_list.templ`

**ステータス**: 対応不要（現状維持）

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

**問題点・改善提案**:

- **[@go/docs/templ-guide.md#テンプレート関数の引数パターン]**: `BacklinkListCards` と `BacklinkListPagination` の新しいコンポーネントが `viewmodel.BacklinkList` を引数に取っているが、親コンポーネント `BacklinkList` の `data` パラメータには `Pagination` を使っていない場合の考慮が不要か確認が必要。`BacklinkListCards` はカードのみを描画し、`BacklinkListPagination` はページネーション部分のみを描画する設計は、SSEフラグメント更新に適しているが、`backlink_list.templ` と `page_backlink_list.templ` でほぼ同じパターンが重複している。

  **修正案**:

  現状のままでも問題ないが、将来的に共通化を検討する価値がある。現時点ではこのまま進める。

  **対応方針**:
  - [x] 現状のまま進める
  - [ ] 共通コンポーネントに統合する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/components/link_list.templ`

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

**問題点・改善提案**:

- **バグ修正の確認**: `linkListLoadMore` コンポーネントで、「もっと見る」ボタンのURLが `templates.PageDraftPagePath(...)` から `templates.PageLinkListPath(...)` に修正されている。これはこのブランチの主要なバグ修正であり、修正内容は正しい。ただし、このバグ修正に対する単体テストが `link_list_test.go` のような形で追加されていない。

  **修正案**:

  バグの再発を防ぐため、`link_list.templ` のレンダリング結果に正しいURL（`/link_list`）が含まれていることを確認するテストの追加を推奨する。

  **対応方針**:
  - [x] テストを追加する
  - [ ] 既存のハンドラーテストで間接的にカバーされるため不要
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `rails/app/controllers/attachments/presigns/create_controller.rb`

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@rails/docs/security-guide.md](/workspace/rails/docs/security-guide.md) - CSRF対策
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - セキュリティガイドライン

**問題点・改善提案**:

- **[@rails/docs/security-guide.md#CSRF対策]**: `skip_forgery_protection` でCSRF保護を完全に無効化している。コメントでGo版のページ編集画面からのリクエストにはGoのCSRFトークンが使用されるため、Rails側で検証できない旨が説明されており、理由は妥当である。ただし、この変更に対応するテストが含まれていない。Go版から呼び出される場合のCSRFトークンなしでのリクエストが成功することを確認するテストが望ましい。

  **修正案**:

  Go版のページ編集画面からpresignエンドポイントが正常に動作するシステムテストの追加を推奨する。

  **対応方針**:
  - [x] テストを追加する
  - [ ] 認証（`require_authentication`）でカバーされるため不要
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

### 作業計画書 `page-edit-rails-go-diff.md` との整合性

**差分1: page_editorsが作成されない**:

- **実装状況**: 修正済み。`resolveAndCreateLinkedPages` に `pageEditorRepo` と `spaceMemberID` を追加し、ページ新規作成時に `pageEditorRepo.FindOrCreate` を呼び出している。`AutoSaveDraftPageUsecase` と `PublishPageUsecase` の両方で対応済み。
- **テスト**: 自動保存・公開の両方で `page_editors` が作成されることを検証するテストが追加されている。
- **評価**: 計画通りに実装されている。

**差分2: 廃棄済みページと同名のWikiリンクでの失敗**:

- **実装状況**: 修正済み。`FindPageByTopicAndTitle` クエリから `AND discarded_at IS NULL` 条件を削除している。
- **テスト**: 廃棄済みページと同名のWikiリンクでの自動保存・公開が成功することを検証するテストが追加されている。`page_test.go` にも「廃棄済みページも取得できる」テストが追加されている。
- **評価**: 計画通りに実装されている。

### 計画書に記載のない追加変更

本ブランチには、差分修正（差分1・差分2）以外にも以下の変更が含まれている:

1. **リンク一覧SSEハンドラー（`page_link_list`）の新規追加**: 自動保存後にリンク一覧を動的更新するためのSSEエンドポイント
2. **ページレベルバックリンクSSEハンドラー（`page_backlinks`）の新規追加**: ページレベルのバックリンク一覧をSSEで動的更新するためのエンドポイント
3. **バックリンクからの自己ページ除外**: バックリンク一覧から編集中のページ自身とリンク先ページを除外するロジック
4. **リンクページの作成時に `published_at` を `NULL` に変更**: `CreateLinkedPage` クエリで `published_at` を作成日時ではなく `NULL` に設定するように変更
5. **Method Override対応**: リバースプロキシミドルウェアでPOSTリクエストをPATCH/PUT/DELETEパターンにマッチさせる対応
6. **テンプレートコンポーネントのリファクタリング**: バックリンク一覧・リンク一覧のSSEフラグメント更新対応

これらは `page-edit-go-migration.md` のスコープ内の改善として妥当だが、差分修正の作業計画書（`page-edit-rails-go-diff.md`）には記載されていない。PRのスコープが広いため、可能であれば差分修正とその他の改善を別PRに分けることが望ましい。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

作業計画書（`page-edit-rails-go-diff.md`）に記載された差分1・差分2の修正は計画通りに実装されており、対応するテストも適切に追加されている。差分修正の品質は高い。

一方で、このブランチには差分修正以外にも多くの変更（SSEハンドラーの新規追加、バックリンク除外ロジック、テンプレートリファクタリングなど）が含まれており、PRのスコープが広い。CLAUDE.mdのPRガイドラインでは「実装コード300行以下を目安」としているが、本ブランチの実装コードの変更量はこれを大幅に超えている。

指摘事項のサマリー:

- **テストでのQuery直接使用（2件）**: アーキテクチャガイドの「Queryへの依存はRepositoryのみ」ルールへの違反（テストコード内）
- **未使用CSSカラーパレット（1件）**: YAGNI原則違反
- **CSRF保護無効化のテスト不足（1件）**: セキュリティ変更のテストカバレッジ
- **バグ修正のテスト不足（1件）**: link_list.templの主要バグ修正に対するテスト
- **テンプレートコンポーネントの重複（1件）**: 将来的な共通化の検討
