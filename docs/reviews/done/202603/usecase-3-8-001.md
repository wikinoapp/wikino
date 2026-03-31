# コードレビュー: usecase-3-8

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3-8                                          |
| ベースブランチ             | usecase-3-7                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 11 ファイル                                          |
| 変更行数（実装）           | +304 / -132 行                                       |
| 変更行数（テスト）         | +362 / -81 行                                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_close/handler.go`
- [x] `go/internal/handler/suggestion_close/create.go`
- [x] `go/internal/handler/suggestion_page_edit/create.go`
- [x] `go/internal/usecase/close_suggestion.go`
- [ ] `go/internal/usecase/start_suggestion_page_edit.go`

### テストファイル

- [x] `go/internal/handler/suggestion_close/create_test.go`
- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/usecase/close_suggestion_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/start_suggestion_page_edit.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase のルール
- [作業計画書](/workspace/docs/plans/1_doing/usecase-orchestration-refactor.md) - 設計との整合性

**問題点・改善提案**:

- **[作業計画書#タスク3-8]**: `checkStatus` メソッド（172-180行目）で、`AppErrCodeConflict` に対して `error_not_found_message` を使用している

  ```go
  // 問題のあるコード (172-180行目)
  func (uc *StartSuggestionPageEditUsecase) checkStatus(ctx context.Context, suggestion *model.Suggestion) error {
      if suggestion.Status != model.SuggestionStatusOpen {
          return &model.AppError{
              Code:    model.AppErrCodeConflict,
              UserMsg: i18n.T(ctx, "error_not_found_message"),
          }
      }
      return nil
  }
  ```

  エラーコードが `AppErrCodeConflict` であるのにメッセージが "not found" 系のため、ログやデバッグ時に混乱する可能性がある。Handler では `AppErrCodeConflict` に対して `changesPath` へリダイレクトしておりメッセージは直接表示されないが、`close_suggestion.go` では `i18n.T(ctx, "suggestion_close_error")` のように適切な Conflict 向けメッセージを使っている。

  **修正案**:

  ```go
  func (uc *StartSuggestionPageEditUsecase) checkStatus(ctx context.Context, suggestion *model.Suggestion) error {
      if suggestion.Status != model.SuggestionStatusOpen {
          return &model.AppError{
              Code:    model.AppErrCodeConflict,
              UserMsg: i18n.T(ctx, "suggestion_page_edit_error_not_open"),
          }
      }
      return nil
  }
  ```

  対応するメッセージが i18n ファイルに存在しない場合は、既存のコンフリクト系メッセージ（`suggestion_close_error` など）の再利用も選択肢。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 専用のコンフリクトメッセージを追加する
  - [ ] 現状のまま（Handler がメッセージを使わないため実害なし）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 設計との整合性チェック

### 作業計画書タスク 3-8 との整合性

作業計画書の記載:

> **3-8**: [Go] close_suggestion, start_suggestion_page_edit UseCase の移行
>
> - 2つの UseCase に TopicPolicy を統合
> - Handler（suggestion_close/create.go, suggestion_page_edit/create.go）を更新

**確認結果**:

- `close_suggestion.go`: TopicPolicy が正しく統合されている ✅
- `start_suggestion_page_edit.go`: TopicPolicy は統合されていない。ただし、移行前の Handler でも TopicPolicy は使用されておらず、スペースメンバーチェックのみが行われていた。移行前の振る舞いが正しく UseCase に移動されている ✅
- 作業計画書の「TopicPolicy を統合」の記載は `close_suggestion` のみに該当し、`start_suggestion_page_edit` は認可が「スペースメンバーチェックのみ」であるため、TopicPolicy の統合対象外だった。**作業計画書の記述が実態より広い**が、実装自体は適切

## 総合評価

**評価**: Approve

**総評**:

タスク 3-8 の要件（close_suggestion と start_suggestion_page_edit の UseCase にオーケストレーション責務を移動）が正しく実装されている。

**良かった点**:

- Handler が HTTP 処理に徹し、UseCase がデータ取得・認可・永続化を統括する新アーキテクチャに正しく移行できている
- `close_suggestion.go` の `checkIdempotency` パターン（べき等性保証）が適切に実装されている
- エラーハンドリングが `model.AppError` / `model.AsAppError` パターンで一貫している
- テストカバレッジが十分（正常系・べき等性・認可失敗・ステータスコンフリクトを網羅）
- Handler テストで `SetupTx` パターンと `GetTestDB` パターンが適切に使い分けられている（UseCase が独自トランザクションを管理するテストでは DB 直接書き込みを使用）

**軽微な指摘**:

- `start_suggestion_page_edit.go` の `checkStatus` で `error_not_found_message` を使用している点のみ（実害なし、改善推奨）
