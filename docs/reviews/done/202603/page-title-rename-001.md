# コードレビュー: page-title-rename

## レビュー情報

| 項目                       | 内容                                                         |
| -------------------------- | ------------------------------------------------------------ |
| レビュー日                 | 2026-03-17                                                   |
| 対象ブランチ               | page-title-rename                                            |
| ベースブランチ             | develop                                                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-title-rename-unpublished-conflict.md |
| 変更ファイル数             | 10 ファイル（うちドキュメント 1、sqlc 自動生成 1）           |
| 変更行数（実装）           | +67 / -23 行                                                 |
| 変更行数（テスト）         | +206 / -0 行                                                 |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/pages.sql`
- [x] `go/internal/query/pages.sql.go`（sqlc自動生成）
- [x] `go/internal/repository/page.go`
- [x] `go/internal/testutil/page_builder.go`
- [x] `go/internal/validator/page.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/usecase/publish_page.go`

### テストファイル

- [x] `go/internal/validator/page_test.go`
- [x] `go/internal/usecase/publish_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/page-title-rename-unpublished-conflict.md`

## ファイルごとのレビュー結果

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書に記載された設計どおりに正確に実装されている。具体的には:

- **バリデーター**: `PageUpdateValidatorResult` に `UnpublishedConflictingPageID` フィールドを追加し、未公開かつ本文が空のページとの競合時はエラーにせず、競合ページIDを返すロジックが正しく実装されている
- **ハンドラー**: バリデーション結果の `UnpublishedConflictingPageID` を `PublishPageInput` に渡すだけのシンプルな変更
- **ユースケース**: トランザクション内で競合ページの論理削除をページ更新の直前に実行しており、データ整合性が保たれている
- **SQLクエリ**: `space_id` によるスコープが適切に設定されており、セキュリティガイドラインに準拠
- **アーキテクチャ**: 依存関係のルール（Handler → UseCase、Validator は Application 層）に従っている
- **テスト**: バリデーター（3ケース: 未公開+空本文、未公開+本文あり、公開済み）とユースケース（論理削除の実行確認）の両方でテストが追加されている

良かった点:

- 実装コード 67 行と軽量で、PR サイズガイドラインの範囲内
- 既存コードのパターン（WithTx、ステップ番号のコメント）を踏襲している
- テストが要件の3パターン（未公開+空、未公開+本文あり、公開済み）を網羅している
