# コードレビュー: suggestion-9-1

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-19                       |
| 対象ブランチ               | suggestion-9-1                   |
| ベースブランチ             | suggestion-8a-1                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 20 ファイル（自動生成除く）      |
| 変更行数（実装）           | +646 行                          |
| 変更行数（テスト）         | +640 行                          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_change/index.go`
- [x] `go/internal/handler/suggestion_page_edit/handler.go`
- [x] `go/internal/handler/suggestion_page_edit/create.go`
- [x] `go/internal/handler/suggestion_page_edit/show.go`
- [x] `go/internal/usecase/start_suggestion_page_edit.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_page_edit/show.templ`

### テストファイル

- [x] `go/internal/handler/suggestion_page_edit/main_test.go`
- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/docs/security-guide.md`
- [x] `go/docs/testing-guide.md`
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion_page_edit/show_templ.go`（自動生成）

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 9-1（編集提案ページ編集開始のUseCase・ハンドラー）の実装として、作業計画書の要件を適切に満たしている。

**良い点**:

- **アーキテクチャ準拠**: Handler → UseCase → Repository の依存方向が正しく守られている。Handler から Repository への直接依存はない
- **セキュリティ**: 認証チェック、スペースメンバーチェック、ステータスチェック、CSRF トークンの扱い、SpaceIDによるクエリスコープがすべて適切
- **UseCase の設計**: `StartSuggestionPageEditUsecase` はread操作（既存下書きの確認）と write操作（トランザクション内でのDraftPage作成/更新）を適切に分離し、`StartSuggestionPageEditStatus` enum で結果を表現している
- **テスト**: UseCaseテスト4ケース + ハンドラーテスト4ケースで、正常系（下書きなし→リダイレクト、リンク済み→リダイレクト、Forceで上書き→リダイレクト）と異常系（未ログイン、非メンバー、コンフリクト）を網羅。DB直接アクセスパターン（GetTestDB）とトランザクション分離パターン（SetupTx）の使い分けも、UseCaseの内部トランザクション有無に応じて正しい
- **命名規則**: ディレクトリ名 `suggestion_page_edit`、ファイル名 `handler.go` / `create.go` / `show.go`、UseCase名 `StartSuggestionPageEditUsecase` がすべてガイドライン通り
- **i18n**: ユーザー向けメッセージはすべて翻訳キー経由。ja/en 両方に追加済み
- **URL設計**: `POST /s/{space}/suggestions/{number}/page_edits`（編集開始）、`GET /s/{space}/suggestions/{number}/page_edits/{suggestion_page_id}`（確認画面）が作業計画書通り
- **既存コードの拡張**: `suggestion_change/index.templ` への編集ボタン追加、`viewmodel.SuggestionPageDiff` への `SuggestionPageID` フィールド追加が最小限の変更で実現されている
- **`SuggestionPageEditShowPath` での `url.PathEscape`**: 防御的プログラミングとして適切
- **ドキュメント更新**: `security-guide.md` のCSRFミドルウェアの説明追加、`testing-guide.md` の並行テスト説明の改善が適切
