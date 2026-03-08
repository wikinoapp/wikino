# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                                |
| -------------------------- | --------------------------------------------------- |
| レビュー日                 | 2026-03-01                                          |
| 対象ブランチ               | page-edit-fix                                       |
| ベースブランチ             | page-edit                                           |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-draft-save-datastar.md |
| 変更ファイル数             | 39 ファイル（自動生成含む）                         |
| 変更行数（実装）           | +110 / -148 行（自動生成・ドキュメント除く）        |
| 変更行数（テスト）         | +196 / -111 行                                      |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page/show.go`（新規）
- [x] `go/internal/handler/draft_page/update.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/handler/page/validator.go`
- [x] `go/internal/handler/page_backlink_list/show.go`
- [x] `go/internal/handler/page_link_list/handler.go`（削除）
- [x] `go/internal/handler/page_location/index.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/markup/batch.go`
- [x] `go/internal/markup/wikilink.go`
- [x] `go/internal/model/id.go`
- [x] `go/internal/model/space.go`
- [x] `go/internal/repository/space.go`
- [x] `go/internal/templates/components/backlink_list.templ`
- [x] `go/internal/templates/components/draft_saved_time.templ`（新規）
- [x] `go/internal/templates/components/link_list.templ`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/publish_page.go`
- [x] `go/internal/viewmodel/backlink_list.go`
- [x] `go/internal/viewmodel/link_list.go`
- [x] `go/internal/viewmodel/space.go`
- [x] `go/web/markdown-editor/markdown-editor.ts`

### テストファイル

- [x] `go/internal/handler/draft_page/show_test.go`（新規）
- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/viewmodel/link_list_test.go`

### 設定・その他

- [x] `go/internal/handler/page_link_list/main_test.go`（削除）
- [x] `docs/plans/1_doing/page-edit-draft-save-datastar.md`
- [x] `docs/reviews/page-edit-fix-001.md`（新規）

### 自動生成ファイル

- [x] `go/internal/templates/components/backlink_list_templ.go`
- [x] `go/internal/templates/components/draft_saved_time_templ.go`
- [x] `go/internal/templates/components/link_list_templ.go`
- [x] `go/internal/templates/pages/page/edit_templ.go`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書に記載された要件をすべて満たしており、設計との乖離は見られません。

**良かった点**:

- **SpaceIdentifier ドメイン型の導入**: `model.SpaceIdentifier` を `id.go` に追加し、`string` から専用型への移行をコードベース全体で一貫して実施。CLAUDE.md のドメインID型ガイドラインに沿った好ましい変更
- **page_link_list パッケージの統合**: リンク一覧と保存時刻を単一の SSE エンドポイント（`draft_page/show.go`）に統合し、パッケージを削除。不要なコードが残っていないことを確認済み
- **セキュリティ**: `show.go` で認証・認可（`TopicPolicy.CanUpdatePage`）が適切に実装されている。`space_id` によるクエリスコープも既存パターンを踏襲
- **テストの充実**: `show_test.go` が 695 行と包括的で、未認証・非メンバー・不正ページ番号・リンクあり/なし・下書き優先・ページネーション・保存時刻フラグメントの有無など、正常系・異常系を幅広くカバー
- **JS の簡素化**: `markdown-editor.ts` から JSON パース・DOM 操作・`savedAtEl` を削除し、`CustomEvent` ディスパッチのみに簡素化。サーバードリブン UI のパターンに適切に移行
- **i18n 対応**: `page_edit_draft_saved_time` を ja/en 両方に追加。`templ.Raw()` の使用は `modifiedAt.Format("15:04")` が数字とコロンのみを出力するため安全
- **アーキテクチャ準拠**: 3 層アーキテクチャの依存関係ルール、ハンドラーの標準ファイル名規則（`show.go`）、Datastar 属性構文（`data-on:` コロン区切り）すべてに準拠
