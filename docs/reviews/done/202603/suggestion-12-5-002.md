# コードレビュー: suggestion-12-5

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-26                       |
| 対象ブランチ               | suggestion-12-5                  |
| ベースブランチ             | suggestion-12-4                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 24 ファイル                      |
| 変更行数（実装）           | +789 / -0 行                     |
| 変更行数（テスト）         | +778 / -4 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/suggestion_comments.sql`
- [x] `go/internal/query/suggestion_comments.sql.go`（自動生成）
- [x] `go/internal/repository/suggestion_comment.go`
- [x] `go/internal/usecase/update_suggestion_comment.go`
- [x] `go/internal/validator/suggestion_comment.go`
- [x] `go/internal/handler/suggestion_comment/handler.go`
- [x] `go/internal/handler/suggestion_comment/edit.go`
- [x] `go/internal/handler/suggestion_comment/update.go`
- [x] `go/internal/templates/pages/suggestion_comment/edit.templ`
- [x] `go/internal/templates/pages/suggestion_comment/edit_templ.go`（自動生成）
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [x] `go/internal/handler/suggestion_comment/edit_test.go`
- [x] `go/internal/handler/suggestion_comment/update_test.go`
- [x] `go/internal/usecase/update_suggestion_comment_test.go`
- [x] `go/internal/validator/suggestion_comment_test.go`
- [x] `go/internal/repository/suggestion_comment_test.go`

### 設定・その他

- [x] `go/internal/testutil/suggestion_comment_builder.go`
- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/suggestion-12-5-001.md`

## ファイルごとのレビュー結果

問題のあるファイルのみ記載します。

### `go/internal/handler/suggestion_comment/edit.go`

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Handler の処理フロー
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#handler-での処理フロー]**: `findComment` でコメント一覧を線形探索しているが、Edit/Update の両方で必要なのは特定の 1 件のコメントのみ。`GetSuggestionDetailUsecase` は全コメントを取得するため、コメント数が多い編集提案では無駄なデータ取得が発生する。ただし、これは既存の `GetSuggestionDetailUsecase` を再利用する設計判断であり、タスク 12-5 の想定ファイル数・行数の範囲内に収めるためのトレードオフとして許容できる。現時点では問題なし。

  **修正案**:

  将来的にパフォーマンス問題が発生した場合、コメント番号で直接取得する読み取り UseCase を作成する。

  **対応方針**:
  - [ ] 現状のまま（パフォーマンス問題が発生したら対応）
  - [x] 今回の PR でコメント番号での直接取得に変更
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 12-5（コメントの編集: Validator・UseCase・ハンドラー・テンプレート）の実装が作業計画書の要件に沿って適切に行われている。

**良い点**:

- 3 層アーキテクチャ（Handler → UseCase → Repository）の依存関係ルールに厳格に従っている
- CSRF 対策（hidden input + Method Override パターン）が適切に実装されている
- SQL クエリに `space_id` スコープが含まれており、セキュリティガイドラインに準拠している
- 既存パターン（suggestion/edit.go, suggestion/update.go）との一貫性が保たれている
- バリデーション（必須チェック・文字数制限）が Create と Update で同一の定数 `suggestionCommentBodyMaxLength` を共有している
- テストカバレッジが十分: Handler（認証・認可・バリデーションエラー・正常系）、UseCase（Markdown→HTML 変換・スペースID スコープ）、Repository（CRUD・スペースID スコープ）、Validator（空文字・最大文字数・正常系）
- 国際化が日英両方に追加され、`description` フィールドも適切
- テストヘルパーのリファクタリング（`newPostRequest` → `newRequest` + `newPostRequest` ラッパー）が後方互換性を保っている

**対応済み**:

- `findComment` の線形探索を `GetSuggestionCommentUsecase`（読み取りUseCase）による直接取得に変更。Edit/Update ハンドラーが `FindByNumber` でコメントを直接取得するようにリファクタリングした
