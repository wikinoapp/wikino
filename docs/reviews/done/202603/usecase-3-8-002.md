# コードレビュー: usecase-3-8

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3-8                                          |
| ベースブランチ             | usecase-3-7                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 14 ファイル                                          |
| 変更行数（実装）           | +312 / -132 行                                       |
| 変更行数（テスト）         | +362 / -81 行                                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_close/handler.go`
- [x] `go/internal/handler/suggestion_close/create.go`
- [x] `go/internal/handler/suggestion_page_edit/create.go`
- [x] `go/internal/usecase/close_suggestion.go`
- [x] `go/internal/usecase/start_suggestion_page_edit.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_close/create_test.go`
- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/usecase/close_suggestion_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`
- [x] `docs/reviews/done/usecase-3-8-001.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/start_suggestion_page_edit.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase の認可チェック
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - 認可

**問題点・改善提案**:

- **作業計画書との乖離: TopicPolicy が統合されていない**

  作業計画書のタスク 3-8 には「2つの UseCase に TopicPolicy を統合」と記載されているが、`StartSuggestionPageEditUsecase` では `policy` パッケージを import しておらず、TopicPolicy によるきめ細かい認可チェックが行われていない。

  `CloseSuggestionUsecase` は `policy.NewTopicPolicy` を使用して `CanCloseSuggestion` を呼び出しているのに対し、`StartSuggestionPageEditUsecase` は `spaceMember == nil` のチェックのみで、「スペースメンバーであれば誰でも編集開始できる」という認可になっている。

  変更前の Handler コードでも TopicPolicy は使用されていなかったが、`GetSuggestionDetailUsecase` を経由していたため、非公開トピックの visibility チェック（`TopicVisibilityPrivate` の場合、スペースオーナーまたはトピックメンバーのみ閲覧可能）が暗黙的に行われていた。新しい UseCase ではこのチェックが失われている可能性がある。

  以下の 2 点について確認が必要:
  1. 非公開トピックに属する編集提案のページを、トピックメンバーでないスペースメンバーが編集開始できてしまう可能性はないか
  2. 意図的に TopicPolicy を省略したのであれば、作業計画書の記載を更新すべき

  **対応方針**:
  - [x] 非公開トピックの visibility チェックを追加し、TopicPolicy を統合する
  - [ ] 現状の「スペースメンバーであれば編集開始可能」は意図通りで、作業計画書の記載を修正する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/close_suggestion.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase のルール

**問題点・改善提案**:

- **`checkIdempotency` と `checkStatus` の責務分離について**

  `Execute` メソッドで `checkIdempotency`（べき等性チェック）と `checkStatus`（ステータスチェック）が連続して呼ばれている。`checkIdempotency` は `*CloseSuggestionOutput` を返し、`checkStatus` は `error` を返すという異なるシグネチャになっている。

  ```go
  // 3. ステータスチェック
  if output := uc.checkIdempotency(data.suggestion); output != nil {
      return output, nil
  }
  if err := uc.checkStatus(ctx, data.suggestion); err != nil {
      return nil, err
  }
  ```

  この 2 つのメソッドは論理的には「ステータスに基づく事前チェック」という同じ責務で、分岐は以下の 3 パターン:
  - Closed → べき等に成功（output を返す）
  - Open → 続行
  - その他（Applied 等）→ Conflict エラー

  1 つのメソッドに統合すると Execute の見通しが良くなる可能性がある。ただし現状でも動作は正しく、既に他の UseCase で同様のパターンが確立されている場合はこのままでも問題ない。

  **対応方針**:
  - [x] 1 つのメソッド（例: `checkStatusForClose`）に統合して `(*CloseSuggestionOutput, error)` を返す
  - [ ] 現状のまま（分離した方が単一責務の原則に沿う）
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

作業計画書タスク 3-8 の方針に沿って、`CloseSuggestionUsecase` と `StartSuggestionPageEditUsecase` にデータ取得・認可・ステータスチェックが統合され、Handler が HTTP 入出力変換に徹する薄い Adapter に変更されている。全体的な設計は良好。

良かった点:

- Handler から `getSuggestionDetailUsecase` と `policy` パッケージへの依存が除去され、UseCase 呼び出し + `handleCreateError` パターンで統一されている
- `AppError` のエラーコードに基づく Handler のエラーハンドリングが一貫している
- テストが充実しており、正常系・異常系（NotFound, Forbidden, Conflict, べき等性）を網羅している
- Handler テストも UseCase が独自トランザクション管理をする前提で `GetTestDB()` パターンに正しく切り替えられている

確認が必要な点:

- `StartSuggestionPageEditUsecase` に TopicPolicy が統合されていない点。非公開トピックの visibility チェックが変更前は `GetSuggestionDetailUsecase` 経由で暗黙的に行われていた可能性があり、セキュリティ上の確認が必要
