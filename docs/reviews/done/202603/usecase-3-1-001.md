# コードレビュー: usecase-3-1

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-27                                           |
| 対象ブランチ               | usecase-3-1                                          |
| ベースブランチ             | usecase-2-1                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 9 ファイル                                           |
| 変更行数（実装）           | +200 / -108 行                                       |
| 変更行数（テスト）         | +170 / -202 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャ、UseCase
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラー
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーション
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [ ] `go/internal/usecase/create_suggestion.go`
- [x] `go/internal/validator/suggestion.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`
- [ ] `go/internal/usecase/create_suggestion_test.go`
- [x] `go/internal/validator/suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/create_suggestion.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase、認可チェック
- [作業計画書](/workspace/docs/plans/1_doing/usecase-orchestration-refactor.md) - 設計との整合性

**問題点・改善提案**:

- **[作業計画書 タスク3-1]**: authorize メソッドが TopicPolicy を使用していない

  作業計画書のタスク3-1には「UseCase に SuggestionCreateValidator + **TopicPolicy** を統合」と明記されている。しかし、`authorize` メソッド（L149-L173）は認可ロジックをインラインで実装しており、既存の `internal/policy/topic.go` の `TopicPolicy` インフラを使用していない。

  `TopicPolicy` には `CanCreateSuggestion` メソッドが未定義のため、追加が必要になる。ただし、既存の認可パターン（`NewTopicPolicy(spaceMember, topicMember)` → ポリシーメソッド呼び出し）と整合させることで、認可ロジックの一元管理が実現できる。

  ```go
  // 現在の実装（インライン認可）
  func (uc *CreateSuggestionUsecase) authorize(ctx context.Context, space *model.Space, spaceMember *model.SpaceMember, topic *model.Topic) error {
      if spaceMember == nil {
          return &model.AppError{Code: model.AppErrCodeForbidden, ...}
      }
      if topic.Visibility == model.TopicVisibilityPrivate {
          if spaceMember.Role != model.SpaceMemberRoleOwner {
              topicMember, err := uc.topicMemberRepo.FindBySpaceMemberAndTopic(...)
              // ...
          }
      }
      return nil
  }
  ```

  **修正案**:
  1. `internal/policy/topic.go` の `TopicPolicy` に `CanCreateSuggestion` メソッドを追加
  2. `authorize` メソッドで `policy.NewTopicPolicy(spaceMember, topicMember)` を使用

  ```go
  func (uc *CreateSuggestionUsecase) authorize(ctx context.Context, space *model.Space, spaceMember *model.SpaceMember, topic *model.Topic) error {
      if spaceMember == nil {
          return &model.AppError{Code: model.AppErrCodeForbidden, UserMsg: i18n.T(ctx, "error_forbidden")}
      }

      var topicMember *model.TopicMember
      if spaceMember.Role != model.SpaceMemberRoleOwner {
          var err error
          topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
          if err != nil {
              return fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
          }
      }

      topicPolicy := policy.NewTopicPolicy(spaceMember, topicMember)
      if !topicPolicy.CanCreateSuggestion(topic) {
          return &model.AppError{Code: model.AppErrCodeForbidden, UserMsg: i18n.T(ctx, "error_forbidden")}
      }
      return nil
  }
  ```

  **対応方針**:
  - [x] TopicPolicy に CanCreateSuggestion を追加し、UseCase から使用する
  - [ ] 現在のインライン実装のまま維持する（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[@go/docs/coding-guide.md]**: `createSuggestion` メソッドの引数が多すぎる（10個）

  `createSuggestion` メソッド（L213）のシグネチャは `ctx` を除いて10個の引数を持っており、可読性が低い。

  ```go
  func (uc *CreateSuggestionUsecase) createSuggestion(ctx context.Context, spaceID model.SpaceID, spaceIdentifier model.SpaceIdentifier, topicID model.TopicID, spaceMemberID model.SpaceMemberID, title, body, bodyHTML string, draftPages []*model.DraftPage, pageRevisions map[model.PageID]*model.PageRevision) (*CreateSuggestionOutput, error)
  ```

  **修正案**:

  `createSuggestion` 用の入力構造体を定義する。

  ```go
  type createSuggestionParams struct {
      SpaceID         model.SpaceID
      SpaceIdentifier model.SpaceIdentifier
      TopicID         model.TopicID
      SpaceMemberID   model.SpaceMemberID
      Title           string
      Body            string
      BodyHTML        string
      DraftPages      []*model.DraftPage
      PageRevisions   map[model.PageID]*model.PageRevision
  }

  func (uc *CreateSuggestionUsecase) createSuggestion(ctx context.Context, params createSuggestionParams) (*CreateSuggestionOutput, error) {
  ```

  **対応方針**:
  - [ ] 構造体を導入して引数を整理する
  - [ ] 現状のまま維持する（理由を回答欄に記入）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  構造体を導入したいです。
  入力構造体は他に定義されていませんか？サフィックスにInputを使った構造体があった気がするのですが、Paramsのほうが良いでしょうか？
  ```

### `go/internal/usecase/create_suggestion_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストカバレッジ
- [作業計画書](/workspace/docs/plans/1_doing/usecase-orchestration-refactor.md) - 設計との整合性

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#テストカバレッジ]**: 既存の正常系テストが削除されている

  以下の2つのテストケースが削除されたが、テスト対象の機能自体は残っている:
  1. **「正常系: Wikiリンクが解決される」**（旧L220-L280）: 編集提案の本文に含まれるWikiリンクが解決されることを検証するテスト。`renderBodyHTML` 内の `resolveLinkedPages` と `markup.ReplaceWikilinks` の統合テストとして重要。
  2. **「異常系: ページリビジョンが存在しない場合はエラー」**（旧L282-L320）: `fetchLatestPageRevisions` が未公開ページ（リビジョンなし）に対してエラーを返すことを検証するテスト。

  UseCase の入力形式が変わったことで書き直しが必要なのは理解できるが、これらの機能カバレッジ自体は維持すべきである。

  **修正案**:

  新しい入力形式に合わせて、上記2つのテストケースを書き直して追加する。

  **対応方針**:
  - [x] 新しい入力形式でテストを書き直して追加する
  - [ ] これらのテストは不要と判断する（理由を回答欄に記入）
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

作業計画書のタスク3-1「create_suggestion UseCaseの移行」として、Handler → UseCase へのオーケストレーション責務移動が適切に実装されている。主な変更点は以下の通り:

- Handler が薄い Adapter になり、HTTP の入出力変換に徹している
- UseCase が データ取得 → 認可 → バリデーション → ビジネスロジック → 永続化 の流れを統括している
- Validator の返り値が `(data, error)` パターンに変更され、`model.ValidationError` を使用している
- エラーハンドリングが `model.AsValidationError` / `model.AsAppError` パターンに統一されている

指摘事項は2点:

1. `authorize` メソッドが既存の `TopicPolicy` インフラを使用していない点（作業計画書との乖離）
2. 既存テストの一部が削除されカバレッジが低下している点

いずれも軽微であり、方針を確認の上で対応すれば問題ない。
