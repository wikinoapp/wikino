# コードレビュー: usecase-3-1

## レビュー情報

| 項目                       | 内容                                                              |
| -------------------------- | ----------------------------------------------------------------- |
| レビュー日                 | 2026-03-27                                                        |
| 対象ブランチ               | usecase-3-1                                                       |
| ベースブランチ             | usecase-2-1                                                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク3-1） |
| 変更ファイル数             | 13 ファイル（ドキュメント除く）                                   |
| 変更行数（実装）           | +237 / -102 行                                                    |
| 変更行数（テスト）         | +238 / -159 行                                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/create_suggestion.go`
- [ ] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/validator/suggestion.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/usecase/create_suggestion_test.go`
- [x] `go/internal/validator/suggestion_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion/create.go`

**ステータス**: 対応済み（案Aを採用: `handler.NotFound` に統一）

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Handler の責務
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - エラーハンドリング
- パイロット実装（suggestion_comment/create.go）との一貫性

**問題点・改善提案**:

- **Forbidden 時のレスポンスが suggestion_comment/create.go と不一致**: `handleCreateError` で `AppErrCodeForbidden` の場合に `handler.NotFound(w, r)` を返しているが、パイロット実装の `suggestion_comment/create.go:handleError` では `http.Error(w, "Forbidden", http.StatusForbidden)` を返している。

  ```go
  // suggestion/create.go（本PR）
  case model.AppErrCodeForbidden:
      handler.NotFound(w, r) // NotFoundで隠す
  ```

  ```go
  // suggestion_comment/create.go（パイロット実装 2-1）
  case model.AppErrCodeForbidden:
      http.Error(w, "Forbidden", http.StatusForbidden) // 実際の403を返す
  ```

  **修正案**:

  どちらかに統一する。セキュリティ観点では Forbidden をリソースの存在を隠す NotFound として返すのは正当だが、プロジェクト全体で一貫したパターンにすべき。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 案A: `handler.NotFound(w, r)` に統一する（suggestion_comment 側も変更）
  - [ ] 案B: `http.Error(w, "Forbidden", http.StatusForbidden)` に統一する（本PR側を変更）
  - [ ] 案C: リソース種別によって使い分ける方針を明文化する（現状のまま）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

タスク3-1の要件（create_suggestion UseCaseにSuggestionCreateValidator + TopicPolicyを統合）は正しく実装されている。

**良い点**:

- UseCase の `Execute` メソッドが作業計画書に沿った処理順序（1.データ取得 → 2.認可チェック → 3.バリデーション → 4.ビジネスロジック → 5.永続化）で実装されている
- `fetchData`, `authorize`, `renderBodyHTML`, `createSuggestion` とメソッドに適切に分割され、Execute内にロジックを直接書かないルールが守られている
- Handler が薄いAdapterになり、UseCase呼び出し → エラーハンドリング → レスポンスのみに簡素化されている
- `CreateSuggestionInput` が外部キー（SpaceID, TopicID等）ではなく識別子（SpaceIdentifier, TopicNumber, UserID）を受け取るよう変更され、Handler の責務が明確に軽減されている
- Validator が `(data, error)` の2値返しに変更され、作業計画書の方針通り
- `validationErrorToFormErrors` を一時的なブリッジとして明示的にコメントしている点は良い
- Policy の `CanCreateSuggestion` が全実装（owner, admin, member, guest）に追加され、guest では公開トピックのみ許可する適切なロジックになっている
- UseCase テストに認可・バリデーションの異常系テストが追加されている
- トランザクション用の内部型 `createSuggestionInput` を分離し、公開 `CreateSuggestionInput` と区別している設計が良い

**指摘事項**:

- Forbidden時のレスポンスがsuggestion_comment handler（パイロット実装）と不一致。方針を決定すれば修正は軽微
