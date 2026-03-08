# コードレビュー: go-page-edit-fix

## レビュー情報

| 項目                       | 内容             |
| -------------------------- | ---------------- |
| レビュー日                 | 2026-03-08       |
| 対象ブランチ               | go-page-edit-fix |
| ベースブランチ             | go-page-edit     |
| 作業計画書（指定があれば） | なし             |
| 変更ファイル数             | 31 ファイル      |
| 変更行数（実装）           | +274 / -158 行   |
| 変更行数（テスト）         | +0 / -0 行       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
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

- [x] `rails/app/components/headers/global_component.rb`
- [ ] `rails/app/components/headers/global_component.html.erb`
- [x] `rails/app/components/breadcrumbs/space_component.rb`
- [x] `rails/app/components/breadcrumbs/space_component.html.erb`
- [x] `rails/app/components/breadcrumbs/topic_component.rb`
- [x] `rails/app/components/breadcrumbs/topic_component.html.erb`
- [x] `rails/app/components/sidebar_component.html.erb`
- [ ] `rails/app/views/pages/show_view.html.erb`
- [ ] `rails/app/views/spaces/settings/attachments/index_view.html.erb`
- [ ] `rails/app/views/spaces/settings/deletions/new_view.html.erb`
- [ ] `rails/app/views/spaces/settings/exports/new_view.html.erb`
- [ ] `rails/app/views/spaces/settings/exports/show_view.html.erb`
- [ ] `rails/app/views/spaces/settings/general/show_view.html.erb`
- [ ] `rails/app/views/spaces/settings/show_view.html.erb`
- [ ] `rails/app/views/spaces/show_view.html.erb`
- [ ] `rails/app/views/topics/new_view.html.erb`
- [ ] `rails/app/views/topics/settings/deletions/new_view.html.erb`
- [ ] `rails/app/views/topics/settings/general/show_view.html.erb`
- [ ] `rails/app/views/topics/settings/show_view.html.erb`
- [ ] `rails/app/views/topics/show_view.html.erb`
- [ ] `rails/app/views/trash/show_view.html.erb`

### ドキュメント

- [x] `docs/plans/1_doing/page-edit-go-rollout.md`
- [x] `docs/specs/page/edit.md`

## ファイルごとのレビュー結果

### `rails/app/components/headers/global_component.html.erb`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

**問題点・改善提案**:

- **Go版とRails版のヘッダーレイアウトの不一致**: Rails版では左右の要素を `flex-1` に変更しているが、Go版の `top_nav.templ` は `w-[32px] flex-none` のまま。Rails版では `content_screen` に基づく `max_width_class_name` が常に適用されるが、Go版では `MaxWidthClass` が空の場合は `mx-auto w-fit` にフォールバックする。左右の要素のflex挙動が異なるため、パンくずリストの中央配置の計算方法が微妙に異なる。

  **Go版**（`top_nav.templ`）:

  ```html
  <div class="w-[32px] flex-none">
    <!-- 左: 固定32px -->
    <div class="flex-1">
      <!-- 中央: 残りスペース -->
      <div class="w-[32px] flex-none"><!-- 右: 固定32px --></div>
    </div>
  </div>
  ```

  **Rails版**（`global_component.html.erb`）:

  ```html
  <div class="flex-1">
    <!-- 左: flex-1 -->
    <div class="max-w-... w-full px-4">
      <!-- 中央: max-width指定 -->
      <div class="flex-1"><!-- 右: flex-1 --></div>
    </div>
  </div>
  ```

  **修正案**:

  Go版の `top_nav.templ` も左右を `flex-1` に揃え、中央のdivから `flex-1` を除去してRails版と同じレイアウト構造にする。

  **対応方針**:
  - [x] Go版をRails版と同じ `flex-1` レイアウトに揃える
  - [ ] 現状のまま（視覚的な差異は許容範囲内）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `signed_in?` と `current_user.present?` の不一致（複数ビューファイル）

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - 既存コードとの一貫性

**問題点・改善提案**:

- **`signed_in?` と `current_user.present?` の使い分けが不統一**: パンくずリストコンポーネントに `signed_in:` パラメータを渡す際、ビューによって `signed_in?` と `current_user.present?` が混在している。

  `signed_in?` を使用:
  - `spaces/show_view.html.erb`
  - `topics/show_view.html.erb`
  - `pages/show_view.html.erb`

  `current_user.present?` を使用:
  - `spaces/settings/*.html.erb`（6ファイル）
  - `topics/settings/*.html.erb`（3ファイル）
  - `topics/new_view.html.erb`
  - `trash/show_view.html.erb`

  設定画面やトピック新規作成画面は認証が必須なので `current_user.present?` は常に `true` になり、動作上の問題はない。しかしセマンティクスとしては `signed_in?` で統一するのが望ましい。

  **修正案**:

  すべてのビューで `signed_in?` に統一する。

  **対応方針**:
  - [x] すべて `signed_in?` に統一する
  - [ ] 現状のまま（動作に影響がないため）
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

5つのコミットで以下の変更が行われている:

1. **仕様書の更新**: Go版への完全移行を反映した仕様書更新。パスの `/go/` プレフィックス除去、Rails版コード削除の記載、新しい採用しなかった方針の追加が適切に行われている
2. **MaxWidthClass追加**: Go版TopNavにMaxWidthClassを追加し、パンくずリストの幅をコンテンツ幅に合わせる対応。構造体ベースのデータ渡しパターンに従っている
3. **content_screenパラメータ**: Rails版GlobalComponentにcontent_screenパラメータを追加し、既存のContentScreen型を活用。デフォルト値の設定により既存の呼び出し元への影響がない
4. **z-indexの調整**: サイドバーのz-index修正
5. **未ログイン時のホームアイコン非表示**: Rails版パンくずリストコンポーネントにsigned_inパラメータを追加

全体として、ガイドラインに従った適切な実装。指摘事項は2件（Go/Rails間のレイアウト構造の不一致、`signed_in?`/`current_user.present?` の不統一）で、いずれも軽微な一貫性の問題であり必須対応ではない。
