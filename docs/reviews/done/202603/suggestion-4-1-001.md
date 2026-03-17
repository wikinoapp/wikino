# コードレビュー: suggestion-4-1

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-17                       |
| 対象ブランチ               | suggestion-4-1                   |
| ベースブランチ             | suggestion-3-2                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 7 ファイル                       |
| 変更行数（実装）           | +332 行                          |
| 変更行数（テスト）         | +203 行                          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/suggestions.sql`
- [x] `go/internal/query/suggestions.sql.go`（sqlc自動生成）
- [x] `go/internal/repository/suggestion.go`
- [x] `go/internal/usecase/get_suggestion_detail.go`
- [x] `go/internal/viewmodel/suggestion.go`

### テストファイル

- [ ] `go/internal/usecase/get_suggestion_detail_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/get_suggestion_detail_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 権限チェックパターン

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#エラーケースを必ずテスト]**: 非公開トピックのアクセス制御がテストされていない

  UseCase内に非公開トピックの権限チェックロジック（`TopicVisibilityPrivate`の場合、スペースオーナーまたはトピックメンバーのみ閲覧可能）があるが、テストケースではすべて `UserID` を未設定（nil）で実行しており、以下のケースがカバーされていない:
  1. 非公開トピック + 未ログインユーザー → nilが返る
  2. 非公開トピック + スペースメンバーだがトピックメンバーでない → nilが返る
  3. 非公開トピック + スペースオーナー → 正常に取得できる
  4. 非公開トピック + トピックメンバー → 正常に取得できる

  **修正案**:

  以下のテストケースを追加する:

  ```go
  t.Run("非公開トピックでスペースオーナーは閲覧できる", func(t *testing.T) {
      // 非公開トピックを作成し、スペースオーナーのUserIDを渡して正常取得を確認
  })

  t.Run("非公開トピックでトピックメンバーでないユーザーはnilが返る", func(t *testing.T) {
      // 非公開トピックを作成し、トピックメンバーでないユーザーのUserIDを渡してnil返却を確認
  })
  ```

  **対応方針**:
  - [x] テストケースを追加する
  - [ ] 現状のまま（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[@go/docs/testing-guide.md#エラーケースを必ずテスト]**: ログインユーザーのフロー（SpaceMember/TopicMemberの取得）がテストされていない

  すべてのテストケースで `UserID` が未設定のため、`Execute` 内の `SpaceMember` / `TopicMember` 取得処理（lines 78-102）が一度も実行されていない。正常系のテストでも `UserID` を渡して、Output の `SpaceMember` や `TopicMember` が正しくセットされることを確認すべき。

  **修正案**:

  既存の正常系テストの一部を `UserID` を渡す形に変更するか、ログインユーザーの正常系テストを追加する:

  ```go
  t.Run("ログインユーザーのSpaceMemberとTopicMemberが取得できる", func(t *testing.T) {
      output, err := uc.Execute(context.Background(), GetSuggestionDetailInput{
          SpaceIdentifier:  "sug-detail-space",
          TopicNumber:      1,
          SuggestionNumber: 1,
          UserID:           &userID,
      })
      // SpaceMember, TopicMember が nil でないことを確認
  })
  ```

  **対応方針**:
  - [x] ログインユーザーのテストケースを追加する
  - [ ] 現状のまま（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

タスク4-1（編集提案詳細のUseCase・ViewModel）の実装として、作業計画書に記載された要件を満たしている。実装コードの品質は高く、既存のパターン（`GetTopicDetailUsecase`, `GetSuggestionListUsecase`）と一貫性が保たれている。

**良い点**:

- `buildUserMap` メソッドによるN+1クエリの回避（バッチ取得パターン）
- 非公開トピックの権限チェックが既存パターンと完全に一致
- ViewModelがInput構造体パターンに統一されており可読性が高い
- セキュリティガイドラインに準拠（すべてのクエリがspace_idでスコープされている）
- sqlcクエリの `FindSuggestionByTopicAndNumber` がspace_idを含めている

**修正が必要な点**:

テストにおいて、非公開トピックのアクセス制御とログインユーザーのフロー（SpaceMember/TopicMember取得）がカバーされていない。UseCase内の重要なロジック（権限チェック、SpaceMember取得）が未テストとなっている。
