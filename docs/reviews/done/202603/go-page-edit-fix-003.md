# コードレビュー: go-page-edit-fix

## レビュー情報

| 項目                       | 内容             |
| -------------------------- | ---------------- |
| レビュー日                 | 2026-03-08       |
| 対象ブランチ               | go-page-edit-fix |
| ベースブランチ             | go-page-edit     |
| 作業計画書（指定があれば） | なし             |
| 変更ファイル数             | 32 ファイル      |
| 変更行数（実装）           | +454 / -164 行   |
| 変更行数（テスト）         | +0 / -0 行       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails 版の開発ガイド
- [@rails/docs/architecture-guide.md](/workspace/rails/docs/architecture-guide.md) - アーキテクチャガイド

## 変更ファイル一覧

### 実装ファイル（Go版）

- [x] `go/internal/templates/components/top_nav.templ`
- [x] `go/internal/templates/components/top_nav_templ.go`（自動生成）
- [x] `go/internal/templates/pages/draft_page/index.templ`
- [x] `go/internal/templates/pages/draft_page/index_templ.go`（自動生成）
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/internal/templates/pages/page_move/new.templ`
- [x] `go/internal/templates/pages/page_move/new_templ.go`（自動生成）

### 実装ファイル（Rails版）

- [x] `rails/app/components/breadcrumbs/space_component.html.erb`
- [x] `rails/app/components/breadcrumbs/space_component.rb`
- [x] `rails/app/components/breadcrumbs/topic_component.html.erb`
- [x] `rails/app/components/breadcrumbs/topic_component.rb`
- [x] `rails/app/components/headers/global_component.html.erb`
- [x] `rails/app/components/headers/global_component.rb`
- [x] `rails/app/components/sidebar_component.html.erb`
- [x] `rails/app/views/pages/show_view.html.erb`
- [x] `rails/app/views/spaces/settings/attachments/index_view.html.erb`
- [x] `rails/app/views/spaces/settings/deletions/new_view.html.erb`
- [x] `rails/app/views/spaces/settings/exports/new_view.html.erb`
- [x] `rails/app/views/spaces/settings/exports/show_view.html.erb`
- [x] `rails/app/views/spaces/settings/general/show_view.html.erb`
- [x] `rails/app/views/spaces/settings/show_view.html.erb`
- [x] `rails/app/views/spaces/show_view.html.erb`
- [x] `rails/app/views/topics/new_view.html.erb`
- [x] `rails/app/views/topics/settings/deletions/new_view.html.erb`
- [x] `rails/app/views/topics/settings/general/show_view.html.erb`
- [x] `rails/app/views/topics/settings/show_view.html.erb`
- [x] `rails/app/views/topics/show_view.html.erb`
- [x] `rails/app/views/trash/show_view.html.erb`

### ドキュメント

- [x] `docs/plans/1_doing/page-edit-go-rollout.md`
- [x] `docs/reviews/done/202603/go-page-edit-fix-002.md`
- [x] `docs/specs/page/edit.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

本PRは、Go版・Rails版間のトップナビゲーション（パンくずリスト）レイアウトの統一と、未ログイン時のパンくずリストにおけるホームアイコンの非表示対応を行う変更です。主な変更は以下の3点です：

1. **トップナビゲーションのレイアウト調整**: サイドバートグルボタンとパンくずリスト周辺の `flex` レイアウトを `w-[32px] flex-none` から `flex-1` に変更し、パンくずリストの配置が `max-width` ベースの制約（`MaxWidthClass`）に従うように修正。Go版とRails版の両方で一貫して適用されている
2. **未ログイン時のパンくずリストのホームアイコン非表示**: `signed_in` パラメータを `SpaceComponent`/`TopicComponent` に追加し、未ログイン時にホームアイコンを非表示にする。全呼び出し箇所（21ファイル）が一貫して更新されている
3. **サイドバーの z-index 調整**: `!z-sidebar` クラスの追加

変更は機械的で一貫性があり、コーディング規約に従っています。Rails版では `private :signed_in` + `alias_method :signed_in?` のパターン、Sorbet型注釈（`sig`）が正しく使用されています。Go版では templ テンプレートガイドに準拠したデータ構造体パターンが使用されています。テストコードの変更はありませんが、UIレイアウトの調整が主であり、既存テストのカバレッジ内と判断します。
