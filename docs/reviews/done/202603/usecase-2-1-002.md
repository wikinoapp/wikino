# コードレビュー: usecase-2-1

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-27                                           |
| 対象ブランチ               | usecase-2-1                                          |
| ベースブランチ             | usecase-1-3                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 13 ファイル（実装 10 + テスト 3、ドキュメント除く）  |
| 変更行数（実装）           | +165 / -71 行                                        |
| 変更行数（テスト）         | +281 / -28 行                                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_comment/create.go`
- [x] `go/internal/handler/suggestion_comment/handler.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/usecase/create_suggestion_comment.go`
- [x] `go/internal/validator/suggestion_comment.go`

### テストファイル

- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [x] `go/internal/usecase/create_suggestion_comment_test.go`
- [x] `go/internal/validator/suggestion_comment_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`
- [x] `docs/reviews/done/usecase-2-1-001.md`

## ファイルごとのレビュー結果

問題のあるファイルのみ記載。

### `go/internal/policy/topic_admin.go` / `topic_member.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#認可チェック](/workspace/go/docs/architecture-guide.md) - Policy の責務
- [@go/docs/security-guide.md#認証・認可](/workspace/go/docs/security-guide.md) - 認可チェック

**問題点・改善提案**:

- **`CanCreateSuggestionComment` のスコープチェックが `CanUpdateSuggestionComment` と異なる**

  `topicAdminPolicy` と `topicMemberPolicy` の `CanCreateSuggestionComment` は `p.spaceMemberActive` のみをチェックしており、`p.topicID == suggestion.TopicID` を確認していない。一方 `CanUpdateSuggestionComment` はトピックIDの一致を確認している。

  ```go
  // CanCreateSuggestionComment - topicIDチェックなし
  func (p *topicAdminPolicy) CanCreateSuggestionComment(_ *model.Suggestion) bool {
      return p.spaceMemberActive
  }

  // CanUpdateSuggestionComment - topicIDチェックあり
  func (p *topicAdminPolicy) CanUpdateSuggestionComment(suggestion *model.Suggestion) bool {
      return p.spaceMemberActive && p.topicID == suggestion.TopicID && suggestion.Status == model.SuggestionStatusOpen
  }
  ```

  リファクタリング前のハンドラーでは「スペースメンバーであればコメント可能」という仕様だったため、トピックIDチェックをしないのは**既存仕様の忠実な再現**と理解できる。ただし、意図的にこの差異がある場合は、将来の開発者が混乱しないようコメントがあると良い。

  **修正案**:

  意図的な仕様であることが確認できれば修正不要。

  **対応方針**:
  - [ ] 意図的な仕様のため現状維持（回答欄に理由を記入）
  - [x] トピックIDチェックを追加する
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

作業計画書のタスク 2-1（パイロット: create_suggestion_comment UseCase にバリデーション・認可を統合）が正確に実装されている。

**良い点**:

- UseCase の `Execute` メソッドが「1. データ取得 → 2. 認可チェック → 3. バリデーション → 4. ビジネスロジック → 5. 永続化」の順序を明確に守っており、作業計画書の設計通り
- Handler が HTTP の入出力変換に徹しており、`handleError` メソッドで `model.AsValidationError` / `model.AsAppError` / 素の `error` の3パターンを正しくハンドリング
- Validator の返り値が Result 型から `error` に変更され、Go の標準的なエラーハンドリングパターンに統一
- UseCase テスト（240行）が正常系2件・異常系3件を網羅し、各エラー型の判別も検証
- Handler テストも UseCase の内部にバリデーション・認可が移動したことに合わせて適切に更新（テストデータの追加含む）
- Policy の `CanCreateSuggestionComment` が全4つのポリシー型に追加されている
- `main.go` の DI 構成が正しく更新されている
