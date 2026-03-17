# コードレビュー: suggestion-3-2

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-17                                     |
| 対象ブランチ               | suggestion-3-2                                 |
| ベースブランチ             | suggestion-3-1                                 |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md（タスク 3-2） |
| 変更ファイル数             | 25 ファイル（docs/reviews 除く）               |
| 変更行数（実装）           | +1118 行（テスト・自動生成・docs 除く）        |
| 変更行数（テスト）         | +922 / -15 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/new.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/usecase/get_suggestion_new.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [x] `go/internal/templates/pages/suggestion/new.templ`
- [x] `go/internal/templates/pages/suggestion/new_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/internal/query/draft_pages.sql.go`（自動生成）
- [x] `go/internal/repository/draft_page.go`

### テストファイル

- [x] `go/internal/handler/suggestion/new_test.go`
- [x] `go/internal/handler/suggestion/create_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/usecase/get_suggestion_new_test.go`
- [x] `go/internal/viewmodel/suggestion_test.go`

### テストユーティリティ

- [x] `go/internal/testutil/page_revision_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/suggestion_builder.go`
- [x] `go/internal/testutil/suggestion_page_builder.go`
- [x] `go/internal/testutil/topic_builder.go`

## ファイルごとのレビュー結果

問題なし。すべてのファイルがガイドラインに従っています。

### レビュー詳細（問題なし項目の確認記録）

**アーキテクチャ（[@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md)）**:

- Handler → UseCase → Repository の依存方向: ✅
- Handler パッケージから repository を import していない: ✅
- Validator は `internal/validator/` パッケージに配置: ✅（前タスク 3-1 で実装済み）
- UseCase の命名: `GetSuggestionNewUsecase`（読み取り UseCase に `Get` プレフィックス）✅
- Handler は UseCase 経由でデータアクセス: ✅

**ハンドラー（[@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md)）**:

- 標準ファイル名（handler.go, new.go, create.go）: ✅
- メソッド名（New, Create）: ✅
- Handler 構造体のフィールド数: 6 個（8 個以下）✅
- ルーティング: GET /suggestions/new → New、POST /suggestions → Create: ✅

**セキュリティ（[@go/docs/security-guide.md](/workspace/go/docs/security-guide.md)）**:

- CSRF トークン: フォームに含まれている ✅
- 認証チェック: New, Create 両方で実施 ✅
- 認可チェック: UseCase でスペースメンバー確認 ✅
- space_id によるクエリスコープ: SQL クエリに `dp.space_id = $3` 含まれている ✅
- エラー詳細をユーザーに漏らしていない: ✅

**国際化（[@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md)）**:

- ユーザー向けメッセージはすべて i18n 対応: ✅
- ja.toml と en.toml の両方に翻訳追加: ✅
- description フィールド記述済み: ✅
- 命名規則 `{機能名}_{種別}_{詳細}`: ✅

**テンプレート（[@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md)）**:

- 構造体ベースのデータ渡し（`NewData` 構造体）: ✅
- `context.Context` を明示的に渡していない: ✅
- ViewModel を使用: ✅
- セキュリティ: templ の自動エスケープを活用 ✅

**テスト（[@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md)）**:

- `t.Parallel()` 使用: ✅
- SetupTx / GetTestDB の使い分け: ✅（UseCase テストは GetTestDB、ハンドラーテストはケースに応じて適切に使い分け）
- 正常系・異常系の網羅: ✅
- テストビルダーパターン: ✅

**作業計画書との整合性**:

- タスク 3-2 の要件をすべて実装: ✅
  - `internal/handler/suggestion/new.go`: New メソッド ✅
  - `internal/handler/suggestion/create.go`: Create メソッド ✅
  - `internal/usecase/get_suggestion_new.go`: 作成画面用データ取得 UseCase ✅
  - `internal/templates/pages/suggestion/new.templ`: 作成フォームテンプレート ✅
  - 翻訳ファイル追加 ✅
  - `cmd/server/main.go` にルーティング登録 ✅
- 下書きページのチェックボックス選択 UI: ✅
- タイトル・概要入力フォーム: ✅
- クエリパラメータによる下書き事前選択: ✅
- バリデーションエラー時のフォーム再表示: ✅

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

編集提案作成のハンドラー・テンプレート・UseCase が適切に実装されています。アーキテクチャガイド（3 層アーキテクチャ、依存方向）、ハンドラーガイド（標準ファイル名・メソッド名）、セキュリティガイド（CSRF、認証認可、space_id スコープ）、国際化ガイド、テンプレートガイドのすべてに準拠しています。テストは正常系・異常系を網羅しており、テストヘルパー（DB ビルダー）も適切に追加されています。作業計画書タスク 3-2 の要件もすべて満たしています。
