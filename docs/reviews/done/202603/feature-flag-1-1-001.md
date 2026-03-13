# コードレビュー: feature-flag-1-1

## レビュー情報

| 項目                       | 内容                                                   |
| -------------------------- | ------------------------------------------------------ |
| レビュー日                 | 2026-03-13                                             |
| 対象ブランチ               | feature-flag-1-1                                       |
| ベースブランチ             | handler-usecase-refactor-7-2                           |
| 作業計画書（指定があれば） | docs/plans/1_doing/feature-flag-anonymous.md           |
| 変更ファイル数             | 15 ファイル                                            |
| 変更行数（実装）           | 約 +100 / -50 行（ミドルウェア、モデル、リポジトリ等） |
| 変更行数（テスト）         | 約 +90 / -23 行                                        |

※ 上記の行数は自動生成コード（sqlc）、スキーマダンプ、作業計画書を除く

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - 開発環境ガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/model/feature_flag.go`
- [x] `go/internal/repository/feature_flag_repository.go`
- [ ] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/repository/page.go`

### テストファイル

- [x] `go/internal/repository/feature_flag_repository_test.go`
- [x] `go/internal/testutil/feature_flag_builder.go`

### 設定・その他

- [x] `go/db/migrations/20260313062541_alter_feature_flags_add_device_token.sql`
- [x] `go/db/queries/feature_flags.sql`
- [x] `go/db/schema.sql`（自動生成）
- [x] `go/internal/query/feature_flags.sql.go`（自動生成）
- [x] `go/internal/query/models.go`（自動生成）
- [x] `go/internal/query/pages.sql.go`（自動生成）
- [x] `go/go.mod`
- [x] `go/sqlc.yaml`
- [x] `docs/plans/1_doing/feature-flag-anonymous.md`

## ファイルごとのレビュー結果

### `go/internal/middleware/reverse_proxy.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - レイヤー間の依存関係
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [docs/plans/1_doing/feature-flag-anonymous.md](/workspace/docs/plans/1_doing/feature-flag-anonymous.md) - 作業計画書

**問題点・改善提案**:

- **[作業計画書との乖離]**: `isFeatureFlagEnabled` メソッドが `device_token` Cookie を読み取っておらず、常に空文字列を `deviceToken` パラメータに渡している（319-348行目）。作業計画書の設計では、`device_token` と `sessionToken` の両方を読み取り、1クエリで判定する仕様になっている

  ```go
  // 現在の実装（338行目）
  enabled, err := m.featureFlagRepo.IsEnabledForDevice(r.Context(), "", sessionToken, flagName)
  ```

  **状況の確認**: 作業計画書のタスクリストを見ると、ミドルウェアの完全な変更はフェーズ2（タスク2-1）で実施予定。今回のフェーズ1では `IsEnabledBySessionToken` の削除に伴い、コンパイルを通すための最小限の変更を行ったと理解している。

  この理解が正しい場合、**現時点の実装は意図通り**であり問題ない。ただし以下の点を確認したい：
  1. フェーズ1での `sessionToken == ""` 時の早期リターン（332-335行目）は、フェーズ2で `device_token` Cookie の確認に置き換わる想定か
  2. `featureFlagChecker` インターフェースの変更がフェーズ2のタスクリストにも記載されているが、実際にはフェーズ1で完了している。タスクリストの更新は不要か

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 理解通り、フェーズ2で対応予定（現状のまま）
  - [ ] フェーズ1で device_token の読み取りも追加すべき
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

フェーズ1（データベースとリポジトリ層）の実装として、品質の高い変更になっている。

**良かった点**:

- マイグレーションが適切に設計されている（CHECK制約、ユニーク制約、down migrationも完備）
- `IsFeatureFlagEnabledForDevice` クエリが1クエリで `device_token` と `user_id` 両方をチェックする効率的な設計
- `sql.NullString` を使い、空文字列時にNULLを送信することで、SQLの比較演算で自然にフィルタが無効化される実装
- テストカバレッジが充実している（device_token有効/無効、session token有効/無効、両方の組み合わせ、両方空の場合）
- `sqlc.yaml` でnullable UUIDを `*string` に変換する設定追加により、`github.com/google/uuid` への直接依存を削除できた
- テストビルダーのバリデーション（`deviceToken` または `userID` のいずれかが必須）が適切

**確認が必要な点**:

- ミドルウェアの変更がフェーズ1とフェーズ2で分割されている点について、意図の確認が1件
