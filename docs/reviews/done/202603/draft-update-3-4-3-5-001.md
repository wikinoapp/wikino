# コードレビュー: draft-update-3-4-3-5

## レビュー情報

| 項目                       | 内容                               |
| -------------------------- | ---------------------------------- |
| レビュー日                 | 2026-03-06                         |
| 対象ブランチ               | draft-update-3-4-3-5               |
| ベースブランチ             | draft-update-3-3                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md |
| 変更ファイル数             | 15 ファイル                        |
| 変更行数（実装）           | +155 / -50 行                      |
| 変更行数（テスト）         | +258 / -128 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page_revision/handler.go`
- [x] `go/internal/handler/draft_page_revision/update.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`
- [x] `go/internal/templates/path.go`
- [ ] `go/internal/usecase/manual_save_draft_page.go`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/main_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`
- [x] `go/internal/usecase/manual_save_draft_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/draft-update.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/manual_save_draft_page.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Usecaseの責務
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **コード重複**: `ManualSaveDraftPageUsecase.Execute` のステップ1〜7（DraftPage の find_or_create、Markdownレンダリング、Wikiリンク解析、添付ファイルフィルター、DraftPage更新）は `AutoSaveDraftPageUsecase.Execute` とほぼ同一の処理。現時点でこの重複がバグやメンテナンスコストを生んでいるわけではないが、将来的に片方だけ修正して不整合が生じるリスクがある。

  **修正案**:

  共通処理を内部関数に抽出する。例:

  ```go
  // saveDraftPageContent はDraftPageのfind_or_create・Markdown処理・更新を行う共通ロジック
  func saveDraftPageContent(ctx context.Context, input saveDraftPageContentInput, ...) (*model.DraftPage, string, error) {
      // 1. find_or_create
      // 2. Markdownレンダリング
      // 3. Wikiリンク解析
      // 4. リンク変換
      // 5. 添付ファイルフィルター
      // 6. 画像ラッピング
      // 7. DraftPage更新
  }
  ```

  `AutoSaveDraftPageUsecase` と `ManualSaveDraftPageUsecase` の両方からこの共通関数を呼び出す形にする。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 共通関数に抽出する
  - [ ] 現状のまま（重複は許容する）
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

タスク3-4（Usecase拡張）と3-5（formaction方式への変更・draft_page_revisionリソース分離）の実装が作業計画書通りに行われている。

- **良い点**:
  - ハンドラーガイドラインに準拠した `draft_page_revision` リソースの分離（`handler.go` + `update.go` + テスト）
  - `formaction` 属性による公開フォームと下書き保存の統合はシンプルで良い設計
  - `reverse_proxy.go` に `draft_page_revision` パターンが追加されている
  - テストケースが正常系・異常系（未ログイン、無効なページ番号、スペース未存在、DraftPage未存在時のfind_or_create）を網羅
  - 既存の `draft_page` ハンドラーから `manualSaveDraftPageUC` 依存を適切に削除

- **指摘事項**: `ManualSaveDraftPageUsecase.Execute` と `AutoSaveDraftPageUsecase.Execute` の処理ステップ1〜7が実質的に同一コード。軽微な指摘であり、修正は任意。
