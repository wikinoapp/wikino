# コードレビュー: suggestion-9-3

## レビュー情報

| 項目                       | 内容                                             |
| -------------------------- | ------------------------------------------------ |
| レビュー日                 | 2026-03-20                                       |
| 対象ブランチ               | suggestion-9-3                                   |
| ベースブランチ             | develop                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md                 |
| 変更ファイル数             | 23 ファイル（実装 11、テスト 5、ドキュメント 7） |
| 変更行数（実装）           | +386 / -65 行                                    |
| 変更行数（テスト）         | +922 / -7 行                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_page/handler.go`
- [x] `go/internal/handler/suggestion_page/update.go`
- [x] `go/internal/usecase/update_suggestion_page.go`
- [x] `go/internal/validator/suggestion_page.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_page/main_test.go`
- [x] `go/internal/handler/suggestion_page/update_test.go`
- [x] `go/internal/usecase/update_suggestion_page_test.go`
- [x] `go/internal/validator/suggestion_page_test.go`
- [x] `go/internal/handler/page/edit_test.go`

### ドキュメント

- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/testing-guide.md`
- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/plans/1_doing/write-usecase-refactoring.md`
- [x] `docs/reviews/suggestion-9-3-001.md`
- [x] `docs/reviews/suggestion-9-3-002.md`
- [x] `docs/reviews/suggestion-9-3-003.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っています。

### レビュー詳細

各ファイルに対して確認した主なポイント:

**アーキテクチャ（3層アーキテクチャの依存関係）**:

- Handler → UseCase → Repository の依存方向が正しい
- Handler パッケージに `repository` の import がない
- Validator は `internal/validator/` パッケージに配置されている
- 書き込み UseCase はデータ取得を行わず、Validator から渡された `DraftPage` を使用している
- Handler の処理フロー（読み取り → 検証 → 書き込み）が architecture-guide.md のパターンに従っている

**セキュリティ**:

- 認証チェック（未ログイン → `/sign_in` リダイレクト）が実装されている
- 認可チェック（スペースメンバーでなければ 403）が実装されている
- ステータスチェック（反映済み・クローズ済みは更新不可）が実装されている
- SuggestionPage が現在の編集提案に属していることを検証している
- `url.PathEscape` で suggestion_page_id をエスケープしている（path.go）
- UseCase 内の `UpdateContent` クエリは `space_id` を WHERE 条件に含めている

**ハンドラーガイドライン**:

- `suggestion_page/` ディレクトリ構成が標準パターンに従っている
- ファイル名が標準の 8 種類（`handler.go`, `update.go`）のみ
- Handler 構造体のフィールド数が 4 個（上限 8 以内）

**バリデーション**:

- `SuggestionPageUpdateValidator` が `internal/validator/` に配置されている
- Result 構造体でバリデーション結果を返している
- Validator が取得した DraftPage を Result に含めて返し、Handler → 書き込み UseCase 間でデータの二重取得を回避している

**国際化**:

- `flash_suggestion_page_updated` が ja.toml, en.toml の両方に追加されている
- description フィールドが記述されている
- フラッシュメッセージで `i18n.T()` を使用している

**テスト**:

- Handler テスト: 未ログイン、非スペースメンバー(403)、正常更新、下書きステータス更新、反映済み(リダイレクト)、クローズ済み(リダイレクト) をカバー
- UseCase テスト: 正常系（タイトルあり）、正常系（タイトル nil）をカバー
- Validator テスト: 正常系、DraftPage 不在、DraftPage が別の SuggestionPage にリンク をカバー
- すべてのトップレベルテストで `t.Parallel()` を呼んでいる
- トランザクション管理を自前で行うテストでは `GetTestDB()` を使用している

**設計との整合性（作業計画書タスク 9-3）**:

- URL パターン `PATCH /s/{space}/suggestions/{number}/suggestion_pages/{suggestion_page_id}` が作業計画書と一致
- 下書きステータスまたはオープンステータスの編集提案のページのみ更新可能（仕様通り）
- DraftPage の `suggestion_page_id` は更新時にクリアしない（仕様通り、テストでも検証済み）
- DraftPage の内容から SuggestionPage のコンテンツ更新 + SuggestionPageRevision 作成（仕様通り）

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 9-3（「編集提案を更新」アクションの実装）が作業計画書の仕様通りに実装されている。

良い点:

- アーキテクチャガイドの「読み取り → 検証 → 書き込み」パターンに忠実に従っている
- Handler 内の各チェック（認証・認可・ステータス・SuggestionPage 所属確認）が適切な順序で実装されている
- Validator が DraftPage の取得・検証を一体化し、書き込み UseCase に検証済みデータを渡す設計が architecture-guide.md のベストプラクティスと一致している
- テストカバレッジが充実している（正常系・異常系を網羅）
- ドキュメント（architecture-guide.md, testing-guide.md）に新しいパターンの解説が追加されている
- 実装コード行数は目安（300行）をわずかに超えるが、テストコードとの一体性・機能の完全性を考慮すると問題ない
