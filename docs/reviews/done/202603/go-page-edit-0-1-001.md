# コードレビュー: go-page-edit-0-1

## レビュー情報

| 項目                       | 内容                                       |
| -------------------------- | ------------------------------------------ |
| レビュー日                 | 2026-03-07                                 |
| 対象ブランチ               | go-page-edit-0-1                           |
| ベースブランチ             | go-page-edit                               |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-rollout.md |
| 変更ファイル数             | 9 ファイル                                 |
| 変更行数（実装）           | +40 / -2 行                                |
| 変更行数（テスト）         | +220 / -8 行                               |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/draft_page_revisions.sql`
- [x] `go/internal/query/draft_page_revisions.sql.go`
- [x] `go/internal/repository/draft_page_revision.go`
- [x] `go/internal/usecase/publish_page.go`

### テストファイル

- [x] `go/internal/repository/draft_page_revision_test.go`
- [ ] `go/internal/testutil/draft_page_revision_builder.go`
- [x] `go/internal/usecase/publish_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/page-edit-go-rollout.md`

## ファイルごとのレビュー結果

### `go/internal/testutil/draft_page_revision_builder.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストヘルパー
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 既存コードとの一貫性

**問題点・改善提案**:

- **既存パターンとの不一致**: `DraftPageRevisionBuilderDB` には DB 版（`*sql.DB` を使用）のみが実装されていますが、`draft_page_builder.go` では `DraftPageBuilder`（`*sql.Tx` を使用）と `DraftPageBuilderDB`（`*sql.DB` を使用）の両方が実装されています。`DraftPageRevisionRepository` のテスト（`draft_page_revision_test.go`）ではトランザクションベースのビルダーを使用しておらず、直接 `repo.Create()` でテストデータを作成しているため、現時点では DB 版のみで問題ありませんが、将来的にトランザクションベースのテストが必要になった場合に `*sql.Tx` 版のビルダーが不足します。

  **修正案**:

  現時点では DB 版のみの使用箇所しかないため、このままでも問題ありません。ただし、他のビルダーとの一貫性を重視する場合は `DraftPageRevisionBuilder`（`*sql.Tx` 版）も追加する方針です。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 現状のまま（DB 版のみで十分）
  - [ ] `*sql.Tx` 版も追加する
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

作業計画書のタスク 0-1 に記載された要件通りに実装されています。

**良かった点**:

- SQLクエリに `space_id` 条件が含まれており、セキュリティガイドラインのスペースIDによるクエリスコープに準拠している
- `WithTx` パターンによるトランザクション管理がアーキテクチャガイドに準拠している
- UseCase の依存性注入が既存パターンと一貫している
- 外部キー制約違反を再現するテストケース（`TestPublishPageUsecase_Execute_WithDraftPageRevisions`）が追加されており、バグの再発を防止できる
- リポジトリテストでも `DeleteByDraftPageID` の動作確認が行われている

**軽微な指摘**:

- テストビルダーが DB 版のみだが、現時点の使用箇所を考えると問題なし（上記の要確認項目を参照）
