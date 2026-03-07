# コードレビュー: draft-update-2-1

## レビュー情報

| 項目                       | 内容                                    |
| -------------------------- | --------------------------------------- |
| レビュー日                 | 2026-03-05                              |
| 対象ブランチ               | draft-update-2-1                        |
| ベースブランチ             | draft-update                            |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md      |
| 変更ファイル数             | 3 ファイル                              |
| 変更行数（実装）           | +98 / -0 行（新規ファイル）             |
| 変更行数（テスト）         | +123 / -0 行（新規ファイル）            |
| 変更行数（ドキュメント）   | +7 / -11 行（作業計画書のリネーム反映） |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（UseCase パターン）
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/manual_save_draft_page.go`

### テストファイル

- [x] `go/internal/usecase/manual_save_draft_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/draft-update.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っています。

### レビュー詳細（問題なし）

#### `go/internal/usecase/manual_save_draft_page.go`

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase パターン、WithTx パターン、命名規則
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コメント、ログ出力
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - スペースIDによるクエリスコープ

**確認結果**:

- ファイル名 `manual_save_draft_page.go` は `{action}_{entity}.go` の命名規則に準拠
- 構造体名 `ManualSaveDraftPageUsecase` は `{Action}{Entity}Usecase`（小文字 `c`）の規則に準拠
- コンストラクタ `NewManualSaveDraftPageUsecase` は規則に準拠
- トランザクションパターン（`BeginTx` → `defer Rollback` → `WithTx` → `Commit`）は `create_account.go` 等の既存パターンと一致
- Repository の `WithTx` パターンを正しく使用（元の Repository を変更せず新しいインスタンスを作成）
- `SpaceID` を `FindByPageAndMember` に渡しており、スペースIDによるクエリスコープのセキュリティ要件を満たす
- エラーメッセージは日本語で `fmt.Errorf` + `%w` によるラップが適切
- Input/Output 構造体にドメインID型を使用（`model.SpaceID`, `model.PageID`, `model.SpaceMemberID`）
- `ErrDraftPageNotFound` のセンチネルエラー定義は適切

#### `go/internal/usecase/manual_save_draft_page_test.go`

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - TestMain パターン、`GetTestDB` vs `SetupTx` の使い分け

**確認結果**:

- UseCase テストで `GetTestDB()` を使用（UseCase は自前でトランザクション管理を行うため、`SetupTx` ではなく `GetTestDB` が正しい）
- 正常系テスト（`TestManualSaveDraftPageUsecase_Execute`）で出力値の検証が網羅的（Title, Body, BodyHTML, SpaceMemberID, CreatedAt）
- 異常系テスト（`TestManualSaveDraftPageUsecase_Execute_DraftPageNotFound`）で DraftPage が存在しない場合のエラーハンドリングを検証
- テストデータのビルダーパターン（`NewXBuilderDB`）が既存パターンと一致

#### `docs/plans/1_doing/draft-update.md`

**確認結果**:

- タスク 2-1 のステータスを `[x]` に更新
- Usecase 名を `CreateDraftPageRevisionUsecase` → `ManualSaveDraftPageUsecase` にリネーム
- 命名方針セクションを自動保存/手動保存の対にした命名に更新
- 実装と一致している

## 設計との整合性チェック

作業計画書タスク 2-1 の要件:

| 要件                                                                    | 実装状況 |
| ----------------------------------------------------------------------- | -------- |
| `internal/usecase/manual_save_draft_page.go` に Usecase 実装            | ✅       |
| DraftPage の現在の内容でスナップショット（DraftPageRevision）を作成する | ✅       |
| トランザクション内で DraftPageRevision を作成                           | ✅       |
| テストファイルの追加                                                    | ✅       |

設計との乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

ManualSaveDraftPageUsecase の実装は、既存の UseCase パターン（`create_account.go` 等）と一貫性があり、アーキテクチャガイドラインに忠実に従っています。トランザクション管理、WithTx パターン、スペースIDによるクエリスコープ、ドメインID型の使用、エラーハンドリングなど、すべてのガイドラインに準拠しています。テストも正常系・異常系の両方をカバーしており、UseCase テストで `GetTestDB()` を使用する正しいパターンに従っています。
