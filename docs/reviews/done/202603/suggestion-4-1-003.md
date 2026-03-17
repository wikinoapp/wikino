# コードレビュー: suggestion-4-1 (3回目)

## レビュー情報

| 項目                       | 内容                                |
| -------------------------- | ----------------------------------- |
| レビュー日                 | 2026-03-17                          |
| 対象ブランチ               | suggestion-4-1                      |
| ベースブランチ             | suggestion-3-2                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md    |
| 変更ファイル数             | 9 ファイル（ドキュメント除く）      |
| 変更行数（実装）           | +200 / -38 行（自動生成コード除く） |
| 変更行数（テスト）         | +382 / -0 行                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/suggestions.sql`
- [x] `go/internal/query/suggestions.sql.go`（自動生成）
- [x] `go/internal/repository/suggestion.go`
- [x] `go/internal/usecase/build_user_map.go`
- [x] `go/internal/usecase/get_suggestion_detail.go`
- [x] `go/internal/usecase/get_suggestion_list.go`
- [x] `go/internal/viewmodel/suggestion.go`

### テストファイル

- [x] `go/internal/usecase/get_suggestion_detail_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`（タスクチェックボックス更新）

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク **4-1**（編集提案詳細のUseCase・ViewModel）の実装として、品質の高いコードが提出されています。

**良かった点**:

- **既存パターンとの一貫性**: `GetTopicDetailUsecase` や `GetSuggestionListUsecase` と同じパターン（Space → SpaceMember → Topic → 対象リソースの順序的な取得、not found 時の `nil, nil` 返却）に忠実に従っている
- **セキュリティ**: SQLクエリの `FindSuggestionByTopicAndNumber` に `space_id` スコープが含まれており、セキュリティガイドラインに準拠。非公開トピックの権限チェックも正しく実装されている
- **良いリファクタリング**: `buildUserMapBySpaceMemberIDs` の共通ヘルパー関数への抽出により、`GetSuggestionListUsecase` と `GetSuggestionDetailUsecase` 間のコード重複を排除。空スライスの早期リターンも適切にヘルパー側に配置されている
- **テストの網羅性**: 正常系（詳細取得、ページ取得、コメント取得、UserMap確認、ログインユーザー情報）と異常系（存在しないスペース/トピック/編集提案番号）に加え、非公開トピックのアクセス制御テスト（未ログイン、オーナー、トピックメンバー、非トピックメンバー）が充実している
- **アーキテクチャ準拠**: UseCase → Repository の依存関係が正しく、Query への直接依存がない。ViewModel は Model のみに依存し、表示に必要な情報だけを持つ設計
- **命名規則の遵守**: ファイル名（`get_suggestion_detail.go`）、構造体名（`GetSuggestionDetailUsecase`）、コンストラクタ（`NewGetSuggestionDetailUsecase`）すべてがガイドラインに従っている

**作業計画書との整合性**:

タスク 4-1 で求められている以下の要件をすべて満たしている:

- `GetSuggestionDetailUsecase` の作成（編集提案 + コメント一覧 + 編集ページ一覧取得）: 実装済み
- `SuggestionForDetail`, `SuggestionCommentForList` ViewModel の追加: 実装済み
- 追加で `SuggestionPageForList` ViewModel も実装されており、ハンドラー実装時にすぐ利用できる状態

想定よりファイル数・行数が多いが、これは `FindByTopicAndNumber` クエリ・リポジトリメソッドの追加と `buildUserMapBySpaceMemberIDs` の共通化リファクタリングによるもので、いずれも妥当な追加。
