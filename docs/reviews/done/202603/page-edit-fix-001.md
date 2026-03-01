# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                                   |
| -------------------------- | ------------------------------------------------------ |
| レビュー日                 | 2026-03-01                                             |
| 対象ブランチ               | page-edit-fix                                          |
| ベースブランチ             | page-edit                                              |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-draft-save-datastar.md    |
| 変更ファイル数             | 37 ファイル                                            |
| 変更行数（実装）           | 約 +175 / -180 行（自動生成ファイル除く、テスト除く）  |
| 変更行数（テスト）         | 約 +540 / -618 行（show_test.go の移植・新規作成含む） |

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
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
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

- [x] `go/internal/handler/draft_page/show_test.go`（新規：旧 page_link_list/show_test.go から移植 + 保存時刻テスト追加）
- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/handler/page_link_list/main_test.go`（削除）
- [x] `go/internal/handler/page_link_list/show_test.go`（削除）
- [x] `go/internal/viewmodel/link_list_test.go`

### 自動生成ファイル

- [x] `go/internal/templates/components/backlink_list_templ.go`
- [x] `go/internal/templates/components/draft_saved_time_templ.go`
- [x] `go/internal/templates/components/link_list_templ.go`
- [x] `go/internal/templates/pages/page/edit_templ.go`

### ドキュメント

- [x] `docs/plans/1_doing/page-edit-draft-save-datastar.md`

## ファイルごとのレビュー結果

### `go/internal/handler/draft_page/show.go`（新規）

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@CLAUDE.md#Pull Requestのガイドライン](/workspace/CLAUDE.md) - 実装とテストのセット化
- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- **[@CLAUDE.md#実装とテストのセット化]**: `show.go`（172 行）に対応するテストファイル `show_test.go` が存在しない

  **対応結果**:

  `draft_page/show_test.go` を新規作成。旧 `page_link_list/show_test.go` から 9 テストケースを移植し、保存時刻フラグメントに関する 2 テストケースを追加（合計 11 テスト）。全テストパス済み。
  - 未ログイン時の 401 レスポンス
  - 存在しないスペースでの 404
  - スペースメンバーでない場合の 404
  - 不正なページ番号での 404
  - リンクなし時の SSE レスポンス
  - リンクあり時の SSE レスポンスにリンク先が含まれること
  - 下書きのリンクが優先されること
  - ページネーションパラメータの反映
  - 下書きにリンク追加後の反映
  - **（追加）** 下書きが存在する場合に保存時刻フラグメントが SSE レスポンスに含まれること
  - **（追加）** 下書きが存在しない場合に保存時刻フラグメントが含まれないこと

### `go/internal/handler/page_link_list/show_test.go`（削除）

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@CLAUDE.md#Pull Requestのガイドライン](/workspace/CLAUDE.md) - 実装とテストのセット化

**対応結果**:

上記 `show.go` の対応と合わせて `draft_page/show_test.go` に全テストケースを移植済み。

## 設計改善の提案

### `go/internal/templates/pages/page/edit.templ`: `data-on:intersect` の URL 変更漏れの確認

**ステータス**: 確認済み（問題なし）

**確認結果**:

base ブランチ（`page-edit`）にも `data-on:intersect` は存在しないことを確認済み。作業計画書の記載が実態と異なるだけであり、実装自体は問題ない。

## 総合評価

**評価**: Approve

**総評**:

作業計画書に記載された機能要件（PATCH の 204 No Content 化、GET `draft_page` SSE エンドポイント新設、`page_link_list` の統合・廃止、保存時刻コンポーネントの新規作成、i18n 対応、TypeScript の簡素化）はすべて実装されており、設計方針に沿った実装になっている。

加えて、`model.SpaceIdentifier` 型の導入による型安全性向上の変更も含まれており、コードベースの品質向上に貢献している。

各ファイルのコーディング規約（slog の使用、日本語コメント、ガイドラインに沿った命名）も遵守されている。セキュリティ面でも、認証チェック、スペースメンバーの確認、TopicPolicy による認可チェックが適切に実装されている。

**レビュー指摘への対応結果**:

- `draft_page/show_test.go` を新規作成（旧 `page_link_list/show_test.go` の 9 テストケース移植 + 保存時刻フラグメントの 2 テストケース追加、合計 11 テスト）
- `data-on:intersect` は base ブランチにも存在しないことを確認済み（問題なし）
- 全テストパス（draft_page パッケージ: 20 テスト、失敗 0）
