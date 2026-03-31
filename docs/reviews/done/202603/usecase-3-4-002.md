# コードレビュー: usecase-3-4

## レビュー情報

| 項目                       | 内容                                                               |
| -------------------------- | ------------------------------------------------------------------ |
| レビュー日                 | 2026-03-27                                                         |
| 対象ブランチ               | usecase-3-4                                                        |
| ベースブランチ             | usecase-3-3                                                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク 3-4） |
| 変更ファイル数             | 19 ファイル（ドキュメント 2 ファイルを除く）                       |
| 変更行数（実装）           | 約 +500 / -400 行                                                  |
| 変更行数（テスト）         | 約 +400 / -450 行                                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [ ] `go/internal/handler/draft_page/update.go`
- [x] `go/internal/handler/draft_page_revision/handler.go`
- [x] `go/internal/handler/draft_page_revision/update.go`
- [x] `go/internal/handler/page/handler.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/get_save_draft_page_data.go`（削除）
- [x] `go/internal/usecase/manual_save_draft_page.go`
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

## ファイルごとのレビュー結果

### `go/internal/handler/draft_page/update.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約（一貫性）
- [@CLAUDE.md#既存コードとの一貫性](/workspace/CLAUDE.md) - 実装時のガイドライン

**問題点・改善提案**:

- **既存コードとの一貫性**: NotFound レスポンスの返し方が他のハンドラーと不一致。`draft_page_revision/update.go`（62 行目）や `page/update.go`（94 行目）は `handler.NotFound(w, r)` を使用しているが、`draft_page/update.go`（57 行目）は `http.Error(w, "Not Found", http.StatusNotFound)` を使用している。コードベース全体で `handler.NotFound` が標準パターン。

  ```go
  // 現状（draft_page/update.go:57）
  case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
      http.Error(w, "Not Found", http.StatusNotFound)
  ```

  **修正案**:

  ```go
  case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
      handler.NotFound(w, r)
  ```

  なお、`handler` パッケージの import 追加も必要。

  **対応方針**:
  - [x] 修正案の通り `handler.NotFound(w, r)` に変更する
  - [ ] 現状のまま（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

タスク 3-4（publish_page, manual_save_draft_page, auto_save_draft_page の UseCase 移行）が作業計画書に従って正しく実装されている。主要な変更点は以下のとおり：

- **UseCase がオーケストレーターに**: 3 つの書き込み UseCase すべてで `fetchData` → `authorize` → 永続化の統一パターンが適用されており、作業計画書の設計方針と一致
- **Handler の薄型化**: Handler から `policy`、`validator` への直接依存が削除され、UseCase のエラーを `model.AsAppError` / `model.AsValidationError` で判別する新パターンに移行済み
- **Validator の返り値変更**: `PageUpdateValidator` が `*PageUpdateValidatorResult` → `(*model.PageID, error)` に変更され、Go の慣習的な 2 値返しパターンに準拠
- **GetSaveDraftPageDataUsecase の削除**: データ取得が各書き込み UseCase 内に統合されたため、不要になった読み取り UseCase が適切に削除されている
- **テストの更新**: 入力パラメータの変更に合わせてテストが正しく更新されており、`TestPublishPageUsecase_Execute_NilTitle` は空タイトルの ValidationError テストに変更

指摘事項は `draft_page/update.go` の NotFound レスポンス方法の不一致 1 件のみ。軽微な一貫性の問題であり、修正は任意。
