# コードレビュー: draft-update-1-2

## レビュー情報

| 項目                       | 内容                               |
| -------------------------- | ---------------------------------- |
| レビュー日                 | 2026-03-05                         |
| 対象ブランチ               | draft-update-1-2                   |
| ベースブランチ             | draft-update                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md |
| 変更ファイル数             | 6 ファイル                         |
| 変更行数（実装）           | +135 / -1 行                       |
| 変更行数（テスト）         | +130 / -0 行                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/draft_page_revisions.sql`
- [x] `go/internal/query/draft_page_revisions.sql.go`（自動生成）
- [x] `go/internal/query/models.go`（自動生成）
- [x] `go/internal/repository/draft_page_revision.go`

### テストファイル

- [x] `go/internal/repository/draft_page_revision_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/draft-update.md`

## ファイルごとのレビュー結果

問題のあるファイルはありませんでした。すべてのファイルがガイドラインに準拠しています。

### レビュー詳細（問題なし）

**`go/db/queries/draft_page_revisions.sql`**:

- SQLクエリでspace_idを含めている（セキュリティガイドラインのスペースIDスコープに準拠）
- RETURNING \* で結果を返しており、sqlcのパターンに従っている

**`go/internal/query/draft_page_revisions.sql.go`** / **`go/internal/query/models.go`**:

- sqlcの自動生成コード。DraftPageRevisionモデルにspace_idが含まれている

**`go/internal/repository/draft_page_revision.go`**:

- 既存の`PageRevisionRepository`と同じパターンに従っている
- ドメインID型を正しく使用（`model.DraftPageID`, `model.SpaceID`, `model.SpaceMemberID`）
- `WithTx`メソッドが実装されている（アーキテクチャガイドに準拠）
- `toModel`メソッドで`query`型から`model`型への変換を実施
- `Create`メソッドで`time.Now()`を使用（既存パターンと一致）
- コメントは日本語で記載されている

**`go/internal/repository/draft_page_revision_test.go`**:

- `t.Parallel()`を使用（テストガイドに準拠）
- `testutil.SetupTx(t)`でトランザクション分離（テストガイドに準拠）
- ビルダーパターンを使用してテストデータを作成
- 正常系テスト2件: 作成と複数リビジョン作成
- 各フィールドの検証が網羅的

## 設計との整合性チェック

作業計画書のタスク1-2に記載された要件との整合性を確認しました。

| 要件                                                         | 状態 |
| ------------------------------------------------------------ | ---- |
| `db/queries/draft_page_revisions.sql`にCreateクエリを追加    | ✅   |
| `internal/repository/draft_page_revision.go`にリポジトリ実装 | ✅   |
| Create, WithTxメソッドの実装                                 | ✅   |
| sqlcコード生成                                               | ✅   |
| テスト                                                       | ✅   |

作業計画書の設計仕様との一貫性:

- DraftPageRevisionモデルのフィールドが作業計画書の設計と一致している（SpaceIDは前PRで追加されたカラム）
- モデル定義（タスク1-1で追加済み）に対応するリポジトリが正しく実装されている

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク1-2（DraftPageRevisionのsqlcクエリとリポジトリ）が作業計画書通りに実装されています。既存の`PageRevisionRepository`のパターンに忠実に従っており、コードベース全体の一貫性が保たれています。セキュリティガイドラインに準拠してspace_idがクエリに含まれており、ドメインID型も正しく使用されています。テストは正常系を適切にカバーしています。
