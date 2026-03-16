# コードレビュー: suggestion-1a-2

## レビュー情報

| 項目                       | 内容                                  |
| -------------------------- | ------------------------------------- |
| レビュー日                 | 2026-03-13                            |
| 対象ブランチ               | suggestion-1a-2                       |
| ベースブランチ             | suggestion-1a-1                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md      |
| 変更ファイル数             | 7 ファイル                            |
| 変更行数（実装）           | +403 / -0 行（queries, repository等） |
| 変更行数（テスト）         | +542 / -0 行（テスト + ビルダー）     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/suggestions.sql`
- [x] `go/internal/query/models.go`（sqlc自動生成）
- [x] `go/internal/query/suggestions.sql.go`（sqlc自動生成）
- [x] `go/internal/repository/suggestion.go`

### テストファイル

- [x] `go/internal/repository/suggestion_test.go`
- [x] `go/internal/testutil/suggestion_builder.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/db/queries/suggestions.sql`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) - スペースIDでのスコープ
- [@go/docs/architecture-guide.md#Queryファイルの命名](/workspace/go/docs/architecture-guide.md) - Queryファイルの命名

**問題点・改善提案**:

- **[@go/docs/security-guide.md#スペースIDによるクエリスコープ]**: `CreateSuggestion` クエリに `id` カラムが含まれていない。作業計画書のテーブル設計では `id (ULID)` と記載されているが、マイグレーションで `DEFAULT generate_ulid()` が設定されているためINSERT時にはDBが自動生成する想定と思われる。これ自体は問題ないが、確認のため記録する。

  クエリ自体のセキュリティは問題なし。すべてのクエリに `space_id` が含まれており、スペースIDによるクエリスコープのガイドラインに従っている。

  **対応方針**:
  - [x] 確認のみ（対応不要）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/repository/suggestion.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - ModelとRepositoryの1:1関係、WithTxパターン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

1. **FindByID の `sql.ErrNoRows` の扱い**: `FindByID` は `sql.ErrNoRows` の場合に `nil, nil` を返している。これは呼び出し元で「見つからなかった」と「エラーが発生した」を区別する方法として有効だが、プロジェクト内の既存リポジトリの慣習と一致しているか確認が必要。

   既存の他のリポジトリ（例: `TopicRepository`, `SpaceRepository`）での `sql.ErrNoRows` の扱いと一致していれば問題ない。

   **対応方針**:
   - [ ] 既存パターンと一致しているため対応不要
   - [ ] 既存パターンに合わせて修正（下の回答欄に記入）
   - [x] その他（下の回答欄に記入）

   **回答**:

   ```
   既存パターンを確認し、一致していなければ修正をお願いします。
   ```

2. **`UpdateStatus` で `sql.ErrNoRows` の処理が欠けている**: `UpdateStatus` メソッドでは、指定した `id` と `space_id` の組み合わせで行が見つからない場合（存在しないIDやスペースをまたいだアクセスを試みた場合）、`sql.ErrNoRows` エラーがそのまま返される。`FindByID` では `sql.ErrNoRows` を `nil, nil` に変換しているのに対し、`UpdateStatus` では異なる動作になる。

   ```go
   // 現在のコード（UpdateStatus）
   row, err := r.q.UpdateSuggestionStatus(ctx, ...)
   if err != nil {
       return nil, err  // sql.ErrNoRows もそのまま返す
   }
   ```

   **修正案**:

   一貫性のために `FindByID` と同じパターンにするか、UseCase層で `sql.ErrNoRows` を処理するか、設計方針を確認する。

   **対応方針**:
   - [x] `FindByID` と同様に `sql.ErrNoRows` を `nil, nil` に変換する
   - [ ] UseCase層で処理するため現状のまま
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

タスク 1a-2 の要件（suggestionsテーブルのsqlcクエリとリポジトリ）を正確に実装している。

**良い点**:

- 既存のリポジトリパターン（コンストラクタ、`WithTx`、`toModel`/`toModels`）に完全に一致している
- すべてのSQLクエリに `space_id` 条件が含まれ、セキュリティガイドラインのスペースIDスコープに準拠している
- ドメインID型（`model.SuggestionID` 等）が正しく使用されている
- テストが充実しており、CRUD各操作・境界ケース（存在しないID、異なるスペースID、空のスライス）がカバーされている
- テストビルダーが既存パターン（`TopicBuilder` 等）と一貫している
- 作業計画書のテーブル設計・API設計と整合が取れている

**確認事項**:

- `UpdateStatus` の `sql.ErrNoRows` 処理方針について要確認（軽微）
