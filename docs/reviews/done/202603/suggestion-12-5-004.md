# コードレビュー: suggestion-12-5

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-26                             |
| 対象ブランチ               | suggestion-12-5                        |
| ベースブランチ             | suggestion-12-4                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 30 ファイル（レビュー・計画書除く 24） |
| 変更行数（実装）           | +730 行（自動生成除く）                |
| 変更行数（テスト）         | +1027 行（テストヘルパー含む）         |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/suggestion_comments.sql`
- [x] `go/internal/handler/suggestion_comment/handler.go`
- [x] `go/internal/handler/suggestion_comment_edit/handler.go`
- [x] `go/internal/handler/suggestion_comment_edit/edit.go`
- [x] `go/internal/handler/suggestion_comment_edit/update.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/query/suggestion_comments.sql.go`（自動生成）
- [x] `go/internal/repository/suggestion_comment.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/pages/suggestion_comment/edit.templ`
- [x] `go/internal/templates/pages/suggestion_comment/edit_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/get_suggestion_comment.go`
- [x] `go/internal/usecase/update_suggestion_comment.go`
- [x] `go/internal/validator/suggestion_comment.go`

### テストファイル

- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [x] `go/internal/handler/suggestion_comment_edit/edit_test.go`
- [x] `go/internal/handler/suggestion_comment_edit/main_test.go`
- [x] `go/internal/handler/suggestion_comment_edit/update_test.go`
- [x] `go/internal/repository/suggestion_comment_test.go`
- [x] `go/internal/testutil/suggestion_comment_builder.go`
- [x] `go/internal/usecase/get_suggestion_comment_test.go`
- [x] `go/internal/usecase/update_suggestion_comment_test.go`
- [x] `go/internal/validator/suggestion_comment_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

問題のあるファイルはありませんでした。すべてのファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

### 作業計画書からの乖離について（情報共有）

作業計画書ではハンドラーの配置先が `internal/handler/suggestion_comment/edit.go` と `internal/handler/suggestion_comment/update.go` と記載されていますが、実装では `internal/handler/suggestion_comment_edit/` という別ディレクトリが作成されています。

これはハンドラーガイドラインの「Handler 構造体のフィールドが 8 個を超えたらリソース分割を検討」ルールに基づく妥当な判断です。`suggestion_comment` ハンドラーに edit/update を統合すると 9 フィールドとなり制限を超えるため、分割は適切です。`suggestion_page_edit/` が `suggestion_page/` から分離されているのと同じパターンです。

## レビューで確認した項目

### セキュリティ

- CSRF トークン: テンプレートのフォームに `csrf_token` hidden input あり、Method Override (`_method=PATCH`) も正しく設定
- SQL インジェクション: sqlc 生成コードによるプリペアドステートメント使用
- スペース ID スコープ: UPDATE クエリに `WHERE id = $1 AND space_id = $5` あり、防御的プログラミングが適用されている
- 認証・認可: Handler で `middleware.UserFromContext` → `policy.CanUpdateSuggestionComment` の順に実行

### アーキテクチャ

- 依存の方向: Handler → UseCase → Repository → Query の 3 層アーキテクチャに準拠
- Handler から Repository への直接依存なし
- Validator は `internal/validator/` に配置
- 認可チェックは Handler で実行（UseCase → Policy の依存なし）
- Handler の処理フロー: 読み取り UC → Policy → Validator → 書き込み UC の順序を遵守

### テストカバレッジ

| レイヤー     | テスト有無 | テスト内容                                      |
| ------------ | ---------- | ----------------------------------------------- |
| Handler      | あり       | 未ログイン、404、422（バリデーション）、正常系  |
| UseCase (読) | あり       | 正常取得、存在しないコメント、異なるスペース ID |
| UseCase (書) | あり       | 正常更新、Markdown 変換、異なるスペース ID      |
| Validator    | あり       | 空本文、長すぎる本文、有効な入力                |
| Repository   | あり       | CRUD、スペース ID スコープ                      |

### 命名規則・コーディング規約

- ファイル名: handler-guide の標準ファイル名（`handler.go`, `edit.go`, `update.go`）に準拠
- UseCase 命名: `Get` プレフィックス（読み取り）、動詞先頭（`update_suggestion_comment.go`）に準拠
- コメント: 日本語で記述
- ログ: `log/slog` を使用
- I18n: ja.toml / en.toml 両方に翻訳を追加、`description` 付き

## 総合評価

**評価**: Approve

**総評**:

コメント編集機能の実装が、プロジェクトのアーキテクチャガイドライン・ハンドラーガイドライン・セキュリティガイドラインに適切に準拠しています。

- ハンドラーのディレクトリ分割は作業計画書から乖離していますが、フィールド数制限を考慮した妥当な判断であり、既存の `suggestion_page_edit/` パターンとも一貫しています
- 全レイヤーにテストが追加されており、正常系・異常系ともにカバーされています
- セキュリティ面（CSRF、スペース ID スコープ、認証・認可）も適切に実装されています
