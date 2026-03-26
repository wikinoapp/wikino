# コードレビュー: suggestion-11-1

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-25                       |
| 対象ブランチ               | suggestion-11-1                  |
| ベースブランチ             | suggestion-fix2                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 17 ファイル                      |
| 変更行数（実装）           | +715 / -651 行（リファクタ）     |
| 変更行数（テスト）         | +0 / -0 行                       |

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
- [x] `go/internal/templates/pages/suggestion/show.templ` - SubNav に置き換え
- [x] `go/internal/templates/pages/suggestion/show_templ.go` - 自動生成
- [x] `go/internal/templates/pages/suggestion/new.templ` - SubNav に置き換え
- [x] `go/internal/templates/pages/suggestion/new_templ.go` - 自動生成
- [x] `go/internal/templates/pages/suggestion_change/index.templ` - SubNav に置き換え
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go` - 自動生成
- [x] `go/internal/templates/pages/topic/show.templ` - SubNav に置き換え
- [x] `go/internal/templates/pages/topic/show_templ.go` - 自動生成

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md` - タスク 11-1 のチェックボックスを完了に更新
- [x] `docs/reviews/done/202603/suggestion-11-1-001.md` - 前回レビュー（done に移動済み）

## ファイルごとのレビュー結果

問題のあるファイルはありません。前回レビュー（suggestion-11-1-001）で指摘された以下の 2 点はいずれも修正済みです：

1. `show.templ` のインデント不整合 → 修正確認済み（70-85 行目）
2. `suggestion_change/index.templ` のフィールドアラインメント不整合 → 修正確認済み（83-98 行目）

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

前回レビューで指摘された 2 点（`show.templ` のインデント不整合、`suggestion_change/index.templ` のフィールドアラインメント不整合）がいずれも正しく修正されている。

**確認事項**:

- `TopicTabs` / `TopicTab` / `showTabs` の参照が完全に除去されていることを確認
- `brand-green-500`（旧 `TopicTabs` 固有のボーダー色）が `border-foreground` に統一されており、既存の `showTabs` 実装と一貫性がある
- 新規アイコン（`chats-circle-fill`, `files-fill`, `files-regular`）が正しく追加されている
- `file-regular` の SVG `fill` 属性バグ修正（`""` → `"currentColor"`）も維持されている
- すべてのページで `SubNav` コンポーネントの呼び出しパターンが統一されている
