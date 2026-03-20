# コードレビュー: suggestion-9-2

## レビュー情報

| 項目                       | 内容                                     |
| -------------------------- | ---------------------------------------- |
| レビュー日                 | 2026-03-20                               |
| 対象ブランチ               | suggestion-9-2                           |
| ベースブランチ             | develop                                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md         |
| 変更ファイル数             | 13 ファイル                              |
| 変更行数（実装）           | 約 +120 / -33 行（自動生成ファイル除く） |
| 変更行数（テスト）         | 約 +162 / -1 行                          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャの依存関係ルール
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/get_page_detail.go`

### テストファイル

- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`
- [x] `go/internal/usecase/get_page_detail_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 設計との整合性チェック

### タスク9-2の仕様確認

作業計画書のタスク9-2で定義されている要件:

- [x] ページ編集画面（`internal/handler/page/edit.go`）で `DraftPage.SuggestionPageID` がNOT NULLの場合の表示切り替え
- [x] 「トピックに公開」ボタンを非表示 → 「編集提案を更新」ボタンを表示
- [x] 「編集提案 #xxx のページを編集中です」メッセージ表示
- [x] `internal/templates/pages/page/edit.templ` の更新
- [x] 翻訳ファイル（ja.toml, en.toml）にメッセージ追加
- [x] テストの追加

仕様との乖離はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク9-2（ページ編集画面の編集提案モード対応）の実装として、以下の点が適切に実装されています:

**アーキテクチャ**: 3層アーキテクチャの依存関係ルールに完全に準拠。Handler → UseCase → Repository の依存方向が守られ、UseCaseに `suggestionPageRepo` と `suggestionRepo` を追加して編集提案データを取得する設計が適切。

**セキュリティ**: CSRFトークンが適切に含まれ、XSS対策（templの自動エスケープ）も問題なし。編集提案モードでは `_method=PATCH` を除外し、POSTでSuggestionPageRevisionを作成する設計が正しい。

**国際化**: `page_edit_suggestion_editing` と `page_edit_update_suggestion_button` の翻訳キーが日英両方に追加され、i18nガイドの命名規則に従っている。

**テスト**: `TestEdit_SuggestionMode` で編集提案モードの表示切り替えを網羅的にテスト。既存テストも新しい依存関係に合わせて適切に更新。`t.Parallel()` の呼び出しも確認済み。

**テンプレート**: `IsSuggestionMode()` メソッドによる条件分岐がシンプルで読みやすい。templ-guideの構造体ベースのパターンに準拠。

**注記**: 「編集提案を更新」ボタンの送信先（`SuggestionPageRevisionsPath`）のエンドポイントはタスク9-3で実装予定。フィーチャーフラグで制御された開発中機能のため問題なし。
