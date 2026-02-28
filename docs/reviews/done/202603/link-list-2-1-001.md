# コードレビュー: link-list-2-1

## レビュー情報

| 項目                       | 内容                                      |
| -------------------------- | ----------------------------------------- |
| レビュー日                 | 2026-02-28                                |
| 対象ブランチ               | link-list-2-1                             |
| ベースブランチ             | page-edit                                 |
| 作業計画書（指定があれば） | docs/plans/1_doing/link-list-alignment.md |
| 変更ファイル数             | 9 ファイル                                |
| 変更行数（実装）           | +126 行                                   |
| 変更行数（テスト）         | +342 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/pages.sql`
- [x] `go/internal/query/pages.sql.go`（自動生成）
- [x] `go/internal/repository/page.go`
- [x] `go/internal/repository/pagination.go`
- [x] `go/internal/viewmodel/pagination.go`
- [x] `go/internal/testutil/page_builder.go`

### テストファイル

- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/viewmodel/pagination_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/link-list-alignment.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

### レビュー詳細（問題なしの確認事項）

#### `go/db/queries/pages.sql`

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md)
- [@go/CLAUDE.md#コーディング規約](/workspace/go/CLAUDE.md)

**確認結果**:

- 4つの新規クエリ（FindLinkedPagesPaginated, CountLinkedPages, FindBacklinkedPagesPaginated, CountBacklinkedPages）すべてに `space_id` 条件が含まれている
- `published_at IS NOT NULL` と `discarded_at IS NULL` の条件が既存クエリと一貫して適用されている
- ソート順 `ORDER BY modified_at DESC, id DESC` が作業計画書の設計に一致
- コメントが日本語で記述されている
- プリペアドステートメント（`$1`, `$2` 等）を使用しておりSQLインジェクション対策済み

#### `go/internal/repository/page.go`

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 依存関係ルール
- [@go/CLAUDE.md#ドメインID型](/workspace/go/CLAUDE.md)

**確認結果**:

- Repository → Query, Model の依存関係ルールに準拠
- `model.PageID`, `model.SpaceID` 等のドメインID型を正しく使用
- `model.PageIDsToStrings()` による変換ヘルパーの活用
- `*PaginatedPages` 構造体による適切な戻り値設計
- エラーハンドリングが既存メソッド（`FindByIDs` 等）と一貫

#### `go/internal/repository/pagination.go`

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Domain/Infrastructure層

**確認結果**:

- `PaginatedPages` 構造体が Domain/Infrastructure 層に正しく配置
- `model.Page` への依存のみで、上位層に依存していない

#### `go/internal/viewmodel/pagination.go`

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#ビューモデル](/workspace/go/docs/architecture-guide.md)

**確認結果**:

- `NewPagination` コンストラクタがPresentation層（viewmodel）に正しく配置
- ceil除算のロジックが正しい: `(totalCount + limit - 1) / limit`
- 0件時の `total = 1` への補正が適切（0ページは表示できないため）
- 上位層への依存なし

#### `go/internal/testutil/page_builder.go`

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md)

**確認結果**:

- `WithModifiedAt` が既存の `With*` メソッドパターンに一致
- ビルダーパターンの一貫性が保たれている

#### `go/internal/repository/page_test.go`

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md)
- [@go/CLAUDE.md#テーブル駆動テスト](/workspace/go/CLAUDE.md)

**確認結果**:

- `t.Parallel()` による並行テスト実行
- `testutil.SetupTx(t)` によるトランザクション分離
- ページネーション（1ページ目・2ページ目）、ソート順、非公開ページ除外、空IDリスト、バックリンクなしページ等の各ケースを網羅
- 既存テストと一貫したスタイル（サブテストの命名規則、アサーションパターン）

#### `go/internal/viewmodel/pagination_test.go`

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テーブル駆動テスト](/workspace/go/CLAUDE.md)

**確認結果**:

- テーブル駆動テストを適切に使用
- 境界値（15件ちょうど、16件、0件、30件ちょうど等）を網羅
- `t.Parallel()` を外側・内側の両方で設定
- バックリンク一覧用のlimit（14件/ページ）もテストケースに含まれている

## 設計との整合性チェック

作業計画書 タスク 2-1 の要件との整合性を確認しました。

| 要件                                                                     | 状態 |
| ------------------------------------------------------------------------ | ---- |
| `internal/repository/pagination.go` に `PaginatedPages` 構造体を新規作成 | ✅   |
| `pages.sql` に4クエリ追加（データ取得2 + 件数カウント2）                 | ✅   |
| sqlc生成コード更新                                                       | ✅   |
| `FindLinkedPagesPaginated` メソッド追加                                  | ✅   |
| `FindBacklinkedPagesPaginated` メソッド追加                              | ✅   |
| ページネーション計算ロジック（`NewPagination`）の追加                    | ✅   |
| リポジトリメソッドのテスト追加                                           | ✅   |
| ページネーション計算のテスト追加                                         | ✅   |

**軽微な差異**（問題なし）:

- 作業計画書では「ページネーション計算ヘルパー」を `repository/pagination.go` に配置する想定でしたが、実装では `viewmodel/pagination.go` に `NewPagination` として配置されています。これはアーキテクチャガイドに照らして正しい判断です（ページネーション計算はPresentation層の関心事であり、Domain/Infrastructure層に置くべきではない）
- Repository メソッドの `page` / `limit` パラメータが設計（`int`）と異なり `int32` ですが、sqlc生成コードの型（`int32`）と一致させるための実用的な判断であり問題ありません

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 2-1（オフセットページネーション基盤とリンク一覧への適用）が設計通りに実装されており、ガイドラインにも準拠しています。

**良かった点**:

- SQLクエリがすべて `space_id` でスコープされており、セキュリティガイドラインに準拠
- `PaginatedPages`（Repository層）と `Pagination`（ViewModel層）の責務が明確に分離されている
- テストが正常系・異常系・境界値を十分にカバーしている（テストコード342行）
- 既存コードとの一貫性が保たれている（エラーハンドリング、命名規則、ビルダーパターン）
- `NewPagination` を ViewModel 層に配置した判断が適切（作業計画書からの改善）
