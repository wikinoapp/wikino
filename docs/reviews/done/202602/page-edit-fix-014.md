# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                         |
| -------------------------- | -------------------------------------------- |
| レビュー日                 | 2026-02-19                                   |
| 対象ブランチ               | page-edit-fix                                |
| ベースブランチ             | page-edit                                    |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md |
| 変更ファイル数             | 27 ファイル                                  |
| 変更行数（実装）           | +102 / -45 行（Go）、+166 / -97 行（templ）  |
| 変更行数（テスト）         | +202 / -6 行                                 |

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

### テンプレートファイル

- [x] `go/internal/templates/components/sidebar.templ`
- [x] `go/internal/templates/components/sidebar_templ.go`（自動生成）
- [x] `go/internal/templates/components/top_nav.templ`
- [x] `go/internal/templates/components/top_nav_templ.go`（自動生成）
- [x] `go/internal/templates/layouts/default.templ`
- [x] `go/internal/templates/layouts/default_templ.go`（自動生成）
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/internal/templates/pages/welcome/show.templ`
- [x] `go/internal/templates/pages/welcome/show_templ.go`（自動生成）

### テストファイル

- [x] `go/internal/viewmodel/page_test.go`
- [x] `go/internal/viewmodel/space_test.go`
- [x] `go/internal/viewmodel/topic_test.go`

### I18n

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### ドキュメント

- [x] `go/CLAUDE.md`
- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/3_done/202602/topic-viewmodel-refactoring.md`
- [x] `docs/reviews/done/202602/page-edit-fix-013.md`

## ファイルごとのレビュー結果

### `go/internal/templates/components/top_nav.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートガイド
- [@go/CLAUDE.md#国際化](/workspace/go/CLAUDE.md) - 国際化

**問題点・改善提案**:

- **パンくずリストのホームリンクからラベルが削除されている**: 以前は `templates.T(ctx, "breadcrumb_home")` というラベルが設定されていたが、変更後はアイコンのみ（`Label` が空文字列）になっている。これは `edit.templ` の `BreadcrumbItem` 定義で確認できる。ラベルが空の場合のレンダリングロジックは `top_nav.templ` に正しく実装されているが、`aria-label` が設定されていないため、アイコンのみのリンクはスクリーンリーダーでアクセスできない可能性がある。

  ```templ
  // edit.templ での使用
  {
      Path:     templates.HomePath(),
      IconName: "house-regular",
      // Label が空 — スクリーンリーダーへのアクセシビリティが不足
  }
  ```

  **修正案**:

  アイコンのみのリンクに `aria-label` を追加する。`top_nav.templ` のアイコンのみリンクの `<a>` タグに `aria-label` を追加するか、`BreadcrumbItem` に `AriaLabel` フィールドを追加する。

  ```templ
  // top_nav.templ: アイコンのみの場合
  if item.Label == "" && item.IconName != "" {
      <a class="hover:text-foreground transition-colors inline-flex items-center" href={ templ.SafeURL(item.Path) } aria-label={ item.AriaLabel }>
          @templates.Icon(item.IconName, "size-4 fill-gray-600")
      </a>
  }
  ```

  **対応方針**:
  - [x] `BreadcrumbItem` に `AriaLabel` フィールドを追加し、アイコンのみリンクで使用する
  - [ ] ラベルをホームリンクにも設定し直す（アイコンのみにしない）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/layouts/default.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートガイド

**問題点・改善提案**:

- **`HideSidebar` と `DefaultSidebarClosed` の関係が暗黙的**: `DefaultLayoutData` に `HideSidebar` と `DefaultSidebarClosed` の2つのフラグがある。`HideSidebar` が `true` の場合は `DefaultSidebarClosed` は無意味になる。現在の使用箇所（edit.go）では `HideSidebar` は設定されず `DefaultSidebarClosed: true` のみ設定されているため問題はないが、将来的に混乱する可能性がある。

  **修正案**:

  コメントで `HideSidebar` と `DefaultSidebarClosed` の関係を明記する、またはフィールドのドキュメントを追加する。

  ```go
  type DefaultLayoutData struct {
      Meta                 viewmodel.PageMeta
      Flash                *session.FlashMessage
      HideFooter           bool // trueの場合、フッターを非表示にする
      HideSidebar          bool // trueの場合、サイドバーを完全に非表示にする（DefaultSidebarClosedは無視される）
      DefaultSidebarClosed bool // trueの場合、サイドバーを閉じた状態で初期表示する
  }
  ```

  **対応方針**:
  - [x] コメントを追加して関係を明記する
  - [ ] 現状のまま（使用箇所が限定的なので問題なし）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/page/edit.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

**問題点・改善提案**:

1. **タイトル入力欄と本文テキストエリアからplaceholder属性が削除されている**: 以前は `placeholder={ templates.T(ctx, "page_edit_title_placeholder") }` と `placeholder={ templates.T(ctx, "page_edit_body_placeholder") }` が設定されていたが、変更後は削除されている。意図的な変更であれば問題ないが、ユーザビリティに影響する可能性がある。

   **修正案**:

   意図的な変更であれば問題なし。そうでなければplaceholderを復元する。

   **対応方針**:
   - [x] 意図的な変更のため、現状のまま
   - [ ] placeholderを復元する
   - [ ] その他（下の回答欄に記入）

   **回答**:

   ```
   （ここに回答を記入）
   ```

2. **`data-attr:disabled` 属性の妥当性**: 公開ボタンに `data-attr:disabled="$isSubmitting == true"` という属性が追加されている。これは標準のHTML属性ではなく、特定のJavaScriptフレームワーク（Alpine.jsなど）の構文に見える。対応するJSフレームワークがプロジェクトに導入されているか確認が必要。

   ```templ
   <button
       class="btn-primary rounded-full w-fit"
       data-attr:disabled="$isSubmitting == true"
       type="submit"
   >
   ```

   **修正案**:

   対応するJSフレームワークが導入されている場合は問題なし。導入されていない場合は属性を削除するか、適切な実装に変更する。

   **対応方針**:
   - [x] 対応するJSフレームワークが導入済みのため問題なし
   - [ ] 属性を削除する（将来の実装で対応）
   - [ ] その他（下の回答欄に記入）

   **回答**:

   ```
   （ここに回答を記入）
   ```

## 設計改善の提案

設計改善の提案はありません。

ViewModel パターンへのリファクタリング（`EditPageData` の個別フィールドを `viewmodel.Page`/`viewmodel.Space`/`viewmodel.Topic` に統合）は、ガイドラインに記載されたベストプラクティスに適合しており、良い設計変更です。`IconName` 型の `templates` から `viewmodel` への移動も、依存方向（templates → viewmodel）が正しく保たれています。

## 総合評価

**評価**: Comment

**総評**:

本PRは、ページ編集画面のViewModel層リファクタリングとサイドバー・レイアウト改善を行う良質なリファクタリングです。

**良かった点**:

- ハンドラーからデータ変換ロジックをViewModelに移動し、`go/CLAUDE.md` のガイドラインに準拠した設計になっている
- `viewmodel.Page`、`viewmodel.Topic`、`viewmodel.Space` の各ViewModelが適切に定義され、テストも網羅的に追加されている
- `AutofocusTitle()` のような派生的な判定をViewModelのメソッドとして提供している点がガイドラインの推奨パターンに合致
- テーブル駆動テストが適切に使用されており、正常系・境界値が網羅されている
- `SpaceHeader` → `Space` へのリネームにより、汎用性が向上
- `topicVisibilityIconName` を非公開関数にしたのは適切（内部実装の隠蔽）
- サイドバーコンポーネントのi18n対応が適切に行われている
- `go/CLAUDE.md` に「テンプレートデータ構造体とViewModelの関係」セクションが追加され、ガイドラインが充実した

**確認が必要な点**:

- パンくずリストのホームリンクでアイコンのみ表示時のアクセシビリティ（aria-label）
- タイトル・本文のplaceholder削除が意図的かどうか
- `data-attr:disabled` 属性の対応JSフレームワークの有無
