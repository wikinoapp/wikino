# コードレビュー: suggestion-7-1b

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-19                       |
| 対象ブランチ               | suggestion-7-1b                  |
| ベースブランチ             | suggestion-7-1a                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 10 ファイル                      |
| 変更行数（実装）           | +236 / -168 行                   |
| 変更行数（テスト）         | +0 / -0 行                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260319074138_add_featured_image_attachment_id_to_draft_pages.sql`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/db/schema.sql`
- [x] `go/internal/model/draft_page.go`
- [x] `go/internal/query/draft_pages.sql.go`（自動生成）
- [x] `go/internal/query/models.go`（自動生成）
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`

### テストファイル

（なし）

### 設定・その他

- [x] `go/internal/testutil/draft_page_builder.go`
- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### 設計との整合性: テストファイルの欠如

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@CLAUDE.md#Pull Requestのガイドライン](/workspace/CLAUDE.md) - 実装とテストのセット化
- 作業計画書タスク 7-1b

**問題点・改善提案**:

- **[@CLAUDE.md#実装とテストのセット化]**: 作業計画書のタスク 7-1b では「想定ファイル数: 実装 6, テスト 1」「想定行数: テスト 約50行」と記載されているが、テストファイルが含まれていない

  CLAUDE.mdの「実装とテストのセット化」ルール: 「新機能や修正を行う場合は、必ず対応するテストを追加・更新する」「テストがない実装は原則としてマージしない」

  テスト対象候補:
  - `auto_save_draft_page.go` の `saveDraftPageContent` で `featured_image_attachment_id` が正しく保存されることの検証
  - `draft_page.go` リポジトリの `Create` / `Update` で `FeaturedImageAttachmentID` が正しく永続化・取得されることの検証

  **修正案**:

  テストを追加する（例: `auto_save_draft_page_test.go` または `draft_page_test.go` にテストケース追加）

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] テストを追加する
  - [ ] 後続タスク 7-1c でまとめてテストを追加する（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

タスク 7-1b の実装内容自体は正確で、既存のコードパターンに一貫して従っている。

- **マイグレーション**: シンプルで正しい（nullable UUID カラム追加）
- **モデル**: 既存の `FeaturedImageAttachmentID` パターン（`page.go`, `suggestion_page.go`）と一致
- **リポジトリ**: `toModel`, `Create`, `Update` の全てで nullable ポインタ型の変換を正しく実装。`toDraftPagesFromMemberTopicRows` にはマッピングが追加されていないが、`SuggestionPageID` も同様に未マッピングであり、既存パターンと一貫している
- **ユースケース**: `saveDraftPageContent` で `extractFeaturedImageAttachmentID` を呼び出し、`linked_page_ids` と同じタイミングで保存するよう正しく更新
- **テストビルダー**: `DraftPageBuilder` と `DraftPageBuilderDB` の両方に `WithFeaturedImageAttachmentID` メソッドを追加済み

唯一の問題点はテストファイルの欠如。作業計画書ではテスト 1 ファイル（約50行）が想定されており、CLAUDE.md のガイドラインでもテストのセット化が求められている。
