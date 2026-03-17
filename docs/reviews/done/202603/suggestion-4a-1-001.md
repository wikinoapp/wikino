# コードレビュー: suggestion-4a-1

## レビュー情報

| 項目                       | 内容                               |
| -------------------------- | ---------------------------------- |
| レビュー日                 | 2026-03-17                         |
| 対象ブランチ               | suggestion-4a-1                    |
| ベースブランチ             | suggestion-4-2                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md   |
| 変更ファイル数             | 8 ファイル                         |
| 変更行数（実装）           | +29 / -6 行（ドキュメント除く）    |
| 変更行数（テスト）         | +0 / -0 行（テストビルダーは実装） |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260317100418_change_suggestion_number_scope_to_space.sql`
- [x] `go/db/queries/suggestions.sql`
- [x] `go/db/schema.sql`
- [x] `go/internal/query/suggestions.sql.go`
- [x] `go/internal/repository/suggestion.go`
- [x] `go/internal/usecase/create_suggestion.go`
- [x] `go/internal/testutil/suggestion_builder.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

各ファイルの確認内容:

- **マイグレーション**: `DROP INDEX IF EXISTS` → `CREATE UNIQUE INDEX` の順序で適切。`migrate:down` でロールバック可能。インデックス名も命名規則に一致
- **SQLクエリ**: `GetNextSuggestionNumber` が `topic_id` → `space_id` に正しく変更されている。コメントも日本語で更新済み
- **sqlc生成コード**: クエリの変更が正しく反映されている（自動生成のため手動編集なし）
- **リポジトリ**: `GetNextNumber` の引数が `topicID model.TopicID` → `spaceID model.SpaceID` に変更。ドメインID型を正しく使用
- **ユースケース**: `GetNextNumber` の呼び出しが `input.TopicID` → `input.SpaceID` に変更。コメントも更新済み
- **テストビルダー**: `SuggestionBuilder` と `SuggestionBuilderDB` の両方で番号採番クエリを `topic_id` → `space_id` に変更
- **作業計画書**: タスク 4a-1 のチェックボックスにチェック。「採用しなかった方針」に「編集提案番号をトピック内で採番する案」を追加。理由が明確に記載されている

## 設計との整合性チェック

### 作業計画書との整合性

タスク 4a-1 の要件:

| 要件                                                                                        | 状況                                                            |
| ------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| マイグレーション: ユニークインデックスを `[topic_id, number]` → `[space_id, number]` に変更 | ✅ 実装済み                                                     |
| `GetNextSuggestionNumber` を `topic_id` → `space_id` ベースに変更                           | ✅ 実装済み                                                     |
| `GetNextNumber` の引数を `topicID` → `spaceID` に変更                                       | ✅ 実装済み                                                     |
| `create_suggestion.go` の `GetNextNumber` 呼び出しを更新                                    | ✅ 実装済み                                                     |
| 関連テストの更新                                                                            | ✅ テストビルダーの番号採番ロジックを更新（直接テスト参照なし） |

未実装や乖離はありません。

### 補足

`FindSuggestionByTopicAndNumber` クエリが `topic_id` と `number` で検索する形式のまま残っているが、これはタスク 4a-2（編集提案詳細URLから `topics/{topic}` を削除）で `FindBySpaceAndNumber` に置き換えられる予定のため、現時点では問題ありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 4a-1 の要件をすべて満たしており、変更が小さく明確で、一貫性のある修正が行われています。

- マイグレーションの `migrate:up` / `migrate:down` が適切に定義されている
- ドメインID型（`model.SpaceID`）を正しく使用している
- コメントが日本語で適切に更新されている
- テストビルダーの番号採番ロジックも忘れずに更新されている
- 作業計画書の「採用しなかった方針」に変更の理由が明確に記載されている
