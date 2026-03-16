# コードレビュー: suggestion-1b-2

## レビュー情報

| 項目                       | 内容                                                  |
| -------------------------- | ----------------------------------------------------- |
| レビュー日                 | 2026-03-16                                            |
| 対象ブランチ               | suggestion-1b-2                                       |
| ベースブランチ             | suggestion-1b-1b                                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md（タスク 1b-2）       |
| 変更ファイル数             | 7 ファイル                                            |
| 変更行数（実装）           | +360 / -8 行（SQL 22, sqlc生成 175, repo 153, id 10） |
| 変更行数（テスト）         | +560 / -0 行（テスト 442, ビルダー 118）              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/suggestion_pages.sql`
- [x] `go/internal/query/models.go`（sqlc自動生成）
- [x] `go/internal/query/suggestion_pages.sql.go`（sqlc自動生成）
- [x] `go/internal/repository/suggestion_page.go`

### テストファイル

- [x] `go/internal/repository/suggestion_page_test.go`
- [x] `go/internal/testutil/suggestion_page_builder.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`（タスクチェックボックス更新のみ）

## ファイルごとのレビュー結果

すべてのファイルが問題なく、ガイドラインに準拠しています。以下に各ファイルの確認結果をまとめます。

### 確認済みの観点

**`go/db/queries/suggestion_pages.sql`**:

- CRUDクエリ（Create, FindByID, ListBySuggestionID, UpdateContent）が作業計画書の仕様に沿って実装されている
- すべてのクエリに `space_id` でのスコープ条件が含まれている（セキュリティガイドライン準拠）
- SQLコメントが日本語で記述されている（コーディング規約準拠）

**`go/internal/repository/suggestion_page.go`**:

- `SuggestionRepository` と同じパターンに従っている（構造体、コンストラクタ、WithTx、toModel/toModels）
- ドメインID型を適切に使用している（`model.SuggestionPageID`, `model.SpaceID` 等）
- nullable フィールド（Title）の `sql.NullString` 変換が正しい
- `FindByID` と `UpdateContent` で `sql.ErrNoRows` を `nil` で返す既存パターンに従っている
- Input構造体の命名が一貫している（`CreateSuggestionPageInput`, `UpdateSuggestionPageContentInput`）

**`go/internal/repository/suggestion_page_test.go`**:

- `t.Parallel()` を適切に使用している
- `testutil.SetupTx(t)` でトランザクション分離されている
- 正常系・異常系のテストケースが網羅されている（Create, FindByID, ListBySuggestionID, UpdateContent の各メソッド）
- セキュリティ観点のテスト（異なるスペースIDでの検索がnilを返すこと）も含まれている
- テストデータのビルダーパターンが既存テストと一貫している

**`go/internal/testutil/suggestion_page_builder.go`**:

- 既存ビルダー（`SuggestionBuilder`）と同じパターンに従っている
- 必須フィールドのバリデーションが `Build()` 内で実施されている
- `t.Helper()` が適切に呼ばれている
- デフォルト値が設定されている
- `WithNilTitle()` メソッドでnullableフィールドのテストも可能

**`go/internal/query/models.go`** / **`go/internal/query/suggestion_pages.sql.go`**:

- sqlc自動生成ファイルのため手動編集なし。生成結果は `suggestion_pages` テーブルスキーマと整合している

## 設計との整合性チェック

作業計画書のタスク **1b-2** の要件:

| 要件                                                                              | 実装状況 |
| --------------------------------------------------------------------------------- | -------- |
| `internal/query/queries/suggestion_pages.sql` にCRUDクエリを作成                  | ✅       |
| `internal/repository/suggestion_page.go` に `SuggestionPageRepository` を作成     | ✅       |
| WithTx パターンの実装                                                             | ✅       |
| toModel 変換メソッド                                                              | ✅       |
| Create, FindByID, ListBySuggestionID メソッド                                     | ✅       |
| UpdateContent メソッド（テーブル設計のコンテンツ直接保持パターンに対応）          | ✅       |
| テスト（想定行数 約130行 → 実際 442行：正常系・異常系を手厚くカバーしており適切） | ✅       |

作業計画書に記載されたタスク 1b-2 の全要件が実装されています。テスト行数は見積もりより多いですが、テストの網羅性が高く品質に貢献しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 1b-2（suggestion_pagesテーブルのsqlcクエリとリポジトリ）が作業計画書に沿って正しく実装されています。

良い点:

- 既存の `SuggestionRepository` のパターンと完全に一貫した実装
- すべてのクエリで `space_id` によるスコープが適用されており、セキュリティガイドラインに準拠
- テストが正常系・異常系（存在しないID、異なるスペースID）を網羅しており品質が高い
- ドメインID型の使用、nullable フィールドの適切な処理など、アーキテクチャガイドラインに完全準拠
- テストビルダーが既存パターンに従っており、今後のテスト作成にも活用できる
