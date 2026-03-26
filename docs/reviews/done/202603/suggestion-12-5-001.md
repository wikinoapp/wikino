# コードレビュー: suggestion-12-5

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-26                       |
| 対象ブランチ               | suggestion-12-5                  |
| ベースブランチ             | suggestion-12-4                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 22 ファイル                      |
| 変更行数（実装）           | +816 / -4 行                     |
| 変更行数（テスト）         | +803 / -1 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/suggestion_comments.sql`
- [x] `go/internal/query/suggestion_comments.sql.go`
- [x] `go/internal/repository/suggestion_comment.go`
- [x] `go/internal/validator/suggestion_comment.go`
- [x] `go/internal/usecase/update_suggestion_comment.go`
- [x] `go/internal/handler/suggestion_comment/handler.go`
- [x] `go/internal/handler/suggestion_comment/edit.go`
- [x] `go/internal/handler/suggestion_comment/update.go`
- [x] `go/internal/templates/pages/suggestion_comment/edit.templ`
- [x] `go/internal/templates/pages/suggestion_comment/edit_templ.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `docs/plans/1_doing/suggestion.md`

### テストファイル

- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [x] `go/internal/handler/suggestion_comment/edit_test.go`
- [x] `go/internal/handler/suggestion_comment/update_test.go`
- [x] `go/internal/validator/suggestion_comment_test.go`
- [x] `go/internal/repository/suggestion_comment_test.go`
- [x] `go/internal/testutil/suggestion_comment_builder.go`

## ファイルごとのレビュー結果

### `go/internal/usecase/update_suggestion_comment.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - レイヤーごとのテストカバレッジ

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#レイヤーごとのテストカバレッジ]**: UseCase のテストファイルが存在しない

  テストガイドでは UseCase のテストは「必須」と定義されている。`update_suggestion_comment.go` に対応する `update_suggestion_comment_test.go` が存在しない。なお、`create_suggestion_comment.go` にもテストファイルが存在しないが、これは本PRのスコープ外のため対象外とする。

  **修正案**:

  `go/internal/usecase/update_suggestion_comment_test.go` を作成し、以下をテストする:
  - 正常系: コメントが正しく更新される（Body と BodyHTML の両方が更新される）
  - スペースIDスコープ: 異なるスペースIDでは更新されない

  **対応方針**:
  - [x] UseCase テストを追加する
  - [ ] Handler テストで間接的にカバーされているため、このPRではスキップし、後続タスクで対応する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

作業計画書のタスク 12-5 で定義された全ファイル（Validator・UseCase・Repository・Handler・テンプレート・翻訳・ルーティング）が漏れなく実装されている。

アーキテクチャガイドラインへの準拠は良好で、3層アーキテクチャの依存関係ルール、Handler での認可チェック、Validator の配置、UseCase の書き込みルール（トランザクション前のデータ取得・トランザクション内は永続化のみ・Execute内にロジックを直接書かない）をすべて遵守している。既存の編集提案編集（suggestion/edit.go, update.go）と一貫したパターンで実装されており、コードベース全体の一貫性が保たれている。

セキュリティ面では、CSRF トークン、スペースIDによるクエリスコープ、Policy による認可チェックが適切に実装されている。テストも Handler・Validator・Repository の各レイヤーで正常系・異常系が網羅されている。

唯一の指摘は UseCase テストの不在だが、Handler テストの `TestUpdate_正常にコメントが更新されリダイレクトされる` で間接的にカバーされており、ブロッカーではない。
