# コードレビュー: usecase-3-6

## レビュー情報

| 項目                       | 内容                                                               |
| -------------------------- | ------------------------------------------------------------------ |
| レビュー日                 | 2026-03-30                                                         |
| 対象ブランチ               | usecase-3-6                                                        |
| ベースブランチ             | usecase-3-5                                                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク 3-6） |
| 変更ファイル数             | 9 ファイル                                                         |
| 変更行数（実装）           | +168 / -109 行                                                     |
| 変更行数（テスト）         | +262 / -156 行                                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ、UseCase の責務
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約、ログ出力
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_page/handler.go`
- [x] `go/internal/handler/suggestion_page/update.go`
- [ ] `go/internal/usecase/update_suggestion_page.go`
- [x] `go/internal/validator/suggestion_page.go`

### テストファイル

- [x] `go/internal/handler/suggestion_page/update_test.go`
- [x] `go/internal/usecase/update_suggestion_page_test.go`
- [x] `go/internal/validator/suggestion_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/update_suggestion_page.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase の構造体定義
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **未使用フィールド `topicRepo`**: `topicRepo` フィールドが構造体に宣言され（20 行目）、コンストラクタで注入（33, 44 行目）されているが、`fetchData`・`authorize`・`updateSuggestionPage` のいずれのメソッドでも使用されていない。同じ移行パターンの `move_page.go` では `topicRepo` を `pageAccessRepos()` ヘルパー内で使用しており、`create_suggestion.go` でも `fetchData` 内で使用している。`update_suggestion_page.go` では不要。

  ```go
  // 問題のあるコード（20行目、33行目、44行目）
  type UpdateSuggestionPageUsecase struct {
      db                         *sql.DB
      spaceRepo                  *repository.SpaceRepository
      spaceMemberRepo            *repository.SpaceMemberRepository
      topicRepo                  *repository.TopicRepository       // 未使用
      topicMemberRepo            *repository.TopicMemberRepository
      // ...
  }
  ```

  **修正案**:

  構造体、コンストラクタ引数、初期化から `topicRepo` を削除する。`main.go` の呼び出し箇所も合わせて更新する。

  ```go
  type UpdateSuggestionPageUsecase struct {
      db                         *sql.DB
      spaceRepo                  *repository.SpaceRepository
      spaceMemberRepo            *repository.SpaceMemberRepository
      topicMemberRepo            *repository.TopicMemberRepository
      suggestionRepo             *repository.SuggestionRepository
      suggestionPageRepo         *repository.SuggestionPageRepository
      suggestionPageRevisionRepo *repository.SuggestionPageRevisionRepository
      updateValidator            *validator.SuggestionPageUpdateValidator
  }
  ```

  **対応方針**:
  - [x] 修正案の通り `topicRepo` を削除する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/internal/usecase/update_suggestion_page.go` + `go/internal/handler/suggestion_page/update.go`: Validator のセンチネルエラーのハンドリング

**ステータス**: 要確認

**現状**:

`SuggestionPageUpdateValidator` は `ErrDraftPageNotFound` と `ErrDraftPageNotLinked` というセンチネルエラー（素の `error`）を返す。UseCase はこれらをそのまま `return nil, err` で呼び出し元に返す。Handler の `handleUpdateError` は `model.AppError` のみチェックし、それ以外は全て 500 Internal Server Error として処理する。

```go
// handler/suggestion_page/update.go（64-80行目）
func (h *Handler) handleUpdateError(...) {
    if ae := model.AsAppError(err); ae != nil {
        // AppError の処理
        return
    }
    // ここに到達 → 500
    slog.ErrorContext(ctx, "編集提案ページの更新に失敗", "error", err)
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
```

結果として、ユーザーの下書きが存在しない・リンクされていないケースが 500 として処理される。

**提案**:

UseCase の `Execute` メソッド内で、Validator のセンチネルエラーを `model.AppError` に変換する。これにより Handler は既存の `handleUpdateError` のパターンで適切に処理できる。

```go
// Execute内のバリデーション呼び出し後
draftPage, err := uc.updateValidator.Validate(ctx, ...)
if err != nil {
    if errors.Is(err, validator.ErrDraftPageNotFound) || errors.Is(err, validator.ErrDraftPageNotLinked) {
        return nil, &model.AppError{
            Code:     model.AppErrCodeConflict,
            UserMsg:  i18n.T(ctx, "error_suggestion_page_update_conflict"),
            Internal: err,
        }
    }
    return nil, err
}
```

**メリット**:

- ユーザーに 500 ではなく適切なエラーレスポンスを返せる
- Handler のエラーハンドリングパターンが統一される
- 他の移行済み UseCase（`create_suggestion` 等）との一貫性が向上する

**トレードオフ**:

- これらのエラーは稀な競合状態（下書きが削除された等）でのみ発生するため、500 でも実用上の問題は小さい
- 新しい翻訳キーの追加が必要になる

**対応方針**:

- [x] 提案通り、UseCase 内でセンチネルエラーを AppError に変換する
- [ ] 現状のまま（500 で問題ない）
- [ ] 今回は対象外とし、別タスクで対応する
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Request Changes

**総評**:

作業計画書タスク 3-6 の要件通り、`update_suggestion_page` UseCase にバリデーション・認可が適切に統合されている。Handler は HTTP の入出力に徹する薄い Adapter に変わり、UseCase が「データ取得 → 認可 → バリデーション → 永続化」のフローを統括する設計に移行できている。

**良い点**:

- UseCase の `Execute` メソッドが `fetchData` → `authorize` → `Validate` → `updateSuggestionPage` と明確に分割されており、可読性が高い
- Handler から `getSuggestionDetailUsecase`、`validator` の直接依存が除去され、`usecase` のみに依存する構造になっている
- テストが正常系・異常系（スペース不在、非メンバー、クローズ済み、反映済み）を網羅している
- 作業計画書の `errors.As` パターン、Validator の `(data, error)` 2 値返しパターンに準拠している

**必須対応**:

- `topicRepo` の未使用フィールドを削除する（1 件）
