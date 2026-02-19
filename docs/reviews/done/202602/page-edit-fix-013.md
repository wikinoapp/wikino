# コードレビュー: page-edit-fix

## レビュー情報

| 項目                         | 内容                                              |
| ---------------------------- | ------------------------------------------------- |
| レビュー日                   | 2026-02-19                                        |
| 対象ブランチ                 | page-edit-fix                                     |
| ベースブランチ               | page-edit                                         |
| 作業計画書（指定があれば）   | docs/plans/1_doing/page-edit-go-migration.md      |
| 変更ファイル数               | 26 ファイル                                       |
| 変更行数（実装）             | +1160 / -366 行（自動生成ファイル含む）            |
| 変更行数（テスト）           | +202 / -6 行                                      |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/viewmodel/icon.go`
- [x] `go/internal/viewmodel/page.go`
- [x] `go/internal/viewmodel/space.go`
- [x] `go/internal/viewmodel/topic.go`
- [x] `go/internal/templates/helper.go`
- [x] `go/internal/templates/icons_phosphor.go`
- [x] `go/internal/templates/icons_custom.go`
- [x] `go/internal/templates/components/sidebar.templ`
- [x] `go/internal/templates/components/top_nav.templ`
- [x] `go/internal/templates/layouts/default.templ`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/welcome/show.templ`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/viewmodel/page_test.go`
- [x] `go/internal/viewmodel/space_test.go`
- [x] `go/internal/viewmodel/topic_test.go`

### 自動生成ファイル（レビュー対象外）

- [x] `go/internal/templates/components/sidebar_templ.go`
- [x] `go/internal/templates/components/top_nav_templ.go`
- [x] `go/internal/templates/layouts/default_templ.go`
- [x] `go/internal/templates/pages/page/edit_templ.go`
- [x] `go/internal/templates/pages/welcome/show_templ.go`

### ドキュメント

- [x] `go/CLAUDE.md`
- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/3_done/202602/topic-viewmodel-refactoring.md`

## ファイルごとのレビュー結果

### `go/internal/templates/icons_phosphor.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - コーディング規約（既存コードとの一貫性）

**問題点・改善提案**:

- **既存コードとの一貫性**: `house-regular` アイコンの `fill` 属性が `#000000` になっている。他のすべてのアイコンは `fill="currentColor"` を使用しており、これによりテキストカラーに追従する。`#000000` だとダークモードや異なるテキストカラーのコンテキストで表示が崩れる。

  ```go
  // 問題のあるコード
  "house-regular": `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="#000000" viewBox="0 0 256 256">...`
  ```

  **修正案**:

  ```go
  // 修正後のコード
  "house-regular": `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" viewBox="0 0 256 256">...`
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->

  - [x] 修正案の通り `fill="currentColor"` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/page/edit.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- 作業計画書 `docs/plans/1_doing/page-edit-go-migration.md` - 設計との整合性

**問題点・改善提案**:

1. **タイトル入力フィールドの `placeholder` 属性が削除されている**: 変更前は `placeholder={ templates.T(ctx, "page_edit_title_placeholder") }` が設定されていたが、新しい実装では削除されている。同様にbody textareaの `placeholder` も削除されている。意図的なUI変更であれば問題ないが、確認が必要。

   ```templ
   // 変更前（placeholderあり）
   <input
       ...
       placeholder={ templates.T(ctx, "page_edit_title_placeholder") }
   />

   // 変更後（placeholderなし）
   <input
       id="page_title"
       name="title"
       type="text"
       value={ data.Page.Title }
   />
   ```

   **修正案**:

   意図的な変更であれば問題なし。そうでなければ `placeholder` を復元する。

   **対応方針**:

   <!-- 開発者が回答を記入してください -->

   - [x] 意図的な変更のため対応不要
   - [ ] placeholderを復元する
   - [ ] その他（下の回答欄に記入）

   **回答**:

   ```
   （ここに回答を記入）
   ```

2. **`data-attr:disabled` 属性の確認**: 公開ボタンに `data-attr:disabled="$isSubmitting == true"` という非標準のHTML属性が使用されている。これはAlpine.jsまたはbasecoat-cssのカスタム属性バインディングと思われるが、このプロジェクトでこのパターンが使用されている前例があるか確認が必要。

   ```templ
   <button
       class="btn-primary rounded-full w-fit"
       data-attr:disabled="$isSubmitting == true"
       type="submit"
   >
   ```

   **修正案**:

   既存のフォーム送信パターンと一致していれば問題なし。

   **対応方針**:

   <!-- 開発者が回答を記入してください -->

   - [ ] 既存パターンと一致しているため対応不要
   - [x] 使用しているJSフレームワークの説明を回答欄に記入
   - [ ] その他（下の回答欄に記入）

   **回答**:

   ```
   Datastar https://data-star.dev/ の記法です。
   ```

## 設計改善の提案

設計改善の提案はありません。

変更全体の設計方針（ViewModel層へのロジック集約、`IconName`型の`viewmodel`パッケージへの移動、テンプレートデータ構造体でのViewModel使用）はアーキテクチャガイドラインに沿っており、適切です。

## 総合評価

**評価**: Comment

**総評**:

ViewModel層のリファクタリングとページ編集画面のUI改善が適切に行われています。主な良い点：

- **アーキテクチャガイドラインへの準拠**: `EditPageData` がプリミティブ値の羅列から `viewmodel.Page`, `viewmodel.Space`, `viewmodel.Topic` を構成要素とする構造に改善され、CLAUDE.mdに追加された新しいガイドラインに準拠
- **変換ロジックのViewModel集約**: ハンドラー内のtitle/bodyフォールバック判定やオートフォーカス判定がViewModelに移動し、ハンドラーがシンプルに
- **`IconName`型の適切な配置**: `templates`から`viewmodel`に移動し、depguardの依存方向ルール（viewmodel → templatesは禁止）に対応
- **テストの追加**: `page_test.go`, `topic_test.go` のテーブル駆動テストが追加され、ViewModelの変換ロジックがテストされている
- **国際化対応**: サイドバー関連の翻訳がja/en両方に追加されている

必須対応は1件（`house-regular`アイコンの`fill`属性）、確認事項は2件（placeholder削除の意図、`data-attr:disabled`パターン）です。
