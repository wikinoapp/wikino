# コードレビュー: go-topic-1-1

## レビュー情報

| 項目                       | 内容                                          |
| -------------------------- | --------------------------------------------- |
| レビュー日                 | 2026-03-08                                    |
| 対象ブランチ               | go-topic-1-1                                  |
| ベースブランチ             | go-topic                                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/topic-show-go-migration.md |
| 変更ファイル数             | 6 ファイル                                    |
| 変更行数（実装）           | +74 / -4 行（SQL 35 行 + Repository 39 行）   |
| 変更行数（テスト）         | +238 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/pages.sql`
- [x] `go/internal/query/pages.sql.go`（sqlc 自動生成）
- [x] `go/internal/repository/page.go`
- [x] `go/internal/testutil/page_builder.go`

### テストファイル

- [x] `go/internal/repository/page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/topic-show-go-migration.md`

## ファイルごとのレビュー結果

全ファイル問題なし。

## 設計との整合性チェック

作業計画書のタスク **1-1** の要件を確認:

| 要件                                                         | 実装状況                                                                       |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| ピン留めページ取得クエリの追加（`queries/pages.sql`）        | ✅ 実装済み（`FindPinnedPagesByTopic`）                                        |
| 通常ページのオフセットベースページネーション取得クエリの追加 | ✅ 実装済み（`FindRegularPagesByTopicPaginated` + `CountRegularPagesByTopic`） |
| sqlc コード生成                                              | ✅ 実装済み（`pages.sql.go`）                                                  |
| `PageRepository` にメソッド追加（ピン留めページ取得）        | ✅ 実装済み（`FindPinnedByTopic`）                                             |
| `PageRepository` にメソッド追加（ページネーション取得）      | ✅ 実装済み（`FindRegularByTopicPaginated`）                                   |

### SQL クエリの仕様確認

- **ピン留めページ**: `topic_id` + `space_id` でスコープ、`pinned_at IS NOT NULL` + `published_at IS NOT NULL` + `discarded_at IS NULL` + `trashed_at IS NULL`、`pinned_at DESC` ソート → ✅ 作業計画書通り
- **通常ページ**: `topic_id` + `space_id` でスコープ、`pinned_at IS NULL` + `published_at IS NOT NULL` + `discarded_at IS NULL` + `trashed_at IS NULL`、`modified_at DESC, id DESC` ソート、LIMIT/OFFSET → ✅ 作業計画書通り
- **セキュリティ**: すべてのクエリに `space_id` 条件が含まれている → ✅ セキュリティガイドラインの「スペースIDによるクエリスコープ」に準拠

### テストカバレッジ確認

- `FindPinnedByTopic`: ピン留めページの取得・ソート順・非公開/廃棄/ゴミ箱ページの除外・別トピック分離 → ✅ 充分
- `FindRegularByTopicPaginated`: ページネーション（1 ページ目/2 ページ目）・TotalCount・ピン留め/非公開/廃棄/ゴミ箱ページの除外 → ✅ 充分
- `PageBuilder` への `WithPinnedAt` / `WithTrashed` 追加: テストデータ作成に必要な拡張 → ✅ 適切

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 1-1 の要件がすべて適切に実装されている。SQL クエリは `space_id` によるスコープが正しく設定されており、セキュリティガイドラインに準拠している。Repository メソッドは既存の `FindLinkedPagesPaginated` / `FindBacklinkedPagesPaginated` と同じパターンに従っており、コードベースの一貫性が保たれている。テストも正常系・異常系（非公開/廃棄/ゴミ箱ページの除外、ページネーション）を網羅しており、品質が高い。`PageBuilder` の拡張（`WithPinnedAt` / `WithTrashed`）も適切で、既存の `WithDiscarded` / `WithUnpublished` パターンに倣っている。
