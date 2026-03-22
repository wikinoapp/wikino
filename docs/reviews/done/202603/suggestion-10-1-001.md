# コードレビュー: suggestion-10-1

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-22                       |
| 対象ブランチ               | suggestion-10-1                  |
| ベースブランチ             | develop                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 14 ファイル                      |
| 変更行数（実装）           | +114 / -32 行                    |
| 変更行数（テスト）         | +114 / -1 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/draft_page_revision/update.go`
- [x] `go/internal/usecase/get_page_detail.go`
- [x] `go/internal/usecase/manual_save_draft_page.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`
- [x] `go/internal/usecase/get_page_detail_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っています。

## 設計との整合性チェック

作業計画書で定義されている今回の変更範囲（ページ編集画面に「下書き保存して編集提案を作成する...」ドロップダウンを追加）について確認:

- ✅ 「下書き保存」ボタンの右側にドロップダウンメニューを表示するアイコンがあり、クリックすると「下書き保存して編集提案を作成する...」アクションが選択できる（`edit.templ` で実装済み）
- ✅ このアクションを実行すると、下書き保存後に保存した下書きページが選択された状態で編集提案作成画面に直接遷移する（`update.go` の `redirect_to=suggestion_new` パラメータで実装済み）
- ✅ フィーチャーフラグ `go_suggestion` が有効な場合のみドロップダウンが表示される（`SuggestionEnabled` フラグで制御）
- ✅ 編集提案モード（`IsSuggestionMode()`）の場合はドロップダウンを表示しない
- ✅ i18n対応済み（`page_edit_save_draft_and_create_suggestion_button` が ja.toml と en.toml の両方に追加）

設計との乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書の仕様に沿って、ページ編集画面に「下書き保存して編集提案を作成する...」機能が正しく実装されています。

**良い点**:

- フィーチャーフラグによる段階的な機能公開が適切に実装されている
- `redirect_to` パラメータは固定値 `"suggestion_new"` との比較のみで使用されており、オープンリダイレクト脆弱性がない
- テンプレートの条件分岐が明確で、通常モード/編集提案モード/フラグ無効時の3パターンが適切に分離されている
- `ManualSaveDraftPageOutput` に `DraftPage` を追加し、リダイレクト先の URL に DraftPage ID を含められるようにした設計が自然
- テスト（`TestUpdate_RedirectToSuggestionNew`）が新機能のリダイレクト先を正しく検証している
- アーキテクチャガイドの依存関係ルールに違反なし（Handler → UseCase → Repository の一方向依存を維持）
