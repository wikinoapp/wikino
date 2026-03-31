# コードレビュー: usecase-2-1

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-27                                           |
| 対象ブランチ               | usecase-2-1                                          |
| ベースブランチ             | usecase-1-3                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 9 ファイル                                           |
| 変更行数（実装）           | +130 / -71 行                                        |
| 変更行数（テスト）         | +280 / -29 行                                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/create_suggestion_comment.go`
- [x] `go/internal/handler/suggestion_comment/create.go`
- [x] `go/internal/handler/suggestion_comment/handler.go`
- [x] `go/internal/validator/suggestion_comment.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/usecase/create_suggestion_comment_test.go`
- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [x] `go/internal/validator/suggestion_comment_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/create_suggestion_comment.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase のオーケストレーション責務
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [作業計画書](/workspace/docs/plans/1_doing/usecase-orchestration-refactor.md) - 検討事項 1, 2, 4

**問題点・改善提案**:

- **[@go/docs/i18n-guide.md#判断基準]**: `authorize` メソッドの `AppError.UserMsg` が国際化されていない

  ```go
  // 問題のあるコード（115-123行目）
  func (uc *CreateSuggestionCommentUsecase) authorize(_ context.Context, spaceMember *model.SpaceMember) error {
      if spaceMember == nil {
          return &model.AppError{
              Code:    model.AppErrCodeForbidden,
              UserMsg: "Forbidden",
          }
      }
      return nil
  }
  ```

  `AppError.UserMsg` はユーザーに表示される可能性があるメッセージであり、国際化が必要。作業計画書の検討事項 1「UseCase でのエラー生成パターン」でも `i18n.T(ctx, "error_forbidden")` を使用する例が示されている。また、`ctx` パラメータが `_` で無視されているが、国際化メッセージの取得に必要。

  **修正案**:

  ```go
  func (uc *CreateSuggestionCommentUsecase) authorize(ctx context.Context, spaceMember *model.SpaceMember) error {
      if spaceMember == nil {
          return &model.AppError{
              Code:    model.AppErrCodeForbidden,
              UserMsg: i18n.T(ctx, "error_forbidden"),
          }
      }
      return nil
  }
  ```

  **対応方針**:
  - [x] 修正案の通り `i18n.T(ctx, "error_forbidden")` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/internal/usecase/create_suggestion_comment.go`: Policy パッケージの使用検討

**ステータス**: 要確認

**現状**:

`authorize` メソッドで `spaceMember == nil` の直接チェックを行い、認可判定をしている。元の Handler コードでも同じ nil チェックだったため、振る舞い自体は正しい。

```go
func (uc *CreateSuggestionCommentUsecase) authorize(_ context.Context, spaceMember *model.SpaceMember) error {
    if spaceMember == nil {
        return &model.AppError{Code: model.AppErrCodeForbidden, UserMsg: "Forbidden"}
    }
    return nil
}
```

**提案**:

作業計画書の「変更対象のファイル一覧」では `create_suggestion_comment.go` の対応する Policy が `topic.go` と記載されており、タスク 2-1 の説明にも「Validator・**Policy** の呼び出しを統合」とある。`policy.TopicPolicy` を使用することで、認可ロジックが Policy パッケージに集約され、将来的に権限モデルが変わった場合の変更箇所が限定される。

ただし、元の Handler コードでも nil チェックのみだったため、現時点では実質的な差異はない。Policy に適切なメソッドが存在するかどうかによって判断が変わる。

**メリット**:

- 認可ロジックが Policy パッケージに集約される（一貫性の向上）
- 将来的に権限モデルが変更された場合の影響範囲が限定される

**トレードオフ**:

- 現時点では nil チェックだけで十分であり、Policy を使うとコードがやや冗長になる
- Policy に適切なメソッドが存在しない場合、新たに追加する必要がある

**対応方針**:

- [x] Policy パッケージのメソッドを使用するように変更する
- [ ] 現状のまま（nil チェックで十分）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Comment

**総評**:

全体として、作業計画書の設計方針（UseCase がオーケストレーターとなり、Handler は薄い Adapter に徹する）に正しく従った実装になっている。

**良かった点**:

- Handler が大幅にシンプルになった（バリデーション・読み取り UseCase・認可チェックの直接呼び出しが除去）
- `errors.As` パターンによるエラーハンドリング（`handleError` メソッド）が作業計画書の設計通りに実装されている
- Validator の Result 型廃止と `(data, error)` 返しへの変更が正しく行われている
- UseCase の処理順序（データ取得 → 認可 → バリデーション → ビジネスロジック → 永続化）が計画書の順序と一致
- UseCase テストが正常系・異常系を網羅し、各エラー型（AppError / ValidationError）の検証も行われている
- Handler テストがバリデーションエラー時にも実データ（スペース・トピック・提案）を作成するよう更新されており、UseCase 経由のテストとして適切

**修正が望ましい点**:

- `authorize` メソッドの `UserMsg` が国際化されていない（1 件）
