# コードレビュー: draft-update-3-1

## レビュー情報

| 項目                       | 内容                               |
| -------------------------- | ---------------------------------- |
| レビュー日                 | 2026-03-06                         |
| 対象ブランチ               | draft-update-3-1                   |
| ベースブランチ             | draft-update                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md |
| 変更ファイル数             | 6 ファイル                         |
| 変更行数（実装）           | +167 / -5 行（SQL含む）            |
| 変更行数（テスト）         | +125 / -1 行                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/joined_draft_pages.sql`
- [x] `go/internal/query/joined_draft_pages.sql.go`（自動生成）
- [x] `go/internal/repository/draft_page.go`

### テストファイル

- [x] `go/internal/repository/draft_page_test.go`
- [x] `go/internal/handler/draft_page/create_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/draft-update.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

**確認した主なポイント**:

- **セキュリティ（SQL）**: `ListDraftPagesByUserForIndex` クエリはすべての JOIN 条件に `dp.space_id` を含めており、スペース間のデータ漏洩を防止している（[@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md)）
- **アーキテクチャ**: Repository が Query に依存し、Model に変換するパターンを正しく踏襲している（[@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md)）
- **既存コードとの一貫性**: `toDraftPagesFromIndexRows` は既存の `toDraftPagesFromJoinedRows` と同じパターンで実装されており、`interface{}` 型の title フィールドの処理も一貫している
- **テスト**: ソート順、空スライス、関連エンティティのフィールド検証が適切にカバーされている
- **作業計画書との整合性**: タスク 3-1 の要件（スペース名・トピック名を含むクエリ、スペース名・トピック名順のソート、`ListByUserForIndex` メソッド）がすべて実装されている
- **create_test.go の変更**: テストデータの識別子をプレフィックス `dp-` 付きに変更し、他のテストとの衝突を回避する修正。テストの並行実行の安全性向上

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3-1（下書き一覧用のクエリとリポジトリメソッド追加）の要件をすべて満たしている。SQL クエリはセキュリティガイドラインのスペースID スコーピングに準拠し、論理削除されたレコードの除外も適切に行われている。リポジトリの変換メソッドは既存パターンに忠実で、テストはソート順・空結果・フィールドマッピングを網羅的にカバーしている。
