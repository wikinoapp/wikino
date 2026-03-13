# コードレビュー: handler-usecase-refactor-7-0

## レビュー情報

| 項目                       | 内容                                                         |
| -------------------------- | ------------------------------------------------------------ |
| レビュー日                 | 2026-03-13                                                   |
| 対象ブランチ               | handler-usecase-refactor-7-0                                 |
| ベースブランチ             | handler-usecase-refactor-6-3                                 |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md（タスク 7-0） |
| 変更ファイル数             | 7 ファイル                                                   |
| 変更行数（実装）           | +29 / -13 行                                                 |
| 変更行数（テスト）         | +9 / -3 行                                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/draft_page/update.go`
- [x] `go/internal/handler/page_move/new.go`
- [x] `go/internal/handler/page_move/create.go`
- [x] `go/internal/usecase/get_page_move_data.go`
- [x] `go/internal/usecase/get_save_draft_page_data.go`

### テストファイル

- [x] `go/internal/usecase/get_page_move_data_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

### レビュー詳細

**UseCase の変更（`get_page_move_data.go`, `get_save_draft_page_data.go`）**:

- `policy` パッケージへの依存が正しく除去されている ✅
- `TopicMember` が Output に追加され、Handler で認可チェックに使用できるようになっている ✅
- `topicMember` の取得後に `nil` チェックで即時リターンしていない（`nil` は「トピックメンバーではない」を意味する有効な状態）のは正しい設計 ✅

**Handler の変更（`draft_page/update.go`, `page_move/new.go`, `page_move/create.go`）**:

- `policy` パッケージを import し、認可チェックを Handler で実行している ✅
- 認可失敗時のレスポンスは 404（情報漏洩防止）で統一されている ✅
- UseCase の Output から `SpaceMember` と `TopicMember` を使って Policy を構築するパターンが 3 箇所で一貫している ✅

**テストの変更（`get_page_move_data_test.go`）**:

- 「権限がないユーザーで nil が返る」テストが「トピックメンバーでないユーザーでもデータが返る」に正しく変更されている ✅
- Output に `TopicMember` フィールドのアサーションが追加されている ✅

## 設計との整合性チェック

作業計画書タスク 7-0 の要件との整合性を確認しました。

| 要件                                                                             | 状態 |
| -------------------------------------------------------------------------------- | ---- |
| `GetSaveDraftPageDataUsecase`: Policy チェックを削除、Output に TopicMember 追加 | ✅   |
| `GetPageMoveDataUsecase`: Policy チェックを削除、Output に TopicMember 追加      | ✅   |
| `draft_page/update.go`: Handler で Policy チェックを実行                         | ✅   |
| `page_move/new.go`: Handler で Policy チェックを実行                             | ✅   |
| `page_move/create.go`: Handler で Policy チェックを実行                          | ✅   |
| テスト更新                                                                       | ✅   |

**テストカバレッジ**: `get_save_draft_page_data_test.go` は存在しないが、Handler テスト（`draft_page/update_test.go` の `TestUpdate_TopicPolicyDenied`）が認可チェックの動作を間接的にカバーしているため、問題なし。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 7-0（UseCase から Policy 依存を除去し、認可チェックを Handler に移動）が作業計画書通りに実装されています。変更は小さく焦点が絞られており、以下の点が評価できます：

- UseCase から `policy` パッケージの import が完全に除去され、UseCase の責務がデータ取得に集中している
- Handler での認可チェックパターンが 3 箇所で一貫しており、可読性が高い
- 既存の Handler テスト（`TestUpdate_TopicPolicyDenied` 等）がそのまま動作し、認可チェックが Handler に移動しても振る舞いが変わらないことが確認できる
- UseCase テストも適切に更新されている
