# コードレビュー: sidebar-sync-5-1

## レビュー情報

| 項目                       | 内容                               |
| -------------------------- | ---------------------------------- |
| レビュー日                 | 2026-03-07                         |
| 対象ブランチ               | sidebar-sync-5-1                   |
| ベースブランチ             | sidebar-sync                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/sidebar-sync.md |
| 変更ファイル数             | 31 ファイル                        |
| 変更行数（実装）           | +64 / -92 行                       |
| 変更行数（テスト）         | +0 / -0 行                         |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版の開発ガイド
- [@rails/docs/architecture-guide.md](/workspace/rails/docs/architecture-guide.md) - アーキテクチャガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `rails/app/assets/stylesheets/application.css`
- [x] `rails/app/components/footers/global_component.html.erb`
- [x] `rails/app/components/footers/global_component.rb`
- [x] `rails/app/components/layouts/column1_component.html.erb`
- [x] `rails/app/components/sidebar_component.html.erb`
- [x] `rails/app/views/home/show_view.html.erb`
- [x] `rails/app/views/pages/show_view.html.erb`
- [x] `rails/app/views/profiles/show_view.html.erb`
- [x] `rails/app/views/search/show_view.html.erb`
- [x] `rails/app/views/settings/account/deletions/new_view.html.erb`
- [x] `rails/app/views/settings/emails/show_view.html.erb`
- [x] `rails/app/views/settings/profiles/show_view.html.erb`
- [x] `rails/app/views/settings/show_view.html.erb`
- [x] `rails/app/views/settings/two_factor_auths/new_view.html.erb`
- [x] `rails/app/views/settings/two_factor_auths/recovery_codes/show_view.html.erb`
- [x] `rails/app/views/settings/two_factor_auths/show_view.html.erb`
- [x] `rails/app/views/spaces/new_view.html.erb`
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

### 設定・その他

- [x] `docs/plans/1_doing/sidebar-sync.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っています。

## 設計との整合性チェック

作業計画書タスク5-1の要件との整合性:

| 要件                                                                   | 状態 |
| ---------------------------------------------------------------------- | ---- |
| SidebarComponentからフッターリンクを削除                               | ✅   |
| Column1ComponentにGo版と同じフッターを追加                             | ✅   |
| フッター内容: 利用規約・プライバシーポリシー・著作権表示               | ✅   |
| Go版と同じHTML構造・スタイリング                                       | ✅   |
| I18n翻訳キー（`nouns.terms_of_service`, `nouns.privacy_policy`）が存在 | ✅   |

Go版 `footer.templ` との構造比較:

- CSS クラス構成が一致（`mt-auto`, `px-4`, `pt-[3rem]`, `pb-[calc(...)]` 等）
- リンクスタイル `link-muted text-sm` が一致
- 著作権表示 `&copy; 2025-2026` + `Wikino` リンクが一致
- レスポンシブ対応（`order-last md:order-first`, `basis-full md:basis-auto`）が一致
- `--bottom-nav-max-height` CSS変数がRails版にも追加済み

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク5-1の要件通りに実装されており、Go版のフッターとHTML構造・スタイリングが正確に一致しています。

良かった点:

- `Footers::GlobalComponent` の `signed_in` / `class_name` パラメータを完全に削除し、不要なロジックをクリーンに除去している
- 25ファイルのビュー修正が漏れなく行われている（`layout.with_footer` を使う全ビューが更新済み）
- `pages/edit_view.html.erb` はフッターを使用しないビューとして正しく除外されている
- サイドバーからフッターリンクだけでなく末尾の `<hr>` 区切り線も正しく削除されている
- 作業計画書のタスクリストとコンポーネント構成図も正しく更新されている
