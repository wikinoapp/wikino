# コードレビュー: page-edit-fix

## レビュー情報

| 項目                         | 内容                                             |
| ---------------------------- | ------------------------------------------------ |
| レビュー日                   | 2026-02-19                                       |
| 対象ブランチ                 | page-edit-fix                                    |
| ベースブランチ               | page-edit                                        |
| 作業計画書（指定があれば）   | docs/plans/1_doing/page-edit-go-migration.md     |
| 変更ファイル数               | 28 ファイル                                      |
| 変更行数（実装）             | +270 / -142 行（手書きGo/templ。自動生成を除く） |
| 変更行数（テスト）           | +202 / -6 行                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/viewmodel/icon.go`
- [x] `go/internal/viewmodel/page.go`
- [x] `go/internal/viewmodel/space.go`
- [x] `go/internal/viewmodel/topic.go`
- [x] `go/internal/templates/helper.go`
- [x] `go/internal/templates/icons_custom.go`
- [x] `go/internal/templates/icons_phosphor.go`
- [x] `go/internal/templates/components/sidebar.templ`
- [x] `go/internal/templates/components/top_nav.templ`
- [x] `go/internal/templates/layouts/default.templ`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/welcome/show.templ`

### テストファイル

- [x] `go/internal/viewmodel/page_test.go`
- [x] `go/internal/viewmodel/space_test.go`
- [x] `go/internal/viewmodel/topic_test.go`

### 設定・翻訳

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### ドキュメント

- [x] `go/CLAUDE.md`
- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/3_done/202602/topic-viewmodel-refactoring.md`
- [x] `docs/reviews/done/202602/page-edit-fix-013.md`
- [x] `docs/reviews/done/202602/page-edit-fix-014.md`

### 自動生成ファイル（レビュー対象外）

- [x] `go/internal/templates/components/sidebar_templ.go`
- [x] `go/internal/templates/components/top_nav_templ.go`
- [x] `go/internal/templates/layouts/default_templ.go`
- [x] `go/internal/templates/pages/page/edit_templ.go`
- [x] `go/internal/templates/pages/welcome/show_templ.go`

## ファイルごとのレビュー結果

問題なし。すべてのファイルがガイドラインに適合しています。

### レビュー観点ごとの確認結果

#### 1. アーキテクチャ（依存関係）

**確認結果: 問題なし**

- `IconName` 型を `templates` パッケージから `viewmodel` パッケージに移動した変更は、アーキテクチャ上正しい方向
  - `viewmodel` は Presentation 層で、`model` に依存可能（OK）
  - `templates` は `viewmodel` に依存可能（同じ Presentation 層内、OK）
  - `viewmodel` は `templates` に依存していない（確認済み）
- `EditPageData` が `viewmodel.Page`, `viewmodel.Space`, `viewmodel.Topic` を使用する設計は、go/CLAUDE.md のガイドラインに合致

#### 2. テンプレートデータ構造体と ViewModel の関係

**確認結果: ガイドラインに適合**

- `EditPageData` のリファクタリングが正しく実施されている:
  - Before: `Title string`, `Body string`, `AutofocusTitle bool`, `PageNumber int32`, `SpaceName string` 等の個別フィールド
  - After: `Page viewmodel.Page`, `Space viewmodel.Space`, `Topic viewmodel.Topic` の ViewModel 構成
- `AutofocusTitle` がハンドラーの計算値からViewModelのメソッド `Page.AutofocusTitle()` に移動（ガイドライン通り）
- ハンドラーでの変換ロジックが `viewmodel.NewPageForEdit()` に集約されている

#### 3. ViewModel の実装

**確認結果: 問題なし**

- `viewmodel.Page`: 編集画面に必要な最小限のフィールド（Title, Body, Number）を持つ
- `viewmodel.NewPageForEdit()`: 下書きの有無に応じたフォールバックロジックを正しく実装
- `viewmodel.Space`: `SpaceHeader` から `Space` にリネーム（より汎用的な命名）
- `viewmodel.Topic`: 新規追加。Name, Number, IconName を持ち、`topicVisibilityIconName` を非公開関数に変更
- `viewmodel.IconName`: `templates.IconName` から移動。型の定義場所として viewmodel は適切

#### 4. テスト

**確認結果: 十分なカバレッジ**

- `page_test.go`: `NewPageForEdit` の4パターン（タイトルあり/なし × 下書きあり/なし）+ `AutofocusTitle` の2パターン
- `topic_test.go`: `NewTopic` の public/private 2パターン
- `space_test.go`: リネームに合わせたテスト名の更新
- テーブル駆動テスト、`t.Parallel()` の使用、命名規則すべてガイドライン準拠

#### 5. 国際化

**確認結果: 問題なし**

- サイドバー関連の新規翻訳キー3つ（`sidebar_toggle_label`, `sidebar_nav_label`, `sidebar_home`）が ja.toml と en.toml の両方に追加されている
- description フィールドがすべてのエントリに記載されている
- 命名規則 `{機能名}_{詳細}` に準拠

#### 6. セキュリティ

**確認結果: 問題なし**

- CSRFトークンが編集フォームに含まれている（`csrf_token` hidden input）
- Method Override パターン（`_method=PATCH`）が使用されている
- templ の自動エスケープが活用されている

#### 7. レイアウトの拡張

**確認結果: 問題なし**

- `DefaultLayoutData` に `HideFooter`, `HideSidebar`, `DefaultSidebarClosed` の3フィールドを追加
- 編集画面で `HideFooter: true`, `DefaultSidebarClosed: true` を設定（エディタ体験の改善）
- サイドバーコンポーネントが basecoat-css のコンポーネントを使用
- アクセシビリティ: `aria-label`, `aria-hidden` が適切に設定

#### 8. パンくずリスト

**確認結果: 問題なし**

- ホームアイコンのみの項目をサポートするため `AriaLabel` フィールドを追加
- ラベルが空でアイコンのみの場合に `aria-label` を設定（アクセシビリティ対応）
- `templates.IconName` から `viewmodel.IconName` への型変更が一貫して適用されている

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

このPRは、ページ編集画面のテンプレートデータ構造をViewModelベースにリファクタリングし、サイドバーコンポーネントを追加する変更です。

**良かった点**:

- go/CLAUDE.md に新しいガイドライン（テンプレートデータ構造体とViewModelの関係）を追加し、それに従った実装を行っている
- `IconName` 型の `templates` → `viewmodel` への移動は、依存関係の方向を正しく保つための適切な判断
- `SpaceHeader` → `Space` へのリネームにより、ヘッダー以外の場面でも使える汎用的な名前になった
- `topicVisibilityIconName` を非公開関数にし、`Topic` ViewModel 経由でアクセスする設計は、カプセル化として適切
- テストが十分に網羅されており、テーブル駆動テストのパターンに従っている
- アクセシビリティへの配慮（aria-label, aria-hidden）が適切
