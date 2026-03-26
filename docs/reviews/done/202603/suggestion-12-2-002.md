# コードレビュー: suggestion-12-2（前回レビュー対応後）

## レビュー情報

| 項目                       | 内容                                                |
| -------------------------- | --------------------------------------------------- |
| レビュー日                 | 2026-03-26                                          |
| 対象ブランチ               | suggestion-12-2                                     |
| ベースブランチ             | suggestion-12-1                                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md                    |
| 変更ファイル数             | 15 ファイル（実装 8, テスト 2, その他 5）           |
| 変更行数（実装）           | +84 / -8 行（自動生成・スキーマ・ドキュメント除く） |
| 変更行数（テスト）         | +106 / -26 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（UseCase、Repository、WithTx パターン）
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン（スペースIDによるクエリスコープ）

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260326051852_add_number_to_suggestion_comments.sql`
- [x] `go/db/queries/suggestion_comments.sql`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/model/id.go`
- [x] `go/internal/model/suggestion_comment.go`
- [x] `go/internal/repository/suggestion_comment.go`
- [x] `go/internal/usecase/create_suggestion_comment.go`
- [x] `go/internal/testutil/suggestion_comment_builder.go`

### テストファイル

- [x] `go/internal/repository/suggestion_comment_test.go`
- [x] `go/internal/handler/suggestion_comment/create_test.go`

### 設定・その他（自動生成含む）

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/suggestion-12-2-001.md`
- [x] `go/db/schema.sql`
- [x] `go/internal/query/models.go`
- [x] `go/internal/query/suggestion_comments.sql.go`

## ファイルごとのレビュー結果

前回レビュー（suggestion-12-2-001.md）で指摘したトランザクション不使用の問題は、`create_suggestion.go` と同じ WithTx パターンで正しく修正されている。以下は修正後の新たな確認結果。

すべてのファイルを確認し、問題は見つかりませんでした。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

前回レビュー（001）で指摘したトランザクション不使用の問題が正しく修正されている。

**修正内容の確認**:

- `create_suggestion_comment.go`: `db *sql.DB` を依存に追加し、`BeginTx` → `WithTx` → `GetNextNumber` → `Create` → `Commit` の WithTx パターンが適用されている。`Execute` 内はメソッド呼び出しのみ（`createComment`）で、書き込み UseCase のルール 3 に従っている
- `cmd/server/main.go`: コンストラクタ呼び出しに `db` 引数を正しく追加
- `create_test.go`: 正常系テスト（`TestCreate_正常にコメントが作成されリダイレクトされる`）が `SetupTx` から `GetTestDB` に正しく変更されている。UseCase が独自トランザクションを管理するため、テストトランザクションとの二重トランザクション問題を回避している。異常系テスト（未ログイン、バリデーションエラー、404、403）は UseCase に到達しないため `SetupTx` のままで正しい

**作業計画書との整合性**: タスク 12-2 の全要件が実装されている。

- マイグレーション: `number` カラム追加、既存データバックフィル（`ROW_NUMBER()`）、ユニークインデックス `[suggestion_id, number]`、ロールバック対応
- モデル: `SuggestionCommentNumber` 型と `Number` フィールドの追加
- リポジトリ: `GetNextNumber` メソッドと `Create` の `Number` パラメータ追加
- UseCase: トランザクション内での番号採番
- sqlc: コード再生成
- テスト: Repository テスト（`GetNextNumber` の 0 件/1 件以上ケース）と Handler テストの更新
