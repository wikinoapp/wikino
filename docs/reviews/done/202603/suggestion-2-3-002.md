# コードレビュー: suggestion-2-3

## レビュー情報

| 項目                       | 内容                                  |
| -------------------------- | ------------------------------------- |
| レビュー日                 | 2026-03-17                            |
| 対象ブランチ               | suggestion-2-3                        |
| ベースブランチ             | page-title-rename                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md      |
| 変更ファイル数             | 11 ファイル（自動生成1、レビュー済1） |
| 変更行数（実装）           | +243 / -79 行（自動生成含む）         |
| 変更行数（テスト）         | +106 / -3 行                          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/topic/show.go`
- [x] `go/internal/usecase/get_topic_detail.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/templates/pages/topic/show.templ`
- [x] `go/internal/templates/pages/topic/show_templ.go`（自動生成ファイル、レビュー対象外）

### テストファイル

- [x] `go/internal/handler/topic/show_test.go`
- [x] `go/internal/usecase/get_topic_detail_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`（タスク2-3のチェックボックスを更新）
- [x] `docs/reviews/done/202603/suggestion-2-3-001.md`（前回レビュー、今回のレビュー対象外）

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計との整合性チェック

作業計画書のタスク **2-3** に記載された要件をすべて確認しました：

| 要件                                                           | 実装状況                                              |
| -------------------------------------------------------------- | ----------------------------------------------------- |
| `show.templ` に「ページ」「編集提案」のタブUIを追加            | ✅                                                    |
| タブコンポーネントの作成（必要に応じて）                       | ✅ （インラインの `showTabs` として実装、適切な判断） |
| タブクリック時に編集提案一覧画面にナビゲーション               | ✅                                                    |
| フィーチャーフラグ対応（アプリケーション内でフラグをチェック） | ✅                                                    |
| 翻訳ファイルにタブラベルのメッセージ追加                       | ✅                                                    |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク2-3（トピック詳細画面への「編集提案」タブ追加）が作業計画書の仕様通りに実装されています。

- **アーキテクチャ**: UseCase（Application層）でフィーチャーフラグを確認し、Handler経由でテンプレートに渡す設計は、3層アーキテクチャのガイドラインに正しく準拠している
- **フィーチャーフラグ**: 未ログインユーザーにはタブを表示しない実装が適切。`featureFlagRepo.IsEnabled` の呼び出しは `input.UserID != nil` で正しくガードされている
- **テンプレート**: `showTabs` を独立した `templ` コンポーネントとして切り出しており、可読性が高い。タブの表示は `SuggestionEnabled` による条件分岐で適切に制御されている
- **国際化**: `topic_show_tab_pages`、`topic_show_tab_suggestions` の翻訳キーが命名規則（`{page_name}_{detail}`）に従っており、ja/en 両方に description 付きで追加されている
- **テスト**: フィーチャーフラグ有効時・無効時の2ケースが追加されており、十分なカバレッジがある。既存テストのコンストラクタ更新も漏れなく行われている
- **PRサイズ**: 自動生成ファイルを除く手動実装は約50行程度で、テストコード約106行。適切なPRサイズ
