# コードレビュー: suggestion-12-4

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-26                                      |
| 対象ブランチ               | suggestion-12-4                                 |
| ベースブランチ             | suggestion-12-3                                 |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md（タスク12-4）  |
| 変更ファイル数             | 28 ファイル（うちレビューファイル4、自動生成2） |
| 変更行数（実装）           | +665 / -2 行（自動生成・ドキュメント除く）      |
| 変更行数（テスト）         | +562 / -41 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/suggestion/edit.go`
- [x] `go/internal/handler/suggestion/update.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/usecase/update_suggestion.go`
- [x] `go/internal/validator/suggestion.go`
- [x] `go/internal/repository/suggestion.go`
- [x] `go/db/queries/suggestions.sql`
- [x] `go/internal/query/suggestions.sql.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion/edit.templ`
- [x] `go/internal/templates/pages/suggestion/edit_templ.go`（自動生成）
- [x] `go/internal/templates/page_name.go`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion/edit_test.go`
- [x] `go/internal/handler/suggestion/update_test.go`
- [x] `go/internal/usecase/update_suggestion_test.go`
- [x] `go/internal/validator/suggestion_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`（ヘルパーリファクタリング）
- [x] `go/internal/handler/suggestion/new_test.go`（ヘルパーリファクタリング）
- [x] `go/internal/handler/suggestion/show_test.go`（ヘルパーリファクタリング）

### 設定・その他

- [x] `go/docs/handler-guide.md`
- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計との整合性チェック

タスク12-4の作業計画書との整合性を確認しました。

| 計画書の要件                                                           | 実装状況                                    |
| ---------------------------------------------------------------------- | ------------------------------------------- |
| `internal/validator/suggestion.go` に `SuggestionUpdateValidator` 作成 | ✅ 実装済み                                 |
| タイトル必須・長さ制限、本文長さ制限                                   | ✅ タイトル200文字、本文10000文字           |
| `internal/usecase/update_suggestion.go` に UseCase 作成                | ✅ 実装済み                                 |
| Markdown→HTML変換                                                      | ✅ Wikiリンク解決含む                       |
| `internal/repository/suggestion.go` に `Update` メソッド追加           | ✅ space_idスコープ付き                     |
| `internal/query/queries/suggestions.sql` に UPDATE クエリ追加          | ✅ 実装済み                                 |
| `internal/handler/suggestion/edit.go` に `Edit` 実装                   | ✅ GET /s/{space}/suggestions/{number}/edit |
| `internal/handler/suggestion/update.go` に `Update` 実装               | ✅ PATCH /s/{space}/suggestions/{number}    |
| `internal/templates/pages/suggestion/edit.templ` 作成                  | ✅ タイトル+本文の編集フォーム              |
| `cmd/server/main.go` にルーティング登録                                | ✅ 2ルート追加                              |
| 翻訳ファイル（ja.toml, en.toml）にメッセージ追加                       | ✅ 各10キー追加                             |
| テスト（ハンドラー・UseCase・Validator）                               | ✅ 全レイヤーにテストあり                   |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク12-4（編集提案本文の編集）の実装がガイドラインに準拠して正しく行われています。

**良い点**:

- **アーキテクチャ準拠**: Handler → UseCase → Repository の依存方向が正しく、Handler から Repository への直接依存がない。認可チェックは Handler で Policy を使って実行
- **セキュリティ**: CSRF トークン、認証チェック、認可チェック（`CanUpdateSuggestion`）、SQLクエリの `space_id` スコープがすべて実装されている
- **UseCaseのルール遵守**: トランザクション前にデータ取得（`renderBodyHTML`）、トランザクション内は永続化のみ、Execute内にロジックを直接書かない、の3ルールに従っている
- **Handler処理フロー**: 「読み取り → 検証 → 書き込み」パターンに従い、GetSuggestionDetail → Policy → Validator → UpdateSuggestion の順序で処理
- **テストカバレッジ**: 全レイヤー（Handler、UseCase、Validator）にテストがあり、正常系・異常系をカバー。テスト関数の配置ルール（`TestEdit_*` → `edit_test.go`、`TestUpdate_*` → `update_test.go`）も守られている
- **コード再利用**: `renderEditForm` をEdit/Updateの両方から使用、バリデーション定数（`suggestionTitleMaxLength`, `suggestionBodyMaxLength`）をCreate/Updateで共有
- **テストヘルパーのリファクタリング**: `newIndexRequest` → `newSuggestionRequest` への汎用化が全既存テストに反映されており、一貫性が保たれている
- **ガイドライン追加**: `handler-guide.md` にテスト関数の配置ルールを追加し、慣習を文書化している
