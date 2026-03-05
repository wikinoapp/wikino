# コードレビュー: draft-update-1-1

## レビュー情報

| 項目                       | 内容                               |
| -------------------------- | ---------------------------------- |
| レビュー日                 | 2026-03-05                         |
| 対象ブランチ               | draft-update-1-1                   |
| ベースブランチ             | draft-update                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md |
| 変更ファイル数             | 6 ファイル                         |
| 変更行数（実装）           | +100 / -4 行                       |
| 変更行数（テスト）         | +0 / -0 行                         |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（ドメインID型、モデル定義）
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約（コメント）
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - 開発環境ガイド（DBマイグレーション、カラム定義）
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン（スペースIDによるクエリスコープ）

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260305154013_create_draft_page_revisions.sql`
- [x] `go/internal/model/draft_page_revision.go`
- [x] `go/internal/model/id.go`

### 設定・その他

- [x] `go/db/schema.sql`（自動生成）
- [x] `docs/plans/1_doing/draft-update.md`（タスクチェック更新 + コード例修正）
- [x] `docs/reviews/done/202603/draft-update-1-1-001.md`（前回レビューの対応記録）

## ファイルごとのレビュー結果

すべてのファイルに問題なし。

前回レビュー（001）で指摘された2点が正しく対応されている:

1. **`space_id` カラムの追加**: マイグレーションとモデルの両方に `space_id` が追加されている。セキュリティガイドラインのスペースIDによるクエリスコープに対応。
2. **作業計画書のコード例修正**: `string` → ドメインID型（`DraftPageRevisionID`, `DraftPageID`, `SpaceMemberID`）に更新済み。

各ファイルの確認結果:

- **マイグレーション**: ULID使用、VARCHAR長さ指定なし、TIMESTAMP WITH TIME ZONE使用、インデックス `[draft_page_id, created_at]` が作業計画書通り。`space_id` に外部キー制約あり。
- **モデル定義**: ドメインID型を正しく使用、既存モデル（`PageRevision`, `DraftPage`）との一貫性あり。`SpaceID` フィールドの追加は作業計画書のデータモデル表にはないが、セキュリティガイドラインへの対応として妥当。
- **id.go**: `DraftPageRevisionID` 型と `String()` メソッドが既存パターンに従って追加されている。配置位置も `PageRevisionID` の直後で論理的。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 1-1（DraftPageRevision のDBマイグレーションとモデル定義）の実装として適切。前回レビュー（001）で指摘された `space_id` カラムの追加と作業計画書のコード例修正の両方が正しく対応されている。

- マイグレーション、モデル定義、ドメインID型すべてがガイドラインに従っている
- 既存の `PageRevision`, `DraftPage` モデルとの一貫性がある
- 作業計画書の要件を満たしている
- テストなしは作業計画書で想定済み（テスト 0）
