# コードレビュー: suggestion-1a-1

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-13                       |
| 対象ブランチ               | suggestion-1a-1                  |
| ベースブランチ             | suggestion-0-1                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 5 ファイル                       |
| 変更行数（実装）           | +141 / -2 行（schema.sql含む）   |
| 変更行数（テスト）         | なし（タスク仕様通りテストなし） |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（ドメインID型、Model）
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約（コメント）
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - 開発環境ガイド（マイグレーション、カラム定義）

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/migrations/20260313091316_create_suggestions.sql`
- [x] `go/internal/model/id.go`
- [x] `go/internal/model/suggestion.go`

### 設定・その他

- [x] `go/db/schema.sql`（自動生成）
- [x] `docs/plans/1_doing/suggestion.md`（タスクチェックボックス更新のみ）

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

### レビュー詳細（問題なし）

#### `go/db/migrations/20260313091316_create_suggestions.sql`

**チェックしたガイドライン**:

- [@go/docs/development-guide.md#カラム定義のガイドライン](/workspace/go/docs/development-guide.md) - カラム定義
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティ
- 作業計画書のテーブル設計セクション

**確認結果**:

- ULID使用（`DEFAULT generate_ulid()`）: OK
- `VARCHAR`（長さ指定なし）: OK（ガイドライン準拠）
- `TIMESTAMP WITH TIME ZONE`: OK（ガイドライン準拠）
- FK制約: `spaces(id)`, `topics(id)`, `space_members(id)` すべて設定済み
- インデックス: タスク仕様通り `[topic_id, status]`, `[space_id]`, `[created_space_member_id]` の3つ
- `migrate:down`: インデックスを先にDROPしてからテーブルDROP（正しい順序）
- `status` カラムのデフォルト値 `DEFAULT 0`（Draft）: 作業計画書の仕様通り

#### `go/internal/model/id.go`

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#ドメインID型](/workspace/go/docs/architecture-guide.md) - ドメインID型

**確認結果**:

- `SuggestionID` 型定義と `String()` メソッド: 既存パターン（`AttachmentID` 等）と一致
- コメントは日本語: OK
- 配置位置: `AttachmentID` の後、`FeatureFlagID` の前（適切）

#### `go/internal/model/suggestion.go`

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Modelの設計方針
- [@go/docs/coding-guide.md#コメントのガイドライン](/workspace/go/docs/coding-guide.md) - コメント

**確認結果**:

- `SuggestionStatus` 型は `int32`（既存の `Plan` 型と同じパターン）: OK
- 定数定義: `Draft=0, Open=1, Applied=2, Closed=3`（作業計画書の仕様通り）
- 各定数にコメントあり（日本語）: OK
- `Suggestion` 構造体: ドメインID型を正しく使用（`SuggestionID`, `SpaceID`, `TopicID`, `SpaceMemberID`）
- `AppliedAt` は `*time.Time`（nullable）: OK
- Modelにビジネスロジックなし: OK（ガイドライン準拠）
- インポートは `time` のみ（最小限）: OK

## 設計との整合性チェック

作業計画書のタスク **1a-1** の仕様:

| 仕様項目                                                                                                                    | 状態                                           |
| --------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `go/db/migrations/` にマイグレーション作成                                                                                  | ✓ 実装済み                                     |
| カラム: id, space_id, topic_id, created_space_member_id, title, body, body_html, status, applied_at, created_at, updated_at | ✓ すべて一致                                   |
| インデックス: `[topic_id, status]`, `[space_id]`, `[created_space_member_id]`                                               | ✓ すべて一致                                   |
| `internal/model/id.go` に `SuggestionID` 型を追加                                                                           | ✓ 実装済み                                     |
| `internal/model/suggestion.go` に `Suggestion` モデル定義                                                                   | ✓ 実装済み                                     |
| `SuggestionStatus` 型（Draft=0, Open=1, Applied=2, Closed=3）                                                               | ✓ すべて一致                                   |
| 想定ファイル数: 実装 3                                                                                                      | ✓ 3ファイル（migration, id.go, suggestion.go） |
| テストなし                                                                                                                  | ✓ テストなし（仕様通り）                       |

設計との乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 1a-1（suggestionsテーブルのマイグレーションとモデル定義）が作業計画書の仕様通りに正確に実装されています。マイグレーションファイルはカラム定義ガイドライン（VARCHAR長さ指定なし、TIMESTAMP WITH TIME ZONE）に準拠し、FK制約とインデックスも適切に設定されています。モデル定義は既存のパターン（ドメインID型、int32の列挙型、日本語コメント）と完全に一致しており、コードベース全体の一貫性が保たれています。
