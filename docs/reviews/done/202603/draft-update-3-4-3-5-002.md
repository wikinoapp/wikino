# コードレビュー: draft-update-3-4-3-5

## レビュー情報

| 項目                       | 内容                                 |
| -------------------------- | ------------------------------------ |
| レビュー日                 | 2026-03-06                           |
| 対象ブランチ               | draft-update-3-4-3-5                 |
| ベースブランチ             | draft-update-3-3                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md   |
| 変更ファイル数             | 17 ファイル                          |
| 変更行数（実装）           | +218 / -82 行（自動生成 templ 含む） |
| 変更行数（テスト）         | +172 / -98 行                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page_revision/handler.go`
- [ ] `go/internal/handler/draft_page_revision/update.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/manual_save_draft_page.go`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/main_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`
- [x] `go/internal/usecase/manual_save_draft_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/draft-update.md`
- [x] `docs/reviews/done/202603/draft-update-3-4-3-5-001.md`

## ファイルごとのレビュー結果

### `go/internal/handler/draft_page_revision/update.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#Handler構造体の定義]**: Handler 構造体に `topicRepo` と `topicMemberRepo` が含まれていますが、`update.go` 内ではトピックの取得（`topicRepo`）とトピックメンバーの取得（`topicMemberRepo`）のどちらも使用しています。ただし、`topicRepo` はトピック名を取得するためだけに使用されており、このトピック名は `ManualSaveDraftPageInput.CurrentTopicName` に渡されています。一方で、ページ自体が `pg.TopicID` を持っているため、ハンドラー内でトピックを再取得する必要性は妥当です。問題なし。

  **実際の問題**: `update.go` の 97-103 行目でトピックを取得していますが、これはフォームパラメータ取得（93-94行目）の後に配置されています。認可チェック後・フォームパラメータ取得後のため処理順序自体は問題ありませんが、エラーハンドリングのパターンは他のハンドラーと一致しており問題ありません。

  取り下げ: 上記は問題ではありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3-4 および 3-5 の作業計画書に沿った実装が適切に行われています。

**良い点**:

- `draft_page/create.go` → `draft_page_revision/update.go` へのリソース分離がハンドラーガイドラインに準拠している（PATCH = `update.go`）
- `saveDraftPageContent` 関数の抽出により、`AutoSaveDraftPageUsecase` と `ManualSaveDraftPageUsecase` のコード重複が適切に解消されている
- 3つの Input 構造体（`AutoSaveDraftPageInput`, `ManualSaveDraftPageInput`, `saveDraftPageContentInput`）のフィールドが完全に一致しており、型変換 `saveDraftPageContentInput(input)` が安全に動作する
- テンプレートの `formaction` 方式への変更により、別フォーム + `onsubmit` JS の同期が不要になり、シンプルになった
- テストが正常系・異常系ともに適切にカバーされている（未ログイン、無効なページ番号、スペース不在、DraftPage未作成時の find_or_create）
- `ErrDraftPageNotFound` の削除と find_or_create 方式への統一が作業計画書の意図に合致している
- リバースプロキシのパターン追加（`draft_page_revision`, `/drafts`）が漏れなく行われている
