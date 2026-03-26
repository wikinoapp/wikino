# コードレビュー: suggestion-11-1

## レビュー情報

| 項目                       | 内容                                 |
| -------------------------- | ------------------------------------ |
| レビュー日                 | 2026-03-25                           |
| 対象ブランチ               | suggestion-11-1                      |
| ベースブランチ             | suggestion-fix2                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md     |
| 変更ファイル数             | 16 ファイル                          |
| 変更行数（実装）           | +545 / -651 行（実質的にリファクタ） |
| 変更行数（テスト）         | +0 / -0 行                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/templates/components/sub_nav.templ` - 新規: 汎用サブナビゲーションコンポーネント
- [x] `go/internal/templates/components/sub_nav_templ.go` - 自動生成
- [x] `go/internal/templates/components/topic_tabs.templ` - 削除: 旧タブコンポーネント
- [x] `go/internal/templates/components/topic_tabs_templ.go` - 削除: 自動生成
- [x] `go/internal/templates/icons_phosphor.go` - アイコン追加・修正
- [x] `go/internal/templates/pages/suggestion/index.templ` - SubNav に置き換え
- [x] `go/internal/templates/pages/suggestion/index_templ.go` - 自動生成
- [ ] `go/internal/templates/pages/suggestion/show.templ` - SubNav に置き換え
- [x] `go/internal/templates/pages/suggestion/show_templ.go` - 自動生成
- [x] `go/internal/templates/pages/suggestion/new.templ` - SubNav に置き換え
- [x] `go/internal/templates/pages/suggestion/new_templ.go` - 自動生成
- [ ] `go/internal/templates/pages/suggestion_change/index.templ` - SubNav に置き換え
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go` - 自動生成
- [x] `go/internal/templates/pages/topic/show.templ` - SubNav に置き換え
- [x] `go/internal/templates/pages/topic/show_templ.go` - 自動生成

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md` - タスク 11-1 のチェックボックスを完了に更新

## ファイルごとのレビュー結果

### `go/internal/templates/pages/suggestion/show.templ`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

**問題点・改善提案**:

- **インデントの不整合**: `SubNav` コンポーネントの呼び出し（70-85行目）で、スライス要素のインデントが親の `@components.SubNav(` と揃っていない。他のファイル（`suggestion/index.templ` 60-74行目、`topic/show.templ` 55-69行目）では正しくインデントされている

  ```templ
  // 現在のコード（show.templ 70-85行目）
  						@components.SubNav([]components.SubNavItem{
  						{
  							Label:          templates.T(ctx, "suggestion_show_conversation_tab"),
  							...
  						},
  						{
  							...
  						},
  					})
  ```

  **修正案**:

  ```templ
  // スライス要素を1段深くインデント（他ファイルと統一）
  						@components.SubNav([]components.SubNavItem{
  							{
  								Label:          templates.T(ctx, "suggestion_show_conversation_tab"),
  								Path:           string(templates.SuggestionShowPath(data.Space.Identifier.String(), data.Suggestion.Number)),
  								IconName:       "chats-circle-regular",
  								ActiveIconName: "chats-circle-fill",
  								IsActive:       true,
  							},
  							{
  								Label:          templates.T(ctx, "suggestion_show_changed_pages_tab"),
  								Path:           string(templates.SuggestionChangesPath(data.Space.Identifier.String(), data.Suggestion.Number)),
  								IconName:       "files-regular",
  								ActiveIconName: "files-fill",
  								Badge:          fmt.Sprintf("%d", len(data.SuggestionPages)),
  							},
  						})
  ```

  **対応方針**:
  - [x] 修正案の通りインデントを統一する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/suggestion_change/index.templ`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

**問題点・改善提案**:

- **構造体フィールドのアラインメント不整合**: 1つ目の `SubNavItem`（84-88行目）で `ActiveIconName:` フィールドのアラインメントが他フィールド（`Label:`, `Path:`, `IconName:`）と揃っていない。2つ目の `SubNavItem`（90-97行目）では正しく揃っている

  ```templ
  // 現在のコード（1つ目の SubNavItem）
  							{
  								Label:    templates.T(ctx, "suggestion_show_conversation_tab"),
  								Path:     string(...),
  								IconName: "chats-circle-regular",
  								ActiveIconName: "chats-circle-fill",
  							},
  ```

  **修正案**:

  ```templ
  // フィールドのアラインメントを統一
  							{
  								Label:          templates.T(ctx, "suggestion_show_conversation_tab"),
  								Path:           string(templates.SuggestionShowPath(data.Space.Identifier.String(), data.Suggestion.Number)),
  								IconName:       "chats-circle-regular",
  								ActiveIconName: "chats-circle-fill",
  							},
  ```

  **対応方針**:
  - [x] 修正案の通りアラインメントを統一する
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

`TopicTabs` と各画面の `showTabs` を汎用的な `SubNav` コンポーネントに統合するリファクタリングが適切に行われている。

**良い点**:

- `SubNav` コンポーネントの設計が柔軟（`Label`, `Path`, `IconName`, `ActiveIconName`, `IsActive`, `Badge` を持つ `SubNavItem` スライス）
- 旧 `TopicTabs` と旧 `showTabs` の参照が完全に除去されている
- アイコンの統一（会話タブ: `chats-circle-*`, 変更ページタブ: `files-*`）が適切
- `file-regular` の SVG `fill` 属性が `""` から `"currentColor"` に修正されている（バグ修正）
- ビルドが正常に通ることを確認済み

**軽微な指摘**:

- `show.templ` のインデント不整合と `suggestion_change/index.templ` のフィールドアラインメント不整合の2点。いずれもフォーマットの問題であり機能には影響しない
