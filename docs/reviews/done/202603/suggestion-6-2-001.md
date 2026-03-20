# コードレビュー: suggestion-6-2

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-19                             |
| 対象ブランチ               | suggestion-6-2                         |
| ベースブランチ             | develop                                |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 17 ファイル                            |
| 変更行数（実装）           | +722 / -123 行（自動生成ファイル含む） |
| 変更行数（テスト）         | +129 / -0 行                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/page_revisions.sql`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/show.go`
- [x] `go/internal/query/page_revisions.sql.go`（自動生成）
- [x] `go/internal/repository/page_revision.go`
- [x] `go/internal/templates/pages/suggestion/diff.templ`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/pages/suggestion/diff_templ.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion/show_templ.go`（自動生成）
- [x] `go/internal/usecase/get_suggestion_diff.go`
- [x] `go/internal/viewmodel/suggestion.go`

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/usecase/get_suggestion_diff_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 6-2（「編集したページ」タブの実装）が作業計画書通りに実装されている。

**良い点**:

- **アーキテクチャの準拠**: 3 層アーキテクチャの依存関係ルールが正しく守られている。UseCase は Repository のみに依存し、Handler は UseCase 経由でデータを取得している
- **効率的なデータ取得**: 差分タブがアクティブな場合のみベースリビジョンを取得しており、会話タブでは不要な DB クエリを発生させない設計になっている
- **セキュリティ**: SQL クエリに `space_id` によるスコープが適切に含まれている（`FindPageRevisionByID` のクエリ）
- **既存コンポーネントの再利用**: フェーズ 6-1 で実装された `components.DiffView` と `viewmodel.ComputeDiffBlocks` を適切に再利用している
- **命名規則**: UseCase 名 `GetSuggestionDiffUsecase`（`Get` プレフィックスで読み取り UseCase であることが明確）、ファイル名 `get_suggestion_diff.go`（動詞先頭）がガイドラインに準拠
- **国際化**: 新しいメッセージ `suggestion_diff_title_change` が日英両方の翻訳ファイルに追加されており、`description` も記述されている
- **テスト**: 正常系（ベースリビジョン取得）と境界値（空の SuggestionPages）のテストが実装されている。`t.Parallel()` と `testutil.SetupTx` を使用した TestMain パターンに準拠
- **templ テンプレート**: データ構造体パターン（`DiffTabData`, `ShowData`）を使用し、templ ガイドの推奨パターンに従っている
