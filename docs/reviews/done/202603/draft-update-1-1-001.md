# コードレビュー: draft-update-1-1

## レビュー情報

| 項目                       | 内容                               |
| -------------------------- | ---------------------------------- |
| レビュー日                 | 2026-03-05                         |
| 対象ブランチ               | draft-update-1-1                   |
| ベースブランチ             | draft-update                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md |
| 変更ファイル数             | 5 ファイル                         |
| 変更行数（実装）           | +89 / -2 行                        |
| 変更行数（テスト）         | +0 / -0 行                         |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（ドメインID型、モデル定義）
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約（コメント）
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - 開発環境ガイド（DBマイグレーション）
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン（スペースIDによるクエリスコープ）

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260305154013_create_draft_page_revisions.sql`
- [ ] `go/internal/model/draft_page_revision.go`
- [x] `go/internal/model/id.go`

### 設定・その他

- [x] `go/db/schema.sql`（自動生成）
- [x] `docs/plans/1_doing/draft-update.md`（タスクリストのチェック更新のみ）

## ファイルごとのレビュー結果

### `go/internal/model/draft_page_revision.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#ドメインID型](/workspace/go/docs/architecture-guide.md) - ドメインID型の使用
- [@go/docs/architecture-guide.md#モデルの重複を避ける](/workspace/go/docs/architecture-guide.md) - モデル設計

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#ドメインID型]**: 作業計画書の設計セクションでは `DraftPageRevision` の `DraftPageID` と `SpaceMemberID` フィールドの型が `string` として記載されているが、実装ではドメインID型（`DraftPageID`, `SpaceMemberID`）を使用している。実装のほうがガイドラインに正しく従っている。ただし、作業計画書の設計セクションのコード例（146〜156行目）が `string` を使用しており、実装と乖離がある。

  作業計画書の設計セクション:

  ```go
  type DraftPageRevision struct {
  	ID            string
  	DraftPageID   string
  	SpaceMemberID string
  	// ...
  }
  ```

  実装:

  ```go
  type DraftPageRevision struct {
  	ID            DraftPageRevisionID
  	DraftPageID   DraftPageID
  	SpaceMemberID SpaceMemberID
  	// ...
  }
  ```

  **修正案**: 作業計画書は設計時の概念的な記載であり、実装がガイドラインに従っているため問題なし。今後の作業計画書更新時に修正してもよいが、必須ではない。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 作業計画書のコード例を修正する
  - [ ] 現状のまま（作業計画書は概念的な記載なので許容）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/db/migrations/20260305154013_create_draft_page_revisions.sql`: `space_id` カラムの追加

**ステータス**: 要確認

**現状**:

`draft_page_revisions` テーブルには `space_id` カラムがなく、`draft_page_id` と `space_member_id` のみで参照先テーブルと紐づいている。

```sql
CREATE TABLE draft_page_revisions (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    draft_page_id UUID NOT NULL REFERENCES draft_pages(id),
    space_member_id UUID NOT NULL REFERENCES space_members(id),
    -- space_id がない
);
```

**提案**:

[@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md) では、スペース内リソースへのクエリに `space_id` を WHERE 条件に含めることが推奨されている。`draft_pages` テーブルは `space_id` カラムを持っているが、`draft_page_revisions` は持っていない。

`draft_page_revisions` は `draft_pages` 経由でスペースに紐づくため、JOIN でスペースをスコープすることは可能（`page_attachment_references` と同様のパターン）。ただし、直接 `space_id` カラムを持たせることで、JOIN なしでスペーススコープのクエリが書けるようになる。

```sql
CREATE TABLE draft_page_revisions (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    draft_page_id UUID NOT NULL REFERENCES draft_pages(id),
    space_id UUID NOT NULL REFERENCES spaces(id),
    space_member_id UUID NOT NULL REFERENCES space_members(id),
    -- ...
);
```

**メリット**:

- JOIN なしでスペーススコープのクエリが書ける
- セキュリティガイドラインの「防御の多層化」に沿っている

**トレードオフ**:

- `draft_pages` にすでに `space_id` があるため、データの冗長性が増す
- `page_attachment_references` は `space_id` を持たず JOIN で解決するパターンが既存コードに存在するため、一貫性の観点からは JOIN パターンでも問題ない
- 今後のクエリパターン（フェーズ 2, 3）で必要になった時点で追加しても遅くない

**対応方針**:

<!-- 開発者が回答を記入してください -->

- [x] `space_id` カラムを追加する
- [ ] 現状のまま（JOIN パターンで十分）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Approve

**総評**:

タスク 1-1（DraftPageRevision のDBマイグレーションとモデル定義）の実装として適切。作業計画書の要件を満たしている。

- マイグレーションファイルはガイドラインに従っている（VARCHAR長さ指定なし、TIMESTAMP WITH TIME ZONE使用、ULID使用）
- インデックス `[draft_page_id, created_at]` が作業計画書の設計通り
- モデル定義はドメインID型を正しく使用しており、既存モデルとの一貫性がある
- `id.go` の型定義と `String()` メソッドも既存パターンに従っている
- テストなしは作業計画書で想定済み（テスト 0）

設計改善として `space_id` カラムの追加を提案しているが、既存の `page_attachment_references` パターンに倣い JOIN で解決するのも妥当な選択肢であり、必須ではない。
