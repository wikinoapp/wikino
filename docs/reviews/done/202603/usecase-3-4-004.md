# コードレビュー: usecase-3-4

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-27                                           |
| 対象ブランチ               | usecase-3-4                                          |
| ベースブランチ             | usecase-3-3                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 24 ファイル                                          |
| 変更行数（実装）           | +756 / -853 行                                       |
| 変更行数（テスト）         | +342 / -478 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page/update.go`
- [x] `go/internal/handler/draft_page_revision/handler.go`
- [x] `go/internal/handler/draft_page_revision/update.go`
- [x] `go/internal/handler/page/handler.go`
- [ ] `go/internal/handler/page/update.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/get_save_draft_page_data.go`（削除）
- [x] `go/internal/usecase/manual_save_draft_page.go`
- [x] `go/internal/usecase/page_access.go`（新規）
- [x] `go/internal/usecase/publish_page.go`
- [x] `go/internal/validator/page.go`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`
- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/usecase/auto_save_draft_page_test.go`
- [x] `go/internal/usecase/manual_save_draft_page_test.go`
- [x] `go/internal/usecase/publish_page_test.go`
- [x] `go/internal/validator/page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`
- [x] `docs/reviews/done/usecase-3-4-001.md`
- [x] `docs/reviews/usecase-3-4-002.md`
- [x] `docs/reviews/usecase-3-4-003.md`

## ファイルごとのレビュー結果

### `go/internal/handler/page/update.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Handler での処理フロー
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ファイル命名規則
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **[コーディング規約: ログ出力]**: `handleUpdateError` 内で `model.AsAppError` の `default` ブランチでは `ae.LogString()` を使ってログ出力しているが、最後のフォールバック（素の `error`）では `"ページの公開に失敗"` というメッセージに `err` を直接渡している。これは作業計画書の「予期しないエラー → 500」パターンと一致しているため問題ないが、`handleUpdateError` 内で `getPageDetailUC.Execute` が失敗した場合にログ出力後に `http.Error` を返しているのに対し、`getErr` が `nil` で `output` も `nil` の場合（到達しにくいが論理上あり得る）のケースも同じパスを通る点は問題ない

  ただし、`handleUpdateError` 内でバリデーションエラー時にフォーム再描画用データを再取得する際、`getPageDetailUC.Execute` のエラーログに元の `err`（バリデーションエラー）の情報が含まれていない。デバッグ時にどのバリデーションエラーだったかの文脈が失われる可能性がある。

  **修正案**:

  ```go
  if getErr != nil || output == nil {
      slog.ErrorContext(ctx, "フォーム再表示用データの取得に失敗", "error", getErr, "original_error", ve.Error())
      http.Error(w, "Internal Server Error", http.StatusInternalServerError)
      return
  }
  ```

  **対応方針**:
  - [x] ログにバリデーションエラーの文脈を追加する
  - [ ] 現状のまま（バリデーションエラーの内容はフォームに表示されるため不要）
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

作業計画書の方針に沿った、高品質なリファクタリングです。主な変更内容：

1. **UseCase のオーケストレーター化**: `auto_save_draft_page.go`、`manual_save_draft_page.go`、`publish_page.go` の3つの書き込み UseCase に、データ取得・認可チェック・（publish_page のみ）バリデーションを統合。Handler から Policy/読み取り UseCase の呼び出しを除去し、UseCase に集約。

2. **共通処理の抽出**: `page_access.go` に `fetchPageAccessData` と `authorizePageUpdate` を定義し、3つの UseCase で共通利用。DRYかつ命名も適切。

3. **Validator の `(data, error)` 返しへの変更**: `PageUpdateValidator` が `session.FormErrors` ベースの Result 型を廃止し、`(*model.PageID, error)` を返すように変更。Go の慣習に従った自然なインターフェース。

4. **`GetSaveDraftPageDataUsecase` の削除**: 書き込み UseCase がデータ取得を内包するようになったため、不要になった読み取り UseCase を適切に削除。

5. **Input の簡素化**: 各 UseCase の Input が `SpaceIdentifier` + `PageNumber` + `UserID` の3つに統一され、Handler が内部 ID を知る必要がなくなった。

6. **テストの適切な更新**: すべてのテストが新しいインターフェースに合わせて更新されており、`TopicMember` の作成も追加されている（認可チェックが UseCase 内に移ったため必要になった）。

全体として、設計との乖離なし、セキュリティ問題なし、アーキテクチャルールへの違反なし。唯一の軽微な指摘は `handleUpdateError` 内のログ情報量の点のみで、必須対応ではありません。
