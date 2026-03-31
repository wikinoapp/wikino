# コードレビュー: usecase-3-1

## レビュー情報

| 項目                       | 内容                                                               |
| -------------------------- | ------------------------------------------------------------------ |
| レビュー日                 | 2026-03-27                                                         |
| 対象ブランチ               | usecase-3-1                                                        |
| ベースブランチ             | usecase-2-1                                                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク 3-1） |
| 変更ファイル数             | 16 ファイル（ドキュメント除く）                                    |
| 変更行数（実装）           | +236 / -103 行                                                     |
| 変更行数（テスト）         | +248 / -153 行                                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ、UseCase
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラー
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーション
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティ

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion_comment/create.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [ ] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/usecase/create_suggestion.go`
- [x] `go/internal/validator/suggestion.go`

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [ ] `go/internal/usecase/create_suggestion_test.go`
- [x] `go/internal/validator/suggestion_test.go`
- [x] `go/internal/testutil/draft_page_builder.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/policy/topic_guest.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - レイヤーごとのテストカバレッジ（Policy: 必須）

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#レイヤーごとのテストカバレッジ]**: `CanCreateSuggestion` メソッドのテストが `topic_test.go` に追加されていない

  `topicGuestPolicy.CanCreateSuggestion` はトピックの Visibility を検査する非自明なロジックを含んでいる（`topic.Visibility != model.TopicVisibilityPrivate`）。テストガイドでは Policy テストは「必須」と定められており、既存の `TestNewTopicPolicy_Guest` 等に `CanCreateSuggestion` のテストケースを追加すべき。他の 3 ロール（Owner, Admin, Member）についても同様。

  ```go
  // topic_guest.go の該当ロジック
  func (p *topicGuestPolicy) CanCreateSuggestion(topic *model.Topic) bool {
      return p.spaceMemberActive && topic.Visibility != model.TopicVisibilityPrivate
  }
  ```

  **修正案**:

  `topic_test.go` の各ロールのテスト関数に `CanCreateSuggestion` のテストケースを追加する。特に guest では以下のケースが必要：
  - 公開トピック → true
  - プライベートトピック → false
  - 非アクティブメンバー → false

  **対応方針**:
  - [x] テストを追加する
  - [ ] テスト追加を別タスクに分離する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/create_suggestion_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - エラーケースを必ずテスト

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#テストのベストプラクティス]**: 存在しないトピックのテストケースが不足

  `fetchData` メソッドは `space == nil` と `topic == nil` の両方で `AppErrCodeResourceNotFound` を返す。存在しないスペースのテスト（「異常系: 存在しないスペースの場合 AppError が返る」）は追加されているが、存在しないトピック（スペースは存在するがトピック番号が無効）のテストケースがない。

  **修正案**:

  ```go
  t.Run("異常系: 存在しないトピックの場合AppErrorが返る", func(t *testing.T) {
      t.Parallel()

      spaceID := testutil.NewSpaceBuilderDB(t, db).
          WithIdentifier("create-sug-notopic").
          Build()

      userID := testutil.NewUserBuilderDB(t, db).
          WithEmail("create-sug-notopic@example.com").
          WithAtname("createsugnotopic").
          Build()

      _, err := uc.Execute(context.Background(), CreateSuggestionInput{
          SpaceIdentifier: "create-sug-notopic",
          TopicNumber:     999,
          UserID:          userID,
          Title:           "テスト",
          Body:            "",
          DraftPageIDs:    []model.DraftPageID{"dummy"},
      })

      ae := model.AsAppError(err)
      if ae == nil {
          t.Fatal("expected AppError, got nil")
      }
      if ae.Code != model.AppErrCodeResourceNotFound {
          t.Errorf("Code = %d, want %d", ae.Code, model.AppErrCodeResourceNotFound)
      }
  })
  ```

  **対応方針**:
  - [x] テストを追加する
  - [ ] 現状のカバレッジで十分（理由を回答欄に記入）
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

作業計画書のタスク 3-1（create_suggestion UseCase の移行）が設計通りに実装されている。パイロット（タスク 2-1）で確立したパターン（UseCase がオーケストレーター、Handler は `errors.As` で判別）が一貫して適用されており、コードの可読性も良好。

**良い点**:

- UseCase 内の処理順序（データ取得 → 認可 → バリデーション → ビジネスロジック → トランザクション）が計画通りに実装されている
- `CreateSuggestionInput` が HTTP レベルの識別子（SpaceIdentifier, TopicNumber, UserID）のみを受け取り、内部 ID は UseCase が解決する設計が適切
- `validationErrorToFormErrors` ブリッジ関数が一時的な措置であることがコメントで明記されている
- UseCase テストに認可・バリデーションの異常系テストが新たに追加されている
- `suggestion_comment/create.go` の Forbidden → NotFound 変更はセキュリティ上の改善

**指摘事項**:

- Policy の `CanCreateSuggestion` テストが未追加（guest の Visibility チェックは非自明なロジック）
- UseCase テストで存在しないトピックのケースが未カバー
