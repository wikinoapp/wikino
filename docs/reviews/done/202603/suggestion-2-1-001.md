# コードレビュー: suggestion-2-1

## レビュー情報

| 項目                       | 内容                                 |
| -------------------------- | ------------------------------------ |
| レビュー日                 | 2026-03-16                           |
| 対象ブランチ               | suggestion-2-1                       |
| ベースブランチ             | develop                              |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md     |
| 変更ファイル数             | 12 ファイル                          |
| 変更行数（実装）           | +247 行（sqlc生成コード +86行 除く） |
| 変更行数（テスト）         | +242 行                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/get_suggestion_list.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [x] `go/internal/repository/space_member.go`
- [x] `go/internal/repository/user_repository.go`
- [x] `go/internal/model/id.go`
- [x] `go/db/queries/space_members.sql`
- [x] `go/db/queries/users.sql`

### テストファイル

- [x] `go/internal/usecase/get_suggestion_list_test.go`
- [x] `go/internal/viewmodel/suggestion_test.go`

### 設定・その他（自動生成）

- [x] `go/internal/query/space_members.sql.go`（sqlc自動生成）
- [x] `go/internal/query/users.sql.go`（sqlc自動生成）

### ドキュメント

- [x] `docs/plans/1_doing/suggestion.md`（タスク2-1のチェックボックスを完了に更新）

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

### レビュー詳細（問題なし）

**アーキテクチャ（依存関係）**:

- UseCase → Repository のみに依存。Query への直接アクセスなし ✅
- ViewModel → Model のみに依存 ✅
- Repository → Query, Model のみに依存 ✅

**命名規則**:

- UseCase: `GetSuggestionListUsecase`（`Get` プレフィックス = 読み取りUseCase）✅
- Output型: `GetSuggestionListOutput`（`Output` サフィックス、既存パターンと一致）✅
- ViewModel: `SuggestionForList`（画面要件に応じた命名）✅
- ファイル名: `get_suggestion_list.go`（`{action}_{entity}.go` パターン）✅

**ドメインID型**:

- `SpaceMemberIDsToStrings`, `UserIDsToStrings` ヘルパーが既存パターン（`PageIDsToStrings` 等）と一致 ✅
- ViewModel の `Number` フィールドが `int32` 型（既存ViewModelの `Page.Number: int32`, `Topic.Number: int32` と一致）✅

**セキュリティ**:

- `FindSpaceMembersByIDs` クエリが `space_id` でスコープされている ✅
- `FindUsersByIDs` はスペーススコープなし（`users` テーブルはグローバルエンティティのため正当）✅

**コーディング規約**:

- コメントは日本語 ✅
- `log/slog` パッケージの使用（ログ出力箇所なし、問題なし）✅
- エラーメッセージは日本語（`fmt.Errorf("編集提案一覧の取得に失敗: %w", err)`）✅

**テスト**:

- `get_suggestion_list_test.go`: `TestMain` が同パッケージに存在、`testutil.SetupTx` でトランザクション分離 ✅
- `suggestion_test.go`: DB不要のテスト、`t.Parallel()` でサブテストも並行実行 ✅
- テーブル駆動テスト（ViewModel）、サブテスト（UseCase）を適切に使い分け ✅

**設計との整合性**:

- 作業計画書タスク2-1の要件をすべて満たしている ✅
  - `GetSuggestionListUsecase`: オープン/クローズのフィルタリング対応（`Statuses` パラメータ）
  - `SuggestionForList`: タイトル、ステータス、作成者名、作成日時を含む
- 作成者情報の取得（SpaceMember → User の2段階取得）が効率的に実装されている（一括取得 + マップ構築）
- 「編集提案作成者がスペースから退会しても、作成済みの編集提案は保持される」要件に対応（`FindSpaceMembersByIDs` が `active` フィルタなし）

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク2-1（編集提案一覧のUseCase・ViewModel）の実装として、アーキテクチャガイド・コーディング規約・セキュリティガイドラインに準拠した質の高い実装です。

良い点:

- 3層アーキテクチャの依存関係ルールを厳密に遵守している
- SpaceMember/Userの一括取得メソッド（`FindByIDs`）を追加し、N+1問題を回避している
- ドメインID型のスライス変換ヘルパー（`SpaceMemberIDsToStrings`, `UserIDsToStrings`）が既存パターンと一貫している
- ViewModelの `userDisplayName` ヘルパーが名前/アットネームのフォールバックを適切に処理している
- テストカバレッジが十分（UseCase: 空リスト・フィルタリング・UserMap、ViewModel: 空リスト・正常系・フォールバック・不明メンバー）
