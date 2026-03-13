# コードレビュー: handler-usecase-refactor-6-3

## レビュー情報

| 項目                       | 内容                                                                          |
| -------------------------- | ----------------------------------------------------------------------------- |
| レビュー日                 | 2026-03-13                                                                    |
| 対象ブランチ               | handler-usecase-refactor-6-3                                                  |
| ベースブランチ             | handler-usecase-refactor-6-2                                                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md                                |
| 変更ファイル数             | 8 ファイル                                                                    |
| 変更行数（実装）           | +62 / -14 行（main.go, handler.go, index.go, usecase, architecture-guide.md） |
| 変更行数（テスト）         | +95 / -3 行（index_test.go, get_draft_pages_test.go）                         |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/get_draft_pages.go`
- [x] `go/internal/handler/draft_page_index/handler.go`
- [x] `go/internal/handler/draft_page_index/index.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/usecase/get_draft_pages_test.go`
- [x] `go/internal/handler/draft_page_index/index_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`
- [x] `go/docs/architecture-guide.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 設計との整合性チェック

### タスク 6-3 の要件との比較

タスク 6-3 は「draft_page_index, draft_page_revision ハンドラーの UseCase 化」です。

- **draft_page_index**: この PR で `GetDraftPagesUsecase` を作成し、Handler の Repository 直接依存を UseCase 経由に変更済み ✅
- **draft_page_revision**: 先行する PR ですでに UseCase 経由に変更済みであることを確認 ✅（`GetPageDetailUsecase` と `ManualSaveDraftPageUsecase` を使用）
- **Handler 構造体から Repository フィールドを削除**: `handler.go` から `repository` import が除去され、`usecase` import に置換済み ✅
- **main.go の更新**: `GetDraftPagesUsecase` の構築と Handler への注入が追加済み ✅
- **テスト追加・更新**: UseCase テスト（正常系 2 ケース）とハンドラーテスト（3 ケース）の更新済み ✅
- **architecture-guide.md の更新**: 読み取り UseCase の `Get` プレフィックス統一ルールを追記 ✅

設計との乖離はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 6-3（draft_page_index, draft_page_revision ハンドラーの UseCase 化）が正しく実装されています。

- `GetDraftPagesUsecase` は既存の読み取り UseCase（`GetTopicDetailUsecase`, `GetPageDetailUsecase` 等）と一貫したパターンで実装されている
- Handler から `repository` パッケージへの直接依存が完全に除去され、`usecase` パッケージ経由に統一されている
- UseCase の命名規則（`Get` プレフィックス統一）がアーキテクチャガイドに追記され、ドキュメントも整合している
- テストは正常系・異常系をカバーしており、既存テストも UseCase 経由に正しく更新されている
- 変更量が適切な範囲（実装 +62/-14 行）に収まっている
